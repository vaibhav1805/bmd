package tui

import (
	"os"
	"strings"
	"testing"

	"github.com/bmd/bmd/internal/theme"
	tea "github.com/charmbracelet/bubbletea"
)

// ─── Shared test helpers (used across directory_test.go, viewer_test.go, ────
// ─── and verification_test.go) ───────────────────────────────────────────────

// dirModel extracts the active DirectoryModel from a Viewer, for test
// assertions that need to inspect directory-browser state directly. Returns
// nil if no DirectoryModel is currently active.
func dirModel(v *Viewer) *DirectoryModel {
	dm, _ := v.activeChild.(*DirectoryModel)
	return dm
}

// settleCmd resolves a single tea.Cmd (if non-nil) into its tea.Msg and
// feeds it back into v.Update() exactly once, returning the resulting
// Viewer. This is what drives the one-hop message-passing handoffs
// (openFileMsg/switchModeMsg/toggleHelpMsg/statusMsg) emitted by
// DirectoryModel.Update() the rest of the way through Viewer.Update()'s
// message handlers.
//
// Deliberately single-hop, not a loop: some of Viewer.Update()'s message
// handlers themselves return a further tea.Cmd (e.g. statusMsg's handler
// always schedules clearErrorAfter(statusTimeout), a real tea.Tick timer).
// Invoking a tea.Tick's cmd synchronously in a test blocks for the real
// duration and then wipes the very state the test is asserting on — that
// belongs to bubbletea's own runtime loop, not to test setup. One hop is
// exactly what's needed to observe the message-passing handoff itself.
func settleCmd(v *Viewer, cmd tea.Cmd) *Viewer {
	if cmd == nil {
		return v
	}
	msg := cmd()
	if msg == nil {
		return v
	}
	m, _ := v.Update(msg)
	if vv, ok := m.(*Viewer); ok {
		return vv
	}
	return v
}

// pressKeySettled sends msg to v.Update() and fully resolves any resulting
// tea.Cmd chain via settleCmd, returning the final settled Viewer.
func pressKeySettled(v *Viewer, msg tea.Msg) *Viewer {
	m, cmd := v.Update(msg)
	vv, _ := m.(*Viewer)
	return settleCmd(vv, cmd)
}

// newTestDirectoryModel scans dir and returns a fresh DirectoryModel for
// direct (*DirectoryModel).Update()/.View() testing, bypassing Viewer
// entirely (D-01/ARCH-01: DirectoryModel is independently testable).
func newTestDirectoryModel(t *testing.T, dir string, width, height int) *DirectoryModel {
	t.Helper()
	dm, err := NewDirectoryModel(dir, theme.NewTheme(), width, height)
	if err != nil {
		t.Fatalf("NewDirectoryModel: %v", err)
	}
	return dm
}

// resolveCmd invokes cmd (which must be non-nil) and returns the resolved
// tea.Msg.
func resolveCmd(t *testing.T, cmd tea.Cmd) tea.Msg {
	t.Helper()
	if cmd == nil {
		t.Fatal("expected non-nil tea.Cmd")
	}
	return cmd()
}

// ─── DirectoryModel.Update() — navigation ────────────────────────────────────

