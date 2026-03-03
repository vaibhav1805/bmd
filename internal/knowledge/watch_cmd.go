package knowledge

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"time"
)

// CmdWatch implements `bmd watch`.
//
// It creates a FileWatcher + IncrementalUpdater for the target directory,
// prints a startup message, then streams change events to stderr until
// the process is interrupted (SIGINT/SIGTERM).
//
// Usage:
//
//	bmd watch [--dir DIR] [--interval-ms N]
func CmdWatch(args []string) error {
	fs := flag.NewFlagSet("watch", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	var dir string
	var intervalMs int
	fs.StringVar(&dir, "dir", ".", "Directory to watch for .md changes")
	fs.IntVar(&intervalMs, "interval-ms", 500, "Poll interval in milliseconds")

	if err := fs.Parse(args); err != nil {
		return fmt.Errorf("watch: %w", err)
	}

	absDir, err := filepath.Abs(dir)
	if err != nil {
		return fmt.Errorf("watch: resolve dir %q: %w", dir, err)
	}

	// Verify directory exists before attempting to scan it.
	if _, statErr := os.Stat(absDir); statErr != nil {
		return fmt.Errorf("watch: directory %q not accessible: %w", absDir, statErr)
	}

	fmt.Fprintf(os.Stderr, "bmd watch: watching %s (interval: %dms)\n", absDir, intervalMs)
	fmt.Fprintln(os.Stderr, "Press Ctrl+C to stop.")

	// Load or build index and graph.
	docs, scanErr := ScanDirectory(absDir)
	if scanErr != nil {
		return fmt.Errorf("watch: scan: %w", scanErr)
	}

	idx := NewIndex()
	if err := idx.Build(docs); err != nil {
		return fmt.Errorf("watch: build index: %w", err)
	}

	graph := NewGraph()
	ex := NewExtractor(absDir)
	for i := range docs {
		_ = graph.AddNode(&Node{ID: docs[i].ID, Title: docs[i].Title, Type: "document"})
		edges := ex.Extract(&docs[i])
		for _, edge := range edges {
			_ = graph.AddEdge(edge)
		}
	}

	reg := NewComponentRegistry()
	reg.InitFromGraph(graph, docs)

	watcher := NewFileWatcher(absDir, time.Duration(intervalMs)*time.Millisecond)

	onChange := func(evt WatchEvent) {
		kindStr := map[WatchEventKind]string{
			WatchCreated:  "created",
			WatchModified: "modified",
			WatchDeleted:  "deleted",
		}[evt.Kind]
		fmt.Fprintf(os.Stderr, "[watch] %s: %s\n", kindStr, evt.RelPath)
	}

	updater := NewIncrementalUpdater(absDir, watcher, idx, graph, reg, nil, onChange)
	watcher.Start()
	updater.Start()

	// Block until SIGINT (Ctrl+C).
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()
	<-ctx.Done()

	updater.Stop()

	stats := updater.Stats()
	fmt.Fprintf(os.Stderr, "\nbmd watch: stopped. indexed=%d removed=%d skipped=%d errors=%d\n",
		stats.FilesIndexed, stats.FilesRemoved, stats.FilesSkipped, stats.Errors)
	return nil
}
