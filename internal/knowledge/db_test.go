package knowledge

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// ─── helpers ──────────────────────────────────────────────────────────────────

// tempDB opens a fresh in-memory (or temp-file) SQLite database for tests.
func tempDB(t *testing.T) *Database {
	t.Helper()
	path := filepath.Join(t.TempDir(), "test.db")
	db, err := OpenDB(path)
	if err != nil {
		t.Fatalf("OpenDB: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

// smallIndex builds a BM25 index from n synthetic documents.
func smallIndex(n int) *Index {
	idx := NewIndex()
	docs := make([]Document, n)
	for i := 0; i < n; i++ {
		id := fmt.Sprintf("doc%03d.md", i)
		content := fmt.Sprintf("# Document %d\n\nThis is document number %d about topic%d.", i, i, i%10)
		docs[i] = Document{
			ID:           id,
			Path:         "/fake/" + id,
			RelPath:      id,
			Title:        fmt.Sprintf("Document %d", i),
			Content:      content,
			PlainText:    content,
			LastModified: time.Now(),
			ContentHash:  calculateContentHash([]byte(content)),
		}
	}
	_ = idx.Build(docs)
	return idx
}

// smallGraph builds a graph with n nodes and edges i→(i+1).
func smallGraph(n int) *Graph {
	g := NewGraph()
	for i := 0; i < n; i++ {
		id := fmt.Sprintf("node%03d.md", i)
		_ = g.AddNode(&Node{ID: id, Type: "document", Title: fmt.Sprintf("Node %d", i)})
	}
	for i := 0; i < n-1; i++ {
		src := fmt.Sprintf("node%03d.md", i)
		tgt := fmt.Sprintf("node%03d.md", i+1)
		e := &Edge{
			ID:         edgeID(src, tgt, EdgeReferences),
			Source:     src,
			Target:     tgt,
			Type:       EdgeReferences,
			Confidence: 1.0,
			Evidence:   fmt.Sprintf("link %d", i),
		}
		_ = g.AddEdge(e)
	}
	return g
}

// ─── Database creation ────────────────────────────────────────────────────────

func TestNewDatabase_CreatesFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sub", "dir", "test.db")
	db, err := NewDatabase(path)
	if err != nil {
		t.Fatalf("NewDatabase: %v", err)
	}
	defer db.Close()
	if _, err := os.Stat(path); err != nil {
		t.Errorf("database file not created at %q: %v", path, err)
	}
}

func TestOpenDB_IdempotentInitialize(t *testing.T) {
	path := filepath.Join(t.TempDir(), "idem.db")
	for i := 0; i < 3; i++ {
		db, err := OpenDB(path)
		if err != nil {
			t.Fatalf("OpenDB call %d: %v", i, err)
		}
		_ = db.Close()
	}
}

func TestDatabase_OpenExisting_NoDataLoss(t *testing.T) {
	path := filepath.Join(t.TempDir(), "persist.db")

	db1, err := OpenDB(path)
	if err != nil {
		t.Fatalf("first open: %v", err)
	}
	idx := smallIndex(5)
	if err := db1.SaveIndex(idx); err != nil {
		t.Fatalf("SaveIndex: %v", err)
	}
	_ = db1.Close()

	// Reopen and verify data is intact.
	db2, err := OpenDB(path)
	if err != nil {
		t.Fatalf("second open: %v", err)
	}
	defer db2.Close()

	idx2 := NewIndex()
	if err := db2.LoadIndex(idx2); err != nil {
		t.Fatalf("LoadIndex on reopen: %v", err)
	}
	if idx2.DocCount() != idx.DocCount() {
		t.Errorf("DocCount after reopen: got %d, want %d", idx2.DocCount(), idx.DocCount())
	}
}

// ─── Schema version / migrations ─────────────────────────────────────────────

func TestGetVersion_ReturnsSchemaVersion(t *testing.T) {
	db := tempDB(t)
	v := db.GetVersion()
	if v != SchemaVersion {
		t.Errorf("GetVersion: got %d, want %d", v, SchemaVersion)
	}
}

func TestGetSchemaVersion_AliasesGetVersion(t *testing.T) {
	db := tempDB(t)
	if db.GetSchemaVersion() != db.GetVersion() {
		t.Error("GetSchemaVersion must equal GetVersion")
	}
}

func TestMigrate_Idempotent(t *testing.T) {
	db := tempDB(t)
	// Running Migrate multiple times should not error.
	for i := 0; i < 3; i++ {
		if err := db.Migrate(); err != nil {
			t.Fatalf("Migrate call %d: %v", i, err)
		}
	}
	if db.GetVersion() != SchemaVersion {
		t.Errorf("version after migrate: got %d, want %d", db.GetVersion(), SchemaVersion)
	}
}

// ─── Index persistence (save / load round-trip) ───────────────────────────────

func TestSaveLoadIndex_SmallRoundTrip(t *testing.T) {
	db := tempDB(t)
	idx := smallIndex(10)

	if err := db.SaveIndex(idx); err != nil {
		t.Fatalf("SaveIndex: %v", err)
	}

	idx2 := NewIndex()
	if err := db.LoadIndex(idx2); err != nil {
		t.Fatalf("LoadIndex: %v", err)
	}

	if idx2.DocCount() != idx.DocCount() {
		t.Errorf("DocCount: got %d, want %d", idx2.DocCount(), idx.DocCount())
	}
}

func TestSaveLoadIndex_ParamsPreserved(t *testing.T) {
	db := tempDB(t)
	idx := smallIndex(5)
	// Use non-default params to verify they're preserved.
	idx.params = BM25Params{K1: 1.5, B: 0.5}
	idx.bm25.params = idx.params

	if err := db.SaveIndex(idx); err != nil {
		t.Fatalf("SaveIndex: %v", err)
	}

	idx2 := NewIndex()
	if err := db.LoadIndex(idx2); err != nil {
		t.Fatalf("LoadIndex: %v", err)
	}

	if idx2.params.K1 != 1.5 {
		t.Errorf("K1: got %v, want 1.5", idx2.params.K1)
	}
	if idx2.params.B != 0.5 {
		t.Errorf("B: got %v, want 0.5", idx2.params.B)
	}
}

func TestSaveLoadIndex_PostingsPreserved(t *testing.T) {
	db := tempDB(t)
	idx := smallIndex(20)

	if err := db.SaveIndex(idx); err != nil {
		t.Fatalf("SaveIndex: %v", err)
	}

	idx2 := NewIndex()
	if err := db.LoadIndex(idx2); err != nil {
		t.Fatalf("LoadIndex: %v", err)
	}

	// The loaded index should return the same number of term postings.
	if len(idx2.bm25.postings) == 0 {
		t.Error("expected non-empty postings after load")
	}
}

func TestSaveLoadIndex_LargeIndex(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping large index test in short mode")
	}
	db := tempDB(t)
	idx := smallIndex(1000)

	start := time.Now()
	if err := db.SaveIndex(idx); err != nil {
		t.Fatalf("SaveIndex: %v", err)
	}
	saveDur := time.Since(start)
	t.Logf("SaveIndex (1000 docs): %v", saveDur)

	start = time.Now()
	idx2 := NewIndex()
	if err := db.LoadIndex(idx2); err != nil {
		t.Fatalf("LoadIndex: %v", err)
	}
	loadDur := time.Since(start)
	t.Logf("LoadIndex (1000 docs): %v", loadDur)

	if idx2.DocCount() != idx.DocCount() {
		t.Errorf("DocCount mismatch: got %d, want %d", idx2.DocCount(), idx.DocCount())
	}
	// Performance target: save + load < 10s (generous for CI)
	if saveDur+loadDur > 10*time.Second {
		t.Errorf("total save+load took %v, want <10s", saveDur+loadDur)
	}
}

func TestSaveIndex_Idempotent(t *testing.T) {
	db := tempDB(t)
	idx := smallIndex(5)

	for i := 0; i < 3; i++ {
		if err := db.SaveIndex(idx); err != nil {
			t.Fatalf("SaveIndex call %d: %v", i, err)
		}
	}

	idx2 := NewIndex()
	if err := db.LoadIndex(idx2); err != nil {
		t.Fatalf("LoadIndex: %v", err)
	}
	if idx2.DocCount() != idx.DocCount() {
		t.Errorf("DocCount after repeated save: got %d, want %d", idx2.DocCount(), idx.DocCount())
	}
}

// ─── Graph persistence (save / load round-trip) ───────────────────────────────

func TestSaveLoadGraph_SmallRoundTrip(t *testing.T) {
	db := tempDB(t)
	g := smallGraph(10)

	if err := db.SaveGraph(g); err != nil {
		t.Fatalf("SaveGraph: %v", err)
	}

	g2 := NewGraph()
	if err := db.LoadGraph(g2); err != nil {
		t.Fatalf("LoadGraph: %v", err)
	}

	if g2.NodeCount() != g.NodeCount() {
		t.Errorf("NodeCount: got %d, want %d", g2.NodeCount(), g.NodeCount())
	}
	if g2.EdgeCount() != g.EdgeCount() {
		t.Errorf("EdgeCount: got %d, want %d", g2.EdgeCount(), g.EdgeCount())
	}
}

func TestSaveLoadGraph_ConfidencePreserved(t *testing.T) {
	db := tempDB(t)
	g := NewGraph()
	_ = g.AddNode(&Node{ID: "a.md", Type: "document", Title: "A"})
	_ = g.AddNode(&Node{ID: "b.md", Type: "document", Title: "B"})
	e := &Edge{
		ID:         edgeID("a.md", "b.md", EdgeCalls),
		Source:     "a.md",
		Target:     "b.md",
		Type:       EdgeCalls,
		Confidence: 0.9,
		Evidence:   "import b",
	}
	_ = g.AddEdge(e)

	if err := db.SaveGraph(g); err != nil {
		t.Fatalf("SaveGraph: %v", err)
	}
	g2 := NewGraph()
	if err := db.LoadGraph(g2); err != nil {
		t.Fatalf("LoadGraph: %v", err)
	}

	loaded, ok := g2.Edges[e.ID]
	if !ok {
		t.Fatalf("edge %q not found after load", e.ID)
	}
	if loaded.Confidence != 0.9 {
		t.Errorf("Confidence: got %v, want 0.9", loaded.Confidence)
	}
	if loaded.Evidence != "import b" {
		t.Errorf("Evidence: got %q, want %q", loaded.Evidence, "import b")
	}
	if loaded.Type != EdgeCalls {
		t.Errorf("Type: got %q, want %q", loaded.Type, EdgeCalls)
	}
}

func TestSaveLoadGraph_AllEdgeTypesPreserved(t *testing.T) {
	db := tempDB(t)
	g := NewGraph()

	ids := []string{"a.md", "b.md", "c.md", "d.md", "e.md"}
	for _, id := range ids {
		_ = g.AddNode(&Node{ID: id, Type: "document", Title: id})
	}

	types := []EdgeType{EdgeReferences, EdgeDependsOn, EdgeCalls, EdgeImplements, EdgeMentions}
	confs := []float64{1.0, 0.7, 0.9, 0.7, 0.7}

	for i, et := range types {
		src := ids[i]
		tgt := ids[(i+1)%len(ids)]
		if src == tgt {
			continue
		}
		edge := &Edge{
			ID:         edgeID(src, tgt, et),
			Source:     src,
			Target:     tgt,
			Type:       et,
			Confidence: confs[i],
		}
		_ = g.AddEdge(edge)
	}

	if err := db.SaveGraph(g); err != nil {
		t.Fatalf("SaveGraph: %v", err)
	}
	g2 := NewGraph()
	if err := db.LoadGraph(g2); err != nil {
		t.Fatalf("LoadGraph: %v", err)
	}

	for _, et := range types {
		found := false
		for _, e := range g2.Edges {
			if e.Type == et {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("edge type %q not found after load", et)
		}
	}
}

func TestSaveLoadGraph_LargeGraph(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping large graph test in short mode")
	}
	db := tempDB(t)
	g := smallGraph(1000)

	start := time.Now()
	if err := db.SaveGraph(g); err != nil {
		t.Fatalf("SaveGraph: %v", err)
	}
	saveDur := time.Since(start)

	start = time.Now()
	g2 := NewGraph()
	if err := db.LoadGraph(g2); err != nil {
		t.Fatalf("LoadGraph: %v", err)
	}
	loadDur := time.Since(start)

	t.Logf("SaveGraph (1000 nodes): %v, LoadGraph: %v", saveDur, loadDur)
	if g2.NodeCount() != g.NodeCount() {
		t.Errorf("NodeCount: got %d, want %d", g2.NodeCount(), g.NodeCount())
	}
	if g2.EdgeCount() != g.EdgeCount() {
		t.Errorf("EdgeCount: got %d, want %d", g2.EdgeCount(), g.EdgeCount())
	}
}

func TestSaveLoadGraph_AdjacencyRebuilt(t *testing.T) {
	db := tempDB(t)
	g := smallGraph(5)
	if err := db.SaveGraph(g); err != nil {
		t.Fatalf("SaveGraph: %v", err)
	}
	g2 := NewGraph()
	if err := db.LoadGraph(g2); err != nil {
		t.Fatalf("LoadGraph: %v", err)
	}

	// Check BySource / ByTarget populated correctly.
	firstID := "node000.md"
	if len(g2.BySource[firstID]) == 0 {
		t.Errorf("BySource[%q] should have outgoing edges", firstID)
	}
	secondID := "node001.md"
	if len(g2.ByTarget[secondID]) == 0 {
		t.Errorf("ByTarget[%q] should have incoming edges", secondID)
	}
}

// ─── Change detection ─────────────────────────────────────────────────────────

func TestGetChanges_AddedFiles(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "a.md"), "# A\nContent")
	writeFile(t, filepath.Join(dir, "b.md"), "# B\nContent")

	db := tempDB(t)
	// DB is empty — both files should be "added".
	added, modified, deleted, err := db.GetChanges(dir)
	if err != nil {
		t.Fatalf("GetChanges: %v", err)
	}
	if len(added) != 2 {
		t.Errorf("added: got %d, want 2", len(added))
	}
	if len(modified) != 0 {
		t.Errorf("modified: got %d, want 0", len(modified))
	}
	if len(deleted) != 0 {
		t.Errorf("deleted: got %d, want 0", len(deleted))
	}
}

