package main

import (
	"fmt"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
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
		case "depends":
			cmdErr = knowledge.CmdDepends(args[1:])
			if cmdErr != nil {
				fmt.Fprintln(os.Stderr, "bmd depends:", cmdErr)
				os.Exit(1)
			}
			return
		case "services":
			cmdErr = knowledge.CmdServices(args[1:])
			if cmdErr != nil {
				fmt.Fprintln(os.Stderr, "bmd services:", cmdErr)
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
		}
	}

	// Legacy viewer mode: requires at least one argument.
	if len(args) < 1 {
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

	// Step 4: Create theme based on terminal background detection.
	th := theme.NewTheme()

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

View a markdown file:
  bmd file.md

Knowledge commands:
  bmd index [DIR] [--dir DIR] [--db DB]
                              Build/update knowledge index
  bmd query TERM [DIR] [--dir DIR] [--format json|text|csv] [--top N]
                              Search markdown files
  bmd depends SERVICE [DIR] [--dir DIR] [--transitive] [--format json|text|dot]
                              Show service dependencies
  bmd services [DIR] [--dir DIR] [--format json|text]
                              List all detected services
  bmd graph [SERVICE] [--dir DIR] [--format dot|json]
                              Export knowledge graph

Examples:
  bmd README.md               View file
  bmd index ./docs            Index directory
  bmd query "authentication"  Search
  bmd depends api-gateway     Show dependencies
  bmd services                List services
  bmd graph --format dot | dot -Tpng > graph.png
`)
}
