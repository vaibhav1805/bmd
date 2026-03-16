package main

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/bmd/bmd/internal/config"
	"github.com/bmd/bmd/internal/knowledge"
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
		case "graph":
			cmdErr = knowledge.CmdGraph(args[1:])
			if cmdErr != nil {
				fmt.Fprintln(os.Stderr, "bmd graph:", cmdErr)
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
	fmt.Fprint(os.Stderr, `Usage: bmd [FILE|OPTIONS]

VIEWER:
  bmd                         Auto-detect and browse .md files in current directory
  bmd --browse [DIR]          Explicit directory browse mode
  bmd file.md                 View a single markdown file

SEARCH & INDEXING:
  bmd index [DIR] [OPTIONS]
    --dir DIR                 Directory to index (default: .)
    --db DB                   Database path (default: .bmd/knowledge.db)
    --strategy pageindex      Use PageIndex for semantic indexing (optional)
    --ignore-dirs DIRS        Comma-separated directory patterns to ignore
    --ignore-files PATTERNS   Comma-separated file patterns to ignore
    -A, --include-hidden      Include hidden directories and files
    --no-ignore-defaults      Disable all default ignore patterns

  bmd query TERM [OPTIONS]
    --dir DIR                 Directory to search (default: .)
    --strategy bm25|pageindex Search strategy (default: bm25)
    --format json|text|csv    Output format (default: json)
    --top N                   Max results (default: 10)

GRAPH:
  bmd graph [OPTIONS]
    --format dot|json         Output format (default: json)
    View markdown link structure as a graph

HELP:
  bmd -h, --help, help        Show this message

Examples:
  bmd                         Browse current directory
  bmd --browse ./docs         Browse specific directory
  bmd README.md               View file
  bmd index ./docs            Index with BM25 (default)
  bmd index ./docs --strategy pageindex  Index with semantic trees
  bmd query "authentication"  Search (fast BM25)
  bmd query "auth" --strategy pageindex  Semantic search (requires trees)
  bmd graph                   Show link graph as JSON

For advanced features (dependencies, components, relationships, discovery):
  → Use graphmd: https://github.com/vaibhav1805/graphmd
`)
}
