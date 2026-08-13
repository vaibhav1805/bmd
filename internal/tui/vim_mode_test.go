package tui

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/bmd/bmd/internal/config"
	"github.com/bmd/bmd/internal/editor"
	"github.com/bmd/bmd/internal/theme"
	tea "github.com/charmbracelet/bubbletea"
)

// newVimTestViewer returns a Viewer in edit mode with vim keybindings
// enabled and the given lines loaded into the buffer.
func newVimTestViewer(lines []string) *Viewer {
	doc := createTestDocument(nil)
	v := New(doc, "test.md", theme.NewTheme(), 80)
	v.editMode = true
	v.editBuffer = editor.NewTextBuffer(lines)
	v.vimEnabled = true
	v.vimMode = vimModeNormal
	return v
}

func vimKey(r rune) tea.KeyMsg {
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}}
}

func vimPress(v *Viewer, keys string) *Viewer {
	for _, r := range keys {
		model, _ := v.updateEdit(vimKey(r))
		v = model.(*Viewer)
	}
	return v
}

func TestVim_DefaultDisabled_UnchangedBehavior(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir()) // isolate from the real ~/.config/bmd -- New() loads config.VimKeybindings, which must default false regardless of the developer's own real setting

	doc := createTestDocument(nil)
	v := New(doc, "test.md", theme.NewTheme(), 80)
	v.editMode = true
	v.editBuffer = editor.NewTextBuffer([]string{"hello"})
	// vimEnabled left at its zero value (false).

	model, _ := v.updateEdit(vimKey('X'))
	v = model.(*Viewer)

	lines := v.editBuffer.GetLines()
	if lines[0] != "Xhello" {
		t.Errorf("Expected plain character insertion when vim disabled, got %q", lines[0])
	}
}

func TestVim_NormalMode_LettersDoNotInsertText(t *testing.T) {
	v := newVimTestViewer([]string{"hello"})
	v = vimPress(v, "x") // vim 'x' deletes forward, does not insert 'x'

	lines := v.editBuffer.GetLines()
	if lines[0] != "ello" {
		t.Errorf("Expected 'x' to delete forward char in normal mode, got %q", lines[0])
	}
}

func TestVim_Motion_hjkl(t *testing.T) {
	v := newVimTestViewer([]string{"abc", "def", "ghi"})
	v = vimPress(v, "ll") // move right twice
	if v.editBuffer.CursorCol() != 2 {
		t.Errorf("Expected col 2 after 'll', got %d", v.editBuffer.CursorCol())
	}
	v = vimPress(v, "j") // move down
	if v.editBuffer.CursorLine() != 1 {
		t.Errorf("Expected line 1 after 'j', got %d", v.editBuffer.CursorLine())
	}
	v = vimPress(v, "h") // move left
	if v.editBuffer.CursorCol() != 1 {
		t.Errorf("Expected col 1 after 'h', got %d", v.editBuffer.CursorCol())
	}
	v = vimPress(v, "k") // move up
	if v.editBuffer.CursorLine() != 0 {
		t.Errorf("Expected line 0 after 'k', got %d", v.editBuffer.CursorLine())
	}
}

func TestVim_Motion_WordForwardBackward(t *testing.T) {
	v := newVimTestViewer([]string{"foo bar baz"})
	v = vimPress(v, "w")
	if v.editBuffer.CursorCol() != 4 {
		t.Errorf("Expected col 4 (start of 'bar') after 'w', got %d", v.editBuffer.CursorCol())
	}
	v = vimPress(v, "w")
	if v.editBuffer.CursorCol() != 8 {
		t.Errorf("Expected col 8 (start of 'baz') after second 'w', got %d", v.editBuffer.CursorCol())
	}
	v = vimPress(v, "b")
	if v.editBuffer.CursorCol() != 4 {
		t.Errorf("Expected col 4 after 'b', got %d", v.editBuffer.CursorCol())
	}
}

func TestVim_Motion_WordEnd(t *testing.T) {
	v := newVimTestViewer([]string{"foo bar baz"})
	v = vimPress(v, "e")
	if v.editBuffer.CursorCol() != 2 {
		t.Errorf("Expected col 2 (end of 'foo') after 'e', got %d", v.editBuffer.CursorCol())
	}
	v = vimPress(v, "e")
	if v.editBuffer.CursorCol() != 6 {
		t.Errorf("Expected col 6 (end of 'bar') after second 'e', got %d", v.editBuffer.CursorCol())
	}
}

