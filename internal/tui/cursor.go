package tui

// Keyboard cursor movement for read/view mode. Before this, a cursor could
// only be committed by a mouse click (mouse.go) — Left/Right and
// Shift+arrows here give keyboard-only users (e.g. over SSH/tmux where
// mouse forwarding is often unreliable) the same "navigate to a line or
// range, then Ctrl+C to copy" workflow. Plain Up/Down/j/k/PgUp/PgDn/g/G
// deliberately keep their existing pure-scroll meaning — introducing a
// competing "these also move a cursor" behavior there would risk breaking
// well-established muscle memory for a feature that's fully reachable via
// Left/Right (which wrap across line boundaries, so they alone can reach
// any row) and Shift+Up/Down (for vertical selection) instead.

// ensureCursorInitialized commits a cursor at the top of the current
// viewport if none exists yet, so the very first Left/Right/Shift+arrow
// press has a starting position to move from/extend from — mirroring how
// a mouse click always commits a fresh position rather than requiring one
// to already exist.
func (v *Viewer) ensureCursorInitialized() {
	if v.hasCursor {
		return
	}
	row := v.Offset
	if row >= len(v.Lines) {
		row = len(v.Lines) - 1
	}
	if row < 0 {
		row = 0
	}
	v.hasCursor = true
	v.cursorRow = row
	v.cursorCol = 0
}

// clampCursorPosition clamps row to a valid v.Lines index and col to the
// visible (ANSI-stripped) rune length of that row, so the cursor can never
// land past the end of the document or mid-line past real content.
func (v *Viewer) clampCursorPosition(row, col int) (int, int) {
	if row < 0 {
		row = 0
	}
	if row >= len(v.Lines) {
		row = len(v.Lines) - 1
	}
	lineLen := len([]rune(stripANSI(v.Lines[row])))
	if col < 0 {
		col = 0
	}
	if col > lineLen {
		col = lineLen
	}
	return row, col
}

// cursorTargetRight returns the position one step right of (row, col): the
// next column on the same line, or the start of the next line if already
// at the end of this one — so repeated Right presses can traverse the
// whole document, not just one line.
func (v *Viewer) cursorTargetRight(row, col int) (int, int) {
	if row < 0 || row >= len(v.Lines) {
		return row, col
	}
	lineLen := len([]rune(stripANSI(v.Lines[row])))
	if col < lineLen {
		return row, col + 1
	}
	if row+1 < len(v.Lines) {
		return row + 1, 0
	}
	return row, col // already at the end of the document
}

// cursorTargetLeft is cursorTargetRight's mirror: the previous column on
// the same line, or the end of the previous line if already at column 0.
func (v *Viewer) cursorTargetLeft(row, col int) (int, int) {
	if col > 0 {
		return row, col - 1
	}
	if row > 0 {
		prevLen := len([]rune(stripANSI(v.Lines[row-1])))
		return row - 1, prevLen
	}
	return row, col // already at the start of the document
}

// scrollCursorIntoView adjusts v.Offset so the cursor's row stays within
// the visible content viewport, the same follow-the-target behavior
// scrollToMatch() already uses for search matches.
func (v *Viewer) scrollCursorIntoView() {
	contentHeight := v.Height - 2 // header + status bar
	if v.cursorRow < v.Offset {
		v.Offset = v.cursorRow
	} else if v.cursorRow >= v.Offset+contentHeight {
		v.Offset = v.cursorRow - contentHeight + 1
	}
	if v.Offset < 0 {
		v.Offset = 0
	}
	if v.Offset > v.maxOffset() {
		v.Offset = v.maxOffset()
	}
}

// moveCursor commits the read-mode cursor to (row, col), clamped to valid
// bounds, scrolling it into view. When extend is true (Shift held) it also
// extends the current selection — starting one anchored at the pre-move
// cursor position if none exists yet — mirroring Shift+Click. When extend
// is false, any active selection is cleared instead, matching a plain
// (non-Shift) mouse click: a bare arrow press always starts fresh rather
// than silently carrying forward whatever was selected before.
func (v *Viewer) moveCursor(row, col int, extend bool) {
	if len(v.Lines) == 0 {
		return
	}
	v.ensureCursorInitialized()
	if extend {
		if !v.HasSelection() {
			v.StartSelection(v.cursorRow, v.cursorCol)
		}
	} else {
		v.ClearSelection()
	}
	row, col = v.clampCursorPosition(row, col)
	v.hasCursor = true
	v.cursorRow = row
	v.cursorCol = col
	if extend {
		v.ExtendSelection(row, col)
	}
	v.scrollCursorIntoView()
}