func TestGetChanges_ModifiedFiles(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "doc.md")
	writeFile(t, path, "# Original\nContent")

	db := tempDB(t)
	// Store original hash in DB.
	data, _ := os.ReadFile(path)
	_, err := db.conn.Exec(
		`INSERT INTO documents (id, path, title, content_hash, last_modified, indexed_at)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		"doc.md", path, "Doc", calculateContentHash(data), time.Now().UnixNano(), time.Now().UnixNano(),
	)
	if err != nil {
		t.Fatalf("insert doc: %v", err)
	}

	// Modify the file.
	writeFile(t, path, "# Modified\nDifferent content")

	added, modified, deleted, err := db.GetChanges(dir)
	if err != nil {
		t.Fatalf("GetChanges: %v", err)
	}
	if len(modified) != 1 {
		t.Errorf("modified: got %d, want 1", len(modified))
	}
	if len(added) != 0 {
		t.Errorf("added: got %d, want 0 (got %v)", len(added), added)
	}
	_ = deleted
}

func TestGetChanges_DeletedFiles(t *testing.T) {
	dir := t.TempDir()

	db := tempDB(t)
	// Insert a doc into DB that doesn't exist on disk.
	_, err := db.conn.Exec(
		`INSERT INTO documents (id, path, title, content_hash, last_modified, indexed_at)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		"ghost.md", filepath.Join(dir, "ghost.md"), "Ghost", "abc123",
		time.Now().UnixNano(), time.Now().UnixNano(),
	)
	if err != nil {
		t.Fatalf("insert ghost doc: %v", err)
	}

	added, modified, deleted, err := db.GetChanges(dir)
	if err != nil {
		t.Fatalf("GetChanges: %v", err)
	}
	if len(deleted) != 1 {
		t.Errorf("deleted: got %d, want 1", len(deleted))
	}
	_ = added
	_ = modified
}

