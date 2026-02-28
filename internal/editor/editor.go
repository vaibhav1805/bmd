package editor

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

// TextBuffer represents an in-memory editable document with cursor position tracking.
type TextBuffer struct {
	lines      []string // document lines (each line is the full text, no newlines)
	cursorLine int      // 0-based line index
	cursorCol  int      // 0-based column (character position in the line)
}

// NewTextBuffer creates a TextBuffer from initial lines.
func NewTextBuffer(initialLines []string) *TextBuffer {
	lines := make([]string, len(initialLines))
	copy(lines, initialLines)
	return &TextBuffer{
		lines:      lines,
		cursorLine: 0,
		cursorCol:  0,
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
	line := tb.lines[tb.cursorLine]
	runes := []rune(line)

	leftPart := string(runes[:tb.cursorCol])
	rightPart := string(runes[tb.cursorCol:])

	tb.lines[tb.cursorLine] = leftPart
	tb.lines = append(tb.lines[:tb.cursorLine+1], append([]string{rightPart}, tb.lines[tb.cursorLine+1:]...)...)

	tb.cursorLine++
	tb.cursorCol = 0
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
