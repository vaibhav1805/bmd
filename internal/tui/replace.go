// Package tui provides the interactive terminal user interface for bmd.
package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// ReplaceState holds the state of a find-and-replace session in edit mode.
type ReplaceState struct {
	Query         string   // the search text
	Replacement   string   // the replacement text
	CaseSensitive bool     // true for case-sensitive matching
	WholeWord     bool     // true for whole-word matching
	Matches       [][2]int // (line, col) pairs of all current matches
	CurrentMatch  int      // index into Matches of the focused match (-1 = none)
	FocusReplace  bool     // true when the Replace field has focus (false = Find field)
}

// updateReplaceMatches re-runs the find query against the edit buffer and updates matches.
func (v *Viewer) updateReplaceMatches() {
	if v.editBuffer == nil || v.replaceState.Query == "" {
		v.replaceState.Matches = nil
		v.replaceState.CurrentMatch = -1
		return
	}

	v.replaceState.Matches = v.editBuffer.FindAll(
		v.replaceState.Query,
		v.replaceState.CaseSensitive,
		v.replaceState.WholeWord,
	)

	if len(v.replaceState.Matches) == 0 {
		v.replaceState.CurrentMatch = -1
	} else {
		// Clamp current match
		if v.replaceState.CurrentMatch < 0 {
			v.replaceState.CurrentMatch = 0
		} else if v.replaceState.CurrentMatch >= len(v.replaceState.Matches) {
			v.replaceState.CurrentMatch = len(v.replaceState.Matches) - 1
		}
	}
}

// updateReplace handles keypresses while the find/replace prompt is open.
func (v *Viewer) updateReplace(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEsc:
		// Close replace mode without making changes
		v.replaceMode = false
		v.replaceState = ReplaceState{}
		return v, nil

	case tea.KeyTab:
		// Toggle focus between Find and Replace fields
		v.replaceState.FocusReplace = !v.replaceState.FocusReplace
		return v, nil

	case tea.KeyEnter:
		// Replace current match and advance to next
		if len(v.replaceState.Matches) > 0 && v.replaceState.CurrentMatch >= 0 {
			m := v.replaceState.Matches[v.replaceState.CurrentMatch]
			v.editBuffer.ReplaceOne(m[0], m[1],
				v.replaceState.Query, v.replaceState.Replacement,
				v.replaceState.CaseSensitive)
			// Re-find matches after replacement
			v.updateReplaceMatches()
			// CurrentMatch index stays the same (points to the next match now)
			// unless we were at the end
			if v.replaceState.CurrentMatch >= len(v.replaceState.Matches) && len(v.replaceState.Matches) > 0 {
				v.replaceState.CurrentMatch = 0
			}
		}
		return v, nil

	case tea.KeyBackspace:
		if v.replaceState.FocusReplace {
			if len(v.replaceState.Replacement) > 0 {
				v.replaceState.Replacement = v.replaceState.Replacement[:len(v.replaceState.Replacement)-1]
			}
		} else {
			if len(v.replaceState.Query) > 0 {
				v.replaceState.Query = v.replaceState.Query[:len(v.replaceState.Query)-1]
				v.updateReplaceMatches()
			}
		}
		return v, nil

	case tea.KeyCtrlA:
		// Replace all matches
		if v.replaceState.Query != "" {
			count := v.editBuffer.Replace(
				v.replaceState.Query, v.replaceState.Replacement,
				v.replaceState.CaseSensitive, v.replaceState.WholeWord,
			)
			v.replaceMode = false
			v.replaceState = ReplaceState{}
			if count > 0 {
				v.errorMsg = fmt.Sprintf("Replaced %d occurrence(s)", count)
			} else {
				v.errorMsg = "No matches found"
			}
			return v, clearErrorAfter(statusTimeout)
		}
		return v, nil

	case tea.KeyCtrlN:
		// Next match
		if len(v.replaceState.Matches) > 0 {
			v.replaceState.CurrentMatch = (v.replaceState.CurrentMatch + 1) % len(v.replaceState.Matches)
			v.scrollToReplaceMatch()
		}
		return v, nil

	case tea.KeyCtrlP:
		// Previous match (Ctrl+P since Ctrl+Shift+N isn't reliably detected in terminals)
		if len(v.replaceState.Matches) > 0 {
			v.replaceState.CurrentMatch = (v.replaceState.CurrentMatch - 1 + len(v.replaceState.Matches)) % len(v.replaceState.Matches)
			v.scrollToReplaceMatch()
		}
		return v, nil
	}

	// Handle key string for toggles
	keyStr := msg.String()
	switch keyStr {
	case "alt+c":
		// Toggle case sensitivity
		v.replaceState.CaseSensitive = !v.replaceState.CaseSensitive
		v.updateReplaceMatches()
		return v, nil
	case "alt+w":
		// Toggle whole word
		v.replaceState.WholeWord = !v.replaceState.WholeWord
		v.updateReplaceMatches()
		return v, nil
	}

	// Character input
	if len(msg.Runes) > 0 {
		ch := string(msg.Runes[0])
		if v.replaceState.FocusReplace {
			v.replaceState.Replacement += ch
		} else {
			v.replaceState.Query += ch
			v.updateReplaceMatches()
		}
		return v, nil
	}

	return v, nil
}

