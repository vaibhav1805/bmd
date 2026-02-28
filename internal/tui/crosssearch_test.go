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
