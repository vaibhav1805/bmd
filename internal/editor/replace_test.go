package editor

import (
	"testing"
)

// TestFindAllBasic tests basic case-insensitive find.
func TestFindAllBasic(t *testing.T) {
	tb := NewTextBuffer([]string{"foo bar foo", "baz foo qux"})
	matches := tb.FindAll("foo", false, false)

	if len(matches) != 3 {
		t.Fatalf("Expected 3 matches, got %d", len(matches))
	}
	// First match: line 0, col 0
	if matches[0] != [2]int{0, 0} {
		t.Errorf("Expected match at (0,0), got %v", matches[0])
	}
	// Second match: line 0, col 8
	if matches[1] != [2]int{0, 8} {
		t.Errorf("Expected match at (0,8), got %v", matches[1])
	}
	// Third match: line 1, col 4
	if matches[2] != [2]int{1, 4} {
		t.Errorf("Expected match at (1,4), got %v", matches[2])
	}
}

// TestFindAllCaseSensitive tests case-sensitive find.
func TestFindAllCaseSensitive(t *testing.T) {
	tb := NewTextBuffer([]string{"Foo bar foo", "FOO baz"})

	// Case-sensitive: "Foo" should match only the capitalized one
	matches := tb.FindAll("Foo", true, false)
	if len(matches) != 1 {
		t.Fatalf("Expected 1 case-sensitive match, got %d", len(matches))
	}
	if matches[0] != [2]int{0, 0} {
		t.Errorf("Expected match at (0,0), got %v", matches[0])
	}

	// Case-insensitive: "foo" should match all three
	matches = tb.FindAll("foo", false, false)
	if len(matches) != 3 {
		t.Fatalf("Expected 3 case-insensitive matches, got %d", len(matches))
	}
}

// TestFindAllWholeWord tests whole-word find.
func TestFindAllWholeWord(t *testing.T) {
	tb := NewTextBuffer([]string{"word wordsmith sword words"})

	// Whole-word: "word" should match only standalone "word"
	matches := tb.FindAll("word", false, true)
	if len(matches) != 1 {
		t.Fatalf("Expected 1 whole-word match, got %d", len(matches))
	}
	if matches[0] != [2]int{0, 0} {
		t.Errorf("Expected match at (0,0), got %v", matches[0])
	}

	// Without whole-word: "word" matches inside wordsmith, sword, words too
	matches = tb.FindAll("word", false, false)
	if len(matches) != 4 {
		t.Fatalf("Expected 4 non-whole-word matches, got %d", len(matches))
	}
}

// TestReplaceBasic tests basic find and replace all.
func TestReplaceBasic(t *testing.T) {
	tb := NewTextBuffer([]string{"foo bar foo", "baz foo qux"})

	count := tb.Replace("foo", "bar", false, false)
	if count != 3 {
		t.Fatalf("Expected 3 replacements, got %d", count)
	}

	lines := tb.GetLines()
	if lines[0] != "bar bar bar" {
		t.Errorf("Expected 'bar bar bar', got '%s'", lines[0])
	}
	if lines[1] != "baz bar qux" {
		t.Errorf("Expected 'baz bar qux', got '%s'", lines[1])
	}
}

// TestReplaceCaseSensitive tests case-sensitive replace.
func TestReplaceCaseSensitive(t *testing.T) {
	tb := NewTextBuffer([]string{"Foo bar foo FOO"})

	count := tb.Replace("Foo", "Baz", true, false)
	if count != 1 {
		t.Fatalf("Expected 1 replacement, got %d", count)
	}

	lines := tb.GetLines()
	if lines[0] != "Baz bar foo FOO" {
		t.Errorf("Expected 'Baz bar foo FOO', got '%s'", lines[0])
	}
}

// TestReplaceWholeWord tests whole-word replace.
func TestReplaceWholeWord(t *testing.T) {
	tb := NewTextBuffer([]string{"word wordsmith sword"})

	count := tb.Replace("word", "term", false, true)
	if count != 1 {
		t.Fatalf("Expected 1 replacement, got %d", count)
	}

	lines := tb.GetLines()
	if lines[0] != "term wordsmith sword" {
		t.Errorf("Expected 'term wordsmith sword', got '%s'", lines[0])
	}
}

// TestReplaceUndo tests that replace all is undoable as a single operation.
func TestReplaceUndo(t *testing.T) {
	tb := NewTextBuffer([]string{"foo bar foo", "baz foo qux"})

	tb.Replace("foo", "XXX", false, false)

	// Verify replacement happened
	lines := tb.GetLines()
	if lines[0] != "XXX bar XXX" {
		t.Errorf("Expected 'XXX bar XXX', got '%s'", lines[0])
	}

	// Single undo should revert ALL replacements
	tb.Undo()

	lines = tb.GetLines()
	if lines[0] != "foo bar foo" {
		t.Errorf("Expected 'foo bar foo' after undo, got '%s'", lines[0])
	}
	if lines[1] != "baz foo qux" {
		t.Errorf("Expected 'baz foo qux' after undo, got '%s'", lines[1])
	}
}

// TestReplaceOneBasic tests single match replacement.
func TestReplaceOneBasic(t *testing.T) {
	tb := NewTextBuffer([]string{"foo bar foo"})

	ok := tb.ReplaceOne(0, 8, "foo", "baz", false)
	if !ok {
		t.Fatal("Expected ReplaceOne to succeed")
	}

	lines := tb.GetLines()
	if lines[0] != "foo bar baz" {
		t.Errorf("Expected 'foo bar baz', got '%s'", lines[0])
	}
}

// TestReplaceNoMatch tests replace when query has no matches.
func TestReplaceNoMatch(t *testing.T) {
	tb := NewTextBuffer([]string{"hello world"})

	count := tb.Replace("xyz", "abc", false, false)
	if count != 0 {
		t.Errorf("Expected 0 replacements, got %d", count)
	}

	// Should not push undo state
	if tb.CanUndo() {
		t.Error("Expected no undo state when no replacements made")
	}
}

// TestReplaceEmptyQuery tests replace with empty query.
func TestReplaceEmptyQuery(t *testing.T) {
	tb := NewTextBuffer([]string{"hello"})

	count := tb.Replace("", "abc", false, false)
	if count != 0 {
		t.Errorf("Expected 0 replacements for empty query, got %d", count)
	}
}

// TestFindAllEmpty tests find with empty query.
func TestFindAllEmpty(t *testing.T) {
	tb := NewTextBuffer([]string{"hello"})

	matches := tb.FindAll("", false, false)
	if matches != nil {
		t.Errorf("Expected nil matches for empty query, got %v", matches)
	}
}

// TestReplaceWithDifferentLength tests replacement with different length strings.
func TestReplaceWithDifferentLength(t *testing.T) {
	tb := NewTextBuffer([]string{"ab ab ab"})

	count := tb.Replace("ab", "xyz", false, false)
	if count != 3 {
		t.Fatalf("Expected 3 replacements, got %d", count)
	}

	lines := tb.GetLines()
	if lines[0] != "xyz xyz xyz" {
		t.Errorf("Expected 'xyz xyz xyz', got '%s'", lines[0])
	}
}
