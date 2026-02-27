package tui

import (
	"strings"
	"testing"

	"github.com/bmd/bmd/internal/search"
	"github.com/bmd/bmd/internal/theme"
)

// --- SearchState tests ---

func TestNewSearchState_Defaults(t *testing.T) {
	s := NewSearchState()
	if s.Active {
		t.Error("Expected Active=false on NewSearchState")
	}
	if s.Query != "" {
		t.Errorf("Expected empty Query, got %q", s.Query)
	}
	if s.Current != -1 {
		t.Errorf("Expected Current=-1, got %d", s.Current)
	}
	if len(s.Matches) != 0 {
		t.Errorf("Expected empty Matches, got %d", len(s.Matches))
	}
}

func TestSearchState_Run_FindsMatches(t *testing.T) {
	lines := []string{"Hello world", "foo bar foo", "no match here"}
	s := NewSearchState()
	s.Query = "foo"
	s.Run(lines)

	if !s.Active {
		t.Error("Expected Active=true after Run")
	}
	if len(s.Matches) != 2 {
		t.Fatalf("Expected 2 matches, got %d", len(s.Matches))
	}
	if s.Current != 0 {
		t.Errorf("Expected Current=0 after Run with matches, got %d", s.Current)
	}
}

func TestSearchState_Run_NoMatches(t *testing.T) {
	lines := []string{"Hello world", "foo bar"}
	s := NewSearchState()
	s.Query = "zzzzz"
	s.Run(lines)

	if !s.Active {
		t.Error("Expected Active=true after Run even with no matches")
	}
	if len(s.Matches) != 0 {
		t.Errorf("Expected 0 matches, got %d", len(s.Matches))
	}
	if s.Current != -1 {
		t.Errorf("Expected Current=-1 with no matches, got %d", s.Current)
	}
}

func TestSearchState_Run_CaseInsensitive(t *testing.T) {
	lines := []string{"Hello WORLD", "hello World"}
	s := NewSearchState()
	s.Query = "hello"
	s.Run(lines)

	if len(s.Matches) != 2 {
		t.Fatalf("Expected 2 case-insensitive matches, got %d", len(s.Matches))
	}
}

func TestSearchState_Next_Wraps(t *testing.T) {
	lines := []string{"a a a"}
	s := NewSearchState()
	s.Query = "a"
	s.Run(lines)
	// Should have 3 matches
	if len(s.Matches) != 3 {
		t.Fatalf("Expected 3 matches, got %d", len(s.Matches))
	}

	// Already at 0 after Run
	s.Next() // -> 1
	if s.Current != 1 {
		t.Errorf("Expected Current=1 after Next, got %d", s.Current)
	}
	s.Next() // -> 2
	if s.Current != 2 {
		t.Errorf("Expected Current=2 after Next, got %d", s.Current)
	}
	s.Next() // -> 0 (wrap)
	if s.Current != 0 {
		t.Errorf("Expected Current=0 after wrap, got %d", s.Current)
	}
}

func TestSearchState_Prev_Wraps(t *testing.T) {
	lines := []string{"a a"}
	s := NewSearchState()
	s.Query = "a"
	s.Run(lines)
	// Current=0 after Run

	s.Prev() // should wrap to 1 (last)
	if s.Current != 1 {
		t.Errorf("Expected Current=1 after Prev from 0 (wrap), got %d", s.Current)
	}
	s.Prev() // -> 0
	if s.Current != 0 {
		t.Errorf("Expected Current=0 after Prev, got %d", s.Current)
	}
}

func TestSearchState_Next_NoMatches(t *testing.T) {
	s := NewSearchState()
	idx := s.Next()
	if idx != -1 {
		t.Errorf("Expected -1 from Next with no matches, got %d", idx)
	}
}

func TestSearchState_Prev_NoMatches(t *testing.T) {
	s := NewSearchState()
	idx := s.Prev()
	if idx != -1 {
		t.Errorf("Expected -1 from Prev with no matches, got %d", idx)
	}
}

func TestSearchState_CurrentMatch_WithMatch(t *testing.T) {
	lines := []string{"find me here"}
	s := NewSearchState()
	s.Query = "find"
	s.Run(lines)

	m, ok := s.CurrentMatch()
	if !ok {
		t.Fatal("Expected CurrentMatch to return ok=true")
	}
	if m.LineIndex != 0 {
		t.Errorf("Expected LineIndex=0, got %d", m.LineIndex)
	}
	if m.PlainStart != 0 {
		t.Errorf("Expected PlainStart=0, got %d", m.PlainStart)
	}
}

func TestSearchState_CurrentMatch_NoFocus(t *testing.T) {
	s := NewSearchState()
	_, ok := s.CurrentMatch()
	if ok {
		t.Error("Expected ok=false when Current=-1")
	}
}

// --- ApplyHighlights tests ---

func TestApplyHighlights_Inactive_ReturnsUnchanged(t *testing.T) {
	lines := []string{"hello world", "foo bar"}
	state := NewSearchState()
	th := theme.NewThemeForScheme(theme.Dark)

	result := ApplyHighlights(lines, state, th)
	if len(result) != len(lines) {
		t.Fatalf("Expected %d lines, got %d", len(lines), len(result))
	}
	for i, line := range lines {
		if result[i] != line {
			t.Errorf("Line %d modified when no search active: got %q, want %q", i, result[i], line)
		}
	}
}

