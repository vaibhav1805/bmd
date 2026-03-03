package knowledge

import (
	"os"
	"path/filepath"
	"testing"
)

// buildSmallIndex creates and builds an index from the standard test corpus.
func buildSmallIndex(t *testing.T) *Index {
	t.Helper()
	idx := NewIndex()
	docs := makeTestDocs()
	if err := idx.Build(docs); err != nil {
		t.Fatalf("Index.Build: %v", err)
	}
	return idx
}

func TestIndex_Build(t *testing.T) {
	idx := buildSmallIndex(t)
	if idx.DocCount() != 3 {
		t.Fatalf("DocCount = %d, want 3", idx.DocCount())
	}
}

func TestIndex_Search_ReturnsRankedResults(t *testing.T) {
	idx := buildSmallIndex(t)

	results, err := idx.Search("authentication", 10)
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(results) == 0 {
		t.Fatal("expected results for 'authentication'")
	}
	// DocID is now chunk-level; check RelPath for the file identity.
	if results[0].RelPath != "auth.md" {
		t.Errorf("top result RelPath = %q, want auth.md", results[0].RelPath)
	}
}

func TestIndex_Search_EmptyQuery(t *testing.T) {
	idx := buildSmallIndex(t)
	results, err := idx.Search("", 10)
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("expected 0 results for empty query, got %d", len(results))
	}
}

func TestIndex_Search_UnknownTerm(t *testing.T) {
	idx := buildSmallIndex(t)
	results, err := idx.Search("xyzzy", 10)
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("expected 0 results for unknown term, got %d", len(results))
	}
}

func TestIndex_Search_SnippetPresent(t *testing.T) {
	idx := buildSmallIndex(t)
	results, err := idx.Search("authentication", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) == 0 {
		t.Fatal("no results")
	}
	if results[0].Snippet == "" {
		t.Error("expected non-empty snippet")
	}
	if len([]rune(results[0].Snippet)) > 200 {
		t.Errorf("snippet length %d exceeds 200", len([]rune(results[0].Snippet)))
	}
}

func TestIndex_Search_MatchCountPositive(t *testing.T) {
	idx := buildSmallIndex(t)
	results, err := idx.Search("authentication token", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) == 0 {
		t.Fatal("no results")
	}
	if results[0].MatchCount <= 0 {
		t.Errorf("expected MatchCount > 0, got %d", results[0].MatchCount)
	}
}

func TestIndex_Search_TopKEnforced(t *testing.T) {
	idx := buildSmallIndex(t)
	results, err := idx.Search("service", 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) > 1 {
		t.Errorf("expected at most 1 result with topK=1, got %d", len(results))
	}
}

func TestIndex_Persist_SaveAndLoad(t *testing.T) {
	idx := buildSmallIndex(t)

	// Save.
	dir := t.TempDir()
	path := filepath.Join(dir, "index.json")
	if err := idx.Save(path); err != nil {
		t.Fatalf("Save: %v", err)
	}

	// Verify file was written.
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("index file not found: %v", err)
	}
	if info.Size() == 0 {
		t.Error("index file is empty")
	}

	// Load into a new index.
	idx2 := NewIndex()
	if err := idx2.Load(path); err != nil {
		t.Fatalf("Load: %v", err)
	}

	if idx2.DocCount() != idx.DocCount() {
		t.Errorf("DocCount after load = %d, want %d", idx2.DocCount(), idx.DocCount())
	}

	// Search result should be identical.
	r1, _ := idx.Search("authentication", 10)
	r2, _ := idx2.Search("authentication", 10)
	if len(r1) != len(r2) {
		t.Errorf("search result count differs after load: %d vs %d", len(r1), len(r2))
	}
	if len(r1) > 0 && r1[0].DocID != r2[0].DocID {
		t.Errorf("top result differs after load: %q vs %q", r1[0].DocID, r2[0].DocID)
	}
}

func TestIndex_Persist_LoadMissingFile(t *testing.T) {
	idx := NewIndex()
	err := idx.Load("/nonexistent/path/index.json")
	if err == nil {
		t.Fatal("expected error loading missing file")
	}
}

func TestIndex_Persist_LoadCorrupt(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.json")
	if err := os.WriteFile(path, []byte("not json {{{{"), 0o644); err != nil {
		t.Fatal(err)
	}
	idx := NewIndex()
	err := idx.Load(path)
	if err == nil {
		t.Fatal("expected error loading corrupt JSON")
	}
}

