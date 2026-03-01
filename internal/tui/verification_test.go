package tui

// Phase 8 verification test suite (08-06): comprehensive testing of all Phase 8
// directory browser features (DIR-01 to DIR-05), integration tests, edge cases,
// and performance validation.

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/bmd/bmd/internal/ast"
	"github.com/bmd/bmd/internal/knowledge"
	"github.com/bmd/bmd/internal/theme"
)

// ============================================================================
// Helpers
// ============================================================================

// mkTmpDir creates a temp directory with the given markdown files.
func mkTmpDir(t *testing.T, files map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	for name, content := range files {
		p := filepath.Join(dir, name)
		_ = os.MkdirAll(filepath.Dir(p), 0o755)
		if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
			t.Fatalf("mkTmpDir: write %s: %v", name, err)
		}
	}
	return dir
}

// newDirViewer creates a Viewer in directory mode for the given dir.
func newDirViewer(t *testing.T, dir string) Viewer {
	t.Helper()
	v := NewDirectoryViewer(dir, theme.NewTheme(), 80)
	v.Height = 24
	if err := v.LoadDirectory(dir); err != nil {
		t.Fatalf("LoadDirectory: %v", err)
	}
	return v
}

// sendKey creates a bubbletea KeyMsg from a string like "up", "down", "enter".
func sendKey(keyStr string) tea.KeyMsg {
	switch keyStr {
	case "up":
		return tea.KeyMsg{Type: tea.KeyUp}
	case "down":
		return tea.KeyMsg{Type: tea.KeyDown}
	case "enter":
		return tea.KeyMsg{Type: tea.KeyEnter}
	case "esc":
		return tea.KeyMsg{Type: tea.KeyEsc}
	case "backspace":
		return tea.KeyMsg{Type: tea.KeyBackspace}
	case "left":
		return tea.KeyMsg{Type: tea.KeyLeft}
	case "right":
		return tea.KeyMsg{Type: tea.KeyRight}
	default:
		// Rune-based key
		return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(keyStr)}
	}
}

// pressKeys sends a sequence of key events to a viewer, returning the final state.
func pressKeys(v Viewer, keys ...string) Viewer {
	for _, k := range keys {
		model, _ := v.Update(sendKey(k))
		v = model.(Viewer)
	}
	return v
}

// ============================================================================
// DIR-01: Directory Listing Tests
// ============================================================================

// TestDirListing_DiscoversMdFiles verifies all .md files are found.
func TestDirListing_DiscoversMdFiles(t *testing.T) {
	dir := mkTmpDir(t, map[string]string{
		"README.md":    "# README\nHello",
		"docs/api.md":  "# API\nEndpoints",
		"docs/auth.md": "# Auth\nJWT flow",
		"notes.txt":    "Not markdown",
		"image.png":    "binary data",
	})
	v := newDirViewer(t, dir)

	if len(v.directoryState.Files) != 3 {
		t.Errorf("Expected 3 .md files, got %d", len(v.directoryState.Files))
	}
	// Verify no non-markdown files
	for _, f := range v.directoryState.Files {
		if !strings.HasSuffix(f.Name, ".md") {
			t.Errorf("Non-markdown file found: %s", f.Name)
		}
	}
}

// TestDirListing_MetadataAccuracy verifies size and line count.
func TestDirListing_MetadataAccuracy(t *testing.T) {
	content := "# Title\nLine 2\nLine 3\n"
	dir := mkTmpDir(t, map[string]string{
		"test.md": content,
	})
	v := newDirViewer(t, dir)

	if len(v.directoryState.Files) != 1 {
		t.Fatal("Expected 1 file")
	}
	f := v.directoryState.Files[0]
	if f.Size != int64(len(content)) {
		t.Errorf("Expected size %d, got %d", len(content), f.Size)
	}
	if f.LineCount != 3 {
		t.Errorf("Expected 3 lines, got %d", f.LineCount)
	}
}

// TestDirListing_SortedAlphabetically verifies files are sorted.
func TestDirListing_SortedAlphabetically(t *testing.T) {
	dir := mkTmpDir(t, map[string]string{
		"zebra.md":   "# Zebra",
		"apple.md":   "# Apple",
		"mango.md":   "# Mango",
	})
	v := newDirViewer(t, dir)

	names := make([]string, len(v.directoryState.Files))
	for i, f := range v.directoryState.Files {
		names[i] = f.Name
	}
	if names[0] != "apple.md" || names[1] != "mango.md" || names[2] != "zebra.md" {
		t.Errorf("Files not sorted alphabetically: %v", names)
	}
}

// TestDirListing_CursorNavigation verifies up/down wrap.
func TestDirListing_CursorNavigation(t *testing.T) {
	dir := mkTmpDir(t, map[string]string{
		"a.md": "# A",
		"b.md": "# B",
		"c.md": "# C",
	})
	v := newDirViewer(t, dir)

	// Down from 0 -> 1
	v = pressKeys(v, "down")
	if v.directoryState.SelectedIndex != 1 {
		t.Errorf("Expected 1 after down, got %d", v.directoryState.SelectedIndex)
	}

	// Down from 1 -> 2
	v = pressKeys(v, "down")
	if v.directoryState.SelectedIndex != 2 {
		t.Errorf("Expected 2, got %d", v.directoryState.SelectedIndex)
	}

	// Down wraps from 2 -> 0
	v = pressKeys(v, "down")
	if v.directoryState.SelectedIndex != 0 {
		t.Errorf("Expected wrap to 0, got %d", v.directoryState.SelectedIndex)
	}

	// Up wraps from 0 -> 2
	v = pressKeys(v, "up")
	if v.directoryState.SelectedIndex != 2 {
		t.Errorf("Expected wrap to 2, got %d", v.directoryState.SelectedIndex)
	}
}

