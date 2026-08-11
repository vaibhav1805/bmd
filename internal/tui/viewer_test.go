package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/bmd/bmd/internal/ast"
	"github.com/bmd/bmd/internal/editor"
	"github.com/bmd/bmd/internal/knowledge"
	"github.com/bmd/bmd/internal/renderer"
	"github.com/bmd/bmd/internal/theme"
	tea "github.com/charmbracelet/bubbletea"
)

// Helper function to create a test document with given lines.
func createTestDocument(lines []string) *ast.Document {
	return &ast.Document{}
}

// TestEditModeToggle tests entering and exiting edit mode.
func TestEditModeToggle(t *testing.T) {
	doc := createTestDocument([]string{})
	v := New(doc, "test.md", theme.NewTheme(), 80)
	v.Lines = []string{"# Header", "This is a test."}

	// Initially not in edit mode
	if v.editMode {
		t.Error("Expected editMode to be false initially")
	}

	// Manually set edit mode (since we can't easily test key presses)
	v.editMode = true
	v.editBuffer = editor.NewTextBuffer(v.Lines)

	output := v.renderEditMode()
	if !strings.Contains(output, "[EDIT MODE]") {
		t.Error("Expected [EDIT MODE] indicator in edit mode output")
	}

	// Exit edit mode
	v.editMode = false
	// Output should no longer contain [EDIT MODE] (it will show rendered view)
}

// TestEditModeTextInsertion tests character insertion in edit mode.
func TestEditModeTextInsertion(t *testing.T) {
	doc := createTestDocument([]string{})
	v := New(doc, "test.md", theme.NewTheme(), 80)
	v.Lines = []string{"hello"}

	v.editMode = true
	v.editBuffer = editor.NewTextBuffer(v.Lines)

	v.editBuffer.Insert('X')

	resultLines := v.editBuffer.GetLines()
	if resultLines[0] != "Xhello" {
		t.Errorf("Expected 'Xhello', got '%s'", resultLines[0])
	}
}

// TestEditModeUndo tests undo in edit mode.
func TestEditModeUndo(t *testing.T) {
	doc := createTestDocument([]string{})
	v := New(doc, "test.md", theme.NewTheme(), 80)
	v.Lines = []string{"hello"}

	v.editMode = true
	v.editBuffer = editor.NewTextBuffer(v.Lines)

	v.editBuffer.Insert('X')
	v.editBuffer.Undo()

	resultLines := v.editBuffer.GetLines()
	if resultLines[0] != "hello" {
		t.Errorf("Expected 'hello' after undo, got '%s'", resultLines[0])
	}
}

// TestEditModeSave tests file persistence setup.
func TestEditModeSave(t *testing.T) {
	doc := createTestDocument([]string{})
	v := New(doc, "test.md", theme.NewTheme(), 80)
	v.Lines = []string{"original content"}

	v.editMode = true
	v.editBuffer = editor.NewTextBuffer(v.Lines)

	// Make an edit
	v.editBuffer.Insert('X')

	resultLines := v.editBuffer.GetLines()
	if resultLines[0] != "Xoriginal content" {
		t.Errorf("Expected 'Xoriginal content', got '%s'", resultLines[0])
	}
}

// TestEditModeNavigation tests cursor movement in edit mode.
func TestEditModeNavigation(t *testing.T) {
	doc := createTestDocument([]string{})
	v := New(doc, "test.md", theme.NewTheme(), 80)
	v.Lines = []string{"line1", "line2", "line3"}

	v.editMode = true
	v.editBuffer = editor.NewTextBuffer(v.Lines)

	// Test CursorDown
	v.editBuffer.CursorDown()
	if v.editBuffer.CursorLine() != 1 {
		t.Errorf("Expected cursor at line 1, got %d", v.editBuffer.CursorLine())
	}

	// Test CursorRight
	v.editBuffer.CursorRight()
	if v.editBuffer.CursorCol() != 1 {
		t.Errorf("Expected cursor at col 1, got %d", v.editBuffer.CursorCol())
	}
}

// TestEditModeDeleteKey tests delete key functionality in edit mode.
func TestEditModeDeleteKey(t *testing.T) {
	doc := createTestDocument([]string{})
	v := New(doc, "test.md", theme.NewTheme(), 80)
	v.Lines = []string{"hello"}

	v.editMode = true
	v.editBuffer = editor.NewTextBuffer(v.Lines)

	// Delete at position 0
	v.editBuffer.Delete()

	resultLines := v.editBuffer.GetLines()
	if resultLines[0] != "ello" {
		t.Errorf("Expected 'ello' after delete, got '%s'", resultLines[0])
	}
}

// TestEditModeBackspaceKey tests backspace key functionality in edit mode.
func TestEditModeBackspaceKey(t *testing.T) {
	doc := createTestDocument([]string{})
	v := New(doc, "test.md", theme.NewTheme(), 80)
	v.Lines = []string{"hello"}

	v.editMode = true
	v.editBuffer = editor.NewTextBuffer(v.Lines)

	// Move to position 2
	v.editBuffer.CursorRight()
	v.editBuffer.CursorRight()

	// Backspace to delete 'e'
	v.editBuffer.Backspace()

	resultLines := v.editBuffer.GetLines()
	if resultLines[0] != "hllo" {
		t.Errorf("Expected 'hllo' after backspace, got '%s'", resultLines[0])
	}
}

// TestEditModeEnterNewLine tests Enter key functionality in edit mode.
func TestEditModeEnterNewLine(t *testing.T) {
	doc := createTestDocument([]string{})
	v := New(doc, "test.md", theme.NewTheme(), 80)
	v.Lines = []string{"hello"}

	v.editMode = true
	v.editBuffer = editor.NewTextBuffer(v.Lines)

	// Move to position 2 (between 'l' and 'l')
	v.editBuffer.CursorRight()
	v.editBuffer.CursorRight()

	// Enter new line
	v.editBuffer.EnterNewLine()

	resultLines := v.editBuffer.GetLines()
	if len(resultLines) != 2 {
		t.Errorf("Expected 2 lines after EnterNewLine, got %d", len(resultLines))
	}

	if resultLines[0] != "he" {
		t.Errorf("Expected 'he', got '%s'", resultLines[0])
	}

	if resultLines[1] != "llo" {
		t.Errorf("Expected 'llo', got '%s'", resultLines[1])
	}
}

// TestEditModeRedo tests redo functionality in edit mode.
func TestEditModeRedo(t *testing.T) {
	doc := createTestDocument([]string{})
	v := New(doc, "test.md", theme.NewTheme(), 80)
	v.Lines = []string{"hello"}

	v.editMode = true
	v.editBuffer = editor.NewTextBuffer(v.Lines)

	// Type, undo, then redo
	v.editBuffer.Insert('X')
	v.editBuffer.Undo()
	v.editBuffer.Redo()

	resultLines := v.editBuffer.GetLines()
	if resultLines[0] != "Xhello" {
		t.Errorf("Expected 'Xhello' after undo then redo, got '%s'", resultLines[0])
	}
}

// TestEditModeMultilineEdit tests editing with multiple lines.
func TestEditModeMultilineEdit(t *testing.T) {
	doc := createTestDocument([]string{})
	v := New(doc, "test.md", theme.NewTheme(), 80)
	v.Lines = []string{"line1", "line2", "line3"}

	v.editMode = true
	v.editBuffer = editor.NewTextBuffer(v.Lines)

	// Move to second line
	v.editBuffer.CursorDown()

	// Insert character
	v.editBuffer.Insert('X')

	resultLines := v.editBuffer.GetLines()
	if resultLines[1] != "Xline2" {
		t.Errorf("Expected 'Xline2' on line 2, got '%s'", resultLines[1])
	}

	// Line 1 should be unchanged
	if resultLines[0] != "line1" {
		t.Errorf("Expected 'line1' on line 1, got '%s'", resultLines[0])
	}
}

// TestEditModeCanUndo tests the CanUndo query method.
func TestEditModeCanUndo(t *testing.T) {
	doc := createTestDocument([]string{})
	v := New(doc, "test.md", theme.NewTheme(), 80)
	v.Lines = []string{"hello"}

	v.editMode = true
	v.editBuffer = editor.NewTextBuffer(v.Lines)

	// Initially no undo available
	if v.editBuffer.CanUndo() {
		t.Error("Expected CanUndo to be false initially")
	}

	// After insert, undo should be available
	v.editBuffer.Insert('X')
	if !v.editBuffer.CanUndo() {
		t.Error("Expected CanUndo to be true after insert")
	}

	// After undo, undo should not be available
	v.editBuffer.Undo()
	if v.editBuffer.CanUndo() {
		t.Error("Expected CanUndo to be false after undo")
	}
}

// TestEditModeCanRedo tests the CanRedo query method.
func TestEditModeCanRedo(t *testing.T) {
	doc := createTestDocument([]string{})
	v := New(doc, "test.md", theme.NewTheme(), 80)
	v.Lines = []string{"hello"}

	v.editMode = true
	v.editBuffer = editor.NewTextBuffer(v.Lines)

	// Initially no redo available
	if v.editBuffer.CanRedo() {
		t.Error("Expected CanRedo to be false initially")
	}

	// After insert and undo, redo should be available
	v.editBuffer.Insert('X')
	v.editBuffer.Undo()
	if !v.editBuffer.CanRedo() {
		t.Error("Expected CanRedo to be true after undo")
	}

	// After redo, redo should not be available
	v.editBuffer.Redo()
	if v.editBuffer.CanRedo() {
		t.Error("Expected CanRedo to be false after redo")
	}
}

// TestEditModeJumpToStart tests jumping to document start.
func TestEditModeJumpToStart(t *testing.T) {
	doc := createTestDocument([]string{})
	v := New(doc, "test.md", theme.NewTheme(), 80)
	v.Lines = []string{"line1", "line2", "line3"}

	v.editMode = true
	v.editBuffer = editor.NewTextBuffer(v.Lines)

	// Move to end
	v.editBuffer.JumpToEnd()

	// Jump back to start
	v.editBuffer.JumpToStart()

	if v.editBuffer.CursorLine() != 0 || v.editBuffer.CursorCol() != 0 {
		t.Errorf("Expected cursor at (0, 0) after JumpToStart, got (%d, %d)",
			v.editBuffer.CursorLine(), v.editBuffer.CursorCol())
	}
}