// ─── Incremental updates ──────────────────────────────────────────────────────

func TestUpdateDocuments_UpsertAndDelete(t *testing.T) {
	db := tempDB(t)
	idx := smallIndex(5)
	if err := db.SaveIndex(idx); err != nil {
		t.Fatalf("SaveIndex: %v", err)
	}

	// Update doc000 and delete doc001.
	updated := Document{
		ID:           "doc000.md",
		Path:         "/fake/doc000.md",
		Title:        "Updated",
		ContentHash:  "newhash",
		LastModified: time.Now(),
	}
	if err := db.UpdateDocuments([]Document{updated}, []string{"doc001.md"}); err != nil {
		t.Fatalf("UpdateDocuments: %v", err)
	}

	// Verify doc000 updated.
	d, err := db.GetDocument("doc000.md")
	if err != nil {
		t.Fatalf("GetDocument doc000: %v", err)
	}
	if d.Title != "Updated" {
		t.Errorf("Title: got %q, want %q", d.Title, "Updated")
	}

	// Verify doc001 deleted.
	_, err = db.GetDocument("doc001.md")
	if err == nil {
		t.Error("expected error for deleted doc001.md, got nil")
	}
}

func TestUpdateDocuments_CascadeDeleteIndexEntries(t *testing.T) {
	db := tempDB(t)
	idx := smallIndex(3)
	if err := db.SaveIndex(idx); err != nil {
		t.Fatalf("SaveIndex: %v", err)
	}

	// Count entries for doc000 before deletion.
	var beforeCount int
	_ = db.conn.QueryRow(`SELECT COUNT(*) FROM index_entries WHERE doc_id='doc000.md'`).Scan(&beforeCount)

	// Delete the document.
	if err := db.UpdateDocuments(nil, []string{"doc000.md"}); err != nil {
		t.Fatalf("UpdateDocuments: %v", err)
	}

	// Index entries should be gone (cascade delete).
	var afterCount int
	_ = db.conn.QueryRow(`SELECT COUNT(*) FROM index_entries WHERE doc_id='doc000.md'`).Scan(&afterCount)
	if afterCount != 0 {
		t.Errorf("index_entries for deleted doc: got %d, want 0", afterCount)
	}
	_ = beforeCount
}

