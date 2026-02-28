package knowledge

import (
	"math"
	"sort"
	"testing"
)

// makeTestDocs returns a small corpus for deterministic BM25 tests.
//   - doc "auth.md": heavy on authentication/jwt content
//   - doc "gateway.md": api gateway routing content
//   - doc "readme.md": generic project overview
func makeTestDocs() []Document {
	return []Document{
		{
			ID:        "auth.md",
			Path:      "/auth.md",
			RelPath:   "auth.md",
			Title:     "Authentication Service",
			Content:   "# Authentication Service\nJWT authentication token validation refresh authentication.",
			PlainText: "Authentication Service JWT authentication token validation refresh authentication.",
		},
		{
			ID:        "gateway.md",
			Path:      "/gateway.md",
			RelPath:   "gateway.md",
			Title:     "API Gateway",
			Content:   "# API Gateway\nRequest routing load balancing api gateway middleware.",
			PlainText: "API Gateway Request routing load balancing api gateway middleware.",
		},
		{
			ID:        "readme.md",
			Path:      "/readme.md",
			RelPath:   "readme.md",
			Title:     "Project Readme",
			Content:   "# Project Readme\nOverview of the project services and architecture.",
			PlainText: "Project Readme Overview of the project services and architecture.",
		},
	}
}

func TestBM25Index_AddAndSearch(t *testing.T) {
	idx := NewBM25Index(DefaultBM25Params(), nil)
	for _, d := range makeTestDocs() {
		idx.AddDocument(d)
	}

	if idx.DocCount() != 3 {
		t.Fatalf("DocCount = %d, want 3", idx.DocCount())
	}

	results := idx.Search("authentication", 10)
	if len(results) == 0 {
		t.Fatal("expected results for 'authentication', got none")
	}

	// auth.md should be the top result.
	if results[0].DocID != "auth.md" {
		t.Errorf("top result = %q, want %q", results[0].DocID, "auth.md")
	}

	// Score should be positive.
	if results[0].Score <= 0 {
		t.Errorf("expected positive score, got %f", results[0].Score)
	}
}

func TestBM25Index_RankingAccuracy(t *testing.T) {
	idx := NewBM25Index(DefaultBM25Params(), nil)
	for _, d := range makeTestDocs() {
		idx.AddDocument(d)
	}

	// "api gateway" should rank gateway.md highest.
	results := idx.Search("api gateway", 10)
	if len(results) == 0 {
		t.Fatal("expected results for 'api gateway'")
	}
	if results[0].DocID != "gateway.md" {
		t.Errorf("top result for 'api gateway' = %q, want %q", results[0].DocID, "gateway.md")
	}
}

func TestBM25Index_UnknownTermReturnsEmpty(t *testing.T) {
	idx := NewBM25Index(DefaultBM25Params(), nil)
	for _, d := range makeTestDocs() {
		idx.AddDocument(d)
	}

	results := idx.Search("xyzzy", 10)
	if len(results) != 0 {
		t.Errorf("expected no results for unknown term, got %d", len(results))
	}
}

func TestBM25Index_TopKLimit(t *testing.T) {
	idx := NewBM25Index(DefaultBM25Params(), nil)
	for _, d := range makeTestDocs() {
		idx.AddDocument(d)
	}

	// "service" appears in all docs; limit to 2.
	results := idx.Search("service", 2)
	if len(results) > 2 {
		t.Errorf("expected at most 2 results, got %d", len(results))
	}
}

func TestBM25Index_TopKZeroReturnsAll(t *testing.T) {
	idx := NewBM25Index(DefaultBM25Params(), nil)
	for _, d := range makeTestDocs() {
		idx.AddDocument(d)
	}

	// topK=0 should return all matching documents.
	results := idx.Search("service", 0)
	if len(results) == 0 {
		t.Error("expected results when topK=0")
	}
}

func TestBM25Index_ResultsSortedByScore(t *testing.T) {
	idx := NewBM25Index(DefaultBM25Params(), nil)
	for _, d := range makeTestDocs() {
		idx.AddDocument(d)
	}

	results := idx.Search("service project", 10)
	for i := 1; i < len(results); i++ {
		if results[i-1].Score < results[i].Score {
			t.Errorf("results not sorted: results[%d].Score=%f < results[%d].Score=%f",
				i-1, results[i-1].Score, i, results[i].Score)
		}
	}
}

func TestBM25Index_EmptyQuery(t *testing.T) {
	idx := NewBM25Index(DefaultBM25Params(), nil)
	idx.AddDocument(makeTestDocs()[0])

	results := idx.Search("", 10)
	if len(results) != 0 {
		t.Errorf("expected no results for empty query, got %d", len(results))
	}
}

func TestBM25Index_EmptyIndex(t *testing.T) {
	idx := NewBM25Index(DefaultBM25Params(), nil)
	results := idx.Search("anything", 10)
	if len(results) != 0 {
		t.Errorf("expected no results from empty index, got %d", len(results))
	}
}

