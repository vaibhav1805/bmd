package tui

import (
	"fmt"
	"strconv"

	tea "github.com/charmbracelet/bubbletea"
)

// vimModeT is the current vim sub-mode within edit mode. Only meaningful
// while v.editMode && v.vimEnabled; ignored otherwise.
type vimModeT int

const (
	vimModeNormal vimModeT = iota
	vimModeInsert
	vimModeVisual
	vimModeVisualLine
)

// isWordChar mirrors internal/editor's word-boundary definition (alphanumeric + underscore).
func isWordChar(r rune) bool {
	return r == '_' ||
		(r >= 'a' && r <= 'z') ||
		(r >= 'A' && r <= 'Z') ||
		(r >= '0' && r <= '9')
}

// updateVimCommand handles a single keystroke while edit mode's vim engine
// owns input (v.vimEnabled && v.vimMode != vimModeInsert). It never falls
// through to plain-text insertion: normal/visual mode must never mutate the
// buffer except through an explicit vim command, which is the whole premise
// of modal editing.
func (v *Viewer) updateVimCommand(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Arrow keys always work as plain motions, count or no count.
	switch msg.Type {
	case tea.KeyUp:
		v.vimMoveCursor(func() { v.editBuffer.CursorUp() }, v.vimTakeCount())
		return v, nil
	case tea.KeyDown:
		v.vimMoveCursor(func() { v.editBuffer.CursorDown() }, v.vimTakeCount())
		return v, nil
	case tea.KeyLeft:
		v.vimMoveCursor(func() { v.editBuffer.CursorLeft() }, v.vimTakeCount())
		return v, nil
	case tea.KeyRight:
		v.vimMoveCursor(func() { v.editBuffer.CursorRight() }, v.vimTakeCount())
		return v, nil
	}

	if len(msg.Runes) != 1 {
		// Unmapped non-rune key (e.g. Enter, Backspace, Delete, Tab) in
		// normal/visual mode: safe no-op rather than falling through to
		// legacy handlers that would mutate the buffer.
		return v, nil
	}
	r := msg.Runes[0]

	// Count prefix: digits accumulate; a leading '0' is the "start of line"
	// motion instead (matches vim: 0 only joins a count once one has started).
	if r >= '1' && r <= '9' || (r == '0' && v.vimPendingCount != "") {
		v.vimPendingCount += string(r)
		return v, nil
	}

	switch v.vimMode {
	case vimModeVisual, vimModeVisualLine:
		return v.updateVimVisual(r)
	default:
		return v.updateVimNormal(r)
	}
}

// vimTakeCount consumes and resets the pending count, defaulting to 1.
func (v *Viewer) vimTakeCount() int {
	n := 1
	if v.vimPendingCount != "" {
		if parsed, err := strconv.Atoi(v.vimPendingCount); err == nil && parsed > 0 {
			n = parsed
		}
	}
	v.vimPendingCount = ""
	return n
}

// vimMoveCursor repeats a motion count times, clearing any selection first
// (matches arrow-key behavior in bmd's non-vim editor: movement alone never
// selects).
func (v *Viewer) vimMoveCursor(step func(), count int) {
	v.editBuffer.ClearSelection()
	for i := 0; i < count; i++ {
		step()
	}
}

