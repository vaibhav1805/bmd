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
	resolved, err := nav.ResolveLink(currentFile, href, dir)
	if err != nil {
		t.Fatalf("ResolveLink external link: expected no error, got %v", err)
	}
	want := "external://" + href
	if resolved != want {
		t.Errorf("ResolveLink external link: got %q, want %q", resolved, want)
	}
}

func TestResolveLink_ExternalLinkHTTPS(t *testing.T) {
	dir, cleanup := setupTestDir(t)
	defer cleanup()

	currentFile := filepath.Join(dir, "index.md")
	href := "https://example.com/file.md"
	resolved, err := nav.ResolveLink(currentFile, href, dir)
	if err != nil {
		t.Fatalf("ResolveLink external link: expected no error, got %v", err)
	}
	want := "external://" + href
	if resolved != want {
		t.Errorf("ResolveLink external link: got %q, want %q", resolved, want)
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

// --- IsWithinDir --------------------------------------------------------

// TestIsWithinDir_ExactMatch treats dir itself as "within" dir.
func TestIsWithinDir_ExactMatch(t *testing.T) {
	if !nav.IsWithinDir("/repo", "/repo") {
		t.Error("expected dir to be considered within itself")
	}
}

// TestIsWithinDir_Nested is the common case: a file somewhere under dir.
func TestIsWithinDir_Nested(t *testing.T) {
	if !nav.IsWithinDir("/repo/docs/getting-started.md", "/repo") {
		t.Error("expected a file nested under dir to be within it")
	}
	if !nav.IsWithinDir(filepath.Join("/repo", "a", "b", "c.md"), "/repo") {
		t.Error("expected a deeply nested file to be within dir")
	}
}

// TestIsWithinDir_OutsideRejected: a file from a completely unrelated
// directory must not be treated as "within" the current one -- this is
// what guards `bmd`'s bare-invocation session restore (cmd/bmd/main.go)
// against silently reopening a file left open in a different project.
func TestIsWithinDir_OutsideRejected(t *testing.T) {
	if nav.IsWithinDir("/other-project/file.md", "/repo") {
		t.Error("expected a file outside dir to be rejected")
	}
}

// TestIsWithinDir_SiblingWithSharedPrefixRejected guards against a naive
// strings.HasPrefix(path, dir) implementation, which would incorrectly
// accept "/repo-other/file.md" as being within "/repo" (string prefix
// match without a path-separator boundary check).
func TestIsWithinDir_SiblingWithSharedPrefixRejected(t *testing.T) {
	if nav.IsWithinDir("/repo-other/file.md", "/repo") {
		t.Error("expected a sibling directory with a shared string prefix to be rejected")
	}
}

// TestIsWithinDir_ParentRejected: dir's own parent directory (or anything
// above dir) is not "within" dir, even though dir is within it.
func TestIsWithinDir_ParentRejected(t *testing.T) {
	if nav.IsWithinDir("/", "/repo") {
		t.Error("expected dir's parent to be rejected")
	}
	if nav.IsWithinDir("/repo/../other/file.md", "/repo") {
		t.Error("expected a path that escapes dir via .. to be rejected")
	}
}
