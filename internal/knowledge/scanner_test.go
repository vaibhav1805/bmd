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

// ─── Tests for directory/file filtering (Phase 21b) ───────────────────────────────

func TestScanConfig_SkipsHiddenDirsByDefault(t *testing.T) {
	root := createTestTree(t, map[string]string{
		"visible/readme.md":    "# Visible",
		".git/config.md":       "# Git config",
		".venv/lib/note.md":    "# Venv",
		".cache/data.md":       "# Cache",
		".hidden/secret.md":    "# Hidden dir",
		"public.md":            "# Public",
	})

	config := ScanConfig{
		IncludeHidden:     false,
		UseDefaultIgnores: true,
	}

	docs, err := ScanDirectory(root, config)
	if err != nil {
		t.Fatalf("ScanDirectory: %v", err)
	}

	// Check that hidden dirs are excluded
	hiddenPaths := map[string]bool{
		".git/config.md":    false,
		".venv/lib/note.md": false,
		".cache/data.md":    false,
		".hidden/secret.md": false,
	}

	for _, doc := range docs {
		if _, exists := hiddenPaths[doc.RelPath]; exists {
			t.Errorf("hidden directory file should be skipped: %q", doc.RelPath)
		}
	}

	// Verify visible files ARE included
	visibleCount := 0
	for _, doc := range docs {
		if doc.RelPath == "public.md" || doc.RelPath == "visible/readme.md" {
			visibleCount++
		}
	}
	if visibleCount != 2 {
		t.Errorf("expected 2 visible files, found %d", visibleCount)
	}
}

func TestScanConfig_IncludesHiddenDirsWithFlag(t *testing.T) {
	root := createTestTree(t, map[string]string{
		"visible/readme.md": "# Visible",
		".git/config.md":    "# Git config (hardcoded, always skipped)",
		".custom/note.md":   "# Custom hidden dir",
		"public.md":         "# Public",
	})

	config := ScanConfig{
		IncludeHidden:     true, // -A flag
		UseDefaultIgnores: true,
	}

	docs, err := ScanDirectory(root, config)
	if err != nil {
		t.Fatalf("ScanDirectory: %v", err)
	}

	// With IncludeHidden, .custom (user-created hidden dir) should be scanned
	// But .git is hardcoded and always skipped
	customFound := false
	gitFound := false

	for _, doc := range docs {
		if doc.RelPath == ".custom/note.md" {
			customFound = true
		}
		if doc.RelPath == ".git/config.md" {
			gitFound = true
		}
	}

	if !customFound {
		t.Error("expected .custom/note.md when IncludeHidden is true")
	}
	if gitFound {
		t.Error(".git/ should always be skipped (hardcoded, not user-created)")
	}
}

func TestScanConfig_IgnoresDefaultPatterns(t *testing.T) {
	root := createTestTree(t, map[string]string{
		"readme.md":              "# Readme",
		"vendor/pkg.md":          "# Vendor package",
		"node_modules/dep.md":    "# Node dep",
		"__pycache__/cache.md":   "# Python cache",
		".gradle/build.md":       "# Gradle",
		"build/artifact.md":      "# Build artifact",
		"dist/output.md":         "# Distribution",
		"src/main.md":            "# Source",
	})

	config := ScanConfig{
		IncludeHidden:     false,
		UseDefaultIgnores: true,
	}

	docs, err := ScanDirectory(root, config)
	if err != nil {
		t.Fatalf("ScanDirectory: %v", err)
	}

	// Check that default ignore patterns exclude vendor, node_modules, etc.
	defaultIgnored := map[string]bool{
		"vendor/pkg.md":        false,
		"node_modules/dep.md":  false,
		"__pycache__/cache.md": false,
		".gradle/build.md":     false,
		"build/artifact.md":    false,
		"dist/output.md":       false,
	}

	for _, doc := range docs {
		if _, shouldIgnore := defaultIgnored[doc.RelPath]; shouldIgnore {
			t.Errorf("default ignored pattern should be skipped: %q", doc.RelPath)
		}
	}

	// Verify non-ignored files ARE included
	srcFound := false
	for _, doc := range docs {
		if doc.RelPath == "src/main.md" {
			srcFound = true
		}
	}
	if !srcFound {
		t.Error("expected src/main.md to be included")
	}
}