// updateVimNormal handles a rune keystroke in normal mode: motions, mode
// entry, operators, and single-key commands (x/X/p/P/u/etc.).
func (v *Viewer) updateVimNormal(r rune) (tea.Model, tea.Cmd) {
	// A pending operator (d/y/c) owns the next keystroke, including "gg" and
	// counts, so it must be checked before the bare-gg case below.
	if v.vimPendingOperator != 0 {
		return v.updateVimOperatorMotion(r)
	}

	// Second key of a pending "g" prefix (gg = jump to start).
	if v.vimPendingG {
		v.vimPendingG = false
		if r == 'g' {
			v.vimTakeCount() // gg ignores an accumulated count in this scope
			v.editBuffer.ClearSelection()
			v.editBuffer.JumpToStart()
		}
		return v, nil
	}

	count := v.vimPendingCount

	switch r {
	case 'g':
		v.vimPendingG = true
		return v, nil
	case 'h':
		v.vimMoveCursor(func() { v.editBuffer.CursorLeft() }, v.vimTakeCount())
	case 'l':
		v.vimMoveCursor(func() { v.editBuffer.CursorRight() }, v.vimTakeCount())
	case 'j':
		v.vimMoveCursor(func() { v.editBuffer.CursorDown() }, v.vimTakeCount())
	case 'k':
		v.vimMoveCursor(func() { v.editBuffer.CursorUp() }, v.vimTakeCount())
	case 'w':
		v.vimMoveCursor(func() { v.editBuffer.CursorWordRight() }, v.vimTakeCount())
	case 'b':
		v.vimMoveCursor(func() { v.editBuffer.CursorWordLeft() }, v.vimTakeCount())
	case 'e':
		v.vimMoveCursor(func() { v.vimWordEnd() }, v.vimTakeCount())
	case '0':
		v.vimPendingCount = ""
		v.editBuffer.ClearSelection()
		v.editBuffer.SetCursorCol(0)
	case '$':
		v.vimTakeCount()
		v.editBuffer.ClearSelection()
		v.editBuffer.SetCursorCol(v.vimLineLen(v.editBuffer.CursorLine()))
	case 'G':
		v.editBuffer.ClearSelection()
		if count != "" {
			if n, err := strconv.Atoi(count); err == nil && n > 0 {
				v.editBuffer.SetCursorLine(n - 1)
			}
			v.vimPendingCount = ""
		} else {
			v.editBuffer.JumpToEnd()
		}
	case 'i':
		v.vimTakeCount()
		v.vimMode = vimModeInsert
	case 'a':
		v.vimTakeCount()
		// Append after the character under the cursor -- but only if the
		// cursor isn't already sitting past the last character (bmd's edit
		// mode uses an insertion-point cursor that CAN rest at col ==
		// len(line), unlike vim's on-character cursor; calling CursorRight
		// there would wrap to the next line instead of staying put).
		if v.editBuffer.CursorCol() < v.vimLineLen(v.editBuffer.CursorLine()) {
			v.editBuffer.CursorRight()
		}
		v.vimMode = vimModeInsert
	case 'I':
		v.vimTakeCount()
		v.editBuffer.SetCursorCol(0)
		v.vimMode = vimModeInsert
	case 'A':
		v.vimTakeCount()
		v.editBuffer.SetCursorCol(v.vimLineLen(v.editBuffer.CursorLine()))
		v.vimMode = vimModeInsert
	case 'o':
		v.vimTakeCount()
		v.editBuffer.SetCursorCol(v.vimLineLen(v.editBuffer.CursorLine()))
		v.editBuffer.EnterNewLine()
		v.vimMode = vimModeInsert
	case 'O':
		v.vimTakeCount()
		// Insert a plain blank line above rather than routing through
		// EnterNewLine at col 0: EnterNewLine's auto-indent is computed
		// from the current line and prepended onto its own (already
		// indented) text when split at column 0, doubling indentation on
		// list items / indented lines.
		v.editBuffer.InsertLinesAbove([]string{""})
		v.vimMode = vimModeInsert
	case 'v':
		v.vimTakeCount()
		v.vimVisualAnchor = [2]int{v.editBuffer.CursorLine(), v.editBuffer.CursorCol()}
		v.editBuffer.StartSelection()
		v.vimMode = vimModeVisual
	case 'V':
		v.vimTakeCount()
		v.vimVisualAnchor = [2]int{v.editBuffer.CursorLine(), v.editBuffer.CursorCol()}
		v.editBuffer.StartSelection()
		v.vimMode = vimModeVisualLine
	case 'd', 'y', 'c':
		v.vimPendingOperator = r
	case 'x':
		n := v.vimTakeCount()
		v.vimDeleteCharsForward(n)
		return v, clearErrorAfter(statusTimeout)
	case 'X':
		n := v.vimTakeCount()
		v.vimDeleteCharsBackward(n)
		return v, clearErrorAfter(statusTimeout)
	case 'p':
		v.vimTakeCount()
		v.vimPaste(false)
		return v, clearErrorAfter(statusTimeout)
	case 'P':
		v.vimTakeCount()
		v.vimPaste(true)
		return v, clearErrorAfter(statusTimeout)
	case 'u':
		v.vimTakeCount()
		v.editBuffer.Undo()
	default:
		v.vimPendingCount = ""
	}
	return v, nil
}

