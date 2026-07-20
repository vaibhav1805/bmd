package tui

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/bmd/bmd/internal/parser"
	"github.com/bmd/bmd/internal/renderer"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/fsnotify/fsnotify"
)

// reloadDebounce is the quiet-period fsnotify events must coalesce within
// before a reload is triggered (D-07, range 200-300ms).
const reloadDebounce = 250 * time.Millisecond

// waitForFileChange returns a tea.Cmd that blocks on a single read from the
// debounced-event channel and returns fileChangedMsg. Re-issue this from
// Update() every time fileChangedMsg is handled to keep listening (Pattern 1,
// official bubbletea channel-drain tea.Cmd shape).
func waitForFileChange(ch <-chan string) tea.Cmd {
	return func() tea.Msg {
		path := <-ch
		return fileChangedMsg{path: path}
	}
}

// reloadFile re-reads, re-parses, and re-renders path, mirroring loadFile's
// render pipeline but WITHOUT resetting v.Offset or clearing search state
// (D-05) — this is the sole behavioral difference from loadFile/
// loadFileNoHistory, which is why this is its own function rather than a
// reuse of either.
//
// Read/parse errors are swallowed silently (D-06): the file may be mid-write
// when the debounced event fires, and the next debounced event will retry.
// The last-good render is never clobbered by a transient failure.
func (v *Viewer) reloadFile(path string) (*Viewer, tea.Cmd) {
	data, err := os.ReadFile(path)
	if err != nil {
		return v, nil
	}
	doc, err := parser.ParseMarkdown(string(data))
	if err != nil {
		return v, nil
	}

	r := renderer.NewRenderer(v.Theme, v.Width).WithLinkSentinels().WithDocDir(filepath.Dir(path))
	rendered := r.Render(doc)
	v.Doc = doc
	v.rawLines = strings.Split(rendered, "\n")
	v.Lines = stripAllSentinels(v.rawLines)
	v.rendered = strings.Join(v.Lines, "\n")
	v.links = BuildRegistry(v.rawLines)
	v.virtualMode = len(v.Lines) > virtualThreshold

	// D-05: clamp instead of reset — never reset v.Offset to 0.
	if v.Offset >= len(v.Lines) {
		v.Offset = clamp(len(v.Lines)-1, 0, v.maxOffset())
	}

	return v, nil
}

// debounce coalesces rapid repeated calls into a single fire after waitFor
// has elapsed since the most recent call. Single-timer form (not a per-path
// map): D-08's exact-basename filter already narrows fsnotify events to at
// most one file of interest at a time (RESEARCH.md Assumption A2), so a
// single package-level timer is sufficient — bmd only ever runs one Viewer
// per process.
//
// debounceFire is stored separately from the *time.Timer and re-read at fire
// time (via runDebounceFire) rather than being baked into time.AfterFunc's
// callback once: a bare time.AfterFunc(waitFor, fire) followed only by
// timer.Reset() on later calls would permanently keep the FIRST call's fire
// closure, silently ignoring every later call's closure even though its
// deadline is what actually gets extended. Always firing the latest call's
// closure is both the correct debounce semantics and what keeps independent
// callers/tests from cross-contaminating each other's stale closures.
var (
	debounceMu    sync.Mutex
	debounceTimer *time.Timer
	debounceFire  func()
)

func debounce(waitFor time.Duration, fire func()) {
	debounceMu.Lock()
	defer debounceMu.Unlock()
	debounceFire = fire
	if debounceTimer == nil {
		debounceTimer = time.AfterFunc(waitFor, runDebounceFire)
		return
	}
	debounceTimer.Reset(waitFor)
}

// runDebounceFire invokes whichever fire closure was passed to the most
// recent debounce() call before the timer fired.
func runDebounceFire() {
	debounceMu.Lock()
	fire := debounceFire
	debounceMu.Unlock()
	if fire != nil {
		fire()
	}
}

// startWatching lazily creates the single long-lived *fsnotify.Watcher and
// buffered reloadCh on first use (one watcher for the whole session, not
// recreated per load — RESEARCH.md Open Question 1), then (re)points it at
// path's parent directory, removing the previously-watched directory first
// if it changed (Pitfall 2 — no unbounded accumulation of watched dirs).
//
// Returns waitForFileChange(v.reloadCh) the first time watching starts, so
// the tea.Cmd enters the bubbletea runtime loop; returns nil on subsequent
// redirects, since the loop is already running and re-issuing another
// waitForFileChange would start a second, redundant listener on the same
// channel.
func (v *Viewer) startWatching(path string) tea.Cmd {
	firstStart := v.watcher == nil
	if firstStart {
		w, err := fsnotify.NewWatcher()
		if err != nil {
			// Live-reload silently unavailable if the OS watcher can't be
			// created (D-06 — reload is a nice-to-have, never a hard error).
			return nil
		}
		v.watcher = w
		v.reloadCh = make(chan string, 1)
	}

	dir := filepath.Dir(path)
	if dir != v.watchedDir {
		if v.watchedDir != "" {
			_ = v.watcher.Remove(v.watchedDir)
		}
		if err := v.watcher.Add(dir); err == nil {
			v.watchedDir = dir
		}
	}

	if firstStart {
		go v.watchLoop()
		return waitForFileChange(v.reloadCh)
	}
	return nil
}

// watchLoop ranges over the watcher's Events channel for the lifetime of the
// session, filters to events matching the currently-open file's exact
// basename (D-08 — this also excludes the differently-named
// .bmd-autosave-* shadow file, Pitfall 3), and debounces matching
// Write/Create/Rename events before pushing the current file path onto
// reloadCh.
//
// ARCH-05: this goroutine never mutates Viewer state directly — it only
// reads v.FilePath (as trigger/filter data, re-validated against the
// already-open file on every event, ASVS V5 — the fsnotify event path
// itself is never passed to os.ReadFile) and writes to v.reloadCh. Only
// Viewer.Update() (via reloadFile) mutates Doc/Lines/Offset/etc.
func (v *Viewer) watchLoop() {
	for {
		select {
		case event, ok := <-v.watcher.Events:
			if !ok {
				return
			}
			if filepath.Base(event.Name) != filepath.Base(v.FilePath) {
				continue
			}
			if event.Op&(fsnotify.Write|fsnotify.Create|fsnotify.Rename) == 0 {
				continue
			}
			debounce(reloadDebounce, func() {
				v.reloadCh <- v.FilePath
			})
		case _, ok := <-v.watcher.Errors:
			if !ok {
				return
			}
			// Silent per D-06 — a watcher-level error doesn't stop the
			// session, it just means this particular event was missed.
		}
	}
}