// TestEditModeJumpToEnd tests jumping to document end.
func TestEditModeJumpToEnd(t *testing.T) {
	doc := createTestDocument([]string{})
	v := New(doc, "test.md", theme.NewTheme(), 80)
	v.Lines = []string{"line1", "line2", "line3"}

	v.editMode = true
	v.editBuffer = editor.NewTextBuffer(v.Lines)

	// Jump to end
	v.editBuffer.JumpToEnd()

	if v.editBuffer.CursorLine() != 2 {
		t.Errorf("Expected cursor at line 2 after JumpToEnd, got %d", v.editBuffer.CursorLine())
	}

	if v.editBuffer.CursorCol() != 5 { // "line3" is 5 chars
		t.Errorf("Expected cursor at col 5 after JumpToEnd, got %d", v.editBuffer.CursorCol())
	}
}

// TestEditModeJumpToLine tests jumping to a specific line.
func TestEditModeJumpToLine(t *testing.T) {
	doc := createTestDocument([]string{})
	v := New(doc, "test.md", theme.NewTheme(), 80)
	v.Lines = []string{"line1", "line2", "line3"}

	v.editMode = true
	v.editBuffer = editor.NewTextBuffer(v.Lines)

	// Jump to line 1 (0-based)
	v.editBuffer.JumpToLine(1)

	if v.editBuffer.CursorLine() != 1 {
		t.Errorf("Expected cursor at line 1 after JumpToLine(1), got %d", v.editBuffer.CursorLine())
	}

	if v.editBuffer.CursorCol() != 0 {
		t.Errorf("Expected cursor at col 0 after JumpToLine(1), got %d", v.editBuffer.CursorCol())
	}
}

// TestEditModeJumpViaUpdateSetsOffset drives Ctrl+G go-to-line through
// v.Update() (not editBuffer.JumpToLine() directly) to catch the case where
// the edit buffer has grown past the original view-mode v.Lines length —
// updateJump() must not clamp v.Offset against the stale view-mode
// maxOffset(), the same bug class fixed for updateOutline() in 30-07.
func TestEditModeJumpViaUpdateSetsOffset(t *testing.T) {
	doc := createTestDocument([]string{})
	v := New(doc, "test.md", theme.NewTheme(), 80)
	v.Height = 24
	v.Width = 80
	// view-mode Lines is short — maxOffset() would clamp to 0 here.
	v.Lines = []string{"line1", "line2"}
	v.editMode = true
	lines := make([]string, 20)
	for i := range lines {
		lines[i] = fmt.Sprintf("line%d", i)
	}
	v.editBuffer = editor.NewTextBuffer(lines)

	v.jumpMode = true
	v.jumpInput = ""
	for _, r := range "10" {
		model, _ := v.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		v = model.(*Viewer)
	}
	model, _ := v.Update(tea.KeyMsg{Type: tea.KeyEnter})
	result := model.(*Viewer)

	if result.jumpMode {
		t.Error("expected jump mode closed after Enter")
	}
	if result.editBuffer.CursorLine() != 9 {
		t.Errorf("expected cursor at line 9, got %d", result.editBuffer.CursorLine())
	}
	if result.Offset != 9 {
		t.Errorf("expected scroll offset=9, got %d", result.Offset)
	}
}

// TestEditModeRenderEditMode tests the renderEditMode output.
func TestEditModeRenderEditMode(t *testing.T) {
	doc := createTestDocument([]string{})
	v := New(doc, "test.md", theme.NewTheme(), 80)
	v.Lines = []string{"hello world", "second line"}

	v.editMode = true
	v.editBuffer = editor.NewTextBuffer(v.Lines)

	output := v.renderEditMode()

	// Check that output contains expected elements
	if !strings.Contains(output, "[EDIT MODE]") {
		t.Error("Expected [EDIT MODE] in output")
	}

	if !strings.Contains(output, "test.md") {
		t.Error("Expected filename 'test.md' in output")
	}

	// Should contain at least one line number
	if !strings.Contains(output, "|") {
		t.Error("Expected line number separator '|' in output")
	}
}

// TestRenderEditMode_KittyPrependsDeleteAllImages is a regression test for
// bmd-xqh: renderEditMode() builds its own header rather than going
// through v.renderHeader(), so toggling into edit mode from a document
// that was showing a Kitty image bypassed the ghost-image cleanup —
// confirmed via a real-terminal screenshot showing the previous view-mode
// render's image stuck on screen behind the edit-mode text, even though
// edit mode itself only ever renders plain source text (v.editBuffer),
// never a rendered image.
func TestRenderEditMode_KittyPrependsDeleteAllImages(t *testing.T) {
	t.Setenv("KITTY_WINDOW_ID", "1")
	if renderer.DetectImageProtocol() != renderer.ProtocolKitty {
		t.Fatal("test setup error: expected ProtocolKitty with KITTY_WINDOW_ID set")
	}

	doc := createTestDocument([]string{})
	v := New(doc, "test.md", theme.NewTheme(), 80)
	v.Lines = []string{"hello world"}
	v.editMode = true
	v.editBuffer = editor.NewTextBuffer(v.Lines)

	output := v.renderEditMode()
	if !strings.HasPrefix(output, "\x1b_Ga=d,d=A\x1b\\") {
		t.Errorf("expected renderEditMode() to start with the Kitty delete-all-images command, got: %.100q...", output)
	}
}

// TestEditModeSetLines tests the SetLines method (for undo/redo).
func TestEditModeSetLines(t *testing.T) {
	doc := createTestDocument([]string{})
	v := New(doc, "test.md", theme.NewTheme(), 80)
	v.Lines = []string{"original1", "original2"}

	v.editMode = true
	v.editBuffer = editor.NewTextBuffer(v.Lines)

	// Change the lines
	newLines := []string{"new1", "new2", "new3"}
	v.editBuffer.SetLines(newLines)

	resultLines := v.editBuffer.GetLines()
	if len(resultLines) != 3 {
		t.Errorf("Expected 3 lines after SetLines, got %d", len(resultLines))
	}

	if resultLines[0] != "new1" {
		t.Errorf("Expected 'new1', got '%s'", resultLines[0])
	}

	if resultLines[2] != "new3" {
		t.Errorf("Expected 'new3', got '%s'", resultLines[2])
	}
}

// TestEditModeGetLines tests the GetLines method returns a copy.
func TestEditModeGetLines(t *testing.T) {
	doc := createTestDocument([]string{})
	v := New(doc, "test.md", theme.NewTheme(), 80)
	v.Lines = []string{"line1", "line2"}

	v.editMode = true
	v.editBuffer = editor.NewTextBuffer(v.Lines)

	// Get lines and modify the returned slice
	retrieved := v.editBuffer.GetLines()
	retrieved[0] = "modified"

	// Original should be unchanged
	newRetrieved := v.editBuffer.GetLines()
	if newRetrieved[0] != "line1" {
		t.Errorf("Expected 'line1', but GetLines returned modified value")
	}
}

// ─── Directory Browser Tests (DIR-01, ARCH-01/03/05) ─────────────────────────
//
// Directory-browser state now lives in DirectoryModel (directory.go); see
// directory_test.go for DirectoryModel-level Update()/View() unit tests
// (navigation, split toggle, file-open/mode-switch message emission).
// This section keeps the Viewer-level integration tests: constructing via
// NewDirectoryViewer/LoadDirectory, the full open-file/back-to-directory
// cycle through Viewer.Update()'s message handlers, and header/breadcrumb
// rendering (which stays a Viewer-owned concern, D-05).

// makeTempDir creates a temporary directory with optional .md files for testing.
// Returns the directory path; caller is responsible for cleanup (os.RemoveAll).
func makeTempDir(t *testing.T, files map[string]string) string {
	t.Helper()
	dir, err := os.MkdirTemp("", "bmd-dir-test-*")
	if err != nil {
		t.Fatalf("makeTempDir: %v", err)
	}
	for name, content := range files {
		p := filepath.Join(dir, name)
		if parent := filepath.Dir(p); parent != dir {
			if mkErr := os.MkdirAll(parent, 0o755); mkErr != nil {
				t.Fatalf("makeTempDir MkdirAll: %v", mkErr)
			}
		}
		if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
			t.Fatalf("makeTempDir WriteFile: %v", err)
		}
	}
	return dir
}

// TestNewDirectoryViewer verifies the constructor activates a DirectoryModel.
func TestNewDirectoryViewer(t *testing.T) {
	v := NewDirectoryViewer("/tmp", theme.NewTheme(), 80)
	if dirModel(v) == nil {
		t.Fatal("Expected activeChild to be a *DirectoryModel after NewDirectoryViewer")
	}
	if dirModel(v).state.RootPath != "/tmp" {
		t.Errorf("Expected RootPath=/tmp, got %q", dirModel(v).state.RootPath)
	}
	if dirModel(v).state.SelectedIndex != 0 {
		t.Errorf("Expected SelectedIndex=0, got %d", dirModel(v).state.SelectedIndex)
	}
	if v.currentView != "directory" {
		t.Errorf("Expected currentView='directory', got %q", v.currentView)
	}
}

// TestLoadDirectoryBasic verifies that LoadDirectory discovers .md files.
func TestLoadDirectoryBasic(t *testing.T) {
	dir := makeTempDir(t, map[string]string{
		"a.md": "# A", "b.md": "# B", "notes.txt": "not markdown",
	})
	defer os.RemoveAll(dir)

	v := NewDirectoryViewer(dir, theme.NewTheme(), 80)
	if err := v.LoadDirectory(dir); err != nil {
		t.Fatalf("LoadDirectory error: %v", err)
	}

	if len(dirModel(v).state.Files) != 2 {
		t.Errorf("Expected 2 .md files, got %d", len(dirModel(v).state.Files))
	}
}

// TestLoadDirectoryRecursive verifies recursive directory scanning.
func TestLoadDirectoryRecursive(t *testing.T) {
	dir := makeTempDir(t, map[string]string{
		"a.md":          "# A",
		"docs/b.md":     "# B",
		"docs/sub/c.md": "# C",
		"d.md":          "# D",
	})
	defer os.RemoveAll(dir)

	v := NewDirectoryViewer(dir, theme.NewTheme(), 80)
	if err := v.LoadDirectory(dir); err != nil {
		t.Fatalf("LoadDirectory error: %v", err)
	}

	if len(dirModel(v).state.Files) != 4 {
		t.Errorf("Expected 4 .md files (recursive), got %d", len(dirModel(v).state.Files))
	}
}

// TestOpenFileFromDirectory_SetsFlagsAndCurrentView drives the full
// keypress -> openFileCmd -> openFileMsg -> loadFile handoff (ARCH-03)
// through Viewer.Update() and verifies the resulting state.
func TestOpenFileFromDirectory_SetsFlagsAndCurrentView(t *testing.T) {
	dir := makeTempDir(t, map[string]string{"file.md": "# File\n"})
	defer os.RemoveAll(dir)

	v := NewDirectoryViewer(dir, theme.NewTheme(), 80)
	v.Height = 24
	if err := v.LoadDirectory(dir); err != nil {
		t.Fatalf("LoadDirectory error: %v", err)
	}

	v = pressKeySettled(v, tea.KeyMsg{Type: tea.KeyEnter})

	if dirModel(v) != nil {
		t.Error("Expected activeChild=nil (directory deactivated) after opening file")
	}
	if !v.openedFromDirectory {
		t.Error("Expected openedFromDirectory=true")
	}
	if v.currentView != "file" {
		t.Errorf("Expected currentView='file', got %q", v.currentView)
	}
}