// updateVimVisual handles a rune keystroke while a visual (charwise or
// linewise) selection is active.
func (v *Viewer) updateVimVisual(r rune) (tea.Model, tea.Cmd) {
	// Second key of a pending "g" prefix (gg = jump to start, extending the
	// selection).
	if v.vimPendingG {
		v.vimPendingG = false
		if r == 'g' {
			v.vimTakeCount()
			v.editBuffer.JumpToStart()
			v.editBuffer.EndSelection()
		}
		return v, nil
	}

	switch r {
	case 'h':
		v.vimExtendVisual(func() { v.editBuffer.CursorLeft() }, v.vimTakeCount())
	case 'l':
		v.vimExtendVisual(func() { v.editBuffer.CursorRight() }, v.vimTakeCount())
	case 'j':
		v.vimExtendVisual(func() { v.editBuffer.CursorDown() }, v.vimTakeCount())
	case 'k':
		v.vimExtendVisual(func() { v.editBuffer.CursorUp() }, v.vimTakeCount())
	case 'w':
		v.vimExtendVisual(func() { v.editBuffer.CursorWordRight() }, v.vimTakeCount())
	case 'b':
		v.vimExtendVisual(func() { v.editBuffer.CursorWordLeft() }, v.vimTakeCount())
	case 'e':
		v.vimExtendVisual(func() { v.vimWordEnd() }, v.vimTakeCount())
	case '0':
		v.vimPendingCount = ""
		v.editBuffer.SetCursorCol(0)
		v.editBuffer.EndSelection()
	case '$':
		v.vimTakeCount()
		v.editBuffer.SetCursorCol(v.vimLineLen(v.editBuffer.CursorLine()))
		v.editBuffer.EndSelection()
	case 'g':
		v.vimPendingG = true
	case 'G':
		v.vimTakeCount()
		v.editBuffer.JumpToEnd()
		v.editBuffer.EndSelection()
	case 'v':
		if v.vimMode == vimModeVisual {
			v.exitVimVisual()
		} else {
			v.vimMode = vimModeVisual
		}
	case 'V':
		if v.vimMode == vimModeVisualLine {
			v.exitVimVisual()
		} else {
			v.vimMode = vimModeVisualLine
		}
	case 'd', 'x':
		v.vimApplyVisualOperator('d')
		return v, clearErrorAfter(statusTimeout)
	case 'y':
		v.vimApplyVisualOperator('y')
		return v, clearErrorAfter(statusTimeout)
	case 'c':
		v.vimApplyVisualOperator('c')
	default:
		v.vimPendingCount = ""
	}
	return v, nil
}

// exitVimVisual clears the selection and returns to normal mode.
func (v *Viewer) exitVimVisual() {
	v.editBuffer.ClearSelection()
	v.vimMode = vimModeNormal
	v.vimPendingCount = ""
}

// vimExtendVisual repeats a motion count times, then extends the active
// selection's end to the new cursor position (rather than clearing it, as
// vimMoveCursor does for plain normal-mode motions).
func (v *Viewer) vimExtendVisual(step func(), count int) {
	for i := 0; i < count; i++ {
		step()
	}
	v.editBuffer.EndSelection()
}

// vimApplyVisualOperator applies d/y/c to the active visual selection, then
// returns to normal mode (or insert mode, for c).
func (v *Viewer) vimApplyVisualOperator(op rune) {
	if v.vimMode == vimModeVisualLine {
		start, end := v.vimVisualAnchor[0], v.editBuffer.CursorLine()
		v.vimApplyLinewiseRange(op, start, end)
		return
	}

	text := v.editBuffer.GetSelectedText()
	v.vimRegister = text
	v.vimRegisterLinewise = false

	switch op {
	case 'y':
		v.editBuffer.ClearSelection()
		v.vimMode = vimModeNormal
		v.errorMsg = fmt.Sprintf("Yanked %d chars", len([]rune(text)))
	case 'd':
		v.editBuffer.DeleteSelection()
		v.vimMode = vimModeNormal
		v.errorMsg = fmt.Sprintf("Deleted %d chars", len([]rune(text)))
	case 'c':
		v.editBuffer.DeleteSelection()
		v.vimMode = vimModeInsert
	}
}

// vimApplyLinewiseRange applies a linewise d/y/c operator to the (inclusive)
// line range covering fromLine and toLine, in either order.
func (v *Viewer) vimApplyLinewiseRange(op rune, fromLine, toLine int) {
	start, end := fromLine, toLine
	if start > end {
		start, end = end, start
	}
	v.editBuffer.ClearSelection()
	v.editBuffer.SetCursorLine(start)
	count := end - start + 1

	switch op {
	case 'y':
		lines := v.editBuffer.GetLines()
		if start >= len(lines) {
			v.vimMode = vimModeNormal
			return
		}
		if end >= len(lines) {
			end = len(lines) - 1
		}
		yanked := append([]string(nil), lines[start:end+1]...)
		v.vimRegister = joinLines(yanked)
		v.vimRegisterLinewise = true
		v.vimMode = vimModeNormal
		v.errorMsg = fmt.Sprintf("Yanked %d line(s)", len(yanked))
	case 'd':
		removed := v.editBuffer.DeleteLines(count)
		v.vimRegister = joinLines(removed)
		v.vimRegisterLinewise = true
		v.vimMode = vimModeNormal
		v.errorMsg = fmt.Sprintf("Deleted %d line(s)", len(removed))
	case 'c':
		removed := v.editBuffer.DeleteLines(count)
		v.vimRegister = joinLines(removed)
		v.vimRegisterLinewise = true
		// Leave a blank line at the cut point and drop into insert mode on
		// it, matching vim's cc/linewise-c behavior of replacing content
		// in place rather than shifting subsequent lines up permanently.
		v.editBuffer.InsertLinesAbove([]string{""})
		v.vimMode = vimModeInsert
	}
}

