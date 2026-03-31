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
	lines      []string         // document lines (each line is the full text, no newlines)
	cursorLine int              // 0-based line index
	cursorCol  int              // 0-based column (character position in the line)
	undoRedo   *UndoRedoManager // undo/redo manager for edit history

	// Selection state: nil when no selection active.
	selectionStart *[2]int // [line, col] anchor where selection began
	selectionEnd   *[2]int // [line, col] current selection end (moves with shift+arrows)
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
// Auto-indents the new line by preserving leading whitespace from the current line.
// Also continues list markers (-, *, +, or ordered digits) on the new line.
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

	// Detect leading whitespace and list markers for auto-indent
	indent := tb.detectAutoIndent(line)
	newLine := indent + rightPart

	tb.lines = append(tb.lines[:tb.cursorLine+1], append([]string{newLine}, tb.lines[tb.cursorLine+1:]...)...)

	tb.cursorLine++
	tb.cursorCol = len([]rune(indent))
}

// detectAutoIndent returns the prefix to apply when pressing Enter on the given line.
// It preserves leading whitespace. If the line starts with a list marker (-, *, +, or N.),
// it continues the marker. If at the end of an empty list item, returns just whitespace.
func (tb *TextBuffer) detectAutoIndent(line string) string {
	if line == "" {
		return ""
	}

	runes := []rune(line)
	n := len(runes)

	// Collect leading whitespace
	wsEnd := 0
	for wsEnd < n && (runes[wsEnd] == ' ' || runes[wsEnd] == '\t') {
		wsEnd++
	}
	ws := string(runes[:wsEnd])

	// Check if the rest starts with a bullet marker (-, *, +) followed by a space
	rest := runes[wsEnd:]
	if len(rest) >= 2 && (rest[0] == '-' || rest[0] == '*' || rest[0] == '+') && rest[1] == ' ' {
		// If the rest after the marker is empty (just "- "), don't continue the list
		if len(rest) == 2 {
			return ws
		}
		return ws + string(rest[0]) + " "
	}

	// Check if the rest starts with an ordered list marker (digits followed by ". ")
	if len(rest) >= 3 {
		digitEnd := 0
		for digitEnd < len(rest) && rest[digitEnd] >= '0' && rest[digitEnd] <= '9' {
			digitEnd++
		}
		if digitEnd > 0 && digitEnd < len(rest) && rest[digitEnd] == '.' && digitEnd+1 < len(rest) && rest[digitEnd+1] == ' ' {
			// Only continue if the ordered item has content after the marker
			if digitEnd+2 < len(rest) {
				// Increment the number
				num := 0
				for _, d := range rest[:digitEnd] {
					num = num*10 + int(d-'0')
				}
				return ws + fmt.Sprintf("%d. ", num+1)
			}
			return ws
		}
	}

	return ws
}

// IndentLine inserts 2 spaces at the start of the current line.
// The cursor column is adjusted to account for the added indentation.
func (tb *TextBuffer) IndentLine() {
	if tb.cursorLine >= len(tb.lines) {
		return
	}
	tb.undoRedo.PushUndo(tb.GetLines())

	tb.lines[tb.cursorLine] = "  " + tb.lines[tb.cursorLine]
	tb.cursorCol += 2
}

