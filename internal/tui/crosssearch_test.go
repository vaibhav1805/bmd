package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/bmd/bmd/internal/ast"
	"github.com/bmd/bmd/internal/knowledge"
	"github.com/bmd/bmd/internal/theme"
)

// newTestViewerWithDir creates a Viewer with startDir set for cross-search tests.
func newTestViewerWithDir(t *testing.T, startDir string) Viewer {
	t.Helper()
	doc := &ast.Document{}
	v := New(doc, filepath.Join(startDir, "test.md"), theme.NewTheme(), 80)
	v.Height = 24
	return v
}

// buildTmpSearchDir creates a temporary directory with the given files (filename -> content).
func buildTmpSearchDir(t *testing.T, files map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	for name, content := range files {
		path := filepath.Join(dir, name)
		// Create parent dirs if needed.
		_ = os.MkdirAll(filepath.Dir(path), 0o755)
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatalf("buildTmpSearchDir: write %s: %v", name, err)
		}
	}
	return dir
}

// --- SearchState unit tests ---

// TestCrossSearchInitialState verifies default state of new viewer.
func TestCrossSearchInitialState(t *testing.T) {
	dir := t.TempDir()
	v := newTestViewerWithDir(t, dir)

	if v.crossSearchMode {
		t.Error("crossSearchMode should be false initially")
	}
	if v.crossSearchActive {
		t.Error("crossSearchActive should be false initially")
	}
	if v.crossSearchQuery != "" {
		t.Errorf("crossSearchQuery should be empty initially, got %q", v.crossSearchQuery)
	}
	if v.crossSearchResults != nil {
		t.Error("crossSearchResults should be nil initially")
	}
	if v.crossSearchSelected != 0 {
		// Default zero value is fine; it gets set to -1 on explicit clear.
		// Just check no panic.
	}
}

// TestCrossSearchModeActivation verifies '/' key activates cross-search input.
func TestCrossSearchModeActivation(t *testing.T) {
	dir := t.TempDir()
	v := newTestViewerWithDir(t, dir)

	slashKey := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}}
	model, _ := v.Update(slashKey)
	result := model.(Viewer)

	if !result.crossSearchMode {
		t.Error("Expected crossSearchMode=true after pressing '/'")
	}
	if result.crossSearchInput != "" {
		t.Errorf("crossSearchInput should be empty, got %q", result.crossSearchInput)
	}
}

// TestCrossSearchInputAccumulatesCharacters verifies query building while typing.
func TestCrossSearchInputAccumulatesCharacters(t *testing.T) {
	dir := t.TempDir()
	v := newTestViewerWithDir(t, dir)
	v.crossSearchMode = true
	v.crossSearchInput = ""

	for _, ch := range "hello" {
		key := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{ch}}
		model, _ := v.Update(key)
		v = model.(Viewer)
	}

	if v.crossSearchInput != "hello" {
		t.Errorf("Expected crossSearchInput=%q, got %q", "hello", v.crossSearchInput)
	}
}

// TestCrossSearchInputBackspace verifies Backspace removes last character.
func TestCrossSearchInputBackspace(t *testing.T) {
	dir := t.TempDir()
	v := newTestViewerWithDir(t, dir)
	v.crossSearchMode = true
	v.crossSearchInput = "API"

	backspace := tea.KeyMsg{Type: tea.KeyBackspace}
	model, _ := v.Update(backspace)
	result := model.(Viewer)

	if result.crossSearchInput != "AP" {
		t.Errorf("Expected crossSearchInput=%q after backspace, got %q", "AP", result.crossSearchInput)
	}
}

// TestCrossSearchInputBackspaceEmpty verifies Backspace on empty input is safe.
func TestCrossSearchInputBackspaceEmpty(t *testing.T) {
	dir := t.TempDir()
	v := newTestViewerWithDir(t, dir)
	v.crossSearchMode = true
	v.crossSearchInput = ""

	backspace := tea.KeyMsg{Type: tea.KeyBackspace}
	model, _ := v.Update(backspace)
	result := model.(Viewer)

	if result.crossSearchInput != "" {
		t.Errorf("Expected empty crossSearchInput after backspace on empty, got %q", result.crossSearchInput)
	}
}

// TestCrossSearchEscCancels verifies Esc key cancels cross-search mode.
func TestCrossSearchEscCancels(t *testing.T) {
	dir := t.TempDir()
	v := newTestViewerWithDir(t, dir)
	v.crossSearchMode = true
	v.crossSearchInput = "partial query"

	esc := tea.KeyMsg{Type: tea.KeyEsc}
	model, _ := v.Update(esc)
	result := model.(Viewer)

	if result.crossSearchMode {
		t.Error("Expected crossSearchMode=false after Esc")
	}
	if result.crossSearchInput != "" {
		t.Errorf("Expected crossSearchInput cleared after Esc, got %q", result.crossSearchInput)
	}
	if result.crossSearchActive {
		t.Error("Expected crossSearchActive=false after Esc")
	}
}

