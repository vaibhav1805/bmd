package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bmd/bmd/internal/ast"
	"github.com/bmd/bmd/internal/knowledge"
	"github.com/bmd/bmd/internal/theme"
	tea "github.com/charmbracelet/bubbletea"
)

// ─── Shared test helpers ──────────────────────────────────────────────────

// csModel extracts the active CrossSearchModel from a Viewer, for test
// assertions that need to inspect cross-search state directly. Returns nil
// if no CrossSearchModel is currently active.
func csModel(v *Viewer) *CrossSearchModel {
	csm, _ := v.activeChild.(*CrossSearchModel)
	return csm
}

// newTestCrossSearchModel returns a fresh CrossSearchModel for direct
// (*CrossSearchModel).Update()/.View() testing, bypassing Viewer entirely
// (D-01/ARCH-02: CrossSearchModel is independently testable).
func newTestCrossSearchModel(rootPath string, width, height int) *CrossSearchModel {
	return NewCrossSearchModel(rootPath, theme.NewTheme(), width, height)
}

// newTestViewerWithDir creates a Viewer with startDir set for cross-search
// integration tests that exercise the message-passing round-trip through
// Viewer.Update().
func newTestViewerWithDir(t *testing.T, startDir string) *Viewer {
	t.Helper()
	doc := &ast.Document{}
	v := New(doc, filepath.Join(startDir, "test.md"), theme.NewTheme(), 80)
	v.Height = 24
	return v
}

// buildTmpSearchDir creates a temporary directory with the given files (filename -> content).
func buildTmpSearchDir(t *testing.T, files map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	for name, content := range files {
		path := filepath.Join(dir, name)
		// Create parent dirs if needed.
		_ = os.MkdirAll(filepath.Dir(path), 0o755)
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatalf("buildTmpSearchDir: write %s: %v", name, err)
		}
	}
	return dir
}

// ─── CrossSearchModel.Update() — input stage: query building ────────────────

func TestCrossSearchModelUpdate_InputAccumulatesCharacters(t *testing.T) {
	m := newTestCrossSearchModel(t.TempDir(), 80, 24)

	for _, ch := range "hello" {
		key := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{ch}}
		model, cmd := m.Update(key)
		m = model.(*CrossSearchModel)
		if cmd != nil {
			t.Errorf("unexpected cmd while typing %q", string(ch))
		}
	}
	if m.input != "hello" {
		t.Errorf("expected input=%q, got %q", "hello", m.input)
	}
}

func TestCrossSearchModelUpdate_Backspace(t *testing.T) {
	m := newTestCrossSearchModel(t.TempDir(), 80, 24)
	m.input = "API"

	model, _ := m.Update(tea.KeyMsg{Type: tea.KeyBackspace})
	m = model.(*CrossSearchModel)
	if m.input != "AP" {
		t.Errorf("expected input=%q after backspace, got %q", "AP", m.input)
	}
}

func TestCrossSearchModelUpdate_BackspaceOnEmptyIsSafe(t *testing.T) {
	m := newTestCrossSearchModel(t.TempDir(), 80, 24)

	model, _ := m.Update(tea.KeyMsg{Type: tea.KeyBackspace})
	m = model.(*CrossSearchModel)
	if m.input != "" {
		t.Errorf("expected empty input after backspace on empty, got %q", m.input)
	}
}

func TestCrossSearchModelUpdate_EscCancelsInput(t *testing.T) {
	dir := t.TempDir()
	m := newTestCrossSearchModel(dir, 80, 24)
	m.input = "partial query"

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	msg := resolveCmd(t, cmd)
	smm, ok := msg.(switchModeMsg)
	if !ok {
		t.Fatalf("expected switchModeMsg, got %T", msg)
	}
	if smm.mode != modeNone {
		t.Errorf("expected mode=modeNone, got %v", smm.mode)
	}
}

func TestCrossSearchModelUpdate_CtrlFCancelsInput(t *testing.T) {
	m := newTestCrossSearchModel(t.TempDir(), 80, 24)
	m.input = "partial"

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlF})
	msg := resolveCmd(t, cmd)
	smm, ok := msg.(switchModeMsg)
	if !ok || smm.mode != modeNone {
		t.Fatalf("expected switchModeMsg{modeNone}, got %#v", msg)
	}
}

func TestCrossSearchModelUpdate_EmptyQueryEnterClosesWithoutActivating(t *testing.T) {
	m := newTestCrossSearchModel(t.TempDir(), 80, 24)
	m.input = "   " // whitespace only

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	msg := resolveCmd(t, cmd)
	smm, ok := msg.(switchModeMsg)
	if !ok {
		t.Fatalf("expected switchModeMsg, got %T", msg)
	}
	if smm.mode != modeNone {
		t.Errorf("expected mode=modeNone for empty query, got %v", smm.mode)
	}
	if m.stage != csStageInput {
		t.Error("expected stage to remain csStageInput for empty query")
	}
}

// ─── CrossSearchModel.Update() — input stage: search execution ──────────────

func TestCrossSearchModelUpdate_EnterExecutesSearchAndEntersResultsStage(t *testing.T) {
	dir := buildTmpSearchDir(t, map[string]string{
		"auth.md": "# Authentication\nJWT tokens and OAuth flow for auth.",
	})
	m := newTestCrossSearchModel(dir, 80, 24)
	m.input = "authentication"

	model, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = model.(*CrossSearchModel)
	if cmd != nil {
		t.Errorf("expected nil cmd on successful search, got non-nil")
	}
	if m.stage != csStageResults {
		t.Fatalf("expected stage=csStageResults after search, got %v", m.stage)
	}
	if len(m.results) == 0 {
		t.Error("expected at least 1 result for 'authentication'")
	}
	if m.selected != 0 {
		t.Errorf("expected selected=0 for non-empty results, got %d", m.selected)
	}
}

