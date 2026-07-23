package tui

import (
	"os"
	"testing"

	"github.com/bmd/bmd/internal/theme"
	tea "github.com/charmbracelet/bubbletea"
)

// TestMouseMsg_RoutesToActiveChild_NotFileViewClickHandler is the CR-01
// regression test (32-REVIEW.md): a left-click while a child model
// (DirectoryModel here) is active must be routed to that child, never fall
// through to the file-view click handler that matches against v.links —
// otherwise a click on the directory listing can silently reload whatever
// file v.links was last built for.
func TestMouseMsg_RoutesToActiveChild_NotFileViewClickHandler(t *testing.T) {
	dir := makeTempDir(t, map[string]string{
		"a.md": "# A\n\n[go to b](./b.md)\n",
		"b.md": "# B\n",
	})
	defer os.RemoveAll(dir)

	v := NewDirectoryViewer(dir, theme.NewTheme(), 80)
	v.Height = 24
	if err := v.LoadDirectory(dir); err != nil {
		t.Fatalf("LoadDirectory error: %v", err)
	}

	// Open a.md from the directory listing — this populates v.links with
	// the "./b.md" link's LineIndex.
	v = pressKeySettled(v, tea.KeyMsg{Type: tea.KeyEnter})
	if v.FilePath == "" || dirModel(v) != nil {
		t.Fatalf("expected a file to be open with activeChild=nil, got FilePath=%q activeChild=%T", v.FilePath, v.activeChild)
	}
	if len(v.links.Links) == 0 {
		t.Fatalf("expected v.links to be populated after opening a.md with a real link")
	}
	openedPath := v.FilePath

	// Go back to the directory listing — CR-01's defense-in-depth fix
	// requires this to clear the stale link registry and cursor.
	vv, cmd := v.BackToDirectory()
	v = settleCmd(vv, cmd)
	if dirModel(v) == nil {
		t.Fatalf("expected DirectoryModel restored after BackToDirectory")
	}
	if len(v.links.Links) != 0 {
		t.Fatalf("expected v.links cleared after BackToDirectory, got %d entries", len(v.links.Links))
	}
	if v.hasCursor {
		t.Fatalf("expected v.hasCursor cleared after BackToDirectory")
	}

	// A left-click anywhere on the directory listing must be routed to the
	// DirectoryModel (a no-op for mouse messages), never trigger
	// followLink/loadFile against the (now-cleared, but even if it weren't,
	// stale) link registry.
	m, _ := v.Update(tea.MouseMsg{Y: 1, X: 0, Action: tea.MouseActionPress, Button: tea.MouseButtonLeft})
	vv2, ok := m.(*Viewer)
	if !ok {
		t.Fatalf("expected *Viewer from Update, got %T", m)
	}
	if dirModel(vv2) == nil {
		t.Fatalf("expected activeChild to remain *DirectoryModel after a click while a child is active, got %T", vv2.activeChild)
	}
	if vv2.FilePath != openedPath {
		t.Fatalf("expected FilePath to remain %q (unchanged by the click), got %q", openedPath, vv2.FilePath)
	}
	if vv2.currentView != "directory" {
		t.Fatalf("expected currentView to remain 'directory', got %q", vv2.currentView)
	}
}