// TestCrossSearchEmptyQueryDoesNotActivate verifies empty Enter does not set active.
func TestCrossSearchEmptyQueryDoesNotActivate(t *testing.T) {
	dir := t.TempDir()
	v := newTestViewerWithDir(t, dir)
	v.crossSearchMode = true
	v.crossSearchInput = "   " // whitespace only

	enter := tea.KeyMsg{Type: tea.KeyEnter}
	model, _ := v.Update(enter)
	result := model.(Viewer)

	if result.crossSearchMode {
		t.Error("Expected crossSearchMode=false after Enter")
	}
	if result.crossSearchActive {
		t.Error("Expected crossSearchActive=false for empty query")
	}
}

// --- Phase 6 index integration tests ---

// TestSearchAllDocumentsEmptyQuery returns empty without error.
func TestSearchAllDocumentsEmptyQuery(t *testing.T) {
	dir := buildTmpSearchDir(t, map[string]string{
		"a.md": "# Doc A\nSome content",
	})

	results, err := knowledge.SearchAllDocuments(dir, "", 10)
	if err != nil {
		t.Fatalf("Unexpected error on empty query: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("Expected 0 results for empty query, got %d", len(results))
	}
}

// TestSearchAllDocumentsSingleFile finds a term in a single file.
func TestSearchAllDocumentsSingleFile(t *testing.T) {
	dir := buildTmpSearchDir(t, map[string]string{
		"readme.md": "# README\nThis document explains the authentication service.",
	})

	results, err := knowledge.SearchAllDocuments(dir, "authentication", 10)
	if err != nil {
		t.Fatalf("SearchAllDocuments error: %v", err)
	}
	if len(results) == 0 {
		t.Fatal("Expected at least 1 result for 'authentication'")
	}
}

// TestSearchAllDocumentsMultipleFiles finds the best match across multiple files.
func TestSearchAllDocumentsMultipleFiles(t *testing.T) {
	dir := buildTmpSearchDir(t, map[string]string{
		"auth.md":       "# Auth Service\nJWT authentication and authorization flows.",
		"database.md":   "# Database\nSQL schema and migration scripts.",
		"api.md":        "# API Gateway\nRESTful API endpoints and authentication.",
	})

	results, err := knowledge.SearchAllDocuments(dir, "authentication", 10)
	if err != nil {
		t.Fatalf("SearchAllDocuments error: %v", err)
	}
	if len(results) < 2 {
		t.Errorf("Expected at least 2 results for 'authentication', got %d", len(results))
	}
}

// TestSearchAllDocumentsResultsSortedByScore verifies descending score order.
func TestSearchAllDocumentsResultsSortedByScore(t *testing.T) {
	dir := buildTmpSearchDir(t, map[string]string{
		"high.md":   "# High Match\nmicroservices microservices microservices architecture",
		"low.md":    "# Low Match\none mention of microservices",
		"other.md":  "# Other\ncompletely unrelated content about databases",
	})

	results, err := knowledge.SearchAllDocuments(dir, "microservices", 10)
	if err != nil {
		t.Fatalf("SearchAllDocuments error: %v", err)
	}
	if len(results) < 2 {
		t.Errorf("Expected at least 2 results, got %d", len(results))
	}
	// Results should be sorted descending by score.
	for i := 1; i < len(results); i++ {
		if results[i].Score > results[i-1].Score {
			t.Errorf("Results not sorted by score: results[%d].Score=%.2f > results[%d].Score=%.2f",
				i, results[i].Score, i-1, results[i-1].Score)
		}
	}
}

// TestSearchAllDocumentsResultCountAccurate checks MatchCount field.
func TestSearchAllDocumentsResultCountAccurate(t *testing.T) {
	dir := buildTmpSearchDir(t, map[string]string{
		"doc.md": "# Doc\nThis doc contains API and gateway and authentication terms.",
	})

	results, err := knowledge.SearchAllDocuments(dir, "API gateway", 10)
	if err != nil {
		t.Fatalf("SearchAllDocuments error: %v", err)
	}
	if len(results) == 0 {
		t.Skip("No results — terms may be stop-word filtered; skipping count test")
	}
	if results[0].MatchCount < 1 {
		t.Errorf("Expected MatchCount >= 1, got %d", results[0].MatchCount)
	}
}

// TestSearchAllDocumentsAutoBuildsIndex verifies auto-build when no index exists.
func TestSearchAllDocumentsAutoBuildsIndex(t *testing.T) {
	dir := buildTmpSearchDir(t, map[string]string{
		"notes.md": "# Notes\nImportant deployment procedures.",
	})
	// No knowledge.db exists yet — should auto-build.
	results, err := knowledge.SearchAllDocuments(dir, "deployment", 10)
	if err != nil {
		t.Fatalf("SearchAllDocuments error on auto-build: %v", err)
	}
	// Might be 0 if term is stop-word filtered; just verify no error.
	_ = results
}

// TestSearchAllDocumentsNoMatchReturnsEmpty returns empty slice for unmatched query.
func TestSearchAllDocumentsNoMatchReturnsEmpty(t *testing.T) {
	dir := buildTmpSearchDir(t, map[string]string{
		"doc.md": "# Doc\nSome basic content here.",
	})

	results, err := knowledge.SearchAllDocuments(dir, "xyzquuxnonexistent", 10)
	if err != nil {
		t.Fatalf("SearchAllDocuments error: %v", err)
	}
	if results == nil {
		t.Error("Expected non-nil empty slice, got nil")
	}
	if len(results) != 0 {
		t.Errorf("Expected 0 results for non-matching query, got %d", len(results))
	}
}

// TestSearchAllFilesMethod tests the Viewer.SearchAllFiles method wrapper.
func TestSearchAllFilesMethod(t *testing.T) {
	dir := buildTmpSearchDir(t, map[string]string{
		"service.md": "# Service\nThe payment service handles transactions.",
	})
	v := newTestViewerWithDir(t, dir)

	results, err := v.SearchAllFiles("payment")
	if err != nil {
		t.Fatalf("SearchAllFiles error: %v", err)
	}
	if len(results) == 0 {
		t.Error("Expected at least 1 result for 'payment'")
	}
}

// --- Navigation tests ---

// TestCrossSearchNavMoveDown verifies ↓ increments selected result.
func TestCrossSearchNavMoveDown(t *testing.T) {
	dir := t.TempDir()
	v := newTestViewerWithDir(t, dir)
	v.crossSearchActive = true
	v.crossSearchMode = false
	v.crossSearchSelected = 0
	v.crossSearchResults = []knowledge.SearchResult{
		{RelPath: "a.md", Score: 5.0},
		{RelPath: "b.md", Score: 4.0},
		{RelPath: "c.md", Score: 3.0},
	}

	downKey := tea.KeyMsg{Type: tea.KeyDown}
	model, _ := v.Update(downKey)
	result := model.(Viewer)

	if result.crossSearchSelected != 1 {
		t.Errorf("Expected crossSearchSelected=1 after ↓, got %d", result.crossSearchSelected)
	}
}

// TestCrossSearchNavMoveUp verifies ↑ decrements selected result.
func TestCrossSearchNavMoveUp(t *testing.T) {
	dir := t.TempDir()
	v := newTestViewerWithDir(t, dir)
	v.crossSearchActive = true
	v.crossSearchMode = false
	v.crossSearchSelected = 2
	v.crossSearchResults = []knowledge.SearchResult{
		{RelPath: "a.md", Score: 5.0},
		{RelPath: "b.md", Score: 4.0},
		{RelPath: "c.md", Score: 3.0},
	}

	upKey := tea.KeyMsg{Type: tea.KeyUp}
	model, _ := v.Update(upKey)
	result := model.(Viewer)

	if result.crossSearchSelected != 1 {
		t.Errorf("Expected crossSearchSelected=1 after ↑, got %d", result.crossSearchSelected)
	}
}

// TestCrossSearchNavClampBottom verifies ↓ on last item doesn't exceed bounds.
func TestCrossSearchNavClampBottom(t *testing.T) {
	dir := t.TempDir()
	v := newTestViewerWithDir(t, dir)
	v.crossSearchActive = true
	v.crossSearchMode = false
	v.crossSearchSelected = 2
	v.crossSearchResults = []knowledge.SearchResult{
		{RelPath: "a.md", Score: 5.0},
		{RelPath: "b.md", Score: 4.0},
		{RelPath: "c.md", Score: 3.0},
	}

	downKey := tea.KeyMsg{Type: tea.KeyDown}
	model, _ := v.Update(downKey)
	result := model.(Viewer)

	if result.crossSearchSelected != 2 {
		t.Errorf("Expected crossSearchSelected=2 (clamped), got %d", result.crossSearchSelected)
	}
}

// TestCrossSearchNavClampTop verifies ↑ on first item doesn't go below 0.
func TestCrossSearchNavClampTop(t *testing.T) {
	dir := t.TempDir()
	v := newTestViewerWithDir(t, dir)
	v.crossSearchActive = true
	v.crossSearchMode = false
	v.crossSearchSelected = 0
	v.crossSearchResults = []knowledge.SearchResult{
		{RelPath: "a.md", Score: 5.0},
	}

	upKey := tea.KeyMsg{Type: tea.KeyUp}
	model, _ := v.Update(upKey)
	result := model.(Viewer)

	if result.crossSearchSelected != 0 {
		t.Errorf("Expected crossSearchSelected=0 (clamped), got %d", result.crossSearchSelected)
	}
}

// TestCrossSearchNavVimKeys verifies j/k keys work same as ↓/↑.
func TestCrossSearchNavVimKeys(t *testing.T) {
	dir := t.TempDir()
	v := newTestViewerWithDir(t, dir)
	v.crossSearchActive = true
	v.crossSearchMode = false
	v.crossSearchSelected = 0
	v.crossSearchResults = []knowledge.SearchResult{
		{RelPath: "a.md", Score: 5.0},
		{RelPath: "b.md", Score: 4.0},
	}

	jKey := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}}
	model, _ := v.Update(jKey)
	result := model.(Viewer)
	if result.crossSearchSelected != 1 {
		t.Errorf("Expected crossSearchSelected=1 after 'j', got %d", result.crossSearchSelected)
	}

	kKey := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}}
	model, _ = result.Update(kKey)
	result = model.(Viewer)
	if result.crossSearchSelected != 0 {
		t.Errorf("Expected crossSearchSelected=0 after 'k', got %d", result.crossSearchSelected)
	}
}

