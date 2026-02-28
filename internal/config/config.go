// Package config provides configuration management for bmd.
// Configuration is persisted to the user's home directory in .config/bmd/ (Unix/macOS)
// or %APPDATA%\bmd\ (Windows).
package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/bmd/bmd/internal/theme"
)

// Config holds bmd configuration settings.
type Config struct {
	Theme string `json:"theme"` // ThemeName as string
}

// DefaultConfig returns a new Config with sensible defaults.
func DefaultConfig() Config {
	return Config{
		Theme: string(theme.ThemeDefault),
	}
}

// configPath returns the path to the bmd config file, creating directories as needed.
// On Unix/macOS: ~/.config/bmd/config.json
// On Windows: %APPDATA%\bmd\config.json
func configPath() (string, error) {
	var configDir string

	// Try to use XDG config dir on Unix (respects $XDG_CONFIG_HOME)
	if xdgHome := os.Getenv("XDG_CONFIG_HOME"); xdgHome != "" {
		configDir = filepath.Join(xdgHome, "bmd")
	} else if home, err := os.UserHomeDir(); err == nil {
		// Standard location: ~/.config/bmd/
		configDir = filepath.Join(home, ".config", "bmd")
	} else {
		return "", fmt.Errorf("cannot determine config directory: %w", err)
	}

	// Ensure the directory exists
	if err := os.MkdirAll(configDir, 0o700); err != nil {
		return "", fmt.Errorf("cannot create config directory: %w", err)
	}

	return filepath.Join(configDir, "config.json"), nil
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