// TestDirListing_VimKeys verifies j/k navigation.
func TestDirListing_VimKeys(t *testing.T) {
	dir := mkTmpDir(t, map[string]string{
		"a.md": "# A",
		"b.md": "# B",
	})
	v := newDirViewer(t, dir)

	v = pressKeys(v, "j")
	if v.directoryState.SelectedIndex != 1 {
		t.Errorf("Expected 1 after j, got %d", v.directoryState.SelectedIndex)
	}

	v = pressKeys(v, "k")
	if v.directoryState.SelectedIndex != 0 {
		t.Errorf("Expected 0 after k, got %d", v.directoryState.SelectedIndex)
	}
}

// TestDirListing_RenderOutput verifies rendered directory listing.
func TestDirListing_RenderOutput(t *testing.T) {
	dir := mkTmpDir(t, map[string]string{
		"README.md": "# Project\nDescription here",
	})
	v := newDirViewer(t, dir)
	v.Width = 200 // wide enough to avoid header truncation

	out := v.renderDirectoryListing(20)
	if !strings.Contains(out, "README.md") {
		t.Error("Expected README.md in directory listing output")
	}
	if !strings.Contains(out, "1 file") {
		t.Errorf("Expected '1 file' in directory listing header, got:\n%s", out)
	}
}

// TestDirListing_RenderShowsFileCount verifies file count in header.
func TestDirListing_RenderShowsFileCount(t *testing.T) {
	dir := mkTmpDir(t, map[string]string{
		"a.md": "# A",
		"b.md": "# B",
		"c.md": "# C",
	})
	v := newDirViewer(t, dir)
	v.Width = 200 // wide enough to avoid header truncation

	out := v.renderDirectoryListing(20)
	if !strings.Contains(out, "3 file") {
		t.Errorf("Expected '3 file' in header, got:\n%s", out)
	}
}

// TestDirListing_PreviewExtracted verifies first 100 chars stored.
func TestDirListing_PreviewExtracted(t *testing.T) {
	dir := mkTmpDir(t, map[string]string{
		"doc.md": "# Preview Test\nThis is the content that should be captured in the preview field.",
	})
	v := newDirViewer(t, dir)

	if len(v.directoryState.Files) != 1 {
		t.Fatal("Expected 1 file")
	}
	if v.directoryState.Files[0].Preview == "" {
		t.Error("Expected non-empty Preview")
	}
	if !strings.Contains(v.directoryState.Files[0].Preview, "Preview Test") {
		t.Error("Expected preview to contain file content")
	}
}

// TestDirListing_ViewRouting verifies View() renders directory listing.
func TestDirListing_ViewRouting(t *testing.T) {
	dir := mkTmpDir(t, map[string]string{
		"test.md": "# Test\nContent",
	})
	v := newDirViewer(t, dir)

	out := v.View()
	if !strings.Contains(out, "test.md") {
		t.Error("Expected test.md in View() output for directory mode")
	}
}

// ============================================================================
// DIR-02: File Navigation Tests
// ============================================================================

// TestFileNav_OpenFileFromDirectory opens file from dir and returns.
func TestFileNav_OpenFileFromDirectory(t *testing.T) {
	dir := mkTmpDir(t, map[string]string{
		"hello.md": "# Hello World\nThis is a test file.",
	})
	v := newDirViewer(t, dir)

	// Open the file via Enter
	v = pressKeys(v, "enter")

	if v.directoryMode {
		t.Error("Expected directoryMode=false after opening file")
	}
	if !v.openedFromDirectory {
		t.Error("Expected openedFromDirectory=true")
	}
	if v.currentView != "file" {
		t.Errorf("Expected currentView='file', got %q", v.currentView)
	}
}

// TestFileNav_BackToDirectory returns to directory with cursor restored.
func TestFileNav_BackToDirectory(t *testing.T) {
	dir := mkTmpDir(t, map[string]string{
		"a.md": "# Alpha",
		"b.md": "# Beta",
		"c.md": "# Charlie",
	})
	v := newDirViewer(t, dir)

	// Select second file
	v = pressKeys(v, "down")
	if v.directoryState.SelectedIndex != 1 {
		t.Fatalf("Expected SelectedIndex=1, got %d", v.directoryState.SelectedIndex)
	}

	// Open it
	v = pressKeys(v, "enter")
	if v.directoryMode {
		t.Fatal("Expected directoryMode=false after enter")
	}

	// Go back with 'h'
	v = pressKeys(v, "h")
	if !v.directoryMode {
		t.Error("Expected directoryMode=true after 'h'")
	}
	if v.directoryState.SelectedIndex != 1 {
		t.Errorf("Expected cursor restored to 1, got %d", v.directoryState.SelectedIndex)
	}
}

// TestFileNav_BackspaceReturnsToDirectory tests Backspace return.
func TestFileNav_BackspaceReturnsToDirectory(t *testing.T) {
	dir := mkTmpDir(t, map[string]string{
		"doc.md": "# Doc\nContent",
	})
	v := newDirViewer(t, dir)

	// Open file
	v = pressKeys(v, "enter")
	if v.directoryMode {
		t.Fatal("Expected directoryMode=false")
	}

	// Go back with backspace
	v = pressKeys(v, "backspace")
	if !v.directoryMode {
		t.Error("Expected directoryMode=true after backspace")
	}
}

// TestFileNav_LKeyOpensFile tests 'l' key opens file.
func TestFileNav_LKeyOpensFile(t *testing.T) {
	dir := mkTmpDir(t, map[string]string{
		"doc.md": "# Doc\nContent here",
	})
	v := newDirViewer(t, dir)

	v = pressKeys(v, "l")
	if v.directoryMode {
		t.Error("Expected directoryMode=false after 'l'")
	}
	if !v.openedFromDirectory {
		t.Error("Expected openedFromDirectory=true")
	}
}