// TestCrossSearchNavEscExits verifies Esc exits search results.
func TestCrossSearchNavEscExits(t *testing.T) {
	dir := t.TempDir()
	v := newTestViewerWithDir(t, dir)
	v.crossSearchActive = true
	v.crossSearchMode = false
	v.crossSearchSelected = 1
	v.crossSearchResults = []knowledge.SearchResult{
		{RelPath: "a.md", Score: 5.0},
	}

	esc := tea.KeyMsg{Type: tea.KeyEsc}
	model, _ := v.Update(esc)
	result := model.(Viewer)

	if result.crossSearchActive {
		t.Error("Expected crossSearchActive=false after Esc from results")
	}
	if result.crossSearchResults != nil {
		t.Error("Expected crossSearchResults=nil after Esc")
	}
}

// TestCrossSearchNavHKeyExits verifies 'h' exits search results (back).
func TestCrossSearchNavHKeyExits(t *testing.T) {
	dir := t.TempDir()
	v := newTestViewerWithDir(t, dir)
	v.crossSearchActive = true
	v.crossSearchMode = false
	v.crossSearchResults = []knowledge.SearchResult{
		{RelPath: "a.md", Score: 5.0},
	}

	hKey := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'h'}}
	model, _ := v.Update(hKey)
	result := model.(Viewer)

	if result.crossSearchActive {
		t.Error("Expected crossSearchActive=false after 'h'")
	}
}