func TestCrossSearchModelUpdate_EnterNoMatchesEntersResultsStageWithNegativeSelection(t *testing.T) {
	dir := buildTmpSearchDir(t, map[string]string{
		"doc.md": "# Doc\nUnrelated basic content.",
	})
	m := newTestCrossSearchModel(dir, 80, 24)
	m.input = "zzznomatchxyzquux"

	model, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = model.(*CrossSearchModel)
	if m.stage != csStageResults {
		t.Fatalf("expected stage=csStageResults even with no matches, got %v", m.stage)
	}
	if len(m.results) != 0 {
		t.Errorf("expected 0 results, got %d", len(m.results))
	}
	if m.selected != -1 {
		t.Errorf("expected selected=-1 for empty results, got %d", m.selected)
	}
}

// ─── CrossSearchModel.Update() — results stage: navigation ──────────────────

func TestCrossSearchModelUpdate_NavMoveDownAndUp(t *testing.T) {
	m := newTestCrossSearchModel(t.TempDir(), 80, 24)
	m.stage = csStageResults
	m.selected = 0
	m.results = []knowledge.SearchResult{
		{RelPath: "a.md", Score: 5.0},
		{RelPath: "b.md", Score: 4.0},
		{RelPath: "c.md", Score: 3.0},
	}

	model, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = model.(*CrossSearchModel)
	if m.selected != 1 {
		t.Errorf("expected selected=1 after down, got %d", m.selected)
	}

	model, _ = m.Update(tea.KeyMsg{Type: tea.KeyUp})
	m = model.(*CrossSearchModel)
	if m.selected != 0 {
		t.Errorf("expected selected=0 after up, got %d", m.selected)
	}
}

func TestCrossSearchModelUpdate_NavClampsAtBounds(t *testing.T) {
	m := newTestCrossSearchModel(t.TempDir(), 80, 24)
	m.stage = csStageResults
	m.results = []knowledge.SearchResult{
		{RelPath: "a.md", Score: 5.0},
		{RelPath: "b.md", Score: 4.0},
	}

	m.selected = 1
	model, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = model.(*CrossSearchModel)
	if m.selected != 1 {
		t.Errorf("expected selected clamped at 1 (last index), got %d", m.selected)
	}

	m.selected = 0
	model, _ = m.Update(tea.KeyMsg{Type: tea.KeyUp})
	m = model.(*CrossSearchModel)
	if m.selected != 0 {
		t.Errorf("expected selected clamped at 0, got %d", m.selected)
	}
}

func TestCrossSearchModelUpdate_NavVimKeys(t *testing.T) {
	m := newTestCrossSearchModel(t.TempDir(), 80, 24)
	m.stage = csStageResults
	m.selected = 0
	m.results = []knowledge.SearchResult{
		{RelPath: "a.md", Score: 5.0},
		{RelPath: "b.md", Score: 4.0},
	}

	model, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	m = model.(*CrossSearchModel)
	if m.selected != 1 {
		t.Errorf("expected selected=1 after 'j', got %d", m.selected)
	}
	model, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("k")})
	m = model.(*CrossSearchModel)
	if m.selected != 0 {
		t.Errorf("expected selected=0 after 'k', got %d", m.selected)
	}
}

// ─── CrossSearchModel.Update() — file-open handoff (ARCH-03) ────────────────

func TestCrossSearchModelUpdate_LAndEnterEmitOpenFileMsg(t *testing.T) {
	absPath := filepath.Join(t.TempDir(), "target.md")
	for _, key := range []tea.KeyMsg{
		{Type: tea.KeyRunes, Runes: []rune("l")},
		{Type: tea.KeyEnter},
	} {
		m := newTestCrossSearchModel(t.TempDir(), 80, 24)
		m.stage = csStageResults
		m.selected = 0
		m.results = []knowledge.SearchResult{{RelPath: "target.md", Score: 5.0, Path: absPath}}

		_, cmd := m.Update(key)
		msg := resolveCmd(t, cmd)
		ofm, ok := msg.(openFileMsg)
		if !ok {
			t.Fatalf("key %v: expected openFileMsg, got %T", key, msg)
		}
		if ofm.path != absPath {
			t.Errorf("key %v: expected path %q, got %q", key, absPath, ofm.path)
		}
		if ofm.origin != originSearch {
			t.Errorf("key %v: expected origin originSearch, got %v", key, ofm.origin)
		}
	}
}

func TestCrossSearchModelUpdate_EnterOnEmptyResultsDoesNothing(t *testing.T) {
	m := newTestCrossSearchModel(t.TempDir(), 80, 24)
	m.stage = csStageResults
	m.selected = -1
	m.results = nil

	model, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = model.(*CrossSearchModel)
	if cmd != nil {
		t.Error("expected nil cmd when opening from empty results")
	}
}

// ─── CrossSearchModel.Update() — mode-transition handoff (ARCH-05) ──────────

func TestCrossSearchModelUpdate_HAndEscEmitSwitchModeDirectory(t *testing.T) {
	dir := t.TempDir()
	for _, key := range []tea.KeyMsg{
		{Type: tea.KeyRunes, Runes: []rune("h")},
		{Type: tea.KeyEsc},
	} {
		m := newTestCrossSearchModel(dir, 80, 24)
		m.stage = csStageResults
		m.results = []knowledge.SearchResult{{RelPath: "a.md", Score: 5.0}}

		_, cmd := m.Update(key)
		msg := resolveCmd(t, cmd)
		smm, ok := msg.(switchModeMsg)
		if !ok {
			t.Fatalf("key %v: expected switchModeMsg, got %T", key, msg)
		}
		if smm.mode != modeDirectory {
			t.Errorf("key %v: expected mode=modeDirectory, got %v", key, smm.mode)
		}
		if smm.arg != dir {
			t.Errorf("key %v: expected arg=%q (rootPath), got %q", key, dir, smm.arg)
		}
	}
}

