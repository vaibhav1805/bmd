package knowledge

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestRunPageIndex_ExecutableNotFound(t *testing.T) {
	cfg := PageIndexConfig{
		ExecutablePath: "/nonexistent/pageindex",
		Model:          "claude-sonnet-4-5",
	}

	_, err := RunPageIndex(cfg, "/some/file.md")
	if err == nil {
		t.Fatal("expected error for nonexistent executable, got nil")
	}

	if !errors.Is(err, ErrPageIndexNotFound) {
		t.Errorf("expected errors.Is(err, ErrPageIndexNotFound) to be true, got: %v", err)
	}
}

func TestDefaultPageIndexConfig(t *testing.T) {
	cfg := DefaultPageIndexConfig()

	if cfg.ExecutablePath != "pageindex" {
		t.Errorf("ExecutablePath: got %q, want %q", cfg.ExecutablePath, "pageindex")
	}
	if cfg.Model != "claude-sonnet-4-5" {
		t.Errorf("Model: got %q, want %q", cfg.Model, "claude-sonnet-4-5")
	}
}

func TestRunPageIndex_BadJSON(t *testing.T) {
	// Create a temporary shell script that exits 0 but prints garbage JSON.
	dir := t.TempDir()
	scriptPath := filepath.Join(dir, "fake-pageindex")
	script := "#!/bin/sh\necho 'not-json'\n"
	if err := os.WriteFile(scriptPath, []byte(script), 0755); err != nil {
		t.Fatalf("write fake script: %v", err)
	}

	cfg := PageIndexConfig{
		ExecutablePath: scriptPath,
		Model:          "claude-sonnet-4-5",
	}

	// Should return an error from JSON parsing, not panic.
	_, err := RunPageIndex(cfg, "/some/file.md")
	if err == nil {
		t.Fatal("expected error when subprocess prints bad JSON, got nil")
	}

	// Must not be ErrPageIndexNotFound (the binary was found, it just printed garbage).
	if errors.Is(err, ErrPageIndexNotFound) {
		t.Error("error should NOT be ErrPageIndexNotFound when binary exists but prints bad JSON")
	}
}

// TestSearchAllDocumentsPageIndex_NoTreeFiles verifies ErrPageIndexNotAvailable
// is returned when no .bmd-tree.json files exist in the directory.
func TestSearchAllDocumentsPageIndex_NoTreeFiles(t *testing.T) {
	dir := t.TempDir()

	_, err := SearchAllDocumentsPageIndex(dir, "test query", 10)
	if err == nil {
		t.Fatal("expected error when no tree files, got nil")
	}
	if !errors.Is(err, ErrPageIndexNotAvailable) {
		t.Errorf("expected ErrPageIndexNotAvailable, got: %v", err)
	}
}

// TestSearchAllDocumentsPageIndex_EmptyQuery returns empty without error.
func TestSearchAllDocumentsPageIndex_EmptyQuery(t *testing.T) {
	dir := t.TempDir()

	results, err := SearchAllDocumentsPageIndex(dir, "", 10)
	if err != nil {
		t.Fatalf("expected no error on empty query, got: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("expected 0 results for empty query, got %d", len(results))
	}
}

// TestSearchAllDocumentsPageIndex_PageIndexBinaryMissing verifies ErrPageIndexNotFound
// is returned when trees exist but the pageindex binary is missing.
func TestSearchAllDocumentsPageIndex_PageIndexBinaryMissing(t *testing.T) {
	dir := t.TempDir()

	// Write a minimal tree file so LoadTreeFiles returns something.
	ft := FileTree{
		File: "doc.md",
		Root: &TreeNode{Heading: "Doc", Summary: "A document"},
	}
	if err := SaveTreeFile(dir, ft); err != nil {
		t.Fatalf("SaveTreeFile: %v", err)
	}

	// Temporarily override PATH to ensure pageindex cannot be found.
	origPath := os.Getenv("PATH")
	t.Setenv("PATH", "")
	defer func() { os.Setenv("PATH", origPath) }()

	_, err := SearchAllDocumentsPageIndex(dir, "query", 10)
	if err == nil {
		t.Fatal("expected error when pageindex binary missing, got nil")
	}
	if !errors.Is(err, ErrPageIndexNotFound) {
		t.Errorf("expected ErrPageIndexNotFound (wrapped), got: %v", err)
	}
}

// TestSearchAllDocumentsPageIndex_FakeSubprocess verifies result conversion
// using a fake pageindex script that returns JSON.
func TestSearchAllDocumentsPageIndex_FakeSubprocess(t *testing.T) {
	dir := t.TempDir()

	// Write a tree file.
	ft := FileTree{
		File: "api.md",
		Root: &TreeNode{Heading: "API", Summary: "API documentation"},
	}
	if err := SaveTreeFile(dir, ft); err != nil {
		t.Fatalf("SaveTreeFile: %v", err)
	}

	// Build fake pageindex that prints a valid JSON array.
	fakeResults := []map[string]interface{}{
		{
			"file":            "api.md",
			"heading_path":    "API > Endpoints",
			"content":         "List of REST endpoints",
			"score":           0.95,
			"reasoning_trace": "High match",
		},
	}
	fakeJSON, _ := json.Marshal(fakeResults)
	scriptPath := filepath.Join(dir, "fake-pageindex")
	script := "#!/bin/sh\necho '" + string(fakeJSON) + "'\n"
	if err := os.WriteFile(scriptPath, []byte(script), 0o755); err != nil {
		t.Fatalf("write fake script: %v", err)
	}

	// Temporarily override PATH so our fake script is found as "pageindex".
	t.Setenv("PATH", dir+":"+os.Getenv("PATH"))

	// Rename the fake script to "pageindex".
	piPath := filepath.Join(dir, "pageindex")
	if err := os.Rename(scriptPath, piPath); err != nil {
		t.Fatalf("rename fake script: %v", err)
	}

	results, err := SearchAllDocumentsPageIndex(dir, "endpoints", 10)
	if err != nil {
		t.Fatalf("SearchAllDocumentsPageIndex error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].HeadingPath != "API > Endpoints" {
		t.Errorf("HeadingPath: got %q, want %q", results[0].HeadingPath, "API > Endpoints")
	}
	if results[0].Score != 0.95 {
		t.Errorf("Score: got %.2f, want 0.95", results[0].Score)
	}
}