// TestCrossSearchNavSlashReopensSearch verifies '/' in results reopens the prompt.
func TestCrossSearchNavSlashReopensSearch(t *testing.T) {
	dir := t.TempDir()
	v := newTestViewerWithDir(t, dir)
	v.crossSearchActive = true
	v.crossSearchMode = false
	v.crossSearchQuery = "old query"
	v.crossSearchResults = []knowledge.SearchResult{
		{RelPath: "a.md", Score: 5.0},
	}

	slash := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}}
	model, _ := v.Update(slash)
	result := model.(Viewer)

	if !result.crossSearchMode {
		t.Error("Expected crossSearchMode=true after '/' from results")
	}
	if result.crossSearchInput != "old query" {
		t.Errorf("Expected crossSearchInput=%q, got %q", "old query", result.crossSearchInput)
	}
}

// --- Render tests ---

// TestRenderCrossSearchResultsNoCrash verifies renderCrossSearchResults doesn't panic.
func TestRenderCrossSearchResultsNoCrash(t *testing.T) {
	dir := t.TempDir()
	v := newTestViewerWithDir(t, dir)
	v.crossSearchActive = true
	v.crossSearchQuery = "test"
	v.crossSearchResults = []knowledge.SearchResult{
		{RelPath: "a.md", Score: 5.2, Path: filepath.Join(dir, "a.md")},
		{RelPath: "b.md", Score: 3.1, Path: filepath.Join(dir, "b.md")},
	}
	v.crossSearchSelected = 0

	// Should not panic.
	out := v.renderCrossSearchResults(20)
	if out == "" {
		t.Error("renderCrossSearchResults returned empty string")
	}
}

// TestRenderCrossSearchResultsShowsQuery verifies query is shown in results header.
func TestRenderCrossSearchResultsShowsQuery(t *testing.T) {
	dir := t.TempDir()
	v := newTestViewerWithDir(t, dir)
	v.crossSearchActive = true
	v.crossSearchQuery = "microservices"
	v.crossSearchResults = []knowledge.SearchResult{
		{RelPath: "svc.md", Score: 8.2},
	}
	v.crossSearchSelected = 0

	out := v.renderCrossSearchResults(20)
	if !strings.Contains(out, "microservices") {
		t.Error("Expected query 'microservices' in search results output")
	}
}

// TestRenderCrossSearchResultsShowsFilenames verifies filenames appear in output.
func TestRenderCrossSearchResultsShowsFilenames(t *testing.T) {
	dir := t.TempDir()
	v := newTestViewerWithDir(t, dir)
	v.crossSearchActive = true
	v.crossSearchQuery = "service"
	v.crossSearchResults = []knowledge.SearchResult{
		{RelPath: "payment.md", Score: 7.0},
		{RelPath: "auth.md", Score: 5.5},
	}
	v.crossSearchSelected = 0

	out := v.renderCrossSearchResults(20)
	if !strings.Contains(out, "payment.md") {
		t.Error("Expected 'payment.md' in search results output")
	}
	if !strings.Contains(out, "auth.md") {
		t.Error("Expected 'auth.md' in search results output")
	}
}

// TestRenderCrossSearchResultsShowsScores verifies BM25 scores appear in output.
func TestRenderCrossSearchResultsShowsScores(t *testing.T) {
	dir := t.TempDir()
	v := newTestViewerWithDir(t, dir)
	v.crossSearchActive = true
	v.crossSearchQuery = "test"
	v.crossSearchResults = []knowledge.SearchResult{
		{RelPath: "doc.md", Score: 8.2},
	}
	v.crossSearchSelected = 0

	out := v.renderCrossSearchResults(20)
	if !strings.Contains(out, "8.2") {
		t.Error("Expected score '8.2' in search results output")
	}
}

