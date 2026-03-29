package search_test

import (
	"reflect"
	"testing"

	"github.com/bmd/bmd/internal/search"
)

// StripANSI tests

func TestStripANSI_Plain(t *testing.T) {
	got := search.StripANSI("hello")
	want := "hello"
	if got != want {
		t.Errorf("StripANSI(%q) = %q; want %q", "hello", got, want)
	}
}

func TestStripANSI_ColorCode(t *testing.T) {
	input := "\x1b[38;5;39mhello\x1b[0m"
	got := search.StripANSI(input)
	want := "hello"
	if got != want {
		t.Errorf("StripANSI(%q) = %q; want %q", input, got, want)
	}
}

func TestStripANSI_BoldUnderline(t *testing.T) {
	input := "\x1b[1m\x1b[4mBold Underline\x1b[0m"
	got := search.StripANSI(input)
	want := "Bold Underline"
	if got != want {
		t.Errorf("StripANSI(%q) = %q; want %q", input, got, want)
	}
}

func TestStripANSI_Empty(t *testing.T) {
	got := search.StripANSI("")
	want := ""
	if got != want {
		t.Errorf("StripANSI(%q) = %q; want %q", "", got, want)
	}
}

func TestStripANSI_OnlyEscape(t *testing.T) {
	input := "\x1b[0m"
	got := search.StripANSI(input)
	want := ""
	if got != want {
		t.Errorf("StripANSI(%q) = %q; want %q", input, got, want)
	}
}

// FindMatches tests

func TestFindMatches_EmptyLines(t *testing.T) {
	got := search.FindMatches([]string{}, "foo", false, false, false)
	if len(got) != 0 {
		t.Errorf("FindMatches([], %q) = %v; want []", "foo", got)
	}
}

func TestFindMatches_EmptyQuery(t *testing.T) {
	got := search.FindMatches([]string{"hello"}, "", false, false, false)
	if len(got) != 0 {
		t.Errorf("FindMatches([%q], %q) = %v; want []", "hello", "", got)
	}
}

func TestFindMatches_SimpleMatch(t *testing.T) {
	got := search.FindMatches([]string{"hello world"}, "hello", false, false, false)
	want := []search.Match{{LineIndex: 0, PlainStart: 0, PlainEnd: 5}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("FindMatches([%q], %q) = %v; want %v", "hello world", "hello", got, want)
	}
}

func TestFindMatches_CaseInsensitive(t *testing.T) {
	got := search.FindMatches([]string{"Hello World"}, "hello", false, false, false)
	want := []search.Match{{LineIndex: 0, PlainStart: 0, PlainEnd: 5}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("FindMatches([%q], %q) = %v; want %v", "Hello World", "hello", got, want)
	}
}

func TestFindMatches_MultipleMatchesSameLine(t *testing.T) {
	got := search.FindMatches([]string{"foo foo foo"}, "foo", false, false, false)
	want := []search.Match{
		{LineIndex: 0, PlainStart: 0, PlainEnd: 3},
		{LineIndex: 0, PlainStart: 4, PlainEnd: 7},
		{LineIndex: 0, PlainStart: 8, PlainEnd: 11},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("FindMatches([%q], %q) = %v; want %v", "foo foo foo", "foo", got, want)
	}
}

func TestFindMatches_ANSIStrippedForSearch(t *testing.T) {
	input := "\x1b[39mfoo\x1b[0m bar"
	got := search.FindMatches([]string{input}, "foo", false, false, false)
	want := []search.Match{{LineIndex: 0, PlainStart: 0, PlainEnd: 3}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("FindMatches([%q], %q) = %v; want %v", input, "foo", got, want)
	}
}

func TestFindMatches_MultipleLines(t *testing.T) {
	lines := []string{"line1 foo", "line2", "line3 Foo"}
	got := search.FindMatches(lines, "foo", false, false, false)
	want := []search.Match{
		{LineIndex: 0, PlainStart: 6, PlainEnd: 9},
		{LineIndex: 2, PlainStart: 6, PlainEnd: 9},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("FindMatches(%v, %q) = %v; want %v", lines, "foo", got, want)
	}
}

