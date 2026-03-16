package editor

import (
	"io/ioutil"
	"os"
	"strings"
	"testing"
)

// TestNewTextBuffer tests TextBuffer initialization.
func TestNewTextBuffer(t *testing.T) {
	lines := []string{"line 1", "line 2", "line 3"}
	tb := NewTextBuffer(lines)

	if tb.CursorLine() != 0 || tb.CursorCol() != 0 {
		t.Errorf("Expected cursor at (0, 0), got (%d, %d)", tb.CursorLine(), tb.CursorCol())
	}

	resultLines := tb.GetLines()
	if len(resultLines) != 3 {
		t.Errorf("Expected 3 lines, got %d", len(resultLines))
	}
}

// TestInsertCharacter tests character insertion.
func TestInsertCharacter(t *testing.T) {
	tb := NewTextBuffer([]string{"hello"})
	tb.Insert('X')

	resultLines := tb.GetLines()
	if resultLines[0] != "Xhello" {
		t.Errorf("Expected 'Xhello', got '%s'", resultLines[0])
	}

	if tb.CursorCol() != 1 {
		t.Errorf("Expected cursor at col 1, got %d", tb.CursorCol())
	}
}

// TestBackspaceCharacter tests backspace deletion.
func TestBackspaceCharacter(t *testing.T) {
	tb := NewTextBuffer([]string{"hello"})
	tb.CursorRight() // Move to col 1
	tb.CursorRight() // Move to col 2
	tb.Backspace()   // Delete 'e', cursor at col 1

	resultLines := tb.GetLines()
	if resultLines[0] != "hllo" {
		t.Errorf("Expected 'hllo', got '%s'", resultLines[0])
	}

	if tb.CursorCol() != 1 {
		t.Errorf("Expected cursor at col 1, got %d", tb.CursorCol())
	}
}

// TestDeleteCharacter tests delete key.
func TestDeleteCharacter(t *testing.T) {
	tb := NewTextBuffer([]string{"hello"})
	tb.Delete() // Delete 'h' at cursor 0

	resultLines := tb.GetLines()
	if resultLines[0] != "ello" {
		t.Errorf("Expected 'ello', got '%s'", resultLines[0])
	}

	if tb.CursorCol() != 0 {
		t.Errorf("Expected cursor at col 0, got %d", tb.CursorCol())
	}
}

// TestEnterNewLine tests line break insertion.
func TestEnterNewLine(t *testing.T) {
	tb := NewTextBuffer([]string{"hello"})
	tb.CursorRight()
	tb.CursorRight() // Cursor at col 2 (between 'l' and 'l')
	tb.EnterNewLine()

	resultLines := tb.GetLines()
	if len(resultLines) != 2 {
		t.Errorf("Expected 2 lines, got %d", len(resultLines))
	}

	if resultLines[0] != "he" {
		t.Errorf("Expected 'he', got '%s'", resultLines[0])
	}

	if resultLines[1] != "llo" {
		t.Errorf("Expected 'llo', got '%s'", resultLines[1])
	}

	if tb.CursorLine() != 1 || tb.CursorCol() != 0 {
		t.Errorf("Expected cursor at (1, 0), got (%d, %d)", tb.CursorLine(), tb.CursorCol())
	}
}

// TestCursorMovement tests arrow key navigation.
func TestCursorMovement(t *testing.T) {
	tb := NewTextBuffer([]string{"line1", "line2", "line3"})

	tb.CursorDown()
	if tb.CursorLine() != 1 {
		t.Errorf("Expected line 1 after CursorDown, got %d", tb.CursorLine())
	}

	tb.CursorUp()
	if tb.CursorLine() != 0 {
		t.Errorf("Expected line 0 after CursorUp, got %d", tb.CursorLine())
	}

	tb.CursorRight()
	tb.CursorRight()
	if tb.CursorCol() != 2 {
		t.Errorf("Expected col 2 after two CursorRight, got %d", tb.CursorCol())
	}

	tb.CursorLeft()
	if tb.CursorCol() != 1 {
		t.Errorf("Expected col 1 after CursorLeft, got %d", tb.CursorCol())
	}
}

// TestUndoSingleEdit tests undo of a single edit.
func TestUndoSingleEdit(t *testing.T) {
	tb := NewTextBuffer([]string{"hello"})
	tb.Insert('X')

	// Undo the insert
	tb.Undo()

	resultLines := tb.GetLines()
	if resultLines[0] != "hello" {
		t.Errorf("Expected 'hello' after undo, got '%s'", resultLines[0])
	}
}