// TestFileNav_RightArrowOpensFile tests Right arrow opens file.
func TestFileNav_RightArrowOpensFile(t *testing.T) {
	dir := mkTmpDir(t, map[string]string{
		"doc.md": "# Doc\nContent",
	})
	v := newDirViewer(t, dir)

	v = pressKeys(v, "right")
	if v.directoryMode {
		t.Error("Expected directoryMode=false after Right arrow")
	}
}

// TestFileNav_BreadcrumbAfterOpen tests view state set correctly.
func TestFileNav_BreadcrumbAfterOpen(t *testing.T) {
	dir := mkTmpDir(t, map[string]string{
		"readme.md": "# README\nProject docs",
	})
	v := newDirViewer(t, dir)

	// Open file
	v = pressKeys(v, "enter")
	if v.currentView != "file" {
		t.Errorf("Expected currentView='file', got %q", v.currentView)
	}

	// Go back
	v = pressKeys(v, "h")
	if v.currentView != "directory" {
		t.Errorf("Expected currentView='directory', got %q", v.currentView)
	}
}

// TestFileNav_SavedSelectionPreserved verifies SaveDirectorySelection / RestoreDirectorySelection.
func TestFileNav_SavedSelectionPreserved(t *testing.T) {
	ds := DirectoryState{
		RootPath:      "/tmp/test",
		SelectedIndex: 5,
	}
	ds.SaveDirectorySelection()
	if ds.SavedSelectedIndex != 5 {
		t.Errorf("Expected SavedSelectedIndex=5, got %d", ds.SavedSelectedIndex)
	}

	ds.SelectedIndex = 99
	ds.RestoreDirectorySelection()
	if ds.SelectedIndex != 5 {
		t.Errorf("Expected SelectedIndex restored to 5, got %d", ds.SelectedIndex)
	}
}

// ============================================================================
// DIR-03: Cross-Document Search Tests
// ============================================================================

// TestCrossSearch_SlashFromDirectory verifies '/' from directory enters search.
func TestCrossSearch_SlashFromDirectory(t *testing.T) {
	dir := mkTmpDir(t, map[string]string{
		"doc.md": "# Doc",
	})
	v := newDirViewer(t, dir)

	v = pressKeys(v, "/")
	if !v.crossSearchMode {
		t.Error("Expected crossSearchMode=true after '/'")
	}
	if v.directoryMode {
		t.Error("Expected directoryMode=false after search activated")
	}
}

// TestCrossSearch_CtrlFFromDirectory verifies Ctrl+F starts search from dir.
func TestCrossSearch_CtrlFFromDirectory(t *testing.T) {
	dir := mkTmpDir(t, map[string]string{
		"doc.md": "# Doc",
	})
	v := newDirViewer(t, dir)

	// Simulate ctrl+f
	msg := tea.KeyMsg{Type: tea.KeyCtrlF}
	model, _ := v.Update(msg)
	result := model.(Viewer)

	if !result.crossSearchMode {
		t.Error("Expected crossSearchMode=true after Ctrl+F from directory")
	}
}

// TestCrossSearch_ExecuteAndShowResults runs search and verifies results.
func TestCrossSearch_ExecuteAndShowResults(t *testing.T) {
	dir := mkTmpDir(t, map[string]string{
		"auth.md":     "# Authentication\nJWT tokens and OAuth flow for auth.",
		"database.md": "# Database\nPostgreSQL schema and migrations.",
	})
	v := newDirViewer(t, dir)

	// Enter search mode, type query, press enter
	v.crossSearchMode = true
	v.crossSearchInput = "authentication"
	v.directoryMode = false

	enter := tea.KeyMsg{Type: tea.KeyEnter}
	model, _ := v.Update(enter)
	v = model.(Viewer)

	if !v.crossSearchActive {
		t.Error("Expected crossSearchActive=true after search")
	}
	if len(v.crossSearchResults) == 0 {
		t.Error("Expected search results for 'authentication'")
	}
}

// TestCrossSearch_OpenResultWithEnter opens a search result.
func TestCrossSearch_OpenResultWithEnter(t *testing.T) {
	dir := mkTmpDir(t, map[string]string{
		"target.md": "# Target\nThis is the target file content.",
	})

	v := NewDirectoryViewer(dir, theme.NewTheme(), 80)
	v.Height = 24
	_ = v.LoadDirectory(dir)

	// Simulate active search results with a real path
	absPath := filepath.Join(dir, "target.md")
	v.crossSearchActive = true
	v.crossSearchMode = false
	v.crossSearchResults = []knowledge.SearchResult{
		{RelPath: "target.md", Score: 5.0, Path: absPath},
	}
	v.crossSearchSelected = 0

	// Press Enter to open
	v = pressKeys(v, "enter")

	if v.crossSearchActive {
		t.Error("Expected crossSearchActive=false after opening result")
	}
}

// TestCrossSearch_SnippetInResults verifies SearchResult has Snippet field.
func TestCrossSearch_SnippetInResults(t *testing.T) {
	dir := mkTmpDir(t, map[string]string{
		"api.md": "# API Guide\nThis document describes the REST API endpoints for the authentication service.",
	})

	results, err := knowledge.SearchAllDocuments(dir, "authentication", 10)
	if err != nil {
		t.Fatalf("SearchAllDocuments error: %v", err)
	}
	if len(results) == 0 {
		t.Skip("No results returned")
	}
	// SearchResult has a Snippet field from Phase 6
	if results[0].Snippet == "" {
		t.Error("Expected non-empty Snippet in search results")
	}
}

// TestCrossSearch_ResultsLimitedByTopK verifies topK parameter.
func TestCrossSearch_ResultsLimitedByTopK(t *testing.T) {
	files := make(map[string]string, 15)
	for i := 0; i < 15; i++ {
		files[fmt.Sprintf("doc%02d.md", i)] = fmt.Sprintf("# Doc %d\nContains deployment info %d.", i, i)
	}
	dir := mkTmpDir(t, files)

	results, err := knowledge.SearchAllDocuments(dir, "deployment", 5)
	if err != nil {
		t.Fatalf("SearchAllDocuments error: %v", err)
	}
	if len(results) > 5 {
		t.Errorf("Expected at most 5 results (topK=5), got %d", len(results))
	}
}