func TestIndex_IsStale_NewFile(t *testing.T) {
	// Build index over an empty directory, then add a file — should be stale.
	root := t.TempDir()
	idx := NewIndex()
	if err := idx.Build(nil); err != nil {
		t.Fatal(err)
	}

	// Add a file after the index was built.
	if err := os.WriteFile(filepath.Join(root, "new.md"), []byte("# New"), 0o644); err != nil {
		t.Fatal(err)
	}

	stale, err := idx.IsStale(root)
	if err != nil {
		t.Fatalf("IsStale: %v", err)
	}
	if !stale {
		t.Error("expected index to be stale after new file added")
	}
}

func TestIndex_IsStale_NoChanges(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "doc.md")
	if err := os.WriteFile(path, []byte("# Doc"), 0o644); err != nil {
		t.Fatal(err)
	}

	docs, err := ScanDirectory(root, ScanConfig{UseDefaultIgnores: true})
	if err != nil {
		t.Fatal(err)
	}

	idx := NewIndex()
	if err := idx.Build(docs); err != nil {
		t.Fatal(err)
	}

	stale, err := idx.IsStale(root)
	if err != nil {
		t.Fatalf("IsStale: %v", err)
	}
	if stale {
		t.Error("expected index to be fresh when nothing changed")
	}
}

func TestIndex_UpdateDocuments_AddNew(t *testing.T) {
	idx := buildSmallIndex(t)
	before := idx.DocCount()

	newDoc := Document{
		ID:          "new.md",
		Path:        "/new.md",
		RelPath:     "new.md",
		Title:       "New Doc",
		PlainText:   "kubernetes cluster deployment scaling",
		ContentHash: "abc123",
	}
	if err := idx.UpdateDocuments([]Document{newDoc}, nil); err != nil {
		t.Fatalf("UpdateDocuments: %v", err)
	}

	if idx.DocCount() != before+1 {
		t.Errorf("DocCount = %d, want %d", idx.DocCount(), before+1)
	}

	results, _ := idx.Search("kubernetes", 5)
	if len(results) == 0 {
		t.Error("expected 'kubernetes' to be findable after update")
	}
}

func TestIndex_UpdateDocuments_RemoveDoc(t *testing.T) {
	idx := buildSmallIndex(t)
	before := idx.DocCount()

	if err := idx.UpdateDocuments(nil, []string{"auth.md"}); err != nil {
		t.Fatal(err)
	}

	if idx.DocCount() != before-1 {
		t.Errorf("DocCount = %d, want %d", idx.DocCount(), before-1)
	}

	results, _ := idx.Search("authentication", 5)
	for _, r := range results {
		if r.DocID == "auth.md" {
			t.Error("removed doc should not appear in search results")
		}
	}
}

func TestIndex_UpdateDocuments_SkipsUnchanged(t *testing.T) {
	idx := buildSmallIndex(t)

	// Same doc with same hash — should be a no-op (no panic, count unchanged).
	docs := makeTestDocs()
	docs[0].ContentHash = "" // reset hash to simulate "no hash"
	before := idx.DocCount()

	if err := idx.UpdateDocuments(docs[:1], nil); err != nil {
		t.Fatal(err)
	}
	if idx.DocCount() != before {
		t.Errorf("expected no change in DocCount, got %d (want %d)", idx.DocCount(), before)
	}
}

func TestIndex_BuildRebuild(t *testing.T) {
	idx := buildSmallIndex(t)

	// Rebuild with a different corpus — old data should be gone.
	newDocs := []Document{{
		ID:        "only.md",
		Path:      "/only.md",
		RelPath:   "only.md",
		PlainText: "completely different content",
	}}
	if err := idx.Build(newDocs); err != nil {
		t.Fatal(err)
	}
	if idx.DocCount() != 1 {
		t.Errorf("DocCount after rebuild = %d, want 1", idx.DocCount())
	}

	// Old docs should not be searchable.
	results, _ := idx.Search("authentication", 5)
	if len(results) != 0 {
		t.Errorf("old doc found after rebuild")
	}
}

// ─── Chunk extraction unit tests ─────────────────────────────────────────────

func TestExtractChunks_EmptyContent(t *testing.T) {
	chunks := extractChunks("test.md", "")
	if chunks != nil {
		t.Errorf("expected nil for empty content, got %d chunks", len(chunks))
	}
}

func TestExtractChunks_WhitespaceOnly(t *testing.T) {
	chunks := extractChunks("test.md", "   \n\n   ")
	if chunks != nil {
		t.Errorf("expected nil for whitespace-only content, got %d chunks", len(chunks))
	}
}

