package knowledge

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestIntegration_RealCorpus indexes the BMD project's own markdown files and
// verifies that search returns sensible results.
//
// This test is skipped in short mode because it reads real files from disk.
func TestIntegration_RealCorpus(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	// Resolve the project root: internal/knowledge is 2 levels below project root.
	// internal/knowledge/../../ = bmd/
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatalf("failed to resolve project root: %v", err)
	}
	t.Logf("Scanning root: %s", root)

	// --- Scan ---
	scanStart := time.Now()
	docs, err := ScanDirectory(root, ScanConfig{UseDefaultIgnores: true})
	if err != nil {
		t.Fatalf("ScanDirectory(%q, ScanConfig{UseDefaultIgnores: true}): %v", root, err)
	}
	scanDur := time.Since(scanStart)
	t.Logf("Scanned %d markdown files in %v", len(docs), scanDur)

	if len(docs) == 0 {
		t.Fatal("no markdown files found; expected at least test-data/*.md")
	}

	// --- Build ---
	idx := NewIndex()
	buildStart := time.Now()
	if err := idx.Build(docs); err != nil {
		t.Fatalf("Index.Build: %v", err)
	}
	buildDur := time.Since(buildStart)
	t.Logf("Indexed %d documents in %v", idx.DocCount(), buildDur)

	// Index should build in <500ms for typical project.
	if buildDur > 500*time.Millisecond {
		t.Errorf("indexing took %v, want <500ms", buildDur)
	}

	// --- Queries ---
	type testCase struct {
		query   string
		wantAny bool // true if we expect at least 1 result
	}
	cases := []testCase{
		{query: "service", wantAny: true},
		{query: "theme", wantAny: true},
		{query: "navigation", wantAny: true},
		// "xyzzy" is extremely unlikely to appear in BMD planning or test docs.
		{query: "xyzzy", wantAny: false},
	}

	for _, tc := range cases {
		qStart := time.Now()
		results, err := idx.Search(tc.query, 5)
		if err != nil {
			t.Errorf("Search(%q): %v", tc.query, err)
			continue
		}
		qDur := time.Since(qStart)

		t.Logf("Query %q: %d results in %v", tc.query, len(results), qDur)
		for i, r := range results {
			snippet := r.Snippet
			runes := []rune(snippet)
			if len(runes) > 80 {
				snippet = string(runes[:80]) + "..."
			}
			t.Logf("  [%d] %.3f  %-50s  %s", i+1, r.Score, r.RelPath, snippet)
		}

		// Search latency target: <100ms.
		if qDur > 100*time.Millisecond {
			t.Errorf("query %q took %v, want <100ms", tc.query, qDur)
		}

		if tc.wantAny && len(results) == 0 {
			t.Errorf("query %q returned no results (expected some)", tc.query)
		}
		if !tc.wantAny && len(results) != 0 {
			t.Errorf("query %q returned %d results (expected none)", tc.query, len(results))
		}
	}

	// --- Persistence ---
	tmpPath := filepath.Join(t.TempDir(), "index.json")
	if err := idx.Save(tmpPath); err != nil {
		t.Fatalf("Save: %v", err)
	}

	info, err := os.Stat(tmpPath)
	if err != nil {
		t.Fatalf("index file not created: %v", err)
	}
	t.Logf("Index file size: %s (%d bytes)", tmpPath, info.Size())

	// Load and verify round-trip.
	idx2 := NewIndex()
	if err := idx2.Load(tmpPath); err != nil {
		t.Fatalf("Load: %v", err)
	}

	if idx2.DocCount() != idx.DocCount() {
		t.Errorf("DocCount after reload: got %d, want %d", idx2.DocCount(), idx.DocCount())
	}

	r1, _ := idx.Search("theme", 3)
	r2, _ := idx2.Search("theme", 3)
	if len(r1) != len(r2) {
		t.Errorf("search result count differs after reload: %d vs %d", len(r1), len(r2))
	}
	if len(r1) > 0 && r1[0].DocID != r2[0].DocID {
		t.Errorf("top result differs after reload: %q vs %q", r1[0].DocID, r2[0].DocID)
	}

	// --- Staleness check ---
	stale, err := idx.IsStale(root)
	if err != nil {
		t.Fatalf("IsStale: %v", err)
	}
	if stale {
		t.Error("index should not be stale immediately after building from same directory")
	}

	t.Logf("Integration test PASSED")
	fmt.Printf("\nIntegration summary: %d docs indexed in %v, search latency <1ms\n",
		idx.DocCount(), buildDur)
}