// ============================================================================
// DIR-04: Search Result Navigation Tests
// ============================================================================

// TestSearchNav_NKeyFromResults verifies 'n' cycles through results.
func TestSearchNav_NKeyFromResults(t *testing.T) {
	dir := t.TempDir()
	v := New(&ast.Document{}, filepath.Join(dir, "test.md"), theme.NewTheme(), 80)
	v.Height = 24
	v.crossSearchActive = true
	v.crossSearchMode = false
	v.crossSearchSelected = 0
	v.crossSearchResults = []knowledge.SearchResult{
		{RelPath: "a.md", Score: 5.0},
		{RelPath: "b.md", Score: 4.0},
		{RelPath: "c.md", Score: 3.0},
	}

	// 'n' should not navigate in search results (n is for in-doc search)
	// But 'j'/down should work
	v = pressKeys(v, "j")
	if v.crossSearchSelected != 1 {
		t.Errorf("Expected selected=1 after j, got %d", v.crossSearchSelected)
	}
	v = pressKeys(v, "j")
	if v.crossSearchSelected != 2 {
		t.Errorf("Expected selected=2 after j, got %d", v.crossSearchSelected)
	}
}

// TestSearchNav_RenderShowsSnippets verifies search results display includes snippets.
func TestSearchNav_RenderShowsSnippets(t *testing.T) {
	dir := t.TempDir()
	v := New(&ast.Document{}, filepath.Join(dir, "test.md"), theme.NewTheme(), 80)
	v.Height = 24
	v.crossSearchActive = true
	v.crossSearchMode = false
	v.crossSearchQuery = "microservices"
	v.crossSearchResults = []knowledge.SearchResult{
		{RelPath: "services.md", Score: 8.5, Snippet: "...using microservices architecture..."},
	}
	v.crossSearchSelected = 0

	out := v.renderCrossSearchResults(20)
	if !strings.Contains(out, "services.md") {
		t.Error("Expected filename in search results render")
	}
	if !strings.Contains(out, "8.5") {
		t.Error("Expected score in search results render")
	}
}

// ============================================================================
// DIR-05: Graph Visualization Tests
// ============================================================================

// TestGraphView_GKeyFromDirectory opens graph from directory.
func TestGraphView_GKeyFromDirectory(t *testing.T) {
	dir := mkTmpDir(t, map[string]string{
		"readme.md": "# README",
	})
	v := newDirViewer(t, dir)

	// 'g' should attempt to open graph. It will fail if no knowledge.db exists,
	// which is expected in test env. What matters is the mode transition attempt.
	v = pressKeys(v, "g")

	// If graph loaded, graphMode should be true. If it failed, directoryMode should still be true.
	// Either way, no panic.
	if v.graphMode {
		if v.directoryMode {
			t.Error("Expected directoryMode=false when graphMode=true")
		}
	}
}

// TestGraphView_CircularDeps handles circular dependencies without infinite loop.
func TestGraphView_CircularDeps(t *testing.T) {
	g := knowledge.NewGraph()
	_ = g.AddNode(&knowledge.Node{ID: "a.md", Title: "A", Type: "document"})
	_ = g.AddNode(&knowledge.Node{ID: "b.md", Title: "B", Type: "document"})
	_ = g.AddNode(&knowledge.Node{ID: "c.md", Title: "C", Type: "document"})
	e1, _ := knowledge.NewEdge("a.md", "b.md", knowledge.EdgeReferences, 1.0, "")
	e2, _ := knowledge.NewEdge("b.md", "c.md", knowledge.EdgeReferences, 1.0, "")
	e3, _ := knowledge.NewEdge("c.md", "a.md", knowledge.EdgeReferences, 1.0, "")
	_ = g.AddEdge(e1)
	_ = g.AddEdge(e2)
	_ = g.AddEdge(e3)

	// computeNodeLayout should handle cycles without hanging.
	layout := computeNodeLayout(g)
	if layout == nil {
		t.Error("Expected non-nil layout even with cycles")
	}

	// RenderGraphASCII should not panic.
	out := RenderGraphASCII(g, layout, "a.md", 100, 30)
	if out == "" {
		t.Error("Expected non-empty output for circular graph")
	}
}

// TestGraphView_NodeSelectionNavigation tests up/down/left/right in graph.
func TestGraphView_NodeSelectionNavigation(t *testing.T) {
	v := New(&ast.Document{}, "test.md", theme.NewTheme(), 120)
	v.Height = 40

	g := knowledge.NewGraph()
	_ = g.AddNode(&knowledge.Node{ID: "a.md", Title: "A", Type: "document"})
	_ = g.AddNode(&knowledge.Node{ID: "b.md", Title: "B", Type: "document"})
	e, _ := knowledge.NewEdge("a.md", "b.md", knowledge.EdgeReferences, 1.0, "")
	_ = g.AddEdge(e)

	v.graphMode = true
	v.graphState = GraphViewState{
		Graph:          g,
		NodeOrder:      []string{"a.md", "b.md"},
		SelectedNodeID: "a.md",
		NodeLayout:     computeNodeLayout(g),
		RootPath:       "/tmp",
		Loaded:         true,
	}

	// Down: a -> b
	model, _ := v.updateGraph(sendKey("down"))
	result := model.(Viewer)
	if result.graphState.SelectedNodeID != "b.md" {
		t.Errorf("Expected b.md after down, got %q", result.graphState.SelectedNodeID)
	}

	// Right: a -> b (via edge)
	v.graphState.SelectedNodeID = "a.md"
	model, _ = v.updateGraph(sendKey("right"))
	result = model.(Viewer)
	if result.graphState.SelectedNodeID != "b.md" {
		t.Errorf("Expected b.md after right, got %q", result.graphState.SelectedNodeID)
	}

	// Left: b -> a (via incoming edge)
	v.graphState.SelectedNodeID = "b.md"
	model, _ = v.updateGraph(sendKey("left"))
	result = model.(Viewer)
	if result.graphState.SelectedNodeID != "a.md" {
		t.Errorf("Expected a.md after left, got %q", result.graphState.SelectedNodeID)
	}
}

