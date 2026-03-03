package main

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/bmd/bmd/internal/config"
	"github.com/bmd/bmd/internal/knowledge"
	bmcmp "github.com/bmd/bmd/internal/mcp"
	"github.com/bmd/bmd/internal/parser"
	"github.com/bmd/bmd/internal/terminal"
	"github.com/bmd/bmd/internal/theme"
	"github.com/bmd/bmd/internal/tui"
)

func main() {
	args := os.Args[1:]

	// Route knowledge commands before the legacy viewer path.
	if len(args) > 0 {
		var cmdErr error
		switch args[0] {
		case "index":
			cmdErr = knowledge.CmdIndex(args[1:])
			if cmdErr != nil {
				fmt.Fprintln(os.Stderr, "bmd index:", cmdErr)
				os.Exit(1)
			}
			return
		case "query":
			cmdErr = knowledge.CmdQuery(args[1:])
			if cmdErr != nil {
				fmt.Fprintln(os.Stderr, "bmd query:", cmdErr)
				os.Exit(1)
			}
			return
		case "depends":
			cmdErr = knowledge.CmdDepends(args[1:])
			if cmdErr != nil {
				fmt.Fprintln(os.Stderr, "bmd depends:", cmdErr)
				os.Exit(1)
			}
			return
		case "components":
			cmdErr = knowledge.CmdComponents(args[1:])
			if cmdErr != nil {
				fmt.Fprintln(os.Stderr, "bmd components:", cmdErr)
				os.Exit(1)
			}
			return
		case "debug":
			cmdErr = knowledge.CmdDebug(args[1:])
			if cmdErr != nil {
				fmt.Fprintln(os.Stderr, "bmd debug:", cmdErr)
				os.Exit(1)
			}
			return
		case "relationships":
			cmdErr = knowledge.CmdRelationships(args[1:])
			if cmdErr != nil {
				fmt.Fprintln(os.Stderr, "bmd relationships:", cmdErr)
				os.Exit(1)
			}
			return
		case "graph":
			cmdErr = knowledge.CmdGraph(args[1:])
			if cmdErr != nil {
				fmt.Fprintln(os.Stderr, "bmd graph:", cmdErr)
				os.Exit(1)
			}
			return
		case "context":
			cmdErr = knowledge.CmdContext(args[1:])
			if cmdErr != nil {
				fmt.Fprintln(os.Stderr, "bmd context:", cmdErr)
				os.Exit(1)
			}
			return
		case "crawl":
			cmdErr = knowledge.CmdCrawl(args[1:])
			if cmdErr != nil {
				fmt.Fprintln(os.Stderr, "bmd crawl:", cmdErr)
				os.Exit(1)
			}
			return
		case "relationships-review":
			cmdErr = knowledge.CmdRelationshipsReview(args[1:])
			if cmdErr != nil {
				fmt.Fprintln(os.Stderr, "bmd relationships-review:", cmdErr)
				os.Exit(1)
			}
			return
		case "watch":
			cmdErr = knowledge.CmdWatch(args[1:])
			if cmdErr != nil {
				fmt.Fprintln(os.Stderr, "bmd watch:", cmdErr)
				os.Exit(1)
			}
			return
		case "export":
			cmdErr = knowledge.CmdExport(args[1:])
			if cmdErr != nil {
				fmt.Fprintln(os.Stderr, "bmd export:", cmdErr)
				os.Exit(1)
			}
			return
		case "import":
			if err := runImport(args[1:]); err != nil {
				fmt.Fprintln(os.Stderr, "bmd import:", err)
				os.Exit(1)
			}
			return
		case "serve":
			if err := runServe(args[1:]); err != nil {
				fmt.Fprintln(os.Stderr, "bmd:", err)
				os.Exit(1)
			}
			return
		case "clean":
			cmdErr = knowledge.CmdClean(args[1:])
			if cmdErr != nil {
				fmt.Fprintln(os.Stderr, "bmd clean:", cmdErr)
				os.Exit(1)
			}
			return
		case "-h", "--help", "help":
			usage()
			return
		case "--browse":
			// Explicit directory browse mode.
			dir := "."
			if len(args) > 1 {
				dir = args[1]
			}
			runDirectoryViewer(dir)
			return
		}
	}

	// No arguments: check if current directory has markdown files → directory mode.
	if len(args) == 0 {
		cwd, err := os.Getwd()
		if err != nil {
			fmt.Fprintln(os.Stderr, "bmd: cannot get current directory:", err)
			os.Exit(1)
		}
		if directoryHasMdFiles(cwd) {
			runDirectoryViewer(cwd)
			return
		}
		// No markdown files: show usage.
		usage()
		os.Exit(1)
	}

	// Viewer mode: argument must be a markdown file or a help flag.
	if strings.HasSuffix(args[0], ".md") || args[0] == "-h" || args[0] == "--help" {
		runViewer(args[0])
		return
	}

	// Unknown command.
	fmt.Fprintf(os.Stderr, "bmd: unknown command %q\n\n", args[0])
	usage()
	os.Exit(1)
}