// TestOpenFileFromDirectory_EmptyDoesNothing verifies that opening from an
// empty directory leaves the DirectoryModel active (nothing to open).
func TestOpenFileFromDirectory_EmptyDoesNothing(t *testing.T) {
	dir := makeTempDir(t, map[string]string{})
	defer os.RemoveAll(dir)

	v := NewDirectoryViewer(dir, theme.NewTheme(), 80)
	v.Height = 24
	if err := v.LoadDirectory(dir); err != nil {
		t.Fatalf("LoadDirectory error: %v", err)
	}

	v = pressKeySettled(v, tea.KeyMsg{Type: tea.KeyEnter})

	if dirModel(v) == nil {
		t.Error("Expected DirectoryModel to remain active when opening from empty directory")
	}
}

// TestBackToDirectory_RestoresModeCursorAndView verifies BackToDirectory
// restores the paused DirectoryModel (no rescan), the saved cursor position,
// clears openedFromDirectory, and resets currentView.
func TestBackToDirectory_RestoresModeCursorAndView(t *testing.T) {
	dir := makeTempDir(t, map[string]string{
		"aaa.md": "a", "bbb.md": "b", "ccc.md": "c",
	})
	defer os.RemoveAll(dir)

	v := NewDirectoryViewer(dir, theme.NewTheme(), 80)
	v.Height = 24
	if err := v.LoadDirectory(dir); err != nil {
		t.Fatalf("LoadDirectory error: %v", err)
	}

	// Select second file, then open and return.
	v = pressKeySettled(v, tea.KeyMsg{Type: tea.KeyDown})
	if dirModel(v).state.SelectedIndex != 1 {
		t.Fatalf("Expected SelectedIndex=1, got %d", dirModel(v).state.SelectedIndex)
	}
	filesBeforeOpen := dirModel(v).state.Files

	v = pressKeySettled(v, tea.KeyMsg{Type: tea.KeyEnter})
	if dirModel(v) != nil {
		t.Fatal("Expected activeChild=nil after opening file")
	}

	vv, cmd := v.BackToDirectory()
	v = settleCmd(vv, cmd)

	if dirModel(v) == nil {
		t.Fatal("Expected DirectoryModel restored after BackToDirectory")
	}
	if v.openedFromDirectory {
		t.Error("Expected openedFromDirectory=false after BackToDirectory")
	}
	if v.currentView != "directory" {
		t.Errorf("Expected currentView='directory', got %q", v.currentView)
	}
	if dirModel(v).state.SelectedIndex != 1 {
		t.Errorf("Expected cursor restored to 1, got %d", dirModel(v).state.SelectedIndex)
	}
	// No rescan: same Files slice header (len/cap/first element) as before.
	if len(dirModel(v).state.Files) != len(filesBeforeOpen) {
		t.Errorf("Expected Files preserved without rescanning, got len=%d want=%d", len(dirModel(v).state.Files), len(filesBeforeOpen))
	}
}

// TestBackToDirectory_NoopWhenNotFromDirectory verifies BackToDirectory does
// nothing if openedFromDirectory is false.
func TestBackToDirectory_NoopWhenNotFromDirectory(t *testing.T) {
	v := New(&ast.Document{}, "test.md", theme.NewTheme(), 80)

	vv, _ := v.BackToDirectory()

	if dirModel(vv) != nil {
		t.Error("Expected no DirectoryModel activated when BackToDirectory called without openedFromDirectory")
	}
}

// TestNavigationCycleDirToFileToDir verifies a full dir->file->dir cycle
// preserves the correct cursor index, repeated across several indices.
func TestNavigationCycleDirToFileToDir(t *testing.T) {
	dir := makeTempDir(t, map[string]string{
		"aaa.md": "a", "bbb.md": "b", "ccc.md": "c",
	})
	defer os.RemoveAll(dir)

	v := NewDirectoryViewer(dir, theme.NewTheme(), 80)
	v.Height = 24
	if err := v.LoadDirectory(dir); err != nil {
		t.Fatalf("LoadDirectory error: %v", err)
	}

	indices := []int{0, 2, 1, 2, 0}
	for i, wantIdx := range indices {
		dirModel(v).state.SelectedIndex = wantIdx

		v = pressKeySettled(v, tea.KeyMsg{Type: tea.KeyEnter})
		if dirModel(v) != nil {
			t.Fatalf("cycle %d: expected activeChild=nil after open", i)
		}

		vv, cmd := v.BackToDirectory()
		v = settleCmd(vv, cmd)
		if dirModel(v) == nil {
			t.Fatalf("cycle %d: expected DirectoryModel restored after back", i)
		}
		if dirModel(v).state.SelectedIndex != wantIdx {
			t.Errorf("cycle %d: expected SelectedIndex=%d after back, got %d", i, wantIdx, dirModel(v).state.SelectedIndex)
		}
	}
}

// TestBreadcrumbInHeader verifies that renderHeader shows breadcrumb when
// openedFromDirectory is true.
func TestBreadcrumbInHeader(t *testing.T) {
	dir := makeTempDir(t, map[string]string{
		"api.md": "# API\n",
	})
	defer os.RemoveAll(dir)

	v := NewDirectoryViewer(dir, theme.NewTheme(), 80)
	v.Height = 24
	if err := v.LoadDirectory(dir); err != nil {
		t.Fatalf("LoadDirectory error: %v", err)
	}
	v = pressKeySettled(v, tea.KeyMsg{Type: tea.KeyEnter})
	if v.currentView != "file" {
		t.Fatalf("Expected currentView='file', got %q", v.currentView)
	}

	header := v.renderHeader()
	plain := stripANSI(header)

	if !strings.Contains(plain, "api.md") {
		t.Errorf("Expected 'api.md' in breadcrumb header, got: %q", plain)
	}
	if !strings.Contains(plain, "[") || !strings.Contains(plain, "]") {
		t.Error("Expected '[dir] filename' breadcrumb format in header")
	}
}

// TestNoBreadcrumbInNormalFileHeader verifies that renderHeader shows normal
// header (no breadcrumb) when file was NOT opened from directory.
func TestNoBreadcrumbInNormalFileHeader(t *testing.T) {
	v := New(&ast.Document{}, "/tmp/file.md", theme.NewTheme(), 80)
	v.Height = 24

	header := v.renderHeader()
	plain := stripANSI(header)

	if !strings.Contains(plain, "file.md") {
		t.Errorf("Expected 'file.md' in header, got: %q", plain)
	}
	if strings.Contains(plain, "[/tmp]") {
		t.Error("Unexpected breadcrumb format '[dir]' in non-directory header")
	}
}

// TestHelp_ScrollsWithinBudget is a regression test for bmd's keymap
// audit: the full keybinding reference (helpContent) covers ~70 lines
// across every mode, which doesn't fit on one screen at typical terminal
// heights. The help overlay must scroll rather than silently overflow —
// j/down moves forward, k/up moves back, both clamped to [0, maxOffset].
func TestHelp_ScrollsWithinBudget(t *testing.T) {
	v := New(&ast.Document{}, "/tmp/file.md", theme.NewTheme(), 80)
	v.Height = 24 // short enough that helpContent (~70 lines) can't fit at once
	v.helpOpen = true

	budget := v.helpContentBudget()
	total := len(v.helpContent())
	if total <= budget {
		t.Fatalf("test setup error: expected helpContent (%d lines) to exceed the budget (%d) at Height=24 so scrolling is actually exercised", total, budget)
	}

	if v.helpScrollOffset != 0 {
		t.Fatalf("expected helpScrollOffset=0 initially, got %d", v.helpScrollOffset)
	}

	vv, _ := v.updateHelp(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	v = vv.(*Viewer)
	if v.helpScrollOffset != 1 {
		t.Errorf("expected helpScrollOffset=1 after one 'j', got %d", v.helpScrollOffset)
	}

	vv, _ = v.updateHelp(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("k")})
	v = vv.(*Viewer)
	if v.helpScrollOffset != 0 {
		t.Errorf("expected helpScrollOffset=0 after 'k' back, got %d", v.helpScrollOffset)
	}

	// 'k' at offset 0 must not go negative.
	vv, _ = v.updateHelp(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("k")})
	v = vv.(*Viewer)
	if v.helpScrollOffset != 0 {
		t.Errorf("expected helpScrollOffset clamped at 0, got %d", v.helpScrollOffset)
	}

	// 'G' jumps to the max offset (last page of content).
	vv, _ = v.updateHelp(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("G")})
	v = vv.(*Viewer)
	wantMax := total - budget
	if v.helpScrollOffset != wantMax {
		t.Errorf("expected 'G' to jump to max offset %d, got %d", wantMax, v.helpScrollOffset)
	}

	// A further 'j' at the max offset must not overshoot.
	vv, _ = v.updateHelp(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	v = vv.(*Viewer)
	if v.helpScrollOffset != wantMax {
		t.Errorf("expected helpScrollOffset clamped at max %d, got %d", wantMax, v.helpScrollOffset)
	}

	// 'g' resets to the top.
	vv, _ = v.updateHelp(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("g")})
	v = vv.(*Viewer)
	if v.helpScrollOffset != 0 {
		t.Errorf("expected 'g' to reset to offset 0, got %d", v.helpScrollOffset)
	}
}

// TestHelp_OpeningResetsScrollOffset verifies both help-open choke points
// (the direct '?' key in file view, and the toggleHelpMsg path used by
// DirectoryModel/CrossSearchModel/GraphModel) reset a stale scroll
// position from a previous help session, so reopening help doesn't start
// mid-scroll from wherever it was last left.
func TestHelp_OpeningResetsScrollOffset(t *testing.T) {
	v := New(&ast.Document{}, "/tmp/file.md", theme.NewTheme(), 80)
	v.Height = 24
	v.helpOpen = true
	v.helpScrollOffset = 5

	// Close via the toggleHelpMsg path (mirrors DirectoryModel/GraphModel).
	m, _ := v.Update(toggleHelpMsg{})
	v = m.(*Viewer)
	if v.helpOpen {
		t.Fatal("expected help closed after toggle")
	}

	// Reopen via the same path; offset must be back to 0, not the stale 5.
	m, _ = v.Update(toggleHelpMsg{})
	v = m.(*Viewer)
	if !v.helpOpen {
		t.Fatal("expected help open after second toggle")
	}
	if v.helpScrollOffset != 0 {
		t.Errorf("expected helpScrollOffset reset to 0 on reopen, got %d", v.helpScrollOffset)
	}
}