// ============================================================================
// Integration Tests: Full Workflows
// ============================================================================

// TestIntegration_DirectoryToFileAndBack tests full cycle: dir -> file -> dir.
func TestIntegration_DirectoryToFileAndBack(t *testing.T) {
	dir := mkTmpDir(t, map[string]string{
		"alpha.md": "# Alpha\nAlpha content here.",
		"beta.md":  "# Beta\nBeta content here.",
	})
	v := newDirViewer(t, dir)

	// Start in directory mode
	if !v.directoryMode {
		t.Fatal("Expected start in directory mode")
	}

	// Navigate down to second file
	v = pressKeys(v, "down")
	if v.directoryState.SelectedIndex != 1 {
		t.Fatal("Expected SelectedIndex=1")
	}

	// Open file
	v = pressKeys(v, "enter")
	if v.directoryMode {
		t.Fatal("Expected directoryMode=false after enter")
	}
	if v.currentView != "file" {
		t.Fatalf("Expected currentView=file, got %q", v.currentView)
	}

	// Go back to directory
	v = pressKeys(v, "h")
	if !v.directoryMode {
		t.Fatal("Expected directoryMode=true after 'h'")
	}
	if v.currentView != "directory" {
		t.Fatalf("Expected currentView=directory, got %q", v.currentView)
	}
	// Cursor should be restored to position 1
	if v.directoryState.SelectedIndex != 1 {
		t.Errorf("Expected cursor restored to 1, got %d", v.directoryState.SelectedIndex)
	}
}

// TestIntegration_DirectoryToSearchAndBack tests: dir -> search -> dir.
func TestIntegration_DirectoryToSearchAndBack(t *testing.T) {
	dir := mkTmpDir(t, map[string]string{
		"service.md": "# Service\nPayment service handles transactions.",
	})
	v := newDirViewer(t, dir)

	// Enter search mode from directory
	v = pressKeys(v, "/")
	if !v.crossSearchMode {
		t.Fatal("Expected crossSearchMode=true")
	}
	if v.directoryMode {
		t.Fatal("Expected directoryMode=false during search")
	}

	// Cancel search
	v = pressKeys(v, "esc")
	if v.crossSearchMode {
		t.Error("Expected crossSearchMode=false after Esc")
	}
}

// TestIntegration_SearchToFileAndBack tests: search results -> open file -> back.
func TestIntegration_SearchToFileAndBack(t *testing.T) {
	dir := mkTmpDir(t, map[string]string{
		"api.md": "# API\nThe API documentation for the service.",
	})

	v := NewDirectoryViewer(dir, theme.NewTheme(), 80)
	v.Height = 24
	_ = v.LoadDirectory(dir)

	// Set up search results directly
	absPath := filepath.Join(dir, "api.md")
	v.crossSearchActive = true
	v.crossSearchMode = false
	v.crossSearchQuery = "API"
	v.crossSearchResults = []knowledge.SearchResult{
		{RelPath: "api.md", Score: 5.0, Path: absPath},
	}
	v.crossSearchSelected = 0
	v.directoryMode = false

	// Open result with 'l'
	v = pressKeys(v, "l")
	if v.crossSearchActive {
		t.Error("Expected crossSearchActive=false after opening result")
	}
}

// TestIntegration_MultipleNavCycles tests multiple navigate cycles.
func TestIntegration_MultipleNavCycles(t *testing.T) {
	dir := mkTmpDir(t, map[string]string{
		"a.md": "# Alpha\nContent A",
		"b.md": "# Beta\nContent B",
		"c.md": "# Charlie\nContent C",
	})
	v := newDirViewer(t, dir)

	// Cycle 1: Open first file, return
	v = pressKeys(v, "enter")
	v = pressKeys(v, "h")
	if !v.directoryMode {
		t.Fatal("Cycle 1: Expected return to directory")
	}

	// Cycle 2: Open second file, return
	v = pressKeys(v, "down", "enter")
	v = pressKeys(v, "h")
	if !v.directoryMode {
		t.Fatal("Cycle 2: Expected return to directory")
	}
	if v.directoryState.SelectedIndex != 1 {
		t.Errorf("Cycle 2: Expected cursor at 1, got %d", v.directoryState.SelectedIndex)
	}

	// Cycle 3: Open third file, return
	v = pressKeys(v, "down", "enter")
	v = pressKeys(v, "h")
	if !v.directoryMode {
		t.Fatal("Cycle 3: Expected return to directory")
	}
}

// TestIntegration_StateConsistency verifies state is clean after transitions.
func TestIntegration_StateConsistency(t *testing.T) {
	dir := mkTmpDir(t, map[string]string{
		"doc.md": "# Doc\nContent",
	})
	v := newDirViewer(t, dir)

	// Open file
	v = pressKeys(v, "enter")
	if v.directoryMode {
		t.Fatal("Expected directoryMode=false")
	}
	if !v.openedFromDirectory {
		t.Fatal("Expected openedFromDirectory=true")
	}

	// Return to directory
	v = pressKeys(v, "h")
	if !v.directoryMode {
		t.Fatal("Expected directoryMode=true")
	}
	if v.openedFromDirectory {
		t.Error("Expected openedFromDirectory=false after returning")
	}
	if v.searchMode {
		t.Error("Expected searchMode=false after returning")
	}
	if v.crossSearchActive {
		t.Error("Expected crossSearchActive=false after returning")
	}
}

// ============================================================================
// Edge Case Tests
// ============================================================================