// directoryHasMdFiles returns true if path contains at least one .md file.
func directoryHasMdFiles(path string) bool {
	found := false
	_ = filepath.WalkDir(path, func(p string, d fs.DirEntry, err error) error {
		if err != nil || found {
			return nil
		}
		if !d.IsDir() && strings.ToLower(filepath.Ext(p)) == ".md" {
			found = true
		}
		return nil
	})
	return found
}

// runDirectoryViewer launches the directory browsing TUI for the given directory.
func runDirectoryViewer(dir string) {
	absDir, err := filepath.Abs(dir)
	if err != nil {
		fmt.Fprintln(os.Stderr, "bmd: cannot resolve directory:", err)
		os.Exit(1)
	}

	termWidth := terminal.DetectTerminalWidth()
	cfg, _ := config.Load()
	th := theme.NewThemeByName(theme.ThemeName(cfg.Theme))

	v := tui.NewDirectoryViewer(absDir, th, termWidth)
	if err := v.LoadDirectory(absDir); err != nil {
		fmt.Fprintln(os.Stderr, "bmd: error loading directory:", err)
		os.Exit(1)
	}

	p := tea.NewProgram(v, tea.WithAltScreen(), tea.WithMouseCellMotion())
	if _, err := p.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "bmd: error running TUI:", err)
		os.Exit(1)
	}
}

func runViewer(filePath string) {
	// Step 1: Read file.
	data, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Fprintf(os.Stderr, "bmd: file not found: %s\n", filePath)
		} else {
			fmt.Fprintf(os.Stderr, "bmd: error reading file %s: %v\n", filePath, err)
		}
		os.Exit(1)
	}

	// Step 2: Parse markdown to AST.
	doc, err := parser.ParseMarkdown(string(data))
	if err != nil {
		fmt.Fprintf(os.Stderr, "bmd: error parsing markdown: %v\n", err)
		os.Exit(1)
	}

	// Step 3: Detect terminal width.
	termWidth := terminal.DetectTerminalWidth()

	// Step 4: Load saved theme preference or detect default.
	cfg, _ := config.Load() // ignore errors; use default if config missing
	th := theme.NewThemeByName(theme.ThemeName(cfg.Theme))

	// Step 5: Create viewer model.
	v := tui.New(doc, filePath, th, termWidth)

	// Step 6: Launch bubbletea TUI in alt screen with mouse support.
	p := tea.NewProgram(v, tea.WithAltScreen(), tea.WithMouseCellMotion())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "bmd: error running TUI: %v\n", err)
		os.Exit(1)
	}
}

