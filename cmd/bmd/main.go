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
		case "export":
			cmdErr = knowledge.CmdExport(args[1:])
			if cmdErr != nil {
				fmt.Fprintln(os.Stderr, "bmd export:", cmdErr)
				os.Exit(1)
			}
			return
		case "serve":
			if len(args) < 2 || args[1] != "--mcp" {
				fmt.Fprintf(os.Stderr, "Usage: bmd serve --mcp\n")
				os.Exit(1)
			}
			cwd, err := os.Getwd()
			if err != nil {
				fmt.Fprintln(os.Stderr, "bmd: cannot get current directory:", err)
				os.Exit(1)
			}
			dbPath := filepath.Join(cwd, ".bmd", "knowledge.db")
			srv := bmcmp.NewServer(cwd, dbPath)
			if err := srv.Start(context.Background()); err != nil {
				fmt.Fprintln(os.Stderr, "bmd: MCP server error:", err)
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
    --db DB                   Database path (default: knowledge.db)
    --strategy pageindex      Use PageIndex for semantic indexing (optional)
    --model MODEL             LLM model for PageIndex (default: claude-sonnet-4-5)
    --pageindex-bin PATH      Path to pageindex CLI (default: pageindex)

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
    --format json|text|dot    Output format (default: json)

  bmd components [DIR] [OPTIONS]
    --format json|text        Output format (default: json)

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
    Package markdown files + indexes + metadata into a portable tar.gz archive.

  bmd serve --mcp [OPTIONS]
    --headless                Skip TUI, MCP server only
    --knowledge-tar FILE      Load pre-built knowledge from tar.gz archive
    Run as a persistent MCP (Model Context Protocol) server on stdin/stdout.
    Exposes all knowledge tools as native MCP endpoints for agent integration.
    Tools: bmd/query, bmd/index, bmd/depends, bmd/components, bmd/graph,
           bmd/context, bmd/graph_crawl

Examples:
  bmd                              Browse current directory
  bmd --browse ./docs              Browse specific directory
  bmd README.md                    View file
  bmd index ./docs                 Index with BM25 (default)
  bmd index ./docs --strategy pageindex  Index with semantic trees
  bmd query "authentication"       BM25 search (fast)
  bmd query "auth" --strategy pageindex  Semantic search (needs trees)
  bmd context "how auth works"     Assemble RAG context block
  bmd depends api-gateway          Show service dependencies
  bmd crawl --from-multiple api.md Crawl graph backward from api.md
  bmd crawl --from-multiple api.md,svc.md --format tree  ASCII tree view
  bmd export --from ./docs --output knowledge.tar.gz    Package for deployment
  bmd serve --headless --mcp --knowledge-tar k.tar.gz   Agent-only server

Notes:
  - Directory browser search uses BM25 (Phase 11+: PageIndex support planned)
  - PageIndex strategy requires: pip install pageindex
  - Tree files: {filename}.bmd-tree.json (created with --strategy pageindex)
  - Help: bmd -h, bmd --help, or bmd help
`)
}