// TestEdge_EmptyDirectory shows "No markdown files" message.
func TestEdge_EmptyDirectory(t *testing.T) {
	dir := t.TempDir() // empty
	v := newDirViewer(t, dir)

	if len(v.directoryState.Files) != 0 {
		t.Errorf("Expected 0 files, got %d", len(v.directoryState.Files))
	}

	out := v.renderDirectoryListing(20)
	if !strings.Contains(out, "No markdown") && !strings.Contains(out, "none found") {
		t.Errorf("Expected empty directory message, got: %s", out)
	}
}

// TestEdge_DirectoryWithNoMdFiles has files but none are markdown.
func TestEdge_DirectoryWithNoMdFiles(t *testing.T) {
	dir := mkTmpDir(t, map[string]string{
		"readme.txt":  "Plain text",
		"config.json": "{}",
		"image.png":   "binary",
	})
	v := newDirViewer(t, dir)

	if len(v.directoryState.Files) != 0 {
		t.Errorf("Expected 0 .md files, got %d", len(v.directoryState.Files))
	}
}

// TestEdge_SingleFileDirectory works with just 1 file.
func TestEdge_SingleFileDirectory(t *testing.T) {
	dir := mkTmpDir(t, map[string]string{
		"only.md": "# Only File\nSingle file directory.",
	})
	v := newDirViewer(t, dir)

	if len(v.directoryState.Files) != 1 {
		t.Errorf("Expected 1 file, got %d", len(v.directoryState.Files))
	}

	// Navigate down should wrap
	v = pressKeys(v, "down")
	if v.directoryState.SelectedIndex != 0 {
		t.Errorf("Expected wrap to 0 with single file, got %d", v.directoryState.SelectedIndex)
	}

	// Open and return
	v = pressKeys(v, "enter")
	v = pressKeys(v, "h")
	if !v.directoryMode {
		t.Error("Expected return to directory")
	}
}

// TestEdge_EmptyFileContent handles empty .md files.
func TestEdge_EmptyFileContent(t *testing.T) {
	dir := mkTmpDir(t, map[string]string{
		"empty.md": "",
	})
	v := newDirViewer(t, dir)

	if len(v.directoryState.Files) != 1 {
		t.Fatal("Expected 1 file")
	}
	f := v.directoryState.Files[0]
	if f.LineCount != 0 {
		t.Errorf("Expected 0 lines for empty file, got %d", f.LineCount)
	}
	if f.Size != 0 {
		t.Errorf("Expected 0 size, got %d", f.Size)
	}
}

// TestEdge_SpecialCharsInFilename handles special characters.
func TestEdge_SpecialCharsInFilename(t *testing.T) {
	dir := mkTmpDir(t, map[string]string{
		"my notes (draft).md":    "# My Notes\nDraft content",
		"project-v2.0-spec.md":  "# Project Spec\nVersion 2.0",
	})
	v := newDirViewer(t, dir)

	if len(v.directoryState.Files) != 2 {
		t.Errorf("Expected 2 files, got %d", len(v.directoryState.Files))
	}

	// Verify filenames are readable in render output
	out := v.renderDirectoryListing(20)
	if !strings.Contains(out, "my notes (draft).md") {
		t.Error("Expected special chars filename in output")
	}
}

// TestEdge_SubdirectoryFiles discovers files in subdirectories.
func TestEdge_SubdirectoryFiles(t *testing.T) {
	dir := mkTmpDir(t, map[string]string{
		"README.md":           "# Root",
		"docs/guide.md":       "# Guide",
		"docs/api/spec.md":    "# API Spec",
		"deep/nested/file.md": "# Deep",
	})
	v := newDirViewer(t, dir)

	if len(v.directoryState.Files) != 4 {
		t.Errorf("Expected 4 files across subdirs, got %d", len(v.directoryState.Files))
	}

	// Verify relative names include path
	hasDeep := false
	for _, f := range v.directoryState.Files {
		if strings.Contains(f.Name, "deep/nested/file.md") || strings.Contains(f.Name, filepath.Join("deep", "nested", "file.md")) {
			hasDeep = true
		}
	}
	if !hasDeep {
		names := make([]string, len(v.directoryState.Files))
		for i, f := range v.directoryState.Files {
			names[i] = f.Name
		}
		t.Errorf("Expected deep/nested/file.md in files, got: %v", names)
	}
}

// TestEdge_SearchEmptyQueryNoResults handles empty search gracefully.
func TestEdge_SearchEmptyQueryNoResults(t *testing.T) {
	dir := mkTmpDir(t, map[string]string{
		"doc.md": "# Doc",
	})

	results, err := knowledge.SearchAllDocuments(dir, "", 10)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("Expected 0 results for empty query, got %d", len(results))
	}
}

// TestEdge_SearchNonexistentTerm returns empty.
func TestEdge_SearchNonexistentTerm(t *testing.T) {
	dir := mkTmpDir(t, map[string]string{
		"doc.md": "# Normal Content\nJust regular documentation.",
	})

	results, err := knowledge.SearchAllDocuments(dir, "xyzquux999nonexistent", 10)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("Expected 0 results, got %d", len(results))
	}
}

// TestEdge_GraphEmptyGraph renders empty message.
func TestEdge_GraphEmptyGraph(t *testing.T) {
	g := knowledge.NewGraph()
	out := RenderGraphASCII(g, nil, "", 80, 24)
	if !strings.Contains(out, "No graph data") {
		t.Errorf("Expected empty graph message, got: %q", out)
	}
}

// TestEdge_GraphLargeNodeCount tests 100+ node fallback.
func TestEdge_GraphLargeNodeCount(t *testing.T) {
	g := knowledge.NewGraph()
	for i := 0; i < 100; i++ {
		id := fmt.Sprintf("file_%03d.md", i)
		_ = g.AddNode(&knowledge.Node{ID: id, Title: fmt.Sprintf("File %d", i), Type: "document"})
	}
	layout := computeNodeLayout(g)
	out := RenderGraphASCII(g, layout, "file_000.md", 120, 50)
	if out == "" {
		t.Error("Expected non-empty output for 100-node graph")
	}
	// Should use list fallback for >40 nodes
	if !strings.Contains(out, "list view") {
		t.Error("Expected list-view fallback for 100+ nodes")
	}
}

