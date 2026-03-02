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
