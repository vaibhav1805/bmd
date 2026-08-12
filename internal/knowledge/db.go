// Package knowledge — SQLite persistence layer for indexes and knowledge graphs.
//
// Schema overview:
//
//	documents     — one row per markdown file (id/path), used only for
//	                staleness detection (IsIndexStale) — see the note below.
//	graph_nodes   — knowledge graph vertices (id, type, file, title, metadata)
//	graph_edges   — knowledge graph directed edges (source, target, type, confidence, evidence)
//	metadata      — key/value store used for schema versioning + built_at
//
// bmd's BM25 search index itself is never persisted or reloaded from
// SQLite — bmd query and SearchAllDocuments always rebuild it in-memory
// from the markdown files on disk on every run (a full rescan is required
// regardless, since chunk content is never stored in SQLite — only
// metadata). This package's job is limited to: (1) tracking whether that
// in-memory rebuild is necessary at all (IsIndexStale, via documents +
// built_at), and (2) persisting the knowledge graph, which — unlike the
// search index — genuinely is loaded back from SQLite (see LoadGraph,
// used by `bmd graph` and the TUI graph view). An earlier version of this
// package also persisted BM25 postings (inverted index term→doc
// frequencies) to support reloading a saved index, but nothing ever
// called that reload path in practice — SaveIndex wrote it on every index
// run and nothing ever read it back — so it was pure write-only
// overhead. Removed in the SchemaVersion 3 migration (migrateV2ToV3).
//
// All multi-step writes are wrapped in transactions to guarantee atomicity.
// Foreign keys with ON DELETE CASCADE ensure referential integrity when
// documents or nodes are removed.
//
// Usage:
//
//	db, err := OpenDB("/path/to/bmd.db")
//	if err != nil { ... }
//	defer db.Close()
//
//	// Record that the index was (re)built, for future staleness checks.
//	if err := db.SaveIndex(idx); err != nil { ... }
//
//	// Later, before searching: rebuild in-memory if stale.
//	stale, err := db.IsIndexStale(rootDir)
package knowledge

import (
	"crypto/md5" //nolint:gosec // used for file change detection, not security
	"database/sql"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "modernc.org/sqlite" // register "sqlite" driver
)

// SchemaVersion is incremented each time the database schema changes.
// Migrations run automatically in Migrate() when an older database is opened.
const SchemaVersion = 3

// Database wraps an open SQLite connection and provides domain-level
// read/write operations for indexes and knowledge graphs.
//
// Zero value is NOT valid; always create via OpenDB or NewDatabase.
type Database struct {
	path string
	conn *sql.DB
}

// ─── construction ────────────────────────────────────────────────────────────

// OpenDB opens (or creates) the SQLite database at path, initialises the
// schema, and runs any outstanding migrations.
//
// Equivalent to NewDatabase(path) followed by Initialize() and Migrate().
func OpenDB(path string) (*Database, error) {
	db, err := NewDatabase(path)
	if err != nil {
		return nil, err
	}
	if err := db.Initialize(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("knowledge.OpenDB: initialize: %w", err)
	}
	if err := db.Migrate(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("knowledge.OpenDB: migrate: %w", err)
	}
	return db, nil
}

// NewDatabase opens the SQLite database at path without schema initialisation.
// Call Initialize() and Migrate() before use, or use OpenDB for convenience.
func NewDatabase(path string) (*Database, error) {
	// Ensure the parent directory exists.
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, fmt.Errorf("knowledge.NewDatabase: mkdir: %w", err)
	}

	conn, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("knowledge.NewDatabase: open %q: %w", path, err)
	}

	// SQLite supports only one writer at a time; WAL mode improves concurrency.
	if _, err := conn.Exec("PRAGMA journal_mode=WAL"); err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("knowledge.NewDatabase: enable WAL: %w", err)
	}
	// Enforce foreign key constraints (disabled by default in SQLite).
	if _, err := conn.Exec("PRAGMA foreign_keys=ON"); err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("knowledge.NewDatabase: enable foreign_keys: %w", err)
	}

	return &Database{path: path, conn: conn}, nil
}

// Close closes the underlying database connection.
func (db *Database) Close() error {
	return db.conn.Close()
}

// ─── schema initialisation ────────────────────────────────────────────────────

