package knowledge

import (
	"os"
	"path/filepath"
	"testing"
)

// createTestTree creates a temporary directory tree with the given files.
// Each key is a relative path; the value is the file content.
func createTestTree(t *testing.T, files map[string]string) string {
	t.Helper()
	root := t.TempDir()
	for rel, content := range files {
		abs := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", filepath.Dir(abs), err)
		}
		if err := os.WriteFile(abs, []byte(content), 0o644); err != nil {
			t.Fatalf("write %s: %v", abs, err)
		}
	}
	return root
}

func TestScanDirectory_BasicMarkdownFiles(t *testing.T) {
	root := createTestTree(t, map[string]string{
		"a.md":          "# A\nContent A",
		"sub/b.md":      "# B\nContent B",
		"sub/deep/c.md": "# C\nContent C",
		"not-md.txt":    "not markdown",
	})

	docs, err := ScanDirectory(root, ScanConfig{UseDefaultIgnores: true})
	if err != nil {
		t.Fatalf("ScanDirectory: %v", err)
	}
	if len(docs) != 3 {
		t.Fatalf("expected 3 docs, got %d", len(docs))
	}

	// Check sorted order (forward slashes).
	expected := []string{"a.md", "sub/b.md", "sub/deep/c.md"}
	for i, doc := range docs {
		if doc.RelPath != expected[i] {
			t.Errorf("docs[%d].RelPath = %q, want %q", i, doc.RelPath, expected[i])
		}
	}
}

func TestScanDirectory_SkipsHiddenDirectories(t *testing.T) {
	root := createTestTree(t, map[string]string{
		"visible.md":       "# Visible",
		".git/HEAD":        "ref: refs/heads/main",
		".hidden/note.md":  "# Hidden",
		".dotfile.md":      "# Dot file", // hidden file — should still be scanned
	})

	docs, err := ScanDirectory(root, ScanConfig{UseDefaultIgnores: true})
	if err != nil {
		t.Fatalf("ScanDirectory: %v", err)
	}

	// .hidden/note.md should NOT appear; .dotfile.md is a file not a dir, depends on OS.
	// The main requirement: hidden DIRECTORIES are skipped.
	for _, d := range docs {
		if d.RelPath == ".hidden/note.md" {
			t.Errorf("hidden dir file should be skipped, but found %q", d.RelPath)
		}
	}

	// visible.md must be present.
	found := false
	for _, d := range docs {
		if d.RelPath == "visible.md" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected visible.md in results")
	}
}

func TestScanDirectory_SkipsKnownVendorDirs(t *testing.T) {
	root := createTestTree(t, map[string]string{
		"readme.md":              "# Readme",
		"node_modules/pkg.md":   "# Pkg",
	})

	docs, err := ScanDirectory(root, ScanConfig{UseDefaultIgnores: true})
	if err != nil {
		t.Fatalf("ScanDirectory: %v", err)
	}

	for _, d := range docs {
		if d.RelPath == "node_modules/pkg.md" {
			t.Errorf("node_modules doc should be skipped")
		}
	}
}

func TestScanDirectory_EmptyDirectory(t *testing.T) {
	root := t.TempDir()
	docs, err := ScanDirectory(root, ScanConfig{UseDefaultIgnores: true})
	if err != nil {
		t.Fatalf("ScanDirectory on empty dir: %v", err)
	}
	if len(docs) != 0 {
		t.Errorf("expected 0 docs, got %d", len(docs))
	}
}

func TestScanDirectory_DeepNesting(t *testing.T) {
	// 6 levels deep.
	root := createTestTree(t, map[string]string{
		"l1/l2/l3/l4/l5/l6/deep.md": "# Deep",
	})
	docs, err := ScanDirectory(root, ScanConfig{UseDefaultIgnores: true})
	if err != nil {
		t.Fatalf("ScanDirectory: %v", err)
	}
	if len(docs) != 1 {
		t.Fatalf("expected 1 doc, got %d", len(docs))
	}
	if docs[0].RelPath != "l1/l2/l3/l4/l5/l6/deep.md" {
		t.Errorf("unexpected RelPath: %q", docs[0].RelPath)
	}
}

func TestScanDirectory_NonExistentRoot(t *testing.T) {
	_, err := ScanDirectory("/nonexistent/path/that/does/not/exist", ScanConfig{UseDefaultIgnores: true})
	if err == nil {
		t.Fatal("expected error for nonexistent root")
	}
}

func TestScanDirectory_RootIsFile(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "file.md")
	if err := os.WriteFile(f, []byte("content"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := ScanDirectory(f, ScanConfig{UseDefaultIgnores: true})
	if err == nil {
		t.Fatal("expected error when root is a file")
	}
}

func TestScanDirectory_SortedByRelPath(t *testing.T) {
	root := createTestTree(t, map[string]string{
		"z.md":   "# Z",
		"a.md":   "# A",
		"m.md":   "# M",
		"b/a.md": "# BA",
		"a/z.md": "# AZ",
	})

	docs, err := ScanDirectory(root, ScanConfig{UseDefaultIgnores: true})
	if err != nil {
		t.Fatalf("ScanDirectory: %v", err)
	}

	for i := 1; i < len(docs); i++ {
		if docs[i-1].RelPath > docs[i].RelPath {
			t.Errorf("not sorted: docs[%d].RelPath=%q > docs[%d].RelPath=%q",
				i-1, docs[i-1].RelPath, i, docs[i].RelPath)
		}
	}
}

func TestScanDirectory_PerformanceBaseline(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping performance test in short mode")
	}
	// Create 200 markdown files in a nested structure and verify the scan
	// completes without error (timing assertions are too platform-specific
	// for unit tests but the benchmark covers performance).
	root := t.TempDir()
	for i := range 200 {
		subdir := filepath.Join(root, "dir", "sub")
		if err := os.MkdirAll(subdir, 0o755); err != nil {
			t.Fatal(err)
		}
		path := filepath.Join(subdir, filepath.FromSlash(
			"file"+string(rune('a'+i%26))+".md"),
		)
		content := "# Doc\n\nSome content for document " + string(rune(i))
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	docs, err := ScanDirectory(root, ScanConfig{UseDefaultIgnores: true})
	if err != nil {
		t.Fatalf("ScanDirectory: %v", err)
	}
	if len(docs) == 0 {
		t.Error("expected docs from performance test tree")
	}
}
