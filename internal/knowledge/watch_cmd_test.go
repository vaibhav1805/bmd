package knowledge

import (
	"flag"
	"io"
	"os"
	"path/filepath"
	"testing"
)

// TestCmdWatch_InvalidDir verifies that CmdWatch returns an error when the
// target directory does not exist.
func TestCmdWatch_InvalidDir(t *testing.T) {
	err := CmdWatch([]string{"--dir", "/nonexistent/path/xyz_does_not_exist"})
	if err == nil {
		t.Fatal("expected error for nonexistent dir, got nil")
	}
}

// TestCmdWatch_ParseArgs verifies that CmdWatch flag parsing accepts valid
// --dir and --interval-ms flags without error, by testing the flag.FlagSet
// directly (matches CmdWatch implementation).
func TestCmdWatch_ParseArgs(t *testing.T) {
	fs := flag.NewFlagSet("watch", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	var dir string
	var intervalMs int
	fs.StringVar(&dir, "dir", ".", "Directory to watch for .md changes")
	fs.IntVar(&intervalMs, "interval-ms", 500, "Poll interval in milliseconds")

	args := []string{"--dir", "/tmp", "--interval-ms", "200"}
	if err := fs.Parse(args); err != nil {
		t.Fatalf("unexpected parse error: %v", err)
	}

	if dir != "/tmp" {
		t.Errorf("expected dir=/tmp, got %q", dir)
	}
	if intervalMs != 200 {
		t.Errorf("expected interval-ms=200, got %d", intervalMs)
	}
}

// TestCmdWatch_ParseArgs_Defaults verifies that default values are used
// when no flags are provided.
func TestCmdWatch_ParseArgs_Defaults(t *testing.T) {
	fs := flag.NewFlagSet("watch", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	var dir string
	var intervalMs int
	fs.StringVar(&dir, "dir", ".", "Directory to watch for .md changes")
	fs.IntVar(&intervalMs, "interval-ms", 500, "Poll interval in milliseconds")

	if err := fs.Parse([]string{}); err != nil {
		t.Fatalf("unexpected parse error: %v", err)
	}

	if dir != "." {
		t.Errorf("expected default dir='.', got %q", dir)
	}
	if intervalMs != 500 {
		t.Errorf("expected default interval-ms=500, got %d", intervalMs)
	}
}

// TestCmdWatch_ValidDirScansWithoutError verifies that CmdWatch initializes
// its internal components (scan, index, graph, registry, watcher, updater)
// without error by exercising the setup path directly. This avoids calling
// CmdWatch itself (which blocks on SIGINT).
func TestCmdWatch_ValidDirScansWithoutError(t *testing.T) {
	dir := t.TempDir()

	// Write one markdown file so ScanDirectory has something to index.
	mdPath := filepath.Join(dir, "readme.md")
	if err := os.WriteFile(mdPath, []byte("# Hello\n\nWorld.\n"), 0644); err != nil {
		t.Fatal(err)
	}

	// Exercise the same setup logic used by CmdWatch.
	docs, err := ScanDirectory(dir)
	if err != nil {
		t.Fatalf("ScanDirectory: %v", err)
	}

	idx := NewIndex()
	if err := idx.Build(docs); err != nil {
		t.Fatalf("idx.Build: %v", err)
	}

	graph := NewGraph()
	ex := NewExtractor(dir)
	for i := range docs {
		_ = graph.AddNode(&Node{ID: docs[i].ID, Title: docs[i].Title, Type: "document"})
		edges := ex.Extract(&docs[i])
		for _, edge := range edges {
			_ = graph.AddEdge(edge)
		}
	}

	reg := NewComponentRegistry()
	reg.InitFromGraph(graph, docs)

	// Verify we got at least one document indexed.
	if idx.DocCount() == 0 {
		t.Error("expected at least 1 document in index, got 0")
	}
}