// TestHelp_RenderShowsScrollStatusWhenContentOverflows is a regression
// test for the same audit finding: renderHelp must surface that there's
// more content below (not just silently truncate), and the visible window
// must actually move to the newer helpScrollOffset — otherwise scrolling
// updates state but the user never sees a different screen.
func TestHelp_RenderShowsScrollStatusWhenContentOverflows(t *testing.T) {
	v := New(&ast.Document{}, "/tmp/file.md", theme.NewTheme(), 80)
	v.Height = 24
	v.helpOpen = true

	out := v.renderHelp()
	if !strings.Contains(out, "scroll") {
		t.Errorf("expected a scroll hint in help output when content overflows the budget, got: %.300q...", out)
	}

	before := v.renderHelp()
	v.helpScrollOffset = 5
	after := v.renderHelp()
	if before == after {
		t.Error("expected renderHelp output to change after scrolling (different content window)")
	}
}

// TestHelp_FollowLinkDoesNotClaimEnter is a regression test for a stale
// help-text bug found via bmd's keymap audit: the box previously listed
// "l / Enter -> Follow focused link", but no "enter" case exists anywhere
// in the file-view key switch — only 'l' actually follows a link. Enter is
// unbound in file view.
func TestHelp_FollowLinkDoesNotClaimEnter(t *testing.T) {
	v := New(&ast.Document{}, "/tmp/file.md", theme.NewTheme(), 80)
	found := false
	for _, l := range v.helpContent() {
		plain := stripANSI(l)
		if strings.Contains(plain, "Follow focused link") {
			found = true
			if strings.Contains(plain, "Enter") {
				t.Errorf("help text incorrectly claims Enter follows a link (only 'l' does): %q", plain)
			}
		}
	}
	if !found {
		t.Fatal("expected a 'Follow focused link' entry in helpContent")
	}
}

// TestRenderHeader_KittyPrependsDeleteAllImages is a regression test for
// bmd-xqh: Kitty images are an out-of-band pixel overlay that persists on
// screen even after bmd redraws different text in the same cells, so a
// previous document's image can be left as a "ghost" after navigating to a
// new file or back to the directory/search view. Since the header is the
// first line of every Viewer.View() frame, prepending the Kitty
// delete-all-images command there (rather than at each individual
// navigation choke point) guarantees any stale image is cleared before the
// frame's own content, if any, draws further down the same output.
func TestRenderHeader_KittyPrependsDeleteAllImages(t *testing.T) {
	t.Setenv("KITTY_WINDOW_ID", "1")
	if renderer.DetectImageProtocol() != renderer.ProtocolKitty {
		t.Fatal("test setup error: expected ProtocolKitty with KITTY_WINDOW_ID set")
	}

	v := New(&ast.Document{}, "/tmp/file.md", theme.NewTheme(), 80)
	v.Height = 24

	header := v.renderHeader()
	if !strings.HasPrefix(header, "\x1b_Ga=d,d=A\x1b\\") {
		t.Errorf("expected renderHeader() to start with the Kitty delete-all-images command, got: %q", header)
	}
	// The cleanup sequence must not leak into the visible header text.
	plain := stripANSI(header)
	if strings.Contains(plain, "_G") {
		t.Errorf("Kitty escape leaked into stripped header text: %q", plain)
	}
}

// TestRenderHeader_NonKittyOmitsDeleteAllImages verifies the Kitty
// ghost-image cleanup (see TestRenderHeader_KittyPrependsDeleteAllImages) is
// only emitted when the detected protocol is actually Kitty — sending an
// unrecognized APC sequence to every other terminal on every frame would be
// pure noise with no corresponding benefit.
func TestRenderHeader_NonKittyOmitsDeleteAllImages(t *testing.T) {
	t.Setenv("TERM", "xterm-256color")
	t.Setenv("TERM_PROGRAM", "")
	t.Setenv("ITERM_PROGRAM", "")
	t.Setenv("ITERM2_SHOULDMANAGEPASTEBOARD", "")
	t.Setenv("KITTY_WINDOW_ID", "")
	t.Setenv("COLORTERM", "")
	if renderer.DetectImageProtocol() == renderer.ProtocolKitty {
		t.Fatal("test setup error: expected non-Kitty protocol")
	}

	v := New(&ast.Document{}, "/tmp/file.md", theme.NewTheme(), 80)
	v.Height = 24

	header := v.renderHeader()
	if strings.Contains(header, "\x1b_Ga=d") {
		t.Errorf("did not expect the Kitty delete-all-images command outside a Kitty terminal, got: %q", header)
	}
}

// TestBreadcrumbShowsBackHint verifies the header hints 'h/Backspace: back to
// directory' when openedFromDirectory is true.
func TestBreadcrumbShowsBackHint(t *testing.T) {
	dir := makeTempDir(t, map[string]string{
		"doc.md": "# Doc\n",
	})
	defer os.RemoveAll(dir)

	v := NewDirectoryViewer(dir, theme.NewTheme(), 200) // wide for hint visibility
	v.Height = 24
	if err := v.LoadDirectory(dir); err != nil {
		t.Fatalf("LoadDirectory error: %v", err)
	}
	v = pressKeySettled(v, tea.KeyMsg{Type: tea.KeyEnter})

	header := v.renderHeader()
	plain := stripANSI(header)

	if !strings.Contains(plain, "back to directory") {
		t.Errorf("Expected 'back to directory' hint in header, got: %q", plain)
	}
}

// TestUpdateDirectoryLKeyCallsOpenFileFromDirectory verifies that pressing
// 'l' in directory mode triggers the file-open handoff end to end.
func TestUpdateDirectoryLKeyCallsOpenFileFromDirectory(t *testing.T) {
	dir := makeTempDir(t, map[string]string{"api.md": "# API\n"})
	defer os.RemoveAll(dir)

	v := NewDirectoryViewer(dir, theme.NewTheme(), 80)
	v.Height = 24
	if err := v.LoadDirectory(dir); err != nil {
		t.Fatalf("LoadDirectory error: %v", err)
	}

	v = pressKeySettled(v, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("l")})

	if dirModel(v) != nil {
		t.Error("Expected activeChild=nil after 'l' in directory mode")
	}
	if !v.openedFromDirectory {
		t.Error("Expected openedFromDirectory=true after 'l' in directory mode")
	}
}

// TestUpdateDirectoryEnterKeyCallsOpenFileFromDirectory verifies Enter also
// triggers file open end to end.
func TestUpdateDirectoryEnterKeyCallsOpenFileFromDirectory(t *testing.T) {
	dir := makeTempDir(t, map[string]string{"doc.md": "# Doc\n"})
	defer os.RemoveAll(dir)

	v := NewDirectoryViewer(dir, theme.NewTheme(), 80)
	v.Height = 24
	if err := v.LoadDirectory(dir); err != nil {
		t.Fatalf("LoadDirectory error: %v", err)
	}

	v = pressKeySettled(v, tea.KeyMsg{Type: tea.KeyEnter})

	if dirModel(v) != nil {
		t.Error("Expected activeChild=nil after Enter in directory mode")
	}
	if !v.openedFromDirectory {
		t.Error("Expected openedFromDirectory=true after Enter in directory mode")
	}
}

// TestBackToDirectoryResetsOffset verifies that returning to directory resets
// the scroll offset.
func TestBackToDirectoryResetsOffset(t *testing.T) {
	dir := makeTempDir(t, map[string]string{"doc.md": "# Doc\n"})
	defer os.RemoveAll(dir)

	v := NewDirectoryViewer(dir, theme.NewTheme(), 80)
	v.Height = 24
	if err := v.LoadDirectory(dir); err != nil {
		t.Fatalf("LoadDirectory error: %v", err)
	}
	v = pressKeySettled(v, tea.KeyMsg{Type: tea.KeyEnter})
	v.Offset = 42 // simulate having scrolled in the file

	vv, cmd := v.BackToDirectory()
	v = settleCmd(vv, cmd)

	if v.Offset != 0 {
		t.Errorf("Expected Offset=0 after BackToDirectory, got %d", v.Offset)
	}
}

// TestBackToDirectoryClearsSearch verifies search state is cleared when
// returning to directory.
func TestBackToDirectoryClearsSearch(t *testing.T) {
	dir := makeTempDir(t, map[string]string{"doc.md": "# Doc\n"})
	defer os.RemoveAll(dir)

	v := NewDirectoryViewer(dir, theme.NewTheme(), 80)
	v.Height = 24
	if err := v.LoadDirectory(dir); err != nil {
		t.Fatalf("LoadDirectory error: %v", err)
	}
	v = pressKeySettled(v, tea.KeyMsg{Type: tea.KeyEnter})
	v.searchMode = true
	v.searchInput = "test"

	vv, cmd := v.BackToDirectory()
	v = settleCmd(vv, cmd)

	if v.searchMode {
		t.Error("Expected searchMode=false after BackToDirectory")
	}
	if v.searchInput != "" {
		t.Errorf("Expected searchInput='' after BackToDirectory, got %q", v.searchInput)
	}
}

// ─── Split-Pane Mode Tests (09-01/09-02) — Viewer-level integration ─────────
//
// Low-level split-pane rendering mechanics (splitPaneWidths,
// renderDirectoryListingSplit, renderFilePreviewSplit, renderSplitPane) are
// now DirectoryModel methods; see directory_test.go for their unit tests.
// This section keeps the Viewer-level "does the full key -> View() pipeline
// still render split-pane output" integration coverage.

// TestSplitModeStateInitialized verifies that splitMode defaults from width.
func TestSplitModeStateInitialized(t *testing.T) {
	v := NewDirectoryViewer("/tmp", theme.NewTheme(), 120)
	if !dirModel(v).splitMode {
		t.Error("Expected splitMode=true by default at width>=80")
	}
	if dirModel(v).splitPreviewOffset != 0 {
		t.Errorf("Expected splitPreviewOffset=0, got %d", dirModel(v).splitPreviewOffset)
	}
}

// TestToggleSplitMode_KeyS verifies 's' key toggles splitMode on/off via
// Viewer.Update() end to end.
func TestToggleSplitMode_KeyS(t *testing.T) {
	dir := makeTempDir(t, map[string]string{"a.md": "# A\n"})
	defer os.RemoveAll(dir)

	v := NewDirectoryViewer(dir, theme.NewTheme(), 120)
	v.Height = 24
	if err := v.LoadDirectory(dir); err != nil {
		t.Fatalf("LoadDirectory error: %v", err)
	}
	initial := dirModel(v).splitMode

	v = pressKeySettled(v, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("s")})

	if dirModel(v).splitMode == initial {
		t.Error("Expected splitMode to flip after 's'")
	}
}