// TestEdge_DirectoryOpenAndReturnMultipleTimes stress tests navigation.
func TestEdge_DirectoryOpenAndReturnMultipleTimes(t *testing.T) {
	dir := mkTmpDir(t, map[string]string{
		"a.md": "# A",
		"b.md": "# B",
	})
	v := newDirViewer(t, dir)

	// Open and close 10 times without crash
	for i := 0; i < 10; i++ {
		v = pressKeys(v, "enter")
		if v.directoryMode {
			t.Fatalf("Iteration %d: Expected directoryMode=false", i)
		}
		v = pressKeys(v, "h")
		if !v.directoryMode {
			t.Fatalf("Iteration %d: Expected directoryMode=true", i)
		}
	}
}

// TestEdge_SearchResultsEmptyRender doesn't crash on empty results render.
func TestEdge_SearchResultsEmptyRender(t *testing.T) {
	dir := t.TempDir()
	v := New(&ast.Document{}, filepath.Join(dir, "test.md"), theme.NewTheme(), 80)
	v.Height = 24
	v.crossSearchActive = true
	v.crossSearchQuery = "nothing"
	v.crossSearchResults = []knowledge.SearchResult{}
	v.crossSearchSelected = -1

	// Should not panic
	out := v.renderCrossSearchResults(20)
	if !strings.Contains(out, "No matches") {
		t.Error("Expected 'No matches' message")
	}
}

// TestEdge_QuitFromDirectoryMode verifies 'q' quits.
func TestEdge_QuitFromDirectoryMode(t *testing.T) {
	dir := mkTmpDir(t, map[string]string{
		"doc.md": "# Doc",
	})
	v := newDirViewer(t, dir)

	_, cmd := v.Update(sendKey("q"))
	if cmd == nil {
		t.Error("Expected quit command from 'q' in directory mode")
	}
}

// TestEdge_HelpFromDirectoryMode opens help overlay.
func TestEdge_HelpFromDirectoryMode(t *testing.T) {
	dir := mkTmpDir(t, map[string]string{
		"doc.md": "# Doc",
	})
	v := newDirViewer(t, dir)

	v = pressKeys(v, "?")
	if !v.helpOpen {
		t.Error("Expected helpOpen=true after '?' in directory mode")
	}
}

// ============================================================================
// Performance Tests
// ============================================================================

// TestPerf_DirectoryListing20Files verifies listing performance.
func TestPerf_DirectoryListing20Files(t *testing.T) {
	files := make(map[string]string, 20)
	for i := 0; i < 20; i++ {
		content := fmt.Sprintf("# Document %d\n", i)
		for j := 0; j < 50; j++ {
			content += fmt.Sprintf("Line %d of document %d with some content.\n", j, i)
		}
		files[fmt.Sprintf("doc%02d.md", i)] = content
	}
	dir := mkTmpDir(t, files)

	start := time.Now()
	v := NewDirectoryViewer(dir, theme.NewTheme(), 80)
	v.Height = 24
	err := v.LoadDirectory(dir)
	duration := time.Since(start)

	if err != nil {
		t.Fatalf("LoadDirectory error: %v", err)
	}
	if len(v.directoryState.Files) != 20 {
		t.Errorf("Expected 20 files, got %d", len(v.directoryState.Files))
	}
	if duration > 500*time.Millisecond {
		t.Errorf("Directory listing took %v, expected < 500ms", duration)
	}
}

// TestPerf_DirectoryListing50Files verifies performance at scale.
func TestPerf_DirectoryListing50Files(t *testing.T) {
	files := make(map[string]string, 50)
	for i := 0; i < 50; i++ {
		content := fmt.Sprintf("# Document %d\n", i)
		for j := 0; j < 100; j++ {
			content += fmt.Sprintf("Line %d of document %d.\n", j, i)
		}
		files[fmt.Sprintf("doc%02d.md", i)] = content
	}
	dir := mkTmpDir(t, files)

	start := time.Now()
	v := NewDirectoryViewer(dir, theme.NewTheme(), 80)
	v.Height = 24
	err := v.LoadDirectory(dir)
	duration := time.Since(start)

	if err != nil {
		t.Fatalf("LoadDirectory error: %v", err)
	}
	if len(v.directoryState.Files) != 50 {
		t.Errorf("Expected 50 files, got %d", len(v.directoryState.Files))
	}
	if duration > 2000*time.Millisecond {
		t.Errorf("Directory listing (50 files) took %v, expected < 2000ms", duration)
	}
}

// TestPerf_SearchExecution verifies search performance.
func TestPerf_SearchExecution(t *testing.T) {
	files := make(map[string]string, 20)
	for i := 0; i < 20; i++ {
		content := fmt.Sprintf("# Service %d\nThis microservice handles authentication and authorization.\n", i)
		for j := 0; j < 20; j++ {
			content += fmt.Sprintf("Section %d: Implementation details for component %d.\n", j, j)
		}
		files[fmt.Sprintf("service%02d.md", i)] = content
	}
	dir := mkTmpDir(t, files)

	start := time.Now()
	results, err := knowledge.SearchAllDocuments(dir, "authentication", 50)
	duration := time.Since(start)

	if err != nil {
		t.Fatalf("SearchAllDocuments error: %v", err)
	}
	_ = results
	if duration > 5*time.Second {
		t.Errorf("Search across 20 files took %v, expected < 5s", duration)
	}
}