// TestUndoMultipleEdits tests undo of multiple edits.
func TestUndoMultipleEdits(t *testing.T) {
	tb := NewTextBuffer([]string{"hello"})

	tb.Insert('A')
	tb.Insert('B')
	tb.Insert('C')

	// Undo three times
	tb.Undo()
	tb.Undo()
	tb.Undo()

	resultLines := tb.GetLines()
	if resultLines[0] != "hello" {
		t.Errorf("Expected 'hello' after three undos, got '%s'", resultLines[0])
	}
}

// TestRedoAfterUndo tests redo after undo.
func TestRedoAfterUndo(t *testing.T) {
	tb := NewTextBuffer([]string{"hello"})

	tb.Insert('X')
	tb.Undo()
	tb.Redo()

	resultLines := tb.GetLines()
	if resultLines[0] != "Xhello" {
		t.Errorf("Expected 'Xhello' after undo then redo, got '%s'", resultLines[0])
	}
}

// TestRedoClearedOnNewEdit tests that redo stack is cleared on new edit.
func TestRedoClearedOnNewEdit(t *testing.T) {
	tb := NewTextBuffer([]string{"hello"})

	tb.Insert('X')        // snapshot ["hello"] pushed, insert at col 0 -> "Xhello", col moves to 1
	tb.Undo()             // restore to ["hello"], cursor stays at col 1
	tb.Insert('Y')        // snapshot ["hello"] pushed, insert at col 1 -> "hYello", col moves to 2

	if tb.CanRedo() {
		t.Error("Expected redo stack to be cleared after new edit")
	}

	resultLines := tb.GetLines()
	// After undo, cursor is still at col 1; inserting 'Y' at col 1 gives "hYello"
	if resultLines[0] != "hYello" {
		t.Errorf("Expected 'hYello', got '%s'", resultLines[0])
	}
}

// TestUndoRedoManagerBasic tests basic undo/redo manager operations.
func TestUndoRedoManagerBasic(t *testing.T) {
	urm := NewUndoRedoManager()

	state1 := []string{"line1", "line2"}
	urm.Push(state1)

	if !urm.CanUndo() {
		t.Error("Expected CanUndo to be true after Push")
	}

	undoState := urm.Undo()
	if undoState == nil {
		t.Error("Expected Undo to return non-nil state")
	}

	if undoState[0] != "line1" || undoState[1] != "line2" {
		t.Errorf("Expected ['line1', 'line2'], got %v", undoState)
	}
}

// TestUndoRedoStackClearing tests that redo stack is cleared on new push.
func TestUndoRedoStackClearing(t *testing.T) {
	urm := NewUndoRedoManager()

	state1 := []string{"line1"}
	state2 := []string{"line2"}

	urm.Push(state1)
	undoState := urm.Undo()
	urm.PushRedo(undoState)

	if !urm.CanRedo() {
		t.Error("Expected CanRedo to be true after PushRedo")
	}

	// New push should clear redo stack
	urm.Push(state2)

	if urm.CanRedo() {
		t.Error("Expected CanRedo to be false after new Push")
	}
}

// TestSaveToFile tests file persistence.
func TestSaveToFile(t *testing.T) {
	// Create temporary test file
	tmpFile, err := ioutil.TempFile("", "test-*.md")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	tmpPath := tmpFile.Name()
	tmpFile.Close()
	defer os.Remove(tmpPath)

	// Create a TextBuffer and save
	tb := NewTextBuffer([]string{"line 1", "line 2", "line 3"})
	tb.Insert('X') // Modify the buffer

	err = tb.SaveToFile(tmpPath)
	if err != nil {
		t.Fatalf("SaveToFile failed: %v", err)
	}

	// Read the file back
	content, err := ioutil.ReadFile(tmpPath)
	if err != nil {
		t.Fatalf("Failed to read temp file: %v", err)
	}

	lines := strings.Split(string(content), "\n")
	// Remove trailing empty line if present
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}

	resultLines := tb.GetLines()
	if len(lines) != len(resultLines) {
		t.Errorf("Expected %d lines in file, got %d", len(resultLines), len(lines))
	}

	if len(lines) > 0 && lines[0] != resultLines[0] {
		t.Errorf("Expected first line '%s', got '%s'", resultLines[0], lines[0])
	}
}