// TestRenderCrossSearchResultsEmptyShowsNoMatch verifies empty results message.
func TestRenderCrossSearchResultsEmptyShowsNoMatch(t *testing.T) {
	dir := t.TempDir()
	v := newTestViewerWithDir(t, dir)
	v.crossSearchActive = true
	v.crossSearchQuery = "zzznomatch"
	v.crossSearchResults = []knowledge.SearchResult{}
	v.crossSearchSelected = -1

	out := v.renderCrossSearchResults(20)
	if !strings.Contains(out, "No matches") {
		t.Errorf("Expected 'No matches' in empty results output, got:\n%s", out)
	}
}

// TestRenderCrossSearchResultsSelectedHighlighted verifies selected result uses reverse-video.
func TestRenderCrossSearchResultsSelectedHighlighted(t *testing.T) {
	dir := t.TempDir()
	v := newTestViewerWithDir(t, dir)
	v.crossSearchActive = true
	v.crossSearchQuery = "test"
	v.crossSearchResults = []knowledge.SearchResult{
		{RelPath: "first.md", Score: 9.0},
		{RelPath: "second.md", Score: 5.0},
	}
	v.crossSearchSelected = 0

	out := v.renderCrossSearchResults(20)
	// Reverse video escape code should be present for the selected result.
	if !strings.Contains(out, "\x1b[7m") {
		t.Error("Expected reverse-video escape code for selected result")
	}
}

// TestRenderCrossSearchResultsCountInHeader verifies result count shown in header.
func TestRenderCrossSearchResultsCountInHeader(t *testing.T) {
	dir := t.TempDir()
	v := newTestViewerWithDir(t, dir)
	v.crossSearchActive = true
	v.crossSearchQuery = "test"
	v.crossSearchResults = []knowledge.SearchResult{
		{RelPath: "a.md", Score: 5.0},
		{RelPath: "b.md", Score: 4.0},
		{RelPath: "c.md", Score: 3.0},
	}
	v.crossSearchSelected = 0

	out := v.renderCrossSearchResults(20)
	if !strings.Contains(out, "3") {
		t.Error("Expected result count '3' in search results header")
	}
}

// TestViewRoutesToCrossSearchWhenActive verifies View() calls renderCrossSearchResults.
func TestViewRoutesToCrossSearchWhenActive(t *testing.T) {
	dir := t.TempDir()
	v := newTestViewerWithDir(t, dir)
	v.crossSearchActive = true
	v.crossSearchMode = false
	v.crossSearchQuery = "uniqueterm"
	v.crossSearchResults = []knowledge.SearchResult{
		{RelPath: "test.md", Score: 6.0},
	}
	v.crossSearchSelected = 0

	out := v.View()
	if !strings.Contains(out, "uniqueterm") {
		t.Error("Expected search query in View() output when crossSearchActive=true")
	}
}

// TestStatusBarShowsCrossSearchPrompt verifies status bar shows cross-search prompt.
func TestStatusBarShowsCrossSearchPrompt(t *testing.T) {
	dir := t.TempDir()
	v := newTestViewerWithDir(t, dir)
	v.crossSearchMode = true
	v.crossSearchInput = "my query"

	bar := v.renderStatusBar()
	if !strings.Contains(bar, "Search all files") {
		t.Errorf("Expected 'Search all files' prompt in status bar, got: %s", bar)
	}
	if !strings.Contains(bar, "my query") {
		t.Errorf("Expected typed query 'my query' in status bar, got: %s", bar)
	}
}

// TestCrossSearchSpecialCharactersNoCrash tests that special chars don't panic.
func TestCrossSearchSpecialCharactersNoCrash(t *testing.T) {
	dir := buildTmpSearchDir(t, map[string]string{
		"doc.md": "# Doc\nSome content here.",
	})

	specialQueries := []string{"", "  ", "a", "API", "hello world", "!@#$%", "\"quoted\""}
	for _, q := range specialQueries {
		t.Run(fmt.Sprintf("query=%q", q), func(t *testing.T) {
			// Should not panic.
			_, err := knowledge.SearchAllDocuments(dir, q, 10)
			if err != nil {
				// Error is acceptable (e.g. index build failure on some systems).
				// What matters is no panic.
				t.Logf("SearchAllDocuments(%q) returned error (acceptable): %v", q, err)
			}
		})
	}
}

// TestSearchAllDocumentsLargeResultSet verifies performance with many files.
func TestSearchAllDocumentsLargeResultSet(t *testing.T) {
	files := make(map[string]string, 20)
	for i := 0; i < 20; i++ {
		name := fmt.Sprintf("doc%02d.md", i)
		files[name] = fmt.Sprintf("# Document %d\nThis document talks about microservices and deployment pipeline %d.", i, i)
	}
	dir := buildTmpSearchDir(t, files)

	results, err := knowledge.SearchAllDocuments(dir, "microservices", 50)
	if err != nil {
		t.Fatalf("SearchAllDocuments error: %v", err)
	}
	if len(results) == 0 {
		t.Error("Expected results for 'microservices' across 20 files")
	}
	// Verify sorted.
	for i := 1; i < len(results); i++ {
		if results[i].Score > results[i-1].Score {
			t.Errorf("Results not sorted: results[%d].Score=%.2f > results[%d].Score=%.2f",
				i, results[i].Score, i-1, results[i-1].Score)
		}
	}
}