// schemaSQL is the complete DDL for SchemaVersion 1.
// Each statement is idempotent (CREATE TABLE IF NOT EXISTS / CREATE INDEX IF
// NOT EXISTS) so Initialize may be called on an already-initialised database.
const schemaSQL = `
-- documents: one row per markdown file in the indexed corpus. Only id is
-- actually read back (by IsIndexStale, to detect added/removed files —
-- see GetChanges/UpdateDocuments too). The rest (title, content_hash,
-- last_modified, indexed_at) are write-only today: kept because trimming
-- them would need a real column-drop migration for existing databases for
-- negligible benefit (a few bytes/row), not because anything currently
-- reads them back. GetDocument can still query the full row.
CREATE TABLE IF NOT EXISTS documents (
  id            TEXT    PRIMARY KEY,
  path          TEXT    NOT NULL UNIQUE,
  title         TEXT,
  content_hash  TEXT    NOT NULL,
  last_modified INTEGER NOT NULL,
  indexed_at    INTEGER NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_documents_path ON documents(path);

-- graph_nodes: vertices in the knowledge graph.
-- metadata is a JSON object (e.g. {"heading_level": 1, "line_range": [1,40]}).
CREATE TABLE IF NOT EXISTS graph_nodes (
  id       TEXT PRIMARY KEY,
  type     TEXT NOT NULL,
  file     TEXT NOT NULL,
  title    TEXT,
  content  TEXT,
  metadata TEXT
);
CREATE INDEX IF NOT EXISTS idx_nodes_type ON graph_nodes(type);

-- graph_edges: directed edges in the knowledge graph.
-- confidence must be in [0.0, 1.0] (enforced by CHECK constraint).
CREATE TABLE IF NOT EXISTS graph_edges (
  id         TEXT    PRIMARY KEY,
  source_id  TEXT    NOT NULL,
  target_id  TEXT    NOT NULL,
  type       TEXT    NOT NULL,
  confidence REAL    NOT NULL CHECK (confidence >= 0.0 AND confidence <= 1.0),
  evidence   TEXT,
  FOREIGN KEY(source_id) REFERENCES graph_nodes(id) ON DELETE CASCADE,
  FOREIGN KEY(target_id) REFERENCES graph_nodes(id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_edges_source ON graph_edges(source_id);
CREATE INDEX IF NOT EXISTS idx_edges_target ON graph_edges(target_id);

-- metadata: arbitrary key/value pairs (schema version, timestamps, etc.).
CREATE TABLE IF NOT EXISTS metadata (
  key   TEXT PRIMARY KEY,
  value TEXT NOT NULL
);
`

// Initialize creates all tables and indexes if they do not yet exist.
// This method is idempotent — calling it on an existing schema is safe.
func (db *Database) Initialize() error {
	if err := db.execMulti(schemaSQL); err != nil {
		return fmt.Errorf("knowledge.Database.Initialize: %w", err)
	}

	// Write the schema version if it is not already present.
	_, err := db.conn.Exec(
		`INSERT OR IGNORE INTO metadata (key, value) VALUES ('schema_version', ?)`,
		fmt.Sprintf("%d", SchemaVersion),
	)
	if err != nil {
		return fmt.Errorf("knowledge.Database.Initialize: write schema_version: %w", err)
	}
	return nil
}

// execMulti splits sql on ";" and executes each non-empty statement.
// This is required because database/sql does not support multi-statement Exec.
func (db *Database) execMulti(sql string) error {
	for _, stmt := range strings.Split(sql, ";") {
		stmt = strings.TrimSpace(stmt)
		if stmt == "" {
			continue
		}
		if _, err := db.conn.Exec(stmt); err != nil {
			return fmt.Errorf("exec %q: %w", stmt[:min(len(stmt), 60)], err)
		}
	}
	return nil
}

// min returns the smaller of a and b (int).
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// ─── schema version & migrations ─────────────────────────────────────────────

// GetVersion returns the schema version stored in the metadata table.
// Returns 0 if the version has not been recorded yet.
func (db *Database) GetVersion() int {
	var v string
	err := db.conn.QueryRow(
		`SELECT value FROM metadata WHERE key='schema_version'`,
	).Scan(&v)
	if err != nil {
		return 0
	}
	var n int
	fmt.Sscanf(v, "%d", &n)
	return n
}

