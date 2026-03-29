package search

import (
	"encoding/json"
	"log"
	"os"
	"path/filepath"
	"time"
)

const maxHistorySize = 20

// historyFile is the JSON structure persisted to disk.
type historyFile struct {
	Queries     []string `json:"queries"`
	LastUpdated int64    `json:"last_updated"`
}

// SearchHistory tracks recent search queries with up/down arrow recall.
type SearchHistory struct {
	queries  []string // FIFO list of queries (max 20)
	current  int      // current navigation index (-1 = not navigating)
	filePath string   // path to JSON persistence file
}

// NewSearchHistory creates a SearchHistory that persists to filePath.
// If filePath is empty, history works in-memory only (no persistence).
func NewSearchHistory(filePath string) *SearchHistory {
	return &SearchHistory{
		current:  -1,
		filePath: filePath,
	}
}

// DefaultHistoryPath returns ~/.config/bmd/search-history.json.
func DefaultHistoryPath() string {
	cfgDir, err := os.UserConfigDir()
	if err != nil {
		cfgDir = filepath.Join(os.Getenv("HOME"), ".config")
	}
	return filepath.Join(cfgDir, "bmd", "search-history.json")
}

// Load reads the history file from disk. If the file doesn't exist or is
// corrupted, the history starts empty.
func (h *SearchHistory) Load() error {
	if h.filePath == "" {
		return nil
	}
	data, err := os.ReadFile(h.filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	var f historyFile
	if err := json.Unmarshal(data, &f); err != nil {
		log.Printf("search history: corrupted file %s, starting fresh", h.filePath)
		h.queries = nil
		return nil
	}
	h.queries = f.Queries
	if len(h.queries) > maxHistorySize {
		h.queries = h.queries[len(h.queries)-maxHistorySize:]
	}
	return nil
}

// Save writes the current history to disk.
func (h *SearchHistory) Save() error {
	if h.filePath == "" {
		return nil
	}
	dir := filepath.Dir(h.filePath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	f := historyFile{
		Queries:     h.queries,
		LastUpdated: time.Now().Unix(),
	}
	data, err := json.MarshalIndent(f, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(h.filePath, data, 0o644)
}

// Push adds a query to the history. Empty queries and consecutive duplicates
// are ignored. Trims to maxHistorySize.
func (h *SearchHistory) Push(query string) {
	if query == "" {
		return
	}
	// Skip consecutive duplicate
	if len(h.queries) > 0 && h.queries[len(h.queries)-1] == query {
		return
	}
	h.queries = append(h.queries, query)
	if len(h.queries) > maxHistorySize {
		h.queries = h.queries[len(h.queries)-maxHistorySize:]
	}
	h.current = -1
}

// Prev returns the previous (older) query. On the first call after Reset(),
// returns the most recent query. Returns "" if history is empty or at the
// oldest entry.
func (h *SearchHistory) Prev() string {
	if len(h.queries) == 0 {
		return ""
	}
	if h.current == -1 {
		// Start navigating from the end (most recent)
		h.current = len(h.queries) - 1
	} else if h.current > 0 {
		h.current--
	}
	return h.queries[h.current]
}

// Next returns the next (newer) query. Returns "" if already past the newest
// entry (resets navigation).
func (h *SearchHistory) Next() string {
	if len(h.queries) == 0 || h.current == -1 {
		return ""
	}
	if h.current < len(h.queries)-1 {
		h.current++
		return h.queries[h.current]
	}
	// Past the newest: reset navigation
	h.current = -1
	return ""
}

// Clear wipes history in memory and on disk.
func (h *SearchHistory) Clear() error {
	h.queries = nil
	h.current = -1
	if h.filePath == "" {
		return nil
	}
	err := os.Remove(h.filePath)
	if os.IsNotExist(err) {
		return nil
	}
	return err
}

// Reset resets the navigation index without changing stored queries.
// Call this when starting a new search session.
func (h *SearchHistory) Reset() {
	h.current = -1
}

// Len returns the number of stored queries.
func (h *SearchHistory) Len() int {
	return len(h.queries)
}