// scrollToReplaceMatch scrolls the viewport to show the current replace match.
func (v *Viewer) scrollToReplaceMatch() {
	if v.replaceState.CurrentMatch < 0 || v.replaceState.CurrentMatch >= len(v.replaceState.Matches) {
		return
	}
	matchLine := v.replaceState.Matches[v.replaceState.CurrentMatch][0]
	contentHeight := v.Height - 4 // header + status + replace prompt (2 lines)
	if matchLine < v.Offset {
		v.Offset = matchLine
	} else if matchLine >= v.Offset+contentHeight {
		v.Offset = matchLine - contentHeight + 1
	}
}

// applyReplaceHighlights overlays match highlighting on a content line during replace mode.
// contentLine is the raw text, highlightedLine has syntax highlighting, lineIdx is the buffer line index.
// Returns the line with matches highlighted: current match in orange, others in yellow.
func (v *Viewer) applyReplaceHighlights(contentLine, highlightedLine string, lineIdx int) string {
	// Collect matches on this line
	type lineMatch struct {
		col       int
		len       int
		isCurrent bool
	}
	var lm []lineMatch
	queryLen := len([]rune(v.replaceState.Query))
	for idx, m := range v.replaceState.Matches {
		if m[0] == lineIdx {
			lm = append(lm, lineMatch{
				col:       m[1],
				len:       queryLen,
				isCurrent: idx == v.replaceState.CurrentMatch,
			})
		}
	}
	if len(lm) == 0 {
		return highlightedLine
	}

	// Work on plain content (no ANSI) for accurate rune positions
	plainRunes := []rune(contentLine)
	var sb strings.Builder
	prev := 0
	for _, m := range lm {
		start := m.col
		end := m.col + m.len
		if start > len(plainRunes) {
			start = len(plainRunes)
		}
		if end > len(plainRunes) {
			end = len(plainRunes)
		}
		if start < prev {
			start = prev
		}
		if end <= start {
			continue
		}
		sb.WriteString(string(plainRunes[prev:start]))
		// Choose highlight color
		if m.isCurrent {
			sb.WriteString(fmt.Sprintf("\x1b[48;5;%dm\x1b[38;5;%dm", SearchCurrentBg, SearchCurrentFg))
		} else {
			sb.WriteString(fmt.Sprintf("\x1b[48;5;%dm\x1b[38;5;%dm", SearchMatchBg, SearchMatchFg))
		}
		sb.WriteString(string(plainRunes[start:end]))
		sb.WriteString("\x1b[0m")
		prev = end
	}
	if prev < len(plainRunes) {
		sb.WriteString(string(plainRunes[prev:]))
	}
	return sb.String()
}

// renderReplacePrompt returns the 2-line find/replace prompt for the status area.
func (v *Viewer) renderReplacePrompt() string {
	// Match counter
	matchCount := len(v.replaceState.Matches)
	currentNum := 0
	if matchCount > 0 && v.replaceState.CurrentMatch >= 0 {
		currentNum = v.replaceState.CurrentMatch + 1
	}
	counter := fmt.Sprintf("%d of %d", currentNum, matchCount)

	// Toggle indicators
	csFlag := " "
	if v.replaceState.CaseSensitive {
		csFlag = "*"
	}
	wwFlag := " "
	if v.replaceState.WholeWord {
		wwFlag = "*"
	}
	toggles := fmt.Sprintf("[%sAa] [%sW]", csFlag, wwFlag)

	// Field cursors
	findCursor := " "
	replaceCursor := " "
	if !v.replaceState.FocusReplace {
		findCursor = "\x1b[7m \x1b[0m" // inverse block cursor
	} else {
		replaceCursor = "\x1b[7m \x1b[0m"
	}

	findLine := fmt.Sprintf(" Find: %s%s  %s  %s", v.replaceState.Query, findCursor, toggles, counter)
	replaceLine := fmt.Sprintf(" Replace: %s%s  [Enter] replace  [Ctrl+A] all  [Esc] cancel",
		v.replaceState.Replacement, replaceCursor)

	// Truncate to terminal width
	if len(findLine) > v.Width {
		findLine = findLine[:v.Width]
	}
	if len(replaceLine) > v.Width {
		replaceLine = replaceLine[:v.Width]
	}

	return strings.Join([]string{findLine, replaceLine}, "\n")
}