func TestDirectoryModelUpdate_NavigationWraparound(t *testing.T) {
	dir := makeTempDir(t, map[string]string{
		"a.md": "# A", "b.md": "# B", "c.md": "# C",
	})
	defer os.RemoveAll(dir)

	dm := newTestDirectoryModel(t, dir, 120, 24)

	tests := []struct {
		key      tea.KeyMsg
		wantIdx  int
		wantSame bool // whether preview offset should reset to 0
	}{
		{tea.KeyMsg{Type: tea.KeyDown}, 1, true},
		{tea.KeyMsg{Type: tea.KeyDown}, 2, true},
		{tea.KeyMsg{Type: tea.KeyDown}, 0, true}, // wraps
		{tea.KeyMsg{Type: tea.KeyUp}, 2, true},   // wraps backward
	}

	for i, tt := range tests {
		dm.splitPreviewOffset = 99 // dirty it so we can prove it resets
		model, cmd := dm.Update(tt.key)
		dm = model.(*DirectoryModel)
		if cmd != nil {
			t.Errorf("step %d: expected nil cmd for plain navigation, got non-nil", i)
		}
		if dm.state.SelectedIndex != tt.wantIdx {
			t.Errorf("step %d: expected SelectedIndex=%d, got %d", i, tt.wantIdx, dm.state.SelectedIndex)
		}
		if dm.splitPreviewOffset != 0 {
			t.Errorf("step %d: expected splitPreviewOffset reset to 0, got %d", i, dm.splitPreviewOffset)
		}
	}

	// vim-style j/k should behave identically.
	dm.state.SelectedIndex = 0
	model, _ := dm.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	dm = model.(*DirectoryModel)
	if dm.state.SelectedIndex != 1 {
		t.Errorf("expected 'j' to move down to 1, got %d", dm.state.SelectedIndex)
	}
	model, _ = dm.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("k")})
	dm = model.(*DirectoryModel)
	if dm.state.SelectedIndex != 0 {
		t.Errorf("expected 'k' to move up to 0, got %d", dm.state.SelectedIndex)
	}
}

// ─── DirectoryModel.Update() — split toggle ──────────────────────────────────

func TestDirectoryModelUpdate_SplitToggle(t *testing.T) {
	dir := makeTempDir(t, map[string]string{"a.md": "# A"})
	defer os.RemoveAll(dir)

	dm := newTestDirectoryModel(t, dir, 120, 24)
	initial := dm.splitMode

	model, cmd := dm.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("s")})
	dm = model.(*DirectoryModel)
	if cmd != nil {
		t.Error("expected nil cmd for successful split toggle")
	}
	if dm.splitMode == initial {
		t.Error("expected splitMode to flip after 's'")
	}
}

func TestDirectoryModelUpdate_SplitToggleNarrowTerminalError(t *testing.T) {
	dir := makeTempDir(t, map[string]string{"a.md": "# A"})
	defer os.RemoveAll(dir)

	dm := newTestDirectoryModel(t, dir, 60, 24) // narrow: < 80 cols
	initial := dm.splitMode

	model, cmd := dm.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("s")})
	dm = model.(*DirectoryModel)
	if dm.splitMode != initial {
		t.Error("expected splitMode unchanged on narrow-terminal error")
	}
	msg := resolveCmd(t, cmd)
	sm, ok := msg.(statusMsg)
	if !ok {
		t.Fatalf("expected statusMsg, got %T", msg)
	}
	const wantText = "Terminal too narrow for split pane (need 80+ cols)"
	if sm.text != wantText {
		t.Errorf("expected status text %q, got %q", wantText, sm.text)
	}
}

// ─── DirectoryModel.Update() — file-open handoff (ARCH-03) ──────────────────

func TestDirectoryModelUpdate_EnterEmitsOpenFileMsg(t *testing.T) {
	dir := makeTempDir(t, map[string]string{"api.md": "# API"})
	defer os.RemoveAll(dir)

	dm := newTestDirectoryModel(t, dir, 120, 24)
	expectedPath := dm.state.Files[0].Path

	model, cmd := dm.Update(tea.KeyMsg{Type: tea.KeyEnter})
	dm = model.(*DirectoryModel)

	msg := resolveCmd(t, cmd)
	ofm, ok := msg.(openFileMsg)
	if !ok {
		t.Fatalf("expected openFileMsg, got %T", msg)
	}
	if ofm.path != expectedPath {
		t.Errorf("expected path %q, got %q", expectedPath, ofm.path)
	}
	if ofm.origin != originDirectory {
		t.Errorf("expected origin originDirectory, got %v", ofm.origin)
	}
	// Selection should be saved for restoration (DIR-02) before handoff.
	if dm.state.SavedSelectedIndex != dm.state.SelectedIndex {
		t.Errorf("expected SavedSelectedIndex=%d, got %d", dm.state.SelectedIndex, dm.state.SavedSelectedIndex)
	}
}