// DedentLine removes up to 2 leading spaces from the start of the current line.
// The cursor column is adjusted accordingly (clamped to 0).
func (tb *TextBuffer) DedentLine() {
	if tb.cursorLine >= len(tb.lines) {
		return
	}
	line := tb.lines[tb.cursorLine]
	if line == "" {
		return
	}

	tb.undoRedo.PushUndo(tb.GetLines())

	removed := 0
	for removed < 2 && removed < len(line) && line[removed] == ' ' {
		removed++
	}

	if removed == 0 {
		// No leading spaces to remove; discard the undo snapshot
		tb.undoRedo.Undo()
		return
	}

	tb.lines[tb.cursorLine] = line[removed:]
	if tb.cursorCol >= removed {
		tb.cursorCol -= removed
	} else {
		tb.cursorCol = 0
	}
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

// CursorWordLeft moves the cursor left to the start of the previous word.
// Word characters are defined as alphanumeric + underscore [a-zA-Z0-9_].
// Behavior: skip backward over non-word chars, then skip backward over word chars.
func (tb *TextBuffer) CursorWordLeft() {
	if tb.cursorLine >= len(tb.lines) {
		return
	}
	line := []rune(tb.lines[tb.cursorLine])
	col := tb.cursorCol

	// If at start of line, wrap to end of previous line
	if col == 0 {
		if tb.cursorLine > 0 {
			tb.cursorLine--
			tb.cursorCol = len([]rune(tb.lines[tb.cursorLine]))
		}
		return
	}

	// Skip backward over non-word characters
	for col > 0 && !isWordChar(line[col-1]) {
		col--
	}

	// Skip backward over word characters
	for col > 0 && isWordChar(line[col-1]) {
		col--
	}

	tb.cursorCol = col
}

// CursorWordRight moves the cursor right to the start of the next word.
// Word characters are defined as alphanumeric + underscore [a-zA-Z0-9_].
// Behavior: skip forward over word chars, then skip forward over non-word chars.
func (tb *TextBuffer) CursorWordRight() {
	if tb.cursorLine >= len(tb.lines) {
		return
	}
	line := []rune(tb.lines[tb.cursorLine])
	col := tb.cursorCol
	lineLen := len(line)

	// If at end of line, wrap to start of next line
	if col == lineLen {
		if tb.cursorLine < len(tb.lines)-1 {
			tb.cursorLine++
			tb.cursorCol = 0
		}
		return
	}

	// Skip forward over word characters
	for col < lineLen && isWordChar(line[col]) {
		col++
	}

	// Skip forward over non-word characters
	for col < lineLen && !isWordChar(line[col]) {
		col++
	}

	tb.cursorCol = col
}

// StartSelection anchors the selection start at the current cursor position.
// If a selection is already active this resets the anchor.
func (tb *TextBuffer) StartSelection() {
	pos := [2]int{tb.cursorLine, tb.cursorCol}
	tb.selectionStart = &pos
	end := [2]int{tb.cursorLine, tb.cursorCol}
	tb.selectionEnd = &end
}

// EndSelection moves the selection end to the current cursor position.
// Does nothing if StartSelection was never called.
func (tb *TextBuffer) EndSelection() {
	if tb.selectionStart == nil {
		return
	}
	end := [2]int{tb.cursorLine, tb.cursorCol}
	tb.selectionEnd = &end
}

// ClearSelection deselects any active selection.
func (tb *TextBuffer) ClearSelection() {
	tb.selectionStart = nil
	tb.selectionEnd = nil
}

// HasSelection returns true when a non-empty selection is active.
func (tb *TextBuffer) HasSelection() bool {
	if tb.selectionStart == nil || tb.selectionEnd == nil {
		return false
	}
	return *tb.selectionStart != *tb.selectionEnd
}

// normalizeSelection returns (start, end) such that start <= end in document order.
func normalizeSelection(a, b [2]int) ([2]int, [2]int) {
	if a[0] < b[0] || (a[0] == b[0] && a[1] <= b[1]) {
		return a, b
	}
	return b, a
}

// GetSelectedText returns the text covered by the current selection.
// Returns "" when no selection is active.
func (tb *TextBuffer) GetSelectedText() string {
	if !tb.HasSelection() {
		return ""
	}
	start, end := normalizeSelection(*tb.selectionStart, *tb.selectionEnd)
	if start[0] == end[0] {
		// Single line
		if start[0] >= len(tb.lines) {
			return ""
		}
		runes := []rune(tb.lines[start[0]])
		s, e := start[1], end[1]
		if s > len(runes) {
			s = len(runes)
		}
		if e > len(runes) {
			e = len(runes)
		}
		return string(runes[s:e])
	}

	// Multi-line
	var sb strings.Builder
	// First line: from start[1] to end of line
	if start[0] < len(tb.lines) {
		runes := []rune(tb.lines[start[0]])
		if start[1] <= len(runes) {
			sb.WriteString(string(runes[start[1]:]))
		}
		sb.WriteRune('\n')
	}
	// Middle lines
	for i := start[0] + 1; i < end[0]; i++ {
		if i < len(tb.lines) {
			sb.WriteString(tb.lines[i])
			sb.WriteRune('\n')
		}
	}
	// Last line: from start to end[1]
	if end[0] < len(tb.lines) {
		runes := []rune(tb.lines[end[0]])
		e := end[1]
		if e > len(runes) {
			e = len(runes)
		}
		sb.WriteString(string(runes[:e]))
	}
	return sb.String()
}

// DeleteSelection removes the selected text and places the cursor at the selection start.
// Pushes to undo stack. Does nothing if no selection.
func (tb *TextBuffer) DeleteSelection() {
	if !tb.HasSelection() {
		return
	}
	tb.undoRedo.PushUndo(tb.GetLines())

	start, end := normalizeSelection(*tb.selectionStart, *tb.selectionEnd)
	tb.ClearSelection()

	if start[0] == end[0] {
		// Single line deletion
		runes := []rune(tb.lines[start[0]])
		s, e := start[1], end[1]
		if s > len(runes) {
			s = len(runes)
		}
		if e > len(runes) {
			e = len(runes)
		}
		tb.lines[start[0]] = string(runes[:s]) + string(runes[e:])
		tb.cursorLine = start[0]
		tb.cursorCol = start[1]
		return
	}

	// Multi-line deletion: merge first and last lines
	startRunes := []rune(tb.lines[start[0]])
	endRunes := []rune(tb.lines[end[0]])

	sc := start[1]
	if sc > len(startRunes) {
		sc = len(startRunes)
	}
	ec := end[1]
	if ec > len(endRunes) {
		ec = len(endRunes)
	}

	merged := string(startRunes[:sc]) + string(endRunes[ec:])

	// Build new lines slice
	newLines := make([]string, 0, len(tb.lines)-(end[0]-start[0]))
	newLines = append(newLines, tb.lines[:start[0]]...)
	newLines = append(newLines, merged)
	newLines = append(newLines, tb.lines[end[0]+1:]...)
	tb.lines = newLines

	tb.cursorLine = start[0]
	tb.cursorCol = start[1]
	tb.clampCursorCol()
}

// InsertText inserts a multi-character string at the current cursor position.
// If a selection is active, it is replaced by the new text.
// Pushes a single undo snapshot for the entire operation.
func (tb *TextBuffer) InsertText(text string) {
	if text == "" {
		return
	}
	if tb.HasSelection() {
		tb.DeleteSelection()
	}
	tb.undoRedo.PushUndo(tb.GetLines())

	// Split text into lines
	parts := strings.Split(text, "\n")
	if len(parts) == 1 {
		// No newlines: simple insert
		if tb.cursorLine >= len(tb.lines) {
			return
		}
		line := tb.lines[tb.cursorLine]
		runes := []rune(line)
		col := tb.cursorCol
		if col > len(runes) {
			col = len(runes)
		}
		newLine := string(runes[:col]) + text + string(runes[col:])
		tb.lines[tb.cursorLine] = newLine
		tb.cursorCol = col + len([]rune(text))
		return
	}

	// Multiple lines: split current line and insert parts
	if tb.cursorLine >= len(tb.lines) {
		return
	}
	line := tb.lines[tb.cursorLine]
	runes := []rune(line)
	col := tb.cursorCol
	if col > len(runes) {
		col = len(runes)
	}

	before := string(runes[:col])
	after := string(runes[col:])

	newLines := make([]string, 0, len(tb.lines)+len(parts)-1)
	newLines = append(newLines, tb.lines[:tb.cursorLine]...)
	newLines = append(newLines, before+parts[0])
	for i := 1; i < len(parts)-1; i++ {
		newLines = append(newLines, parts[i])
	}
	lastPart := parts[len(parts)-1]
	newLines = append(newLines, lastPart+after)
	newLines = append(newLines, tb.lines[tb.cursorLine+1:]...)

	tb.lines = newLines
	tb.cursorLine = tb.cursorLine + len(parts) - 1
	tb.cursorCol = len([]rune(lastPart))
}

// DuplicateLine inserts a copy of the current line below it and keeps the
// cursor on the original line. The operation is a single undo step.
func (tb *TextBuffer) DuplicateLine() {
	if len(tb.lines) == 0 {
		return
	}
	tb.undoRedo.PushUndo(tb.GetLines())

	lineContent := tb.lines[tb.cursorLine]
	newLines := make([]string, 0, len(tb.lines)+1)
	newLines = append(newLines, tb.lines[:tb.cursorLine+1]...)
	newLines = append(newLines, lineContent)
	newLines = append(newLines, tb.lines[tb.cursorLine+1:]...)
	tb.lines = newLines
	// Cursor stays on the original line.
}

// DeleteLine removes the current line from the buffer. Cursor moves to the
// next line (or stays on the last line when deleting at the end). If the
// buffer has only one line, the line is cleared rather than removed.
// The operation is a single undo step.
func (tb *TextBuffer) DeleteLine() {
	if len(tb.lines) == 0 {
		return
	}
	tb.undoRedo.PushUndo(tb.GetLines())

	if len(tb.lines) == 1 {
		tb.lines[0] = ""
		tb.cursorCol = 0
		return
	}

	tb.lines = append(tb.lines[:tb.cursorLine], tb.lines[tb.cursorLine+1:]...)
	if tb.cursorLine >= len(tb.lines) {
		tb.cursorLine = len(tb.lines) - 1
	}
	tb.clampCursorCol()
}

// MoveLineUp swaps the current line with the line above and moves the cursor
// up so it follows the same content. No-op when on the first line.
// The operation is a single undo step.
func (tb *TextBuffer) MoveLineUp() {
	if tb.cursorLine == 0 {
		return
	}
	tb.undoRedo.PushUndo(tb.GetLines())

	above := tb.cursorLine - 1
	tb.lines[above], tb.lines[tb.cursorLine] = tb.lines[tb.cursorLine], tb.lines[above]
	tb.cursorLine--
	tb.clampCursorCol()
}

// MoveLineDown swaps the current line with the line below and moves the cursor
// down so it follows the same content. No-op when on the last line.
// The operation is a single undo step.
func (tb *TextBuffer) MoveLineDown() {
	if tb.cursorLine >= len(tb.lines)-1 {
		return
	}
	tb.undoRedo.PushUndo(tb.GetLines())

	below := tb.cursorLine + 1
	tb.lines[tb.cursorLine], tb.lines[below] = tb.lines[below], tb.lines[tb.cursorLine]
	tb.cursorLine++
	tb.clampCursorCol()
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

// FindAll returns the (line, col) positions of all non-overlapping occurrences of
// oldText in the buffer. If caseSensitive is false, matching is case-insensitive.
// If wholeWord is true, matches must be bounded by non-alphanumeric characters or
// string boundaries.
func (tb *TextBuffer) FindAll(oldText string, caseSensitive, wholeWord bool) [][2]int {
	if oldText == "" {
		return nil
	}

	query := oldText
	if !caseSensitive {
		query = strings.ToLower(query)
	}
	queryRunes := []rune(query)
	queryLen := len(queryRunes)

	var results [][2]int
	for lineIdx, line := range tb.lines {
		lineText := line
		if !caseSensitive {
			lineText = strings.ToLower(lineText)
		}
		lineRunes := []rune(lineText)
		n := len(lineRunes)

		for i := 0; i <= n-queryLen; i++ {
			if !runesEq(lineRunes[i:i+queryLen], queryRunes) {
				continue
			}
			if wholeWord {
				if i > 0 && isWordChar(lineRunes[i-1]) {
					continue
				}
				if i+queryLen < n && isWordChar(lineRunes[i+queryLen]) {
					continue
				}
			}
			results = append(results, [2]int{lineIdx, i})
			i += queryLen - 1 // non-overlapping
		}
	}
	return results
}

// Replace replaces all occurrences of oldText with newText in the buffer.
// The entire operation is pushed as a single undo snapshot.
// Returns the number of replacements made.
func (tb *TextBuffer) Replace(oldText, newText string, caseSensitive, wholeWord bool) int {
	if oldText == "" {
		return 0
	}

	matches := tb.FindAll(oldText, caseSensitive, wholeWord)
	if len(matches) == 0 {
		return 0
	}

	// Push current state to undo stack as a single operation
	tb.undoRedo.PushUndo(tb.GetLines())

	// Replace in reverse order (bottom-right to top-left) so earlier positions stay valid
	oldRunes := []rune(oldText)
	newRuneStr := newText

	for i := len(matches) - 1; i >= 0; i-- {
		lineIdx := matches[i][0]
		col := matches[i][1]
		line := tb.lines[lineIdx]
		lineRunes := []rune(line)

		// Use the original text case from the line (not the lowered version)
		before := string(lineRunes[:col])
		after := string(lineRunes[col+len(oldRunes):])
		tb.lines[lineIdx] = before + newRuneStr + after
	}

	// Clamp cursor
	if tb.cursorLine >= len(tb.lines) {
		tb.cursorLine = max(0, len(tb.lines)-1)
	}
	tb.clampCursorCol()

	return len(matches)
}

// ReplaceOne replaces the match at the given (line, col) position.
// Returns true if a replacement was made.
func (tb *TextBuffer) ReplaceOne(lineIdx, col int, oldText, newText string, caseSensitive bool) bool {
	if lineIdx < 0 || lineIdx >= len(tb.lines) || oldText == "" {
		return false
	}

	line := tb.lines[lineIdx]
	lineRunes := []rune(line)

	oldRunes := []rune(oldText)
	if col < 0 || col+len(oldRunes) > len(lineRunes) {
		return false
	}

	// Verify the text at this position actually matches
	candidate := string(lineRunes[col : col+len(oldRunes)])
	if caseSensitive {
		if candidate != oldText {
			return false
		}
	} else {
		if strings.ToLower(candidate) != strings.ToLower(oldText) {
			return false
		}
	}

	// Push undo
	tb.undoRedo.PushUndo(tb.GetLines())

	before := string(lineRunes[:col])
	after := string(lineRunes[col+len(oldRunes):])
	tb.lines[lineIdx] = before + newText + after

	tb.clampCursorCol()
	return true
}

// runesEq reports whether two rune slices are identical.
func runesEq(a, b []rune) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// isWordChar returns true if r is a letter, digit, or underscore.
func isWordChar(r rune) bool {
	return (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_'
}
