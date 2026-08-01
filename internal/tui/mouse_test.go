package tui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bmd/bmd/internal/parser"
	"github.com/bmd/bmd/internal/theme"
	tea "github.com/charmbracelet/bubbletea"
)

// newTestFileViewer parses content and returns a file-mode Viewer (no
// directory browser) with the given width/height, along with the path it
// was "loaded" from.
func newTestFileViewer(t *testing.T, dir, name, content string, width, height int) *Viewer {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", name, err)
	}
	doc, err := parser.ParseMarkdown(content)
	if err != nil {
		t.Fatalf("ParseMarkdown: %v", err)
	}
	v := New(doc, path, theme.NewTheme(), width)
	v.Height = height
	return v
}

// linkLine returns the document line index of the link entry whose URL
// contains needle, failing the test if none is found.
func linkLine(t *testing.T, v *Viewer, needle string) int {
	t.Helper()
	for _, l := range v.links.Links {
		if strings.Contains(l.URL, needle) {
			return l.LineIndex
		}
	}
	t.Fatalf("no link matching %q found in registry: %+v", needle, v.links.Links)
	return -1
}

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

// TestUpdateMouse_ClickOnLink_StillFollowsOnPlainClick is bmd-zag part 1's
// baseline: a plain mouse-down-then-up on a link line (no movement in
// between) must still navigate, exactly like the pre-fix immediate-follow
// behavior — only a drag should be exempted.
func TestUpdateMouse_ClickOnLink_StillFollowsOnPlainClick(t *testing.T) {
	dir := t.TempDir()
	v := newTestFileViewer(t, dir, "overview.md",
		"# Overview\n\n- [Payment Processing](payments.md) - desc\n", 100, 24)
	os.WriteFile(filepath.Join(dir, "payments.md"), []byte("# Payments\n"), 0o644)

	line := linkLine(t, v, "payments.md")

	overviewPath := filepath.Join(dir, "overview.md")
	m, _ := v.updateMouse(tea.MouseMsg{Y: line + 1, X: 5, Action: tea.MouseActionPress, Button: tea.MouseButtonLeft})
	v = m.(*Viewer)
	if v.FilePath != overviewPath {
		t.Fatalf("mid-press: expected navigation deferred until release, but already navigated to %q", v.FilePath)
	}
	m, _ = v.updateMouse(tea.MouseMsg{Y: line + 1, X: 5, Action: tea.MouseActionRelease, Button: tea.MouseButtonLeft})
	v = m.(*Viewer)

	if v.FilePath != filepath.Join(dir, "payments.md") {
		t.Errorf("expected plain click on link line to navigate to payments.md, got FilePath=%q", v.FilePath)
	}
}

// TestUpdateMouse_DragOverLink_SelectsInsteadOfNavigating is bmd-zag part 1:
// a click-drag gesture that starts on a link line must extend a text
// selection, not follow the link, since the mouse moved before release.
func TestUpdateMouse_DragOverLink_SelectsInsteadOfNavigating(t *testing.T) {
	dir := t.TempDir()
	v := newTestFileViewer(t, dir, "overview.md",
		"# Overview\n\n- [Payment Processing](payments.md) - desc\n", 100, 24)
	os.WriteFile(filepath.Join(dir, "payments.md"), []byte("# Payments\n"), 0o644)

	line := linkLine(t, v, "payments.md")

	m, _ := v.updateMouse(tea.MouseMsg{Y: line + 1, X: 2, Action: tea.MouseActionPress, Button: tea.MouseButtonLeft})
	v = m.(*Viewer)
	m, _ = v.updateMouse(tea.MouseMsg{Y: line + 1, X: 15, Action: tea.MouseActionMotion, Button: tea.MouseButtonLeft})
	v = m.(*Viewer)
	m, _ = v.updateMouse(tea.MouseMsg{Y: line + 1, X: 15, Action: tea.MouseActionRelease, Button: tea.MouseButtonLeft})
	v = m.(*Viewer)

	if strings.Contains(v.FilePath, "payments.md") {
		t.Fatalf("expected drag over link line NOT to navigate, but FilePath=%q", v.FilePath)
	}
	if !v.HasSelection() {
		t.Errorf("expected a text selection to be active after dragging over a link line")
	}
	if v.SelectedText() == "" {
		t.Errorf("expected non-empty selected text after dragging over a link line")
	}
}

// TestView_MouseHoverCursor_SurvivesBackNavigation_NoRawEscapeLeak is bmd-zag
// part 2: the mouse-hover cursor overlay used to insert its reverse-video
// marker by slicing the line at a raw rune index (insertCursorAt), which
// doesn't account for embedded ANSI color codes. When the hover position
// landed inside a colored link span — exactly what happens when a drag
// starts on a link line, navigates away, and the user then returns via
// history with the mouse left in place — it truncated the color escape
// mid-sequence, leaking a raw fragment (e.g. "141m") as literal text and
// losing that line's styling. Fixed by switching to the ANSI-aware
// insertCursorAtVisual.
func TestView_MouseHoverCursor_SurvivesBackNavigation_NoRawEscapeLeak(t *testing.T) {
	dir := t.TempDir()
	v := newTestFileViewer(t, dir, "overview.md",
		"# Overview\n\n- [Payment Processing](payments.md) - desc\n", 100, 24)
	os.WriteFile(filepath.Join(dir, "payments.md"), []byte("# Payments\n"), 0o644)

	line := linkLine(t, v, "payments.md")
	baselineLine := strings.Split(v.View(), "\n")[line+1]

	// Press-then-drag-motion onto the link line (as in the drag-over-link
	// scenario), landing the hover position mid-way through the colored
	// link text.
	m, _ := v.updateMouse(tea.MouseMsg{Y: line + 1, X: 2, Action: tea.MouseActionPress, Button: tea.MouseButtonLeft})
	v = m.(*Viewer)
	m, _ = v.updateMouse(tea.MouseMsg{Y: line + 1, X: 8, Action: tea.MouseActionMotion, Button: tea.MouseButtonLeft})
	v = m.(*Viewer)

	// Simulate a prior navigation-away-and-back (e.g. via a stray click that
	// did follow a different link, then Ctrl+B) without moving the mouse:
	// mouseRow/mouseCol persist across loadFileNoHistory by design (the
	// physical mouse didn't move), landing back on this exact line/column.
	vv, _ := v.loadFileNoHistory(filepath.Join(dir, "payments.md"))
	v = vv
	vv, _ = v.loadFileNoHistory(filepath.Join(dir, "overview.md"))
	v = vv

	out := v.View()
	lines := strings.Split(out, "\n")
	if line+1 >= len(lines) {
		t.Fatalf("rendered output too short: %d lines", len(lines))
	}
	afterLine := lines[line+1]

	// If a color escape got truncated mid-sequence, the leftover bytes (e.g.
	// "39m" or "141m") show up as literal, unstrippable text, so the plain
	// text gains extra characters compared to the pristine baseline render.
	if stripANSI(afterLine) != stripANSI(baselineLine) {
		t.Errorf("plain text of the line changed after hover-cursor overlay (raw escape fragment leaked?): got %q, want %q", stripANSI(afterLine), stripANSI(baselineLine))
	}
	// The hover cursor legitimately interrupts the link text with a
	// reverse-video single character, so check for the link's color escape
	// surviving intact (not truncated mid-sequence into a bare numeric
	// fragment like "39m") rather than for unbroken literal text.
	if !strings.Contains(afterLine, "\x1b[38;5;39m") {
		t.Errorf("expected the link's color escape sequence to survive intact, got %q", afterLine)
	}
}
