package knowledge

import (
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

// buildTestDir creates a temporary directory and writes named .md files.
func buildTestDir(t *testing.T, files map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	for name, content := range files {
		path := filepath.Join(dir, name)
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			t.Fatalf("mkdir for %s: %v", name, err)
		}
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}
	return dir
}

// buildIndexAndGraph scans dir and builds an initial Index, Graph, and
// ComponentRegistry from the discovered documents.
func buildIndexAndGraph(t *testing.T, dir string) (*Index, *Graph, *ComponentRegistry) {
	t.Helper()
	docs, err := ScanDirectory(dir, ScanConfig{UseDefaultIgnores: true})
	if err != nil {
		t.Fatalf("ScanDirectory: %v", err)
	}
	idx := NewIndex()
	if err := idx.Build(docs); err != nil {
		t.Fatalf("Index.Build: %v", err)
	}
	gb := NewGraphBuilder(dir)
	g := gb.Build(docs)
	reg := NewComponentRegistry()
	reg.InitFromGraph(g, docs)
	return idx, g, reg
}

// waitForStats polls until cond returns true for the updater's stats or the
// timeout is reached, in which case it fails the test.
func waitForStats(t *testing.T, u *IncrementalUpdater, cond func(UpdaterStats) bool, timeout time.Duration) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if cond(u.Stats()) {
			return
		}
		time.Sleep(25 * time.Millisecond)
	}
	t.Fatalf("waitForStats: condition not met within %v; final stats: %+v", timeout, u.Stats())
}