func TestExtractChunks_NoHeadings(t *testing.T) {
	content := "Just some plain text.\nWith multiple lines.\nNo headings here."
	chunks := extractChunks("test.md", content)
	if len(chunks) != 1 {
		t.Fatalf("expected 1 chunk for no-heading content, got %d", len(chunks))
	}
	if chunks[0].HeadingPath != "" {
		t.Errorf("HeadingPath = %q, want empty string", chunks[0].HeadingPath)
	}
	if chunks[0].StartLine != 1 {
		t.Errorf("StartLine = %d, want 1", chunks[0].StartLine)
	}
	if chunks[0].RelPath != "test.md" {
		t.Errorf("RelPath = %q, want test.md", chunks[0].RelPath)
	}
}

func TestExtractChunks_OneHeadingNoPreamble(t *testing.T) {
	content := "# Installation\n\nStep 1. Do this.\nStep 2. Do that."
	chunks := extractChunks("readme.md", content)
	// Document starts with heading on line 1 → no preamble chunk, just 1 heading chunk.
	if len(chunks) != 1 {
		t.Fatalf("expected 1 chunk (first line is heading), got %d", len(chunks))
	}
	if chunks[0].HeadingPath != "Installation" {
		t.Errorf("HeadingPath = %q, want %q", chunks[0].HeadingPath, "Installation")
	}
	if chunks[0].StartLine != 1 {
		t.Errorf("StartLine = %d, want 1", chunks[0].StartLine)
	}
	if chunks[0].EndLine < chunks[0].StartLine {
		t.Errorf("EndLine (%d) < StartLine (%d)", chunks[0].EndLine, chunks[0].StartLine)
	}
}

func TestExtractChunks_PreamblePlusHeading(t *testing.T) {
	content := "Intro paragraph.\n\n# Section One\n\nSection content."
	chunks := extractChunks("doc.md", content)
	if len(chunks) != 2 {
		t.Fatalf("expected 2 chunks (preamble + heading), got %d", len(chunks))
	}
	// First chunk is the preamble.
	if chunks[0].HeadingPath != "" {
		t.Errorf("preamble chunk HeadingPath = %q, want empty", chunks[0].HeadingPath)
	}
	// Second chunk is the heading.
	if chunks[1].HeadingPath != "Section One" {
		t.Errorf("heading chunk HeadingPath = %q, want %q", chunks[1].HeadingPath, "Section One")
	}
	if chunks[1].StartLine <= chunks[0].EndLine {
		t.Errorf("heading chunk StartLine (%d) should be after preamble EndLine (%d)",
			chunks[1].StartLine, chunks[0].EndLine)
	}
}

func TestExtractChunks_NestedHeadingBreadcrumb(t *testing.T) {
	content := "# Parent\n\nParent text.\n\n## Child\n\nChild text."
	chunks := extractChunks("doc.md", content)
	if len(chunks) < 2 {
		t.Fatalf("expected at least 2 chunks for nested headings, got %d", len(chunks))
	}
	// Find the ## Child chunk.
	var childChunk *Chunk
	for i := range chunks {
		if chunks[i].HeadingPath == "Parent > Child" {
			childChunk = &chunks[i]
			break
		}
	}
	if childChunk == nil {
		t.Errorf("expected chunk with HeadingPath 'Parent > Child'; got paths: %v",
			func() []string {
				paths := make([]string, len(chunks))
				for i, c := range chunks {
					paths[i] = c.HeadingPath
				}
				return paths
			}())
	}
}

func TestExtractChunks_RepeatedHeadingUniqueDocIDs(t *testing.T) {
	content := "# Setup\n\nFirst section.\n\n# Setup\n\nSecond section."
	chunks := extractChunks("doc.md", content)
	if len(chunks) < 2 {
		t.Fatalf("expected at least 2 chunks, got %d", len(chunks))
	}
	// DocIDs must be unique even though heading text repeats.
	seen := make(map[string]bool)
	for _, c := range chunks {
		if seen[c.DocID] {
			t.Errorf("duplicate DocID: %q", c.DocID)
		}
		seen[c.DocID] = true
	}
}

// ─── Chunk-level search integration tests ────────────────────────────────────