// Migrate inspects the stored schema version and runs any applicable
// migration functions in order.  Migrations are idempotent by design.
func (db *Database) Migrate() error {
	current := db.GetVersion()

	if current < 2 {
		if err := db.migrateV1ToV2(); err != nil {
			return fmt.Errorf("knowledge.Database.Migrate: v1→v2: %w", err)
		}
	}

	if current < 3 {
		if err := db.migrateV2ToV3(); err != nil {
			return fmt.Errorf("knowledge.Database.Migrate: v2→v3: %w", err)
		}
	}

	// Ensure the stored version reflects the latest schema.
	if current < SchemaVersion {
		if _, err := db.conn.Exec(
			`INSERT OR REPLACE INTO metadata (key, value) VALUES ('schema_version', ?)`,
			fmt.Sprintf("%d", SchemaVersion),
		); err != nil {
			return fmt.Errorf("knowledge.Database.Migrate: update version: %w", err)
		}
	}
	return nil
}

// migrateV1ToV2 adds chunk metadata columns to index_entries.
// These columns are nullable and default to NULL for pre-existing rows.
func (db *Database) migrateV1ToV2() error {
	alterStatements := []string{
		`ALTER TABLE index_entries ADD COLUMN chunk_id     TEXT`,
		`ALTER TABLE index_entries ADD COLUMN heading_path TEXT`,
		`ALTER TABLE index_entries ADD COLUMN start_line   INTEGER`,
		`ALTER TABLE index_entries ADD COLUMN end_line     INTEGER`,
	}
	for _, stmt := range alterStatements {
		if _, err := db.conn.Exec(stmt); err != nil {
			// SQLite returns an error if the column already exists (from a CREATE TABLE
			// that included the column on a fresh database).  Ignore "duplicate column"
			// errors so Initialize() + Migrate() is idempotent.
			if !strings.Contains(err.Error(), "duplicate column") {
				return fmt.Errorf("exec %q: %w", stmt, err)
			}
		}
	}
	return nil
}

// migrateV2ToV3 drops the BM25 postings persistence tables (index_entries,
// bm25_stats). Nothing in the codebase ever read them back in practice —
// SaveIndex wrote them on every index run, but bmd query/SearchAllDocuments
// always rebuild the in-memory index from disk instead (a full rescan is
// required regardless, since chunk content itself was never persisted to
// SQLite — only metadata), so LoadIndex/SearchTerms were dead code kept in
// sync for no benefit. Safe to drop unconditionally: index_entries' only
// foreign key points OUT to documents, nothing references INTO either
// table, so this can't orphan or break anything else.
func (db *Database) migrateV2ToV3() error {
	stmts := []string{
		`DROP TABLE IF EXISTS index_entries`,
		`DROP TABLE IF EXISTS bm25_stats`,
	}
	for _, stmt := range stmts {
		if _, err := db.conn.Exec(stmt); err != nil {
			return fmt.Errorf("exec %q: %w", stmt, err)
		}
	}
	return nil
}

// GetSchemaVersion is an alias for GetVersion provided for the plan's API.
func (db *Database) GetSchemaVersion() int { return db.GetVersion() }

// ─── transaction helper ───────────────────────────────────────────────────────

// transaction executes fn inside a database transaction.  If fn returns an
// error the transaction is rolled back; otherwise it is committed.
func transaction(dbConn *sql.DB, fn func(*sql.Tx) error) error {
	tx, err := dbConn.Begin()
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	if err := fn(tx); err != nil {
		_ = tx.Rollback()
		return err
	}
	return tx.Commit()
}

// ─── index persistence ────────────────────────────────────────────────────────

const batchSize = 1000