func TestCrossSearchModelUpdate_HExitsToModeNoneWhenNoRootPath(t *testing.T) {
	m := newTestCrossSearchModel("", 80, 24)
	m.stage = csStageResults
	m.results = []knowledge.SearchResult{{RelPath: "a.md", Score: 5.0}}

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("h")})
	msg := resolveCmd(t, cmd)
	smm, ok := msg.(switchModeMsg)
	if !ok || smm.mode != modeNone {
		t.Fatalf("expected switchModeMsg{modeNone} when rootPath is empty, got %#v", msg)
	}
}

func TestCrossSearchModelUpdate_QQuits(t *testing.T) {
	m := newTestCrossSearchModel(t.TempDir(), 80, 24)
	m.stage = csStageResults
	m.results = []knowledge.SearchResult{{RelPath: "a.md", Score: 5.0}}

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	if cmd == nil {
		t.Fatal("expected tea.Quit cmd for 'q'")
	}
}

func TestCrossSearchModelUpdate_SlashReopensInputWithPriorQuery(t *testing.T) {
	m := newTestCrossSearchModel(t.TempDir(), 80, 24)
	m.stage = csStageResults
	m.query = "old query"
	m.results = []knowledge.SearchResult{{RelPath: "a.md", Score: 5.0}}

	model, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("/")})
	m = model.(*CrossSearchModel)
	if cmd != nil {
		t.Error("expected nil cmd when reopening search prompt")
	}
	if m.stage != csStageInput {
		t.Error("expected stage=csStageInput after '/'")
	}
	if m.input != "old query" {
		t.Errorf("expected input=%q, got %q", "old query", m.input)
	}
}

// ─── CrossSearchModel.Update() — window resize ──────────────────────────────

func TestCrossSearchModelUpdate_WindowSizeMsgUpdatesDimensions(t *testing.T) {
	m := newTestCrossSearchModel(t.TempDir(), 80, 24)

	model, cmd := m.Update(tea.WindowSizeMsg{Width: 150, Height: 50})
	m = model.(*CrossSearchModel)
	if cmd != nil {
		t.Error("expected nil cmd for WindowSizeMsg")
	}
	if m.width != 150 || m.height != 50 {
		t.Errorf("expected width=150 height=50, got width=%d height=%d", m.width, m.height)
	}
}

// TestCrossSearchModelUpdate_NeverMutatesViewer is a static/structural
// assertion that CrossSearchModel.Update() has no way to reach a *Viewer:
// its signature only takes/returns tea.Model/tea.Cmd, and it holds no
// *Viewer field (D-06). This test documents the contract by driving a full
// input->search->nav->open sequence with no Viewer ever constructed.
func TestCrossSearchModelUpdate_NeverMutatesViewer(t *testing.T) {
	dir := buildTmpSearchDir(t, map[string]string{
		"a.md": "# A\nmicroservices content",
		"b.md": "# B\nmicroservices content too",
	})
	m := newTestCrossSearchModel(dir, 80, 24)

	for _, ch := range "microservices" {
		model, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{ch}})
		m = model.(*CrossSearchModel)
	}
	model, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = model.(*CrossSearchModel)
	if m.stage != csStageResults {
		t.Fatal("expected results stage after search")
	}
	model, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = model.(*CrossSearchModel)
	if len(m.results) > 1 && m.selected != 1 {
		t.Errorf("expected selected=1 after down (multiple results), got %d", m.selected)
	}
}

// ─── CrossSearchModel — search error handling ────────────────────────────────

func TestCrossSearchModelUpdate_SearchErrorClosesAndShowsStatus(t *testing.T) {
	// A NUL-byte rootPath deterministically fails index open/build
	// (mkdir: invalid argument) on every platform this project supports.
	m := newTestCrossSearchModel(string([]byte{0}), 80, 24)
	m.input = "anything"

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("expected a batched cmd on search error")
	}
	msg := resolveCmd(t, cmd)
	batch, ok := msg.(tea.BatchMsg)
	if !ok {
		t.Fatalf("expected tea.BatchMsg on search error, got %T", msg)
	}
	var sawModeNone, sawStatus bool
	for _, sub := range batch {
		switch m := sub().(type) {
		case switchModeMsg:
			if m.mode == modeNone {
				sawModeNone = true
			}
		case statusMsg:
			if strings.HasPrefix(m.text, "Search error:") {
				sawStatus = true
			}
		}
	}
	if !sawModeNone {
		t.Error("expected batched switchModeMsg{modeNone} on search error")
	}
	if !sawStatus {
		t.Error("expected batched statusMsg with 'Search error:' prefix")
	}
}

// ─── CrossSearchModel.View() — regression baseline (UI-SPEC) ────────────────

func TestCrossSearchModelView_InputStageReturnsEmpty(t *testing.T) {
	m := newTestCrossSearchModel(t.TempDir(), 80, 24)
	if out := m.View(); out != "" {
		t.Errorf("expected empty View() during input stage, got %q", out)
	}
}

func TestCrossSearchModelView_TitleShowsQueryCountAndStrategy(t *testing.T) {
	m := newTestCrossSearchModel(t.TempDir(), 80, 24)
	m.stage = csStageResults
	m.query = "microservices"
	m.strategy = "bm25"
	m.selected = 0
	m.results = []knowledge.SearchResult{
		{RelPath: "svc.md", Score: 8.2},
	}

	out := m.View()
	if !strings.Contains(out, `Search Results for "microservices" (1 result) [bm25]`) {
		t.Errorf("expected locked title string in output, got: %q", out)
	}
}