// TestSplitModeWarningNarrowTerminal verifies narrow terminals show the
// locked error message via the statusMsg mechanism (routed to v.errorMsg).
func TestSplitModeWarningNarrowTerminal(t *testing.T) {
	dir := makeTempDir(t, map[string]string{"a.md": "# A\n"})
	defer os.RemoveAll(dir)

	v := NewDirectoryViewer(dir, theme.NewTheme(), 60) // narrow
	v.Height = 24
	if err := v.LoadDirectory(dir); err != nil {
		t.Fatalf("LoadDirectory error: %v", err)
	}

	v = pressKeySettled(v, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("s")})

	if v.errorMsg != "Terminal too narrow for split pane (need 80+ cols)" {
		t.Errorf("Expected narrow-terminal error in v.errorMsg, got %q", v.errorMsg)
	}
}

// TestSplitModeExitToFullScreen verifies opening a file from split-pane mode
// exits split mode (full-screen file view), matching pre-refactor behavior.
func TestSplitModeExitToFullScreen(t *testing.T) {
	dir := makeTempDir(t, map[string]string{"a.md": "# A\n"})
	defer os.RemoveAll(dir)

	v := NewDirectoryViewer(dir, theme.NewTheme(), 120)
	v.Height = 24
	if err := v.LoadDirectory(dir); err != nil {
		t.Fatalf("LoadDirectory error: %v", err)
	}
	if !dirModel(v).splitMode {
		t.Fatal("expected splitMode=true at width 120 before opening a file")
	}

	v = pressKeySettled(v, tea.KeyMsg{Type: tea.KeyEnter})
	vv, cmd := v.BackToDirectory()
	v = settleCmd(vv, cmd)

	if dirModel(v).splitMode {
		t.Error("Expected splitMode=false after opening a file from split-pane view")
	}
}

// TestViewRoutesSplitMode verifies View() renders split-pane output (border
// character) when splitMode is active, and plain listing otherwise.
func TestViewRoutesSplitMode(t *testing.T) {
	dir := makeTempDir(t, map[string]string{
		"readme.md": "# README\nProject description here.\n",
	})
	defer os.RemoveAll(dir)

	v := NewDirectoryViewer(dir, theme.NewTheme(), 120)
	v.Height = 24
	if err := v.LoadDirectory(dir); err != nil {
		t.Fatalf("LoadDirectory error: %v", err)
	}

	dirModel(v).splitMode = true
	out := v.View()
	if !strings.Contains(out, "│") {
		t.Error("Expected split-pane border character in View() output when splitMode=true")
	}

	dirModel(v).splitMode = false
	out = v.View()
	if strings.Contains(out, "│") {
		t.Error("Expected no split-pane border character in View() output when splitMode=false")
	}
	if !strings.Contains(out, "Markdown Files in") {
		t.Error("Expected directory listing header in View() output when splitMode=false")
	}
}

// ============================================================================
// Phase 30.4: Cursor Position in Status Bar & Word Count Modal Tests
// ============================================================================

// TestCursorPositionInStatusBar verifies that when hasCursor=true, the status
// bar shows "Ln N, Col C" instead of the default line counter.
func TestCursorPositionInStatusBar(t *testing.T) {
	doc := createTestDocument([]string{})
	v := New(doc, "test.md", theme.NewTheme(), 80)
	v.Width = 80
	v.Lines = []string{"# Hello World", "This is a test line.", "Third line here."}

	// Without cursor: should show line counter, not Ln/Col
	status := v.renderStatusBar()
	if strings.Contains(status, "Ln ") && strings.Contains(status, ", Col ") {
		t.Error("Expected no Ln/Col display when hasCursor=false")
	}

	// With cursor set: should show Ln N, Col C
	v.hasCursor = true
	v.cursorRow = 1 // 0-based → displays as Ln 2
	v.cursorCol = 4 // 0-based → displays as Col 5
	status = v.renderStatusBar()
	if !strings.Contains(status, "Ln 2") {
		t.Errorf("Expected 'Ln 2' in status bar, got: %s", status)
	}
	if !strings.Contains(status, "Col 5") {
		t.Errorf("Expected 'Col 5' in status bar, got: %s", status)
	}
}

// TestCursorPositionFirstRow verifies cursor at (0,0) shows Ln 1, Col 1.
func TestCursorPositionFirstRow(t *testing.T) {
	doc := createTestDocument([]string{})
	v := New(doc, "test.md", theme.NewTheme(), 80)
	v.Width = 80
	v.Lines = []string{"Hello"}
	v.hasCursor = true
	v.cursorRow = 0
	v.cursorCol = 0
	status := v.renderStatusBar()
	if !strings.Contains(status, "Ln 1") {
		t.Errorf("Expected 'Ln 1' in status bar for row 0, got: %s", status)
	}
	if !strings.Contains(status, "Col 1") {
		t.Errorf("Expected 'Col 1' in status bar for col 0, got: %s", status)
	}
}

// TestCountDocumentStats verifies word/char/line counting.
func TestCountDocumentStats(t *testing.T) {
	lines := []string{
		"Hello world",
		"This is a test",
		"Three words here",
	}
	stats := CountDocumentStats(lines)
	if stats.Words != 9 {
		t.Errorf("Expected 9 words, got %d", stats.Words)
	}
	if stats.Lines != 3 {
		t.Errorf("Expected 3 lines, got %d", stats.Lines)
	}
	// Characters = sum non-whitespace runes
	// "Helloworld" (10) + "Thisisatest" (11) + "Threewordshere" (14) = 35
	if stats.Characters != 35 {
		t.Errorf("Expected 35 characters (no whitespace), got %d", stats.Characters)
	}
}

// TestCountDocumentStatsEmpty verifies empty document returns zero stats.
func TestCountDocumentStatsEmpty(t *testing.T) {
	stats := CountDocumentStats([]string{})
	if stats.Words != 0 || stats.Characters != 0 || stats.Lines != 0 || stats.ReadingMins != 0 {
		t.Errorf("Expected all zeros for empty doc, got %+v", stats)
	}
}

// TestCountDocumentStatsReadingTime verifies reading time calculation.
func TestCountDocumentStatsReadingTime(t *testing.T) {
	// 200 words → 1 min (200/200 = 1)
	words200 := make([]string, 200)
	for i := range words200 {
		words200[i] = "word"
	}
	stats := CountDocumentStats(words200)
	if stats.ReadingMins != 1 {
		t.Errorf("Expected 1 min for 200 words, got %d", stats.ReadingMins)
	}
	// 400 words → 2 min
	words400 := make([]string, 400)
	for i := range words400 {
		words400[i] = "word"
	}
	stats2 := CountDocumentStats(words400)
	if stats2.ReadingMins != 2 {
		t.Errorf("Expected 2 min for 400 words, got %d", stats2.ReadingMins)
	}
}

// TestWordCountModalOpensWithCtrlI verifies Ctrl+I opens the word count modal.
func TestWordCountModalOpensWithCtrlI(t *testing.T) {
	doc := createTestDocument([]string{})
	v := New(doc, "test.md", theme.NewTheme(), 80)
	v.Width = 80
	v.Height = 24
	v.Lines = []string{"Hello world", "Second line"}

	if v.wordCountVisible {
		t.Error("Expected wordCountVisible=false initially")
	}

	// Send Ctrl+I
	model, _ := v.Update(tea.KeyMsg{Type: tea.KeyCtrlW})
	result := model.(*Viewer)
	if !result.wordCountVisible {
		t.Error("Expected wordCountVisible=true after Ctrl+I")
	}
}

// TestWordCountModalClosesWithEsc verifies Esc closes the word count modal.
func TestWordCountModalClosesWithEsc(t *testing.T) {
	doc := createTestDocument([]string{})
	v := New(doc, "test.md", theme.NewTheme(), 80)
	v.Width = 80
	v.Height = 24
	v.wordCountVisible = true

	model, _ := v.Update(tea.KeyMsg{Type: tea.KeyEsc})
	result := model.(*Viewer)
	if result.wordCountVisible {
		t.Error("Expected wordCountVisible=false after Esc")
	}
}

// TestWordCountModalRendersStats verifies the modal contains stat labels.
func TestWordCountModalRendersStats(t *testing.T) {
	doc := createTestDocument([]string{})
	v := New(doc, "test.md", theme.NewTheme(), 80)
	v.Width = 80
	v.Height = 24
	v.Lines = []string{"Hello world", "This is a test line."}

	output := v.renderWordCount()
	if !strings.Contains(output, "Word Count") {
		t.Error("Expected 'Word Count' heading in modal")
	}
	if !strings.Contains(output, "Words:") {
		t.Error("Expected 'Words:' label in modal")
	}
	if !strings.Contains(output, "Characters:") {
		t.Error("Expected 'Characters:' label in modal")
	}
	if !strings.Contains(output, "Lines:") {
		t.Error("Expected 'Lines:' label in modal")
	}
	if !strings.Contains(output, "Reading time:") {
		t.Error("Expected 'Reading time:' label in modal")
	}
	if !strings.Contains(output, "Esc: close") {
		t.Error("Expected 'Esc: close' instruction in modal")
	}
}

// TestWordCountModalRoutedInView verifies View() shows modal when wordCountVisible=true.
func TestWordCountModalRoutedInView(t *testing.T) {
	doc := createTestDocument([]string{})
	v := New(doc, "test.md", theme.NewTheme(), 80)
	v.Width = 80
	v.Height = 24
	v.Lines = []string{"Hello world"}

	// Without modal: should not contain word count content
	view := v.View()
	if strings.Contains(view, "Word Count") {
		t.Error("Expected no 'Word Count' in view when modal is closed")
	}

	// With modal: should contain word count content
	v.wordCountVisible = true
	view = v.View()
	if !strings.Contains(view, "Word Count") {
		t.Error("Expected 'Word Count' in view when modal is open")
	}
}

// --- Auto-Save and Crash Recovery Tests (30-06) ---

// TestAutoSaveFilePathHelper verifies autoSaveFilePath generates the correct path.
func TestAutoSaveFilePathHelper(t *testing.T) {
	cases := []struct {
		input    string
		expected string
	}{
		{"/tmp/notes.md", "/tmp/.bmd-autosave-notes.md"},
		{"/home/user/docs/readme.md", "/home/user/docs/.bmd-autosave-readme.md"},
		{"", ""},
	}
	for _, tc := range cases {
		got := autoSaveFilePath(tc.input)
		if got != tc.expected {
			t.Errorf("autoSaveFilePath(%q): got %q, want %q", tc.input, got, tc.expected)
		}
	}
}

