package main

import (
	"fmt"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "Usage: bmd <file.md>")
		fmt.Fprintln(os.Stderr, "  Render a markdown file beautifully in your terminal.")
		os.Exit(1)
	}

	filePath := os.Args[1]

	data, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Fprintf(os.Stderr, "bmd: file not found: %s\n", filePath)
		} else {
			fmt.Fprintf(os.Stderr, "bmd: error reading file %s: %v\n", filePath, err)
		}
		os.Exit(1)
	}

	// TODO: parse and render — for now just print raw content
	fmt.Print(string(data))
}