func TestCrossSearchModelView_ShowsFilenamesAndScores(t *testing.T) {
	m := newTestCrossSearchModel(t.TempDir(), 80, 24)
	m.stage = csStageResults
	m.query = "service"
	m.selected = 0
	m.results = []knowledge.SearchResult{
		{RelPath: "payment.md", Score: 7.0},
		{RelPath: "auth.md", Score: 5.5},
	}

	out := m.View()
	for _, want := range []string{"payment.md", "auth.md", "7.0", "5.5"} {
		if !strings.Contains(out, want) {
			t.Errorf("expected %q in results output, got:\n%s", want, out)
		}
	}
}

func TestCrossSearchModelView_EmptyStateBaseline(t *testing.T) {
	m := newTestCrossSearchModel(t.TempDir(), 80, 24)
	m.stage = csStageResults
	m.query = "zzznomatch"
	m.selected = -1
	m.results = []knowledge.SearchResult{}

	out := m.View()
	const want = `No matches found for "zzznomatch"`
	if !strings.Contains(out, want) {
		t.Errorf("expected locked empty-state string in output, got: %q", out)
	}
}

func TestCrossSearchModelView_SelectedResultReverseVideo(t *testing.T) {
	m := newTestCrossSearchModel(t.TempDir(), 80, 24)
	m.stage = csStageResults
	m.query = "test"
	m.selected = 0
	m.results = []knowledge.SearchResult{
		{RelPath: "first.md", Score: 9.0},
		{RelPath: "second.md", Score: 5.0},
	}

	out := m.View()
	if !strings.Contains(out, "\x1b[7m") {
		t.Error("expected reverse-video escape code for selected result")
	}
}

func TestCrossSearchModelView_FooterBaseline(t *testing.T) {
	m := newTestCrossSearchModel(t.TempDir(), 80, 24)
	m.stage = csStageResults
	m.query = "test"
	m.selected = 0
	m.results = []knowledge.SearchResult{{RelPath: "a.md", Score: 5.0}}

	out := m.View()
	const wantFooter = "[↑/↓] Navigate  [l/Enter] Open  [h/Esc] Back  [/] New Search"
	if !strings.Contains(out, wantFooter) {
		t.Errorf("expected locked footer string in output, got: %q", out)
	}
}

func TestCrossSearchModelView_ShowsSnippetAndFallback(t *testing.T) {
	dir := buildTmpSearchDir(t, map[string]string{
		"svc.md": "# Service\nThe payment microservices handle billing and invoices.",
	})
	m := newTestCrossSearchModel(dir, 80, 24)
	m.stage = csStageResults
	m.query = "microservices"
	m.selected = 0
	m.results = []knowledge.SearchResult{
		{RelPath: "svc.md", Score: 8.2, Path: filepath.Join(dir, "svc.md"), Snippet: "payment microservices handle billing"},
	}

	out := strings.ToLower(m.View())
	if !strings.Contains(out, "microservices") {
		t.Error("expected snippet content containing 'microservices' in rendered output")
	}

	// Fallback snippet when the file no longer exists.
	m.results = []knowledge.SearchResult{
		{RelPath: "missing.md", Score: 5.0, Path: "/nonexistent/missing.md", Snippet: "fallback snippet text"},
	}
	out = m.View()
	if !strings.Contains(out, "fallback snippet text") {
		t.Error("expected fallback snippet in rendered output when file is missing")
	}
}

func TestCrossSearchModelView_DisplaysStrategyIndicators(t *testing.T) {
	tests := []struct {
		name     string
		strategy string
		want     string
		notWant  string
	}{
		{"bm25", "bm25", "[bm25]", "[pageindex]"},
		{"pageindex", "pageindex", "[pageindex]", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := newTestCrossSearchModel(t.TempDir(), 80, 24)
			m.stage = csStageResults
			m.query = "test"
			m.strategy = tt.strategy
			m.selected = 0
			m.results = []knowledge.SearchResult{{RelPath: "doc.md", Score: 5.0}}

			out := m.View()
			if !strings.Contains(out, tt.want) {
				t.Errorf("expected %q in output, got:\n%s", tt.want, out)
			}
			if tt.notWant != "" && strings.Contains(out, tt.notWant) {
				t.Errorf("expected no %q in output", tt.notWant)
			}
		})
	}
}

// ─── NewCrossSearchModel — construction ──────────────────────────────────────

func TestNewCrossSearchModel_StartsInInputStage(t *testing.T) {
	m := newTestCrossSearchModel(t.TempDir(), 80, 24)
	if m.stage != csStageInput {
		t.Error("expected new CrossSearchModel to start in csStageInput")
	}
	if m.input != "" {
		t.Errorf("expected empty input, got %q", m.input)
	}
}

// ─── SearchAllFiles — strategy selection (moved from Viewer) ────────────────

func TestCrossSearchModel_SearchAllFiles_DefaultStrategyIsBM25(t *testing.T) {
	dir := buildTmpSearchDir(t, map[string]string{
		"doc.md": "# Document\nThis is about authentication and microservices.",
	})
	m := newTestCrossSearchModel(dir, 80, 24)
	t.Setenv("BMD_STRATEGY", "")

	_, strategy, err := m.SearchAllFiles("authentication")
	if err != nil {
		t.Fatalf("SearchAllFiles error: %v", err)
	}
	if strategy != "bm25" {
		t.Errorf("expected default strategy 'bm25', got %q", strategy)
	}
}

func TestCrossSearchModel_SearchAllFiles_StrategyEnvVarBM25(t *testing.T) {
	dir := buildTmpSearchDir(t, map[string]string{
		"doc.md": "# Document\nContent about services.",
	})
	m := newTestCrossSearchModel(dir, 80, 24)
	t.Setenv("BMD_STRATEGY", "bm25")

	_, strategy, err := m.SearchAllFiles("services")
	if err != nil {
		t.Fatalf("SearchAllFiles error: %v", err)
	}
	if strategy != "bm25" {
		t.Errorf("expected strategy 'bm25', got %q", strategy)
	}
}