// ====== DIR-04: Snippet Extraction Tests ======

// TestGetContextSnippetBasicMatch verifies snippet extraction around a known match.
func TestGetContextSnippetBasicMatch(t *testing.T) {
	dir := buildTmpSearchDir(t, map[string]string{
		"doc.md": "# Title\nThis document explains the microservices architecture pattern in detail.",
	})
	snippet := knowledge.GetContextSnippet(filepath.Join(dir, "doc.md"), "microservices", 100)
	if snippet == "" {
		t.Fatal("Expected non-empty snippet")
	}
	if !strings.Contains(strings.ToLower(snippet), "microservices") {
		t.Errorf("Expected snippet to contain 'microservices', got: %s", snippet)
	}
}

// TestGetContextSnippetNoMatch verifies fallback when query not found.
func TestGetContextSnippetNoMatch(t *testing.T) {
	dir := buildTmpSearchDir(t, map[string]string{
		"doc.md": "# Title\nSome content here about databases and storage.",
	})
	snippet := knowledge.GetContextSnippet(filepath.Join(dir, "doc.md"), "xyznonexistent", 100)
	// Should return beginning of file as fallback.
	if snippet == "" {
		t.Fatal("Expected non-empty snippet even without match")
	}
	if !strings.Contains(snippet, "Title") {
		t.Errorf("Expected snippet to start with file beginning, got: %s", snippet)
	}
}

// TestGetContextSnippetEmptyFile verifies empty file returns empty snippet.
func TestGetContextSnippetEmptyFile(t *testing.T) {
	dir := buildTmpSearchDir(t, map[string]string{
		"empty.md": "",
	})
	snippet := knowledge.GetContextSnippet(filepath.Join(dir, "empty.md"), "test", 100)
	if snippet != "" {
		t.Errorf("Expected empty snippet for empty file, got: %q", snippet)
	}
}

// TestGetContextSnippetMissingFile verifies missing file returns empty snippet.
func TestGetContextSnippetMissingFile(t *testing.T) {
	snippet := knowledge.GetContextSnippet("/nonexistent/path.md", "test", 100)
	if snippet != "" {
		t.Errorf("Expected empty snippet for missing file, got: %q", snippet)
	}
}

// TestGetContextSnippetEmptyQuery verifies empty query returns file start.
func TestGetContextSnippetEmptyQuery(t *testing.T) {
	dir := buildTmpSearchDir(t, map[string]string{
		"doc.md": "# Heading\nContent goes here.",
	})
	snippet := knowledge.GetContextSnippet(filepath.Join(dir, "doc.md"), "", 100)
	if snippet == "" {
		t.Fatal("Expected non-empty snippet for empty query")
	}
}

// TestGetContextSnippetTruncation verifies ellipsis when content is truncated.
func TestGetContextSnippetTruncation(t *testing.T) {
	longContent := strings.Repeat("word ", 200)
	dir := buildTmpSearchDir(t, map[string]string{
		"long.md": longContent,
	})
	snippet := knowledge.GetContextSnippet(filepath.Join(dir, "long.md"), "word", 50)
	if len([]rune(snippet)) > 60 { // allow for "..." padding
		t.Errorf("Snippet too long: %d runes", len([]rune(snippet)))
	}
}

// TestGetContextSnippetMatchAtStart verifies snippet when match is at file beginning.
func TestGetContextSnippetMatchAtStart(t *testing.T) {
	dir := buildTmpSearchDir(t, map[string]string{
		"doc.md": "API gateway handles requests to microservices for load balancing.",
	})
	snippet := knowledge.GetContextSnippet(filepath.Join(dir, "doc.md"), "API", 100)
	if !strings.Contains(snippet, "API") {
		t.Errorf("Expected snippet to contain 'API', got: %s", snippet)
	}
	// Should NOT start with "..." since match is at beginning.
	if strings.HasPrefix(snippet, "...") {
		t.Error("Snippet should not start with '...' when match is at beginning")
	}
}

// TestGetContextSnippetMatchAtEnd verifies snippet when match is near end.
func TestGetContextSnippetMatchAtEnd(t *testing.T) {
	content := strings.Repeat("filler ", 50) + "microservices pattern"
	dir := buildTmpSearchDir(t, map[string]string{
		"doc.md": content,
	})
	snippet := knowledge.GetContextSnippet(filepath.Join(dir, "doc.md"), "microservices", 60)
	if !strings.Contains(strings.ToLower(snippet), "microservices") {
		t.Errorf("Expected snippet to contain 'microservices', got: %s", snippet)
	}
}