func TestChunkLevelSearch_HeadingPathInResult(t *testing.T) {
	idx := NewIndex()
	doc := Document{
		ID:      "guide.md",
		Path:    "/guide.md",
		RelPath: "guide.md",
		Title:   "Guide",
		Content: "# Installation\n\nRun the installer binary.\n\n# Usage\n\nQuery with bmd query command.",
	}
	if err := idx.Build([]Document{doc}); err != nil {
		t.Fatalf("Build: %v", err)
	}

	results, err := idx.Search("installer", 10)
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(results) == 0 {
		t.Fatal("expected results for 'installer'")
	}
	r := results[0]
	if r.HeadingPath == "" {
		t.Error("HeadingPath should be non-empty for chunk-level result")
	}
	if r.StartLine <= 0 {
		t.Errorf("StartLine = %d, want > 0", r.StartLine)
	}
	if r.EndLine < r.StartLine {
		t.Errorf("EndLine (%d) < StartLine (%d)", r.EndLine, r.StartLine)
	}
	if r.ContentPreview == "" {
		t.Error("ContentPreview should be non-empty")
	}
}

func TestChunkLevelSearch_MultiSectionRanking(t *testing.T) {
	idx := NewIndex()
	doc := Document{
		ID:      "multi.md",
		Path:    "/multi.md",
		RelPath: "multi.md",
		Title:   "Multi",
		Content: "# Install\n\nInstall the package here.\n\n# Usage\n\nUsage instructions go here.",
	}
	if err := idx.Build([]Document{doc}); err != nil {
		t.Fatalf("Build: %v", err)
	}

	// Search for term in "Install" section.
	installResults, err := idx.Search("install", 10)
	if err != nil {
		t.Fatalf("Search install: %v", err)
	}
	if len(installResults) == 0 {
		t.Fatal("expected results for 'install'")
	}
	if installResults[0].HeadingPath != "Install" {
		t.Errorf("HeadingPath for 'install' query = %q, want %q", installResults[0].HeadingPath, "Install")
	}

	// Search for term in "Usage" section.
	usageResults, err := idx.Search("usage", 10)
	if err != nil {
		t.Fatalf("Search usage: %v", err)
	}
	if len(usageResults) == 0 {
		t.Fatal("expected results for 'usage'")
	}
	if usageResults[0].HeadingPath != "Usage" {
		t.Errorf("HeadingPath for 'usage' query = %q, want %q", usageResults[0].HeadingPath, "Usage")
	}
}

func TestChunkLevelSearch_NoHeadingFallback(t *testing.T) {
	idx := NewIndex()
	doc := Document{
		ID:      "flat.md",
		Path:    "/flat.md",
		RelPath: "flat.md",
		Title:   "Flat Doc",
		Content: "Plain content without any headings. Some words to search.",
	}
	if err := idx.Build([]Document{doc}); err != nil {
		t.Fatalf("Build: %v", err)
	}

	results, err := idx.Search("plain", 10)
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(results) == 0 {
		t.Fatal("expected result for no-heading doc")
	}
	if results[0].HeadingPath != "" {
		t.Errorf("HeadingPath for no-heading doc = %q, want empty string", results[0].HeadingPath)
	}
}

func TestChunkLevelSearch_MultipleChunksFromSameFile(t *testing.T) {
	idx := NewIndex()
	doc := Document{
		ID:      "big.md",
		Path:    "/big.md",
		RelPath: "big.md",
		Title:   "Big Doc",
		Content: "# Authentication\n\nToken based auth system.\n\n# Authorization\n\nRole based access control.",
	}
	if err := idx.Build([]Document{doc}); err != nil {
		t.Fatalf("Build: %v", err)
	}

	// Both "auth" related sections should rank independently.
	results, err := idx.Search("token role", 10)
	if err != nil {
		t.Fatalf("Search: %v", err)
	}

	// Verify we can get results from both sections of the same file.
	headingPaths := make(map[string]bool)
	for _, r := range results {
		if r.RelPath == "big.md" {
			headingPaths[r.HeadingPath] = true
		}
	}
	// Multiple chunks from big.md should appear independently.
	if len(headingPaths) < 2 {
		t.Logf("Results: %+v", results)
		t.Errorf("expected chunks from 2 sections of big.md to rank independently, got %d unique heading paths", len(headingPaths))
	}
}

func BenchmarkIndex_Search1000Docs(b *testing.B) {
	idx := NewIndex()
	docs := make([]Document, 1000)
	words := []string{"authentication", "gateway", "service", "token", "cluster",
		"database", "cache", "queue", "worker", "handler"}

	for i := range 1000 {
		w := words[i%len(words)]
		docs[i] = Document{
			ID:        w + "_" + string(rune(i)) + ".md",
			Path:      "/" + w + ".md",
			RelPath:   w + ".md",
			PlainText: w + " integration " + words[(i+2)%len(words)] + " service",
		}
	}

	if err := idx.Build(docs); err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for range b.N {
		_, _ = idx.Search("authentication service", 10)
	}
}