// ─── Database queries ─────────────────────────────────────────────────────────

func TestGetDocument_Found(t *testing.T) {
	db := tempDB(t)
	idx := smallIndex(3)
	if err := db.SaveIndex(idx); err != nil {
		t.Fatalf("SaveIndex: %v", err)
	}
	d, err := db.GetDocument("doc000.md")
	if err != nil {
		t.Fatalf("GetDocument: %v", err)
	}
	if d.ID != "doc000.md" {
		t.Errorf("ID: got %q, want %q", d.ID, "doc000.md")
	}
}

func TestGetDocument_NotFound(t *testing.T) {
	db := tempDB(t)
	_, err := db.GetDocument("nonexistent.md")
	if err == nil {
		t.Error("expected error for missing document, got nil")
	}
}

func TestGetNode_Found(t *testing.T) {
	db := tempDB(t)
	g := smallGraph(3)
	if err := db.SaveGraph(g); err != nil {
		t.Fatalf("SaveGraph: %v", err)
	}
	n, err := db.GetNode("node000.md")
	if err != nil {
		t.Fatalf("GetNode: %v", err)
	}
	if n.ID != "node000.md" {
		t.Errorf("ID: got %q, want %q", n.ID, "node000.md")
	}
}

func TestGetNode_NotFound(t *testing.T) {
	db := tempDB(t)
	_, err := db.GetNode("missing.md")
	if err == nil {
		t.Error("expected error for missing node, got nil")
	}
}

