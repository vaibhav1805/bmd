package knowledge

import (
	"archive/tar"
	"compress/gzip"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseExportArgs(t *testing.T) {
	tests := []struct {
		name       string
		args       []string
		wantFrom   string
		wantOutput string
		wantVer    string
		wantGit    bool
		wantPub    string
		wantErr    bool
	}{
		{
			name:       "defaults",
			args:       []string{},
			wantFrom:   ".",
			wantOutput: "knowledge.tar.gz",
		},
		{
			name:       "explicit flags",
			args:       []string{"--from", "/tmp/docs", "--output", "out.tar.gz"},
			wantFrom:   "/tmp/docs",
			wantOutput: "out.tar.gz",
		},
		{
			name:       "positional overrides from",
			args:       []string{"./my-docs"},
			wantFrom:   "./my-docs",
			wantOutput: "knowledge.tar.gz",
		},
		{
			name:       "version flag",
			args:       []string{"--version", "2.1.0"},
			wantFrom:   ".",
			wantOutput: "knowledge.tar.gz",
			wantVer:    "2.1.0",
		},
		{
			name:       "git-version flag",
			args:       []string{"--git-version"},
			wantFrom:   ".",
			wantOutput: "knowledge.tar.gz",
			wantGit:    true,
		},
		{
			name:       "publish flag",
			args:       []string{"--publish", "s3://my-bucket/knowledge"},
			wantFrom:   ".",
			wantOutput: "knowledge.tar.gz",
			wantPub:    "s3://my-bucket/knowledge",
		},
		{
			name:       "all flags combined",
			args:       []string{"--from", "./docs", "--output", "v2.tar.gz", "--version", "2.0.0", "--git-version", "--publish", "s3://b/k"},
			wantFrom:   "./docs",
			wantOutput: "v2.tar.gz",
			wantVer:    "2.0.0",
			wantGit:    true,
			wantPub:    "s3://b/k",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a, err := ParseExportArgs(tt.args)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ParseExportArgs() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}
			if a.From != tt.wantFrom {
				t.Errorf("From = %q, want %q", a.From, tt.wantFrom)
			}
			if a.Output != tt.wantOutput {
				t.Errorf("Output = %q, want %q", a.Output, tt.wantOutput)
			}
			if a.Version != tt.wantVer {
				t.Errorf("Version = %q, want %q", a.Version, tt.wantVer)
			}
			if a.GitVersion != tt.wantGit {
				t.Errorf("GitVersion = %v, want %v", a.GitVersion, tt.wantGit)
			}
			if a.Publish != tt.wantPub {
				t.Errorf("Publish = %q, want %q", a.Publish, tt.wantPub)
			}
		})
	}
}

func TestCmdExport(t *testing.T) {
	// Create a temp directory with some markdown files.
	tmpDir := t.TempDir()
	writeTestFile(t, filepath.Join(tmpDir, "readme.md"), "# Hello\n\nWorld")
	writeTestFile(t, filepath.Join(tmpDir, "guide.md"), "# Guide\n\nSome content here")
	writeTestFile(t, filepath.Join(tmpDir, "sub", "nested.md"), "# Nested\n\nDeep file")

	outputPath := filepath.Join(tmpDir, "output", "knowledge.tar.gz")

	// Run export.
	err := CmdExport([]string{"--from", tmpDir, "--output", outputPath})
	if err != nil {
		t.Fatalf("CmdExport() error: %v", err)
	}

	// Verify archive exists.
	if _, err := os.Stat(outputPath); os.IsNotExist(err) {
		t.Fatal("Output archive does not exist")
	}

	// Read archive contents.
	entries := listTarGzEntries(t, outputPath)

	// Should contain: knowledge.json, readme.md, guide.md, sub/nested.md, .bmd/knowledge.db
	expectedFiles := []string{"knowledge.json", "readme.md", "guide.md", "sub/nested.md", ".bmd/knowledge.db"}
	for _, expected := range expectedFiles {
		found := false
		for _, entry := range entries {
			if entry == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected file %q not found in archive. Archive contains: %v", expected, entries)
		}
	}
}