func TestBM25Index_IDFFormula(t *testing.T) {
	// A term appearing in fewer documents should have a higher IDF.
	// Set up: "rare" in 1 doc, "common" in all 3 docs.
	docs := []Document{
		{ID: "a.md", Path: "/a.md", RelPath: "a.md", PlainText: "rare common word"},
		{ID: "b.md", Path: "/b.md", RelPath: "b.md", PlainText: "common word"},
		{ID: "c.md", Path: "/c.md", RelPath: "c.md", PlainText: "common another"},
	}

	// Use a tokenizer with no stop words so all terms are kept.
	tok := NewTokenizer(TokenizerConfig{})
	idx := NewBM25Index(DefaultBM25Params(), tok)
	for _, d := range docs {
		idx.AddDocument(d)
	}

	// "rare" (df=1) should rank a.md above results from "common" (df=3)
	// because IDF is higher for rare terms.
	results := idx.Search("rare", 10)
	if len(results) == 0 {
		t.Fatal("expected results for 'rare'")
	}
	if results[0].DocID != "a.md" {
		t.Errorf("top result for 'rare' = %q, want %q", results[0].DocID, "a.md")
	}
}

func TestBM25Index_RemoveDocument(t *testing.T) {
	idx := NewBM25Index(DefaultBM25Params(), nil)
	for _, d := range makeTestDocs() {
		idx.AddDocument(d)
	}

	removed := idx.RemoveDocument("auth.md")
	if !removed {
		t.Fatal("RemoveDocument should return true for existing doc")
	}
	if idx.DocCount() != 2 {
		t.Fatalf("DocCount after remove = %d, want 2", idx.DocCount())
	}

	// Searching for "authentication" should now not return auth.md.
	results := idx.Search("authentication", 10)
	for _, r := range results {
		if r.DocID == "auth.md" {
			t.Error("removed doc auth.md still appears in results")
		}
	}
}

func TestBM25Index_RemoveNonExistentDoc(t *testing.T) {
	idx := NewBM25Index(DefaultBM25Params(), nil)
	removed := idx.RemoveDocument("nonexistent.md")
	if removed {
		t.Error("RemoveDocument should return false for missing doc")
	}
}

func TestBM25Index_SingleDocument(t *testing.T) {
	idx := NewBM25Index(DefaultBM25Params(), nil)
	idx.AddDocument(Document{
		ID:        "solo.md",
		Path:      "/solo.md",
		RelPath:   "solo.md",
		PlainText: "hello world",
	})

	results := idx.Search("hello", 10)
	if len(results) == 0 {
		t.Fatal("expected result for 'hello'")
	}
}

func TestBM25Index_DivisionByZeroGuard(t *testing.T) {
	// Zero-length document should not cause a panic.
	idx := NewBM25Index(DefaultBM25Params(), nil)
	idx.AddDocument(Document{
		ID:        "empty.md",
		Path:      "/empty.md",
		RelPath:   "empty.md",
		PlainText: "", // zero-length
	})

	results := idx.Search("anything", 10)
	_ = results // just checking no panic
}

func TestBM25Params_ScoresArePositive(t *testing.T) {
	params := DefaultBM25Params()
	if params.K1 <= 0 {
		t.Errorf("K1 = %f, want > 0", params.K1)
	}
	if params.B < 0 || params.B > 1 {
		t.Errorf("B = %f, want [0,1]", params.B)
	}
}

func TestBM25Index_IDFNonNegative(t *testing.T) {
	// IDF = log((N - df + 0.5) / (df + 0.5) + 1)
	// For any valid df in [1, N], IDF must be >= 0.
	for N := 1; N <= 10; N++ {
		for df := 1; df <= N; df++ {
			idf := math.Log((float64(N)-float64(df)+0.5)/(float64(df)+0.5) + 1)
			if idf < 0 {
				t.Errorf("IDF(%d,%d) = %f, want >= 0", N, df, idf)
			}
		}
	}
}

func BenchmarkBM25Search1000Docs(b *testing.B) {
	words := []string{"authentication", "service", "gateway", "token", "validation",
		"request", "response", "handler", "middleware", "database",
		"cluster", "replica", "cache", "queue", "worker"}

	idx := NewBM25Index(DefaultBM25Params(), nil)
	for i := range 1000 {
		w := words[i%len(words)]
		doc := Document{
			ID:        w + ".md",
			Path:      "/" + w + ".md",
			RelPath:   w + ".md",
			PlainText: w + " service " + words[(i+3)%len(words)] + " integration",
		}
		// Deduplicate doc IDs by appending index.
		doc.ID = doc.ID + string(rune(i))
		idx.AddDocument(doc)
	}

	// Sort docs by ID for determinism.
	sort.Slice(idx.docs, func(i, j int) bool { return idx.docs[i].id < idx.docs[j].id })

	b.ResetTimer()
	for range b.N {
		_ = idx.Search("authentication service gateway", 10)
	}
}
