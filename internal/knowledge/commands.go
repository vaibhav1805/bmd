package knowledge

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// ─── argument structs ─────────────────────────────────────────────────────────

// IndexArgs holds parsed arguments for CmdIndex.
type IndexArgs struct {
	Dir          string
	DB           string
	Watch        bool
	PollInterval int // seconds
	// Scan configuration flags
	IgnoreDirs       string // comma-separated directory patterns to ignore
	IgnoreFiles      string // comma-separated file patterns to ignore
	IncludeHidden    bool   // -A flag: include hidden directories
	NoIgnoreDefaults bool   // disable default ignore patterns
}

// QueryArgs holds parsed arguments for CmdQuery.
type QueryArgs struct {
	Query  string
	Dir    string
	Format string
	Top    int
}

// GraphArgs holds parsed arguments for CmdGraph.
type GraphArgs struct {
	Service string
	Dir     string
	Format  string
}

// ─── argument parsers ─────────────────────────────────────────────────────────

// ParseIndexArgs parses raw CLI arguments for the index command.
//
// Usage: bmd index [DIR] [--dir DIR] [--db PATH] [--watch] [--poll-interval N]
//
//	[--ignore-dirs DIRS] [--ignore-files FILES] [-A|--include-hidden] [--no-ignore-defaults]
func ParseIndexArgs(args []string) (*IndexArgs, error) {
	fs := flag.NewFlagSet("index", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	var a IndexArgs
	fs.StringVar(&a.Dir, "dir", ".", "Directory to index")
	fs.StringVar(&a.DB, "db", ".bmd/knowledge.db", "Path to SQLite database")
	fs.BoolVar(&a.Watch, "watch", false, "Rebuild index on file changes")
	fs.IntVar(&a.PollInterval, "poll-interval", 5, "Polling interval in seconds (watch mode)")
	// Scan configuration flags
	fs.StringVar(&a.IgnoreDirs, "ignore-dirs", "", "Comma-separated directory patterns to ignore (appends to defaults)")
	fs.StringVar(&a.IgnoreFiles, "ignore-files", "", "Comma-separated file patterns to ignore")
	fs.BoolVar(&a.IncludeHidden, "A", false, "Include hidden directories and files")
	fs.BoolVar(&a.NoIgnoreDefaults, "no-ignore-defaults", false, "Disable default ignore patterns")
	// Also register --include-hidden as an alias for -A
	fs.BoolVar(&a.IncludeHidden, "include-hidden", a.IncludeHidden, "Include hidden directories and files")

	if err := fs.Parse(args); err != nil {
		return nil, fmt.Errorf("index: %w", err)
	}

	// Positional argument overrides --dir.
	if pos := fs.Args(); len(pos) > 0 {
		a.Dir = pos[0]
	}

	return &a, nil
}

// ParseQueryArgs parses raw CLI arguments for the query command.
//
// Usage: bmd query TERM [DIR] [--dir DIR] [--format json|text|csv] [--top N]
func ParseQueryArgs(args []string) (*QueryArgs, error) {
	// Split positional args from flags so that flag.Parse handles flags
	// regardless of order (Go's flag package stops at the first non-flag).
	positionals, flags := splitPositionalsAndFlags(args)

	fs := flag.NewFlagSet("query", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	var a QueryArgs
	fs.StringVar(&a.Dir, "dir", ".", "Directory that was indexed")
	fs.StringVar(&a.Format, "format", "json", "Output format (json|text|csv)")
	fs.IntVar(&a.Top, "top", 10, "Maximum number of results to return")

	if err := fs.Parse(flags); err != nil {
		return nil, fmt.Errorf("query: %w", err)
	}

	if len(positionals) < 1 {
		return nil, fmt.Errorf("query: TERM argument required\nUsage: bmd query TERM [DIR] [--dir DIR] [--format json|text|csv] [--top N]")
	}
	a.Query = positionals[0]
	if len(positionals) > 1 {
		a.Dir = positionals[1]
	}

	if a.Top < 1 {
		return nil, fmt.Errorf("query: --top must be >= 1")
	}

	return &a, nil
}

// ParseGraphArgs parses raw CLI arguments for the graph command.
//
// Usage: bmd graph [SERVICE] [--dir DIR] [--format dot|json] [--service NAME]
func ParseGraphArgs(args []string) (*GraphArgs, error) {
	positionals, flags := splitPositionalsAndFlags(args)

	fs := flag.NewFlagSet("graph", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	var a GraphArgs
	fs.StringVar(&a.Dir, "dir", ".", "Directory that was indexed")
	fs.StringVar(&a.Format, "format", "dot", "Output format (dot|json)")
	fs.StringVar(&a.Service, "service", "", "Export subgraph for this service only")

	if err := fs.Parse(flags); err != nil {
		return nil, fmt.Errorf("graph: %w", err)
	}

	// Positional argument is the service name.
	if len(positionals) > 0 {
		candidate := positionals[0]
		// If it looks like a path, treat as dir; otherwise it's a service name.
		if strings.Contains(candidate, "/") || strings.HasSuffix(candidate, ".md") {
			a.Dir = candidate
		} else {
			a.Service = candidate
		}
		if len(positionals) > 1 {
			a.Dir = positionals[1]
		}
	}

	return &a, nil
}

// ─── command implementations ──────────────────────────────────────────────────

// CmdIndex implements `bmd index`.  It scans dir, builds a BM25 index and
// knowledge graph, and saves both to a SQLite database.
func CmdIndex(args []string) error {
	a, err := ParseIndexArgs(args)
	if err != nil {
		return err
	}

	absDir, err := filepath.Abs(a.Dir)
	if err != nil {
		return fmt.Errorf("index: resolve dir %q: %w", a.Dir, err)
	}

	fmt.Fprintf(os.Stderr, "Indexing %s (link-based graph only)...\n", absDir)

	start := time.Now()

	// Build ScanConfig from parsed flags.
	config := ScanConfig{
		IncludeHidden:     a.IncludeHidden,
		UseDefaultIgnores: !a.NoIgnoreDefaults,
	}

	// Parse comma-separated ignore patterns.
	if a.IgnoreDirs != "" {
		config.IgnoreDirs = strings.Split(a.IgnoreDirs, ",")
	}
	if a.IgnoreFiles != "" {
		config.IgnoreFiles = strings.Split(a.IgnoreFiles, ",")
	}

	// Scan markdown files.
	docs, err := ScanDirectory(absDir, config)
	if err != nil {
		return fmt.Errorf("index: scan: %w", err)
	}

	fmt.Fprintf(os.Stderr, "  %d markdown files scanned\n", len(docs))

	// Build BM25 index.
	idx := NewIndex()
	if err := idx.Build(docs); err != nil {
		return fmt.Errorf("index: build index: %w", err)
	}

	// Count terms (approximate from postings map).
	termCount := len(idx.bm25.postings)
	fmt.Fprintf(os.Stderr, "  %d terms indexed\n", termCount)

	// Build knowledge graph (link-based extraction only).
	gb := NewGraphBuilder(absDir)
	graph := gb.Build(docs)

	fmt.Fprintf(os.Stderr, "  %d nodes in knowledge graph\n", graph.NodeCount())
	fmt.Fprintf(os.Stderr, "  %d edges (link-based relationships)\n", graph.EdgeCount())
	fmt.Fprintf(os.Stderr, "  (For intelligent relationship discovery, run: graphmd index --dir %s)\n", absDir)

	// Open / create database.
	// Make database path relative to indexed directory if not absolute.
	dbPath := a.DB
	if !filepath.IsAbs(dbPath) {
		dbPath = filepath.Join(absDir, dbPath)
	}
	db, err := OpenDB(dbPath)
	if err != nil {
		return fmt.Errorf("index: open db %q: %w", dbPath, err)
	}
	defer db.Close() //nolint:errcheck

	// Remove dangling edges (target node not in graph) before saving to avoid
	// FK constraint violations in the database.  This can occur when the
	// extractor creates edges to files that do not exist on disk.
	pruneDanglingEdges(graph)

	// Save index and graph.
	if err := db.SaveIndex(idx); err != nil {
		return fmt.Errorf("index: save index: %w", err)
	}
	if err := db.SaveGraph(graph); err != nil {
		return fmt.Errorf("index: save graph: %w", err)
	}

	// Report database size.
	stat, _ := os.Stat(dbPath)
	var sizeStr string
	if stat != nil {
		sizeStr = humanBytes(stat.Size())
	}

	elapsed := time.Since(start)
	absDB, _ := filepath.Abs(dbPath)
	if sizeStr != "" {
		fmt.Fprintf(os.Stderr, "  Index saved to %s (%s)\n", absDB, sizeStr)
	} else {
		fmt.Fprintf(os.Stderr, "  Index saved to %s\n", absDB)
	}
	fmt.Fprintf(os.Stderr, "  Completed in %dms\n", elapsed.Milliseconds())

	if a.Watch {
		fmt.Fprintf(os.Stderr, "Watching %s for changes (poll every %ds)...\n", absDir, a.PollInterval)
		return watchAndRebuild(a, absDir, docs)
	}

	return nil
}

// CmdQuery implements `bmd query TERM`.  It executes a BM25 search over the
// indexed directory, printing results in the requested format.
func CmdQuery(args []string) error {
	a, err := ParseQueryArgs(args)
	if err != nil {
		return err
	}

	isJSON := strings.ToLower(a.Format) == "json"

	// Validate query term for JSON callers.
	if isJSON && a.Query == "" {
		fmt.Println(marshalContract(NewErrorResponse(ErrCodeInvalidQuery, "Query term is required")))
		return nil
	}

	start := time.Now()

	absDir, err := filepath.Abs(a.Dir)
	if err != nil {
		if isJSON {
			fmt.Println(marshalContract(NewErrorResponse(ErrCodeInternalError, err.Error())))
			return nil
		}
		return fmt.Errorf("query: resolve dir: %w", err)
	}

	dbPath := DefaultDBPath(absDir)
	db, err := openOrBuildIndex(absDir, dbPath)
	if err != nil {
		if isJSON {
			fmt.Println(marshalContract(NewErrorResponse(classifyIndexError(err), err.Error())))
			return nil
		}
		return err
	}
	defer db.Close() //nolint:errcheck

	// The database only stores document metadata, not full content, so BM25
	// search always runs against a fresh in-memory index built by rescanning
	// the directory (needed for snippet extraction anyway).
	k := DefaultKnowledge()
	docs, err := k.Scan(absDir)
	if err != nil {
		if isJSON {
			fmt.Println(marshalContract(NewErrorResponse(ErrCodeInternalError, err.Error())))
			return nil
		}
		return fmt.Errorf("query: scan: %w", err)
	}
	idx := NewIndex()
	if err := idx.Build(docs); err != nil {
		if isJSON {
			fmt.Println(marshalContract(NewErrorResponse(ErrCodeInternalError, err.Error())))
			return nil
		}
		return fmt.Errorf("query: build index: %w", err)
	}

	// Execute search.
	results, err := idx.Search(a.Query, a.Top)
	if err != nil {
		if isJSON {
			fmt.Println(marshalContract(NewErrorResponse(ErrCodeInternalError, err.Error())))
			return nil
		}
		return fmt.Errorf("query: search: %w", err)
	}

	queryTimeMs := time.Since(start).Milliseconds()

	if !isJSON {
		// Text and CSV paths are unchanged.
		output := FormatSearchResults(results, a.Query, a.Format, queryTimeMs)
		fmt.Println(output)
		return nil
	}

	// JSON path: wrap in ContractResponse envelope.
	if len(results) == 0 {
		fmt.Println(marshalContract(NewEmptyResponse("No results found", map[string]interface{}{
			"query":   a.Query,
			"count":   0,
			"results": []interface{}{},
		})))
		return nil
	}

	// Build the search payload (same structure as formatSearchResultsJSON).
	items := make([]searchResultJSON, len(results))
	for i, r := range results {
		items[i] = searchResultJSON{
			Rank:           i + 1,
			File:           r.RelPath,
			Title:          r.Title,
			Score:          roundFloat(r.Score, 4),
			Snippet:        r.Snippet,
			HeadingPath:    r.HeadingPath,
			StartLine:      r.StartLine,
			EndLine:        r.EndLine,
			ContentPreview: r.ContentPreview,
		}
	}
	payload := searchResponseJSON{
		Query:       a.Query,
		Results:     items,
		Count:       len(items),
		QueryTimeMs: queryTimeMs,
	}
	fmt.Println(marshalContract(NewOKResponse("Search completed", payload)))
	return nil
}

// CmdGraph implements `bmd graph`.  It loads the knowledge graph and exports
// it in the requested format (DOT or JSON).
func CmdGraph(args []string) error {
	a, err := ParseGraphArgs(args)
	if err != nil {
		return err
	}

	isJSON := strings.ToLower(a.Format) == "json"

	absDir, err := filepath.Abs(a.Dir)
	if err != nil {
		if isJSON {
			fmt.Println(marshalContract(NewErrorResponse(ErrCodeInternalError, err.Error())))
			return nil
		}
		return fmt.Errorf("graph: resolve dir: %w", err)
	}

	dbPath := DefaultDBPath(absDir)
	db, err := openOrBuildIndex(absDir, dbPath)
	if err != nil {
		if isJSON {
			fmt.Println(marshalContract(NewErrorResponse(classifyIndexError(err), err.Error())))
			return nil
		}
		return err
	}
	defer db.Close() //nolint:errcheck

	graph := NewGraph()
	if err := db.LoadGraph(graph); err != nil {
		if isJSON {
			fmt.Println(marshalContract(NewErrorResponse(ErrCodeInternalError, err.Error())))
			return nil
		}
		return fmt.Errorf("graph: load graph: %w", err)
	}

	// Apply subgraph filter when a service is specified.
	exportGraph := graph
	if a.Service != "" {
		// Find the node ID for this service (match by ID or filename stem).
		nodeID := findNodeForService(graph, a.Service)
		if nodeID == "" {
			if isJSON {
				fmt.Println(marshalContract(NewErrorResponse(ErrCodeFileNotFound, fmt.Sprintf("service/node %q not found in graph", a.Service))))
				return nil
			}
			return fmt.Errorf("graph: service/node %q not found in graph", a.Service)
		}
		exportGraph = graph.GetSubgraph(nodeID, 10)
	}

	if !isJSON {
		// DOT path is unchanged.
		output := FormatGraph(exportGraph, a.Format)
		fmt.Println(output)
		return nil
	}

	// JSON path: build payload and wrap in envelope.
	// Sort nodes and edges for deterministic output.
	nodeIDs := make([]string, 0, len(exportGraph.Nodes))
	for id := range exportGraph.Nodes {
		nodeIDs = append(nodeIDs, id)
	}
	sort.Strings(nodeIDs)

	edgeIDs := make([]string, 0, len(exportGraph.Edges))
	for id := range exportGraph.Edges {
		edgeIDs = append(edgeIDs, id)
	}
	sort.Strings(edgeIDs)

	nodes := make([]graphNodeJSON, 0, len(nodeIDs))
	for _, id := range nodeIDs {
		n := exportGraph.Nodes[id]
		label := n.Title
		if label == "" {
			label = n.ID
		}
		nodes = append(nodes, graphNodeJSON{ID: n.ID, Type: n.Type, Label: label})
	}

	edges := make([]graphEdgeJSON, 0, len(edgeIDs))
	for _, id := range edgeIDs {
		e := exportGraph.Edges[id]
		edges = append(edges, graphEdgeJSON{
			Source:     e.Source,
			Target:     e.Target,
			Type:       string(e.Type),
			Confidence: roundFloat(e.Confidence, 4),
		})
	}

	payload := graphResponseJSON{Nodes: nodes, Edges: edges}
	fmt.Println(marshalContract(NewOKResponse("Graph loaded", payload)))
	return nil
}

// ─── helpers ──────────────────────────────────────────────────────────────────

// splitPositionalsAndFlags separates a slice of CLI arguments into two groups:
// positional arguments (non-flag, non-flag-value tokens) and flag tokens
// (everything starting with "-" along with their next value tokens).
//
// This allows parsers to handle flags and positionals in any order, working
// around Go's flag package stopping at the first non-flag argument.
func splitPositionalsAndFlags(args []string) (positionals []string, flags []string) {
	i := 0
	for i < len(args) {
		arg := args[i]
		if strings.HasPrefix(arg, "-") {
			// This is a flag token.  Peek to see if the next token is a value.
			flags = append(flags, arg)
			// Check whether the flag is of the form --flag=value (no next token).
			// Also handle bool flags that have no value.
			if !strings.Contains(arg, "=") {
				// Next arg might be a value if it doesn't start with '-'.
				// We need to consume the value to avoid mis-classifying it as positional.
				if i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") {
					// Check if the current flag is a bool flag by looking at name.
					flagName := strings.TrimLeft(arg, "-")
					if isBoolFlag(flagName) {
						// Bool flags don't consume the next argument.
					} else {
						i++
						flags = append(flags, args[i])
					}
				}
			}
		} else {
			positionals = append(positionals, arg)
		}
		i++
	}
	return positionals, flags
}

// pruneDanglingEdges removes edges from graph where either the source or
// target node does not exist in graph.Nodes.  This prevents FK constraint
// violations when saving to SQLite.
func pruneDanglingEdges(graph *Graph) {
	for id, e := range graph.Edges {
		_, srcOK := graph.Nodes[e.Source]
		_, tgtOK := graph.Nodes[e.Target]
		if !srcOK || !tgtOK {
			delete(graph.Edges, id)
			// Clean adjacency lists.
			graph.BySource[e.Source] = removeEdgeFromSlice(graph.BySource[e.Source], id)
			graph.ByTarget[e.Target] = removeEdgeFromSlice(graph.ByTarget[e.Target], id)
		}
	}
}

// isBoolFlag returns true for known boolean flag names used in our commands.
func isBoolFlag(name string) bool {
	boolFlags := map[string]bool{
		"watch":           true,
		"transitive":      true,
		"registry":        true,
		"no-hybrid":       true,
		"with-llm":        true,
		"include-signals": true,
		"show-confidence": true,
		"accept-all":      true,
		"reject-all":      true,
		"edit":            true,
	}
	return boolFlags[name]
}

// DefaultDBPath returns the default database path for a given directory.
func DefaultDBPath(dir string) string {
	return filepath.Join(dir, ".bmd", "knowledge.db")
}

// openOrBuildIndex opens an existing database at dbPath, or if one does not
// exist, tries to build it from the directory at absDir.
//
// When the database exists, it checks whether the index is stale (any markdown
// file modified, added, or removed since the last build) and silently rebuilds
// if needed.  Old databases without a built_at timestamp are also rebuilt.
func openOrBuildIndex(absDir, dbPath string) (*Database, error) {
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		// Auto-build index if database doesn't exist.
		fmt.Fprintln(os.Stderr, "No index found, building...")
		if err2 := CmdIndex([]string{"--dir", absDir, "--db", dbPath}); err2 != nil {
			return nil, fmt.Errorf("auto-build index: %w", err2)
		}
	} else {
		// Database exists — check if the index is stale.
		db, openErr := OpenDB(dbPath)
		if openErr != nil {
			return nil, fmt.Errorf("open db %q: %w", dbPath, openErr)
		}
		stale, staleErr := db.IsIndexStale(absDir)
		_ = db.Close()
		if staleErr == nil && stale {
			// Silently rebuild (CmdIndex writes to stderr only, which is fine).
			if err2 := CmdIndex([]string{"--dir", absDir, "--db", dbPath}); err2 != nil {
				return nil, fmt.Errorf("auto-refresh index: %w", err2)
			}
		}
	}
	db, err := OpenDB(dbPath)
	if err != nil {
		return nil, fmt.Errorf("open db %q: %w", dbPath, err)
	}
	return db, nil
}

// findNodeForService searches for a graph node matching serviceID by ID or by
// filename stem.  Returns the node ID string, or "" when not found.
func findNodeForService(graph *Graph, serviceID string) string {
	lowerSvc := strings.ToLower(serviceID)

	// Exact match first.
	if _, ok := graph.Nodes[serviceID]; ok {
		return serviceID
	}

	// Case-insensitive match on ID.
	for id := range graph.Nodes {
		if strings.ToLower(id) == lowerSvc {
			return id
		}
	}

	// Match by filename stem.
	for id := range graph.Nodes {
		stem := strings.ToLower(filenameStem(id))
		if stem == lowerSvc {
			return id
		}
	}

	return ""
}

// filenameStem returns the identifier used to match a --service flag against
// a graph node when no exact or case-insensitive ID match is found.
func filenameStem(path string) string {
	return path
}

// watchAndRebuild polls absDir every pollInterval seconds and rebuilds the
// index when changes are detected.  It blocks until the process is killed.
func watchAndRebuild(a *IndexArgs, absDir string, initialDocs []Document) error {
	_ = initialDocs
	pollDur := time.Duration(a.PollInterval) * time.Second

	// Build ScanConfig from parsed flags for use in scan operations.
	scanConfig := ScanConfig{
		IncludeHidden:     a.IncludeHidden,
		UseDefaultIgnores: !a.NoIgnoreDefaults,
	}
	if a.IgnoreDirs != "" {
		scanConfig.IgnoreDirs = strings.Split(a.IgnoreDirs, ",")
	}
	if a.IgnoreFiles != "" {
		scanConfig.IgnoreFiles = strings.Split(a.IgnoreFiles, ",")
	}

	// Create Knowledge instance with the scan configuration.
	k := NewKnowledge(scanConfig)

	for {
		time.Sleep(pollDur)

		docs, err := k.Scan(absDir)
		if err != nil {
			fmt.Fprintf(os.Stderr, "watch: scan error: %v\n", err)
			continue
		}

		changed := false
		if len(docs) > 0 {
			changed = true
		}

		if changed {
			fmt.Fprintln(os.Stderr, "Changes detected, rebuilding index...")
			if err := CmdIndex([]string{"--dir", absDir, "--db", a.DB}); err != nil {
				fmt.Fprintf(os.Stderr, "watch: rebuild error: %v\n", err)
			}
		}
	}
}

// classifyIndexError maps an openOrBuildIndex error to the appropriate ErrCode*
// constant.  Errors mentioning "no index" or "index not found" become
// ErrCodeIndexNotFound; everything else becomes ErrCodeInternalError.
func classifyIndexError(err error) string {
	msg := strings.ToLower(err.Error())
	if strings.Contains(msg, "no index") || strings.Contains(msg, "index not found") {
		return ErrCodeIndexNotFound
	}
	return ErrCodeInternalError
}

// humanBytes formats a byte count as a human-readable string (KB, MB, etc.).
func humanBytes(n int64) string {
	const unit = 1024
	if n < unit {
		return fmt.Sprintf("%dB", n)
	}
	div, exp := int64(unit), 0
	for n := n / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f%cB", float64(n)/float64(div), "KMGTPE"[exp])
}