func TestExportMetadata(t *testing.T) {
	tmpDir := t.TempDir()
	writeTestFile(t, filepath.Join(tmpDir, "test.md"), "# Test\n\nContent")

	outputPath := filepath.Join(tmpDir, "meta-test.tar.gz")

	err := CmdExport([]string{"--from", tmpDir, "--output", outputPath})
	if err != nil {
		t.Fatalf("CmdExport() error: %v", err)
	}

	// Extract and parse knowledge.json from the archive.
	metaJSON := extractFileFromTarGz(t, outputPath, "knowledge.json")
	if metaJSON == nil {
		t.Fatal("knowledge.json not found in archive")
	}

	var meta KnowledgeMetadata
	if err := json.Unmarshal(metaJSON, &meta); err != nil {
		t.Fatalf("Failed to parse knowledge.json: %v", err)
	}

	if meta.Version != "1.0.0" {
		t.Errorf("Version = %q, want %q", meta.Version, "1.0.0")
	}
	if meta.FileCount != 1 {
		t.Errorf("FileCount = %d, want 1", meta.FileCount)
	}
	if meta.CreatedAt.IsZero() {
		t.Error("CreatedAt should not be zero")
	}
	if meta.DBSize <= 0 {
		t.Errorf("DBSize = %d, should be > 0", meta.DBSize)
	}
}

func TestExportNonexistentDir(t *testing.T) {
	err := CmdExport([]string{"--from", "/nonexistent/path", "--output", "/tmp/test.tar.gz"})
	if err == nil {
		t.Fatal("Expected error for nonexistent directory")
	}
	if !strings.Contains(err.Error(), "does not exist") {
		t.Errorf("Error should mention 'does not exist', got: %v", err)
	}
}

// ─── Phase 16: Versioning & Checksum Tests ───────────────────────────────────

func TestExportVersionedMetadata(t *testing.T) {
	tmpDir := t.TempDir()
	writeTestFile(t, filepath.Join(tmpDir, "readme.md"), "# Hello\n\nWorld")

	outputPath := filepath.Join(tmpDir, "versioned.tar.gz")

	err := CmdExport([]string{"--from", tmpDir, "--output", outputPath, "--version", "2.5.1"})
	if err != nil {
		t.Fatalf("CmdExport() error: %v", err)
	}

	metaJSON := extractFileFromTarGz(t, outputPath, "knowledge.json")
	if metaJSON == nil {
		t.Fatal("knowledge.json not found in archive")
	}

	var meta KnowledgeMetadata
	if err := json.Unmarshal(metaJSON, &meta); err != nil {
		t.Fatalf("Failed to parse knowledge.json: %v", err)
	}

	// Verify version is set from flag.
	if meta.Version != "2.5.1" {
		t.Errorf("Version = %q, want %q", meta.Version, "2.5.1")
	}

	// Verify checksum is present and has sha256 prefix.
	if meta.Checksum == "" {
		t.Error("Checksum should not be empty")
	}
	if !strings.HasPrefix(meta.Checksum, "sha256:") {
		t.Errorf("Checksum should start with 'sha256:', got %q", meta.Checksum)
	}

	// SHA256 hex digest should be 64 chars.
	hexPart := strings.TrimPrefix(meta.Checksum, "sha256:")
	if len(hexPart) != 64 {
		t.Errorf("Checksum hex length = %d, want 64", len(hexPart))
	}
}

