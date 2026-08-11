package tui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func keyType(t tea.KeyType) tea.KeyMsg { return tea.KeyMsg{Type: t} }

// lineContaining returns the v.Lines index of the first rendered line whose
// plain (ANSI-stripped) text contains needle, failing the test if none is
// found. Markdown paragraphs are rendered with a leading margin and
// consecutive non-blank source lines merge into one paragraph, so tests
// search for content rather than assuming a fixed row offset.
func lineContaining(t *testing.T, v *Viewer, needle string) int {
	t.Helper()
	for i, l := range v.Lines {
		if strings.Contains(stripANSI(l), needle) {
			return i
		}
	}
	t.Fatalf("no line containing %q found in v.Lines: %#v", needle, v.Lines)
	return -1
}

// TestKeyboardCursor_RightWrapsAcrossLines is a regression test for the
// keyboard cursor feature: repeated Right presses must reach every line in
// the document (not just columns within one line), since Left/Right are
// the only way a keyboard-only user (no mouse) can reach a cursor position
// at all. Markdown paragraphs must be blank-line-separated to render as
// distinct rows — see lineContaining.
func TestKeyboardCursor_RightWrapsAcrossLines(t *testing.T) {
	dir := t.TempDir()
	v := newTestFileViewer(t, dir, "a.md", "ab\n\ncd\n", 100, 24)

	abRow := lineContaining(t, v, "ab")
	cdRow := lineContaining(t, v, "cd")
	abLen := len([]rune(stripANSI(v.Lines[abRow])))

	v.moveCursor(abRow, 0, false)
	for i := 0; i < abLen+1; i++ { // step past every column of "ab", then wrap
		row, col := v.cursorTargetRight(v.cursorRow, v.cursorCol)
		v.moveCursor(row, col, false)
	}

	if !v.hasCursor {
		t.Fatal("expected hasCursor=true after cursor movement")
	}
	if v.cursorRow != cdRow || v.cursorCol != 0 {
		t.Errorf("expected wrap to (row=%d, col=0) on the next line, got (row=%d, col=%d)", cdRow, v.cursorRow, v.cursorCol)
	}
}

// TestKeyboardCursor_LeftWrapsAcrossLines mirrors the Right-wrap test in
// the other direction.
func TestKeyboardCursor_LeftWrapsAcrossLines(t *testing.T) {
	dir := t.TempDir()
	v := newTestFileViewer(t, dir, "a.md", "ab\n\ncd\n", 100, 24)

	abRow := lineContaining(t, v, "ab")
	cdRow := lineContaining(t, v, "cd")
	abLen := len([]rune(stripANSI(v.Lines[abRow])))

	v.moveCursor(cdRow, 0, false) // start of "cd"
	row, col := v.cursorTargetLeft(v.cursorRow, v.cursorCol)
	v.moveCursor(row, col, false)

	if v.cursorRow != abRow || v.cursorCol != abLen {
		t.Errorf("expected Left from the start of a line to land at the end of the prior line (row=%d, col=%d), got (row=%d, col=%d)", abRow, abLen, v.cursorRow, v.cursorCol)
	}
}

// TestKeyboardCursor_ClampsAtDocumentBoundaries verifies Left at the very
// start and Right at the very end of the document are no-ops rather than
// going out of bounds.
func TestKeyboardCursor_ClampsAtDocumentBoundaries(t *testing.T) {
	dir := t.TempDir()
	v := newTestFileViewer(t, dir, "a.md", "ab\n", 100, 24)

	v.moveCursor(0, 0, false)
	row, col := v.cursorTargetLeft(v.cursorRow, v.cursorCol)
	v.moveCursor(row, col, false)
	if v.cursorRow != 0 || v.cursorCol != 0 {
		t.Errorf("expected Left at document start to stay at (0,0), got (%d,%d)", v.cursorRow, v.cursorCol)
	}

	lastRow := len(v.Lines) - 1
	lastCol := len([]rune(stripANSI(v.Lines[lastRow])))
	v.moveCursor(lastRow, lastCol, false)
	row, col = v.cursorTargetRight(v.cursorRow, v.cursorCol)
	v.moveCursor(row, col, false)
	if v.cursorRow != lastRow || v.cursorCol != lastCol {
		t.Errorf("expected Right at document end to stay at (%d,%d), got (%d,%d)", lastRow, lastCol, v.cursorRow, v.cursorCol)
	}
}

// TestKeyboardCursor_ShiftRightSelectsWithinLine drives the actual key
// path (Update, not the internal helpers directly) to confirm Shift+Right
// builds a real, copyable selection matching Shift+Click's behavior.
func TestKeyboardCursor_ShiftRightSelectsWithinLine(t *testing.T) {
	dir := t.TempDir()
	v := newTestFileViewer(t, dir, "a.md", "hello world\n", 100, 24)

	row := lineContaining(t, v, "hello world")
	plain := stripANSI(v.Lines[row])
	startCol := strings.Index(plain, "hello") // rendered lines carry a left margin, so don't assume column 0
	if startCol < 0 {
		t.Fatalf("test setup error: %q not found in rendered line %q", "hello", plain)
	}
	v.moveCursor(row, startCol, false)
	for i := 0; i < 5; i++ {
		m, _ := v.Update(keyType(tea.KeyShiftRight))
		v = m.(*Viewer)
	}

	if got := v.SelectedText(); got != "hello" {
		t.Errorf("expected Shift+Right x5 from the start of %q to select %q, got %q", "hello", "hello", got)
	}
}