func TestFindMatches_NonOverlappingAdjacent(t *testing.T) {
	got := search.FindMatches([]string{"abcabc"}, "abc", false, false, false)
	want := []search.Match{
		{LineIndex: 0, PlainStart: 0, PlainEnd: 3},
		{LineIndex: 0, PlainStart: 3, PlainEnd: 6},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("FindMatches([%q], %q) = %v; want %v", "abcabc", "abc", got, want)
	}
}

func TestFindMatches_NonOverlappingConsumes(t *testing.T) {
	got := search.FindMatches([]string{"aaa"}, "aa", false, false, false)
	want := []search.Match{
		{LineIndex: 0, PlainStart: 0, PlainEnd: 2},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("FindMatches([%q], %q) = %v; want %v", "aaa", "aa", got, want)
	}
}

func TestFindMatches_NoMatch(t *testing.T) {
	got := search.FindMatches([]string{"hello"}, "xyz", false, false, false)
	if len(got) != 0 {
		t.Errorf("FindMatches([%q], %q) = %v; want []", "hello", "xyz", got)
	}
}

// Regex search tests

func TestFindMatches_RegexDigits(t *testing.T) {
	lines := []string{"abc 42 def 999 ghi"}
	got := search.FindMatches(lines, `\d+`, false, false, true)
	if len(got) != 2 {
		t.Fatalf("expected 2 regex matches, got %d: %v", len(got), got)
	}
	if got[0].PlainStart != 4 || got[0].PlainEnd != 6 {
		t.Errorf("match[0] = (%d,%d); want (4,6)", got[0].PlainStart, got[0].PlainEnd)
	}
	if got[1].PlainStart != 11 || got[1].PlainEnd != 14 {
		t.Errorf("match[1] = (%d,%d); want (11,14)", got[1].PlainStart, got[1].PlainEnd)
	}
}

func TestFindMatches_RegexCaseSensitive(t *testing.T) {
	lines := []string{"Hello hello HELLO"}
	// Case-sensitive regex: [A-Z][a-z]+ should match "Hello" only
	got := search.FindMatches(lines, `[A-Z][a-z]+`, true, false, true)
	if len(got) != 1 {
		t.Fatalf("expected 1 match, got %d: %v", len(got), got)
	}
	if got[0].PlainStart != 0 || got[0].PlainEnd != 5 {
		t.Errorf("match = (%d,%d); want (0,5)", got[0].PlainStart, got[0].PlainEnd)
	}
}

func TestFindMatches_RegexCaseInsensitive(t *testing.T) {
	lines := []string{"Hello hello HELLO"}
	// Case-insensitive regex: hello should match all three
	got := search.FindMatches(lines, `hello`, false, false, true)
	if len(got) != 3 {
		t.Fatalf("expected 3 matches, got %d: %v", len(got), got)
	}
}

func TestFindMatches_RegexWholeWord(t *testing.T) {
	lines := []string{"foobar foo barfoo"}
	got := search.FindMatches(lines, `foo`, false, true, true)
	if len(got) != 1 {
		t.Fatalf("expected 1 whole-word regex match, got %d: %v", len(got), got)
	}
	if got[0].PlainStart != 7 || got[0].PlainEnd != 10 {
		t.Errorf("match = (%d,%d); want (7,10)", got[0].PlainStart, got[0].PlainEnd)
	}
}

func TestFindMatches_RegexInvalidPattern(t *testing.T) {
	lines := []string{"hello world"}
	got := search.FindMatches(lines, `[invalid`, false, false, true)
	if got != nil {
		t.Errorf("expected nil for invalid regex, got %v", got)
	}
}

func TestFindMatches_RegexNoMatchReturnsNil(t *testing.T) {
	lines := []string{"hello world"}
	got := search.FindMatches(lines, `\d+`, false, false, true)
	if got != nil {
		t.Errorf("expected nil for no regex matches, got %v", got)
	}
}

func TestIsValidRegex(t *testing.T) {
	if !search.IsValidRegex(`\d+`) {
		t.Error("expected valid regex")
	}
	if search.IsValidRegex(`[invalid`) {
		t.Error("expected invalid regex")
	}
}
