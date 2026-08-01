package tui

import (
	tea "github.com/charmbracelet/bubbletea"
)

// updateMouse handles mouse wheel scrolling, edit-mode clicks (cursor
// placement), and view-mode clicks (link follow, text selection).
func (v *Viewer) updateMouse(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	// Handle mouse wheel scrolling (SCROLL-01) using the Type field.
	// MouseWheelUp and MouseWheelDown are deprecated types but still work.
	scrollLines := 3
	if msg.Type == tea.MouseWheelUp {
		if v.editMode {
			// Scroll up in edit mode
			if v.Offset > scrollLines {
				v.Offset -= scrollLines
			} else {
				v.Offset = 0
			}
		} else {
			v.Offset = clamp(v.Offset-scrollLines, 0, v.maxOffset())
		}
		return v, nil
	} else if msg.Type == tea.MouseWheelDown {
		if v.editMode {
			// Scroll down in edit mode
			lines := v.editBuffer.GetLines()
			pageSize := v.Height - 2
			if v.Offset+pageSize < len(lines) {
				v.Offset += scrollLines
			} else {
				v.Offset = max(0, len(lines)-pageSize)
			}
		} else {
			v.Offset = clamp(v.Offset+scrollLines, 0, v.maxOffset())
		}
		return v, nil
	}

	// Handle clicks in edit mode
	if v.editMode {
		switch msg.Action {
		case tea.MouseActionPress:
			if msg.Button == tea.MouseButtonLeft {
				// Ignore clicks on header (Y=0) or status bar (Y >= Height-1).
				if msg.Y == 0 || msg.Y >= v.Height-1 {
					return v, nil
				}
				// Y=1 is the first content row; subtract 1 for header offset.
				clickLine := msg.Y - 1 + v.Offset
				lines := v.editBuffer.GetLines()
				if clickLine >= 0 && clickLine < len(lines) {
					// Move cursor to the clicked line
					v.editBuffer.SetCursorLine(clickLine)
					// Move cursor to approximate column position in the line
					line := lines[clickLine]
					col := msg.X
					// Clamp column to line length (in runes, not bytes)
					runeCount := len([]rune(line))
					if col > runeCount {
						col = runeCount
					}
					v.editBuffer.SetCursorCol(col)
				}
				return v, nil
			}
		}
		return v, nil
	}

	switch msg.Action {
	case tea.MouseActionMotion:
		// Track mouse position for hover cursor rendering (MOUSE-01).
		v.mouseRow = msg.Y
		v.mouseCol = msg.X

		// If currently selecting, extend the selection
		if v.isSelecting && v.selectionStart != nil {
			docLine := msg.Y - 1 + v.Offset
			if docLine >= 0 && docLine < len(v.Lines) {
				if docLine != v.selectionStart.LineIndex || msg.X != v.selectionStart.ColumnIndex {
					// The mouse has moved off the press point: this is a
					// drag-select, not a click, so cancel any pending
					// link-follow queued by mouse-down (bmd-zag).
					v.mouseDragged = true
				}
				v.ExtendSelection(docLine, msg.X)
			}
		}
		return v, nil

	case tea.MouseActionPress:
		if msg.Button == tea.MouseButtonLeft {
			// Ignore clicks on header (Y=0) or status bar (Y >= Height-1).
			if msg.Y == 0 || msg.Y >= v.Height-1 {
				return v, nil
			}
			// Y=1 is the first content row; subtract 1 for header offset.
			clickLine := msg.Y - 1 + v.Offset
			v.mouseDragged = false

			// If a link is registered at this line, queue the follow instead
			// of navigating immediately (bmd-zag): a click-drag gesture that
			// starts on a link line must be able to become a text selection
			// rather than always jumping away on mouse-down.
			v.pendingLinkURL = ""
			for _, entry := range v.links.Links {
				if entry.LineIndex == clickLine {
					v.pendingLinkURL = entry.URL
					break
				}
			}

			// Check for Shift+Click to extend selection
			if msg.Shift {
				if v.HasSelection() {
					v.ExtendSelection(clickLine, msg.X)
				} else {
					// Start new selection if Shift+Click but no prior selection
					v.StartSelection(clickLine, msg.X)
				}
			} else {
				// Normal click: start new selection
				v.StartSelection(clickLine, msg.X)
				// Also commit the cursor position as before
				v.hasCursor = true
				v.cursorRow = clickLine
				v.cursorCol = msg.X
			}
			return v, nil
		}

	case tea.MouseActionRelease:
		// A pending link-follow only fires if the mouse never moved off the
		// press point (a real click); any drag in between cancels it and
		// leaves the in-progress text selection as the result (bmd-zag).
		if v.pendingLinkURL != "" {
			url := v.pendingLinkURL
			v.pendingLinkURL = ""
			if !v.mouseDragged {
				v.ClearSelection()
				return v.followLink(url)
			}
		}
		return v, nil
	}

	return v, nil
}