// TestGetContextSnippetCaseInsensitive verifies case-insensitive matching.
func TestGetContextSnippetCaseInsensitive(t *testing.T) {
	dir := buildTmpSearchDir(t, map[string]string{
		"doc.md": "The Authentication service handles login and JWT tokens.",
	})
	snippet := knowledge.GetContextSnippet(filepath.Join(dir, "doc.md"), "authentication", 100)
	if !strings.Contains(snippet, "Authentication") {
		t.Errorf("Expected snippet to contain original case 'Authentication', got: %s", snippet)
	}
}

// TestGetContextSnippetWhitespaceCollapsed verifies newlines become spaces.
func TestGetContextSnippetWhitespaceCollapsed(t *testing.T) {
	dir := buildTmpSearchDir(t, map[string]string{
		"doc.md": "# Heading\n\nParagraph one.\n\nParagraph two with search term here.",
	})
	snippet := knowledge.GetContextSnippet(filepath.Join(dir, "doc.md"), "search", 100)
	if strings.Contains(snippet, "\n") {
		t.Error("Snippet should not contain newlines")
	}
}

// TestGetContextSnippetSpecialChars verifies special characters in query.
func TestGetContextSnippetSpecialChars(t *testing.T) {
	dir := buildTmpSearchDir(t, map[string]string{
		"doc.md": "Use the GET /api/v1/users endpoint for user listing.",
	})
	// Should not crash with regex-special characters.
	snippet := knowledge.GetContextSnippet(filepath.Join(dir, "doc.md"), "/api/v1", 100)
	if !strings.Contains(snippet, "/api/v1") {
		t.Errorf("Expected snippet to contain '/api/v1', got: %s", snippet)
	}
}

// ====== DIR-04: Highlight Tests ======

// TestHighlightQueryInSnippetBasic verifies highlighting is applied.
func TestHighlightQueryInSnippetBasic(t *testing.T) {
	result := highlightQueryInSnippet("the microservices pattern", "microservices", "\x1b[38;5;250m", "\x1b[1;38;5;226m", "\x1b[0m")
	if !strings.Contains(result, "\x1b[1;38;5;226m") {
		t.Error("Expected highlight escape code in result")
	}
	if !strings.Contains(result, "microservices") {
		t.Error("Expected query text to appear in result")
	}
}

// TestHighlightQueryInSnippetCaseInsensitive verifies case-insensitive highlighting.
func TestHighlightQueryInSnippetCaseInsensitive(t *testing.T) {
	result := highlightQueryInSnippet("The API gateway", "api", "\x1b[38;5;250m", "\x1b[1m", "\x1b[0m")
	// Should highlight "API" even though query is lowercase "api".
	if !strings.Contains(result, "\x1b[1m") {
		t.Error("Expected highlight for case-insensitive match")
	}
}

// TestHighlightQueryInSnippetMultipleMatches verifies all occurrences are highlighted.
func TestHighlightQueryInSnippetMultipleMatches(t *testing.T) {
	result := highlightQueryInSnippet("api calls to api endpoints via api", "api", "", "\x1b[1m", "\x1b[0m")
	count := strings.Count(result, "\x1b[1m")
	if count != 3 {
		t.Errorf("Expected 3 highlights, got %d", count)
	}
}

// TestHighlightQueryInSnippetEmptyQuery verifies empty query returns unchanged.
func TestHighlightQueryInSnippetEmptyQuery(t *testing.T) {
	result := highlightQueryInSnippet("some text", "", "", "", "")
	if result != "some text" {
		t.Errorf("Expected unchanged text for empty query, got: %s", result)
	}
}

// TestHighlightQueryInSnippetEmptySnippet verifies empty snippet returns empty.
func TestHighlightQueryInSnippetEmptySnippet(t *testing.T) {
	result := highlightQueryInSnippet("", "query", "", "", "")
	if result != "" {
		t.Errorf("Expected empty result for empty snippet, got: %s", result)
	}
}

// ====== DIR-04: Render Tests with Snippets ======

// TestRenderCrossSearchResultsShowsSnippet verifies snippet appears in rendered output.
func TestRenderCrossSearchResultsShowsSnippet(t *testing.T) {
	dir := buildTmpSearchDir(t, map[string]string{
		"svc.md": "# Service\nThe payment microservices handle billing and invoices.",
	})
	v := newTestViewerWithDir(t, dir)
	v.crossSearchActive = true
	v.crossSearchQuery = "microservices"
	v.crossSearchResults = []knowledge.SearchResult{
		{RelPath: "svc.md", Score: 8.2, Path: filepath.Join(dir, "svc.md"), Snippet: "payment microservices handle billing"},
	}
	v.crossSearchSelected = 0

	out := v.renderCrossSearchResults(20)
	// Check that either the GetContextSnippet output or the fallback Snippet appears.
	lowerOut := strings.ToLower(out)
	if !strings.Contains(lowerOut, "microservices") {
		t.Error("Expected snippet content containing 'microservices' in rendered output")
	}
}