func TestCrossSearchModel_SearchAllFiles_PageIndexFallback(t *testing.T) {
	dir := buildTmpSearchDir(t, map[string]string{
		"doc.md": "# Document\nContent about authentication microservices.",
	})
	m := newTestCrossSearchModel(dir, 80, 24)
	t.Setenv("BMD_STRATEGY", "pageindex")

	// No .bmd-tree.json files exist, so PageIndex will fail → BM25 fallback.
	_, strategy, _ := m.SearchAllFiles("authentication")
	if strategy != "bm25" {
		t.Errorf("expected fallback strategy 'bm25', got %q", strategy)
	}
}

func TestCrossSearchModel_SearchAllFiles_Method(t *testing.T) {
	dir := buildTmpSearchDir(t, map[string]string{
		"service.md": "# Service\nThe payment service handles transactions.",
	})
	m := newTestCrossSearchModel(dir, 80, 24)

	results, _, err := m.SearchAllFiles("payment")
	if err != nil {
		t.Fatalf("SearchAllFiles error: %v", err)
	}
	if len(results) == 0 {
		t.Error("expected at least 1 result for 'payment'")
	}
}

// ─── internal/knowledge integration (unchanged — package-level, not Viewer-dependent) ──

func TestSearchAllDocumentsEmptyQuery(t *testing.T) {
	dir := buildTmpSearchDir(t, map[string]string{
		"a.md": "# Doc A\nSome content",
	})

	results, err := knowledge.SearchAllDocuments(dir, "", 10)
	if err != nil {
		t.Fatalf("Unexpected error on empty query: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("Expected 0 results for empty query, got %d", len(results))
	}
}

func TestSearchAllDocumentsSingleFile(t *testing.T) {
	dir := buildTmpSearchDir(t, map[string]string{
		"readme.md": "# README\nThis document explains the authentication service.",
	})

	results, err := knowledge.SearchAllDocuments(dir, "authentication", 10)
	if err != nil {
		t.Fatalf("SearchAllDocuments error: %v", err)
	}
	if len(results) == 0 {
		t.Fatal("Expected at least 1 result for 'authentication'")
	}
}

func TestSearchAllDocumentsMultipleFiles(t *testing.T) {
	dir := buildTmpSearchDir(t, map[string]string{
		"auth.md":     "# Auth Service\nJWT authentication and authorization flows.",
		"database.md": "# Database\nSQL schema and migration scripts.",
		"api.md":      "# API Gateway\nRESTful API endpoints and authentication.",
	})

	results, err := knowledge.SearchAllDocuments(dir, "authentication", 10)
	if err != nil {
		t.Fatalf("SearchAllDocuments error: %v", err)
	}
	if len(results) < 2 {
		t.Errorf("Expected at least 2 results for 'authentication', got %d", len(results))
	}
}

func TestSearchAllDocumentsResultsSortedByScore(t *testing.T) {
	dir := buildTmpSearchDir(t, map[string]string{
		"high.md":  "# High Match\nmicroservices microservices microservices architecture",
		"low.md":   "# Low Match\none mention of microservices",
		"other.md": "# Other\ncompletely unrelated content about databases",
	})

	results, err := knowledge.SearchAllDocuments(dir, "microservices", 10)
	if err != nil {
		t.Fatalf("SearchAllDocuments error: %v", err)
	}
	if len(results) < 2 {
		t.Errorf("Expected at least 2 results, got %d", len(results))
	}
	for i := 1; i < len(results); i++ {
		if results[i].Score > results[i-1].Score {
			t.Errorf("Results not sorted by score: results[%d].Score=%.2f > results[%d].Score=%.2f",
				i, results[i].Score, i-1, results[i-1].Score)
		}
	}
}

func TestSearchAllDocumentsResultCountAccurate(t *testing.T) {
	dir := buildTmpSearchDir(t, map[string]string{
		"doc.md": "# Doc\nThis doc contains API and gateway and authentication terms.",
	})

	results, err := knowledge.SearchAllDocuments(dir, "API gateway", 10)
	if err != nil {
		t.Fatalf("SearchAllDocuments error: %v", err)
	}
	if len(results) == 0 {
		t.Skip("No results — terms may be stop-word filtered; skipping count test")
	}
	if results[0].MatchCount < 1 {
		t.Errorf("Expected MatchCount >= 1, got %d", results[0].MatchCount)
	}
}

func TestSearchAllDocumentsAutoBuildsIndex(t *testing.T) {
	dir := buildTmpSearchDir(t, map[string]string{
		"notes.md": "# Notes\nImportant deployment procedures.",
	})
	// No knowledge.db exists yet — should auto-build.
	results, err := knowledge.SearchAllDocuments(dir, "deployment", 10)
	if err != nil {
		t.Fatalf("SearchAllDocuments error on auto-build: %v", err)
	}
	_ = results
}

func TestSearchAllDocumentsNoMatchReturnsEmpty(t *testing.T) {
	dir := buildTmpSearchDir(t, map[string]string{
		"doc.md": "# Doc\nSome basic content here.",
	})

	results, err := knowledge.SearchAllDocuments(dir, "xyzquuxnonexistent", 10)
	if err != nil {
		t.Fatalf("SearchAllDocuments error: %v", err)
	}
	if results == nil {
		t.Error("Expected non-nil empty slice, got nil")
	}
	if len(results) != 0 {
		t.Errorf("Expected 0 results for non-matching query, got %d", len(results))
	}
}

func TestCrossSearchSpecialCharactersNoCrash(t *testing.T) {
	dir := buildTmpSearchDir(t, map[string]string{
		"doc.md": "# Doc\nSome content here.",
	})

	specialQueries := []string{"", "  ", "a", "API", "hello world", "!@#$%", "\"quoted\""}
	for _, q := range specialQueries {
		t.Run(fmt.Sprintf("query=%q", q), func(t *testing.T) {
			_, err := knowledge.SearchAllDocuments(dir, q, 10)
			if err != nil {
				t.Logf("SearchAllDocuments(%q) returned error (acceptable): %v", q, err)
			}
		})
	}
}

func TestSearchAllDocumentsLargeResultSet(t *testing.T) {
	files := make(map[string]string, 20)
	for i := 0; i < 20; i++ {
		name := fmt.Sprintf("doc%02d.md", i)
		files[name] = fmt.Sprintf("# Document %d\nThis document talks about microservices and deployment pipeline %d.", i, i)
	}
	dir := buildTmpSearchDir(t, files)

	results, err := knowledge.SearchAllDocuments(dir, "microservices", 50)
	if err != nil {
		t.Fatalf("SearchAllDocuments error: %v", err)
	}
	if len(results) == 0 {
		t.Error("Expected results for 'microservices' across 20 files")
	}
	for i := 1; i < len(results); i++ {
		if results[i].Score > results[i-1].Score {
			t.Errorf("Results not sorted: results[%d].Score=%.2f > results[%d].Score=%.2f",
				i, results[i].Score, i-1, results[i-1].Score)
		}
	}
}

// ====== DIR-04: Snippet Extraction Tests (unchanged — internal/knowledge) ======

func TestGetContextSnippetBasicMatch(t *testing.T) {
	dir := buildTmpSearchDir(t, map[string]string{
		"doc.md": "# Title\nThis document explains the microservices architecture pattern in detail.",
	})
	snippet := knowledge.GetContextSnippet(filepath.Join(dir, "doc.md"), "microservices", 100)
	if snippet == "" {
		t.Fatal("Expected non-empty snippet")
	}
	if !strings.Contains(strings.ToLower(snippet), "microservices") {
		t.Errorf("Expected snippet to contain 'microservices', got: %s", snippet)
	}
}

func TestGetContextSnippetNoMatch(t *testing.T) {
	dir := buildTmpSearchDir(t, map[string]string{
		"doc.md": "# Title\nSome content here about databases and storage.",
	})
	snippet := knowledge.GetContextSnippet(filepath.Join(dir, "doc.md"), "xyznonexistent", 100)
	if snippet == "" {
		t.Fatal("Expected non-empty snippet even without match")
	}
	if !strings.Contains(snippet, "Title") {
		t.Errorf("Expected snippet to start with file beginning, got: %s", snippet)
	}
}

func TestGetContextSnippetEmptyFile(t *testing.T) {
	dir := buildTmpSearchDir(t, map[string]string{
		"empty.md": "",
	})
	snippet := knowledge.GetContextSnippet(filepath.Join(dir, "empty.md"), "test", 100)
	if snippet != "" {
		t.Errorf("Expected empty snippet for empty file, got: %q", snippet)
	}
}

func TestGetContextSnippetMissingFile(t *testing.T) {
	snippet := knowledge.GetContextSnippet("/nonexistent/path.md", "test", 100)
	if snippet != "" {
		t.Errorf("Expected empty snippet for missing file, got: %q", snippet)
	}
}

func TestGetContextSnippetEmptyQuery(t *testing.T) {
	dir := buildTmpSearchDir(t, map[string]string{
		"doc.md": "# Heading\nContent goes here.",
	})
	snippet := knowledge.GetContextSnippet(filepath.Join(dir, "doc.md"), "", 100)
	if snippet == "" {
		t.Fatal("Expected non-empty snippet for empty query")
	}
}

func TestGetContextSnippetTruncation(t *testing.T) {
	longContent := strings.Repeat("word ", 200)
	dir := buildTmpSearchDir(t, map[string]string{
		"long.md": longContent,
	})
	snippet := knowledge.GetContextSnippet(filepath.Join(dir, "long.md"), "word", 50)
	if len([]rune(snippet)) > 60 { // allow for "..." padding
		t.Errorf("Snippet too long: %d runes", len([]rune(snippet)))
	}
}

func TestGetContextSnippetMatchAtStart(t *testing.T) {
	dir := buildTmpSearchDir(t, map[string]string{
		"doc.md": "API gateway handles requests to microservices for load balancing.",
	})
	snippet := knowledge.GetContextSnippet(filepath.Join(dir, "doc.md"), "API", 100)
	if !strings.Contains(snippet, "API") {
		t.Errorf("Expected snippet to contain 'API', got: %s", snippet)
	}
	if strings.HasPrefix(snippet, "...") {
		t.Error("Snippet should not start with '...' when match is at beginning")
	}
}

func TestGetContextSnippetMatchAtEnd(t *testing.T) {
	content := strings.Repeat("filler ", 50) + "microservices pattern"
	dir := buildTmpSearchDir(t, map[string]string{
		"doc.md": content,
	})
	snippet := knowledge.GetContextSnippet(filepath.Join(dir, "doc.md"), "microservices", 60)
	if !strings.Contains(strings.ToLower(snippet), "microservices") {
		t.Errorf("Expected snippet to contain 'microservices', got: %s", snippet)
	}
}

func TestGetContextSnippetCaseInsensitive(t *testing.T) {
	dir := buildTmpSearchDir(t, map[string]string{
		"doc.md": "The Authentication service handles login and JWT tokens.",
	})
	snippet := knowledge.GetContextSnippet(filepath.Join(dir, "doc.md"), "authentication", 100)
	if !strings.Contains(snippet, "Authentication") {
		t.Errorf("Expected snippet to contain original case 'Authentication', got: %s", snippet)
	}
}

func TestGetContextSnippetWhitespaceCollapsed(t *testing.T) {
	dir := buildTmpSearchDir(t, map[string]string{
		"doc.md": "# Heading\n\nParagraph one.\n\nParagraph two with search term here.",
	})
	snippet := knowledge.GetContextSnippet(filepath.Join(dir, "doc.md"), "search", 100)
	if strings.Contains(snippet, "\n") {
		t.Error("Snippet should not contain newlines")
	}
}

func TestGetContextSnippetSpecialChars(t *testing.T) {
	dir := buildTmpSearchDir(t, map[string]string{
		"doc.md": "Use the GET /api/v1/users endpoint for user listing.",
	})
	snippet := knowledge.GetContextSnippet(filepath.Join(dir, "doc.md"), "/api/v1", 100)
	if !strings.Contains(snippet, "/api/v1") {
		t.Errorf("Expected snippet to contain '/api/v1', got: %s", snippet)
	}
}

// ====== DIR-04: Highlight Tests (unchanged — free functions) ======

func TestHighlightQueryInSnippetBasic(t *testing.T) {
	result := highlightQueryInSnippet("the microservices pattern", "microservices", "\x1b[38;5;250m", "\x1b[1;38;5;226m", "\x1b[0m")
	if !strings.Contains(result, "\x1b[1;38;5;226m") {
		t.Error("Expected highlight escape code in result")
	}
	if !strings.Contains(result, "microservices") {
		t.Error("Expected query text to appear in result")
	}
}

func TestHighlightQueryInSnippetCaseInsensitive(t *testing.T) {
	result := highlightQueryInSnippet("The API gateway", "api", "\x1b[38;5;250m", "\x1b[1m", "\x1b[0m")
	if !strings.Contains(result, "\x1b[1m") {
		t.Error("Expected highlight for case-insensitive match")
	}
}

func TestHighlightQueryInSnippetMultipleMatches(t *testing.T) {
	result := highlightQueryInSnippet("api calls to api endpoints via api", "api", "", "\x1b[1m", "\x1b[0m")
	count := strings.Count(result, "\x1b[1m")
	if count != 3 {
		t.Errorf("Expected 3 highlights, got %d", count)
	}
}

func TestHighlightQueryInSnippetEmptyQuery(t *testing.T) {
	result := highlightQueryInSnippet("some text", "", "", "", "")
	if result != "some text" {
		t.Errorf("Expected unchanged text for empty query, got: %s", result)
	}
}

func TestHighlightQueryInSnippetEmptySnippet(t *testing.T) {
	result := highlightQueryInSnippet("", "query", "", "", "")
	if result != "" {
		t.Errorf("Expected empty result for empty snippet, got: %s", result)
	}
}

func TestStripANSIForLen(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"plain text", "plain text"},
		{"\x1b[1mBold\x1b[0m", "Bold"},
		{"\x1b[38;5;226mYellow\x1b[0m text", "Yellow text"},
		{"no escapes", "no escapes"},
		{"", ""},
	}
	for _, tc := range tests {
		got := stripANSIForLen(tc.input)
		if got != tc.expected {
			t.Errorf("stripANSIForLen(%q) = %q, want %q", tc.input, got, tc.expected)
		}
	}
}

