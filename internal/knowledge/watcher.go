package knowledge

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// WatchEventKind identifies the type of filesystem change detected.
type WatchEventKind int

const (
	WatchCreated  WatchEventKind = iota // a new .md file appeared
	WatchModified                       // an existing .md file was changed
	WatchDeleted                        // an existing .md file was removed
)

// WatchEvent is emitted by FileWatcher for each detected change.
type WatchEvent struct {
	Kind    WatchEventKind
	Path    string // absolute path of the affected file
	RelPath string // relative to watched dir, forward-slash separators
}

// snapshotEntry holds the recorded state of a file at poll time.
type snapshotEntry struct {
	ModTime time.Time
	Size    int64
}

// FileWatcher watches a directory tree for .md file changes using polling.
//
// It emits WatchEvent values on the Events channel for Created, Modified,
// and Deleted events. The poll interval defaults to 500ms. Start() launches
// a background goroutine; Stop() terminates it and closes Events.
//
// Hidden directories (names starting with ".") and well-known vendor dirs
// (node_modules etc.) are skipped — consistent with ScanDirectory behaviour.
type FileWatcher struct {
	dir      string
	interval time.Duration
	Events   chan WatchEvent
	stop     chan struct{}
	once     sync.Once
	snapshot map[string]snapshotEntry
}

// NewFileWatcher creates a FileWatcher for the given directory with the
// provided poll interval. A buffered Events channel (capacity 32) is
// created; callers should drain it regularly to avoid dropped events.
func NewFileWatcher(dir string, interval time.Duration) *FileWatcher {
	absDir, err := filepath.Abs(dir)
	if err != nil {
		absDir = dir
	}
	return &FileWatcher{
		dir:      absDir,
		interval: interval,
		Events:   make(chan WatchEvent, 32),
		stop:     make(chan struct{}),
		snapshot: make(map[string]snapshotEntry),
	}
}

// Start begins polling the watched directory. If the directory does not
// exist, Start closes Events and returns immediately without launching
// a goroutine.
//
// Start should be called once; calling it multiple times is undefined.
func (w *FileWatcher) Start() {
	if _, err := os.Stat(w.dir); err != nil {
		close(w.Events)
		return
	}

	// Build the initial snapshot silently (no events for pre-existing files).
	w.snapshot = w.buildSnapshot()

	go func() {
		ticker := time.NewTicker(w.interval)
		defer ticker.Stop()
		for {
			select {
			case <-w.stop:
				close(w.Events)
				return
			case <-ticker.C:
				w.poll()
			}
		}
	}()
}

// Stop signals the polling goroutine to exit. It is safe to call Stop
// multiple times; subsequent calls are no-ops. After Stop returns, no
// further events will be sent and the Events channel will be closed.
func (w *FileWatcher) Stop() {
	w.once.Do(func() {
		close(w.stop)
	})
}

// poll scans the directory tree, compares against the current snapshot, emits
// events for any differences, then updates the snapshot.
func (w *FileWatcher) poll() {
	current := w.buildSnapshot()

	// Detect Created and Modified events.
	for path, entry := range current {
		prev, existed := w.snapshot[path]
		if !existed {
			w.send(WatchEvent{Kind: WatchCreated, Path: path, RelPath: w.relPath(path)})
			continue
		}
		if entry.ModTime != prev.ModTime || entry.Size != prev.Size {
			w.send(WatchEvent{Kind: WatchModified, Path: path, RelPath: w.relPath(path)})
		}
	}

	// Detect Deleted events.
	for path := range w.snapshot {
		if _, exists := current[path]; !exists {
			w.send(WatchEvent{Kind: WatchDeleted, Path: path, RelPath: w.relPath(path)})
		}
	}

	w.snapshot = current
}

// buildSnapshot walks the watched directory and records mtime+size for each
// .md file that is not inside a hidden or excluded directory.
func (w *FileWatcher) buildSnapshot() map[string]snapshotEntry {
	snap := make(map[string]snapshotEntry)
	_ = filepath.WalkDir(w.dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil // soft-skip unreadable entries
		}

		name := d.Name()

		if d.IsDir() {
			if path == w.dir {
				return nil // descend into watched root
			}
			// Skip hidden directories (names starting with ".").
			if strings.HasPrefix(name, ".") {
				return filepath.SkipDir
			}
			// Skip well-known vendor/tooling directories.
			if _, skip := hiddenDirs[name]; skip {
				return filepath.SkipDir
			}
			return nil
		}

		// Only regular .md files (skip symlinks and other special files).
		if d.Type()&fs.ModeSymlink != 0 {
			return nil
		}
		if !d.Type().IsRegular() {
			return nil
		}
		if !strings.HasSuffix(strings.ToLower(name), ".md") {
			return nil
		}

		info, err := d.Info()
		if err != nil {
			return nil // soft-skip
		}
		snap[path] = snapshotEntry{ModTime: info.ModTime(), Size: info.Size()}
		return nil
	})
	return snap
}

// send emits an event to the Events channel using a non-blocking send.
// If the channel buffer is full the event is dropped to prevent stalling
// the polling goroutine.
func (w *FileWatcher) send(evt WatchEvent) {
	select {
	case w.Events <- evt:
	default:
	}
}

// relPath converts an absolute path to a forward-slash relative path.
func (w *FileWatcher) relPath(absPath string) string {
	rel, err := filepath.Rel(w.dir, absPath)
	if err != nil {
		return absPath
	}
	return filepath.ToSlash(rel)
}