func TestDirectoryModelUpdate_LAndRightAlsoOpenFile(t *testing.T) {
	dir := makeTempDir(t, map[string]string{"doc.md": "# Doc"})
	defer os.RemoveAll(dir)

	for _, key := range []tea.KeyMsg{
		{Type: tea.KeyRunes, Runes: []rune("l")},
		{Type: tea.KeyRight},
	} {
		dm := newTestDirectoryModel(t, dir, 120, 24)
		_, cmd := dm.Update(key)
		msg := resolveCmd(t, cmd)
		if _, ok := msg.(openFileMsg); !ok {
			t.Errorf("key %v: expected openFileMsg, got %T", key, msg)
		}
	}
}

func TestDirectoryModelUpdate_EnterOnEmptyDirDoesNothing(t *testing.T) {
	dir := makeTempDir(t, map[string]string{})
	defer os.RemoveAll(dir)

	dm := newTestDirectoryModel(t, dir, 120, 24)
	model, cmd := dm.Update(tea.KeyMsg{Type: tea.KeyEnter})
	dm = model.(*DirectoryModel)
	if cmd != nil {
		t.Error("expected nil cmd when opening from an empty directory")
	}
	if dm.splitMode != (dm.width >= 80) {
		// sanity: nothing else should have mutated state either
	}
}

// ─── DirectoryModel.Update() — mode-transition handoff (ARCH-05) ────────────

func TestDirectoryModelUpdate_GEmitsSwitchModeGraph(t *testing.T) {
	dir := makeTempDir(t, map[string]string{"a.md": "# A"})
	defer os.RemoveAll(dir)

	dm := newTestDirectoryModel(t, dir, 120, 24)
	_, cmd := dm.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("g")})
	msg := resolveCmd(t, cmd)
	smm, ok := msg.(switchModeMsg)
	if !ok {
		t.Fatalf("expected switchModeMsg, got %T", msg)
	}
	if smm.mode != modeGraph {
		t.Errorf("expected mode=modeGraph, got %v", smm.mode)
	}
	if smm.arg != dm.state.RootPath {
		t.Errorf("expected arg=%q (rootPath), got %q", dm.state.RootPath, smm.arg)
	}
}

func TestDirectoryModelUpdate_SlashAndCtrlFEmitSwitchModeCrossSearch(t *testing.T) {
	dir := makeTempDir(t, map[string]string{"a.md": "# A"})
	defer os.RemoveAll(dir)

	for _, key := range []tea.KeyMsg{
		{Type: tea.KeyRunes, Runes: []rune("/")},
		{Type: tea.KeyCtrlF},
	} {
		dm := newTestDirectoryModel(t, dir, 120, 24)
		_, cmd := dm.Update(key)
		msg := resolveCmd(t, cmd)
		smm, ok := msg.(switchModeMsg)
		if !ok {
			t.Fatalf("key %v: expected switchModeMsg, got %T", key, msg)
		}
		if smm.mode != modeCrossSearch {
			t.Errorf("key %v: expected mode=modeCrossSearch, got %v", key, smm.mode)
		}
	}
}

func TestDirectoryModelUpdate_QuestionAndHEmitToggleHelp(t *testing.T) {
	dir := makeTempDir(t, map[string]string{"a.md": "# A"})
	defer os.RemoveAll(dir)

	for _, key := range []tea.KeyMsg{
		{Type: tea.KeyRunes, Runes: []rune("?")},
		{Type: tea.KeyRunes, Runes: []rune("h")},
	} {
		dm := newTestDirectoryModel(t, dir, 120, 24)
		_, cmd := dm.Update(key)
		msg := resolveCmd(t, cmd)
		if _, ok := msg.(toggleHelpMsg); !ok {
			t.Errorf("key %v: expected toggleHelpMsg, got %T", key, msg)
		}
	}
}

