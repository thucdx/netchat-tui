package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// AuthConfig holds the credentials persisted to disk.
// Token is the raw MMAUTHTOKEN value; UserID is the Mattermost user ID.
type AuthConfig struct {
	Token  string `json:"token"`
	UserID string `json:"user_id"`
}

// configPath returns the absolute path to the auth config file:
// $XDG_CONFIG_HOME/netchat-tui/auth.json  (falls back to ~/.config on most systems).
func configPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("failed to locate user config directory: %w", err)
	}
	return filepath.Join(dir, "netchat-tui", "auth.json"), nil
}

// Load reads the auth config from disk.
// If the file does not exist, an empty AuthConfig is returned with no error.
func Load() (*AuthConfig, error) {
	path, err := configPath()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return &AuthConfig{}, nil
		}
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg AuthConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	return &cfg, nil
}

// Save writes cfg to disk as pretty-printed JSON.
// The config directory is created with permissions 0700 if it does not exist.
// The file is written with permissions 0600 (owner read/write only).
func Save(cfg *AuthConfig) error {
	path, err := configPath()
	if err != nil {
		return err
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to encode config: %w", err)
	}

	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	return nil
}
