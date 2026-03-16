package nav_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bmd/bmd/internal/nav"
)

// setupTestDir creates a temporary directory tree with real files for testing.
// Returns the temp dir path and a cleanup function.
func setupTestDir(t *testing.T) (string, func()) {
	t.Helper()
	dir, err := os.MkdirTemp("", "navtest-*")
	if err != nil {
		t.Fatalf("MkdirTemp: %v", err)
	}

	// Create directory structure:
	// dir/
	//   index.md
	//   api.md
	//   guide/
	//     intro.md
	//   sub/
	//     page.md
	//   README.md
	//   page.html (non-markdown, for rejection test)

	dirs := []string{
		filepath.Join(dir, "guide"),
		filepath.Join(dir, "sub"),
	}
	for _, d := range dirs {
		if err := os.MkdirAll(d, 0755); err != nil {
			t.Fatalf("MkdirAll %s: %v", d, err)
		}
	}

	files := []string{
		filepath.Join(dir, "index.md"),
		filepath.Join(dir, "api.md"),
		filepath.Join(dir, "guide", "intro.md"),
		filepath.Join(dir, "sub", "page.md"),
		filepath.Join(dir, "README.md"),
		filepath.Join(dir, "page.html"),
	}
	for _, f := range files {
		if err := os.WriteFile(f, []byte("# Test"), 0644); err != nil {
			t.Fatalf("WriteFile %s: %v", f, err)
		}
	}

	return dir, func() { os.RemoveAll(dir) }
}

func TestResolveLink_SimpleRelative(t *testing.T) {
	dir, cleanup := setupTestDir(t)
	defer cleanup()

	currentFile := filepath.Join(dir, "index.md")
	href := "./api.md"
	got, err := nav.ResolveLink(currentFile, href, dir)
	if err != nil {
		t.Fatalf("ResolveLink(%q, %q): unexpected error: %v", currentFile, href, err)
	}
	want := filepath.Join(dir, "api.md")
	if got != want {
		t.Errorf("ResolveLink: got %q, want %q", got, want)
	}
}

func TestResolveLink_RelativeParentDir(t *testing.T) {
	dir, cleanup := setupTestDir(t)
	defer cleanup()

	currentFile := filepath.Join(dir, "guide", "intro.md")
	href := "../README.md"
	got, err := nav.ResolveLink(currentFile, href, dir)
	if err != nil {
		t.Fatalf("ResolveLink(%q, %q): unexpected error: %v", currentFile, href, err)
	}
	want := filepath.Join(dir, "README.md")
	if got != want {
		t.Errorf("ResolveLink: got %q, want %q", got, want)
	}
}

func TestResolveLink_SubDirectory(t *testing.T) {
	dir, cleanup := setupTestDir(t)
	defer cleanup()

	currentFile := filepath.Join(dir, "index.md")
	href := "./sub/page.md"
	got, err := nav.ResolveLink(currentFile, href, dir)
	if err != nil {
		t.Fatalf("ResolveLink(%q, %q): unexpected error: %v", currentFile, href, err)
	}
	want := filepath.Join(dir, "sub", "page.md")
	if got != want {
		t.Errorf("ResolveLink: got %q, want %q", got, want)
	}
}

func TestResolveLink_TraversalAboveStartDir(t *testing.T) {
	dir, cleanup := setupTestDir(t)
	defer cleanup()

	currentFile := filepath.Join(dir, "index.md")
	href := "../../../etc/passwd.md"
	_, err := nav.ResolveLink(currentFile, href, dir)
	if err == nil {
		t.Fatal("ResolveLink with traversal: expected error, got nil")
	}
	if !strings.Contains(err.Error(), "traversal") {
		t.Errorf("ResolveLink traversal error: got %q, want message containing 'traversal'", err.Error())
	}
}

func TestResolveLink_ExternalLink(t *testing.T) {
	dir, cleanup := setupTestDir(t)
	defer cleanup()

	currentFile := filepath.Join(dir, "index.md")
	href := "http://example.com/file.md"
	_, err := nav.ResolveLink(currentFile, href, dir)
	if err == nil {
		t.Fatal("ResolveLink external link: expected error, got nil")
	}
	if !strings.Contains(err.Error(), "external") {
		t.Errorf("ResolveLink external error: got %q, want message containing 'external'", err.Error())
	}
}

func TestResolveLink_ExternalLinkHTTPS(t *testing.T) {
	dir, cleanup := setupTestDir(t)
	defer cleanup()

	currentFile := filepath.Join(dir, "index.md")
	href := "https://example.com/file.md"
	_, err := nav.ResolveLink(currentFile, href, dir)
	if err == nil {
		t.Fatal("ResolveLink https link: expected error, got nil")
	}
	if !strings.Contains(err.Error(), "external") {
		t.Errorf("ResolveLink https error: got %q, want message containing 'external'", err.Error())
	}
}

func TestResolveLink_AnchorLink(t *testing.T) {
	dir, cleanup := setupTestDir(t)
	defer cleanup()

	currentFile := filepath.Join(dir, "index.md")
	href := "#section"
	_, err := nav.ResolveLink(currentFile, href, dir)
	if err == nil {
		t.Fatal("ResolveLink anchor: expected error, got nil")
	}
	if !strings.Contains(err.Error(), "anchor") {
		t.Errorf("ResolveLink anchor error: got %q, want message containing 'anchor'", err.Error())
	}
}

func TestResolveLink_NonMarkdownFile(t *testing.T) {
	dir, cleanup := setupTestDir(t)
	defer cleanup()

	currentFile := filepath.Join(dir, "index.md")
	href := "./page.html"
	_, err := nav.ResolveLink(currentFile, href, dir)
	if err == nil {
		t.Fatal("ResolveLink non-.md file: expected error, got nil")
	}
	if !strings.Contains(err.Error(), ".md") {
		t.Errorf("ResolveLink non-md error: got %q, want message containing '.md'", err.Error())
	}
}

func TestResolveLink_FileNotFound(t *testing.T) {
	dir, cleanup := setupTestDir(t)
	defer cleanup()

	currentFile := filepath.Join(dir, "index.md")
	href := "./nonexistent.md"
	_, err := nav.ResolveLink(currentFile, href, dir)
	if err == nil {
		t.Fatal("ResolveLink missing file: expected error, got nil")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("ResolveLink not-found error: got %q, want message containing 'not found'", err.Error())
	}
}

func TestResolveLink_Symlink(t *testing.T) {
	dir, cleanup := setupTestDir(t)
	defer cleanup()

	// Create a symlink inside startDir pointing to a real .md file
	target := filepath.Join(dir, "api.md")
	link := filepath.Join(dir, "symlinked.md")
	if err := os.Symlink(target, link); err != nil {
		t.Skipf("os.Symlink not supported: %v", err)
	}

	currentFile := filepath.Join(dir, "index.md")
	href := "./symlinked.md"
	_, err := nav.ResolveLink(currentFile, href, dir)
	if err == nil {
		t.Fatal("ResolveLink symlink: expected error, got nil")
	}
	if !strings.Contains(err.Error(), "symlink") {
		t.Errorf("ResolveLink symlink error: got %q, want message containing 'symlink'", err.Error())
	}
}
