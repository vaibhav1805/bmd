package knowledge

import (
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