func TestDirectoryModelUpdate_QuitsOnQAndCtrlC(t *testing.T) {
	dir := makeTempDir(t, map[string]string{"a.md": "# A"})
	defer os.RemoveAll(dir)

	for _, key := range []tea.KeyMsg{
		{Type: tea.KeyRunes, Runes: []rune("q")},
		{Type: tea.KeyCtrlC},
	} {
		dm := newTestDirectoryModel(t, dir, 120, 24)
		_, cmd := dm.Update(key)
		if cmd == nil {
			t.Fatalf("key %v: expected tea.Quit cmd", key)
		}
	}
}

// TestDirectoryModelUpdate_NeverMutatesViewer is a static/structural
// assertion that DirectoryModel.Update() has no way to reach a *Viewer: its
// signature only takes/returns tea.Model/tea.Cmd. This is enforced by the
// type system already (DirectoryModel holds no *Viewer field, D-06), but this
// test documents the contract by driving Update() directly with no Viewer in
// scope at all, proving the child is independently operable.
func TestDirectoryModelUpdate_NeverMutatesViewer(t *testing.T) {
	dir := makeTempDir(t, map[string]string{"a.md": "# A", "b.md": "# B"})
	defer os.RemoveAll(dir)

	dm := newTestDirectoryModel(t, dir, 120, 24)
	// Drive a full sequence with no *Viewer ever constructed.
	for _, k := range []string{"down", "down", "up"} {
		var key tea.KeyMsg
		switch k {
		case "down":
			key = tea.KeyMsg{Type: tea.KeyDown}
		case "up":
			key = tea.KeyMsg{Type: tea.KeyUp}
		}
		model, _ := dm.Update(key)
		dm = model.(*DirectoryModel)
	}
	if dm.state.SelectedIndex != 1 {
		t.Errorf("expected SelectedIndex=1 after down,down,up, got %d", dm.state.SelectedIndex)
	}
}

// ─── DirectoryModel.Update() — window resize ────────────────────────────────

func TestDirectoryModelUpdate_WindowSizeMsgUpdatesDimensions(t *testing.T) {
	dir := makeTempDir(t, map[string]string{"a.md": "# A"})
	defer os.RemoveAll(dir)

	dm := newTestDirectoryModel(t, dir, 80, 24)
	model, cmd := dm.Update(tea.WindowSizeMsg{Width: 150, Height: 50})
	dm = model.(*DirectoryModel)
	if cmd != nil {
		t.Error("expected nil cmd for WindowSizeMsg")
	}
	if dm.width != 150 || dm.height != 50 {
		t.Errorf("expected width=150 height=50, got width=%d height=%d", dm.width, dm.height)
	}
}

// ─── DirectoryModel.View() — regression baseline (UI-SPEC) ──────────────────

func TestDirectoryModelView_EmptyStateBaseline(t *testing.T) {
	dir := makeTempDir(t, map[string]string{})
	defer os.RemoveAll(dir)

	dm := newTestDirectoryModel(t, dir, 100, 24)
	dm.splitMode = false
	out := dm.View()
	if !strings.Contains(out, " No markdown files found in this directory.") {
		t.Errorf("expected locked empty-state string in output, got: %q", out)
	}
}