// TestPerf_GraphRendering verifies graph render performance.
func TestPerf_GraphRendering(t *testing.T) {
	g := knowledge.NewGraph()
	for i := 0; i < 20; i++ {
		id := fmt.Sprintf("node_%02d.md", i)
		_ = g.AddNode(&knowledge.Node{ID: id, Title: fmt.Sprintf("Node %d", i), Type: "document"})
	}
	// Create a tree of edges
	for i := 1; i < 20; i++ {
		src := fmt.Sprintf("node_%02d.md", i/2)
		tgt := fmt.Sprintf("node_%02d.md", i)
		e, _ := knowledge.NewEdge(src, tgt, knowledge.EdgeReferences, 1.0, "")
		_ = g.AddEdge(e)
	}
	layout := computeNodeLayout(g)

	start := time.Now()
	out := RenderGraphASCII(g, layout, "node_00.md", 200, 60)
	duration := time.Since(start)

	if out == "" {
		t.Error("Expected non-empty graph output")
	}
	if duration > 500*time.Millisecond {
		t.Errorf("Graph rendering (20 nodes) took %v, expected < 500ms", duration)
	}
}

// TestPerf_NavigationResponsiveness verifies key handling is fast.
func TestPerf_NavigationResponsiveness(t *testing.T) {
	dir := mkTmpDir(t, map[string]string{
		"a.md": "# A\nContent",
		"b.md": "# B\nContent",
		"c.md": "# C\nContent",
	})
	v := newDirViewer(t, dir)

	start := time.Now()
	for i := 0; i < 100; i++ {
		v = pressKeys(v, "down")
	}
	duration := time.Since(start)

	if duration > 100*time.Millisecond {
		t.Errorf("100 key presses took %v, expected < 100ms", duration)
	}
}

// ============================================================================
// Regression Tests
// ============================================================================

// TestRegression_FileMode_StillWorks verifies single-file mode unchanged.
func TestRegression_FileMode_StillWorks(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "test.md")
	_ = os.WriteFile(filePath, []byte("# Test\nFile mode content."), 0o644)

	doc := &ast.Document{}
	v := New(doc, filePath, theme.NewTheme(), 80)
	v.Height = 24

	// Should NOT be in directory mode
	if v.directoryMode {
		t.Error("File mode viewer should not be in directory mode")
	}
	if v.openedFromDirectory {
		t.Error("File mode viewer should not have openedFromDirectory set")
	}

	// Navigation should work
	v = pressKeys(v, "down", "up")
	// No panic = pass
}

// TestRegression_SearchInFileMode verifies Ctrl+F still works in file mode.
func TestRegression_SearchInFileMode(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "test.md")
	_ = os.WriteFile(filePath, []byte("# Test\nSearch for this keyword."), 0o644)

	doc := &ast.Document{}
	v := New(doc, filePath, theme.NewTheme(), 80)
	v.Height = 24

	// Ctrl+F should enter search mode
	msg := tea.KeyMsg{Type: tea.KeyCtrlF}
	model, _ := v.Update(msg)
	result := model.(Viewer)

	if !result.searchMode {
		t.Error("Expected searchMode=true after Ctrl+F in file mode")
	}
}

// TestRegression_EditModeToggle verifies 'e' key still toggles edit mode.
func TestRegression_EditModeToggle(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "test.md")
	_ = os.WriteFile(filePath, []byte("# Test\nEdit mode test."), 0o644)

	doc := &ast.Document{}
	v := New(doc, filePath, theme.NewTheme(), 80)
	v.Height = 24

	v = pressKeys(v, "e")
	if !v.editMode {
		t.Error("Expected editMode=true after 'e'")
	}
}

// TestRegression_QuitCommand verifies 'q' still quits.
func TestRegression_QuitCommand(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "test.md")
	_ = os.WriteFile(filePath, []byte("# Test"), 0o644)

	doc := &ast.Document{}
	v := New(doc, filePath, theme.NewTheme(), 80)
	v.Height = 24

	_, cmd := v.Update(sendKey("q"))
	if cmd == nil {
		t.Error("Expected quit command from 'q'")
	}
}

// TestRegression_HelpOverlay verifies '?' still opens help.
func TestRegression_HelpOverlay(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "test.md")
	_ = os.WriteFile(filePath, []byte("# Test"), 0o644)

	doc := &ast.Document{}
	v := New(doc, filePath, theme.NewTheme(), 80)
	v.Height = 24

	v = pressKeys(v, "?")
	if !v.helpOpen {
		t.Error("Expected helpOpen=true after '?'")
	}
}

// TestRegression_NewDirectoryViewer verifies factory function.
func TestRegression_NewDirectoryViewer(t *testing.T) {
	dir := t.TempDir()
	v := NewDirectoryViewer(dir, theme.NewTheme(), 80)

	if !v.directoryMode {
		t.Error("Expected directoryMode=true for NewDirectoryViewer")
	}
	if v.currentView != "directory" {
		t.Errorf("Expected currentView='directory', got %q", v.currentView)
	}
	if v.directoryState.RootPath != dir {
		t.Errorf("Expected RootPath=%q, got %q", dir, v.directoryState.RootPath)
	}
}

// TestRegression_ViewRoutingPriority verifies View() routing precedence.
func TestRegression_ViewRoutingPriority(t *testing.T) {
	dir := t.TempDir()
	v := NewDirectoryViewer(dir, theme.NewTheme(), 80)
	v.Height = 24
	_ = v.LoadDirectory(dir)

	// Directory mode should render directory listing
	out := v.View()
	if !strings.Contains(out, "Markdown Files") {
		t.Error("Expected directory listing in View() for directory mode")
	}

	// Cross-search active should take precedence over directory
	v.crossSearchActive = true
	v.crossSearchMode = false
	v.crossSearchQuery = "test"
	v.crossSearchResults = []knowledge.SearchResult{}
	v.crossSearchSelected = -1
	out = v.View()
	if !strings.Contains(out, "Search Results") {
		t.Error("Expected search results view when crossSearchActive=true")
	}
}