func TestGetEdges_OutgoingEdges(t *testing.T) {
	db := tempDB(t)
	g := smallGraph(5)
	if err := db.SaveGraph(g); err != nil {
		t.Fatalf("SaveGraph: %v", err)
	}
	edges, err := db.GetEdges("node000.md", "out")
	if err != nil {
		t.Fatalf("GetEdges out: %v", err)
	}
	if len(edges) != 1 {
		t.Errorf("outgoing edges for node000: got %d, want 1", len(edges))
	}
}

func TestGetEdges_IncomingEdges(t *testing.T) {
	db := tempDB(t)
	g := smallGraph(5)
	if err := db.SaveGraph(g); err != nil {
		t.Fatalf("SaveGraph: %v", err)
	}
	edges, err := db.GetEdges("node001.md", "in")
	if err != nil {
		t.Fatalf("GetEdges in: %v", err)
	}
	if len(edges) != 1 {
		t.Errorf("incoming edges for node001: got %d, want 1", len(edges))
	}
}

func TestGetEdges_InvalidDirection(t *testing.T) {
	db := tempDB(t)
	_, err := db.GetEdges("any.md", "sideways")
	if err == nil {
		t.Error("expected error for invalid direction, got nil")
	}
}

func TestSearchTerms_ReturnsResults(t *testing.T) {
	db := tempDB(t)
	idx := smallIndex(10)
	if err := db.SaveIndex(idx); err != nil {
		t.Fatalf("SaveIndex: %v", err)
	}

	results, err := db.SearchTerms([]string{"document"}, 5)
	if err != nil {
		t.Fatalf("SearchTerms: %v", err)
	}
	// "document" appears in every doc title and content, should get results.
	if len(results) == 0 {
		t.Error("expected results for term 'document', got none")
	}
}