// joinLines joins lines with "\n" for register storage.
func joinLines(lines []string) string {
	out := ""
	for i, l := range lines {
		if i > 0 {
			out += "\n"
		}
		out += l
	}
	return out
}

// updateVimOperatorMotion handles the second key after d/y/c: either the
// same letter again (dd/yy/cc, linewise on the current line), "gg" (to the
// start of the document), "G" (to the end), or a motion (dw, de, db, d0,
// d$) applied charwise via the selection API.
func (v *Viewer) updateVimOperatorMotion(r rune) (tea.Model, tea.Cmd) {
	op := v.vimPendingOperator
	v.vimPendingOperator = 0

	// Second key of an operator+g prefix (dgg/ygg/cgg).
	if v.vimPendingG {
		v.vimPendingG = false
		if r == 'g' {
			v.vimApplyLinewiseRange(op, v.editBuffer.CursorLine(), 0)
			return v, clearErrorAfter(statusTimeout)
		}
		return v, nil
	}

	// Doubled letter: linewise on N lines starting at the cursor.
	if r == op {
		count := v.vimTakeCount()
		cur := v.editBuffer.CursorLine()
		v.vimApplyLinewiseRange(op, cur, cur+count-1)
		return v, clearErrorAfter(statusTimeout)
	}

	switch r {
	case 'g':
		// Re-arm the operator for the second 'g' of "gg".
		v.vimPendingG = true
		v.vimPendingOperator = op
		return v, nil
	case 'G':
		lines := v.editBuffer.GetLines()
		v.vimApplyLinewiseRange(op, v.editBuffer.CursorLine(), len(lines)-1)
		return v, clearErrorAfter(statusTimeout)
	}

	count := v.vimTakeCount()

	switch r {
	case 'w':
		v.vimOperatorCharwiseMotion(op, count, func() { v.editBuffer.CursorWordRight() })
	case 'b':
		v.vimOperatorCharwiseMotion(op, count, func() { v.editBuffer.CursorWordLeft() })
	case 'e':
		v.vimOperatorCharwiseMotion(op, count, func() {
			v.vimWordEnd()
			// 'e' as a motion lands ON the last character of the word;
			// an operator's selection end is exclusive, so extend by one
			// more to include that character in the operated-on span.
			v.editBuffer.SetCursorCol(v.editBuffer.CursorCol() + 1)
		})
	case 'h':
		v.vimOperatorCharwiseMotion(op, count, func() { v.editBuffer.CursorLeft() })
	case 'l':
		v.vimOperatorCharwiseMotion(op, count, func() { v.editBuffer.CursorRight() })
	case '0':
		v.vimOperatorCharwiseMotion(op, 1, func() { v.editBuffer.SetCursorCol(0) })
	case '$':
		v.vimOperatorCharwiseMotion(op, 1, func() { v.editBuffer.SetCursorCol(v.vimLineLen(v.editBuffer.CursorLine())) })
	default:
		// Unrecognized motion after an operator: abort the pending command.
		return v, nil
	}
	return v, clearErrorAfter(statusTimeout)
}

// vimOperatorCharwiseMotion applies operator op over the charwise span from
// the current cursor position to the position reached after calling motion
// count times, via the existing selection primitives.
func (v *Viewer) vimOperatorCharwiseMotion(op rune, count int, motion func()) {
	v.editBuffer.StartSelection()
	for i := 0; i < count; i++ {
		motion()
	}
	v.editBuffer.EndSelection()

	text := v.editBuffer.GetSelectedText()
	v.vimRegister = text
	v.vimRegisterLinewise = false

	switch op {
	case 'y':
		v.editBuffer.ClearSelection()
		v.errorMsg = fmt.Sprintf("Yanked %d chars", len([]rune(text)))
	case 'd':
		v.editBuffer.DeleteSelection()
		v.errorMsg = fmt.Sprintf("Deleted %d chars", len([]rune(text)))
	case 'c':
		v.editBuffer.DeleteSelection()
		v.vimMode = vimModeInsert
	}
}