func TestVim_Motion_0AndEndOfLine(t *testing.T) {
	v := newVimTestViewer([]string{"hello world"})
	v = vimPress(v, "$")
	if v.editBuffer.CursorCol() != 11 {
		t.Errorf("Expected col 11 after '$', got %d", v.editBuffer.CursorCol())
	}
	v = vimPress(v, "0")
	if v.editBuffer.CursorCol() != 0 {
		t.Errorf("Expected col 0 after '0', got %d", v.editBuffer.CursorCol())
	}
}

func TestVim_Motion_gg_G(t *testing.T) {
	v := newVimTestViewer([]string{"a", "b", "c", "d"})
	v = vimPress(v, "G")
	if v.editBuffer.CursorLine() != 3 {
		t.Errorf("Expected line 3 after 'G', got %d", v.editBuffer.CursorLine())
	}
	v = vimPress(v, "gg")
	if v.editBuffer.CursorLine() != 0 {
		t.Errorf("Expected line 0 after 'gg', got %d", v.editBuffer.CursorLine())
	}
}

func TestVim_Motion_CountPrefix(t *testing.T) {
	v := newVimTestViewer([]string{"a", "b", "c", "d", "e"})
	v = vimPress(v, "3j")
	if v.editBuffer.CursorLine() != 3 {
		t.Errorf("Expected line 3 after '3j', got %d", v.editBuffer.CursorLine())
	}
}

func TestVim_InsertMode_iEntersInsertAndTypes(t *testing.T) {
	v := newVimTestViewer([]string{"ello"})
	v = vimPress(v, "i")
	if v.vimMode != vimModeInsert {
		t.Fatalf("Expected insert mode after 'i', got mode %d", v.vimMode)
	}
	model, _ := v.updateEdit(vimKey('h'))
	v = model.(*Viewer)
	lines := v.editBuffer.GetLines()
	if lines[0] != "hello" {
		t.Errorf("Expected 'hello' after typing in insert mode, got %q", lines[0])
	}
}

func TestVim_InsertMode_EscReturnsToNormalWithoutExitingEditMode(t *testing.T) {
	v := newVimTestViewer([]string{"hello"})
	v = vimPress(v, "i")
	model, _ := v.updateEdit(tea.KeyMsg{Type: tea.KeyEsc})
	v = model.(*Viewer)
	if v.vimMode != vimModeNormal {
		t.Errorf("Expected normal mode after Esc from insert, got mode %d", v.vimMode)
	}
	if !v.editMode {
		t.Errorf("Expected edit mode to remain active after Esc from insert mode")
	}
}

func TestVim_NormalMode_EscExitsEditMode(t *testing.T) {
	v := newVimTestViewer([]string{"hello"})
	model, _ := v.updateEdit(tea.KeyMsg{Type: tea.KeyEsc})
	v = model.(*Viewer)
	if v.editMode {
		t.Errorf("Expected Esc in normal mode to exit edit mode")
	}
}

func TestVim_AppendA_AtEndOfLineDoesNotWrapToNextLine(t *testing.T) {
	v := newVimTestViewer([]string{"hi", "there"})
	v = vimPress(v, "$") // cursor at col 2 (past 'i'), end of "hi"
	v = vimPress(v, "a")
	if v.vimMode != vimModeInsert {
		t.Fatalf("Expected insert mode after 'a'")
	}
	if v.editBuffer.CursorLine() != 0 || v.editBuffer.CursorCol() != 2 {
		t.Errorf("Expected cursor to stay at (0,2), got (%d,%d)", v.editBuffer.CursorLine(), v.editBuffer.CursorCol())
	}
	model, _ := v.updateEdit(vimKey('!'))
	v = model.(*Viewer)
	lines := v.editBuffer.GetLines()
	if lines[0] != "hi!" {
		t.Errorf("Expected 'hi!' on line 0, got %q (lines=%v)", lines[0], lines)
	}
}

