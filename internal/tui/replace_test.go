package tui

import (
	"testing"

	"github.com/bmd/bmd/internal/editor"
)

// TestReplaceStateInitialization tests that ReplaceState starts with sane defaults.
func TestReplaceStateInitialization(t *testing.T) {
	rs := ReplaceState{CurrentMatch: -1}
	if rs.Query != "" {
		t.Errorf("Expected empty Query, got '%s'", rs.Query)
	}
	if rs.Replacement != "" {
		t.Errorf("Expected empty Replacement, got '%s'", rs.Replacement)
	}
	if rs.CaseSensitive {
		t.Error("Expected CaseSensitive to be false")
	}
	if rs.WholeWord {
		t.Error("Expected WholeWord to be false")
	}
	if rs.CurrentMatch != -1 {
		t.Errorf("Expected CurrentMatch -1, got %d", rs.CurrentMatch)
	}
}

// TestUpdateReplaceMatches tests that match finding works through the viewer.
func TestUpdateReplaceMatches(t *testing.T) {
	v := &Viewer{
		editBuffer: editor.NewTextBuffer([]string{"foo bar foo", "baz foo qux"}),
		replaceMode: true,
		replaceState: ReplaceState{
			Query:        "foo",
			CurrentMatch: -1,
		},
	}

	v.updateReplaceMatches()

	if len(v.replaceState.Matches) != 3 {
		t.Fatalf("Expected 3 matches, got %d", len(v.replaceState.Matches))
	}
	if v.replaceState.CurrentMatch != 0 {
		t.Errorf("Expected CurrentMatch 0, got %d", v.replaceState.CurrentMatch)
	}
}

// TestUpdateReplaceMatchesCaseSensitive tests case-sensitive matching via viewer.
func TestUpdateReplaceMatchesCaseSensitive(t *testing.T) {
	v := &Viewer{
		editBuffer: editor.NewTextBuffer([]string{"Foo bar foo FOO"}),
		replaceMode: true,
		replaceState: ReplaceState{
			Query:         "Foo",
			CaseSensitive: true,
			CurrentMatch:  -1,
		},
	}

	v.updateReplaceMatches()

	if len(v.replaceState.Matches) != 1 {
		t.Fatalf("Expected 1 case-sensitive match, got %d", len(v.replaceState.Matches))
	}
}

// TestUpdateReplaceMatchesWholeWord tests whole-word matching via viewer.
func TestUpdateReplaceMatchesWholeWord(t *testing.T) {
	v := &Viewer{
		editBuffer: editor.NewTextBuffer([]string{"word wordsmith sword"}),
		replaceMode: true,
		replaceState: ReplaceState{
			Query:        "word",
			WholeWord:    true,
			CurrentMatch: -1,
		},
	}

	v.updateReplaceMatches()

	if len(v.replaceState.Matches) != 1 {
		t.Fatalf("Expected 1 whole-word match, got %d", len(v.replaceState.Matches))
	}
}

// TestUpdateReplaceMatchesEmptyQuery tests that empty query clears matches.
func TestUpdateReplaceMatchesEmptyQuery(t *testing.T) {
	v := &Viewer{
		editBuffer: editor.NewTextBuffer([]string{"foo bar"}),
		replaceMode: true,
		replaceState: ReplaceState{
			Query:        "",
			CurrentMatch: -1,
		},
	}

	v.updateReplaceMatches()

	if len(v.replaceState.Matches) != 0 {
		t.Errorf("Expected 0 matches for empty query, got %d", len(v.replaceState.Matches))
	}
	if v.replaceState.CurrentMatch != -1 {
		t.Errorf("Expected CurrentMatch -1, got %d", v.replaceState.CurrentMatch)
	}
}

// TestRenderReplacePrompt tests that the prompt renders without panic.
func TestRenderReplacePrompt(t *testing.T) {
	v := &Viewer{
		Width: 80,
		replaceMode: true,
		replaceState: ReplaceState{
			Query:       "test",
			Replacement: "new",
			Matches:     [][2]int{{0, 0}, {1, 5}},
			CurrentMatch: 0,
		},
	}

	prompt := v.renderReplacePrompt()
	if prompt == "" {
		t.Error("Expected non-empty prompt")
	}

	// Should contain key elements
	if !containsStr(prompt, "Find:") {
		t.Error("Expected prompt to contain 'Find:'")
	}
	if !containsStr(prompt, "Replace:") {
		t.Error("Expected prompt to contain 'Replace:'")
	}
	if !containsStr(prompt, "1 of 2") {
		t.Error("Expected prompt to contain '1 of 2'")
	}
}

func containsStr(s, substr string) bool {
	return len(s) >= len(substr) && searchStr(s, substr)
}

func searchStr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
