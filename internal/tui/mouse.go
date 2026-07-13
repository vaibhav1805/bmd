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
				v.ExtendSelection(docLine, msg.X)
			}
		}
		return v, nil

	case tea.MouseActionPress:
		if msg.Button == tea.MouseButtonLeft {
			// In split-pane mode, ignore clicks on the right pane (preview) to prevent corruption
			if dm, ok := v.activeChild.(*DirectoryModel); ok && dm.splitMode {
				leftWidth, _, ok := splitPaneWidths(v.Width)
				if ok && msg.X > leftWidth {
					// Click is on the right pane preview - ignore it
					return v, nil
				}
			}

			// Ignore clicks on header (Y=0) or status bar (Y >= Height-1).
			if msg.Y == 0 || msg.Y >= v.Height-1 {
				return v, nil
			}
			// Y=1 is the first content row; subtract 1 for header offset.
			clickLine := msg.Y - 1 + v.Offset
			// Check if any link is registered at this line.
			for _, entry := range v.links.Links {
				if entry.LineIndex == clickLine {
					v.ClearSelection()
					return v.followLink(entry.URL)
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
	}

	return v, nil
}
