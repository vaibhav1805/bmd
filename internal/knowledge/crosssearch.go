package knowledge

import (
	"os"
	"strings"
)

// GetContextSnippet reads the file at filePath and extracts up to maxChars
// characters of context centered around the first occurrence of query.
// Returns "..." padding at start/end when the snippet is truncated.
// If the query is not found, returns the first maxChars chars of the file.
func GetContextSnippet(filePath, query string, maxChars int) string {
	if filePath == "" || maxChars <= 0 {
		return ""
	}
	data, err := os.ReadFile(filePath)
	if err != nil {
		return ""
	}
	content := string(data)
	if content == "" {
		return ""
	}
	// Strip markdown heading markers for cleaner snippets.
	content = strings.ReplaceAll(content, "\r\n", "\n")

	// Find the first case-insensitive occurrence of query.
	lowerContent := strings.ToLower(content)
	lowerQuery := strings.ToLower(strings.TrimSpace(query))
	matchIdx := -1
	if lowerQuery != "" {
		matchIdx = strings.Index(lowerContent, lowerQuery)
	}

	runes := []rune(content)
	totalLen := len(runes)

	if matchIdx < 0 {
		// No match found — return first maxChars.
		if totalLen <= maxChars {
			snippet := collapseWhitespace(string(runes))
			return snippet
		}
		snippet := collapseWhitespace(string(runes[:maxChars]))
		return snippet + "..."
	}

	// Convert byte offset to rune offset.
	matchRuneIdx := len([]rune(content[:matchIdx]))
	queryRuneLen := len([]rune(query))

	// Center the snippet around the match.
	contextBefore := (maxChars - queryRuneLen) / 2
	if contextBefore < 0 {
		contextBefore = 0
	}

	start := matchRuneIdx - contextBefore
	if start < 0 {
		start = 0
	}
	end := start + maxChars
	if end > totalLen {
		end = totalLen
		start = end - maxChars
		if start < 0 {
			start = 0
		}
	}

	snippet := string(runes[start:end])
	snippet = collapseWhitespace(snippet)

	prefix := ""
	suffix := ""
	if start > 0 {
		prefix = "..."
	}
	if end < totalLen {
		suffix = "..."
	}

	return prefix + snippet + suffix
}

// collapseWhitespace replaces runs of whitespace (newlines, tabs, multiple spaces)
// with single spaces for clean snippet display.
func collapseWhitespace(s string) string {
	var b strings.Builder
	inSpace := false
	for _, r := range s {
		if r == '\n' || r == '\r' || r == '\t' || r == ' ' {
			if !inSpace {
				b.WriteByte(' ')
				inSpace = true
			}
		} else {
			b.WriteRune(r)
			inSpace = false
		}
	}
	return strings.TrimSpace(b.String())
}

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