// TestCursorWrapping tests vim-like cursor wrapping.
func TestCursorWrappingLeft(t *testing.T) {
	tb := NewTextBuffer([]string{"line1", "line2"})
	tb.CursorDown() // Move to line 2
	tb.CursorLeft() // Try to move left from start of line 2, should wrap to end of line 1

	if tb.CursorLine() != 0 {
		t.Errorf("Expected to wrap to line 0, got %d", tb.CursorLine())
	}

	if tb.CursorCol() != 5 {
		t.Errorf("Expected to wrap to col 5 (end of 'line1'), got %d", tb.CursorCol())
	}
}

// TestCursorWrappingRight tests vim-like cursor wrapping at line end.
func TestCursorWrappingRight(t *testing.T) {
	tb := NewTextBuffer([]string{"line1", "line2"})
	// Move to end of line 1
	for i := 0; i < 5; i++ {
		tb.CursorRight()
	}

	tb.CursorRight() // Move right from end of line 1, should wrap to start of line 2

	if tb.CursorLine() != 1 {
		t.Errorf("Expected to wrap to line 1, got %d", tb.CursorLine())
	}

	if tb.CursorCol() != 0 {
		t.Errorf("Expected to wrap to col 0 (start of 'line2'), got %d", tb.CursorCol())
	}
}

// TestInsertMultipleCharacters tests inserting multiple characters.
func TestInsertMultipleCharacters(t *testing.T) {
	tb := NewTextBuffer([]string{"abc"})

	tb.Insert('X')
	tb.Insert('Y')
	tb.Insert('Z')

	resultLines := tb.GetLines()
	if resultLines[0] != "XYZabc" {
		t.Errorf("Expected 'XYZabc', got '%s'", resultLines[0])
	}

	if tb.CursorCol() != 3 {
		t.Errorf("Expected cursor at col 3, got %d", tb.CursorCol())
	}
}

// TestEmptyLineOperations tests operations on empty lines.
func TestEmptyLineOperations(t *testing.T) {
	tb := NewTextBuffer([]string{""})

	tb.Insert('A')

	resultLines := tb.GetLines()
	if resultLines[0] != "A" {
		t.Errorf("Expected 'A', got '%s'", resultLines[0])
	}

	tb.Backspace()

	resultLines = tb.GetLines()
	if resultLines[0] != "" {
		t.Errorf("Expected empty string, got '%s'", resultLines[0])
	}
}

// TestMultilineUndo tests undo with multiline content.
func TestMultilineUndo(t *testing.T) {
	tb := NewTextBuffer([]string{"line1", "line2", "line3"})

	tb.CursorDown()
	tb.EnterNewLine()

	if len(tb.GetLines()) != 4 {
		t.Errorf("Expected 4 lines after EnterNewLine, got %d", len(tb.GetLines()))
	}

	tb.Undo()

	if len(tb.GetLines()) != 3 {
		t.Errorf("Expected 3 lines after undo, got %d", len(tb.GetLines()))
	}
}

// TestDeleteAtLineEnd tests delete at end of line (joins next line).
func TestDeleteAtLineEnd(t *testing.T) {
	tb := NewTextBuffer([]string{"line1", "line2"})

	// Move to end of first line
	for i := 0; i < 5; i++ {
		tb.CursorRight()
	}

	// Delete at EOL should join next line
	tb.Delete()

	resultLines := tb.GetLines()
	if len(resultLines) != 1 {
		t.Errorf("Expected 1 line after delete at EOL, got %d", len(resultLines))
	}

	if resultLines[0] != "line1line2" {
		t.Errorf("Expected 'line1line2', got '%s'", resultLines[0])
	}
}

// TestBackspaceAtLineStart tests backspace at start of line (joins previous line).
func TestBackspaceAtLineStart(t *testing.T) {
	tb := NewTextBuffer([]string{"line1", "line2"})

	tb.CursorDown() // Move to line 2

	// Backspace at line start should join previous line
	tb.Backspace()

	resultLines := tb.GetLines()
	if len(resultLines) != 1 {
		t.Errorf("Expected 1 line after backspace at start, got %d", len(resultLines))
	}

	if resultLines[0] != "line1line2" {
		t.Errorf("Expected 'line1line2', got '%s'", resultLines[0])
	}
}