// vimWordEnd moves the cursor to the end of the current or next word,
// matching vim's 'e' motion. internal/editor has no equivalent primitive
// (CursorWordRight lands on the start of the next word instead), so this
// walks the buffer directly via the public GetLines/CursorLine/CursorCol/
// SetCursorLine/SetCursorCol API.
func (v *Viewer) vimWordEnd() {
	lines := v.editBuffer.GetLines()
	line := v.editBuffer.CursorLine()
	col := v.editBuffer.CursorCol()
	if line >= len(lines) {
		return
	}

	// Step forward at least one character so 'e' always makes progress.
	col++
	for {
		if line >= len(lines) {
			line = len(lines) - 1
			col = len([]rune(lines[line]))
			break
		}
		runes := []rune(lines[line])
		if col >= len(runes) {
			if line >= len(lines)-1 {
				col = len(runes)
				break
			}
			line++
			col = 0
			continue
		}
		if !isWordChar(runes[col]) {
			// Skip non-word characters.
			col++
			continue
		}
		// On a word character: stop once the next character ends the word.
		if col+1 >= len(runes) || !isWordChar(runes[col+1]) {
			break
		}
		col++
	}

	v.editBuffer.SetCursorLine(line)
	v.editBuffer.SetCursorCol(col)
}

// vimLineLen returns the byte length of the given line, matching
// internal/editor's own (non-rune-aware) column-clamping convention so
// SetCursorCol lands in the same place editor.go's own methods would.
func (v *Viewer) vimLineLen(line int) int {
	lines := v.editBuffer.GetLines()
	if line < 0 || line >= len(lines) {
		return 0
	}
	return len(lines[line])
}

// vimDeleteCharsForward implements 'x': delete n characters starting at the
// cursor (like the Delete key, repeated).
func (v *Viewer) vimDeleteCharsForward(n int) {
	v.editBuffer.ClearSelection()
	v.editBuffer.StartSelection()
	for i := 0; i < n; i++ {
		v.editBuffer.CursorRight()
	}
	v.editBuffer.EndSelection()
	text := v.editBuffer.GetSelectedText()
	if text == "" {
		v.editBuffer.ClearSelection()
		return
	}
	v.vimRegister = text
	v.vimRegisterLinewise = false
	v.editBuffer.DeleteSelection()
	v.errorMsg = fmt.Sprintf("Deleted %d char(s)", len([]rune(text)))
}

// vimDeleteCharsBackward implements 'X': delete n characters before the
// cursor (like Backspace, repeated).
func (v *Viewer) vimDeleteCharsBackward(n int) {
	v.editBuffer.ClearSelection()
	v.editBuffer.StartSelection()
	for i := 0; i < n; i++ {
		v.editBuffer.CursorLeft()
	}
	v.editBuffer.EndSelection()
	text := v.editBuffer.GetSelectedText()
	if text == "" {
		v.editBuffer.ClearSelection()
		return
	}
	v.vimRegister = text
	v.vimRegisterLinewise = false
	v.editBuffer.DeleteSelection()
	v.errorMsg = fmt.Sprintf("Deleted %d char(s)", len([]rune(text)))
}

// vimPaste implements 'p' (before=false, paste after cursor/line) and 'P'
// (before=true, paste before). Linewise vs. charwise follows the register's
// source operation (dd/yy/cc are linewise; x/X/charwise d/y/c are charwise).
func (v *Viewer) vimPaste(before bool) {
	if v.vimRegister == "" {
		return
	}
	if v.vimRegisterLinewise {
		lines := splitLines(v.vimRegister)
		if before {
			v.editBuffer.InsertLinesAbove(lines)
		} else {
			v.editBuffer.InsertLinesBelow(lines)
		}
		v.errorMsg = fmt.Sprintf("Pasted %d line(s)", len(lines))
		return
	}
	if !before {
		v.editBuffer.CursorRight()
	}
	v.editBuffer.InsertText(v.vimRegister)
	v.errorMsg = fmt.Sprintf("Pasted %d chars", len([]rune(v.vimRegister)))
}

// splitLines splits a register's joined-with-"\n" string back into lines.
func splitLines(s string) []string {
	if s == "" {
		return nil
	}
	var lines []string
	start := 0
	for i, r := range s {
		if r == '\n' {
			lines = append(lines, s[start:i])
			start = i + 1
		}
	}
	lines = append(lines, s[start:])
	return lines
}