// SaveIndex records that idx was (re)built, for future staleness checks
// (IsIndexStale) — it does NOT persist the BM25 postings/stats themselves.
// bmd query and SearchAllDocuments always rebuild the in-memory index from
// disk on every run regardless (chunk content is never stored in SQLite —
// only metadata — so a full rescan is required either way), so there was
// never a real reload path to serve; see the package doc comment and
// migrateV2ToV3. All changes are wrapped in a single transaction.
func (db *Database) SaveIndex(idx *Index) error {
	return transaction(db.conn, func(tx *sql.Tx) error {
		if _, err := tx.Exec(`DELETE FROM documents`); err != nil {
			return fmt.Errorf("clear documents: %w", err)
		}

		now := time.Now().UnixNano()

		// After chunk-level indexing, multiple indexedDoc entries share the
		// same relPath. Deduplicate: the documents table stores one row per
		// FILE (keyed by relPath), not one row per chunk.
		docs := idx.bm25.docs

		type fileDoc struct {
			relPath string
			path    string
			title   string
		}
		seenFiles := make(map[string]bool, len(docs))
		var fileDocs []fileDoc
		for _, d := range docs {
			if seenFiles[d.relPath] {
				continue
			}
			seenFiles[d.relPath] = true
			fileDocs = append(fileDocs, fileDoc{relPath: d.relPath, path: d.path, title: d.title})
		}

		for start := 0; start < len(fileDocs); start += batchSize {
			end := start + batchSize
			if end > len(fileDocs) {
				end = len(fileDocs)
			}
			for _, fd := range fileDocs[start:end] {
				// docMeta is keyed by the document ID (= RelPath for file-level docs).
				// After chunk indexing, the relPath is still the file-relative path.
				meta, hasMeta := idx.docMeta[fd.relPath]
				hash := meta.Hash
				lastMod := meta.LastModified
				if !hasMeta {
					hash = ""
					lastMod = now
				}
				_, err := tx.Exec(
					`INSERT OR REPLACE INTO documents
					 (id, path, title, content_hash, last_modified, indexed_at)
					 VALUES (?, ?, ?, ?, ?, ?)`,
					fd.relPath, fd.path, fd.title, hash, lastMod, now,
				)
				if err != nil {
					return fmt.Errorf("insert document %q: %w", fd.relPath, err)
				}
			}
		}

		// Record the build timestamp in metadata for staleness detection.
		if _, err := tx.Exec(
			`INSERT OR REPLACE INTO metadata (key, value) VALUES ('built_at', ?)`,
			fmt.Sprintf("%d", now),
		); err != nil {
			return fmt.Errorf("write built_at: %w", err)
		}

		return nil
	})
}

// GetIndexBuiltAt returns the Unix nanosecond timestamp when the index was last
// built, or zero if no timestamp is recorded (backwards-compatible with old
// databases that don't have the built_at metadata key).
func (db *Database) GetIndexBuiltAt() time.Time {
	var val string
	err := db.conn.QueryRow(
		`SELECT value FROM metadata WHERE key='built_at'`,
	).Scan(&val)
	if err != nil {
		return time.Time{} // zero time — treat as stale
	}
	var ns int64
	fmt.Sscanf(val, "%d", &ns)
	if ns == 0 {
		return time.Time{}
	}
	return time.Unix(0, ns)
}

// IsIndexStale returns true if any markdown file under root has been modified
// since the index was last built, or if files have been added or removed.
// It compares file modification times on disk against the stored built_at
// timestamp and the document list in the database.
//
// Returns true (stale) when:
//   - Any .md file has an mtime newer than the index build time
//   - A .md file exists on disk but is not in the database (new file)
//   - A document in the database no longer exists on disk (deleted file)
//   - The built_at timestamp is missing (old index, backwards-compatible)
func (db *Database) IsIndexStale(root string) (bool, error) {
	builtAt := db.GetIndexBuiltAt()
	if builtAt.IsZero() {
		return true, nil // no timestamp — consider stale
	}

	// Collect document IDs from the database.
	rows, err := db.conn.Query(`SELECT id FROM documents`)
	if err != nil {
		return false, fmt.Errorf("knowledge.Database.IsIndexStale: query documents: %w", err)
	}
	defer rows.Close()

	dbDocs := make(map[string]struct{})
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return false, fmt.Errorf("knowledge.Database.IsIndexStale: scan: %w", err)
		}
		dbDocs[id] = struct{}{}
	}
	if err := rows.Err(); err != nil {
		return false, fmt.Errorf("knowledge.Database.IsIndexStale: iterate: %w", err)
	}

	// Walk disk and compare.
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return false, fmt.Errorf("knowledge.Database.IsIndexStale: abs: %w", err)
	}

	diskDocs := make(map[string]struct{})
	stale := false

	walkErr := filepath.Walk(absRoot, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return nil // skip unreadable
		}
		if info.IsDir() {
			name := info.Name()
			if strings.HasPrefix(name, ".") && path != absRoot {
				return filepath.SkipDir
			}
			if _, skip := map[string]struct{}{
				"node_modules": {}, ".git": {}, ".svn": {},
			}[name]; skip {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(strings.ToLower(info.Name()), ".md") {
			return nil
		}
		rel, relErr := filepath.Rel(absRoot, path)
		if relErr != nil {
			return nil
		}
		id := filepath.ToSlash(rel)
		diskDocs[id] = struct{}{}

		// New file?
		if _, inDB := dbDocs[id]; !inDB {
			stale = true
		}
		// Modified after index build?
		if info.ModTime().After(builtAt) {
			stale = true
		}
		return nil
	})
	if walkErr != nil {
		return false, fmt.Errorf("knowledge.Database.IsIndexStale: walk: %w", walkErr)
	}

	// Deleted files?
	if !stale {
		for id := range dbDocs {
			if _, onDisk := diskDocs[id]; !onDisk {
				stale = true
				break
			}
		}
	}

	return stale, nil
}