// ─── Viewer integration (D-08): mode entry/exit, back-navigation, header ─────

// TestViewerIntegration_SlashFromFileViewEntersCrossSearch verifies the
// file-view '/' trigger routes through switchModeCmd(modeCrossSearch, "")
// rather than writing crossSearch* fields inline (ARCH-05).
func TestViewerIntegration_SlashFromFileViewEntersCrossSearch(t *testing.T) {
	dir := buildTmpSearchDir(t, map[string]string{"doc.md": "# Doc"})
	v := newTestViewerWithDir(t, dir)

	v = pressKeySettled(v, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})

	m := csModel(v)
	if m == nil {
		t.Fatal("expected activeChild to be a *CrossSearchModel after '/'")
	}
	if m.stage != csStageInput {
		t.Error("expected new CrossSearchModel to start in csStageInput")
	}
}

// TestViewerIntegration_OpenResultAndBackPreservesState verifies the
// open-from-search / back-to-search round trip preserves results and
// selection without re-searching (DIR-04).
func TestViewerIntegration_OpenResultAndBackPreservesState(t *testing.T) {
	dir := buildTmpSearchDir(t, map[string]string{
		"a.md": "# Doc A\nContent A.",
		"b.md": "# Doc B\nContent B.",
	})
	v := newTestViewerWithDir(t, dir)
	v.startDir = dir

	m := newTestCrossSearchModel(dir, v.Width, v.Height)
	m.stage = csStageResults
	m.query = "content"
	m.results = []knowledge.SearchResult{
		{RelPath: "a.md", Score: 5.0, Path: filepath.Join(dir, "a.md")},
		{RelPath: "b.md", Score: 4.0, Path: filepath.Join(dir, "b.md")},
	}
	m.selected = 1
	v.activeChild = m

	// Open the selected (second) result.
	v = pressKeySettled(v, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}})
	if !v.openedFromSearch {
		t.Fatal("expected openedFromSearch=true after opening result")
	}
	if v.activeChild != nil {
		t.Error("expected activeChild=nil while viewing the opened file")
	}
	if v.crossSearchQuery != "content" {
		t.Errorf("expected header breadcrumb query preserved, got %q", v.crossSearchQuery)
	}

	// Return via 'h'.
	v = pressKeySettled(v, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'h'}})
	back := csModel(v)
	if back == nil {
		t.Fatal("expected activeChild restored to *CrossSearchModel after 'h'")
	}
	if back.selected != 1 {
		t.Errorf("expected selected=1 (preserved), got %d", back.selected)
	}
	if len(back.results) != 2 {
		t.Errorf("expected 2 results preserved, got %d", len(back.results))
	}
	if v.openedFromSearch {
		t.Error("expected openedFromSearch=false after returning")
	}
}

