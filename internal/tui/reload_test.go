package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/bmd/bmd/internal/editor"
	"github.com/bmd/bmd/internal/parser"
	"github.com/bmd/bmd/internal/theme"
)

// manyParagraphs builds n one-line markdown paragraphs, each rendering to
// exactly one line (verified empirically against internal/renderer's
// output), so tests can control the resulting v.Lines length precisely.
func manyParagraphs(n int) string {
	md := ""
	for i := 1; i <= n; i++ {
		md += fmt.Sprintf("Paragraph line %d\n\n", i)
	}
	return md
}

// TestReloadFile_PreservesOffset covers both halves of D-05: a mid-document
// offset survives a reload unchanged when the new document is still long
// enough, and gets clamped into range (never negative, never >= len(Lines),
// never reset to 0) when the new document is shorter than the old offset.
func TestReloadFile_PreservesOffset(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.md")
	initial := manyParagraphs(20)
	if err := os.WriteFile(path, []byte(initial), 0o644); err != nil {
		t.Fatalf("write initial file: %v", err)
	}

	doc, err := parser.ParseMarkdown(initial)
	if err != nil {
		t.Fatalf("parse initial file: %v", err)
	}
	v := New(doc, path, theme.NewTheme(), 80)
	v.Height = 5

	t.Run("offset unchanged when new document is still long enough", func(t *testing.T) {
		v.Offset = 5
		updated := manyParagraphs(15)
		if err := os.WriteFile(path, []byte(updated), 0o644); err != nil {
			t.Fatalf("write updated file: %v", err)
		}

		vv, cmd := v.reloadFile(path)
		if cmd != nil {
			t.Fatalf("expected reloadFile to return a nil tea.Cmd, got non-nil")
		}
		if vv.Offset != 5 {
			t.Fatalf("expected Offset to remain 5 (preserved), got %d", vv.Offset)
		}
		if len(vv.Lines) == 0 {
			t.Fatalf("expected reloadFile to re-render Lines, got empty")
		}
	})

	t.Run("offset clamped, not reset to 0, when new document shrank below it", func(t *testing.T) {
		v.Offset = 30 // beyond what the next, much shorter document will have
		// 8 paragraphs render to 10 lines (blank + 8 + blank); with
		// Height=5, maxOffset() is 5 — large enough that the correctly
		// clamped value is non-zero, distinguishing "clamped in range" from
		// "unconditionally reset to 0".
		short := manyParagraphs(8)
		if err := os.WriteFile(path, []byte(short), 0o644); err != nil {
			t.Fatalf("write shrunk file: %v", err)
		}

		vv, _ := v.reloadFile(path)
		wantOffset := clamp(len(vv.Lines)-1, 0, vv.maxOffset())
		if wantOffset == 0 {
			t.Fatalf("test setup error: expected clamp target to be non-zero to meaningfully assert against a hard reset")
		}
		if vv.Offset < 0 {
			t.Fatalf("Offset must never be negative, got %d", vv.Offset)
		}
		if len(vv.Lines) > 0 && vv.Offset >= len(vv.Lines) {
			t.Fatalf("Offset %d must be clamped below len(Lines) %d", vv.Offset, len(vv.Lines))
		}
		if vv.Offset != wantOffset {
			t.Fatalf("Offset must be clamped to %d (valid in-range value), not hard-reset: got %d", wantOffset, vv.Offset)
		}
	})
}