// ─── graph persistence ────────────────────────────────────────────────────────

// SaveGraph serialises graph to the database, replacing any previously stored
// graph data.  All changes are wrapped in a single transaction.
func (db *Database) SaveGraph(graph *Graph) error {
	return transaction(db.conn, func(tx *sql.Tx) error {
		// Clear old data (cascade deletes edges automatically).
		if _, err := tx.Exec(`DELETE FROM graph_edges`); err != nil {
			return fmt.Errorf("clear graph_edges: %w", err)
		}
		if _, err := tx.Exec(`DELETE FROM graph_nodes`); err != nil {
			return fmt.Errorf("clear graph_nodes: %w", err)
		}

		// Insert nodes.
		nodes := make([]*Node, 0, len(graph.Nodes))
		for _, n := range graph.Nodes {
			nodes = append(nodes, n)
		}
		for start := 0; start < len(nodes); start += batchSize {
			end := start + batchSize
			if end > len(nodes) {
				end = len(nodes)
			}
			for _, n := range nodes[start:end] {
				_, err := tx.Exec(
					`INSERT OR REPLACE INTO graph_nodes (id, type, file, title, content, metadata)
					 VALUES (?, ?, ?, ?, ?, ?)`,
					n.ID, n.Type, n.ID /* file == ID */, n.Title, nil, nil,
				)
				if err != nil {
					return fmt.Errorf("insert graph_node %q: %w", n.ID, err)
				}
			}
		}

		// Insert edges.
		edges := make([]*Edge, 0, len(graph.Edges))
		for _, e := range graph.Edges {
			edges = append(edges, e)
		}
		for start := 0; start < len(edges); start += batchSize {
			end := start + batchSize
			if end > len(edges) {
				end = len(edges)
			}
			for _, e := range edges[start:end] {
				_, err := tx.Exec(
					`INSERT OR REPLACE INTO graph_edges
					 (id, source_id, target_id, type, confidence, evidence)
					 VALUES (?, ?, ?, ?, ?, ?)`,
					e.ID, e.Source, e.Target, string(e.Type), e.Confidence, e.Evidence,
				)
				if err != nil {
					return fmt.Errorf("insert graph_edge %q: %w", e.ID, err)
				}
			}
		}

		return nil
	})
}

