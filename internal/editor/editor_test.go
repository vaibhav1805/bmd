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

// TestCursorWordLeft tests backward word-jump movement.
func TestCursorWordLeft(t *testing.T) {
	tb := NewTextBuffer([]string{"hello world foo"})

	// Start at col 15 (end of line)
	tb.SetCursorCol(15)
	tb.CursorWordLeft() // skip non-word (none), skip "foo" -> col 12
	if tb.CursorCol() != 12 {
		t.Errorf("Expected col 12 after first CursorWordLeft, got %d", tb.CursorCol())
	}

	tb.CursorWordLeft() // skip space, skip "world" -> col 6
	if tb.CursorCol() != 6 {
		t.Errorf("Expected col 6 after second CursorWordLeft, got %d", tb.CursorCol())
	}

	tb.CursorWordLeft() // skip space, skip "hello" -> col 0
	if tb.CursorCol() != 0 {
		t.Errorf("Expected col 0 after third CursorWordLeft, got %d", tb.CursorCol())
	}

	// At start of line: should not move further (only one line)
	tb.CursorWordLeft()
	if tb.CursorLine() != 0 || tb.CursorCol() != 0 {
		t.Errorf("Expected (0,0) at start, got (%d,%d)", tb.CursorLine(), tb.CursorCol())
	}
}

// TestCursorWordLeftWrapsLine tests that CursorWordLeft wraps to previous line.
func TestCursorWordLeftWrapsLine(t *testing.T) {
	tb := NewTextBuffer([]string{"line1", "line2"})
	tb.SetCursorLine(1)
	tb.SetCursorCol(0)

	tb.CursorWordLeft() // wraps to end of line1
	if tb.CursorLine() != 0 || tb.CursorCol() != 5 {
		t.Errorf("Expected (0,5) after wrap, got (%d,%d)", tb.CursorLine(), tb.CursorCol())
	}
}

// TestCursorWordRight tests forward word-jump movement.
func TestCursorWordRight(t *testing.T) {
	tb := NewTextBuffer([]string{"hello world foo"})

	// Start at col 0
	tb.CursorWordRight() // skip "hello", skip " " -> col 6
	if tb.CursorCol() != 6 {
		t.Errorf("Expected col 6 after first CursorWordRight, got %d", tb.CursorCol())
	}

	tb.CursorWordRight() // skip "world", skip " " -> col 12
	if tb.CursorCol() != 12 {
		t.Errorf("Expected col 12 after second CursorWordRight, got %d", tb.CursorCol())
	}

	tb.CursorWordRight() // skip "foo" -> col 15 (end of line)
	if tb.CursorCol() != 15 {
		t.Errorf("Expected col 15 after third CursorWordRight, got %d", tb.CursorCol())
	}

	// At end of line (single line): should not move further
	tb.CursorWordRight()
	if tb.CursorLine() != 0 || tb.CursorCol() != 15 {
		t.Errorf("Expected (0,15) at end, got (%d,%d)", tb.CursorLine(), tb.CursorCol())
	}
}

// TestCursorWordRightWrapsLine tests that CursorWordRight wraps to next line.
func TestCursorWordRightWrapsLine(t *testing.T) {
	tb := NewTextBuffer([]string{"line1", "line2"})
	// Move to end of line1
	tb.SetCursorCol(5)

	tb.CursorWordRight() // wraps to start of line2
	if tb.CursorLine() != 1 || tb.CursorCol() != 0 {
		t.Errorf("Expected (1,0) after wrap, got (%d,%d)", tb.CursorLine(), tb.CursorCol())
	}
}

// TestEnterNewLineAutoIndent tests that Enter preserves leading whitespace.
func TestEnterNewLineAutoIndent(t *testing.T) {
	tb := NewTextBuffer([]string{"  hello"})
	tb.SetCursorCol(7) // end of line

	tb.EnterNewLine()

	lines := tb.GetLines()
	if len(lines) != 2 {
		t.Fatalf("Expected 2 lines, got %d", len(lines))
	}
	if lines[1] != "  " {
		t.Errorf("Expected '  ' (2 spaces indent), got %q", lines[1])
	}
	if tb.CursorLine() != 1 || tb.CursorCol() != 2 {
		t.Errorf("Expected cursor at (1,2), got (%d,%d)", tb.CursorLine(), tb.CursorCol())
	}
}

