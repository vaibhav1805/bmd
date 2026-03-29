package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestSessionRoundTrip(t *testing.T) {
	// Use a temp dir as XDG_CONFIG_HOME to avoid writing to real config.
	tmp := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmp)

	// Initially no session exists.
	s, err := LoadSession()
	if err != nil {
		t.Fatalf("LoadSession on empty dir: %v", err)
	}
	if s != nil {
		t.Fatalf("expected nil session, got %+v", s)
	}

	// Save a session.
	want := &SessionState{
		LastFilePath: "/tmp/test.md",
		CursorLine:   10,
		CursorCol:    5,
		ScrollOffset: 42,
		EditMode:     false,
	}
	if err := SaveSession(want); err != nil {
		t.Fatalf("SaveSession: %v", err)
	}

	// Load it back.
	got, err := LoadSession()
	if err != nil {
		t.Fatalf("LoadSession: %v", err)
	}
	if got == nil {
		t.Fatal("expected session, got nil")
	}
	if got.LastFilePath != want.LastFilePath {
		t.Errorf("LastFilePath = %q, want %q", got.LastFilePath, want.LastFilePath)
	}
	if got.CursorLine != want.CursorLine {
		t.Errorf("CursorLine = %d, want %d", got.CursorLine, want.CursorLine)
	}
	if got.CursorCol != want.CursorCol {
		t.Errorf("CursorCol = %d, want %d", got.CursorCol, want.CursorCol)
	}
	if got.ScrollOffset != want.ScrollOffset {
		t.Errorf("ScrollOffset = %d, want %d", got.ScrollOffset, want.ScrollOffset)
	}
	if got.Timestamp == 0 {
		t.Error("Timestamp should be set after save")
	}
}

func TestSessionExpiry(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmp)

	// Write a session with a timestamp > 30 days old.
	old := &SessionState{
		LastFilePath: "/tmp/old.md",
		Timestamp:    time.Now().Add(-31 * 24 * time.Hour).Unix(),
	}
	data, _ := json.MarshalIndent(old, "", "  ")
	dir := filepath.Join(tmp, "bmd")
	_ = os.MkdirAll(dir, 0o700)
	_ = os.WriteFile(filepath.Join(dir, "session.json"), data, 0o600)

	got, err := LoadSession()
	if err != nil {
		t.Fatalf("LoadSession: %v", err)
	}
	if got != nil {
		t.Errorf("expected nil for expired session, got %+v", got)
	}
}

func TestSessionCorruptFile(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmp)

	dir := filepath.Join(tmp, "bmd")
	_ = os.MkdirAll(dir, 0o700)
	_ = os.WriteFile(filepath.Join(dir, "session.json"), []byte("not json"), 0o600)

	got, err := LoadSession()
	if err != nil {
		t.Fatalf("LoadSession on corrupt file should not error: %v", err)
	}
	if got != nil {
		t.Errorf("expected nil for corrupt session, got %+v", got)
	}
}

func TestClearSession(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmp)

	// Save then clear.
	_ = SaveSession(&SessionState{LastFilePath: "/tmp/x.md"})
	ClearSession()

	got, _ := LoadSession()
	if got != nil {
		t.Errorf("expected nil after ClearSession, got %+v", got)
	}
}
