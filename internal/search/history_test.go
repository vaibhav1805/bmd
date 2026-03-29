package search

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSearchHistory_PushAndRecall(t *testing.T) {
	h := NewSearchHistory("")

	h.Push("foo")
	h.Push("bar")
	h.Push("baz")

	if h.Len() != 3 {
		t.Fatalf("expected 3 queries, got %d", h.Len())
	}

	// Prev walks backward from most recent
	if got := h.Prev(); got != "baz" {
		t.Errorf("first Prev() = %q, want %q", got, "baz")
	}
	if got := h.Prev(); got != "bar" {
		t.Errorf("second Prev() = %q, want %q", got, "bar")
	}
	if got := h.Prev(); got != "foo" {
		t.Errorf("third Prev() = %q, want %q", got, "foo")
	}
	// At oldest, stays there
	if got := h.Prev(); got != "foo" {
		t.Errorf("fourth Prev() = %q, want %q", got, "foo")
	}

	// Next walks forward
	if got := h.Next(); got != "bar" {
		t.Errorf("Next() = %q, want %q", got, "bar")
	}
	if got := h.Next(); got != "baz" {
		t.Errorf("Next() = %q, want %q", got, "baz")
	}
	// Past newest returns ""
	if got := h.Next(); got != "" {
		t.Errorf("Next() past end = %q, want %q", got, "")
	}
}

func TestSearchHistory_Reset(t *testing.T) {
	h := NewSearchHistory("")
	h.Push("alpha")
	h.Push("beta")

	h.Prev() // beta
	h.Prev() // alpha

	h.Reset()

	// After reset, Prev starts from most recent again
	if got := h.Prev(); got != "beta" {
		t.Errorf("Prev() after Reset = %q, want %q", got, "beta")
	}
}

func TestSearchHistory_SkipEmpty(t *testing.T) {
	h := NewSearchHistory("")
	h.Push("")
	if h.Len() != 0 {
		t.Errorf("empty push should be ignored, got len %d", h.Len())
	}
}

func TestSearchHistory_SkipConsecutiveDuplicates(t *testing.T) {
	h := NewSearchHistory("")
	h.Push("foo")
	h.Push("foo")
	h.Push("foo")
	if h.Len() != 1 {
		t.Errorf("consecutive duplicates should be ignored, got len %d", h.Len())
	}
}

func TestSearchHistory_NonConsecutiveDuplicatesAllowed(t *testing.T) {
	h := NewSearchHistory("")
	h.Push("foo")
	h.Push("bar")
	h.Push("foo")
	if h.Len() != 3 {
		t.Errorf("non-consecutive duplicates should be kept, got len %d", h.Len())
	}
}

func TestSearchHistory_MaxSize(t *testing.T) {
	h := NewSearchHistory("")
	for i := 0; i < 30; i++ {
		h.Push(string(rune('A' + i)))
	}
	if h.Len() != maxHistorySize {
		t.Errorf("expected max %d, got %d", maxHistorySize, h.Len())
	}
	// Oldest should have been trimmed; most recent should be last pushed
	h.Reset()
	got := h.Prev()
	want := string(rune('A' + 29))
	if got != want {
		t.Errorf("most recent = %q, want %q", got, want)
	}
}

func TestSearchHistory_EmptyPrevNext(t *testing.T) {
	h := NewSearchHistory("")
	if got := h.Prev(); got != "" {
		t.Errorf("Prev() on empty = %q, want %q", got, "")
	}
	if got := h.Next(); got != "" {
		t.Errorf("Next() on empty = %q, want %q", got, "")
	}
}

func TestSearchHistory_Persistence(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "history.json")

	// Save
	h1 := NewSearchHistory(path)
	h1.Push("one")
	h1.Push("two")
	if err := h1.Save(); err != nil {
		t.Fatalf("Save: %v", err)
	}

	// Load in new instance
	h2 := NewSearchHistory(path)
	if err := h2.Load(); err != nil {
		t.Fatalf("Load: %v", err)
	}
	if h2.Len() != 2 {
		t.Fatalf("loaded len = %d, want 2", h2.Len())
	}
	if got := h2.Prev(); got != "two" {
		t.Errorf("loaded Prev() = %q, want %q", got, "two")
	}
}

func TestSearchHistory_LoadMissingFile(t *testing.T) {
	h := NewSearchHistory("/nonexistent/path/history.json")
	if err := h.Load(); err != nil {
		t.Errorf("Load of missing file should not error, got: %v", err)
	}
	if h.Len() != 0 {
		t.Errorf("should be empty after loading missing file")
	}
}

func TestSearchHistory_LoadCorruptedFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "history.json")
	os.WriteFile(path, []byte("{invalid json"), 0o644)

	h := NewSearchHistory(path)
	if err := h.Load(); err != nil {
		t.Errorf("Load of corrupted file should not error, got: %v", err)
	}
	if h.Len() != 0 {
		t.Errorf("should be empty after loading corrupted file")
	}
}

func TestSearchHistory_Clear(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "history.json")

	h := NewSearchHistory(path)
	h.Push("x")
	h.Save()

	if err := h.Clear(); err != nil {
		t.Fatalf("Clear: %v", err)
	}
	if h.Len() != 0 {
		t.Errorf("expected empty after clear, got %d", h.Len())
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Errorf("file should be removed after clear")
	}
}

func TestSearchHistory_ClearMissingFile(t *testing.T) {
	h := NewSearchHistory("/nonexistent/path/history.json")
	if err := h.Clear(); err != nil {
		t.Errorf("Clear of missing file should not error, got: %v", err)
	}
}
