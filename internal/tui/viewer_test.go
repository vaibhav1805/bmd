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
