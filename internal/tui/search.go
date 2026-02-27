// Package tui provides the interactive terminal user interface for bmd.
package tui

import (
	"github.com/bmd/bmd/internal/search"
	"github.com/bmd/bmd/internal/theme"
)

// Search highlight ANSI 256-color constants.
// These are hardcoded (not in Theme) because search highlighting is a
// functional UI state, not a document style.
const (
	SearchMatchBg    = 226 // bright yellow background for all matches
	SearchMatchFg    = 16  // black foreground on yellow
	SearchCurrentBg  = 214 // orange background for the focused match
	SearchCurrentFg  = 16  // black foreground on orange
)

// SearchState manages the active search query and match navigation.
type SearchState struct {
	Active  bool           // true when a search has been committed (Enter pressed)
	Query   string         // current committed query
	Matches []search.Match // all matches in current document
	Current int            // index of focused match (-1 = none)
}

// NewSearchState returns a zeroed SearchState with Current = -1.
func NewSearchState() SearchState {
	return SearchState{Current: -1}
}

// Run executes search.FindMatches against displayLines for the current Query.
// Updates Matches, sets Active to true, resets Current to 0 if matches are
// found, or -1 if none.
func (s *SearchState) Run(displayLines []string) {
	s.Matches = search.FindMatches(displayLines, s.Query)
	s.Active = true
	if len(s.Matches) > 0 {
		s.Current = 0
	} else {
		s.Current = -1
	}
}

// Next advances to the next match (wraps around). Returns the new Current index.
// Returns -1 if no matches exist.
func (s *SearchState) Next() int {
	if len(s.Matches) == 0 {
		return -1
	}
	if s.Current < 0 {
		s.Current = 0
	} else {
		s.Current = (s.Current + 1) % len(s.Matches)
	}
	return s.Current
}

// Prev goes to the previous match (wraps around). Returns the new Current index.
// Returns -1 if no matches exist.
func (s *SearchState) Prev() int {
	if len(s.Matches) == 0 {
		return -1
	}
	if s.Current < 0 {
		s.Current = len(s.Matches) - 1
	} else {
		s.Current = (s.Current - 1 + len(s.Matches)) % len(s.Matches)
	}
	return s.Current
}

// CurrentMatch returns the Match at Current index, or a zero-value Match and
// false if no match is focused.
func (s *SearchState) CurrentMatch() (search.Match, bool) {
	if s.Current < 0 || s.Current >= len(s.Matches) {
		return search.Match{}, false
	}
	return s.Matches[s.Current], true
}

// ApplyHighlights returns a copy of displayLines with match highlighting
// injected. All matches get a yellow background highlight; the current match
// (state.Current) gets an orange background to distinguish it as focused.
//
// Highlight strategy (simpler approach): strip ANSI from the line, apply
// highlights to the plain text. Original ANSI styling is lost on matched lines —
// this is an acceptable tradeoff for Phase 2 search highlighting.
//
// The th parameter is accepted for future extensibility but highlights use
// hardcoded ANSI 256-color constants (SearchMatchBg, SearchCurrentBg).
func ApplyHighlights(lines []string, state SearchState, th theme.Theme) []string {
	if !state.Active || len(state.Matches) == 0 {
		// Return a copy unchanged
		out := make([]string, len(lines))
		copy(out, lines)
		return out
	}

	// Group matches by line for efficient lookup.
	matchesByLine := make(map[int][]indexedMatch)
	for i, m := range state.Matches {
		matchesByLine[m.LineIndex] = append(matchesByLine[m.LineIndex], indexedMatch{
			Match:      m,
			MatchIndex: i,
		})
	}

	out := make([]string, len(lines))
	for lineIdx, line := range lines {
		lineMatches, hasMatches := matchesByLine[lineIdx]
		if !hasMatches {
			out[lineIdx] = line
			continue
		}

		// Strip ANSI from this line for plain-text highlighting.
		plain := search.StripANSI(line)
		plainRunes := []rune(plain)

		var sb []byte
		prevEnd := 0 // rune index in plainRunes

		for _, im := range lineMatches {
			start := im.PlainStart
			end := im.PlainEnd

			// Clamp to valid range.
			if start > len(plainRunes) {
				start = len(plainRunes)
			}
			if end > len(plainRunes) {
				end = len(plainRunes)
			}
			if start < prevEnd {
				start = prevEnd
			}
			if end <= start {
				continue
			}

			// Copy plain text before this match.
			sb = append(sb, []byte(string(plainRunes[prevEnd:start]))...)

			// Choose highlight colors: current match gets orange, others get yellow.
			var bgCode, fgCode string
			if im.MatchIndex == state.Current {
				bgCode = theme.BgCode(SearchCurrentBg)
				fgCode = theme.FgCode(SearchCurrentFg)
			} else {
				bgCode = theme.BgCode(SearchMatchBg)
				fgCode = theme.FgCode(SearchMatchFg)
			}

			// Inject: open color, matched text, reset.
			sb = append(sb, []byte(bgCode+fgCode)...)
			sb = append(sb, []byte(string(plainRunes[start:end]))...)
			sb = append(sb, []byte(theme.Reset)...)

			prevEnd = end
		}

		// Append remaining text after last match.
		if prevEnd < len(plainRunes) {
			sb = append(sb, []byte(string(plainRunes[prevEnd:]))...)
		}

		out[lineIdx] = string(sb)
	}

	return out
}

// indexedMatch pairs a search.Match with its global index in SearchState.Matches.
// This is used internally by ApplyHighlights to identify the current match.
type indexedMatch struct {
	search.Match
	MatchIndex int
}
