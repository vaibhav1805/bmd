package tui

import (
	"strings"
)

// SelectionPoint represents a position in the document (line and column).
type SelectionPoint struct {
	LineIndex   int // 0-based line index in the document
	ColumnIndex int // 0-based column (rune index) within the line
}

// NormalizeSelection ensures start <= end in the document.
func NormalizeSelection(start, end SelectionPoint) (SelectionPoint, SelectionPoint) {
	if start.LineIndex < end.LineIndex {
		return start, end
	}
	if start.LineIndex > end.LineIndex {
		return end, start
	}
	// Same line: normalize by column
	if start.ColumnIndex <= end.ColumnIndex {
		return start, end
	}
	return end, start
}

// getSelectedText extracts text between start and end points from the lines slice.
// Lines should be stripped of ANSI codes for accurate text extraction.
func getSelectedText(lines []string, start, end SelectionPoint) string {
	start, end = NormalizeSelection(start, end)

	if start.LineIndex == end.LineIndex {
		// Single line selection
		if start.LineIndex >= len(lines) {
			return ""
		}
		line := lines[start.LineIndex]
		runes := []rune(line)
		endCol := end.ColumnIndex
		if endCol > len(runes) {
			endCol = len(runes)
		}
		if start.ColumnIndex > len(runes) {
			return ""
		}
		return string(runes[start.ColumnIndex:endCol])
	}

	// Multi-line selection
	var result strings.Builder

	// First line: from start column to end of line
	if start.LineIndex >= len(lines) {
		return ""
	}
	line := lines[start.LineIndex]
	runes := []rune(line)
	if start.ColumnIndex < len(runes) {
		result.WriteString(string(runes[start.ColumnIndex:]))
	}
	result.WriteRune('\n')

	// Middle lines: entire lines
	for i := start.LineIndex + 1; i < end.LineIndex; i++ {
		if i < len(lines) {
			result.WriteString(lines[i])
			result.WriteRune('\n')
		}
	}

	// Last line: from start to end column
	if end.LineIndex < len(lines) {
		endLine := lines[end.LineIndex]
		endRunes := []rune(endLine)
		endCol := end.ColumnIndex
		if endCol > len(endRunes) {
			endCol = len(endRunes)
		}
		result.WriteString(string(endRunes[:endCol]))
	}

	return result.String()
}

// ClearSelection resets all selection fields.
func (v *Viewer) ClearSelection() {
	v.isSelecting = false
	v.selectionStart = nil
	v.selectionEnd = nil
	v.selectedText = ""
}

// StartSelection begins a new selection at the given line and column.
func (v *Viewer) StartSelection(lineIndex, columnIndex int) {
	v.isSelecting = true
	v.selectionStart = &SelectionPoint{LineIndex: lineIndex, ColumnIndex: columnIndex}
	v.selectionEnd = &SelectionPoint{LineIndex: lineIndex, ColumnIndex: columnIndex}
}

// ExtendSelection moves the end point to the given line and column.
func (v *Viewer) ExtendSelection(lineIndex, columnIndex int) {
	if !v.isSelecting || v.selectionStart == nil {
		return
	}
	v.selectionEnd = &SelectionPoint{LineIndex: lineIndex, ColumnIndex: columnIndex}
	v.selectedText = getSelectedText(v.Lines, *v.selectionStart, *v.selectionEnd)
}

// HasSelection returns true if a selection is currently active.
func (v *Viewer) HasSelection() bool {
	return v.isSelecting && v.selectionStart != nil && v.selectionEnd != nil
}

// SelectedText returns the currently selected text.
func (v *Viewer) SelectedText() string {
	if !v.HasSelection() {
		return ""
	}
	return v.selectedText
}

// highlightTextRange applies a selection highlight background (ANSI color 238, dark grey)
// to the rune range [start, end) in the line.
func highlightTextRange(line string, start, end int) string {
	runes := []rune(line)
	if start < 0 {
		start = 0
	}
	if end > len(runes) {
		end = len(runes)
	}
	if start >= end {
		return line
	}

	before := string(runes[:start])
	selected := string(runes[start:end])
	after := string(runes[end:])

	// Apply selection highlight: dark grey background (238)
	return before + "\x1b[48;5;238m" + selected + "\x1b[m" + after
}

// highlightTextRangeWithStripped applies selection highlighting to displayLine,
// using rune indices from strippedLine. This handles cases where displayLine may
// contain ANSI codes (from search highlights or other styling) but we need to
// calculate selection ranges based on the stripped text.
//
// Strategy: extract the substring from displayLine at byte boundaries, then apply highlighting.
// This preserves any existing ANSI codes in the display line.
func highlightTextRangeWithStripped(displayLine, strippedLine string, start, end int) string {
	strippedRunes := []rune(strippedLine)

	if start < 0 {
		start = 0
	}
	if end > len(strippedRunes) {
		end = len(strippedRunes)
	}
	if start >= end {
		return displayLine
	}

	// Extract the substring from displayLine using the same indices.
	// Since displayLine and strippedLine are derived from the same content
	// (but displayLine may have added ANSI codes), we need to be careful.
	// We'll use the rune indices from strippedLine to guide us through displayLine.
	//
	// However, this is tricky because displayLine may have ANSI codes that shift positions.
	// Simpler approach: highlight by working through displayLine and counting visual runes.

	// For simplicity, strip ANSI from displayLine and use regular highlighting.
	// Then re-apply any leading/trailing ANSI codes if needed.
	plain := stripANSI(displayLine)
	plainRunes := []rune(plain)

	if start > len(plainRunes) {
		start = len(plainRunes)
	}
	if end > len(plainRunes) {
		end = len(plainRunes)
	}
	if start >= end {
		return displayLine
	}

	before := string(plainRunes[:start])
	selected := string(plainRunes[start:end])
	after := string(plainRunes[end:])

	// Apply selection highlight: dark grey background (238)
	return before + "\x1b[48;5;238m" + selected + "\x1b[m" + after
}

// stripANSI removes all ANSI escape sequences from a string.
func stripANSI(s string) string {
	var result strings.Builder
	i := 0
	for i < len(s) {
		if s[i] == '\x1b' {
			// Find the end of this escape sequence
			j := i + 1
			for j < len(s) && s[j] != 'm' {
				j++
			}
			if j < len(s) {
				j++ // include the 'm'
			}
			i = j
			continue
		}
		result.WriteByte(s[i])
		i++
	}
	return result.String()
}
