package editor

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

// UndoRedoManager maintains undo and redo stacks of document states.
type UndoRedoManager struct {
	undoStack [][]string // Each entry is a snapshot of []string (buffer lines)
	redoStack [][]string
}

// NewUndoRedoManager creates a new undo/redo manager.
func NewUndoRedoManager() *UndoRedoManager {
	return &UndoRedoManager{
		undoStack: make([][]string, 0),
		redoStack: make([][]string, 0),
	}
}

// Push saves the current state to the undo stack and clears the redo stack.
// This is called BEFORE an edit operation, so the undo stack contains the pre-edit state.
func (urm *UndoRedoManager) Push(state []string) {
	// Deep copy the state
	snapshot := make([]string, len(state))
	copy(snapshot, state)
	urm.undoStack = append(urm.undoStack, snapshot)
	// Clear redo stack when a new edit is made
	urm.redoStack = make([][]string, 0)
}

// Undo reverts to the previous state, if available.
// Returns the pre-undo state to restore, or nil if no undo available.
func (urm *UndoRedoManager) Undo() []string {
	if len(urm.undoStack) == 0 {
		return nil
	}

	// Pop from undo stack
	undoState := urm.undoStack[len(urm.undoStack)-1]
	urm.undoStack = urm.undoStack[:len(urm.undoStack)-1]

	// Push current state to redo stack (will be provided by caller)
	// This is a bit awkward: we need the current state to push to redo
	// See the Undo method in TextBuffer below for the full interaction

	return undoState
}

// Redo reapplies a previously undone state, if available.
func (urm *UndoRedoManager) Redo() []string {
	if len(urm.redoStack) == 0 {
		return nil
	}

	// Pop from redo stack
	redoState := urm.redoStack[len(urm.redoStack)-1]
	urm.redoStack = urm.redoStack[:len(urm.redoStack)-1]

	return redoState
}

// CanUndo returns true if undo is available.
func (urm *UndoRedoManager) CanUndo() bool {
	return len(urm.undoStack) > 0
}

// CanRedo returns true if redo is available.
func (urm *UndoRedoManager) CanRedo() bool {
	return len(urm.redoStack) > 0
}

// PushRedo saves the current state to the redo stack (called by TextBuffer.Undo).
func (urm *UndoRedoManager) PushRedo(state []string) {
	snapshot := make([]string, len(state))
	copy(snapshot, state)
	urm.redoStack = append(urm.redoStack, snapshot)
}

// PushUndo saves the current state to the undo stack (called by TextBuffer before edits).
func (urm *UndoRedoManager) PushUndo(state []string) {
	urm.Push(state)
}

// TextBuffer represents an in-memory editable document with cursor position tracking.
type TextBuffer struct {
	lines      []string          // document lines (each line is the full text, no newlines)
	cursorLine int               // 0-based line index
	cursorCol  int               // 0-based column (character position in the line)
	undoRedo   *UndoRedoManager  // undo/redo manager for edit history
}

// NewTextBuffer creates a TextBuffer from initial lines.
func NewTextBuffer(initialLines []string) *TextBuffer {
	lines := make([]string, len(initialLines))
	copy(lines, initialLines)
	return &TextBuffer{
		lines:      lines,
		cursorLine: 0,
		cursorCol:  0,
		undoRedo:   NewUndoRedoManager(),
	}
}

// GetLines returns the current buffer lines (snapshot copy).
func (tb *TextBuffer) GetLines() []string {
	lines := make([]string, len(tb.lines))
	copy(lines, tb.lines)
	return lines
}

// SetLines replaces the entire buffer (used for undo/redo; see Plan 05).
func (tb *TextBuffer) SetLines(newLines []string) {
	tb.lines = make([]string, len(newLines))
	copy(tb.lines, newLines)
	// Clamp cursor to valid position
	if tb.cursorLine >= len(tb.lines) {
		tb.cursorLine = max(0, len(tb.lines)-1)
	}
	if tb.cursorLine < len(tb.lines) && tb.cursorCol > len(tb.lines[tb.cursorLine]) {
		tb.cursorCol = len(tb.lines[tb.cursorLine])
	}
}

// CursorLine returns the current line index (0-based).
func (tb *TextBuffer) CursorLine() int {
	return tb.cursorLine
}

// CursorCol returns the current column (0-based character position in line).
func (tb *TextBuffer) CursorCol() int {
	return tb.cursorCol
}

