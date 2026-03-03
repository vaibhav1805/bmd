package knowledge

import "time"

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
// Stub implementation — types only, no behaviour yet.
type FileWatcher struct {
	dir      string
	interval time.Duration
	Events   chan WatchEvent
	stop     chan struct{}
	snapshot map[string]snapshotEntry
}

// NewFileWatcher creates a new FileWatcher stub. Not yet functional.
func NewFileWatcher(dir string, interval time.Duration) *FileWatcher {
	return &FileWatcher{
		dir:      dir,
		interval: interval,
		Events:   make(chan WatchEvent, 32),
		stop:     make(chan struct{}),
		snapshot: make(map[string]snapshotEntry),
	}
}

// Start is a stub — does not launch any goroutine.
func (w *FileWatcher) Start() {}

// Stop is a stub — does not stop anything.
func (w *FileWatcher) Stop() {}