func TestVim_OpenLineBelowAndAbove(t *testing.T) {
	v := newVimTestViewer([]string{"middle"})
	v = vimPress(v, "o")
	if v.vimMode != vimModeInsert {
		t.Fatalf("Expected insert mode after 'o'")
	}
	lines := v.editBuffer.GetLines()
	if len(lines) != 2 || lines[0] != "middle" || lines[1] != "" {
		t.Fatalf("Expected ['middle', ''] after 'o', got %v", lines)
	}

	model, _ := v.updateEdit(tea.KeyMsg{Type: tea.KeyEsc})
	v = model.(*Viewer)
	v = vimPress(v, "gg") // back to line 0
	v = vimPress(v, "O")
	if v.vimMode != vimModeInsert {
		t.Fatalf("Expected insert mode after 'O'")
	}
	lines = v.editBuffer.GetLines()
	if len(lines) != 3 || lines[0] != "" || lines[1] != "middle" {
		t.Fatalf("Expected ['', 'middle', ''] after 'O', got %v", lines)
	}
}

func TestVim_DD_DeletesLineAndYy_Pastes(t *testing.T) {
	v := newVimTestViewer([]string{"alpha", "beta", "gamma"})
	v = vimPress(v, "j")  // cursor on "beta"
	v = vimPress(v, "dd") // delete "beta"

	lines := v.editBuffer.GetLines()
	if len(lines) != 2 || lines[0] != "alpha" || lines[1] != "gamma" {
		t.Fatalf("Expected ['alpha','gamma'] after dd, got %v", lines)
	}

	// After dd, the cursor sits on the line that took "beta"'s place --
	// "gamma" (matches real vim) -- so p pastes "beta" below THAT line.
	v = vimPress(v, "p")
	lines = v.editBuffer.GetLines()
	if len(lines) != 3 || lines[0] != "alpha" || lines[1] != "gamma" || lines[2] != "beta" {
		t.Fatalf("Expected ['alpha','gamma','beta'] after p, got %v", lines)
	}
}

func TestVim_CountedDD(t *testing.T) {
	v := newVimTestViewer([]string{"a", "b", "c", "d"})
	v = vimPress(v, "2dd")
	lines := v.editBuffer.GetLines()
	if len(lines) != 2 || lines[0] != "c" || lines[1] != "d" {
		t.Fatalf("Expected ['c','d'] after 2dd, got %v", lines)
	}
}

func TestVim_Yy_DoesNotDeleteThenPastesAsCopy(t *testing.T) {
	v := newVimTestViewer([]string{"alpha", "beta"})
	v = vimPress(v, "yy")
	lines := v.editBuffer.GetLines()
	if len(lines) != 2 || lines[0] != "alpha" || lines[1] != "beta" {
		t.Fatalf("Expected buffer unchanged after yy, got %v", lines)
	}
	v = vimPress(v, "p")
	lines = v.editBuffer.GetLines()
	if len(lines) != 3 || lines[0] != "alpha" || lines[1] != "alpha" || lines[2] != "beta" {
		t.Fatalf("Expected ['alpha','alpha','beta'] after yy then p, got %v", lines)
	}
}

func TestVim_Cc_ClearsLineAndEntersInsert(t *testing.T) {
	v := newVimTestViewer([]string{"alpha", "beta"})
	v = vimPress(v, "cc")
	if v.vimMode != vimModeInsert {
		t.Fatalf("Expected insert mode after 'cc'")
	}
	lines := v.editBuffer.GetLines()
	if len(lines) != 2 || lines[0] != "" || lines[1] != "beta" {
		t.Fatalf("Expected ['','beta'] after cc, got %v", lines)
	}
}

func TestVim_OperatorMotion_dw(t *testing.T) {
	v := newVimTestViewer([]string{"foo bar baz"})
	v = vimPress(v, "dw")
	lines := v.editBuffer.GetLines()
	if lines[0] != "bar baz" {
		t.Errorf("Expected 'bar baz' after dw, got %q", lines[0])
	}
}

func TestVim_OperatorMotion_de(t *testing.T) {
	v := newVimTestViewer([]string{"foo bar baz"})
	v = vimPress(v, "de")
	lines := v.editBuffer.GetLines()
	if lines[0] != " bar baz" {
		t.Errorf("Expected ' bar baz' after de, got %q", lines[0])
	}
}