// CursorUp moves the cursor up one line, maintaining column position if possible.
func (tb *TextBuffer) CursorUp() {
	if tb.cursorLine > 0 {
		tb.cursorLine--
		tb.clampCursorCol()
	}
}

// CursorDown moves the cursor down one line, maintaining column position if possible.
func (tb *TextBuffer) CursorDown() {
	if tb.cursorLine < len(tb.lines)-1 {
		tb.cursorLine++
		tb.clampCursorCol()
	}
}

// CursorLeft moves the cursor left one character, wrapping to end of previous line if at column 0.
func (tb *TextBuffer) CursorLeft() {
	if tb.cursorCol > 0 {
		tb.cursorCol--
	} else if tb.cursorLine > 0 {
		tb.cursorLine--
		tb.cursorCol = len(tb.lines[tb.cursorLine])
	}
}

// CursorRight moves the cursor right one character, wrapping to start of next line if at end.
func (tb *TextBuffer) CursorRight() {
	if tb.cursorCol < len(tb.lines[tb.cursorLine]) {
		tb.cursorCol++
	} else if tb.cursorLine < len(tb.lines)-1 {
		tb.cursorLine++
		tb.cursorCol = 0
	}
}

// Insert inserts a rune at the current cursor position and moves cursor to the right.
func (tb *TextBuffer) Insert(r rune) {
	if tb.cursorLine >= len(tb.lines) {
		return // cursor past end of buffer
	}
	// Push current state to undo stack BEFORE making the edit
	tb.undoRedo.PushUndo(tb.GetLines())

	line := tb.lines[tb.cursorLine]
	if tb.cursorCol > len(line) {
		tb.cursorCol = len(line)
	}
	// Insert rune at cursor position
	newLine := string([]rune(line)[:tb.cursorCol]) + string(r) + string([]rune(line)[tb.cursorCol:])
	tb.lines[tb.cursorLine] = newLine
	tb.cursorCol++
}

// Delete removes the character at the cursor position (like Delete key).
// If at end of line, joins with next line.
func (tb *TextBuffer) Delete() {
	if tb.cursorLine >= len(tb.lines) {
		return
	}
	// Push current state to undo stack BEFORE making the edit
	tb.undoRedo.PushUndo(tb.GetLines())

	line := tb.lines[tb.cursorLine]
	if tb.cursorCol >= len(line) {
		// At end of line: join with next line
		if tb.cursorLine < len(tb.lines)-1 {
			tb.lines[tb.cursorLine] = line + tb.lines[tb.cursorLine+1]
			tb.lines = append(tb.lines[:tb.cursorLine+1], tb.lines[tb.cursorLine+2:]...)
		}
	} else {
		// Delete character at cursor
		runes := []rune(line)
		newLine := string(runes[:tb.cursorCol]) + string(runes[tb.cursorCol+1:])
		tb.lines[tb.cursorLine] = newLine
	}
}

// Backspace removes the character before the cursor (like Backspace key).
// If at start of line, joins with previous line.
func (tb *TextBuffer) Backspace() {
	// Push current state to undo stack BEFORE making the edit
	tb.undoRedo.PushUndo(tb.GetLines())

	if tb.cursorCol > 0 {
		// Delete character before cursor
		line := tb.lines[tb.cursorLine]
		runes := []rune(line)
		newLine := string(runes[:tb.cursorCol-1]) + string(runes[tb.cursorCol:])
		tb.lines[tb.cursorLine] = newLine
		tb.cursorCol--
	} else if tb.cursorLine > 0 {
		// At start of line: join with previous line
		prevLine := tb.lines[tb.cursorLine-1]
		tb.cursorCol = len(prevLine)
		tb.lines[tb.cursorLine-1] = prevLine + tb.lines[tb.cursorLine]
		tb.lines = append(tb.lines[:tb.cursorLine], tb.lines[tb.cursorLine+1:]...)
		tb.cursorLine--
	}
}

// EnterNewLine splits the current line at cursor and creates a new line.
func (tb *TextBuffer) EnterNewLine() {
	if tb.cursorLine >= len(tb.lines) {
		return
	}
	// Push current state to undo stack BEFORE making the edit
	tb.undoRedo.PushUndo(tb.GetLines())

	line := tb.lines[tb.cursorLine]
	runes := []rune(line)

	leftPart := string(runes[:tb.cursorCol])
	rightPart := string(runes[tb.cursorCol:])

	tb.lines[tb.cursorLine] = leftPart
	tb.lines = append(tb.lines[:tb.cursorLine+1], append([]string{rightPart}, tb.lines[tb.cursorLine+1:]...)...)

	tb.cursorLine++
	tb.cursorCol = 0
}