func TestExportChecksumValidatesOnImport(t *testing.T) {
	// Create source directory with files.
	tmpDir := t.TempDir()
	writeTestFile(t, filepath.Join(tmpDir, "api.md"), "# API\n\nEndpoints here")
	writeTestFile(t, filepath.Join(tmpDir, "guide.md"), "# Guide\n\nStep 1: ...")

	outputPath := filepath.Join(tmpDir, "checksum-test.tar.gz")
	err := CmdExport([]string{"--from", tmpDir, "--output", outputPath, "--version", "1.0.0"})
	if err != nil {
		t.Fatalf("CmdExport() error: %v", err)
	}

	// Import should succeed (checksum is valid).
	destDir := t.TempDir()
	result, err := ImportKnowledgeTar(outputPath, destDir)
	if err != nil {
		t.Fatalf("ImportKnowledgeTar() error: %v", err)
	}

	if result.Metadata == nil {
		t.Fatal("Metadata should not be nil")
	}
	if result.Metadata.Version != "1.0.0" {
		t.Errorf("Version = %q, want %q", result.Metadata.Version, "1.0.0")
	}
	if result.Metadata.Checksum == "" {
		t.Error("Checksum should not be empty in metadata")
	}
}

func TestChecksumMismatchRejectsImport(t *testing.T) {
	// Create a tar.gz with an invalid checksum.
	tmpDir := t.TempDir()

	// Build a tar manually with a bad checksum in knowledge.json.
	archivePath := filepath.Join(tmpDir, "bad-checksum.tar.gz")
	outFile, err := os.Create(archivePath)
	if err != nil {
		t.Fatal(err)
	}
	gzw := gzip.NewWriter(outFile)
	tw := tar.NewWriter(gzw)

	// Write the markdown file.
	mdContent := []byte("# Test\n\nContent")
	if err := tw.WriteHeader(&tar.Header{
		Name: "test.md",
		Size: int64(len(mdContent)),
		Mode: 0o644,
	}); err != nil {
		t.Fatal(err)
	}
	tw.Write(mdContent)

	// Write knowledge.json with a deliberately wrong checksum.
	meta := KnowledgeMetadata{
		Version:   "1.0.0",
		Checksum:  "sha256:0000000000000000000000000000000000000000000000000000000000000000",
		FileCount: 1,
	}
	metaJSON, _ := json.MarshalIndent(meta, "", "  ")
	if err := tw.WriteHeader(&tar.Header{
		Name: "knowledge.json",
		Size: int64(len(metaJSON)),
		Mode: 0o644,
	}); err != nil {
		t.Fatal(err)
	}
	tw.Write(metaJSON)

	tw.Close()
	gzw.Close()
	outFile.Close()

	// Import should FAIL because checksum doesn't match.
	destDir := t.TempDir()
	_, err = ImportKnowledgeTar(archivePath, destDir)
	if err == nil {
		t.Fatal("Expected error for checksum mismatch, got nil")
	}
	if !strings.Contains(err.Error(), "checksum mismatch") {
		t.Errorf("Error should mention 'checksum mismatch', got: %v", err)
	}
}

func TestComputeDirectoryChecksum(t *testing.T) {
	// Create a temp directory with known content.
	tmpDir := t.TempDir()
	writeTestFile(t, filepath.Join(tmpDir, "a.md"), "alpha")
	writeTestFile(t, filepath.Join(tmpDir, "b.md"), "beta")
	// Add a knowledge.json that should be skipped.
	writeTestFile(t, filepath.Join(tmpDir, "knowledge.json"), `{"version":"1.0"}`)

	checksum1, err := ComputeDirectoryChecksum(tmpDir)
	if err != nil {
		t.Fatalf("ComputeDirectoryChecksum() error: %v", err)
	}

	// Same content should produce the same checksum.
	checksum2, err := ComputeDirectoryChecksum(tmpDir)
	if err != nil {
		t.Fatalf("ComputeDirectoryChecksum() second call error: %v", err)
	}
	if checksum1 != checksum2 {
		t.Errorf("Checksums should be deterministic: %s != %s", checksum1, checksum2)
	}

	// Different content should produce different checksum.
	writeTestFile(t, filepath.Join(tmpDir, "a.md"), "modified-alpha")
	checksum3, err := ComputeDirectoryChecksum(tmpDir)
	if err != nil {
		t.Fatalf("ComputeDirectoryChecksum() after modification error: %v", err)
	}
	if checksum1 == checksum3 {
		t.Error("Checksum should change when content changes")
	}
}