func TestVim_OperatorMotion_dEndOfLine(t *testing.T) {
	v := newVimTestViewer([]string{"foo bar baz"})
	v = vimPress(v, "ll") // col 2
	v = vimPress(v, "d$")
	lines := v.editBuffer.GetLines()
	if lines[0] != "fo" {
		t.Errorf("Expected 'fo' after d$, got %q", lines[0])
	}
}

func TestVim_OperatorMotion_dG(t *testing.T) {
	v := newVimTestViewer([]string{"a", "b", "c", "d"})
	v = vimPress(v, "j") // line 1 ("b")
	v = vimPress(v, "dG")
	lines := v.editBuffer.GetLines()
	if len(lines) != 1 || lines[0] != "a" {
		t.Fatalf("Expected ['a'] after dG from line 1, got %v", lines)
	}
}

func TestVim_VisualMode_CharwiseDelete(t *testing.T) {
	v := newVimTestViewer([]string{"hello world"})
	v = vimPress(v, "v")
	if v.vimMode != vimModeVisual {
		t.Fatalf("Expected visual mode after 'v'")
	}
	v = vimPress(v, "llll") // extend selection right 4 chars from col 0 to col 4
	v = vimPress(v, "d")
	if v.vimMode != vimModeNormal {
		t.Errorf("Expected normal mode after visual 'd'")
	}
	lines := v.editBuffer.GetLines()
	// bmd's selection primitives (shared with Shift+arrow text selection)
	// use an exclusive end, one character short of vim's inclusive visual
	// selection convention -- "hell" (cols 0-3) is removed, not "hello".
	if lines[0] != "o world" {
		t.Errorf("Expected 'o world' after v llll d, got %q", lines[0])
	}
}

func TestVim_VisualLineMode_DeletesWholeLines(t *testing.T) {
	v := newVimTestViewer([]string{"alpha", "beta", "gamma", "delta"})
	v = vimPress(v, "j") // line 1
	v = vimPress(v, "V") // enter visual line mode
	v = vimPress(v, "j") // extend to line 2
	v = vimPress(v, "d") // delete lines 1-2 (beta, gamma)

	lines := v.editBuffer.GetLines()
	if len(lines) != 2 || lines[0] != "alpha" || lines[1] != "delta" {
		t.Fatalf("Expected ['alpha','delta'] after V j d, got %v", lines)
	}
	if v.vimMode != vimModeNormal {
		t.Errorf("Expected normal mode after visual-line delete")
	}
}

func TestVim_Undo(t *testing.T) {
	v := newVimTestViewer([]string{"alpha", "beta", "gamma"})
	v = vimPress(v, "dd")
	lines := v.editBuffer.GetLines()
	if len(lines) != 2 {
		t.Fatalf("Expected 2 lines after dd, got %v", lines)
	}
	v = vimPress(v, "u")
	lines = v.editBuffer.GetLines()
	if len(lines) != 3 || lines[0] != "alpha" || lines[1] != "beta" || lines[2] != "gamma" {
		t.Fatalf("Expected original 3 lines after undo, got %v", lines)
	}
}

func TestVim_CtrlSStillSavesInNormalMode(t *testing.T) {
	path := filepath.Join(t.TempDir(), "test.md")
	v := newVimTestViewer([]string{"hello"})
	v.FilePath = path // real path so SaveToFile has somewhere valid to write, rather than leaking a "test.md" into the package dir

	model, _ := v.updateEdit(tea.KeyMsg{Type: tea.KeyCtrlS})
	v = model.(*Viewer)

	// Ctrl+S is a meta shortcut that must remain reachable in vim normal
	// mode, routed through to the same save handler used when vim is off.
	if v.errorMsg != "Saved" {
		t.Errorf("Expected errorMsg 'Saved' after Ctrl+S in vim normal mode, got %q", v.errorMsg)
	}
	if _, err := os.Stat(path); err != nil {
		t.Errorf("Expected file to be written to %s: %v", path, err)
	}
}

