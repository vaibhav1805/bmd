// Package tui: shared tea.Msg vocabulary for mode-transition and file-open
// handoff between the parent Viewer and its independent child models
// (DirectoryModel, and — in later phases — CrossSearchModel/GraphModel).
//
// Designed once, upfront (D-02): these types are frozen here and reused
// unchanged by every child model. No child model may invent its own
// transition/file-open message dialect.
package tui

import (
	tea "github.com/charmbracelet/bubbletea"
)

// origin identifies which mode a file-open request originated from, so the
// parent Viewer can enable the correct "back" navigation (ARCH-03).
type origin int

const (
	originDirectory origin = iota
	originSearch
	originGraph
)

// openFileMsg asks the parent Viewer to load a file. A child model never
// calls Viewer.loadFile() directly; it returns openFileCmd's tea.Cmd instead.
type openFileMsg struct {
	path   string
	origin origin
}

// openFileCmd returns a tea.Cmd that resolves to an openFileMsg for the given
// path and origin.
func openFileCmd(path string, o origin) tea.Cmd {
	return func() tea.Msg { return openFileMsg{path: path, origin: o} }
}

// appMode identifies which child mode should become active. modeNone means
// plain file-view/edit-mode, handled by Viewer itself (no child active).
type appMode int

const (
	modeNone appMode = iota
	modeDirectory
	modeCrossSearch
	modeGraph
)

// switchModeMsg asks the parent Viewer to activate a different mode/child
// model. arg carries mode-specific context (e.g. rootPath for modeGraph and
// modeDirectory; empty for modeCrossSearch).
type switchModeMsg struct {
	mode appMode
	arg  string
}

// switchModeCmd returns a tea.Cmd that resolves to a switchModeMsg for the
// given mode and arg.
func switchModeCmd(mode appMode, arg string) tea.Cmd {
	return func() tea.Msg { return switchModeMsg{mode: mode, arg: arg} }
}

// toggleHelpMsg asks the parent Viewer to toggle the help overlay. helpOpen
// stays a plain bool on Viewer (D-04) but every mode toggles it via this
// message instead of writing v.helpOpen directly.
type toggleHelpMsg struct{}

// toggleHelpCmd returns a tea.Cmd that resolves to a toggleHelpMsg.
func toggleHelpCmd() tea.Cmd { return func() tea.Msg { return toggleHelpMsg{} } }

// statusMsg asks the parent Viewer to display a transient status/error
// message in its header/status bar (and schedule it to clear after
// statusTimeout), without giving a child model a back-pointer into Viewer
// (RESEARCH.md Open Question 2). This keeps self-contained, non-mode-
// transitioning side effects — e.g. a "terminal too narrow" warning — on the
// same message-passing footing as file-open/mode-switch requests, without
// requiring the child to reach into Viewer's errorMsg field directly.
type statusMsg struct {
	text string
}

// statusCmd returns a tea.Cmd that resolves to a statusMsg with the given text.
func statusCmd(text string) tea.Cmd {
	return func() tea.Msg { return statusMsg{text: text} }
}

// fileChangedMsg is delivered once the debounce timer fires for a coalesced
// burst of fsnotify events on the currently-watched file (RELOAD-01, D-07).
type fileChangedMsg struct {
	path string
}
