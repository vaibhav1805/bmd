package knowledge

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestNewDocument_Valid(t *testing.T) {
	now := time.Now()
	doc, err := NewDocument("id1", "/abs/path.md", "rel/path.md", "Title", "content", "plain", now)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if doc.ID != "id1" {
		t.Errorf("ID = %q, want %q", doc.ID, "id1")
	}
	if doc.Path != "/abs/path.md" {
		t.Errorf("Path = %q", doc.Path)
	}
	if doc.RelPath != "rel/path.md" {
		t.Errorf("RelPath = %q", doc.RelPath)
	}
	if doc.Title != "Title" {
		t.Errorf("Title = %q", doc.Title)
	}
	if doc.Content != "content" {
		t.Errorf("Content = %q", doc.Content)
	}
	if doc.PlainText != "plain" {
		t.Errorf("PlainText = %q", doc.PlainText)
	}
	if !doc.LastModified.Equal(now) {
		t.Errorf("LastModified = %v, want %v", doc.LastModified, now)
	}
}

func TestNewDocument_MissingID(t *testing.T) {
	_, err := NewDocument("", "/path.md", "rel.md", "T", "c", "p", time.Now())
	if err == nil {
		t.Fatal("expected error for empty ID, got nil")
	}
}

func TestNewDocument_MissingPath(t *testing.T) {
	_, err := NewDocument("id", "", "rel.md", "T", "c", "p", time.Now())
	if err == nil {
		t.Fatal("expected error for empty path, got nil")
	}
}

func TestNewDocument_MissingRelPath(t *testing.T) {
	_, err := NewDocument("id", "/path.md", "", "T", "c", "p", time.Now())
	if err == nil {
		t.Fatal("expected error for empty relPath, got nil")
	}
}

func TestDocumentFromFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "hello.md")
	content := "# Hello World\n\nThis is a test document.\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	doc, err := DocumentFromFile(path, "hello.md")
	if err != nil {
		t.Fatalf("DocumentFromFile: %v", err)
	}

	if doc.ID != "hello.md" {
		t.Errorf("ID = %q, want %q", doc.ID, "hello.md")
	}
	if doc.Path != path {
		t.Errorf("Path = %q, want %q", doc.Path, path)
	}
	if doc.RelPath != "hello.md" {
		t.Errorf("RelPath = %q", doc.RelPath)
	}
	if doc.Title != "Hello World" {
		t.Errorf("Title = %q, want %q", doc.Title, "Hello World")
	}
	if doc.Content != content {
		t.Errorf("Content mismatch")
	}
	if doc.ContentHash == "" {
		t.Error("ContentHash should not be empty")
	}
	if doc.LastModified.IsZero() {
		t.Error("LastModified should not be zero")
	}
}

func TestDocumentFromFile_NoH1FallsBackToFilename(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "my-service.md")
	if err := os.WriteFile(path, []byte("## Secondary\n\nContent"), 0o644); err != nil {
		t.Fatal(err)
	}

	doc, err := DocumentFromFile(path, "my-service.md")
	if err != nil {
		t.Fatal(err)
	}
	if doc.Title != "my-service" {
		t.Errorf("Title = %q, want %q", doc.Title, "my-service")
	}
}

func TestDocumentFromFile_MissingRelPath(t *testing.T) {
	_, err := DocumentFromFile("/some/path.md", "")
	if err == nil {
		t.Fatal("expected error for empty relPath")
	}
}

func TestDocumentFromFile_MissingFile(t *testing.T) {
	_, err := DocumentFromFile("/nonexistent/path.md", "path.md")
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestDocumentCollection(t *testing.T) {
	col := NewDocumentCollection()
	if col.Len() != 0 {
		t.Fatalf("expected 0 docs, got %d", col.Len())
	}

	d1 := &Document{ID: "a", Path: "/a.md", RelPath: "a.md"}
	d2 := &Document{ID: "b", Path: "/b.md", RelPath: "b.md"}

	col.Add(d1)
	col.Add(d2)
	if col.Len() != 2 {
		t.Fatalf("expected 2, got %d", col.Len())
	}

	if got := col.Get("a"); got != d1 {
		t.Errorf("Get(a) = %v, want %v", got, d1)
	}

	// Replace existing doc.
	d1b := &Document{ID: "a", Path: "/a-new.md", RelPath: "a-new.md"}
	col.Add(d1b)
	if col.Len() != 2 {
		t.Fatalf("expected 2 after replace, got %d", col.Len())
	}
	if got := col.Get("a"); got != d1b {
		t.Errorf("Get after replace = %v, want %v", got, d1b)
	}

	// Remove.
	col.Remove("a")
	if col.Len() != 1 {
		t.Fatalf("expected 1 after remove, got %d", col.Len())
	}
	if got := col.Get("a"); got != nil {
		t.Errorf("Get after remove = %v, want nil", got)
	}

	// Remove non-existent — should be no-op.
	col.Remove("nonexistent")
	if col.Len() != 1 {
		t.Fatalf("expected 1 after no-op remove, got %d", col.Len())
	}

	all := col.All()
	if len(all) != 1 || all[0].ID != "b" {
		t.Errorf("All() = %v, unexpected", all)
	}
}
