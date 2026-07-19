package tui

import (
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// updateEdit handles keyboard input while editMode is active: cursor
// movement, selection, clipboard, undo/redo, save, and mode transitions
// (search/jump/replace/outline/exit) reachable from the editor.
func (v *Viewer) updateEdit(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Route to find/replace handler when replace prompt is open
	if v.replaceMode {
		return v.updateReplace(msg)
	}
	switch msg.Type {
	case tea.KeyUp:
		v.editBuffer.ClearSelection()
		v.editBuffer.CursorUp()
		return v, nil
	case tea.KeyDown:
		v.editBuffer.ClearSelection()
		v.editBuffer.CursorDown()
		return v, nil
	case tea.KeyLeft:
		v.editBuffer.ClearSelection()
		v.editBuffer.CursorLeft()
		return v, nil
	case tea.KeyRight:
		v.editBuffer.ClearSelection()
		v.editBuffer.CursorRight()
		return v, nil
	case tea.KeyShiftUp:
		if !v.editBuffer.HasSelection() {
			v.editBuffer.StartSelection()
		}
		v.editBuffer.CursorUp()
		v.editBuffer.EndSelection()
		return v, nil
	case tea.KeyShiftDown:
		if !v.editBuffer.HasSelection() {
			v.editBuffer.StartSelection()
		}
		v.editBuffer.CursorDown()
		v.editBuffer.EndSelection()
		return v, nil
	case tea.KeyShiftLeft:
		if !v.editBuffer.HasSelection() {
			v.editBuffer.StartSelection()
		}
		v.editBuffer.CursorLeft()
		v.editBuffer.EndSelection()
		return v, nil
	case tea.KeyShiftRight:
		if !v.editBuffer.HasSelection() {
			v.editBuffer.StartSelection()
		}
		v.editBuffer.CursorRight()
		v.editBuffer.EndSelection()
		return v, nil
	case tea.KeyBackspace:
		if v.editBuffer.HasSelection() {
			v.editBuffer.DeleteSelection()
		} else {
			v.editBuffer.Backspace()
		}
		return v, nil
	case tea.KeyDelete:
		if v.editBuffer.HasSelection() {
			v.editBuffer.DeleteSelection()
		} else {
			v.editBuffer.Delete()
		}
		return v, nil
	case tea.KeyEnter:
		if v.editBuffer.HasSelection() {
			v.editBuffer.DeleteSelection()
		}
		v.editBuffer.EnterNewLine()
		return v, nil
	case tea.KeyCtrlC:
		// Copy selected text or current line; do not quit in edit mode
		if v.editBuffer.HasSelection() {
			text := v.editBuffer.GetSelectedText()
			v.editEditClipboard = text
			if _, err := copyWithFallback(text); err != nil {
				v.errorMsg = "Clipboard unavailable"
			} else {
				v.errorMsg = fmt.Sprintf("Copied %d chars", len([]rune(text)))
			}
		} else {
			lines := v.editBuffer.GetLines()
			line := ""
			if v.editBuffer.CursorLine() < len(lines) {
				line = lines[v.editBuffer.CursorLine()]
			}
			v.editEditClipboard = line
			if _, err := copyWithFallback(line); err != nil {
				v.errorMsg = "Clipboard unavailable"
			} else {
				v.errorMsg = fmt.Sprintf("Copied line (%d chars)", len([]rune(line)))
			}
		}
		return v, clearErrorAfter(statusTimeout)
	case tea.KeyCtrlX:
		// Cut selected text (or current line if no selection)
		if v.editBuffer.HasSelection() {
			text := v.editBuffer.GetSelectedText()
			v.editEditClipboard = text
			_, cbErr := copyWithFallback(text)
			v.editBuffer.DeleteSelection()
			if cbErr != nil {
				v.errorMsg = "Clipboard unavailable"
			} else {
				v.errorMsg = fmt.Sprintf("Cut %d chars", len([]rune(text)))
			}
		} else {
			lines := v.editBuffer.GetLines()
			line := ""
			if v.editBuffer.CursorLine() < len(lines) {
				line = lines[v.editBuffer.CursorLine()]
			}
			v.editEditClipboard = line
			if _, err := copyWithFallback(line); err != nil {
				v.errorMsg = "Clipboard unavailable"
			} else {
				v.errorMsg = "Cut line"
			}
		}
		return v, clearErrorAfter(statusTimeout)
	case tea.KeyCtrlV:
		// Paste from internal clipboard (text copied via Ctrl+C/X in this session)
		if v.editEditClipboard != "" {
			v.editBuffer.InsertText(v.editEditClipboard)
			v.errorMsg = fmt.Sprintf("Pasted %d chars", len([]rune(v.editEditClipboard)))
			return v, clearErrorAfter(statusTimeout)
		}
		// Also handle bracketed paste (Paste: true on KeyRunes)
		return v, nil
	case tea.KeyCtrlS:
		// Save the file
		err := v.editBuffer.SaveToFile(v.FilePath)
		if err != nil {
			v.errorMsg = fmt.Sprintf("Save failed: %v", err)
			// Schedule error message to clear after timeout
			return v, tea.Tick(statusTimeout, func(t time.Time) tea.Msg {
				return clearErrorMsg{}
			})
		} else {
			// Delete the autosave file — user has explicitly saved.
			v.deleteAutoSave()
			v.errorMsg = "Saved"
			// Clear the saved message after a shorter timeout
			return v, tea.Tick(1*time.Second, func(t time.Time) tea.Msg {
				return clearErrorMsg{}
			})
		}
	case tea.KeyCtrlZ:
		// Undo
		v.editBuffer.Undo()
		return v, nil
	case tea.KeyCtrlY:
		// Redo
		v.editBuffer.Redo()
		return v, nil
	case tea.KeyCtrlF:
		// Enter search mode (search the edited buffer)
		v.searchMode = true
		v.searchInput = ""
		v.searchState = NewSearchState() // Clear previous search results
		return v, nil
	case tea.KeyCtrlG:
		// Enter jump-to-line mode
		v.jumpMode = true
		v.jumpInput = ""
		return v, nil
	case tea.KeyCtrlHome:
		// Jump to beginning of document
		v.editBuffer.JumpToStart()
		v.Offset = 0
		return v, nil
	case tea.KeyCtrlEnd:
		// Jump to end of document
		v.editBuffer.JumpToEnd()
		// Scroll viewport to show the end
		lines := v.editBuffer.GetLines()
		if len(lines) > v.Height-2 {
			v.Offset = len(lines) - (v.Height - 2)
		} else {
			v.Offset = 0
		}
		return v, nil
	case tea.KeyEsc:
		// Exit edit mode and reload file to show saved changes
		v.editMode = false
		v.markdownSyntaxOpen = false
		result, cmd := v.loadFileNoHistory(v.FilePath)
		// Restore the saved scroll position from when we entered edit mode
		if v.savedScrollOffset > 0 {
			result.Offset = v.savedScrollOffset
			// Clamp to valid range
			if result.Offset > result.maxOffset() {
				result.Offset = result.maxOffset()
			}
		}
		return result, cmd
	case tea.KeyCtrlH:
		// Open Find & Replace prompt in edit mode
		v.replaceMode = true
		v.replaceState = ReplaceState{CurrentMatch: -1}
		return v, nil
	}
	// Handle additional keys by string matching (Page Up/Down, word movement, indentation)
	keyStr := msg.String()
	switch keyStr {
	case "ctrl+left":
		v.editBuffer.CursorWordLeft()
		return v, nil
	case "ctrl+right":
		v.editBuffer.CursorWordRight()
		return v, nil
	case "tab":
		v.editBuffer.IndentLine()
		return v, nil
	case "shift+tab":
		v.editBuffer.DedentLine()
		return v, nil
	case "ctrl+d":
		// Ctrl+D: duplicate current line
		v.editBuffer.DuplicateLine()
		v.errorMsg = "Duplicated line"
		return v, clearErrorAfter(statusTimeout)
	case "ctrl+shift+k":
		// Ctrl+Shift+K: delete current line
		v.editBuffer.DeleteLine()
		v.errorMsg = "Deleted line"
		return v, clearErrorAfter(statusTimeout)
	case "alt+up":
		// Alt+Up: move line up
		v.editBuffer.MoveLineUp()
		v.errorMsg = "Moved line up"
		return v, clearErrorAfter(statusTimeout)
	case "alt+down":
		// Alt+Down: move line down
		v.editBuffer.MoveLineDown()
		v.errorMsg = "Moved line down"
		return v, clearErrorAfter(statusTimeout)
	case "pgup":
		// Scroll up one page (Height - 2 for header/status)
		pageSize := v.Height - 2
		if v.Offset > pageSize {
			v.Offset -= pageSize
		} else {
			v.Offset = 0
		}
		// Move cursor up to keep it visible
		for i := 0; i < pageSize && v.editBuffer.CursorLine() > 0; i++ {
			v.editBuffer.CursorUp()
		}
		return v, nil
	case "pgdn":
		// Scroll down one page
		pageSize := v.Height - 2
		lines := v.editBuffer.GetLines()
		if v.Offset+pageSize < len(lines) {
			v.Offset += pageSize
		} else {
			v.Offset = max(0, len(lines)-pageSize)
		}
		// Move cursor down to keep it visible
		for i := 0; i < pageSize && v.editBuffer.CursorLine() < len(lines)-1; i++ {
			v.editBuffer.CursorDown()
		}
		return v, nil
	case "ctrl+o":
		// Open outline/TOC in edit mode — scan buffer for headings
		v.outlineMode = true
		v.outlineHeadings = v.extractEditHeadings()
		v.outlineSelection = 0
		v.errorMsg = "Outline (Enter to jump, Esc to close)"
		return v, clearErrorAfter(statusTimeout)
	}
	// Bracketed paste: msg.Paste is true and Runes contains pasted content
	if msg.Paste && len(msg.Runes) > 0 {
		text := string(msg.Runes)
		v.editEditClipboard = text
		v.editBuffer.InsertText(text)
		v.errorMsg = fmt.Sprintf("Pasted %d chars", len(msg.Runes))
		return v, clearErrorAfter(statusTimeout)
	}
	// Character input (letter, number, symbol, space)
	if len(msg.Runes) > 0 {
		if v.editBuffer.HasSelection() {
			v.editBuffer.DeleteSelection()
		}
		v.editBuffer.Insert(msg.Runes[0])
		return v, nil
	}
	return v, nil
}
