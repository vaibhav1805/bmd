package knowledge

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// waitForEvent blocks until an event matching kind and relPath arrives on ch,
// or until timeout elapses. If the expected event is not received, the test fails.
func waitForEvent(t *testing.T, ch <-chan WatchEvent, timeout time.Duration, kind WatchEventKind, relPath string) {
	t.Helper()
	deadline := time.NewTimer(timeout)
	defer deadline.Stop()
	for {
		select {
		case <-deadline.C:
			t.Fatalf("timeout waiting for event kind=%d relPath=%q", kind, relPath)
		case evt, ok := <-ch:
			if !ok {
				t.Fatalf("Events channel closed before receiving expected event kind=%d relPath=%q", kind, relPath)
			}
			if evt.Kind == kind && evt.RelPath == relPath {
				return
			}
		}
	}
}

// waitNoEvent asserts that no event arrives on ch within the given duration.
func waitNoEvent(t *testing.T, ch <-chan WatchEvent, duration time.Duration) {
	t.Helper()
	timer := time.NewTimer(duration)
	defer timer.Stop()
	select {
	case <-timer.C:
		// Good: no event arrived.
	case evt := <-ch:
		t.Fatalf("unexpected event received: kind=%d path=%q relPath=%q", evt.Kind, evt.Path, evt.RelPath)
	}
}

// TestWatcher_Created verifies that creating a new .md file fires a WatchCreated event.
func TestWatcher_Created(t *testing.T) {
	dir := t.TempDir()
	w := NewFileWatcher(dir, 50*time.Millisecond)
	w.Start()
	defer w.Stop()

	// Give the watcher one poll cycle to establish its initial snapshot.
	time.Sleep(100 * time.Millisecond)

	// Write a new markdown file.
	path := filepath.Join(dir, "new.md")
	if err := os.WriteFile(path, []byte("# Hello"), 0o644); err != nil {
		t.Fatal(err)
	}

	waitForEvent(t, w.Events, 2*time.Second, WatchCreated, "new.md")
}

// TestWatcher_Modified verifies that modifying an existing .md file fires a WatchModified event.
func TestWatcher_Modified(t *testing.T) {
	dir := t.TempDir()

	// Create the file before starting the watcher so it is in the initial snapshot.
	path := filepath.Join(dir, "existing.md")
	if err := os.WriteFile(path, []byte("# Original"), 0o644); err != nil {
		t.Fatal(err)
	}

	w := NewFileWatcher(dir, 50*time.Millisecond)
	w.Start()
	defer w.Stop()

	// Give the watcher one poll cycle to snapshot the existing file.
	time.Sleep(100 * time.Millisecond)

	// Modify the file content and ensure mtime advances.
	time.Sleep(10 * time.Millisecond)
	if err := os.WriteFile(path, []byte("# Modified"), 0o644); err != nil {
		t.Fatal(err)
	}

	waitForEvent(t, w.Events, 2*time.Second, WatchModified, "existing.md")
}

// TestWatcher_Deleted verifies that deleting a .md file fires a WatchDeleted event.
func TestWatcher_Deleted(t *testing.T) {
	dir := t.TempDir()

	// Create the file before starting the watcher.
	path := filepath.Join(dir, "todelete.md")
	if err := os.WriteFile(path, []byte("# Bye"), 0o644); err != nil {
		t.Fatal(err)
	}

	w := NewFileWatcher(dir, 50*time.Millisecond)
	w.Start()
	defer w.Stop()

	// Give the watcher one poll cycle to snapshot.
	time.Sleep(100 * time.Millisecond)

	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}

	waitForEvent(t, w.Events, 2*time.Second, WatchDeleted, "todelete.md")
}

// TestWatcher_IgnoresHidden verifies that .md files inside hidden directories are ignored.
func TestWatcher_IgnoresHidden(t *testing.T) {
	dir := t.TempDir()
	w := NewFileWatcher(dir, 50*time.Millisecond)
	w.Start()
	defer w.Stop()

	// Give the watcher one poll cycle to establish snapshot.
	time.Sleep(100 * time.Millisecond)

	// Write a .md file inside a hidden directory.
	hiddenDir := filepath.Join(dir, ".git")
	if err := os.MkdirAll(hiddenDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(hiddenDir, "secret.md"), []byte("secret"), 0o644); err != nil {
		t.Fatal(err)
	}

	// No event should arrive.
	waitNoEvent(t, w.Events, 200*time.Millisecond)
}

// TestWatcher_IgnoresNonMd verifies that non-.md files do not fire events.
func TestWatcher_IgnoresNonMd(t *testing.T) {
	dir := t.TempDir()
	w := NewFileWatcher(dir, 50*time.Millisecond)
	w.Start()
	defer w.Stop()

	// Give the watcher one poll cycle to establish snapshot.
	time.Sleep(100 * time.Millisecond)

	// Write a .txt file.
	if err := os.WriteFile(filepath.Join(dir, "notes.txt"), []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}

	// No event should arrive.
	waitNoEvent(t, w.Events, 200*time.Millisecond)
}

// TestWatcher_StopClean verifies that Stop() causes the Events channel to be closed.
func TestWatcher_StopClean(t *testing.T) {
	dir := t.TempDir()
	w := NewFileWatcher(dir, 50*time.Millisecond)
	w.Start()

	w.Stop()

	// Events channel must be closed within 1 second.
	timer := time.NewTimer(1 * time.Second)
	defer timer.Stop()
	select {
	case <-timer.C:
		t.Fatal("Events channel was not closed within 1s after Stop()")
	case _, ok := <-w.Events:
		if ok {
			// A stray event arrived; drain and check closure.
			// Retry once for the closed signal.
			select {
			case <-timer.C:
				t.Fatal("Events channel was not closed within 1s after Stop()")
			case _, ok2 := <-w.Events:
				if ok2 {
					t.Fatal("Events channel still open after Stop()")
				}
			}
		}
		// Channel closed (ok == false) — test passes.
	}
}

// TestWatcher_Idempotent verifies that calling Stop() twice does not panic.
func TestWatcher_Idempotent(t *testing.T) {
	dir := t.TempDir()
	w := NewFileWatcher(dir, 50*time.Millisecond)
	w.Start()

	// This must not panic.
	w.Stop()
	w.Stop()
}
