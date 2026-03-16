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
	got := search.FindMatches([]string{}, "foo")
	if len(got) != 0 {
		t.Errorf("FindMatches([], %q) = %v; want []", "foo", got)
	}
}

func TestFindMatches_EmptyQuery(t *testing.T) {
	got := search.FindMatches([]string{"hello"}, "")
	if len(got) != 0 {
		t.Errorf("FindMatches([%q], %q) = %v; want []", "hello", "", got)
	}
}

func TestFindMatches_SimpleMatch(t *testing.T) {
	got := search.FindMatches([]string{"hello world"}, "hello")
	want := []search.Match{{LineIndex: 0, PlainStart: 0, PlainEnd: 5}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("FindMatches([%q], %q) = %v; want %v", "hello world", "hello", got, want)
	}
}

func TestFindMatches_CaseInsensitive(t *testing.T) {
	got := search.FindMatches([]string{"Hello World"}, "hello")
	want := []search.Match{{LineIndex: 0, PlainStart: 0, PlainEnd: 5}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("FindMatches([%q], %q) = %v; want %v", "Hello World", "hello", got, want)
	}
}

func TestFindMatches_MultipleMatchesSameLine(t *testing.T) {
	got := search.FindMatches([]string{"foo foo foo"}, "foo")
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
	got := search.FindMatches([]string{input}, "foo")
	want := []search.Match{{LineIndex: 0, PlainStart: 0, PlainEnd: 3}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("FindMatches([%q], %q) = %v; want %v", input, "foo", got, want)
	}
}

func TestFindMatches_MultipleLines(t *testing.T) {
	lines := []string{"line1 foo", "line2", "line3 Foo"}
	got := search.FindMatches(lines, "foo")
	want := []search.Match{
		{LineIndex: 0, PlainStart: 6, PlainEnd: 9},
		{LineIndex: 2, PlainStart: 6, PlainEnd: 9},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("FindMatches(%v, %q) = %v; want %v", lines, "foo", got, want)
	}
}

func TestFindMatches_NonOverlappingAdjacent(t *testing.T) {
	got := search.FindMatches([]string{"abcabc"}, "abc")
	want := []search.Match{
		{LineIndex: 0, PlainStart: 0, PlainEnd: 3},
		{LineIndex: 0, PlainStart: 3, PlainEnd: 6},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("FindMatches([%q], %q) = %v; want %v", "abcabc", "abc", got, want)
	}
}

func TestFindMatches_NonOverlappingConsumes(t *testing.T) {
	got := search.FindMatches([]string{"aaa"}, "aa")
	want := []search.Match{
		{LineIndex: 0, PlainStart: 0, PlainEnd: 2},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("FindMatches([%q], %q) = %v; want %v", "aaa", "aa", got, want)
	}
}

func TestFindMatches_NoMatch(t *testing.T) {
	got := search.FindMatches([]string{"hello"}, "xyz")
	if len(got) != 0 {
		t.Errorf("FindMatches([%q], %q) = %v; want []", "hello", "xyz", got)
	}
}