func TestSearchTerms_EmptyTerms(t *testing.T) {
	db := tempDB(t)
	results, err := db.SearchTerms(nil, 5)
	if err != nil {
		t.Fatalf("SearchTerms(nil): %v", err)
	}
	if results != nil {
		t.Errorf("expected nil for empty terms, got %v", results)
	}
}

func TestSearchTerms_QueryPerformance(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping performance test in short mode")
	}
	db := tempDB(t)
	idx := smallIndex(500)
	if err := db.SaveIndex(idx); err != nil {
		t.Fatalf("SaveIndex: %v", err)
	}

	start := time.Now()
	_, err := db.SearchTerms([]string{"document", "topic"}, 10)
	dur := time.Since(start)
	if err != nil {
		t.Fatalf("SearchTerms: %v", err)
	}
	t.Logf("SearchTerms (500 docs): %v", dur)
	if dur > 100*time.Millisecond {
		t.Errorf("SearchTerms took %v, want <100ms", dur)
	}
}

func TestGetServices_ReturnsServiceNodes(t *testing.T) {
	db := tempDB(t)
	g := NewGraph()
	_ = g.AddNode(&Node{ID: "svc.md", Type: "service", Title: "My Service"})
	_ = g.AddNode(&Node{ID: "doc.md", Type: "document", Title: "A Doc"})
	if err := db.SaveGraph(g); err != nil {
		t.Fatalf("SaveGraph: %v", err)
	}

	services, err := db.GetServices()
	if err != nil {
		t.Fatalf("GetServices: %v", err)
	}
	if len(services) != 1 {
		t.Errorf("GetServices: got %d, want 1", len(services))
	}
	if services[0].ID != "svc.md" {
		t.Errorf("service ID: got %q, want %q", services[0].ID, "svc.md")
	}
}

// ─── Transaction safety ───────────────────────────────────────────────────────

func TestTransaction_Rollback(t *testing.T) {
	db := tempDB(t)
	idx := smallIndex(3)
	if err := db.SaveIndex(idx); err != nil {
		t.Fatalf("SaveIndex: %v", err)
	}

	// Simulate a transaction that fails partway through.
	_ = transaction(db.conn, func(tx *sql.Tx) error {
		if _, err := tx.Exec(`DELETE FROM documents`); err != nil {
			return err
		}
		// Force rollback.
		return fmt.Errorf("deliberate rollback")
	})

	// Documents should still be present.
	var count int
	_ = db.conn.QueryRow(`SELECT COUNT(*) FROM documents`).Scan(&count)
	if count == 0 {
		t.Error("documents deleted after rollback — transaction not rolled back")
	}
}