// TestViewerIntegration_DirectoryToSearchInputCancel verifies dir -> search
// input -> Esc: the input-stage cancel closes the child entirely (activeChild
// becomes nil) without restoring the paused DirectoryModel. This exactly
// matches pre-refactor behavior — the old updateCrossSearch's "esc"/"ctrl+f"
// branch never restored directoryMode either, only updateCrossSearchNav's
// "h"/"esc"/"q" (results-stage) branch did (see
// TestViewerIntegration_SearchResultsToDirectoryPreservesSelection below).
func TestViewerIntegration_DirectoryToSearchInputCancel(t *testing.T) {
	dir := makeTempDir(t, map[string]string{
		"a.md": "# A", "b.md": "# B",
	})
	defer os.RemoveAll(dir)

	dm, err := NewDirectoryModel(dir, theme.NewTheme(), 100, 24)
	if err != nil {
		t.Fatalf("NewDirectoryModel: %v", err)
	}
	dm.state.SelectedIndex = 1
	v := newTestViewerWithDir(t, dir)
	v.startDir = dir
	v.activeChild = dm

	// '/' from directory -> cross-search input.
	v = pressKeySettled(v, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	if csModel(v) == nil {
		t.Fatal("expected CrossSearchModel active after '/'")
	}

	// Esc from input stage -> closes entirely (modeNone), matching
	// pre-refactor's updateCrossSearch esc/ctrl+f branch.
	v = pressKeySettled(v, tea.KeyMsg{Type: tea.KeyEsc})
	if csModel(v) != nil {
		t.Error("expected CrossSearchModel closed after Esc")
	}
	if dirModel(v) != nil {
		t.Error("expected no DirectoryModel restored on input-stage cancel (pre-refactor parity)")
	}
}

// TestViewerIntegration_SearchResultsToDirectoryPreservesSelection verifies
// h/esc from search *results* (not input) also restores the paused
// DirectoryModel without rescanning.
func TestViewerIntegration_SearchResultsToDirectoryPreservesSelection(t *testing.T) {
	dir := makeTempDir(t, map[string]string{"a.md": "# A", "b.md": "# B"})
	defer os.RemoveAll(dir)

	dm, err := NewDirectoryModel(dir, theme.NewTheme(), 100, 24)
	if err != nil {
		t.Fatalf("NewDirectoryModel: %v", err)
	}
	dm.state.SelectedIndex = 1
	v := newTestViewerWithDir(t, dir)
	v.startDir = dir
	v.activeChild = dm

	v = pressKeySettled(v, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	m := csModel(v)
	if m == nil {
		t.Fatal("expected CrossSearchModel active")
	}
	m.stage = csStageResults
	m.results = []knowledge.SearchResult{{RelPath: "a.md", Score: 5.0}}

	v = pressKeySettled(v, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'h'}})
	restored := dirModel(v)
	if restored == nil {
		t.Fatal("expected DirectoryModel restored after 'h' from results")
	}
	if restored.state.SelectedIndex != 1 {
		t.Errorf("expected selection preserved at 1, got %d", restored.state.SelectedIndex)
	}
}

