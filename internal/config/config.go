// Package config provides configuration management for bmd.
// Configuration is persisted to the user's home directory in .config/bmd/ (Unix/macOS)
// or %APPDATA%\bmd\ (Windows).
package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/bmd/bmd/internal/theme"
)

// sessionMaxAge is the maximum age of a saved session before it is ignored.
const sessionMaxAge = 30 * 24 * time.Hour // 30 days

// SessionState holds the last editor state for session restoration.
type SessionState struct {
	LastFilePath string `json:"last_file_path"`
	CursorLine   int    `json:"cursor_line"`
	CursorCol    int    `json:"cursor_col"`
	ScrollOffset int    `json:"scroll_offset"`
	EditMode     bool   `json:"edit_mode"`
	Timestamp    int64  `json:"timestamp"` // Unix timestamp
}

// Config holds bmd configuration settings.
type Config struct {
	Theme            string        `json:"theme"`              // ThemeName as string
	AutoSaveEnabled  bool          `json:"auto_save_enabled"`  // whether auto-save is active while editing
	AutoSaveInterval time.Duration `json:"auto_save_interval"` // interval between auto-saves (e.g. 30s)
}

// DefaultConfig returns a new Config with sensible defaults.
func DefaultConfig() Config {
	return Config{
		Theme:            string(theme.ThemeDefault),
		AutoSaveEnabled:  true,
		AutoSaveInterval: 30 * time.Second,
	}
}

// GetAutoSaveInterval returns the configured auto-save interval, falling back to 30s.
func (c Config) GetAutoSaveInterval() time.Duration {
	if c.AutoSaveInterval <= 0 {
		return 30 * time.Second
	}
	return c.AutoSaveInterval
}

// configDir returns the bmd config directory, creating it if needed.
func configDir() (string, error) {
	var dir string

	// Try to use XDG config dir on Unix (respects $XDG_CONFIG_HOME)
	if xdgHome := os.Getenv("XDG_CONFIG_HOME"); xdgHome != "" {
		dir = filepath.Join(xdgHome, "bmd")
	} else if home, err := os.UserHomeDir(); err == nil {
		// Standard location: ~/.config/bmd/
		dir = filepath.Join(home, ".config", "bmd")
	} else {
		return "", fmt.Errorf("cannot determine config directory: %w", err)
	}

	// Ensure the directory exists
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", fmt.Errorf("cannot create config directory: %w", err)
	}

	return dir, nil
}

// configPath returns the path to the bmd config file, creating directories as needed.
// On Unix/macOS: ~/.config/bmd/config.json
// On Windows: %APPDATA%\bmd\config.json
func configPath() (string, error) {
	dir, err := configDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "config.json"), nil
}

// Load reads the config file from disk. If the file doesn't exist,
// returns a DefaultConfig without error.
func Load() (Config, error) {
	path, err := configPath()
	if err != nil {
		return DefaultConfig(), err
	}

	// If file doesn't exist, return default (not an error)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return DefaultConfig(), nil
		}
		return DefaultConfig(), fmt.Errorf("cannot read config: %w", err)
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		// If unmarshal fails, log but return default
		return DefaultConfig(), fmt.Errorf("cannot parse config: %w", err)
	}

	// Validate theme name is recognized
	validTheme := false
	for _, t := range theme.AvailableThemes() {
		if string(t) == cfg.Theme {
			validTheme = true
			break
		}
	}
	if !validTheme {
		// Invalid theme in config: use default
		cfg.Theme = string(theme.ThemeDefault)
	}

	// Apply defaults for auto-save fields missing from older config files.
	if cfg.AutoSaveInterval <= 0 {
		cfg.AutoSaveInterval = 30 * time.Second
	}

	return cfg, nil
}

// Save writes the config to disk.
func (c Config) Save() error {
	path, err := configPath()
	if err != nil {
		return err
	}

	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return fmt.Errorf("cannot marshal config: %w", err)
	}

	if err := os.WriteFile(path, data, 0o600); err != nil {
		return fmt.Errorf("cannot write config: %w", err)
	}

	return nil
}

// sessionPath returns the path to the session state file.
func sessionPath() (string, error) {
	dir, err := configDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "session.json"), nil
}

// LoadSession reads session state from disk. Returns nil, nil if no session
// file exists or the session is older than 30 days.
func LoadSession() (*SessionState, error) {
	path, err := sessionPath()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("cannot read session: %w", err)
	}

	var s SessionState
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, nil // corrupt session file — treat as missing
	}

	// Invalidate sessions older than 30 days.
	if time.Since(time.Unix(s.Timestamp, 0)) > sessionMaxAge {
		return nil, nil
	}

	return &s, nil
}

// SaveSession writes session state to disk.
func SaveSession(s *SessionState) error {
	path, err := sessionPath()
	if err != nil {
		return err
	}

	s.Timestamp = time.Now().Unix()

	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return fmt.Errorf("cannot marshal session: %w", err)
	}

	return os.WriteFile(path, data, 0o600)
}

// ClearSession removes the session file.
func ClearSession() {
	path, err := sessionPath()
	if err != nil {
		return
	}
	os.Remove(path)
}