// TestEnterNewLineBulletList tests that Enter continues bullet list markers.
func TestEnterNewLineBulletList(t *testing.T) {
	tb := NewTextBuffer([]string{"- item one"})
	tb.SetCursorCol(10) // end of line

	tb.EnterNewLine()

	lines := tb.GetLines()
	if len(lines) != 2 {
		t.Fatalf("Expected 2 lines, got %d", len(lines))
	}
	if lines[1] != "- " {
		t.Errorf("Expected '- ' on new line, got %q", lines[1])
	}
}

// TestEnterNewLineEmptyBullet tests that Enter on an empty bullet stops continuation.
func TestEnterNewLineEmptyBullet(t *testing.T) {
	tb := NewTextBuffer([]string{"- "})
	tb.SetCursorCol(2)

	tb.EnterNewLine()

	lines := tb.GetLines()
	if len(lines) != 2 {
		t.Fatalf("Expected 2 lines, got %d", len(lines))
	}
	if lines[1] != "" {
		t.Errorf("Expected empty new line after empty bullet, got %q", lines[1])
	}
}

// TestEnterNewLineOrderedList tests that Enter increments ordered list numbers.
func TestEnterNewLineOrderedList(t *testing.T) {
	tb := NewTextBuffer([]string{"1. first item"})
	tb.SetCursorCol(13)

	tb.EnterNewLine()

	lines := tb.GetLines()
	if len(lines) != 2 {
		t.Fatalf("Expected 2 lines, got %d", len(lines))
	}
	if lines[1] != "2. " {
		t.Errorf("Expected '2. ' on new line, got %q", lines[1])
	}
}

// TestEnterNewLineNoIndent tests that plain lines don't get unexpected indentation.
func TestEnterNewLineNoIndent(t *testing.T) {
	tb := NewTextBuffer([]string{"hello"})
	tb.SetCursorCol(5)

	tb.EnterNewLine()

	lines := tb.GetLines()
	if lines[1] != "" {
		t.Errorf("Expected empty new line, got %q", lines[1])
	}
}

// TestIndentLine tests adding 2-space indentation.
func TestIndentLine(t *testing.T) {
	tb := NewTextBuffer([]string{"hello"})
	tb.SetCursorCol(3)

	tb.IndentLine()

	lines := tb.GetLines()
	if lines[0] != "  hello" {
		t.Errorf("Expected '  hello', got %q", lines[0])
	}
	if tb.CursorCol() != 5 {
		t.Errorf("Expected cursor col 5 after indent, got %d", tb.CursorCol())
	}
}

// TestDedentLine tests removing up to 2 leading spaces.
func TestDedentLine(t *testing.T) {
	tb := NewTextBuffer([]string{"    hello"})
	tb.SetCursorCol(5)

	tb.DedentLine() // removes 2 spaces -> "  hello"

	lines := tb.GetLines()
	if lines[0] != "  hello" {
		t.Errorf("Expected '  hello', got %q", lines[0])
	}
	if tb.CursorCol() != 3 {
		t.Errorf("Expected cursor col 3 after dedent, got %d", tb.CursorCol())
	}

	tb.DedentLine() // removes 2 more -> "hello"

	lines = tb.GetLines()
	if lines[0] != "hello" {
		t.Errorf("Expected 'hello', got %q", lines[0])
	}
}

// TestDedentLineNoSpaces tests that DedentLine is a no-op when no leading spaces.
func TestDedentLineNoSpaces(t *testing.T) {
	tb := NewTextBuffer([]string{"hello"})
	tb.DedentLine()

	lines := tb.GetLines()
	if lines[0] != "hello" {
		t.Errorf("Expected 'hello' unchanged, got %q", lines[0])
	}
}

// TestDedentLineClampsCursor tests that cursor clamps to col 0 when ahead of removed spaces.
func TestDedentLineClampsCursor(t *testing.T) {
	tb := NewTextBuffer([]string{"  hello"})
	tb.SetCursorCol(1) // cursor inside the whitespace

	tb.DedentLine() // removes 2 spaces, cursor was at 1 -> clamp to 0

	if tb.CursorCol() != 0 {
		t.Errorf("Expected cursor col 0 after dedent clamp, got %d", tb.CursorCol())
	}
}

// TestIndentDedentUndo tests that indent/dedent operations are undoable.
func TestIndentDedentUndo(t *testing.T) {
	tb := NewTextBuffer([]string{"hello"})
	tb.IndentLine()

	lines := tb.GetLines()
	if lines[0] != "  hello" {
		t.Fatalf("Expected '  hello' after indent, got %q", lines[0])
	}

	tb.Undo()
	lines = tb.GetLines()
	if lines[0] != "hello" {
		t.Errorf("Expected 'hello' after undo, got %q", lines[0])
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