// TestKeyboardCursor_ShiftDownSelectsAcrossLines verifies vertical
// keyboard selection spans multiple lines correctly.
func TestKeyboardCursor_ShiftDownSelectsAcrossLines(t *testing.T) {
	dir := t.TempDir()
	v := newTestFileViewer(t, dir, "a.md", "first\n\nsecond\n\nthird\n", 100, 24)

	firstRow := lineContaining(t, v, "first")
	v.moveCursor(firstRow, 0, false)
	m, _ := v.Update(keyType(tea.KeyShiftDown))
	v = m.(*Viewer)
	m, _ = v.Update(keyType(tea.KeyShiftDown))
	v = m.(*Viewer)

	got := v.SelectedText()
	if !strings.Contains(got, "first") || !strings.Contains(got, "second") {
		t.Errorf("expected a multi-line selection spanning 'first' and 'second', got %q", got)
	}
	if strings.Contains(got, "third") {
		t.Errorf("expected the selection to stop before 'third' after only 2 Shift+Down presses, got %q", got)
	}
}

// TestKeyboardCursor_PlainMoveClearsSelection is a regression test for the
// same class of bug as StartSelection's stale-text fix: after building a
// selection with Shift+Right, a subsequent PLAIN (non-Shift) Right press
// must drop the selection entirely, not leave stale selected text behind
// — matching how a plain mouse click always starts fresh.
func TestKeyboardCursor_PlainMoveClearsSelection(t *testing.T) {
	dir := t.TempDir()
	v := newTestFileViewer(t, dir, "a.md", "hello world\n", 100, 24)

	row := lineContaining(t, v, "hello world")
	v.moveCursor(row, 0, false)
	m, _ := v.Update(keyType(tea.KeyShiftRight))
	v = m.(*Viewer)
	if v.SelectedText() == "" {
		t.Fatal("test setup error: expected a selection after Shift+Right")
	}

	// Plain (non-Shift) Right via the real key path.
	m, _ = v.Update(keyType(tea.KeyRight))
	v = m.(*Viewer)

	if text := v.SelectedText(); text != "" {
		t.Errorf("expected plain Right to clear the selection, got stale text: %q", text)
	}
}

// TestKeyboardCursor_MovesCursorForCtrlCLineCopy is the end-to-end
// motivating use case: a keyboard-only user (no mouse) navigates to a line
// with Right and can then copy it via Ctrl+C, exercising the exact same
// "committed cursor, no selection" path the mouse-click flow already used.
func TestKeyboardCursor_MovesCursorForCtrlCLineCopy(t *testing.T) {
	dir := t.TempDir()
	v := newTestFileViewer(t, dir, "a.md", "first\n\nsecond\n\nthird\n", 100, 24)

	secondRow := lineContaining(t, v, "second")
	v.moveCursor(secondRow, 0, false)

	if !v.hasCursor || v.cursorRow != secondRow {
		t.Fatalf("expected committed cursor at row %d, got hasCursor=%v row=%d", secondRow, v.hasCursor, v.cursorRow)
	}
	if v.SelectedText() != "" {
		t.Fatalf("expected no selection from a plain cursor move, got %q", v.SelectedText())
	}
	plainLine := strings.TrimSpace(stripANSI(v.Lines[v.cursorRow]))
	if plainLine != "second" {
		t.Errorf("expected the line at the committed cursor to be %q, got %q", "second", plainLine)
	}
}

// TestKeyboardCursor_ScrollsCursorIntoView verifies moving the cursor past
// the bottom of a short viewport scrolls the document to keep it visible,
// mirroring scrollToMatch()'s behavior for search matches.
func TestKeyboardCursor_ScrollsCursorIntoView(t *testing.T) {
	dir := t.TempDir()
	var sb strings.Builder
	for i := 0; i < 40; i++ {
		sb.WriteString("line\n\n")
	}
	if err := os.WriteFile(filepath.Join(dir, "a.md"), []byte(sb.String()), 0o644); err != nil {
		t.Fatal(err)
	}
	v := newTestFileViewer(t, dir, "a.md", sb.String(), 100, 10) // short viewport: 8 content rows

	v.moveCursor(0, 0, false)
	if v.Offset != 0 {
		t.Fatalf("test setup error: expected Offset=0 initially, got %d", v.Offset)
	}

	v.moveCursor(30, 0, false)
	contentHeight := v.Height - 2
	if v.cursorRow < v.Offset || v.cursorRow >= v.Offset+contentHeight {
		t.Errorf("expected cursor row %d to be within the visible viewport [%d, %d), got Offset=%d", v.cursorRow, v.Offset, v.Offset+contentHeight, v.Offset)
	}
}
