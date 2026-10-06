package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
)

// CurrentConfigVersion defines the schema version for config.json.
const CurrentConfigVersion = 1

// ErrCorruptedConfig is returned when config.json cannot be parsed as valid JSON.
var ErrCorruptedConfig = errors.New("configuration file is corrupted")

// AppConfig represents global user settings stored in ~/.claudeswap/config.json.
type AppConfig struct {
	Version          int    `json:"version"`
	DefaultProfile   string `json:"defaultProfile,omitempty"`
	ClaudeBinaryPath string `json:"claudeBinaryPath,omitempty"`
	ShellIntegrated  bool   `json:"shellIntegrated,omitempty"`
	SharedContext    bool   `json:"sharedContext,omitempty"`
	AutoHandoff      *bool  `json:"autoHandoff,omitempty"`
}

// IsAutoHandoffEnabled returns true by default unless explicitly disabled by user.
func (c *AppConfig) IsAutoHandoffEnabled() bool {
	if c == nil || c.AutoHandoff == nil {
		return true
	}
	return *c.AutoHandoff
}

// ConfigStore handles safe reading and atomic writing of AppConfig.
type ConfigStore struct {
	paths *Paths
	mu    sync.RWMutex
}

// NewConfigStore creates a new ConfigStore instance.
func NewConfigStore(paths *Paths) *ConfigStore {
	return &ConfigStore{paths: paths}
}

// Load reads and parses config.json. If the file doesn't exist, it returns default config.
// If the file is corrupted, it returns ErrCorruptedConfig.
func (s *ConfigStore) Load() (*AppConfig, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	cfgPath := s.paths.ConfigFile()
	data, err := os.ReadFile(cfgPath)
	if os.IsNotExist(err) {
		return &AppConfig{
			Version: CurrentConfigVersion,
		}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to read config file %q: %w", cfgPath, err)
	}

	var cfg AppConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrCorruptedConfig, err)
	}

	if cfg.Version == 0 {
		cfg.Version = CurrentConfigVersion
	}

	return &cfg, nil
}

// Save atomically writes AppConfig to config.json and updates backups/config.json.bak.
func (s *ConfigStore) Save(cfg *AppConfig) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.paths.EnsureDirectories(); err != nil {
		return err
	}

	if cfg.Version == 0 {
		cfg.Version = CurrentConfigVersion
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to serialize config: %w", err)
	}
	data = append(data, '\n')

	cfgPath := s.paths.ConfigFile()

	// Backup existing file if valid
	if _, err := os.Stat(cfgPath); err == nil {
		var dummy AppConfig
		if existingData, err := os.ReadFile(cfgPath); err == nil && json.Unmarshal(existingData, &dummy) == nil {
			bakPath := s.paths.ConfigBakFile()
			_ = copyFile(cfgPath, bakPath)
		}
	}

	// Atomic write via temporary file
	tmpFile, err := os.CreateTemp(s.paths.HomeDir(), "config-*.json.tmp")
	if err != nil {
		return fmt.Errorf("failed to create temporary config file: %w", err)
	}
	tmpName := tmpFile.Name()
	defer func() {
		_ = tmpFile.Close()
		_ = os.Remove(tmpName)
	}()

	if err := tmpFile.Chmod(0600); err != nil {
		return fmt.Errorf("failed to set permissions on temp config: %w", err)
	}

	if _, err := tmpFile.Write(data); err != nil {
		return fmt.Errorf("failed to write to temp config: %w", err)
	}

	if err := tmpFile.Sync(); err != nil {
		return fmt.Errorf("failed to sync temp config: %w", err)
	}

	if err := tmpFile.Close(); err != nil {
		return fmt.Errorf("failed to close temp config: %w", err)
	}

	if err := os.Rename(tmpName, cfgPath); err != nil {
		return fmt.Errorf("failed to atomically replace config file: %w", err)
	}

	return nil
}

// RestoreBackup restores config.json from the backup file if available.
func (s *ConfigStore) RestoreBackup() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	bakPath := s.paths.ConfigBakFile()
	cfgPath := s.paths.ConfigFile()

	if _, err := os.Stat(bakPath); err != nil {
		return fmt.Errorf("no config backup found at %q", bakPath)
	}

	return copyFile(bakPath, cfgPath)
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	if err := os.MkdirAll(filepath.Dir(dst), 0700); err != nil {
		return err
	}

	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600)
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return out.Sync()
}