// TestAutoSaveCreatesFile verifies AutoSave() writes the autosave file.
func TestAutoSaveCreatesFile(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "test.md")
	if err := os.WriteFile(filePath, []byte("# Hello\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	doc := createTestDocument([]string{})
	v := New(doc, filePath, theme.NewTheme(), 80)
	v.autoSaveEnabled = true // explicitly enable regardless of config file on disk
	v.editMode = true
	v.editBuffer = editor.NewTextBuffer([]string{"# Hello", "edited line"})

	v.AutoSave()

	autoPath := autoSaveFilePath(filePath)
	data, err := os.ReadFile(autoPath)
	if err != nil {
		t.Fatalf("autosave file not created: %v", err)
	}
	if !strings.Contains(string(data), "edited line") {
		t.Errorf("autosave file missing expected content; got: %s", string(data))
	}
}

// TestAutoSaveNoopWhenDisabled verifies AutoSave() does nothing when disabled.
func TestAutoSaveNoopWhenDisabled(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "test.md")
	if err := os.WriteFile(filePath, []byte("# Hello\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	doc := createTestDocument([]string{})
	v := New(doc, filePath, theme.NewTheme(), 80)
	v.autoSaveEnabled = false
	v.editMode = true
	v.editBuffer = editor.NewTextBuffer([]string{"# Hello", "edited"})

	v.AutoSave()

	autoPath := autoSaveFilePath(filePath)
	if _, err := os.Stat(autoPath); !os.IsNotExist(err) {
		t.Error("autosave file should not be created when auto-save is disabled")
	}
}

// TestAutoSaveDeletedOnCtrlS verifies autosave file is removed after explicit save.
func TestAutoSaveDeletedOnCtrlS(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "test.md")
	if err := os.WriteFile(filePath, []byte("# Hello\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	doc := createTestDocument([]string{})
	v := New(doc, filePath, theme.NewTheme(), 80)
	v.editMode = true
	v.editBuffer = editor.NewTextBuffer([]string{"# Hello", "edited"})

	// Create an autosave file manually.
	autoPath := autoSaveFilePath(filePath)
	if err := os.WriteFile(autoPath, []byte("# Hello\nedited\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	// Simulate Ctrl+S
	result, _ := v.Update(tea.KeyMsg{Type: tea.KeyCtrlS})
	vv := result.(*Viewer)

	// Autosave file should be gone.
	if _, err := os.Stat(autoPath); !os.IsNotExist(err) {
		t.Error("expected autosave file to be deleted after Ctrl+S save")
	}
	if !strings.Contains(vv.errorMsg, "Saved") {
		t.Errorf("expected 'Saved' status message; got %q", vv.errorMsg)
	}
}

// TestCrashRecoveryDetected verifies checkAutoSaveRecovery sets recoveryAvailable when
// an autosave file is newer than the main file.
func TestCrashRecoveryDetected(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "test.md")

	// Write the main file first.
	if err := os.WriteFile(filePath, []byte("# Hello\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	// Wait a moment so the autosave file has a clearly newer mtime.
	time.Sleep(10 * time.Millisecond)

	// Write a newer autosave file.
	autoPath := autoSaveFilePath(filePath)
	if err := os.WriteFile(autoPath, []byte("# Hello\nrecovered content\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	doc := createTestDocument([]string{})
	v := New(doc, filePath, theme.NewTheme(), 80)
	v.autoSavePath = autoPath

	v.checkAutoSaveRecovery(filePath)

	if !v.recoveryAvailable {
		t.Error("expected recoveryAvailable=true when autosave file is newer")
	}
	if !strings.Contains(v.recoveryContent, "recovered content") {
		t.Errorf("unexpected recovery content: %q", v.recoveryContent)
	}
	if !strings.Contains(strings.ToLower(v.errorMsg), "autosave") {
		t.Errorf("expected autosave prompt in errorMsg, got %q", v.errorMsg)
	}
}

// TestCrashRecoveryNotDetectedWhenOlder verifies no recovery when the main file is newer.
func TestCrashRecoveryNotDetectedWhenOlder(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "test.md")
	autoPath := autoSaveFilePath(filePath)

	// Write the autosave first, then the main file (so main is newer).
	if err := os.WriteFile(autoPath, []byte("# Hello\nstale autosave\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	time.Sleep(10 * time.Millisecond)
	if err := os.WriteFile(filePath, []byte("# Hello\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	doc := createTestDocument([]string{})
	v := New(doc, filePath, theme.NewTheme(), 80)
	v.checkAutoSaveRecovery(filePath)

	if v.recoveryAvailable {
		t.Error("expected recoveryAvailable=false when autosave file is older than main file")
	}
}

// TestRecoveryKeyRestoresContent verifies 'r' key loads recovery content into edit mode.
func TestRecoveryKeyRestoresContent(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "test.md")
	if err := os.WriteFile(filePath, []byte("# Hello\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	doc := createTestDocument([]string{})
	v := New(doc, filePath, theme.NewTheme(), 80)
	v.Height = 24
	v.Width = 80

	// Simulate recovery state.
	v.recoveryAvailable = true
	v.recoveryContent = "# Hello\nrecovered line\n"

	// Write autosave file so deleteAutoSave() has something to remove.
	autoPath := autoSaveFilePath(filePath)
	_ = os.WriteFile(autoPath, []byte(v.recoveryContent), 0o600)

	// Press 'r'
	result, _ := v.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})
	vv := result.(*Viewer)

	if !vv.editMode {
		t.Error("expected edit mode to be activated after recovery")
	}
	if vv.recoveryAvailable {
		t.Error("expected recoveryAvailable=false after recovery")
	}
	lines := vv.editBuffer.GetLines()
	found := false
	for _, l := range lines {
		if strings.Contains(l, "recovered line") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected recovered content in buffer; lines: %v", lines)
	}
	// Autosave file should be gone.
	if _, err := os.Stat(autoPath); !os.IsNotExist(err) {
		t.Error("expected autosave file deleted after recovery")
	}
}

// TestDiscardKeyRemovesAutosave verifies 'd' key deletes the autosave file without recovery.
func TestDiscardKeyRemovesAutosave(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "test.md")
	if err := os.WriteFile(filePath, []byte("# Hello\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	doc := createTestDocument([]string{})
	v := New(doc, filePath, theme.NewTheme(), 80)
	v.Height = 24
	v.Width = 80

	v.recoveryAvailable = true
	v.recoveryContent = "# Hello\nrecovered line\n"

	autoPath := autoSaveFilePath(filePath)
	_ = os.WriteFile(autoPath, []byte(v.recoveryContent), 0o600)

	// Press 'd'
	result, _ := v.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	vv := result.(*Viewer)

	if vv.editMode {
		t.Error("expected edit mode NOT activated after discard")
	}
	if vv.recoveryAvailable {
		t.Error("expected recoveryAvailable=false after discard")
	}
	if _, err := os.Stat(autoPath); !os.IsNotExist(err) {
		t.Error("expected autosave file deleted after discard")
	}
}

// --- Outline in Edit Mode Tests (30-07) ---

// TestExtractEditHeadings verifies headings are extracted from buffer lines.
func TestExtractEditHeadings(t *testing.T) {
	doc := createTestDocument([]string{})
	v := New(doc, "test.md", theme.NewTheme(), 80)
	v.Height = 24
	v.Width = 80
	v.editMode = true
	v.editBuffer = editor.NewTextBuffer([]string{
		"# First Heading",
		"Some paragraph text.",
		"## Second Heading",
		"More text.",
		"### Third Level",
		"#notaheading",
		"###### Sixth Level",
	})

	headings := v.extractEditHeadings()

	if len(headings) != 4 {
		t.Fatalf("expected 4 headings, got %d", len(headings))
	}
	if headings[0].Level != 1 || headings[0].Text != "First Heading" || headings[0].LineIdx != 0 {
		t.Errorf("heading[0] wrong: %+v", headings[0])
	}
	if headings[1].Level != 2 || headings[1].Text != "Second Heading" || headings[1].LineIdx != 2 {
		t.Errorf("heading[1] wrong: %+v", headings[1])
	}
	if headings[2].Level != 3 || headings[2].Text != "Third Level" || headings[2].LineIdx != 4 {
		t.Errorf("heading[2] wrong: %+v", headings[2])
	}
	if headings[3].Level != 6 || headings[3].Text != "Sixth Level" || headings[3].LineIdx != 6 {
		t.Errorf("heading[3] wrong: %+v", headings[3])
	}
}

// TestExtractEditHeadingsNilBuffer returns nil when no edit buffer is set.
func TestExtractEditHeadingsNilBuffer(t *testing.T) {
	doc := createTestDocument([]string{})
	v := New(doc, "test.md", theme.NewTheme(), 80)
	v.editBuffer = nil

	headings := v.extractEditHeadings()
	if headings != nil {
		t.Errorf("expected nil headings with no buffer, got %v", headings)
	}
}

// TestEditModeOutlineOpensWithCtrlO verifies Ctrl+O opens outline in edit mode.
func TestEditModeOutlineOpensWithCtrlO(t *testing.T) {
	doc := createTestDocument([]string{})
	v := New(doc, "test.md", theme.NewTheme(), 80)
	v.Height = 24
	v.Width = 80
	v.editMode = true
	v.editBuffer = editor.NewTextBuffer([]string{
		"# Intro",
		"paragraph",
		"## Details",
	})

	model, _ := v.Update(tea.KeyMsg{Type: tea.KeyCtrlO})
	result := model.(*Viewer)

	if !result.outlineMode {
		t.Error("expected outlineMode=true after Ctrl+O in edit mode")
	}
	if len(result.outlineHeadings) != 2 {
		t.Errorf("expected 2 headings, got %d", len(result.outlineHeadings))
	}
	if result.outlineSelection != 0 {
		t.Error("expected outlineSelection reset to 0")
	}
}

// TestEditModeOutlineJumpsToHeading verifies Enter in outline sets cursor line.
func TestEditModeOutlineJumpsToHeading(t *testing.T) {
	doc := createTestDocument([]string{})
	v := New(doc, "test.md", theme.NewTheme(), 80)
	v.Height = 24
	v.Width = 80
	v.editMode = true
	v.editBuffer = editor.NewTextBuffer([]string{
		"# First",
		"text",
		"text",
		"## Second",
		"more text",
	})
	// Open outline and select second heading
	v.outlineMode = true
	v.outlineHeadings = v.extractEditHeadings()
	v.outlineSelection = 1 // select "## Second" at line 3

	model, _ := v.Update(tea.KeyMsg{Type: tea.KeyEnter})
	result := model.(*Viewer)

	if result.outlineMode {
		t.Error("expected outline closed after Enter")
	}
	// Cursor should be at line 3 (0-based)
	if result.editBuffer.CursorLine() != 3 {
		t.Errorf("expected cursor at line 3, got %d", result.editBuffer.CursorLine())
	}
	// Offset should be set to the heading line
	if result.Offset != 3 {
		t.Errorf("expected scroll offset=3, got %d", result.Offset)
	}
}

// TestEditModeOutlineEscPreservesEdits verifies Esc does not clear the edit buffer.
func TestEditModeOutlineEscPreservesEdits(t *testing.T) {
	doc := createTestDocument([]string{})
	v := New(doc, "test.md", theme.NewTheme(), 80)
	v.Height = 24
	v.Width = 80
	v.editMode = true
	v.editBuffer = editor.NewTextBuffer([]string{"# Hello", "modified line"})
	v.editBuffer.Insert('X') // make a modification
	v.outlineMode = true
	v.outlineHeadings = v.extractEditHeadings()
	v.outlineSelection = 0

	model, _ := v.Update(tea.KeyMsg{Type: tea.KeyEsc})
	result := model.(*Viewer)

	if result.outlineMode {
		t.Error("expected outline closed after Esc")
	}
	// Edit buffer should still be present with content intact
	if result.editBuffer == nil {
		t.Error("expected edit buffer to be preserved after Esc")
	}
	lines := result.editBuffer.GetLines()
	if len(lines) == 0 {
		t.Error("expected edit buffer content preserved after Esc")
	}
	// editMode should still be on
	if !result.editMode {
		t.Error("expected editMode still active after outline Esc")
	}
}

// TestOutlineNavigationUpDown verifies arrow keys navigate heading list.
func TestOutlineNavigationUpDown(t *testing.T) {
	doc := createTestDocument([]string{})
	v := New(doc, "test.md", theme.NewTheme(), 80)
	v.Height = 24
	v.Width = 80
	v.outlineMode = true
	v.outlineHeadings = []HeadingInfo{
		{Level: 1, Text: "A", LineIdx: 0},
		{Level: 2, Text: "B", LineIdx: 5},
		{Level: 3, Text: "C", LineIdx: 10},
	}
	v.outlineSelection = 0

	// Down
	model, _ := v.Update(tea.KeyMsg{Type: tea.KeyDown})
	result := model.(*Viewer)
	if result.outlineSelection != 1 {
		t.Errorf("expected selection=1 after Down, got %d", result.outlineSelection)
	}

	// Up
	result.outlineMode = true
	model2, _ := result.Update(tea.KeyMsg{Type: tea.KeyUp})
	result2 := model2.(*Viewer)
	if result2.outlineSelection != 0 {
		t.Errorf("expected selection=0 after Up, got %d", result2.outlineSelection)
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// TestViewerEndToEnd* — Plan 32-04 (D-08, ARCH-03 full, ARCH-05 integration
// half): Viewer-level integration tests that drive Viewer.Update() through
// full key-sequences and assert on final *Viewer state, independent of
// internal model structure. These are the regression oracle proving the
// message-passing contract (messages.go + activeChild dispatch, established
// in Plan 01 and reused unchanged by Plans 02/03) works end to end once
// DirectoryModel/CrossSearchModel/GraphModel are wired together — not just
// per-model in isolation (which directory_test.go/crosssearch_test.go/
// graph_test.go already cover).
//
// "Program step" resolution: every key press below goes through
// pressKeySettled (directory_test.go), which sends the tea.KeyMsg to
// v.Update() and resolves any single returned tea.Cmd into its tea.Msg,
// feeding it back into v.Update() exactly once (settleCmd). This is
// deliberately single-hop, not a resolution loop — see settleCmd's doc
// comment for why (a resolved statusMsg handler always schedules a real
// tea.Tick timer that must not be invoked synchronously in a test). Every
// transition exercised in this section (openFileMsg, switchModeMsg,
// toggleHelpMsg) fully applies its state mutation on the first hop, so
// single-hop resolution is sufficient to drive each sequence to
// quiescence — confirmed by asserting the expected terminal state after
// each step below, not just after the final one.
// ═══════════════════════════════════════════════════════════════════════════

// runeKey is a small convenience wrapper for constructing a single-rune
// tea.KeyMsg, matching the tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)}
// convention already used throughout directory_test.go/graph_test.go.
func runeKey(s string) tea.KeyMsg {
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)}
}

// newE2EGraphDir builds a temp directory containing two real markdown files
// (a.md, b.md) plus a knowledge.db (via graph_test.go's buildTestKnowledgeDB
// helper) whose two nodes' IDs are exactly those two relative filenames — so
// NewGraphModel's synchronous SQLite-read constructor (Plan 03) and
// DirectoryModel's file scan (Plan 01) both operate on a consistent fixture.
func newE2EGraphDir(t *testing.T) string {
	t.Helper()
	dir := makeTempDir(t, map[string]string{
		"a.md": "# A\nFirst document.",
		"b.md": "# B\nSecond document.",
	})
	g := newTestGraph([]knowledge.Node{
		{ID: "a.md", Title: "A", Type: "document"},
		{ID: "b.md", Title: "B", Type: "document"},
	}, nil)
	buildTestKnowledgeDB(t, dir, g)
	return dir
}

// enterCrossSearchResults drives directory -> '/' -> type query -> enter
// through to the cross-search results stage, returning the settled Viewer.
// Shared by the open-result sequence test and the help-from-cross-search-
// results test to avoid duplicating the setup.
func enterCrossSearchResults(t *testing.T, v *Viewer, query string) *Viewer {
	t.Helper()
	v = pressKeySettled(v, runeKey("/"))
	if csModel(v) == nil || csModel(v).stage != csStageInput {
		t.Fatalf("expected CrossSearchModel input stage active after '/', got %#v", csModel(v))
	}
	for _, ch := range query {
		v = pressKeySettled(v, runeKey(string(ch)))
	}
	if csModel(v).input != query {
		t.Fatalf("expected input=%q after typing, got %q", query, csModel(v).input)
	}
	v = pressKeySettled(v, tea.KeyMsg{Type: tea.KeyEnter})
	if csModel(v) == nil || csModel(v).stage != csStageResults {
		t.Fatalf("expected CrossSearchModel results stage active after Enter, got %#v", csModel(v))
	}
	return v
}

// TestViewerEndToEnd_DirectoryToCrossSearchToOpenResult drives the full
// key-sequence: directory -> '/' -> type query -> enter -> select result ->
// file opens (VALIDATION.md Wave 0, sequence 1).
func TestViewerEndToEnd_DirectoryToCrossSearchToOpenResult(t *testing.T) {
	dir := makeTempDir(t, map[string]string{
		"auth.md": "# Authentication\nJWT tokens and OAuth flow for auth.",
	})
	defer os.RemoveAll(dir)

	v := NewDirectoryViewer(dir, theme.NewTheme(), 100)
	v.Height = 24
	if err := v.LoadDirectory(dir); err != nil {
		t.Fatalf("LoadDirectory error: %v", err)
	}
	if v.currentView != "directory" || dirModel(v) == nil {
		t.Fatalf("expected directory mode active at start, currentView=%q dirModel=%v", v.currentView, dirModel(v))
	}

	v = enterCrossSearchResults(t, v, "authentication")
	if len(csModel(v).results) == 0 {
		t.Fatal("expected at least 1 search result for 'authentication'")
	}

	// Select the first (only) result and open it.
	v = pressKeySettled(v, tea.KeyMsg{Type: tea.KeyEnter})

	if v.currentView != "file" {
		t.Errorf("expected currentView='file', got %q", v.currentView)
	}
	if filepath.Base(v.FilePath) != "auth.md" {
		t.Errorf("expected FilePath basename 'auth.md', got %q", v.FilePath)
	}
	if !v.openedFromSearch {
		t.Error("expected openedFromSearch=true")
	}
	if v.activeChild != nil {
		t.Errorf("expected activeChild=nil after file open, got %T", v.activeChild)
	}
}

// TestViewerEndToEnd_DirectoryToGraphToOpenNode drives the full
// key-sequence: directory -> 'g' -> graph -> navigate -> enter -> node file
// opens (VALIDATION.md Wave 0, sequence 2).
func TestViewerEndToEnd_DirectoryToGraphToOpenNode(t *testing.T) {
	dir := newE2EGraphDir(t)
	defer os.RemoveAll(dir)

	v := NewDirectoryViewer(dir, theme.NewTheme(), 100)
	v.Height = 24
	if err := v.LoadDirectory(dir); err != nil {
		t.Fatalf("LoadDirectory error: %v", err)
	}

	v = pressKeySettled(v, runeKey("g"))
	gm, ok := v.activeChild.(*GraphModel)
	if !ok {
		t.Fatalf("expected activeChild=*GraphModel after 'g', got %T", v.activeChild)
	}
	if !gm.state.Loaded {
		t.Fatal("expected graph to be loaded")
	}
	if gm.state.SelectedNodeID != "a.md" {
		t.Fatalf("expected default selection 'a.md' (alphabetically first, tied in-degree), got %q", gm.state.SelectedNodeID)
	}

	// Navigate to the second node.
	v = pressKeySettled(v, tea.KeyMsg{Type: tea.KeyDown})
	gm = v.activeChild.(*GraphModel)
	if gm.state.SelectedNodeID != "b.md" {
		t.Fatalf("expected selection 'b.md' after Down, got %q", gm.state.SelectedNodeID)
	}

	// Open the selected node's file.
	v = pressKeySettled(v, tea.KeyMsg{Type: tea.KeyEnter})

	if v.currentView != "file" {
		t.Errorf("expected currentView='file', got %q", v.currentView)
	}
	wantPath := filepath.Join(dir, "b.md")
	if v.FilePath != wantPath {
		t.Errorf("expected FilePath=%q, got %q", wantPath, v.FilePath)
	}
	if v.activeChild != nil {
		t.Errorf("expected activeChild=nil after file open, got %T", v.activeChild)
	}
	// origin=originGraph sets neither openedFromDirectory nor openedFromSearch.
	if v.openedFromDirectory || v.openedFromSearch {
		t.Errorf("expected both openedFromDirectory/openedFromSearch=false for a graph-origin open, got dir=%v search=%v",
			v.openedFromDirectory, v.openedFromSearch)
	}
}

// TestViewerEndToEnd_HelpFromDirectory drives: directory -> '?' -> help ->
// esc -> back to directory (VALIDATION.md Wave 0, sequence 3, directory leg).
// Also asserts the focused Pitfall-5 property: while help is open, a key
// that would normally mutate the originating child (Down, in directory mode)
// is absorbed by help handling instead of leaking through to the child.
func TestViewerEndToEnd_HelpFromDirectory(t *testing.T) {
	dir := makeTempDir(t, map[string]string{"a.md": "# A", "b.md": "# B"})
	defer os.RemoveAll(dir)

	v := NewDirectoryViewer(dir, theme.NewTheme(), 100)
	v.Height = 24
	if err := v.LoadDirectory(dir); err != nil {
		t.Fatalf("LoadDirectory error: %v", err)
	}

	v = pressKeySettled(v, runeKey("?"))
	if !v.helpOpen {
		t.Fatal("expected helpOpen=true after '?'")
	}
	if dirModel(v) == nil {
		t.Fatal("expected DirectoryModel to remain the active child while help is open")
	}

	// Pitfall 5: a child-specific key (Down) must not leak through to
	// DirectoryModel while help is open — helpOpen is checked first.
	v = pressKeySettled(v, tea.KeyMsg{Type: tea.KeyDown})
	if !v.helpOpen {
		t.Error("expected helpOpen to remain true after an unrelated key while help is open")
	}
	if dirModel(v).state.SelectedIndex != 0 {
		t.Errorf("expected DirectoryModel selection unchanged (absorbed by help), got %d", dirModel(v).state.SelectedIndex)
	}

	v = pressKeySettled(v, tea.KeyMsg{Type: tea.KeyEsc})
	if v.helpOpen {
		t.Error("expected helpOpen=false after esc")
	}
	if dirModel(v) == nil {
		t.Error("expected DirectoryModel (originating mode) active again after closing help")
	}
}

// TestViewerEndToEnd_HelpFromCrossSearchResults drives: cross-search results
// -> '?' -> help -> esc -> back to cross-search results (VALIDATION.md Wave
// 0, sequence 3, cross-search leg). CrossSearchModel's results-navigation
// stage was missing a '?' case (a gap discovered writing this test — see
// SUMMARY Deviations); fixed in cross_search.go as part of this plan.
func TestViewerEndToEnd_HelpFromCrossSearchResults(t *testing.T) {
	dir := makeTempDir(t, map[string]string{
		"auth.md": "# Authentication\nJWT tokens and OAuth flow for auth.",
	})
	defer os.RemoveAll(dir)

	v := NewDirectoryViewer(dir, theme.NewTheme(), 100)
	v.Height = 24
	if err := v.LoadDirectory(dir); err != nil {
		t.Fatalf("LoadDirectory error: %v", err)
	}
	v = enterCrossSearchResults(t, v, "authentication")

	v = pressKeySettled(v, runeKey("?"))
	if !v.helpOpen {
		t.Fatal("expected helpOpen=true after '?' from cross-search results")
	}
	if csModel(v) == nil || csModel(v).stage != csStageResults {
		t.Fatal("expected CrossSearchModel to remain the active child (results stage) while help is open")
	}

	v = pressKeySettled(v, tea.KeyMsg{Type: tea.KeyEsc})
	if v.helpOpen {
		t.Error("expected helpOpen=false after esc")
	}
	if csModel(v) == nil || csModel(v).stage != csStageResults {
		t.Error("expected cross-search results (originating mode) active again after closing help")
	}
}

// TestViewerEndToEnd_HelpFromGraph drives: graph -> '?' -> help -> esc ->
// back to graph (VALIDATION.md Wave 0, sequence 3, graph leg).
func TestViewerEndToEnd_HelpFromGraph(t *testing.T) {
	dir := newE2EGraphDir(t)
	defer os.RemoveAll(dir)

	v := NewDirectoryViewer(dir, theme.NewTheme(), 100)
	v.Height = 24
	if err := v.LoadDirectory(dir); err != nil {
		t.Fatalf("LoadDirectory error: %v", err)
	}
	v = pressKeySettled(v, runeKey("g"))
	if _, ok := v.activeChild.(*GraphModel); !ok {
		t.Fatalf("expected activeChild=*GraphModel after 'g', got %T", v.activeChild)
	}

	v = pressKeySettled(v, runeKey("?"))
	if !v.helpOpen {
		t.Fatal("expected helpOpen=true after '?' from graph view")
	}
	if _, ok := v.activeChild.(*GraphModel); !ok {
		t.Fatal("expected GraphModel to remain the active child while help is open")
	}

	v = pressKeySettled(v, tea.KeyMsg{Type: tea.KeyEsc})
	if v.helpOpen {
		t.Error("expected helpOpen=false after esc")
	}
	if _, ok := v.activeChild.(*GraphModel); !ok {
		t.Error("expected GraphModel (originating mode) active again after closing help")
	}
}

// TestViewWithBrowser_InlineImage_NoRawEscapeLeak is the bmd-uc6 regression
// test: the legacy in-viewer file-browser split (v.browserOpen) truncates
// its main content column to a fixed width. A document containing an
// inline image embeds the full OSC 1337 escape sequence (base64 payload
// included) directly in that line; truncating it blindly — as the old
// padOrTruncate did — leaves an unterminated escape that corrupts the
// terminal, the same failure mode bmd-fbq fixed for the split-pane
// directory browser.
func TestViewWithBrowser_InlineImage_NoRawEscapeLeak(t *testing.T) {
	t.Setenv("TERM_PROGRAM", "iTerm.app")
	if renderer.DetectImageProtocol() == renderer.ProtocolNone {
		t.Skip("no inline image protocol detected in this environment")
	}

	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "sample.png"), []byte("not a real png but non-empty binary-ish content"), 0o644); err != nil {
		t.Fatal(err)
	}
	v := newTestFileViewer(t, dir, "readme.md", "# Image Test\n\n![Sample](./sample.png)\n", 100, 24)
	v.browserOpen = true
	v.browserFiles = []string{filepath.Join(dir, "readme.md")}

	out := v.View()

	if strings.Contains(out, "\x1b]1337") {
		t.Errorf("viewWithBrowser leaked a raw inline-image escape sequence: %q", out)
	}
	if !strings.Contains(out, "[image]") {
		t.Errorf("expected the inline-image placeholder '[image]' in browser-split output, got: %q", out)
	}
}

// TestViewWithBrowser_LargeKittyImage_SinglePlaceholderNotRepeated is a
// regression test for bmd-xqh: a large image (base64-encoded well over
// Kitty's 4096-byte-per-chunk limit — any real photo/screenshot always is)
// is transmitted as many consecutive APC escape sequences with no
// separator between them. stripInlineImageEscapes previously emitted the
// "[image]" placeholder once per matched escape sequence, so a large
// image collapsed into a long run of repeated "[image][image][image]..."
// tokens in the browser-split main content column instead of a single
// placeholder.
func TestViewWithBrowser_LargeKittyImage_SinglePlaceholderNotRepeated(t *testing.T) {
	t.Setenv("KITTY_WINDOW_ID", "1")

	dir := t.TempDir()
	imgData := make([]byte, 50000)
	for i := range imgData {
		imgData[i] = byte(i % 256)
	}
	if err := os.WriteFile(filepath.Join(dir, "big.png"), imgData, 0o644); err != nil {
		t.Fatal(err)
	}

	v := newTestFileViewer(t, dir, "readme.md", "# Test\n\n![img](./big.png)\n", 80, 24)
	v.browserOpen = true
	v.browserFiles = []string{filepath.Join(dir, "readme.md")}

	out := v.View()

	// "\x1b_Ga=T" is the image transmit escape itself; "\x1b_Ga=d" is the
	// unrelated ghost-image cleanup command every header legitimately
	// carries when Kitty is detected (bmd-xqh) — only the former would be
	// a genuine leak of raw image content into the truncated main column.
	if strings.Contains(out, "\x1b_Ga=T") {
		t.Errorf("viewWithBrowser leaked a raw Kitty image-transmit escape sequence: %.200q...", out)
	}
	if got := strings.Count(out, "[image]"); got != 1 {
		t.Errorf("expected exactly 1 '[image]' placeholder for one large chunked image, got %d", got)
	}
}

// TestView_KittyProtocolImage_NoWrapCorruption is a regression test for a
// real bug found via manual testing: the main (non-split-pane) viewer's
// wrapLineToWidth() only recognized CSI escape sequences (\x1b[...m).
// Kitty's graphics protocol uses APC (\x1b_Ga=T,...;<base64>\x1b\\), which
// wrapLineToWidth treated as literal text — worse, its embedded base64
// payload incidentally contains the literal byte 'm' near the very start
// ("f=100,m=0;..."), so even the CSI-only stripANSI() used to decide
// whether wrapping was needed stopped scanning there and mis-measured the
// line as enormous, triggering a hard wrap that split the sequence (and
// the rest of the payload) across many lines of visible garbage instead
// of ever drawing an image.
func TestView_KittyProtocolImage_NoWrapCorruption(t *testing.T) {
	t.Setenv("KITTY_WINDOW_ID", "1")

	dir := t.TempDir()
	imgData := make([]byte, 50000)
	for i := range imgData {
		imgData[i] = byte(i % 256)
	}
	if err := os.WriteFile(filepath.Join(dir, "big.png"), imgData, 0o644); err != nil {
		t.Fatal(err)
	}

	v := newTestFileViewer(t, dir, "readme.md", "# Test\n\n![img](./big.png)\n", 80, 24)
	out := v.View()

	// A 50000-byte image needs chunking (see bmd-wt6), so the first chunk
	// is marked m=1 (more chunks follow), not m=0. It also now carries a
	// r= display-size constraint (bmd-xqh), so match around the dynamic
	// height value rather than hardcoding it.
	const prefixStart = "\x1b_Ga=T,f=100,r="
	if !strings.Contains(out, prefixStart) {
		t.Fatalf("expected the Kitty graphics protocol escape prefix intact in output, got: %.200q...", out)
	}
	if !strings.Contains(out, ",m=1;") {
		t.Fatalf("expected the first chunk marked m=1 (more chunks follow) intact in output, got: %.200q...", out)
	}

	// Every chunked escape sequence (\x1b_G...\x1b\\) must survive as one
	// contiguous run with no embedded newline splitting it.
	rest := out
	chunks := 0
	for {
		idx := strings.Index(rest, "\x1b_G")
		if idx == -1 {
			break
		}
		rest = rest[idx:]
		end := strings.Index(rest, "\x1b\\")
		if end == -1 {
			t.Fatalf("chunk %d: no ST terminator found", chunks)
		}
		segment := rest[:end]
		if strings.Contains(segment, "\n") {
			t.Errorf("chunk %d: Kitty escape sequence was split by a newline (wrap corruption) — %d-byte segment contains an embedded newline", chunks, len(segment))
		}
		rest = rest[end+len("\x1b\\"):]
		chunks++
	}
	if chunks < 2 {
		t.Fatalf("expected multiple chunked escape sequences for a 50000-byte image, got %d", chunks)
	}
}