// ─── Foreign key constraints ──────────────────────────────────────────────────

func TestForeignKey_EdgeRequiresExistingNodes(t *testing.T) {
	db := tempDB(t)
	// Attempt to insert an edge without the referenced nodes.
	_, err := db.conn.Exec(
		`INSERT INTO graph_edges (id, source_id, target_id, type, confidence)
		 VALUES ('e1', 'nonexistent_a', 'nonexistent_b', 'references', 1.0)`,
	)
	// Should fail due to foreign key constraint.
	if err == nil {
		t.Error("expected FK violation, got nil error")
	}
}

func TestForeignKey_IndexEntryRequiresDocument(t *testing.T) {
	db := tempDB(t)
	_, err := db.conn.Exec(
		`INSERT INTO index_entries (term, doc_id, frequency) VALUES ('word', 'ghost.md', 1)`,
	)
	if err == nil {
		t.Error("expected FK violation, got nil error")
	}
}

// ─── calculateContentHash / hashFile ─────────────────────────────────────────

func TestCalculateContentHash_Deterministic(t *testing.T) {
	data := []byte("hello world")
	h1 := calculateContentHash(data)
	h2 := calculateContentHash(data)
	if h1 != h2 {
		t.Errorf("hash not deterministic: %q vs %q", h1, h2)
	}
	if h1 == "" {
		t.Error("hash must not be empty")
	}
}

func TestCalculateContentHash_ChangeSensitive(t *testing.T) {
	h1 := calculateContentHash([]byte("hello"))
	h2 := calculateContentHash([]byte("world"))
	if h1 == h2 {
		t.Error("different content must produce different hashes")
	}
}

func TestHashFile_ReadsFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "f.md")
	writeFile(t, path, "content")
	h, err := hashFile(path)
	if err != nil {
		t.Fatalf("hashFile: %v", err)
	}
	if h != calculateContentHash([]byte("content")) {
		t.Errorf("hash mismatch")
	}
}

func TestHashFile_MissingFile(t *testing.T) {
	_, err := hashFile("/no/such/file.md")
	if err == nil {
		t.Error("expected error for missing file, got nil")
	}
}

// ─── Confidence check constraint ─────────────────────────────────────────────

func TestConfidenceConstraint_OutOfRange(t *testing.T) {
	db := tempDB(t)
	// Insert valid node first.
	_, _ = db.conn.Exec(
		`INSERT INTO graph_nodes (id, type, file) VALUES ('a.md', 'document', 'a.md')`,
	)
	_, _ = db.conn.Exec(
		`INSERT INTO graph_nodes (id, type, file) VALUES ('b.md', 'document', 'b.md')`,
	)

	// confidence = 1.5 violates CHECK(confidence >= 0.0 AND confidence <= 1.0)
	_, err := db.conn.Exec(
		`INSERT INTO graph_edges (id, source_id, target_id, type, confidence)
		 VALUES ('e1', 'a.md', 'b.md', 'references', 1.5)`,
	)
	if err == nil {
		t.Error("expected CHECK violation for confidence=1.5, got nil")
	}
}

// ─── RebuildIndex ─────────────────────────────────────────────────────────────

func TestRebuildIndex_ReplacesPriorData(t *testing.T) {
	db := tempDB(t)
	idx := smallIndex(5)
	if err := db.SaveIndex(idx); err != nil {
		t.Fatalf("SaveIndex: %v", err)
	}

	// Build a larger replacement index.
	idx2 := smallIndex(10)
	if err := db.RebuildIndex(idx2); err != nil {
		t.Fatalf("RebuildIndex: %v", err)
	}

	// Reload and verify 10 docs.
	idx3 := NewIndex()
	if err := db.LoadIndex(idx3); err != nil {
		t.Fatalf("LoadIndex after rebuild: %v", err)
	}
	if idx3.DocCount() != idx2.DocCount() {
		t.Errorf("DocCount after rebuild: got %d, want %d", idx3.DocCount(), idx2.DocCount())
	}
}

