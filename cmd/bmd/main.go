package main

import (
	"fmt"
	"os"

	"github.com/bmd/bmd/internal/parser"
	"github.com/bmd/bmd/internal/renderer"
	"github.com/bmd/bmd/internal/terminal"
	"github.com/bmd/bmd/internal/theme"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "Usage: bmd <file.md>")
		fmt.Fprintln(os.Stderr, "  Render a markdown file beautifully in your terminal.")
		os.Exit(1)
	}

	filePath := os.Args[1]

	// Step 1: Read file
	data, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Fprintf(os.Stderr, "bmd: file not found: %s\n", filePath)
		} else {
			fmt.Fprintf(os.Stderr, "bmd: error reading file %s: %v\n", filePath, err)
		}
		os.Exit(1)
	}

	// Step 2: Parse markdown to AST
	doc, err := parser.ParseMarkdown(string(data))
	if err != nil {
		fmt.Fprintf(os.Stderr, "bmd: error parsing markdown: %v\n", err)
		os.Exit(1)
	}

	// Step 3: Detect terminal width
	termWidth := terminal.DetectTerminalWidth()

	// Step 4: Create theme based on terminal background detection
	th := theme.NewTheme()

	// Step 5: Render AST to string
	r := renderer.NewRenderer(th, termWidth)
	output := r.Render(doc)

	// Step 6: Print to stdout
	fmt.Print(output)
}
