package tui

import "testing"

// TestMessageOpenFileCmd verifies openFileCmd resolves to an openFileMsg
// carrying the expected path and origin.
func TestMessageOpenFileCmd(t *testing.T) {
	tests := []struct {
		name   string
		path   string
		origin origin
	}{
		{"directory origin", "/tmp/docs/readme.md", originDirectory},
		{"search origin", "/tmp/docs/api.md", originSearch},
		{"graph origin", "/tmp/docs/graph.md", originGraph},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := openFileCmd(tt.path, tt.origin)
			if cmd == nil {
				t.Fatal("openFileCmd returned nil tea.Cmd")
			}
			msg := cmd()
			ofm, ok := msg.(openFileMsg)
			if !ok {
				t.Fatalf("expected openFileMsg, got %T", msg)
			}
			if ofm.path != tt.path {
				t.Errorf("expected path %q, got %q", tt.path, ofm.path)
			}
			if ofm.origin != tt.origin {
				t.Errorf("expected origin %v, got %v", tt.origin, ofm.origin)
			}
		})
	}
}

// TestMessageSwitchModeCmd verifies switchModeCmd resolves to a
// switchModeMsg carrying the expected mode and arg.
func TestMessageSwitchModeCmd(t *testing.T) {
	tests := []struct {
		name string
		mode appMode
		arg  string
	}{
		{"directory mode", modeDirectory, "/tmp/docs"},
		{"cross search mode", modeCrossSearch, ""},
		{"graph mode", modeGraph, "/tmp/docs"},
		{"none mode", modeNone, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := switchModeCmd(tt.mode, tt.arg)
			if cmd == nil {
				t.Fatal("switchModeCmd returned nil tea.Cmd")
			}
			msg := cmd()
			smm, ok := msg.(switchModeMsg)
			if !ok {
				t.Fatalf("expected switchModeMsg, got %T", msg)
			}
			if smm.mode != tt.mode {
				t.Errorf("expected mode %v, got %v", tt.mode, smm.mode)
			}
			if smm.arg != tt.arg {
				t.Errorf("expected arg %q, got %q", tt.arg, smm.arg)
			}
		})
	}
}

// TestMessageToggleHelpCmd verifies toggleHelpCmd resolves to a toggleHelpMsg.
func TestMessageToggleHelpCmd(t *testing.T) {
	cmd := toggleHelpCmd()
	if cmd == nil {
		t.Fatal("toggleHelpCmd returned nil tea.Cmd")
	}
	msg := cmd()
	if _, ok := msg.(toggleHelpMsg); !ok {
		t.Fatalf("expected toggleHelpMsg, got %T", msg)
	}
}