// runImport handles `bmd import <file.tar.gz|s3://...> [--dir <dest>]`.
func runImport(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: bmd import <knowledge.tar.gz|s3://...> [--dir <dest>]")
	}

	source := args[0]
	destDir := "."
	for i := 1; i < len(args); i++ {
		if args[i] == "--dir" && i+1 < len(args) {
			destDir = args[i+1]
			i++
		}
	}

	absDestDir, err := filepath.Abs(destDir)
	if err != nil {
		return fmt.Errorf("resolve dir %q: %w", destDir, err)
	}

	fmt.Fprintf(os.Stderr, "Importing knowledge from %s to %s...\n", source, absDestDir)

	var result *knowledge.ImportResult

	if strings.HasPrefix(source, "s3://") {
		result, err = knowledge.ImportKnowledgeFromS3(source, absDestDir)
	} else {
		result, err = knowledge.ImportKnowledgeTar(source, absDestDir)
	}
	if err != nil {
		return err
	}

	fmt.Fprintf(os.Stderr, "  Extracted %d markdown files\n", result.FileCount)
	if result.Metadata != nil {
		fmt.Fprintf(os.Stderr, "  Version: %s\n", result.Metadata.Version)
		if result.Metadata.Checksum != "" {
			fmt.Fprintf(os.Stderr, "  Checksum: %s (valid)\n", result.Metadata.Checksum)
		}
		fmt.Fprintf(os.Stderr, "  Created: %s\n", result.Metadata.CreatedAt.Format("2006-01-02 15:04:05 UTC"))
	}
	if result.DBPath != "" {
		fmt.Fprintf(os.Stderr, "  Database: %s\n", result.DBPath)
	}
	fmt.Fprintf(os.Stderr, "  Destination: %s\n", result.ExtractDir)
	fmt.Fprintln(os.Stderr, "  Import complete. Indexes loaded (no rebuild needed).")

	return nil
}

// runServe handles `bmd serve --mcp [--headless] [--watch] [--knowledge-tar <file>]`.
func runServe(args []string) error {
	hasMCP := false
	headless := false
	watchEnabled := false
	knowledgeTar := ""

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--mcp":
			hasMCP = true
		case "--headless":
			headless = true
		case "--watch":
			watchEnabled = true
		case "--knowledge-tar":
			if i+1 < len(args) {
				knowledgeTar = args[i+1]
				i++
			} else {
				return fmt.Errorf("--knowledge-tar requires a file path")
			}
		}
	}

	if !hasMCP {
		return fmt.Errorf("usage: bmd serve --mcp [--headless] [--knowledge-tar <file>]")
	}

	var baseDir, dbPath string

	if knowledgeTar != "" {
		// Load from knowledge tar: extract to temp dir, use pre-built DB.
		fmt.Fprintf(os.Stderr, "Loading knowledge from %s...\n", knowledgeTar)
		result, err := knowledge.ImportKnowledgeTar(knowledgeTar, "")
		if err != nil {
			return fmt.Errorf("load knowledge tar: %w", err)
		}
		baseDir = result.ExtractDir
		dbPath = result.DBPath
		if dbPath == "" {
			return fmt.Errorf("knowledge tar does not contain a database")
		}
		fmt.Fprintf(os.Stderr, "  Loaded %d files, database at %s\n", result.FileCount, dbPath)
	} else {
		// Use current directory.
		var err error
		baseDir, err = os.Getwd()
		if err != nil {
			return fmt.Errorf("cannot get current directory: %w", err)
		}
		dbPath = filepath.Join(baseDir, ".bmd", "knowledge.db")
	}

	if headless {
		fmt.Fprintf(os.Stderr, "Starting headless MCP server (dir=%s)...\n", baseDir)
	}

	if watchEnabled {
		fmt.Fprintf(os.Stderr, "Watch mode enabled: graph updates will be incremental (agents use bmd/watch_start tool).\n")
	}

	srv := bmcmp.NewServer(baseDir, dbPath)
	if err := srv.Start(context.Background()); err != nil {
		return fmt.Errorf("MCP server error: %w", err)
	}

	return nil
}