// TestHeaderShowsSearchBreadcrumb verifies header shows search context breadcrumb.
func TestHeaderShowsSearchBreadcrumb(t *testing.T) {
	dir := t.TempDir()
	v := newTestViewerWithDir(t, dir)
	v.openedFromSearch = true
	v.crossSearchQuery = "microservices"
	v.Width = 80

	header := v.renderHeader()
	if !strings.Contains(header, "search: microservices") {
		t.Errorf("Expected search breadcrumb in header, got: %s", header)
	}
}

// TestStatusBarShowsCrossSearchPrompt verifies the status bar shows the
// cross-search prompt while a CrossSearchModel in input stage is active.
func TestStatusBarShowsCrossSearchPrompt(t *testing.T) {
	dir := t.TempDir()
	v := newTestViewerWithDir(t, dir)
	m := newTestCrossSearchModel(dir, v.Width, v.Height)
	m.input = "my query"
	v.activeChild = m

	bar := v.renderStatusBar()
	if !strings.Contains(bar, "Search all files") {
		t.Errorf("Expected 'Search all files' prompt in status bar, got: %s", bar)
	}
	if !strings.Contains(bar, "my query") {
		t.Errorf("Expected typed query 'my query' in status bar, got: %s", bar)
	}
}

// TestViewRoutesToCrossSearchResultsWhenActive verifies View() renders the
// full-screen results view (header + CrossSearchModel.View()) when the
// active child is in the results stage.
func TestViewRoutesToCrossSearchResultsWhenActive(t *testing.T) {
	dir := t.TempDir()
	v := newTestViewerWithDir(t, dir)
	m := newTestCrossSearchModel(dir, v.Width, v.Height)
	m.stage = csStageResults
	m.query = "uniqueterm"
	m.selected = 0
	m.results = []knowledge.SearchResult{{RelPath: "test.md", Score: 6.0}}
	v.activeChild = m

	out := v.View()
	if !strings.Contains(out, "uniqueterm") {
		t.Error("Expected search query in View() output when results stage is active")
	}
}

// TestBackToSearchResultsNotOpenedFromSearch verifies no-op when not from search.
func TestBackToSearchResultsNotOpenedFromSearch(t *testing.T) {
	dir := t.TempDir()
	v := newTestViewerWithDir(t, dir)
	v.openedFromSearch = false

	backV, _ := v.BackToSearchResults()
	if csModel(backV) != nil {
		t.Error("Expected no CrossSearchModel restored when not opened from search")
	}
}