func TestValidateChecksumSuccess(t *testing.T) {
	tmpDir := t.TempDir()
	writeTestFile(t, filepath.Join(tmpDir, "doc.md"), "hello world")

	checksum, err := ComputeDirectoryChecksum(tmpDir)
	if err != nil {
		t.Fatal(err)
	}

	meta := KnowledgeMetadata{Checksum: "sha256:" + checksum}
	if err := ValidateChecksum(tmpDir, meta); err != nil {
		t.Errorf("ValidateChecksum should pass for valid checksum: %v", err)
	}
}

func TestValidateChecksumEmptySkips(t *testing.T) {
	meta := KnowledgeMetadata{Checksum: ""}
	if err := ValidateChecksum("/nonexistent", meta); err != nil {
		t.Errorf("ValidateChecksum should skip when checksum is empty: %v", err)
	}
}

func TestDetectGitVersion(t *testing.T) {
	// Test in the bmd repo itself (which is a git repo).
	cwd, _ := os.Getwd()
	ver, err := DetectGitVersion(cwd)
	// We don't require a tag, but git describe --always should return a commit hash.
	if err != nil {
		t.Skipf("Git not available or not a repo: %v", err)
	}
	if ver == "" {
		t.Error("DetectGitVersion returned empty string")
	}
}

func TestDetectGitProvenance(t *testing.T) {
	cwd, _ := os.Getwd()
	_, _, gitCommit := DetectGitProvenance(cwd)
	// In the bmd repo, we should at least get a commit hash.
	if gitCommit == "" {
		t.Skip("Not in a git repo or git not available")
	}
}

func TestKnowledgeMetadataGitFields(t *testing.T) {
	tmpDir := t.TempDir()
	writeTestFile(t, filepath.Join(tmpDir, "test.md"), "# Git Test\n\nContent")

	outputPath := filepath.Join(tmpDir, "git-meta.tar.gz")
	err := CmdExport([]string{"--from", tmpDir, "--output", outputPath, "--version", "3.0.0"})
	if err != nil {
		t.Fatalf("CmdExport() error: %v", err)
	}

	metaJSON := extractFileFromTarGz(t, outputPath, "knowledge.json")
	var meta KnowledgeMetadata
	if err := json.Unmarshal(metaJSON, &meta); err != nil {
		t.Fatalf("Failed to parse knowledge.json: %v", err)
	}

	// Version should be explicitly set.
	if meta.Version != "3.0.0" {
		t.Errorf("Version = %q, want %q", meta.Version, "3.0.0")
	}

	// Git commit should be populated if we're in a git repo.
	if meta.GitCommit != "" {
		t.Logf("GitCommit: %s", meta.GitCommit)
	}
	if meta.FromRepo != "" {
		t.Logf("FromRepo: %s", meta.FromRepo)
	}
}

// --- test helpers ---

func writeTestFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %q: %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %q: %v", path, err)
	}
}

func listTarGzEntries(t *testing.T, path string) []string {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("open %q: %v", path, err)
	}
	defer f.Close()

	gz, err := gzip.NewReader(f)
	if err != nil {
		t.Fatalf("gzip reader: %v", err)
	}
	defer gz.Close()

	tr := tar.NewReader(gz)
	var entries []string
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("tar next: %v", err)
		}
		entries = append(entries, header.Name)
	}
	return entries
}

func extractFileFromTarGz(t *testing.T, archivePath, targetFile string) []byte {
	t.Helper()
	f, err := os.Open(archivePath)
	if err != nil {
		t.Fatalf("open %q: %v", archivePath, err)
	}
	defer f.Close()

	gz, err := gzip.NewReader(f)
	if err != nil {
		t.Fatalf("gzip reader: %v", err)
	}
	defer gz.Close()

	tr := tar.NewReader(gz)
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("tar next: %v", err)
		}
		if header.Name == targetFile {
			data, err := io.ReadAll(tr)
			if err != nil {
				t.Fatalf("read %q: %v", targetFile, err)
			}
			return data
		}
	}
	return nil
}