// TestReloadFile_EditModeGuard verifies D-09: a fileChangedMsg arriving while
// v.editMode is true must never mutate the edit buffer or any edit-mode
// state — the guard in Viewer.Update() must drop the reload entirely (no
// reloadFile call), not merely queue it.
func TestReloadFile_EditModeGuard(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.md")
	initial := manyParagraphs(5)
	if err := os.WriteFile(path, []byte(initial), 0o644); err != nil {
		t.Fatalf("write initial file: %v", err)
	}

	doc, err := parser.ParseMarkdown(initial)
	if err != nil {
		t.Fatalf("parse initial file: %v", err)
	}
	v := New(doc, path, theme.NewTheme(), 80)
	v.reloadCh = make(chan string, 1)
	v.editMode = true
	v.editBuffer = editor.NewTextBuffer([]string{"unsaved edit content", "second line"})

	beforeDoc := v.Doc
	beforeLines := append([]string(nil), v.Lines...)
	beforeBufferLines := append([]string(nil), v.editBuffer.GetLines()...)

	// Change the file on disk after entering edit mode — this must never be
	// applied while editMode is true.
	updated := manyParagraphs(50)
	if err := os.WriteFile(path, []byte(updated), 0o644); err != nil {
		t.Fatalf("write updated file: %v", err)
	}

	m, cmd := v.Update(fileChangedMsg{path: path})
	vv, ok := m.(*Viewer)
	if !ok {
		t.Fatalf("expected *Viewer from Update, got %T", m)
	}
	if cmd == nil {
		t.Fatalf("expected Update to re-issue waitForFileChange to keep listening, got nil cmd")
	}
	if !vv.editMode {
		t.Fatalf("editMode must remain true after a dropped reload")
	}
	if vv.Doc != beforeDoc {
		t.Fatalf("Doc must be byte-identical (same pointer, untouched) while editMode is true")
	}
	if len(vv.Lines) != len(beforeLines) {
		t.Fatalf("Lines length changed while editMode is true: got %d, want %d", len(vv.Lines), len(beforeLines))
	}
	for i := range beforeLines {
		if vv.Lines[i] != beforeLines[i] {
			t.Fatalf("Lines[%d] mutated while editMode is true", i)
		}
	}
	afterBufferLines := vv.editBuffer.GetLines()
	if len(afterBufferLines) != len(beforeBufferLines) {
		t.Fatalf("edit buffer length changed while editMode is true: got %d, want %d", len(afterBufferLines), len(beforeBufferLines))
	}
	for i := range beforeBufferLines {
		if afterBufferLines[i] != beforeBufferLines[i] {
			t.Fatalf("edit buffer line %d mutated while editMode is true", i)
		}
	}
}

// waitForEventOrTimeout blocks on ch for a debounced reload path, failing
// the test instead of hanging the suite if nothing arrives within timeout
// (Pitfall 5 — never a bare time.Sleep for fsnotify-driven assertions).
func waitForEventOrTimeout(t *testing.T, ch <-chan string, timeout time.Duration) string {
	t.Helper()
	select {
	case path := <-ch:
		return path
	case <-time.After(timeout):
		t.Fatalf("timed out after %s waiting for a file-change event", timeout)
		return ""
	}
}

// TestFileWatcher_Integration exercises the real fsnotify watcher end to
// end: startWatching on a temp file, an external write to that file, and
// exactly one debounced path arriving on reloadCh (D-07, D-08, Pitfall 1/2).
func TestFileWatcher_Integration(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "watched.md")
	initial := manyParagraphs(3)
	if err := os.WriteFile(path, []byte(initial), 0o644); err != nil {
		t.Fatalf("write initial file: %v", err)
	}

	doc, err := parser.ParseMarkdown(initial)
	if err != nil {
		t.Fatalf("parse initial file: %v", err)
	}
	v := New(doc, path, theme.NewTheme(), 80)
	if v.watcher == nil {
		t.Fatalf("expected New() to start the watcher for a non-empty FilePath")
	}
	defer v.watcher.Close()

	updated := manyParagraphs(4)
	if err := os.WriteFile(path, []byte(updated), 0o644); err != nil {
		t.Fatalf("write updated file: %v", err)
	}

	got := waitForEventOrTimeout(t, v.reloadCh, 2*time.Second)
	if got != path {
		t.Fatalf("expected reload event for %q, got %q", path, got)
	}

	select {
	case extra := <-v.reloadCh:
		t.Fatalf("expected exactly one coalesced reload event, got a second: %q", extra)
	case <-time.After(2 * reloadDebounce):
		// no second event arrived — correct.
	}
}

// TestDebounce_CoalescesBurst verifies D-07: invoking debounce N times in
// rapid succession within the wait window fires the callback exactly once,
// after the window elapses following the LAST call (not the first).
func TestDebounce_CoalescesBurst(t *testing.T) {
	fired := make(chan struct{}, 10)
	waitFor := 50 * time.Millisecond

	const burst = 5
	for i := 0; i < burst; i++ {
		debounce(waitFor, func() { fired <- struct{}{} })
		time.Sleep(waitFor / 5) // well within the debounce window
	}

	// Wait past the debounce window (measured from the last call) for the
	// single coalesced fire, bounded so a failure can't hang the suite.
	select {
	case <-fired:
	case <-time.After(2 * waitFor):
		t.Fatalf("expected debounce to fire once after the burst, got none")
	}

	select {
	case <-fired:
		t.Fatalf("expected exactly one fire for the burst, got a second")
	case <-time.After(2 * waitFor):
		// no second fire arrived — correct.
	}
}
