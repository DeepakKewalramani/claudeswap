package profile

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"

	"github.com/claudeswap/claudeswap/internal/config"
)

// ErrCorruptedProfiles is returned when profiles.json exists but contains invalid JSON.
var ErrCorruptedProfiles = errors.New("ClaudeSwap profile database is corrupted")

// Store manages the thread-safe, atomic persistence of profiles.
type Store struct {
	paths *config.Paths
	mu    sync.RWMutex
}

// NewStore creates a new Store instance.
func NewStore(paths *config.Paths) *Store {
	return &Store{paths: paths}
}

// Load reads and parses ~/.claudeswap/profiles.json.
// If the file does not exist, an empty store data structure with the current version is returned.
// If the file exists but cannot be parsed, ErrCorruptedProfiles is returned.
func (s *Store) Load() (*StoreData, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	filePath := s.paths.ProfilesFile()
	data, err := os.ReadFile(filePath)
	if os.IsNotExist(err) {
		return &StoreData{
			Version:  CurrentProfileSchemaVersion,
			Profiles: []Profile{},
		}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to read profile store at %q: %w", filePath, err)
	}

	var storeData StoreData
	if err := json.Unmarshal(data, &storeData); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrCorruptedProfiles, err)
	}

	if storeData.Version == 0 {
		storeData.Version = CurrentProfileSchemaVersion
	}
	if storeData.Profiles == nil {
		storeData.Profiles = []Profile{}
	}

	return &storeData, nil
}

// Save atomically writes the profiles to profiles.json and updates backups/profiles.json.bak.
func (s *Store) Save(storeData *StoreData) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.paths.EnsureDirectories(); err != nil {
		return err
	}

	if storeData.Version == 0 {
		storeData.Version = CurrentProfileSchemaVersion
	}
	if storeData.Profiles == nil {
		storeData.Profiles = []Profile{}
	}

	data, err := json.MarshalIndent(storeData, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to encode profile store: %w", err)
	}
	data = append(data, '\n')

	filePath := s.paths.ProfilesFile()

	// Backup existing file if valid
	if _, err := os.Stat(filePath); err == nil {
		var dummy StoreData
		if existingData, err := os.ReadFile(filePath); err == nil && json.Unmarshal(existingData, &dummy) == nil {
			bakPath := s.paths.ProfilesBakFile()
			_ = copyFile(filePath, bakPath)
		}
	}

	// Atomic write using a temporary file in the same directory
	tmpFile, err := os.CreateTemp(s.paths.HomeDir(), "profiles-*.json.tmp")
	if err != nil {
		return fmt.Errorf("failed to create temporary profile file: %w", err)
	}
	tmpName := tmpFile.Name()
	defer func() {
		_ = tmpFile.Close()
		_ = os.Remove(tmpName)
	}()

	if err := tmpFile.Chmod(0600); err != nil {
		return fmt.Errorf("failed to set permissions on temp profile store: %w", err)
	}

	if _, err := tmpFile.Write(data); err != nil {
		return fmt.Errorf("failed to write to temp profile store: %w", err)
	}

	if err := tmpFile.Sync(); err != nil {
		return fmt.Errorf("failed to sync temp profile store: %w", err)
	}

	if err := tmpFile.Close(); err != nil {
		return fmt.Errorf("failed to close temp profile store: %w", err)
	}

	if err := os.Rename(tmpName, filePath); err != nil {
		return fmt.Errorf("failed to atomically replace profile store: %w", err)
	}

	return nil
}

// HasBackup checks if a valid backup file exists.
func (s *Store) HasBackup() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	bakPath := s.paths.ProfilesBakFile()
	info, err := os.Stat(bakPath)
	return err == nil && info.Size() > 0
}

// RestoreBackup restores profiles.json from backups/profiles.json.bak.
func (s *Store) RestoreBackup() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	bakPath := s.paths.ProfilesBakFile()
	filePath := s.paths.ProfilesFile()

	if _, err := os.Stat(bakPath); err != nil {
		return fmt.Errorf("no backup found at %q", bakPath)
	}

	// Verify that backup itself is valid JSON before restoring
	data, err := os.ReadFile(bakPath)
	if err != nil {
		return fmt.Errorf("failed to read backup: %w", err)
	}

	var dummy StoreData
	if err := json.Unmarshal(data, &dummy); err != nil {
		return fmt.Errorf("backup file at %q is also corrupted: %w", bakPath, err)
	}

	return copyFile(bakPath, filePath)
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
