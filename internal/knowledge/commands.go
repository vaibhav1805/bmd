package knowledge

import (
	"errors"
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
	PollInterval int    // seconds
	Strategy     string // "" | "pageindex" — default "" means BM25-only
	Model        string // LLM model for pageindex strategy; default "claude-sonnet-4-5"
	PageIndexBin string // path to pageindex executable; default "pageindex"
}

// QueryArgs holds parsed arguments for CmdQuery.
type QueryArgs struct {
	Query    string
	Dir      string
	Format   string
	Top      int
	Strategy string // "" | "bm25" | "pageindex" — default "" (BM25)
	Model    string // LLM model for pageindex strategy; default "claude-sonnet-4-5"
}

// DependsArgs holds parsed arguments for CmdDepends.
type DependsArgs struct {
	Service         string
	Dir             string
	Transitive      bool
	Format          string
	Registry        bool    // --registry: augment results from .bmd-registry.json
	MinConfidence   float64 // --min-confidence: only show edges >= this confidence
	ShowConfidence  bool    // --show-confidence: display confidence scores in output
	IncludeSignals  bool    // --include-signals: show signal type breakdown
	NoHybrid        bool    // --no-hybrid: skip registry signal merging
}

// ComponentsArgs holds parsed arguments for CmdComponents.
type ComponentsArgs struct {
	Dir      string
	Format   string
	Registry bool // --registry: load registry from .bmd-registry.json and enrich output
}

// GraphArgs holds parsed arguments for CmdGraph.
type GraphArgs struct {
	Service  string
	Dir      string
	Format   string
	NoHybrid bool // --no-hybrid: skip registry signal merging
}

// CrawlArgs holds parsed arguments for CmdCrawl.
type CrawlArgs struct {
	FromMultiple  []string // starting node IDs (document relative paths)
	Dir           string
	Direction     string // "backward" | "forward" | "both"
	Depth         int
	Format        string  // "json" | "tree" | "dot" | "list"
	MinConfidence float64 // --min-confidence: filter edges below this threshold
	NoHybrid      bool    // --no-hybrid: skip registry signal merging
}

// ─── argument parsers ─────────────────────────────────────────────────────────

// resolveStrategy returns the strategy in precedence order: flag value → env var → default "bm25".
// This allows users to set a global preference via BMD_STRATEGY env var while allowing
// command-line flags to override it.
func resolveStrategy(flagValue string) string {
	// Flag value takes precedence
	if flagValue != "" {
		return flagValue
	}
	// Environment variable next
	if env := os.Getenv("BMD_STRATEGY"); env != "" {
		return env
	}
	// Default to BM25
	return "bm25"
}