func TestApplyHighlights_NoMatches_ReturnsUnchanged(t *testing.T) {
	lines := []string{"hello world"}
	state := SearchState{Active: true, Query: "zzz", Matches: nil, Current: -1}
	th := theme.NewThemeForScheme(theme.Dark)

	result := ApplyHighlights(lines, state, th)
	if result[0] != lines[0] {
		t.Errorf("Expected unchanged line, got %q", result[0])
	}
}

func TestApplyHighlights_SingleMatch_ContainsHighlight(t *testing.T) {
	lines := []string{"hello world"}
	state := SearchState{
		Active:  true,
		Query:   "world",
		Current: 0,
		Matches: []search.Match{
			{LineIndex: 0, PlainStart: 6, PlainEnd: 11},
		},
	}
	th := theme.NewThemeForScheme(theme.Dark)

	result := ApplyHighlights(lines, state, th)
	if len(result) != 1 {
		t.Fatalf("Expected 1 result line, got %d", len(result))
	}
	// The result should contain "hello " before the highlight.
	if !strings.HasPrefix(result[0], "hello ") {
		t.Errorf("Expected line to start with 'hello ', got %q", result[0])
	}
	// Should contain the matched text "world".
	if !strings.Contains(result[0], "world") {
		t.Errorf("Expected result to contain 'world', got %q", result[0])
	}
	// Should contain an ANSI escape (highlight).
	if !strings.Contains(result[0], "\x1b[") {
		t.Errorf("Expected ANSI highlight codes in result, got %q", result[0])
	}
	// Should end with a reset code.
	if !strings.Contains(result[0], "\x1b[0m") {
		t.Errorf("Expected reset code in result, got %q", result[0])
	}
}

func TestApplyHighlights_CurrentVsNonCurrent_DifferentColors(t *testing.T) {
	// Line 0: match 0 (non-current), Line 1: match 1 (current)
	lines := []string{"find this", "find this"}
	state := SearchState{
		Active:  true,
		Query:   "find",
		Current: 1, // second match is focused
		Matches: []search.Match{
			{LineIndex: 0, PlainStart: 0, PlainEnd: 4},
			{LineIndex: 1, PlainStart: 0, PlainEnd: 4},
		},
	}
	th := theme.NewThemeForScheme(theme.Dark)

	result := ApplyHighlights(lines, state, th)
	if len(result) != 2 {
		t.Fatalf("Expected 2 result lines, got %d", len(result))
	}

	// Non-current match should contain bg=226 (SearchMatchBg = yellow).
	nonCurrentBg := "\x1b[48;5;226m"
	if !strings.Contains(result[0], nonCurrentBg) {
		t.Errorf("Expected non-current match to have yellow bg (226), got %q", result[0])
	}

	// Current match should contain bg=214 (SearchCurrentBg = orange).
	currentBg := "\x1b[48;5;214m"
	if !strings.Contains(result[1], currentBg) {
		t.Errorf("Expected current match to have orange bg (214), got %q", result[1])
	}
}

func TestApplyHighlights_PreservesUnmatchedLines(t *testing.T) {
	lines := []string{"no match here", "find this", "also no match"}
	state := SearchState{
		Active:  true,
		Query:   "find",
		Current: 0,
		Matches: []search.Match{
			{LineIndex: 1, PlainStart: 0, PlainEnd: 4},
		},
	}
	th := theme.NewThemeForScheme(theme.Dark)

	result := ApplyHighlights(lines, state, th)
	if result[0] != "no match here" {
		t.Errorf("Line 0 should be unchanged, got %q", result[0])
	}
	if result[2] != "also no match" {
		t.Errorf("Line 2 should be unchanged, got %q", result[2])
	}
	// Line 1 should have highlighting.
	if !strings.Contains(result[1], "\x1b[") {
		t.Errorf("Expected highlighted line 1, got %q", result[1])
	}
}

func TestApplyHighlights_MultipleMatchesOnOneLine(t *testing.T) {
	lines := []string{"foo bar foo"}
	state := SearchState{
		Active:  true,
		Query:   "foo",
		Current: 0,
		Matches: []search.Match{
			{LineIndex: 0, PlainStart: 0, PlainEnd: 3},
			{LineIndex: 0, PlainStart: 8, PlainEnd: 11},
		},
	}
	th := theme.NewThemeForScheme(theme.Dark)

	result := ApplyHighlights(lines, state, th)
	// Both "foo" occurrences should appear in the result.
	stripped := search.StripANSI(result[0])
	if stripped != "foo bar foo" {
		t.Errorf("Expected stripped content 'foo bar foo', got %q", stripped)
	}
	// Should have 2 reset codes (one per highlighted match).
	resetCount := strings.Count(result[0], "\x1b[0m")
	if resetCount < 2 {
		t.Errorf("Expected at least 2 reset codes for 2 matches, got %d", resetCount)
	}
}

func TestApplyHighlights_LinesNotAffectedBySearch_AreCopied(t *testing.T) {
	original := []string{"line one", "line two"}
	state := SearchState{Active: true, Query: "xyz", Matches: nil, Current: -1}
	th := theme.NewThemeForScheme(theme.Dark)

	result := ApplyHighlights(original, state, th)
	// Modify original to prove result is a copy.
	original[0] = "mutated"
	if result[0] != "line one" {
		t.Errorf("Expected result to be independent copy, got %q", result[0])
	}
}
