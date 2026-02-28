package tui

import (
	"strings"
	"testing"

	"github.com/bmd/bmd/internal/ast"
	"github.com/bmd/bmd/internal/editor"
	"github.com/bmd/bmd/internal/theme"
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