// JumpToStart moves the cursor to the beginning of the document.
func (tb *TextBuffer) JumpToStart() {
	tb.cursorLine = 0
	tb.cursorCol = 0
}

// JumpToEnd moves the cursor to the end of the document.
func (tb *TextBuffer) JumpToEnd() {
	if len(tb.lines) > 0 {
		tb.cursorLine = len(tb.lines) - 1
		tb.cursorCol = len(tb.lines[tb.cursorLine])
	}
}

// JumpToLine moves the cursor to a specific line (0-based).
func (tb *TextBuffer) JumpToLine(lineNum int) {
	if lineNum >= 0 && lineNum < len(tb.lines) {
		tb.cursorLine = lineNum
		tb.cursorCol = 0
	}
}

// SetCursorLine sets the cursor to a specific line (0-based).
// Clamps the column to valid range for the new line.
func (tb *TextBuffer) SetCursorLine(lineNum int) {
	if lineNum >= 0 && lineNum < len(tb.lines) {
		tb.cursorLine = lineNum
		tb.clampCursorCol()
	}
}

// SetCursorCol sets the cursor to a specific column (0-based).
// Clamps to the line length.
func (tb *TextBuffer) SetCursorCol(col int) {
	tb.cursorCol = col
	tb.clampCursorCol()
}

// Undo reverts to the previous state.
func (tb *TextBuffer) Undo() {
	undoState := tb.undoRedo.Undo()
	if undoState == nil {
		return // No undo available
	}

	// Push current state to redo stack before restoring undo state
	tb.undoRedo.PushRedo(tb.GetLines())

	// Restore the undo state
	tb.SetLines(undoState)
}

// Redo reapplies a previously undone state.
func (tb *TextBuffer) Redo() {
	redoState := tb.undoRedo.Redo()
	if redoState == nil {
		return // No redo available
	}

	// Push current state to undo stack before restoring redo state
	tb.undoRedo.PushUndo(tb.GetLines())

	// Restore the redo state
	tb.SetLines(redoState)
}

// CanUndo returns true if undo is available.
func (tb *TextBuffer) CanUndo() bool {
	return tb.undoRedo.CanUndo()
}

// CanRedo returns true if redo is available.
func (tb *TextBuffer) CanRedo() bool {
	return tb.undoRedo.CanRedo()
}

// clampCursorCol ensures the cursor column is within valid bounds for the current line.
func (tb *TextBuffer) clampCursorCol() {
	if tb.cursorLine >= len(tb.lines) {
		tb.cursorLine = len(tb.lines) - 1
	}
	if tb.cursorCol > len(tb.lines[tb.cursorLine]) {
		tb.cursorCol = len(tb.lines[tb.cursorLine])
	}
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// SaveToFile writes the buffer content to the specified file path using an atomic write pattern.
// This prevents data loss if the write is interrupted (writes to temp file, then renames).
// Returns nil on success, or an error if the write fails.
func (tb *TextBuffer) SaveToFile(filePath string) error {
	// Ensure the file path is absolute or resolve it relative to the current directory
	absPath, err := filepath.Abs(filePath)
	if err != nil {
		return fmt.Errorf("failed to resolve file path: %w", err)
	}

	// Create the directory if it doesn't exist
	dir := filepath.Dir(absPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Join all lines with newline
	content := strings.Join(tb.GetLines(), "\n")

	// Write to a temporary file in the same directory as the target file
	// This ensures the temp file is on the same filesystem for atomic rename
	tempFile, err := ioutil.TempFile(dir, ".bmd-*.tmp")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	defer tempFile.Close()

	tempPath := tempFile.Name()

	// Write the content to the temp file
	if _, err := tempFile.WriteString(content); err != nil {
		// Clean up the temp file on write failure
		os.Remove(tempPath)
		return fmt.Errorf("failed to write to temp file: %w", err)
	}

	// Ensure all data is flushed to disk
	if err := tempFile.Sync(); err != nil {
		os.Remove(tempPath)
		return fmt.Errorf("failed to sync temp file: %w", err)
	}

	// Atomically rename the temp file to the target path
	// This is atomic on most filesystems (POSIX semantics)
	if err := os.Rename(tempPath, absPath); err != nil {
		os.Remove(tempPath)
		return fmt.Errorf("failed to rename temp file to target: %w", err)
	}

	return nil
}