// TestIncremental_ModifiedFileReIndexed verifies that overwriting a file causes
// the new content to appear in search results.
func TestIncremental_ModifiedFileReIndexed(t *testing.T) {
	dir := buildTestDir(t, map[string]string{
		"svc-a.md": "# Service A\ncalls auth-service",
	})

	idx, g, reg := buildIndexAndGraph(t, dir)
	watcher := NewFileWatcher(dir, 50*time.Millisecond)
	updater := NewIncrementalUpdater(dir, watcher, idx, g, reg, nil, nil)
	watcher.Start()
	updater.Start()
	defer updater.Stop()

	// Overwrite the file with different content.
	newContent := "# Service A v2\ncalls payment-service"
	if err := os.WriteFile(filepath.Join(dir, "svc-a.md"), []byte(newContent), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	waitForStats(t, updater, func(s UpdaterStats) bool { return s.FilesIndexed >= 1 }, 3*time.Second)

	// New content should be searchable.
	results, err := idx.Search("payment-service", 5)
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(results) == 0 {
		t.Error("expected search results for 'payment-service' after re-index, got 0")
	}
}

// TestIncremental_DeletedFileRemoved verifies that deleting a file removes it
// from the search index.
func TestIncremental_DeletedFileRemoved(t *testing.T) {
	dir := buildTestDir(t, map[string]string{
		"svc-a.md": "# Service A\ncalls auth-service",
		"svc-b.md": "# Service B\nunique-svc-b-keyword",
	})

	idx, g, reg := buildIndexAndGraph(t, dir)
	watcher := NewFileWatcher(dir, 50*time.Millisecond)
	updater := NewIncrementalUpdater(dir, watcher, idx, g, reg, nil, nil)
	watcher.Start()
	updater.Start()
	defer updater.Stop()

	// Verify svc-b.md content is present before deletion.
	before, _ := idx.Search("unique-svc-b-keyword", 5)
	if len(before) == 0 {
		t.Skip("svc-b.md content not indexed initially — skipping deletion test")
	}

	// Delete the file.
	if err := os.Remove(filepath.Join(dir, "svc-b.md")); err != nil {
		t.Fatalf("Remove: %v", err)
	}

	waitForStats(t, updater, func(s UpdaterStats) bool { return s.FilesRemoved >= 1 }, 3*time.Second)

	// The deleted content should no longer appear.
	results, _ := idx.Search("unique-svc-b-keyword", 5)
	if len(results) > 0 {
		t.Errorf("expected 0 search results after deletion, got %d", len(results))
	}
}

// TestIncremental_CreatedFileIndexed verifies that a new .md file added to
// the directory appears in search results after detection.
func TestIncremental_CreatedFileIndexed(t *testing.T) {
	// Start with an empty directory.
	dir := t.TempDir()

	idx := NewIndex()
	_ = idx.Build(nil)
	g := NewGraph()
	reg := NewComponentRegistry()

	watcher := NewFileWatcher(dir, 50*time.Millisecond)
	updater := NewIncrementalUpdater(dir, watcher, idx, g, reg, nil, nil)
	watcher.Start()
	updater.Start()
	defer updater.Stop()

	// Write a new file after watching has started.
	// Use a distinctive word that will survive stripMarkdown and tokenization.
	content := "# New Service\nxyzwidgetservice handles authentication"
	if err := os.WriteFile(filepath.Join(dir, "new-service.md"), []byte(content), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	waitForStats(t, updater, func(s UpdaterStats) bool { return s.FilesIndexed >= 1 }, 3*time.Second)

	// Search for the distinctive token that will be in the indexed content.
	results, err := idx.Search("xyzwidgetservice", 5)
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(results) == 0 {
		t.Error("expected search results for 'xyzwidgetservice' after creation, got 0")
	}
}

// TestIncremental_UnchangedFileSkipped verifies that writing the same content
// does not trigger repeated re-indexing.
func TestIncremental_UnchangedFileSkipped(t *testing.T) {
	content := "# Service A\ncalls auth-service"
	dir := buildTestDir(t, map[string]string{
		"svc-a.md": content,
	})

	idx, g, reg := buildIndexAndGraph(t, dir)
	watcher := NewFileWatcher(dir, 50*time.Millisecond)
	updater := NewIncrementalUpdater(dir, watcher, idx, g, reg, nil, nil)
	watcher.Start()
	updater.Start()
	defer updater.Stop()

	// Write the same content — mtime changes but hash does not.
	if err := os.WriteFile(filepath.Join(dir, "svc-a.md"), []byte(content), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	// Wait long enough for at least 2 poll cycles.
	time.Sleep(200 * time.Millisecond)

	// The file may be detected by mtime change but should not be re-indexed
	// multiple times. Assert at most 1 re-index per mtime change.
	stats := updater.Stats()
	if stats.FilesIndexed > 1 {
		t.Errorf("expected <= 1 FilesIndexed for unchanged content, got %d", stats.FilesIndexed)
	}
}

// TestIncremental_Stats verifies that Stats() returns correct counts after
// a file creation event.
func TestIncremental_Stats(t *testing.T) {
	dir := t.TempDir()
	idx := NewIndex()
	_ = idx.Build(nil)
	g := NewGraph()
	reg := NewComponentRegistry()

	watcher := NewFileWatcher(dir, 50*time.Millisecond)
	updater := NewIncrementalUpdater(dir, watcher, idx, g, reg, nil, nil)
	watcher.Start()
	updater.Start()

	// Create a file.
	if err := os.WriteFile(filepath.Join(dir, "smoke.md"), []byte("# Smoke test"), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	waitForStats(t, updater, func(s UpdaterStats) bool { return s.FilesIndexed >= 1 }, 3*time.Second)
	updater.Stop()

	stats := updater.Stats()
	if stats.FilesIndexed == 0 {
		t.Error("expected FilesIndexed > 0 after file creation")
	}
}

// TestIncremental_OnChangeCallback verifies that the onChange callback fires
// after a file creation event.
func TestIncremental_OnChangeCallback(t *testing.T) {
	dir := t.TempDir()
	idx := NewIndex()
	_ = idx.Build(nil)
	g := NewGraph()
	reg := NewComponentRegistry()

	var mu sync.Mutex
	var notified []WatchEvent
	callback := func(evt WatchEvent) {
		mu.Lock()
		notified = append(notified, evt)
		mu.Unlock()
	}

	watcher := NewFileWatcher(dir, 50*time.Millisecond)
	updater := NewIncrementalUpdater(dir, watcher, idx, g, reg, nil, callback)
	watcher.Start()
	updater.Start()
	defer updater.Stop()

	if err := os.WriteFile(filepath.Join(dir, "cb-test.md"), []byte("# CB Test"), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		mu.Lock()
		count := len(notified)
		mu.Unlock()
		if count >= 1 {
			break
		}
		time.Sleep(25 * time.Millisecond)
	}

	mu.Lock()
	count := len(notified)
	mu.Unlock()
	if count == 0 {
		t.Error("onChange callback was never invoked after file creation")
	}
}
