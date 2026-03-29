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
// When useRegex is true, query is compiled as a regular expression pattern.
// If the regex pattern is invalid, returns nil (graceful degradation).
// Returns an empty (nil) slice when no matches are found.
func FindMatches(lines []string, query string, caseSensitive, wholeWord, useRegex bool) []Match {
	if len(query) == 0 {
		return nil
	}

	if useRegex {
		return findMatchesRegex(lines, query, caseSensitive, wholeWord)
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

// findMatchesRegex handles regex-based matching. Returns nil if the pattern
// fails to compile (graceful degradation).
func findMatchesRegex(lines []string, pattern string, caseSensitive, wholeWord bool) []Match {
	if !caseSensitive {
		pattern = "(?i)" + pattern
	}
	if wholeWord {
		pattern = `\b(?:` + pattern + `)\b`
	}
	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil
	}

	var matches []Match
	for lineIdx, line := range lines {
		plain := StripANSI(line)
		plainRunes := []rune(plain)
		locs := re.FindAllStringIndex(plain, -1)
		for _, loc := range locs {
			// Convert byte offsets to rune offsets
			startRune := len([]rune(plain[:loc[0]]))
			endRune := len([]rune(plain[:loc[1]]))
			if endRune > len(plainRunes) {
				endRune = len(plainRunes)
			}
			matches = append(matches, Match{
				LineIndex:  lineIdx,
				PlainStart: startRune,
				PlainEnd:   endRune,
			})
		}
	}
	return matches
}

// IsValidRegex returns true if the given pattern compiles as a valid regex.
func IsValidRegex(pattern string) bool {
	_, err := regexp.Compile(pattern)
	return err == nil
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
