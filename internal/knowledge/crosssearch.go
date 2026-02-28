package knowledge

// SearchAllDocuments loads the BM25 index from rootPath (building it if missing)
// and executes a full-text search across all indexed markdown files.
//
// It reuses the existing openOrBuildIndex infrastructure from Phase 6.
// Returns SearchResult slice sorted by BM25 score descending.
// Returns an empty slice (not nil) when no documents match.
func SearchAllDocuments(rootPath, query string, topK int) ([]SearchResult, error) {
	if query == "" {
		return []SearchResult{}, nil
	}
	if topK <= 0 {
		topK = 50
	}

	dbPath := defaultDBPath(rootPath)
	db, err := openOrBuildIndex(rootPath, dbPath)
	if err != nil {
		return nil, err
	}
	defer db.Close() //nolint:errcheck

	idx := NewIndex()
	if err := db.LoadIndex(idx); err != nil {
		return nil, err
	}

	// Re-scan to populate content for snippet extraction.
	docs, scanErr := ScanDirectory(rootPath)
	if scanErr == nil && len(docs) > 0 {
		_ = idx.Build(docs)
	}

	return idx.Search(query, topK)
}