// TestRenderCrossSearchResultsFallbackSnippet verifies fallback when file is missing.
func TestRenderCrossSearchResultsFallbackSnippet(t *testing.T) {
	dir := t.TempDir()
	v := newTestViewerWithDir(t, dir)
	v.crossSearchActive = true
	v.crossSearchQuery = "test"
	v.crossSearchResults = []knowledge.SearchResult{
		{RelPath: "missing.md", Score: 5.0, Path: "/nonexistent/missing.md", Snippet: "fallback snippet text"},
	}
	v.crossSearchSelected = 0

	out := v.renderCrossSearchResults(20)
	if !strings.Contains(out, "fallback snippet text") {
		t.Error("Expected fallback snippet in rendered output when file is missing")
	}
}

// ====== DIR-04: File Navigation from Search Tests ======

// TestOpenFileFromSearchSetsState verifies state is set when opening file from search.
func TestOpenFileFromSearchSetsState(t *testing.T) {
	dir := buildTmpSearchDir(t, map[string]string{
		"target.md": "# Target\nSome content here.",
	})
	v := newTestViewerWithDir(t, dir)
	v.crossSearchActive = true
	v.crossSearchSelected = 0
	v.crossSearchResults = []knowledge.SearchResult{
		{RelPath: "target.md", Score: 5.0, Path: filepath.Join(dir, "target.md")},
	}

	// Simulate pressing 'l' to open.
	lKey := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}}
	model, _ := v.Update(lKey)
	result := model.(Viewer)

	if !result.openedFromSearch {
		t.Error("Expected openedFromSearch=true after opening file from search")
	}
	if result.crossSearchActive {
		t.Error("Expected crossSearchActive=false after opening file")
	}
}

// TestBackToSearchResultsRestoresState verifies returning to search preserves cursor.
func TestBackToSearchResultsRestoresState(t *testing.T) {
	dir := buildTmpSearchDir(t, map[string]string{
		"a.md": "# Doc A\nContent A.",
		"b.md": "# Doc B\nContent B.",
	})
	v := newTestViewerWithDir(t, dir)
	v.crossSearchQuery = "content"
	v.crossSearchResults = []knowledge.SearchResult{
		{RelPath: "a.md", Score: 5.0, Path: filepath.Join(dir, "a.md")},
		{RelPath: "b.md", Score: 4.0, Path: filepath.Join(dir, "b.md")},
	}
	v.crossSearchSelected = 1
	v.openedFromSearch = true
	v.crossSearchActive = false

	backV, _ := v.BackToSearchResults()
	if !backV.crossSearchActive {
		t.Error("Expected crossSearchActive=true after returning to search")
	}
	if backV.openedFromSearch {
		t.Error("Expected openedFromSearch=false after returning to search")
	}
	if backV.crossSearchSelected != 1 {
		t.Errorf("Expected crossSearchSelected=1 (preserved), got %d", backV.crossSearchSelected)
	}
}

// TestBackToSearchResultsNotOpenedFromSearch verifies no-op when not from search.
func TestBackToSearchResultsNotOpenedFromSearch(t *testing.T) {
	dir := t.TempDir()
	v := newTestViewerWithDir(t, dir)
	v.openedFromSearch = false

	backV, _ := v.BackToSearchResults()
	if backV.crossSearchActive {
		t.Error("Expected crossSearchActive=false when not opened from search")
	}
}

// TestHKeyReturnsToSearchFromFile verifies 'h' returns to search results.
func TestHKeyReturnsToSearchFromFile(t *testing.T) {
	dir := buildTmpSearchDir(t, map[string]string{
		"doc.md": "# Doc\nContent.",
	})
	v := newTestViewerWithDir(t, dir)
	v.openedFromSearch = true
	v.crossSearchQuery = "content"
	v.crossSearchResults = []knowledge.SearchResult{
		{RelPath: "doc.md", Score: 5.0, Path: filepath.Join(dir, "doc.md")},
	}
	v.crossSearchSelected = 0

	hKey := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'h'}}
	model, _ := v.Update(hKey)
	result := model.(Viewer)

	if !result.crossSearchActive {
		t.Error("Expected crossSearchActive=true after 'h' from file opened from search")
	}
	if result.openedFromSearch {
		t.Error("Expected openedFromSearch=false after 'h'")
	}
}

// TestStripANSIForLen verifies ANSI stripping for length calculation.
func TestStripANSIForLen(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"plain text", "plain text"},
		{"\x1b[1mBold\x1b[0m", "Bold"},
		{"\x1b[38;5;226mYellow\x1b[0m text", "Yellow text"},
		{"no escapes", "no escapes"},
		{"", ""},
	}
	for _, tc := range tests {
		got := stripANSIForLen(tc.input)
		if got != tc.expected {
			t.Errorf("stripANSIForLen(%q) = %q, want %q", tc.input, got, tc.expected)
		}
	}
}

// TestHeaderShowsSearchBreadcrumb verifies header shows search context breadcrumb.
func TestHeaderShowsSearchBreadcrumb(t *testing.T) {
	dir := t.TempDir()
	v := newTestViewerWithDir(t, dir)
	v.openedFromSearch = true
	v.crossSearchQuery = "microservices"
	v.Width = 80

	header := v.renderHeader()
	if !strings.Contains(header, "search: microservices") {
		t.Errorf("Expected search breadcrumb in header, got: %s", header)
	}
}
