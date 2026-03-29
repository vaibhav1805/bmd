// Package search provides pure functions for finding term occurrences in
// ANSI-styled terminal lines.
package search

import (
	"regexp"
	"strings"
	"unicode"
)

// ansiEscape matches ANSI terminal escape sequences such as color codes.
var ansiEscape = regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]`)

// StripANSI removes all ANSI escape sequences from s, returning plain text.
func StripANSI(s string) string {
	return ansiEscape.ReplaceAllString(s, "")
}

// Match records the position of a single search-term occurrence within the
// lines slice passed to FindMatches.
type Match struct {
	LineIndex  int // 0-based index in the lines slice
	PlainStart int // rune offset in the stripped (plain) line (inclusive)
	PlainEnd   int // rune offset in the stripped (plain) line (exclusive)
}

// FindMatches returns all non-overlapping occurrences of query in lines.
// Lines may contain ANSI escape sequences; they are stripped before matching.
// Matches are ordered by LineIndex then PlainStart.
// When caseSensitive is false, matching is case-insensitive (default behavior).
// When wholeWord is true, matches must be bounded by non-alphanumeric characters.
// Returns an empty (nil) slice when no matches are found.
func FindMatches(lines []string, query string, caseSensitive, wholeWord bool) []Match {
	if len(query) == 0 {
		return nil
	}

	var searchQuery []rune
	if caseSensitive {
		searchQuery = []rune(query)
	} else {
		searchQuery = []rune(strings.ToLower(query))
	}
	queryLen := len(searchQuery)

	var matches []Match

	for lineIdx, line := range lines {
		plain := StripANSI(line)
		var plainRunes []rune
		if caseSensitive {
			plainRunes = []rune(plain)
		} else {
			plainRunes = []rune(strings.ToLower(plain))
		}
		n := len(plainRunes)

		i := 0
		for i <= n-queryLen {
			if runesEqual(plainRunes[i:i+queryLen], searchQuery) {
				if wholeWord && !isWholeWord(plainRunes, i, i+queryLen) {
					i++
					continue
				}
				matches = append(matches, Match{
					LineIndex:  lineIdx,
					PlainStart: i,
					PlainEnd:   i + queryLen,
				})
				i += queryLen // non-overlapping: advance past the match
			} else {
				i++
			}
		}
	}

	return matches
}

// isWholeWord checks that the match at [start, end) in runes is bounded by
// non-word characters (or string boundaries). A word character is a letter,
// digit, or underscore.
func isWholeWord(runes []rune, start, end int) bool {
	if start > 0 && isWordChar(runes[start-1]) {
		return false
	}
	if end < len(runes) && isWordChar(runes[end]) {
		return false
	}
	return true
}

// isWordChar returns true if r is a letter, digit, or underscore.
func isWordChar(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_'
}

// runesEqual reports whether two rune slices are identical.
func runesEqual(a, b []rune) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