// LoadGraph reconstructs graph from the database.
// graph is reset before loading; any existing data is discarded.
func (db *Database) LoadGraph(graph *Graph) error {
	// Reset the graph.
	*graph = *NewGraph()

	// Load nodes in deterministic order (sorted by ID).
	nRows, err := db.conn.Query(
		`SELECT id, type, file, title FROM graph_nodes ORDER BY id ASC`,
	)
	if err != nil {
		return fmt.Errorf("knowledge.Database.LoadGraph: query graph_nodes: %w", err)
	}
	defer nRows.Close()

	for nRows.Next() {
		var id, nodeType, file string
		var title sql.NullString
		if err := nRows.Scan(&id, &nodeType, &file, &title); err != nil {
			return fmt.Errorf("knowledge.Database.LoadGraph: scan node: %w", err)
		}
		n := &Node{ID: id, Type: nodeType, Title: title.String}
		graph.Nodes[id] = n
	}
	if err := nRows.Err(); err != nil {
		return fmt.Errorf("knowledge.Database.LoadGraph: iterate nodes: %w", err)
	}

	// Load edges in deterministic order (sorted by source, target, then ID).
	eRows, err := db.conn.Query(
		`SELECT id, source_id, target_id, type, confidence, evidence FROM graph_edges ORDER BY source_id ASC, target_id ASC, id ASC`,
	)
	if err != nil {
		return fmt.Errorf("knowledge.Database.LoadGraph: query graph_edges: %w", err)
	}
	defer eRows.Close()

	for eRows.Next() {
		var id, sourceID, targetID, edgeTypeStr string
		var confidence float64
		var evidence sql.NullString
		if err := eRows.Scan(&id, &sourceID, &targetID, &edgeTypeStr, &confidence, &evidence); err != nil {
			return fmt.Errorf("knowledge.Database.LoadGraph: scan edge: %w", err)
		}
		e := &Edge{
			ID:         id,
			Source:     sourceID,
			Target:     targetID,
			Type:       EdgeType(edgeTypeStr),
			Confidence: confidence,
			Evidence:   evidence.String,
		}
		graph.Edges[id] = e
		graph.BySource[sourceID] = append(graph.BySource[sourceID], e)
		graph.ByTarget[targetID] = append(graph.ByTarget[targetID], e)
	}
	if err := eRows.Err(); err != nil {
		return fmt.Errorf("knowledge.Database.LoadGraph: iterate edges: %w", err)
	}

	return nil
}

// ─── incremental update detection ────────────────────────────────────────────

// GetChanges scans root and compares the found files against the documents
// table.  It returns three lists:
//   - added:    files present on disk but not in the database.
//   - modified: files present in both but with a different content hash.
//   - deleted:  files in the database but not found on disk.
func (db *Database) GetChanges(root string) (added, modified, deleted []string, err error) {
	// Hash all markdown files under root.
	diskFiles := make(map[string]string) // relPath → hash
	walkErr := filepath.Walk(root, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return nil // skip unreadable paths
		}
		if info.IsDir() || !strings.HasSuffix(strings.ToLower(info.Name()), ".md") {
			return nil
		}
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return nil
		}
		rel, relErr := filepath.Rel(root, path)
		if relErr != nil {
			return nil
		}
		diskFiles[filepath.ToSlash(rel)] = calculateContentHash(data)
		return nil
	})
	if walkErr != nil {
		return nil, nil, nil, fmt.Errorf("knowledge.Database.GetChanges: walk: %w", walkErr)
	}

	// Load stored documents.
	rows, queryErr := db.conn.Query(`SELECT id, content_hash FROM documents`)
	if queryErr != nil {
		return nil, nil, nil, fmt.Errorf("knowledge.Database.GetChanges: query: %w", queryErr)
	}
	defer rows.Close()

	dbFiles := make(map[string]string) // id → hash
	for rows.Next() {
		var id, hash string
		if scanErr := rows.Scan(&id, &hash); scanErr != nil {
			return nil, nil, nil, fmt.Errorf("knowledge.Database.GetChanges: scan: %w", scanErr)
		}
		dbFiles[id] = hash
	}
	if err = rows.Err(); err != nil {
		return nil, nil, nil, fmt.Errorf("knowledge.Database.GetChanges: iterate: %w", err)
	}

	for relPath, diskHash := range diskFiles {
		if dbHash, found := dbFiles[relPath]; !found {
			added = append(added, relPath)
		} else if diskHash != dbHash {
			modified = append(modified, relPath)
		}
	}
	for id := range dbFiles {
		if _, found := diskFiles[id]; !found {
			deleted = append(deleted, id)
		}
	}

	return added, modified, deleted, nil
}

// UpdateDocuments performs a partial update of the documents table.
//
// docs contains documents to add or replace (by their ID).
// deletedIDs lists document IDs to remove.
// All changes are wrapped in a single transaction.
func (db *Database) UpdateDocuments(docs []Document, deletedIDs []string) error {
	return transaction(db.conn, func(tx *sql.Tx) error {
		now := time.Now().UnixNano()

		// Delete removed documents.
		for _, id := range deletedIDs {
			if _, err := tx.Exec(`DELETE FROM documents WHERE id=?`, id); err != nil {
				return fmt.Errorf("delete document %q: %w", id, err)
			}
		}

		// Upsert changed/added documents.
		for _, doc := range docs {
			_, err := tx.Exec(
				`INSERT OR REPLACE INTO documents
				 (id, path, title, content_hash, last_modified, indexed_at)
				 VALUES (?, ?, ?, ?, ?, ?)`,
				doc.ID, doc.Path, doc.Title, doc.ContentHash,
				doc.LastModified.UnixNano(), now,
			)
			if err != nil {
				return fmt.Errorf("upsert document %q: %w", doc.ID, err)
			}
		}

		return nil
	})
}