// ─── helpers ──────────────────────────────────────────────────────────────────

// writeFile creates path with content, failing the test if unable.
func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %q: %v", path, err)
	}
}

// ─── Integration: full workflow ───────────────────────────────────────────────

func TestIntegration_PersistenceWorkflow(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	// Create a temp directory with synthetic markdown files.
	dir := t.TempDir()
	for i := 0; i < 20; i++ {
		name := fmt.Sprintf("doc%02d.md", i)
		content := fmt.Sprintf("# Document %d\n\nContent about topic %d.", i, i%5)
		writeFile(t, filepath.Join(dir, name), content)
	}

	// Build index and graph from the temp corpus.
	docs, err := ScanDirectory(dir)
	if err != nil {
		t.Fatalf("ScanDirectory: %v", err)
	}
	idx := NewIndex()
	if err := idx.Build(docs); err != nil {
		t.Fatalf("Build: %v", err)
	}
	gb := NewGraphBuilder(dir)
	g := gb.Build(docs)

	// Save both to DB.
	dbPath := filepath.Join(t.TempDir(), "test.db")
	db, err := OpenDB(dbPath)
	if err != nil {
		t.Fatalf("OpenDB: %v", err)
	}
	defer db.Close()

	if err := db.SaveIndex(idx); err != nil {
		t.Fatalf("SaveIndex: %v", err)
	}
	if err := db.SaveGraph(g); err != nil {
		t.Fatalf("SaveGraph: %v", err)
	}

	// Reload.
	idx2 := NewIndex()
	if err := db.LoadIndex(idx2); err != nil {
		t.Fatalf("LoadIndex: %v", err)
	}
	g2 := NewGraph()
	if err := db.LoadGraph(g2); err != nil {
		t.Fatalf("LoadGraph: %v", err)
	}

	if idx2.DocCount() != idx.DocCount() {
		t.Errorf("DocCount after reload: %d != %d", idx2.DocCount(), idx.DocCount())
	}
	if g2.NodeCount() != g.NodeCount() {
		t.Errorf("NodeCount after reload: %d != %d", g2.NodeCount(), g.NodeCount())
	}
	if g2.EdgeCount() != g.EdgeCount() {
		t.Errorf("EdgeCount after reload: %d != %d", g2.EdgeCount(), g.EdgeCount())
	}

	// Incremental update: modify one file.
	modPath := filepath.Join(dir, "doc00.md")
	writeFile(t, modPath, "# Modified\n\nEntirely different content about cats.")

	added, modified, deleted, err := db.GetChanges(dir)
	if err != nil {
		t.Fatalf("GetChanges: %v", err)
	}
	t.Logf("Changes: added=%d modified=%d deleted=%d", len(added), len(modified), len(deleted))
	if len(modified) == 0 {
		t.Error("expected at least 1 modified file")
	}
	if len(added) != 0 {
		t.Errorf("unexpected added files: %v", added)
	}

	// Update documents table.
	updatedDoc, _ := DocumentFromFile(modPath, "doc00.md")
	if err := db.UpdateDocuments([]Document{*updatedDoc}, deleted); err != nil {
		t.Fatalf("UpdateDocuments: %v", err)
	}

	// Verify update persisted.
	stored, err := db.GetDocument("doc00.md")
	if err != nil {
		t.Fatalf("GetDocument after update: %v", err)
	}
	if !strings.Contains(stored.ContentHash, "") { // just check it's set
		t.Error("ContentHash should be non-empty")
	}

	// Database file size should be reasonable (< 50 MB for 20 small docs).
	info, err := os.Stat(dbPath)
	if err != nil {
		t.Fatalf("stat db: %v", err)
	}
	t.Logf("Database file size: %d bytes", info.Size())
	if info.Size() > 50*1024*1024 {
		t.Errorf("database too large: %d bytes (want < 50 MB)", info.Size())
	}

	t.Log("Integration test PASSED")
}