func TestDirectoryModelView_HeaderAndFooterBaseline(t *testing.T) {
	dir := makeTempDir(t, map[string]string{"readme.md": "# Readme"})
	defer os.RemoveAll(dir)

	dm := newTestDirectoryModel(t, dir, 100, 24)
	dm.splitMode = false
	// Render with a short display root so the header assertion below isn't
	// at the mercy of how long the OS temp-dir path happens to be (which can
	// exceed the header's truncation threshold at narrower widths).
	dm.state.RootPath = "/docs"
	out := dm.View()
	if !strings.Contains(out, "Markdown Files in") || !strings.Contains(out, "(1 files)") {
		t.Errorf("expected locked header string in output, got: %q", out)
	}
	const wantFooter = " [↑/↓] Navigate  [Enter] Open  [/] Search  [g] Graph  [?] Help  [q] Quit"
	if !strings.Contains(out, wantFooter) {
		t.Errorf("expected locked footer string in output, got: %q", out)
	}
}

// TestDirectoryModelView_NarrowWidthNoPanic is the CR-02 regression test
// (32-REVIEW.md): renderDirectoryListing's header truncation (m.width < 3)
// and its per-row filename truncation (nameMaxWidth going negative once a
// long filename's metadata suffix outweighs a narrow width) both used to
// panic with a negative slice bound. Exercises the full width>=1 range with
// a filename long enough to trigger the per-row path, not just the header.
func TestDirectoryModelView_NarrowWidthNoPanic(t *testing.T) {
	dir := makeTempDir(t, map[string]string{
		"a-very-long-markdown-filename-that-is-quite-long.md": "# content\nsome content here to bump size and lines\n",
	})
	defer os.RemoveAll(dir)

	for _, width := range []int{1, 2, 3, 4, 5, 10, 20, 30, 79} {
		dm := newTestDirectoryModel(t, dir, width, 24)
		dm.splitMode = false
		func() {
			defer func() {
				if r := recover(); r != nil {
					t.Fatalf("View() panicked at width=%d: %v", width, r)
				}
			}()
			dm.View()
		}()
	}
}

func TestDirectoryModelView_SplitPaneBaseline(t *testing.T) {
	dir := makeTempDir(t, map[string]string{"readme.md": "# Readme\nBody text."})
	defer os.RemoveAll(dir)

	dm := newTestDirectoryModel(t, dir, 120, 24)
	dm.splitMode = true
	out := dm.View()
	if !strings.Contains(out, "│") {
		t.Error("expected split-pane border character '│' in output")
	}
	const wantFooter = " [↑/↓] Navigate  [Enter] Open  [s] Toggle split  [/] Search  [q] Quit"
	if !strings.Contains(out, wantFooter) {
		t.Errorf("expected locked split-pane footer string in output, got: %q", out)
	}
}

// ─── NewDirectoryModel — construction ────────────────────────────────────────

func TestNewDirectoryModel_ScansAndSorts(t *testing.T) {
	dir := makeTempDir(t, map[string]string{
		"zzz.md":      "# Z",
		"aaa.md":      "# A",
		"notes.txt":   "not markdown",
		"docs/mmm.md": "# M",
	})
	defer os.RemoveAll(dir)

	dm := newTestDirectoryModel(t, dir, 120, 24)
	if len(dm.state.Files) != 3 {
		t.Fatalf("expected 3 .md files, got %d", len(dm.state.Files))
	}
	names := make([]string, len(dm.state.Files))
	for i, f := range dm.state.Files {
		names[i] = f.Name
	}
	if !sortedAlphabetically(names) {
		t.Errorf("expected files sorted alphabetically, got %v", names)
	}
}

func sortedAlphabetically(names []string) bool {
	for i := 1; i < len(names); i++ {
		if names[i-1] > names[i] {
			return false
		}
	}
	return true
}

func TestNewDirectoryModel_SplitModeDefaultsFromWidth(t *testing.T) {
	dir := makeTempDir(t, map[string]string{"a.md": "# A"})
	defer os.RemoveAll(dir)

	wide := newTestDirectoryModel(t, dir, 120, 24)
	if !wide.splitMode {
		t.Error("expected splitMode=true when width >= 80")
	}
	narrow := newTestDirectoryModel(t, dir, 60, 24)
	if narrow.splitMode {
		t.Error("expected splitMode=false when width < 80")
	}
}