func TestScanConfig_CustomIgnoreDirs(t *testing.T) {
	root := createTestTree(t, map[string]string{
		"src/code.md":       "# Source",
		"custom/note.md":    "# Custom",
		"local/data.md":     "# Local",
		"vendor/lib.md":     "# Vendor",
		"test/spec.md":      "# Test",
	})

	config := ScanConfig{
		IgnoreDirs:        []string{"custom", "local"},
		IncludeHidden:     false,
		UseDefaultIgnores: false, // Disable defaults to test custom-only
	}

	docs, err := ScanDirectory(root, config)
	if err != nil {
		t.Fatalf("ScanDirectory: %v", err)
	}

	// custom/ and local/ should be skipped
	customFound := false
	localFound := false
	vendorFound := false // vendor should NOT be skipped since defaults are disabled

	for _, doc := range docs {
		if doc.RelPath == "custom/note.md" {
			customFound = true
		}
		if doc.RelPath == "local/data.md" {
			localFound = true
		}
		if doc.RelPath == "vendor/lib.md" {
			vendorFound = true
		}
	}

	if customFound {
		t.Error("custom/ should be ignored by custom pattern")
	}
	if localFound {
		t.Error("local/ should be ignored by custom pattern")
	}
	if !vendorFound {
		t.Error("vendor/ should NOT be ignored when defaults are disabled")
	}
}

func TestScanConfig_IgnoreFilesPatterns(t *testing.T) {
	root := createTestTree(t, map[string]string{
		"readme.md":        "# Readme",
		"draft.md":         "# Draft",
		"DRAFT.md":         "# Draft uppercase",
		"config.backup":    "backup",
		"settings.lock":    "lock",
		"script.py":        "python",
		"src/main.md":      "# Source",
		"src/test.backup":  "test backup",
	})

	config := ScanConfig{
		IgnoreFiles:       []string{"*.backup", "*.lock", "DRAFT.md"},
		IncludeHidden:     false,
		UseDefaultIgnores: true,
	}

	docs, err := ScanDirectory(root, config)
	if err != nil {
		t.Fatalf("ScanDirectory: %v", err)
	}

	// Files matching patterns should be excluded
	for _, doc := range docs {
		if doc.RelPath == "config.backup" {
			t.Errorf("*.backup should be ignored: %q", doc.RelPath)
		}
		if doc.RelPath == "settings.lock" {
			t.Errorf("*.lock should be ignored: %q", doc.RelPath)
		}
		if doc.RelPath == "DRAFT.md" {
			t.Errorf("DRAFT.md should be ignored: %q", doc.RelPath)
		}
		if doc.RelPath == "src/test.backup" {
			t.Errorf("*.backup in subdirs should be ignored: %q", doc.RelPath)
		}
	}

	// Verify included files
	readmeFound := false
	draftFound := false
	srcMainFound := false

	for _, doc := range docs {
		if doc.RelPath == "readme.md" {
			readmeFound = true
		}
		if doc.RelPath == "draft.md" {
			draftFound = true
		}
		if doc.RelPath == "src/main.md" {
			srcMainFound = true
		}
	}

	if !readmeFound {
		t.Error("readme.md should be included")
	}
	if !draftFound {
		t.Error("draft.md (lowercase) should be included, only DRAFT.md is ignored")
	}
	if !srcMainFound {
		t.Error("src/main.md should be included")
	}
}

func TestScanConfig_CombinedCustomAndDefaults(t *testing.T) {
	root := createTestTree(t, map[string]string{
		"readme.md":         "# Readme",
		"vendor/pkg.md":     "# Vendor (default ignored)",
		"custom/note.md":    "# Custom (custom ignored)",
		"src/main.md":       "# Source",
		"test.backup":       "# Test (file ignored)",
	})

	config := ScanConfig{
		IgnoreDirs:        []string{"custom"},
		IgnoreFiles:       []string{"*.backup"},
		IncludeHidden:     false,
		UseDefaultIgnores: true, // Use both defaults AND custom
	}

	docs, err := ScanDirectory(root, config)
	if err != nil {
		t.Fatalf("ScanDirectory: %v", err)
	}

	// Verify vendor/ (default) and custom/ (custom) are both ignored
	vendorFound := false
	customFound := false
	testBackupFound := false

	for _, doc := range docs {
		if doc.RelPath == "vendor/pkg.md" {
			vendorFound = true
		}
		if doc.RelPath == "custom/note.md" {
			customFound = true
		}
		if doc.RelPath == "test.backup" {
			testBackupFound = true
		}
	}

	if vendorFound {
		t.Error("vendor/ should be ignored (default pattern)")
	}
	if customFound {
		t.Error("custom/ should be ignored (custom pattern)")
	}
	if testBackupFound {
		t.Error("*.backup should be ignored (custom file pattern)")
	}

	// Verify other files are included
	readmeFound := false
	srcMainFound := false
	for _, doc := range docs {
		if doc.RelPath == "readme.md" {
			readmeFound = true
		}
		if doc.RelPath == "src/main.md" {
			srcMainFound = true
		}
	}

	if !readmeFound || !srcMainFound {
		t.Error("readme.md and src/main.md should be included")
	}
}