// TestVim_MetaShortcutsSurviveNormalMode guards against a real regression:
// an earlier version of the vim-mode/legacy-editor guard in edit_mode.go
// allow-listed meta shortcuts by exact KeyType, and missed every one that
// bmd matches via msg.String() instead of a dedicated case (Ctrl+O,
// Alt+Up/Down, Tab/Shift+Tab, PgUp/PgDn) -- they were silently swallowed as
// vim no-ops whenever vim mode was on. vimShouldHandle() now decides by
// what the vim engine DOES own (plain arrows, plain runes, Enter/Backspace/
// Delete) rather than enumerating what it doesn't; this exercises each
// previously-broken shortcut plus a couple of the ones that already worked,
// all from vim normal mode.
func TestVim_MetaShortcutsSurviveNormalMode(t *testing.T) {
	t.Run("Ctrl+O opens outline", func(t *testing.T) {
		v := newVimTestViewer([]string{"# Intro", "paragraph", "## Details"})
		v.Height, v.Width = 24, 80
		model, _ := v.updateEdit(tea.KeyMsg{Type: tea.KeyCtrlO})
		v = model.(*Viewer)
		if !v.outlineMode {
			t.Error("expected outlineMode=true after Ctrl+O in vim normal mode")
		}
	})

	// NOTE: Alt+Up/Alt+Down (move line) turn out to be pre-existing dead
	// keybindings, unrelated to vim mode and not fixed here (see bmd-*
	// filed for it) -- edit_mode.go's plain `case tea.KeyDown` never
	// checks msg.Alt, so it always wins over the later "alt+down" string
	// case, with or without vim. These subtests just confirm vim mode
	// doesn't change that pre-existing (if broken) behavior -- i.e. vim
	// mode isn't making anything worse here, even though it isn't making
	// it better either.
	t.Run("Alt+Down leaves buffer as plain cursor-down, same as vim off", func(t *testing.T) {
		v := newVimTestViewer([]string{"alpha", "beta"})
		model, _ := v.updateEdit(tea.KeyMsg{Type: tea.KeyDown, Alt: true})
		v = model.(*Viewer)
		lines := v.editBuffer.GetLines()
		if lines[0] != "alpha" || lines[1] != "beta" || v.editBuffer.CursorLine() != 1 {
			t.Errorf("expected unchanged lines with cursor moved to line 1, got %v (cursor=%d)", lines, v.editBuffer.CursorLine())
		}
	})

	t.Run("Tab indents the current line", func(t *testing.T) {
		v := newVimTestViewer([]string{"alpha"})
		model, _ := v.updateEdit(tea.KeyMsg{Type: tea.KeyTab})
		v = model.(*Viewer)
		lines := v.editBuffer.GetLines()
		if lines[0] == "alpha" {
			t.Errorf("expected Tab to indent the line, got unchanged %q", lines[0])
		}
	})

	t.Run("Ctrl+D duplicates the current line", func(t *testing.T) {
		v := newVimTestViewer([]string{"alpha"})
		model, _ := v.updateEdit(tea.KeyMsg{Type: tea.KeyCtrlD})
		v = model.(*Viewer)
		lines := v.editBuffer.GetLines()
		if len(lines) != 2 || lines[0] != "alpha" || lines[1] != "alpha" {
			t.Errorf("expected Ctrl+D to duplicate the line, got %v", lines)
		}
	})

	t.Run("PgDn does not panic and stays in normal mode", func(t *testing.T) {
		v := newVimTestViewer([]string{"alpha", "beta", "gamma"})
		v.Height = 24
		model, _ := v.updateEdit(tea.KeyMsg{Type: tea.KeyPgDown})
		v = model.(*Viewer)
		if v.vimMode != vimModeNormal {
			t.Errorf("expected PgDn to leave vim mode unchanged, got mode %d", v.vimMode)
		}
	})
}

func TestVim_ToggleTogglesModeAndPersists(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir()) // avoid writing to the real ~/.config/bmd

	doc := createTestDocument(nil)
	v := New(doc, "test.md", theme.NewTheme(), 80)
	before := v.vimEnabled

	model, _ := v.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'v'}})
	v = model.(*Viewer)

	if v.vimEnabled == before {
		t.Fatalf("Expected 'v' in read mode to toggle vimEnabled (was %v)", before)
	}

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("config.Load failed: %v", err)
	}
	if cfg.VimKeybindings != v.vimEnabled {
		t.Errorf("Expected persisted VimKeybindings=%v to match v.vimEnabled=%v", cfg.VimKeybindings, v.vimEnabled)
	}

	model, _ = v.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'v'}})
	v = model.(*Viewer)
	if v.vimEnabled != before {
		t.Errorf("Expected second 'v' press to toggle back to %v, got %v", before, v.vimEnabled)
	}
}
