// Package search provides pure functions for finding term occurrences in
// ANSI-styled terminal lines.
package search

import (
	"regexp"
	"strings"
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

// FindMatches returns all non-overlapping, case-insensitive occurrences of
// query in lines. Lines may contain ANSI escape sequences; they are stripped
// before matching. Matches are ordered by LineIndex then PlainStart.
// Returns an empty (nil) slice when no matches are found.
func FindMatches(lines []string, query string) []Match {
	if len(query) == 0 {
		return nil
	}

	lowerQuery := []rune(strings.ToLower(query))
	queryLen := len(lowerQuery)

	var matches []Match

	for lineIdx, line := range lines {
		plain := StripANSI(line)
		plainRunes := []rune(strings.ToLower(plain))
		n := len(plainRunes)

		i := 0
		for i <= n-queryLen {
			if runesEqual(plainRunes[i:i+queryLen], lowerQuery) {
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