// ParseIndexArgs parses raw CLI arguments for the index command.
//
// Usage: bmd index [DIR] [--dir DIR] [--db PATH] [--watch] [--poll-interval N]
func ParseIndexArgs(args []string) (*IndexArgs, error) {
	fs := flag.NewFlagSet("index", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	var a IndexArgs
	fs.StringVar(&a.Dir, "dir", ".", "Directory to index")
	fs.StringVar(&a.DB, "db", ".bmd/knowledge.db", "Path to SQLite database")
	fs.BoolVar(&a.Watch, "watch", false, "Rebuild index on file changes")
	fs.IntVar(&a.PollInterval, "poll-interval", 5, "Polling interval in seconds (watch mode)")
	fs.StringVar(&a.Strategy, "strategy", "", "Indexing strategy: '' (BM25 default) | 'pageindex'")
	fs.StringVar(&a.Model, "model", "claude-sonnet-4-5", "LLM model for pageindex strategy")
	fs.StringVar(&a.PageIndexBin, "pageindex-bin", "pageindex", "Path to pageindex CLI binary")

	if err := fs.Parse(args); err != nil {
		return nil, fmt.Errorf("index: %w", err)
	}

	// Positional argument overrides --dir.
	if pos := fs.Args(); len(pos) > 0 {
		a.Dir = pos[0]
	}

	// Resolve strategy: flag → env var → default
	a.Strategy = resolveStrategy(a.Strategy)

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
	fs.StringVar(&a.Strategy, "strategy", "", "Search strategy: '' or 'bm25' (default) | 'pageindex'")
	fs.StringVar(&a.Model, "model", "claude-sonnet-4-5", "LLM model for pageindex strategy")

	if err := fs.Parse(flags); err != nil {
		return nil, fmt.Errorf("query: %w", err)
	}

	if len(positionals) < 1 {
		return nil, fmt.Errorf("query: TERM argument required\nUsage: bmd query TERM [DIR] [--dir DIR] [--format json|text|csv] [--top N] [--strategy bm25|pageindex] [--model MODEL]")
	}
	a.Query = positionals[0]
	if len(positionals) > 1 {
		a.Dir = positionals[1]
	}

	if a.Top < 1 {
		return nil, fmt.Errorf("query: --top must be >= 1")
	}

	// Resolve strategy: flag → env var → default
	a.Strategy = resolveStrategy(a.Strategy)

	return &a, nil
}

// ParseDependsArgs parses raw CLI arguments for the depends command.
//
// Usage: bmd depends SERVICE [DIR] [--dir DIR] [--transitive] [--format json|text|dot] [--registry] [--min-confidence 0.0] [--no-hybrid]
func ParseDependsArgs(args []string) (*DependsArgs, error) {
	positionals, flags := splitPositionalsAndFlags(args)

	fs := flag.NewFlagSet("depends", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	var a DependsArgs
	fs.StringVar(&a.Dir, "dir", ".", "Directory that was indexed")
	fs.BoolVar(&a.Transitive, "transitive", false, "Show transitive dependencies")
	fs.StringVar(&a.Format, "format", "json", "Output format (json|text|dot)")
	fs.BoolVar(&a.Registry, "registry", false, "Augment results from .bmd-registry.json")
	fs.Float64Var(&a.MinConfidence, "min-confidence", 0.0, "Only show dependencies with confidence >= this value (0.0–1.0)")
	fs.BoolVar(&a.ShowConfidence, "show-confidence", false, "Display confidence scores in output")
	fs.BoolVar(&a.IncludeSignals, "include-signals", false, "Show signal type breakdown for each relationship")
	fs.BoolVar(&a.NoHybrid, "no-hybrid", false, "Skip registry signal merging (use base graph only)")

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

	if a.MinConfidence < 0.0 || a.MinConfidence > 1.0 {
		return nil, fmt.Errorf("depends: --min-confidence must be in [0.0, 1.0]")
	}

	return &a, nil
}

// ParseComponentsArgs parses raw CLI arguments for the components command.
//
// Usage: bmd services [--dir DIR] [--format json|text] [--registry]
func ParseComponentsArgs(args []string) (*ComponentsArgs, error) {
	positionals, flags := splitPositionalsAndFlags(args)

	fs := flag.NewFlagSet("services", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	var a ComponentsArgs
	fs.StringVar(&a.Dir, "dir", ".", "Directory that was indexed")
	fs.StringVar(&a.Format, "format", "json", "Output format (json|text)")
	fs.BoolVar(&a.Registry, "registry", false, "Enrich output from .bmd-registry.json")

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
// Usage: bmd graph [SERVICE] [--dir DIR] [--format dot|json] [--service NAME] [--no-hybrid]
func ParseGraphArgs(args []string) (*GraphArgs, error) {
	positionals, flags := splitPositionalsAndFlags(args)

	fs := flag.NewFlagSet("graph", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	var a GraphArgs
	fs.StringVar(&a.Dir, "dir", ".", "Directory that was indexed")
	fs.StringVar(&a.Format, "format", "dot", "Output format (dot|json)")
	fs.StringVar(&a.Service, "service", "", "Export subgraph for this service only")
	fs.BoolVar(&a.NoHybrid, "no-hybrid", false, "Skip registry signal merging (use base graph only)")

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

// ParseCrawlArgs parses raw CLI arguments for the crawl command.
//
// Usage: bmd crawl --from-multiple FILE[,FILE...] [--dir DIR] [--direction backward|forward|both] [--depth N] [--format json|tree|dot|list] [--min-confidence 0.0] [--no-hybrid]
func ParseCrawlArgs(args []string) (*CrawlArgs, error) {
	fs := flag.NewFlagSet("crawl", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	var a CrawlArgs
	var fromMultiple string
	fs.StringVar(&fromMultiple, "from-multiple", "", "Comma-separated list of starting files")
	fs.StringVar(&a.Dir, "dir", ".", "Directory that was indexed")
	fs.StringVar(&a.Direction, "direction", "backward", "Traversal direction: backward|forward|both")
	fs.IntVar(&a.Depth, "depth", 3, "Maximum traversal depth")
	fs.StringVar(&a.Format, "format", "json", "Output format: json|tree|dot|list")
	fs.Float64Var(&a.MinConfidence, "min-confidence", 0.0, "Only traverse edges with confidence >= this value (0.0–1.0)")
	fs.BoolVar(&a.NoHybrid, "no-hybrid", false, "Skip registry signal merging (use base graph only)")

	if err := fs.Parse(args); err != nil {
		return nil, fmt.Errorf("crawl: %w", err)
	}

	// Allow positional arguments as start files if --from-multiple not set.
	if fromMultiple == "" {
		pos := fs.Args()
		if len(pos) > 0 {
			fromMultiple = strings.Join(pos, ",")
		}
	}

	if fromMultiple == "" {
		return nil, fmt.Errorf("crawl: --from-multiple argument required\nUsage: bmd crawl --from-multiple FILE[,FILE...] [--direction backward|forward|both] [--depth N] [--format json|tree|dot|list]")
	}

	// Parse comma-separated file list.
	for _, f := range strings.Split(fromMultiple, ",") {
		f = strings.TrimSpace(f)
		if f != "" {
			a.FromMultiple = append(a.FromMultiple, f)
		}
	}

	if len(a.FromMultiple) == 0 {
		return nil, fmt.Errorf("crawl: at least one file required in --from-multiple")
	}

	// Validate direction.
	switch strings.ToLower(a.Direction) {
	case "backward", "forward", "both":
		a.Direction = strings.ToLower(a.Direction)
	default:
		return nil, fmt.Errorf("crawl: invalid direction %q (must be backward, forward, or both)", a.Direction)
	}

	// Validate format.
	switch strings.ToLower(a.Format) {
	case "json", "tree", "dot", "list":
		a.Format = strings.ToLower(a.Format)
	default:
		return nil, fmt.Errorf("crawl: invalid format %q (must be json, tree, dot, or list)", a.Format)
	}

	if a.Depth < 0 {
		return nil, fmt.Errorf("crawl: --depth must be >= 0")
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
	docs, err := ScanDirectory(absDir, ScanConfig{UseDefaultIgnores: true})
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

	// Discover additional relationships via co-occurrence and structural analysis.
	discovered := DiscoverRelationships(docs, nil)
	for _, de := range discovered {
		if de.Edge != nil {
			_ = graph.AddEdge(de.Edge)
		}
	}

	fmt.Fprintf(os.Stderr, "  %d nodes in knowledge graph\n", graph.NodeCount())
	fmt.Fprintf(os.Stderr, "  %d edges (relationships)\n", graph.EdgeCount())

	// Detect services (with optional components.yaml config).
	sd := NewComponentDetector()
	cfgPath := filepath.Join(absDir, "components.yaml")
	if cfg, cfgErr := LoadComponentConfig(cfgPath); cfgErr == nil && cfg != nil {
		sd = NewComponentDetectorWithConfig(cfg)
	}
	services := sd.DetectComponents(graph, docs)
	fmt.Fprintf(os.Stderr, "  %d microservices detected\n", len(services))

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

	// Generate discovered relationship manifest.
	edgeList := make([]*Edge, 0, len(graph.Edges))
	for _, e := range graph.Edges {
		edgeList = append(edgeList, e)
	}

	// Build a registry for richer signal data (non-fatal if it fails).
	var registry *ComponentRegistry
	registryPath := filepath.Join(absDir, RegistryFileName)
	registry, _ = LoadRegistry(registryPath)

	manifest := GenerateRelationshipManifest(edgeList, registry)
	discoveredPath := filepath.Join(absDir, DiscoveredManifestFile)
	if err := SaveRelationshipManifest(manifest, discoveredPath); err != nil {
		fmt.Fprintf(os.Stderr, "  warning: save discovered manifest: %v\n", err)
	} else {
		fmt.Fprintf(os.Stderr, "  %d relationships written to %s\n", len(manifest.Relationships), DiscoveredManifestFile)
	}

	// Create and save registry for subsequent commands to reuse index without rebuilding.
	// Pass absDir so registry can load components.yaml if present
	reg := NewComponentRegistry()
	reg.InitFromGraphWithDir(graph, docs, absDir)
	if err := SaveRegistry(reg, registryPath); err != nil {
		fmt.Fprintf(os.Stderr, "  warning: save registry: %v\n", err)
	} else {
		fmt.Fprintf(os.Stderr, "  Registry saved to %s\n", RegistryFileName)
	}

	// Check for accepted relationships and merge into graph.
	acceptedPath := filepath.Join(absDir, AcceptedManifestFile)
	acceptedEdges, loadErr := LoadAcceptedRelationships(acceptedPath)
	if loadErr != nil {
		fmt.Fprintf(os.Stderr, "  warning: load accepted manifest: %v\n", loadErr)
	} else if len(acceptedEdges) > 0 {
		added := 0
		for _, e := range acceptedEdges {
			if err := graph.AddEdge(e); err == nil {
				added++
			}
		}
		if added > 0 {
			fmt.Fprintf(os.Stderr, "  %d accepted relationships merged into graph\n", added)
			// Re-save graph with accepted edges.
			if err := db.SaveGraph(graph); err != nil {
				fmt.Fprintf(os.Stderr, "  warning: re-save graph: %v\n", err)
			}
		}
	} else if acceptedEdges == nil {
		fmt.Fprintf(os.Stderr, "  Run 'bmd relationships-review' to review discovered relationships\n")
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

	// PageIndex tree generation (opt-in, after BM25 so BM25 always succeeds first).
	if strings.ToLower(a.Strategy) == "pageindex" {
		fmt.Fprintf(os.Stderr, "  Generating PageIndex trees (strategy=pageindex)...\n")
		cfg := PageIndexConfig{
			ExecutablePath: a.PageIndexBin,
			Model:          a.Model,
		}
		treeCount := 0
		for _, doc := range docs {
			ft, err := RunPageIndex(cfg, doc.Path)
			if errors.Is(err, ErrPageIndexNotFound) {
				return fmt.Errorf("index: pageindex strategy requires pageindex CLI: %w", err)
			}
			if err != nil {
				fmt.Fprintf(os.Stderr, "  warning: pageindex failed for %s: %v\n", doc.RelPath, err)
				continue
			}
			if err := SaveTreeFile(absDir, ft); err != nil {
				fmt.Fprintf(os.Stderr, "  warning: save tree failed for %s: %v\n", doc.RelPath, err)
				continue
			}
			treeCount++
		}
		fmt.Fprintf(os.Stderr, "  %d tree files written\n", treeCount)
	}

	if a.Watch {
		fmt.Fprintf(os.Stderr, "Watching %s for changes (poll every %ds)...\n", absDir, a.PollInterval)
		return watchAndRebuild(a, absDir, docs)
	}

	return nil
}

// CmdQuery implements `bmd query TERM`.  It loads the index from the database
// and executes a BM25 search (default) or a PageIndex semantic search
// (--strategy pageindex), printing results in the requested format.
func CmdQuery(args []string) error {
	a, err := ParseQueryArgs(args)
	if err != nil {
		return err
	}

	// Route to pageindex strategy when requested.
	usePageIndex := strings.ToLower(a.Strategy) == "pageindex"
	if usePageIndex {
		return cmdQueryPageIndex(a)
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

	dbPath := defaultDBPath(absDir)
	db, err := openOrBuildIndex(absDir, dbPath)
	if err != nil {
		if isJSON {
			fmt.Println(marshalContract(NewErrorResponse(classifyIndexError(err), err.Error())))
			return nil
		}
		return err
	}
	defer db.Close() //nolint:errcheck

	// Load index.
	idx := NewIndex()
	if err := db.LoadIndex(idx); err != nil {
		if isJSON {
			fmt.Println(marshalContract(NewErrorResponse(ErrCodeInternalError, err.Error())))
			return nil
		}
		return fmt.Errorf("query: load index: %w", err)
	}

	// Re-scan to populate content for snippets (db stores only metadata).
	docs, scanErr := ScanDirectory(absDir, ScanConfig{UseDefaultIgnores: true})
	if scanErr == nil {
		// Re-build in-memory so snippets are available.
		_ = idx.Build(docs)
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

// cmdQueryPageIndex executes `bmd query` with strategy=pageindex.
// It loads .bmd-tree.json files from absDir, calls RunPageIndexQuery,
// and returns results wrapped in a CONTRACT-01 envelope with reasoning_trace
// fields per result.
//
// Graceful fallback paths:
//   - No .bmd-tree.json files → INDEX_NOT_FOUND error
//   - pageindex binary not found → PAGEINDEX_NOT_AVAILABLE error
func cmdQueryPageIndex(a *QueryArgs) error {
	isJSON := strings.ToLower(a.Format) == "json"

	absDir, err := filepath.Abs(a.Dir)
	if err != nil {
		if isJSON {
			fmt.Println(marshalContract(NewErrorResponse(ErrCodeInternalError, err.Error())))
			return nil
		}
		return fmt.Errorf("query: resolve dir: %w", err)
	}

	start := time.Now()

	// Load tree files.
	trees, err := LoadTreeFiles(absDir)
	if err != nil || len(trees) == 0 {
		msg := "No .bmd-tree.json files found. Run: bmd index --strategy pageindex"
		if isJSON {
			fmt.Println(marshalContract(NewErrorResponse(ErrCodeIndexNotFound, msg)))
			return nil
		}
		fmt.Fprintln(os.Stderr, msg)
		return nil
	}

	cfg := PageIndexConfig{
		ExecutablePath: "pageindex",
		Model:          a.Model,
	}

	sections, err := RunPageIndexQuery(cfg, a.Query, trees, a.Top)
	if err != nil {
		if errors.Is(err, ErrPageIndexNotFound) {
			msg := "pageindex CLI not found. Install via: pip install pageindex"
			if isJSON {
				fmt.Println(marshalContract(NewErrorResponse(ErrCodePageIndexNotAvailable, msg)))
				return nil
			}
			return fmt.Errorf("%s", msg)
		}
		if isJSON {
			fmt.Println(marshalContract(NewErrorResponse(ErrCodeInternalError, err.Error())))
			return nil
		}
		return fmt.Errorf("query: pageindex: %w", err)
	}

	queryTimeMs := time.Since(start).Milliseconds()

	if !isJSON {
		// Text output for pageindex strategy.
		if len(sections) == 0 {
			fmt.Println("No results found.")
			return nil
		}
		for i, s := range sections {
			fmt.Printf("%d. %s", i+1, s.File)
			if s.HeadingPath != "" {
				fmt.Printf(" \u00a7 %s", s.HeadingPath)
			}
			fmt.Printf(" (score: %.4f)\n", s.Score)
			if s.ReasoningTrace != "" {
				fmt.Printf("   Reasoning: %s\n", s.ReasoningTrace)
			}
		}
		return nil
	}

	// JSON path: build pageindexResponseJSON wrapped in ContractResponse.
	if len(sections) == 0 {
		fmt.Println(marshalContract(NewEmptyResponse("No results found", pageindexResponseJSON{
			Query:       a.Query,
			Strategy:    "pageindex",
			Model:       a.Model,
			Results:     []reasoningResultJSON{},
			Count:       0,
			QueryTimeMs: queryTimeMs,
		})))
		return nil
	}

	items := make([]reasoningResultJSON, len(sections))
	for i, s := range sections {
		items[i] = reasoningResultJSON{
			Rank:           i + 1,
			File:           s.File,
			HeadingPath:    s.HeadingPath,
			Content:        s.Content,
			ContentPreview: contentPreview(s.Content, 200),
			Score:          roundFloat(s.Score, 4),
			ReasoningTrace: s.ReasoningTrace,
		}
	}
	payload := pageindexResponseJSON{
		Query:       a.Query,
		Strategy:    "pageindex",
		Model:       a.Model,
		Results:     items,
		Count:       len(items),
		QueryTimeMs: queryTimeMs,
	}
	fmt.Println(marshalContract(NewOKResponse("Search completed", payload)))
	return nil
}

// CmdDepends implements `bmd depends SERVICE`.  It loads the service graph and
// prints direct or transitive dependencies.
func CmdDepends(args []string) error {
	a, err := ParseDependsArgs(args)
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
		return fmt.Errorf("depends: resolve dir: %w", err)
	}

	db, graph, services, err := loadGraphAndServices(absDir)
	if err != nil {
		if isJSON {
			fmt.Println(marshalContract(NewErrorResponse(classifyIndexError(err), err.Error())))
			return nil
		}
		return fmt.Errorf("depends: %w", err)
	}
	defer db.Close() //nolint:errcheck

	// Build dependency analyzer.
	da := NewDependencyAnalyzer(graph, services)
	sg := da.GetComponentGraph()

	// Validate component exists.
	if _, ok := sg.Components[a.Service]; !ok {
		// Try case-insensitive match.
		matched := ""
		for id := range sg.Components {
			if strings.EqualFold(id, a.Service) {
				matched = id
				break
			}
		}
		if matched == "" {
			if isJSON {
				fmt.Println(marshalContract(NewErrorResponse(ErrCodeFileNotFound, "Service not found: "+a.Service)))
				return nil
			}
			return fmt.Errorf("depends: service %q not found. Run `bmd services` to list available services", a.Service)
		}
		a.Service = matched
	}

	refs := sg.Dependencies[a.Service]

	// Apply confidence filtering if requested.
	if a.MinConfidence > 0 {
		var filtered []ComponentRef
		for _, r := range refs {
			if r.Confidence >= a.MinConfidence {
				filtered = append(filtered, r)
			}
		}
		refs = filtered
	}

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

	if !isJSON {
		// Text and DOT paths.
		output := FormatDependencies(a.Service, refs, a.Transitive, transitivePaths, cycles, a.Format)
		fmt.Println(output)
		return nil
	}

	// JSON path: build payload and wrap in envelope.
	// Re-use the same inner structs as formatDependenciesJSON.
	var payload interface{}
	var cleanCycles [][]string
	if len(cycles) > 0 {
		cleanCycles = cycles
	}
	if a.Transitive {
		items := make([]depTransitiveJSON, 0, len(transitivePaths))
		for _, path := range transitivePaths {
			items = append(items, depTransitiveJSON{
				Path:     path,
				Distance: len(path) - 1,
			})
		}
		payload = depsTransitiveResponseJSON{
			Service:                a.Service,
			TransitiveDependencies: items,
			Cycles:                 cleanCycles,
		}
	} else {
		items := make([]depRefJSON, len(refs))
		for i, r := range refs {
			items[i] = depRefJSON{
				Service:    r.ComponentID,
				Type:       r.Type,
				Confidence: roundFloat(r.Confidence, 2),
			}
		}
		payload = depsDirectResponseJSON{
			Service:      a.Service,
			Dependencies: items,
			Cycles:       cleanCycles,
		}
	}
	fmt.Println(marshalContract(NewOKResponse("Dependencies found", payload)))
	return nil
}

// cmdComponentsLegacy implements the legacy `bmd components` behavior (no subcommand).
// It loads the knowledge graph and prints all detected services/components.
// Called by CmdComponents in registry_cmd.go when no subcommand is given.
func cmdComponentsLegacy(args []string) error {
	a, err := ParseComponentsArgs(args)
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
		return fmt.Errorf("services: resolve dir: %w", err)
	}

	db, graph, services, err := loadGraphAndServices(absDir)
	if err != nil {
		if isJSON {
			fmt.Println(marshalContract(NewErrorResponse(classifyIndexError(err), err.Error())))
			return nil
		}
		return fmt.Errorf("services: %w", err)
	}
	defer db.Close() //nolint:errcheck

	// Build dependency counts.
	da := NewDependencyAnalyzer(graph, services)
	sg := da.GetComponentGraph()

	depCounts := make(map[string]int, len(services))
	for id, refs := range sg.Dependencies {
		depCounts[id] = len(refs)
	}

	if !isJSON {
		// Text path is unchanged.
		output := FormatComponents(services, depCounts, a.Format)
		fmt.Println(output)
		return nil
	}

	// JSON path: build payload and wrap in envelope.
	items := make([]componentEntryJSON, len(services))
	for i, s := range services {
		cnt := depCounts[s.ID]
		items[i] = componentEntryJSON{
			ID:              s.ID,
			Name:            s.Name,
			File:            s.File,
			Confidence:      roundFloat(s.Confidence, 2),
			DependencyCount: cnt,
		}
	}
	payload := componentsResponseJSON{Components: items}
	fmt.Println(marshalContract(NewOKResponse("Components detected", payload)))
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

	dbPath := defaultDBPath(absDir)
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

	// Augment graph with registry signals unless --no-hybrid is set.
	if !a.NoHybrid {
		registryPath := filepath.Join(absDir, RegistryFileName)
		if reg, regErr := LoadRegistry(registryPath); regErr == nil && reg != nil {
			_ = graph.MergeRegistry(reg)
		}
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

// CmdCrawl implements `bmd crawl`.  It loads the knowledge graph and performs
// a multi-start BFS traversal, printing results in the requested format.
func CmdCrawl(args []string) error {
	a, err := ParseCrawlArgs(args)
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
		return fmt.Errorf("crawl: resolve dir: %w", err)
	}

	dbPath := defaultDBPath(absDir)
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
		return fmt.Errorf("crawl: load graph: %w", err)
	}

	// Normalize file paths to match graph node IDs.
	// Graph node IDs are stored as relative paths with forward slashes (e.g., "services/user.md").
	// User input may be relative ("./services/user.md") or absolute paths that need normalization.
	normalizedFiles := make([]string, len(a.FromMultiple))
	for i, f := range a.FromMultiple {
		// Clean path and convert to forward slashes.
		cleaned := filepath.Clean(f)
		normalized := filepath.ToSlash(cleaned)

		// Remove leading "./" if present.
		normalized = strings.TrimPrefix(normalized, "./")

		normalizedFiles[i] = normalized
	}

	// Validate that at least one start file exists in the graph.
	validFiles := 0
	for _, f := range normalizedFiles {
		if _, ok := graph.Nodes[f]; ok {
			validFiles++
		}
	}
	if validFiles == 0 {
		msg := fmt.Sprintf("none of the specified files found in graph: %s", strings.Join(a.FromMultiple, ", "))
		if isJSON {
			fmt.Println(marshalContract(NewErrorResponse(ErrCodeFileNotFound, msg)))
			return nil
		}
		return fmt.Errorf("crawl: %s", msg)
	}

	// Execute crawl.
	result := graph.CrawlMulti(CrawlOptions{
		FromFiles:     normalizedFiles,
		Direction:     a.Direction,
		MaxDepth:      a.Depth,
		IncludeCycles: true,
	})

	// Format output.
	output := FormatCrawl(result, a.Format)

	if isJSON {
		fmt.Println(marshalContract(NewOKResponse("Crawl completed", output)))
	} else {
		fmt.Println(output)
	}

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

// defaultDBPath returns the default database path for a given directory.
func defaultDBPath(dir string) string {
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

// loadGraphAndServices is a convenience helper that opens the database,
// loads the graph, re-scans for documents, and detects services.
func loadGraphAndServices(absDir string) (*Database, *Graph, []Component, error) {
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
	docs, scanErr := ScanDirectory(absDir, ScanConfig{UseDefaultIgnores: true})
	if scanErr != nil {
		// Non-fatal: service detection works with empty docs.
		docs = nil
	}

	sd := NewComponentDetector()

	// Try to load optional components.yaml config.
	cfgPath := filepath.Join(absDir, "components.yaml")
	if cfg, cfgErr := LoadComponentConfig(cfgPath); cfgErr == nil && cfg != nil {
		sd = NewComponentDetectorWithConfig(cfg)
	}

	services := sd.DetectComponents(graph, docs)
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

		docs, err := ScanDirectory(absDir, ScanConfig{UseDefaultIgnores: true})
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