// ─── database queries ─────────────────────────────────────────────────────────

// GetDocument retrieves a document row by ID.
// Returns nil and a non-nil error when the document is not found.
func (db *Database) GetDocument(id string) (*Document, error) {
	var d Document
	var lastMod int64
	err := db.conn.QueryRow(
		`SELECT id, path, title, content_hash, last_modified FROM documents WHERE id=?`,
		id,
	).Scan(&d.ID, &d.Path, &d.Title, &d.ContentHash, &lastMod)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("knowledge.Database.GetDocument: %q not found", id)
	}
	if err != nil {
		return nil, fmt.Errorf("knowledge.Database.GetDocument: %w", err)
	}
	d.RelPath = filepath.FromSlash(id)
	d.LastModified = time.Unix(0, lastMod)
	return &d, nil
}

// GetNode retrieves a graph node by ID.
// Returns nil and a non-nil error when the node is not found.
func (db *Database) GetNode(id string) (*Node, error) {
	var n Node
	var title sql.NullString
	err := db.conn.QueryRow(
		`SELECT id, type, title FROM graph_nodes WHERE id=?`, id,
	).Scan(&n.ID, &n.Type, &title)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("knowledge.Database.GetNode: %q not found", id)
	}
	if err != nil {
		return nil, fmt.Errorf("knowledge.Database.GetNode: %w", err)
	}
	n.Title = title.String
	return &n, nil
}

// GetEdges returns outgoing ("out") or incoming ("in") edges for nodeID.
// direction must be "out" or "in"; any other value returns an error.
func (db *Database) GetEdges(nodeID string, direction string) ([]*Edge, error) {
	var query string
	switch direction {
	case "out":
		query = `SELECT id, source_id, target_id, type, confidence, evidence
		         FROM graph_edges WHERE source_id=?`
	case "in":
		query = `SELECT id, source_id, target_id, type, confidence, evidence
		         FROM graph_edges WHERE target_id=?`
	default:
		return nil, fmt.Errorf("knowledge.Database.GetEdges: direction must be 'in' or 'out', got %q", direction)
	}

	rows, err := db.conn.Query(query, nodeID)
	if err != nil {
		return nil, fmt.Errorf("knowledge.Database.GetEdges: %w", err)
	}
	defer rows.Close()

	var edges []*Edge
	for rows.Next() {
		var e Edge
		var evidence sql.NullString
		if err := rows.Scan(&e.ID, &e.Source, &e.Target, (*string)(&e.Type), &e.Confidence, &evidence); err != nil {
			return nil, fmt.Errorf("knowledge.Database.GetEdges: scan: %w", err)
		}
		e.Evidence = evidence.String
		edges = append(edges, &e)
	}
	return edges, rows.Err()
}

// GetServices returns all nodes whose type is "service".
func (db *Database) GetServices() ([]Node, error) {
	rows, err := db.conn.Query(
		`SELECT id, type, title FROM graph_nodes WHERE type='service'`,
	)
	if err != nil {
		return nil, fmt.Errorf("knowledge.Database.GetServices: %w", err)
	}
	defer rows.Close()

	var services []Node
	for rows.Next() {
		var n Node
		var title sql.NullString
		if err := rows.Scan(&n.ID, &n.Type, &title); err != nil {
			return nil, fmt.Errorf("knowledge.Database.GetServices: scan: %w", err)
		}
		n.Title = title.String
		services = append(services, n)
	}
	return services, rows.Err()
}

// ─── helpers ──────────────────────────────────────────────────────────────────

// calculateContentHash returns the hex-encoded MD5 digest of data.
// Used here for content-change detection (not cryptographic security).
func calculateContentHash(data []byte) string {
	h := md5.Sum(data) //nolint:gosec
	return hex.EncodeToString(h[:])
}

// hashFile reads the file at path and returns its MD5 content hash.
func hashFile(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("knowledge.hashFile: read %q: %w", path, err)
	}
	return calculateContentHash(data), nil
}
