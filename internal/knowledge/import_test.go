package knowledge

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestImportKnowledgeTar(t *testing.T) {
	// Create a temp dir with markdown files and export it.
	srcDir := t.TempDir()
	writeTestFile(t, filepath.Join(srcDir, "readme.md"), "# Hello\n\nWorld")
	writeTestFile(t, filepath.Join(srcDir, "guide.md"), "# Guide\n\nContent here")

	tarPath := filepath.Join(srcDir, "test-export.tar.gz")
	err := CmdExport([]string{"--from", srcDir, "--output", tarPath})
	if err != nil {
		t.Fatalf("CmdExport() error: %v", err)
	}

	// Import into a new directory.
	destDir := t.TempDir()
	result, err := ImportKnowledgeTar(tarPath, destDir)
	if err != nil {
		t.Fatalf("ImportKnowledgeTar() error: %v", err)
	}

	if result.ExtractDir != destDir {
		t.Errorf("ExtractDir = %q, want %q", result.ExtractDir, destDir)
	}
	if result.Metadata == nil {
		t.Fatal("Metadata should not be nil")
	}
	if result.Metadata.Version != "1.0.0" {
		t.Errorf("Version = %q, want %q", result.Metadata.Version, "1.0.0")
	}
	if result.FileCount != 2 {
		t.Errorf("FileCount = %d, want 2", result.FileCount)
	}
	if result.DBPath == "" {
		t.Error("DBPath should not be empty")
	}

	// Verify extracted files exist.
	for _, name := range []string{"readme.md", "guide.md", "knowledge.json"} {
		path := filepath.Join(destDir, name)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("Expected extracted file %q not found", name)
		}
	}
}

func TestImportTempDir(t *testing.T) {
	// Create export.
	srcDir := t.TempDir()
	writeTestFile(t, filepath.Join(srcDir, "test.md"), "# Test")

	tarPath := filepath.Join(srcDir, "export.tar.gz")
	if err := CmdExport([]string{"--from", srcDir, "--output", tarPath}); err != nil {
		t.Fatalf("CmdExport() error: %v", err)
	}

	// Import with empty destDir (should create temp dir).
	result, err := ImportKnowledgeTar(tarPath, "")
	if err != nil {
		t.Fatalf("ImportKnowledgeTar() error: %v", err)
	}
	defer os.RemoveAll(result.ExtractDir)

	if result.ExtractDir == "" {
		t.Error("ExtractDir should not be empty")
	}
	// Verify the temp dir was created and has files.
	if _, err := os.Stat(filepath.Join(result.ExtractDir, "knowledge.json")); os.IsNotExist(err) {
		t.Error("knowledge.json not found in temp extract dir")
	}
}

func TestImportInvalidArchive(t *testing.T) {
	// Create a fake tar.gz with no knowledge.json.
	tmpDir := t.TempDir()
	fakeTar := filepath.Join(tmpDir, "fake.tar.gz")
	if err := os.WriteFile(fakeTar, []byte("not a tar"), 0o644); err != nil {
		t.Fatalf("write fake tar: %v", err)
	}

	_, err := ImportKnowledgeTar(fakeTar, t.TempDir())
	if err == nil {
		t.Fatal("Expected error for invalid archive")
	}
}

func TestImportNonexistentTar(t *testing.T) {
	_, err := ImportKnowledgeTar("/nonexistent/path.tar.gz", t.TempDir())
	if err == nil {
		t.Fatal("Expected error for nonexistent tar")
	}
}

func TestImportPreBuiltIndex(t *testing.T) {
	// Create export with indexed content.
	srcDir := t.TempDir()
	writeTestFile(t, filepath.Join(srcDir, "api.md"), "# API Reference\n\nAuthentication endpoints for users")
	writeTestFile(t, filepath.Join(srcDir, "guide.md"), "# Getting Started\n\nInstall the SDK first")

	tarPath := filepath.Join(srcDir, "indexed-export.tar.gz")
	if err := CmdExport([]string{"--from", srcDir, "--output", tarPath}); err != nil {
		t.Fatalf("CmdExport() error: %v", err)
	}

	// Import and load database.
	db, extractDir, err := LoadFromKnowledgeTar(tarPath, "")
	if err != nil {
		t.Fatalf("LoadFromKnowledgeTar() error: %v", err)
	}
	defer db.Close()
	defer os.RemoveAll(extractDir)

	// Verify the database has content (pre-built, no rebuild needed).
	idx := NewIndex()
	if err := db.LoadIndex(idx); err != nil {
		t.Fatalf("LoadIndex() error: %v", err)
	}

	if idx.DocCount() == 0 {
		t.Error("Pre-built index should have documents loaded")
	}
}

func TestRoundTripExportImport(t *testing.T) {
	// Full round-trip: create files -> export -> import -> verify content matches.
	srcDir := t.TempDir()
	originalContent := "# Architecture\n\nMicroservices pattern with API gateway"
	writeTestFile(t, filepath.Join(srcDir, "arch.md"), originalContent)

	tarPath := filepath.Join(srcDir, "round-trip.tar.gz")
	if err := CmdExport([]string{"--from", srcDir, "--output", tarPath}); err != nil {
		t.Fatalf("CmdExport() error: %v", err)
	}

	destDir := t.TempDir()
	result, err := ImportKnowledgeTar(tarPath, destDir)
	if err != nil {
		t.Fatalf("ImportKnowledgeTar() error: %v", err)
	}

	// Read the imported file and compare.
	importedContent, err := os.ReadFile(filepath.Join(destDir, "arch.md"))
	if err != nil {
		t.Fatalf("Read imported file: %v", err)
	}

	if string(importedContent) != originalContent {
		t.Errorf("Content mismatch.\nGot: %q\nWant: %q", string(importedContent), originalContent)
	}

	// Verify metadata.
	if result.Metadata.FileCount != 1 {
		t.Errorf("FileCount = %d, want 1", result.Metadata.FileCount)
	}
	if !strings.HasPrefix(result.Metadata.Checksum, "sha256:") {
		t.Errorf("Checksum should start with 'sha256:', got %q", result.Metadata.Checksum)
	}
}