func TestScanConfig_NoIgnoreDefaults(t *testing.T) {
	root := createTestTree(t, map[string]string{
		"readme.md":         "# Readme",
		"vendor/pkg.md":     "# Vendor",
		"node_modules/d.md": "# Node (hardcoded, always skipped)",
		"build/out.md":      "# Build",
		"src/main.md":       "# Source",
	})

	config := ScanConfig{
		IncludeHidden:     false,
		UseDefaultIgnores: false, // Disable default ignores
	}

	docs, err := ScanDirectory(root, config)
	if err != nil {
		t.Fatalf("ScanDirectory: %v", err)
	}

	// With no default ignores, vendor/ and build/ should be included
	// node_modules/ is hardcoded and always skipped
	vendorFound := false
	nodeModulesFound := false
	buildFound := false

	for _, doc := range docs {
		if doc.RelPath == "vendor/pkg.md" {
			vendorFound = true
		}
		if doc.RelPath == "node_modules/d.md" {
			nodeModulesFound = true
		}
		if doc.RelPath == "build/out.md" {
			buildFound = true
		}
	}

	if !vendorFound {
		t.Error("vendor/ should be included when defaults are disabled")
	}
	if nodeModulesFound {
		t.Error("node_modules/ should NOT be included (hardcoded, always skipped)")
	}
	if !buildFound {
		t.Error("build/ should be included when defaults are disabled")
	}
}

// ─── Tests for matchPattern helper ─────────────────────────────────────────────

func TestMatchPattern_ExactMatch(t *testing.T) {
	tests := []struct {
		name    string
		pattern string
		name_   string
		want    bool
	}{
		{"exact vendor", "vendor", "vendor", true},
		{"exact build", "build", "build", true},
		{"exact mismatch", "vendor", "vendor2", false},
		{"case sensitive", "Vendor", "vendor", false},
		{"empty pattern", "", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := matchPattern(tt.name_, tt.pattern)
			if result != tt.want {
				t.Errorf("matchPattern(%q, %q) = %v, want %v", tt.name_, tt.pattern, result, tt.want)
			}
		})
	}
}

func TestMatchPattern_SuffixWildcard(t *testing.T) {
	tests := []struct {
		name    string
		pattern string
		name_   string
		want    bool
	}{
		{"*.lock matches package.lock", "*.lock", "package.lock", true},
		{"*.backup matches file.backup", "*.backup", "file.backup", true},
		{"*.md matches readme.md", "*.md", "readme.md", true},
		{"*.lock does not match package.txt", "*.lock", "package.txt", false},
		{"*.lock does not match lock", "*.lock", "lock", false},
		{"universal wildcard", "*", "anything", true},
		{"universal wildcard empty", "*", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := matchPattern(tt.name_, tt.pattern)
			if result != tt.want {
				t.Errorf("matchPattern(%q, %q) = %v, want %v", tt.name_, tt.pattern, result, tt.want)
			}
		})
	}
}

func TestMatchPattern_PrefixWildcard(t *testing.T) {
	tests := []struct {
		name    string
		pattern string
		name_   string
		want    bool
	}{
		{"node_* matches node_modules", "node_*", "node_modules", true},
		{"node_* matches node_stuff", "node_*", "node_stuff", true},
		{"node_* does not match nodejs", "node_*", "nodejs", false},
		{"py* matches python", "py*", "python", true},
		{"py* does not matchepy", "py*", "epy", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := matchPattern(tt.name_, tt.pattern)
			if result != tt.want {
				t.Errorf("matchPattern(%q, %q) = %v, want %v", tt.name_, tt.pattern, result, tt.want)
			}
		})
	}
}

func TestMatchPattern_MultipleWildcards(t *testing.T) {
	// Patterns with multiple wildcards are not currently supported;
	// they should only match exactly as-is
	tests := []struct {
		name    string
		pattern string
		name_   string
		want    bool
	}{
		{"*test* exact match", "*test*", "*test*", true},
		{"*test* no prefix/suffix wildcard support", "*test*", "atest", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := matchPattern(tt.name_, tt.pattern)
			if result != tt.want {
				t.Errorf("matchPattern(%q, %q) = %v, want %v", tt.name_, tt.pattern, result, tt.want)
			}
		})
	}
}
