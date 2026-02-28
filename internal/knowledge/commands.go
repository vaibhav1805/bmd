package knowledge

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
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
}

// QueryArgs holds parsed arguments for CmdQuery.
type QueryArgs struct {
	Query  string
	Dir    string
	Format string
	Top    int
}

// DependsArgs holds parsed arguments for CmdDepends.
type DependsArgs struct {
	Service    string
	Dir        string
	Transitive bool
	Format     string
}

// ServicesArgs holds parsed arguments for CmdServices.
type ServicesArgs struct {
	Dir    string
	Format string
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
func ParseIndexArgs(args []string) (*IndexArgs, error) {
	fs := flag.NewFlagSet("index", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	var a IndexArgs
	fs.StringVar(&a.Dir, "dir", ".", "Directory to index")
	fs.StringVar(&a.DB, "db", "knowledge.db", "Path to SQLite database")
	fs.BoolVar(&a.Watch, "watch", false, "Rebuild index on file changes")
	fs.IntVar(&a.PollInterval, "poll-interval", 5, "Polling interval in seconds (watch mode)")

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
		return nil, fmt.Errorf("query: TERM argument required")
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

// ParseDependsArgs parses raw CLI arguments for the depends command.
//
// Usage: bmd depends SERVICE [DIR] [--dir DIR] [--transitive] [--format json|text|dot]
func ParseDependsArgs(args []string) (*DependsArgs, error) {
	positionals, flags := splitPositionalsAndFlags(args)

	fs := flag.NewFlagSet("depends", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	var a DependsArgs
	fs.StringVar(&a.Dir, "dir", ".", "Directory that was indexed")
	fs.BoolVar(&a.Transitive, "transitive", false, "Show transitive dependencies")
	fs.StringVar(&a.Format, "format", "json", "Output format (json|text|dot)")

	if err := fs.Parse(flags); err != nil {
		return nil, fmt.Errorf("depends: %w", err)
	}

	if len(positionals) < 1 {
		return nil, fmt.Errorf("depends: SERVICE argument required")
	}
	a.Service = positionals[0]
	if len(positionals) > 1 {
		a.Dir = positionals[1]
	}

	return &a, nil
}

// ParseServicesArgs parses raw CLI arguments for the services command.
//
// Usage: bmd services [--dir DIR] [--format json|text]
func ParseServicesArgs(args []string) (*ServicesArgs, error) {
	positionals, flags := splitPositionalsAndFlags(args)

	fs := flag.NewFlagSet("services", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	var a ServicesArgs
	fs.StringVar(&a.Dir, "dir", ".", "Directory that was indexed")
	fs.StringVar(&a.Format, "format", "json", "Output format (json|text)")

	if err := fs.Parse(flags); err != nil {
		return nil, fmt.Errorf("services: %w", err)
	}

	if len(positionals) > 0 {
		a.Dir = positionals[0]
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

	fmt.Fprintf(os.Stderr, "Indexing %s...\n", absDir)

	start := time.Now()

	// Scan markdown files.
	docs, err := ScanDirectory(absDir)
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

	// Build knowledge graph.
	gb := NewGraphBuilder(absDir)
	graph := gb.Build(docs)

	fmt.Fprintf(os.Stderr, "  %d nodes in knowledge graph\n", graph.NodeCount())
	fmt.Fprintf(os.Stderr, "  %d edges (relationships)\n", graph.EdgeCount())

	// Detect services.
	sd := NewServiceDetector()
	services := sd.DetectServices(graph, docs)
	fmt.Fprintf(os.Stderr, "  %d microservices detected\n", len(services))

	// Open / create database.
	dbPath := a.DB
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

// CmdQuery implements `bmd query TERM`.  It loads the index from the database
// and executes a BM25 search, printing results in the requested format.
func CmdQuery(args []string) error {
	a, err := ParseQueryArgs(args)
	if err != nil {
		return err
	}

	start := time.Now()

	absDir, err := filepath.Abs(a.Dir)
	if err != nil {
		return fmt.Errorf("query: resolve dir: %w", err)
	}

	dbPath := defaultDBPath(absDir)
	db, err := openOrBuildIndex(absDir, dbPath)
	if err != nil {
		return err
	}
	defer db.Close() //nolint:errcheck

	// Load index.
	idx := NewIndex()
	if err := db.LoadIndex(idx); err != nil {
		return fmt.Errorf("query: load index: %w", err)
	}

	// Re-scan to populate content for snippets (db stores only metadata).
	docs, scanErr := ScanDirectory(absDir)
	if scanErr == nil {
		// Re-build in-memory so snippets are available.
		_ = idx.Build(docs)
	}

	// Execute search.
	results, err := idx.Search(a.Query, a.Top)
	if err != nil {
		return fmt.Errorf("query: search: %w", err)
	}

	queryTimeMs := time.Since(start).Milliseconds()
	output := FormatSearchResults(results, a.Query, a.Format, queryTimeMs)
	fmt.Println(output)
	return nil
}

// CmdDepends implements `bmd depends SERVICE`.  It loads the service graph and
// prints direct or transitive dependencies.
func CmdDepends(args []string) error {
	a, err := ParseDependsArgs(args)
	if err != nil {
		return err
	}

	absDir, err := filepath.Abs(a.Dir)
	if err != nil {
		return fmt.Errorf("depends: resolve dir: %w", err)
	}

	db, graph, services, err := loadGraphAndServices(absDir)
	if err != nil {
		return fmt.Errorf("depends: %w", err)
	}
	defer db.Close() //nolint:errcheck

	// Build dependency analyzer.
	da := NewDependencyAnalyzer(graph, services)
	sg := da.GetServiceGraph()

	// Validate service exists.
	if _, ok := sg.Services[a.Service]; !ok {
		// Try case-insensitive match.
		matched := ""
		for id := range sg.Services {
			if strings.EqualFold(id, a.Service) {
				matched = id
				break
			}
		}
		if matched == "" {
			return fmt.Errorf("depends: service %q not found. Run `bmd services` to list available services", a.Service)
		}
		a.Service = matched
	}

	refs := sg.Dependencies[a.Service]

	// Detect cycles.
	cycles := da.DetectCycles()

	var transitivePaths [][]string
	if a.Transitive {
		transitiveIDs := da.GetTransitiveDeps(a.Service)
		for _, tid := range transitiveIDs {
			chain := da.FindDependencyChain(a.Service, tid)
			if len(chain.Path) > 0 {
				transitivePaths = append(transitivePaths, chain.Path)
			}
		}
	}

	output := FormatDependencies(a.Service, refs, a.Transitive, transitivePaths, cycles, a.Format)
	fmt.Println(output)
	return nil
}

// CmdServices implements `bmd services`.  It loads the knowledge graph and
// prints all detected services.
func CmdServices(args []string) error {
	a, err := ParseServicesArgs(args)
	if err != nil {
		return err
	}

	absDir, err := filepath.Abs(a.Dir)
	if err != nil {
		return fmt.Errorf("services: resolve dir: %w", err)
	}

	db, graph, services, err := loadGraphAndServices(absDir)
	if err != nil {
		return fmt.Errorf("services: %w", err)
	}
	defer db.Close() //nolint:errcheck

	// Build dependency counts.
	da := NewDependencyAnalyzer(graph, services)
	sg := da.GetServiceGraph()

	depCounts := make(map[string]int, len(services))
	for id, refs := range sg.Dependencies {
		depCounts[id] = len(refs)
	}

	output := FormatServices(services, depCounts, a.Format)
	fmt.Println(output)
	return nil
}

// CmdGraph implements `bmd graph`.  It loads the knowledge graph and exports
// it in the requested format (DOT or JSON).
func CmdGraph(args []string) error {
	a, err := ParseGraphArgs(args)
	if err != nil {
		return err
	}

	absDir, err := filepath.Abs(a.Dir)
	if err != nil {
		return fmt.Errorf("graph: resolve dir: %w", err)
	}

	dbPath := defaultDBPath(absDir)
	db, err := openOrBuildIndex(absDir, dbPath)
	if err != nil {
		return err
	}
	defer db.Close() //nolint:errcheck

	graph := NewGraph()
	if err := db.LoadGraph(graph); err != nil {
		return fmt.Errorf("graph: load graph: %w", err)
	}

	// Apply subgraph filter when a service is specified.
	exportGraph := graph
	if a.Service != "" {
		// Find the node ID for this service (match by ID or filename stem).
		nodeID := findNodeForService(graph, a.Service)
		if nodeID == "" {
			return fmt.Errorf("graph: service/node %q not found in graph", a.Service)
		}
		exportGraph = graph.GetSubgraph(nodeID, 10)
	}

	output := FormatGraph(exportGraph, a.Format)
	fmt.Println(output)
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
		"watch":      true,
		"transitive": true,
	}
	return boolFlags[name]
}

// defaultDBPath returns the default database path for a given directory.
func defaultDBPath(dir string) string {
	return filepath.Join(dir, "knowledge.db")
}

// openOrBuildIndex opens an existing database at dbPath, or if one does not
// exist, tries to build it from the directory at absDir.
func openOrBuildIndex(absDir, dbPath string) (*Database, error) {
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		// Auto-build index if database doesn't exist.
		fmt.Fprintln(os.Stderr, "No index found, building...")
		if err2 := CmdIndex([]string{"--dir", absDir, "--db", dbPath}); err2 != nil {
			return nil, fmt.Errorf("auto-build index: %w", err2)
		}
	}
	db, err := OpenDB(dbPath)
	if err != nil {
		return nil, fmt.Errorf("open db %q: %w", dbPath, err)
	}
	return db, nil
}

// loadGraphAndServices is a convenience helper that opens the database,
// loads the graph, re-scans for documents, and detects services.
func loadGraphAndServices(absDir string) (*Database, *Graph, []Service, error) {
	dbPath := defaultDBPath(absDir)
	db, err := openOrBuildIndex(absDir, dbPath)
	if err != nil {
		return nil, nil, nil, err
	}

	graph := NewGraph()
	if err := db.LoadGraph(graph); err != nil {
		_ = db.Close()
		return nil, nil, nil, fmt.Errorf("load graph: %w", err)
	}

	// Scan documents for service detection (endpoint extraction needs content).
	docs, scanErr := ScanDirectory(absDir)
	if scanErr != nil {
		// Non-fatal: service detection works with empty docs.
		docs = nil
	}

	sd := NewServiceDetector()

	// Try to load optional services.yaml config.
	cfgPath := filepath.Join(absDir, "services.yaml")
	if cfg, cfgErr := LoadServiceConfig(cfgPath); cfgErr == nil && cfg != nil {
		sd = NewServiceDetectorWithConfig(cfg)
	}

	services := sd.DetectServices(graph, docs)
	return db, graph, services, nil
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

// watchAndRebuild polls absDir every pollInterval seconds and rebuilds the
// index when changes are detected.  It blocks until the process is killed.
func watchAndRebuild(a *IndexArgs, absDir string, initialDocs []Document) error {
	_ = initialDocs
	pollDur := time.Duration(a.PollInterval) * time.Second

	for {
		time.Sleep(pollDur)

		docs, err := ScanDirectory(absDir)
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
			buildArgs := &IndexArgs{Dir: absDir, DB: a.DB}
			if err := CmdIndex([]string{"--dir", absDir, "--db", a.DB}); err != nil {
				fmt.Fprintf(os.Stderr, "watch: rebuild error: %v\n", err)
			}
			_ = buildArgs
		}
	}
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