func usage() {
	fmt.Fprint(os.Stderr, `Usage: bmd COMMAND [OPTIONS]

Browse markdown files in current directory:
  bmd                         Auto-detects and browses .md files
  bmd --browse [DIR]          Explicit directory browse mode

View a single markdown file:
  bmd file.md

Knowledge commands:
  bmd index [DIR] [OPTIONS]
    --dir DIR                 Directory to index (default: .)
    --db DB                   Database path (default: .bmd/knowledge.db)
    --strategy pageindex      Use PageIndex for semantic indexing (optional)
    --model MODEL             LLM model for PageIndex (default: claude-sonnet-4-5)
    --pageindex-bin PATH      Path to pageindex CLI (default: pageindex)
    --ignore-dirs DIRS        Comma-separated directory patterns to ignore (appends to defaults)
    --ignore-files PATTERNS   Comma-separated file patterns to ignore
    -A, --include-hidden      Include hidden directories and files (default: skip .* dirs)
    --no-ignore-defaults      Disable all default ignore patterns

  bmd query TERM [DIR] [OPTIONS]
    --dir DIR                 Directory to search (default: .)
    --strategy bm25|pageindex Search strategy (default: bm25)
    --model MODEL             LLM model for PageIndex (default: claude-sonnet-4-5)
    --format json|text|csv    Output format (default: json)
    --top N                   Max results (default: 10)

  bmd context QUERY [OPTIONS]
    --dir DIR                 Directory to search (default: .)
    --format markdown|json    Output format (default: markdown)
    --top N                   Max sections (default: 5)

  bmd depends SERVICE [DIR] [OPTIONS]
    --transitive              Include transitive dependencies
    --min-confidence <0-1>    Filter dependencies below this confidence score
    --show-confidence         Display confidence scores in output
    --include-signals         Show signal type breakdown (link/mention/llm)
    --format json|text|dot    Output format (default: json)

  bmd components [SUBCOMMAND] [OPTIONS]
    list [--dir DIR] [--format table|json]
      List all detected components
    search QUERY [--dir DIR] [--format table|json]
      Search components by name or ID
    inspect COMPONENT_ID [--dir DIR] [--format table|json]
      Detailed view of a single component with all relationships
    graph [--dir DIR] [--format ascii|json]
      Show component dependency graph (edges with confidence scores)
    --format json|text        Output format for legacy usage (default: json)

  bmd debug [OPTIONS]
    --component COMPONENT     Component name to debug (required)
    --query QUERY             What are you debugging? (optional context)
    --depth N                 BFS traversal depth (1-5, default: 2)
    --output json|text        Output format (default: json)
    --dir DIR                 Directory to scan for components (default: .)
    Aggregate documentation and relationships for debugging a component.
    Runs BFS from the target component to discover all related services
    and outputs a STATUS-01 compliant DebugContext for agent troubleshooting.

  bmd relationships [OPTIONS]
    --from <component>        Show downstream dependencies of this component
    --to <component>          Show upstream dependents of this component
    --confidence <0-1>        Filter by minimum confidence threshold
    --include-signals         Show signal breakdown (link/mention/llm)
    --format table|json|dot   Output format (default: table)

  bmd relationships-review [OPTIONS]
    --dir DIR                        Directory containing manifest files (default: .)
    --accept-all                     Auto-accept all discovered relationships
    --reject-all                     Reject all discovered relationships
    --edit                           Open manifest in $EDITOR for manual review
    --export-to PATH                 Save accepted relationships to specific path
    --llm-validate                   Validate pending relationships via LLM subprocess
    --llm-validate-bin PATH          Path to LLM validator binary (default: pageindex)
    --llm-model MODEL                LLM model for validation (default: claude-sonnet-4-5)
    --auto-accept-threshold N        Auto-accept if LLM confidence >= N (0.0 = off)
    --auto-reject-threshold N        Auto-reject if LLM confidence < N (0.0 = off)
    Review discovered relationships and persist review decisions.
    Discovered relationships are written by 'bmd index' to .bmd-relationships-discovered.yaml.
    User decisions are saved to .bmd-relationships.yaml, which is merged into the graph on index.
    With --llm-validate, each pending relationship is validated via LLM before review.

  bmd graph [SERVICE] [OPTIONS]
    --format dot|json         Output format (default: json)

  bmd crawl --from-multiple FILE[,FILE...] [OPTIONS]
    --from-multiple FILES     Comma-separated starting files
    --dir DIR                 Directory that was indexed (default: .)
    --direction DIR           backward|forward|both (default: backward)
    --depth N                 Max traversal depth (default: 3)
    --format FMT              json|tree|dot|list (default: json)

  bmd export [OPTIONS]
    --from DIR                Source directory to export (default: .)
    --output FILE             Output tar.gz file path (default: knowledge.tar.gz)
    --version VER             Semantic version tag (e.g. 2.0.0)
    --git-version             Auto-detect version from git describe --tags
    --publish S3_URI          Publish to S3 after export (e.g. s3://bucket/path)
    Package markdown files + indexes + metadata into a versioned tar.gz archive
    with SHA256 checksums and git provenance metadata.

  bmd import <knowledge.tar.gz|s3://...> [OPTIONS]
    --dir DIR                 Destination directory (default: .)
    Extract a knowledge archive, validate checksums, and load pre-built indexes.
    Supports local files and S3 URIs (requires AWS CLI for S3).

  bmd watch [OPTIONS]
    --dir DIR                 Directory to watch (default: .)
    --interval-ms N           Poll interval in ms (default: 500)
    Monitor .md changes and update indexes incrementally in real-time.
    Prints change events to stderr. Press Ctrl+C to stop.

  bmd serve --mcp [OPTIONS]
    --headless                Skip TUI, MCP server only
    --watch                   Log when watch sessions are created (informational)
    --knowledge-tar FILE      Load pre-built knowledge from tar.gz archive
    Run as a persistent MCP (Model Context Protocol) server on stdin/stdout.
    Exposes all knowledge tools as native MCP endpoints for agent integration.
    Tools: bmd/query, bmd/index, bmd/depends, bmd/components, bmd/graph,
           bmd/context, bmd/graph_crawl, bmd/watch_start, bmd/watch_poll, bmd/watch_stop

  bmd clean [OPTIONS]
    --dir DIR                 Directory to clean (default: .)
    Remove all BMD-generated files: database files (.bmd/, knowledge.db*),
    registry (.bmd-registry.json), and metadata (.bmd-*.yaml, .bmd-*.json).
    Useful for re-indexing or cleanup.

Examples:
  bmd                              Browse current directory
  bmd --browse ./docs              Browse specific directory
  bmd README.md                    View file
  bmd index ./docs                 Index with BM25 (default)
  bmd index ./docs --strategy pageindex  Index with semantic trees
  bmd index ./docs --ignore-dirs vendor,build  Skip additional dirs
  bmd index ./docs -A              Include hidden directories
  bmd index ./docs --no-ignore-defaults --ignore-files "*.backup"  Custom ignore rules
  bmd query "authentication"       BM25 search (fast)
  bmd query "auth" --strategy pageindex  Semantic search (needs trees)
  bmd context "how auth works"     Assemble RAG context block
  bmd depends api-gateway          Show service dependencies
  bmd depends api-gateway --show-confidence --include-signals  With signal breakdown
  bmd depends auth --min-confidence 0.8   High-confidence deps only
  bmd components list              List all detected components
  bmd components search auth       Search for "auth" components
  bmd components inspect api-gateway  Detailed component view
  bmd components graph             Show component dependency graph
  bmd components graph --format json  Component graph as JSON
  bmd debug --component payment    Debug payment component context
  bmd debug --component payment --query "Why are refunds failing?" --depth 2
  bmd relationships --from api-gateway           What does api-gateway depend on?
  bmd relationships --to auth-service            What depends on auth-service?
  bmd relationships --from api-gateway --confidence 0.8  High-confidence deps only
  bmd crawl --from-multiple api.md Crawl graph backward from api.md
  bmd crawl --from-multiple api.md,svc.md --format tree  ASCII tree view
  bmd export --from ./docs --output knowledge.tar.gz    Package for deployment
  bmd serve --headless --mcp --knowledge-tar k.tar.gz   Agent-only server
  bmd clean --dir ./docs                                Remove all BMD files

Notes:
  - Directory browser search uses BM25 (Phase 11+: PageIndex support planned)
  - PageIndex strategy requires: pip install pageindex
  - Tree files: {filename}.bmd-tree.json (created with --strategy pageindex)
  - Help: bmd -h, bmd --help, or bmd help
`)
}
