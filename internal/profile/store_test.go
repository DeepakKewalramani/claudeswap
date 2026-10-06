package profile

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/claudeswap/claudeswap/internal/config"
)

func TestStore_SaveAndLoad(t *testing.T) {
	tempDir := t.TempDir()
	paths := config.NewPathsWithBase(tempDir)
	store := NewStore(paths)

	// Initially empty
	data, err := store.Load()
	if err != nil {
		t.Fatalf("Load on non-existent file failed: %v", err)
	}
	if len(data.Profiles) != 0 {
		t.Errorf("Expected 0 profiles, got %d", len(data.Profiles))
	}
	if data.Version != CurrentProfileSchemaVersion {
		t.Errorf("Expected version %d, got %d", CurrentProfileSchemaVersion, data.Version)
	}

	// Add profile and save
	p := Profile{
		ID:        "deepak",
		Name:      "Deepak",
		Shortcut:  "claudeswap-deepak",
		ConfigDir: paths.ProfileConfigDir("deepak"),
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}
	data.Profiles = append(data.Profiles, p)

	if err := store.Save(data); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	// Re-load and verify
	loaded, err := store.Load()
	if err != nil {
		t.Fatalf("Re-load failed: %v", err)
	}
	if len(loaded.Profiles) != 1 {
		t.Fatalf("Expected 1 profile, got %d", len(loaded.Profiles))
	}
	if loaded.Profiles[0].ID != "deepak" || loaded.Profiles[0].Name != "Deepak" {
		t.Errorf("Profile data mismatch: %+v", loaded.Profiles[0])
	}
}

func TestStore_BackupAndCorruptionRecovery(t *testing.T) {
	tempDir := t.TempDir()
	paths := config.NewPathsWithBase(tempDir)
	store := NewStore(paths)

	data := &StoreData{
		Version: CurrentProfileSchemaVersion,
		Profiles: []Profile{
			{
				ID:       "personal",
				Name:     "Personal",
				Shortcut: "claudeswap-personal",
			},
		},
	}

	// First save
	if err := store.Save(data); err != nil {
		t.Fatalf("First save failed: %v", err)
	}

	// Modify and save again to generate a backup of the first save
	data.Profiles[0].Name = "Personal Updated"
	if err := store.Save(data); err != nil {
		t.Fatalf("Second save failed: %v", err)
	}

	if !store.HasBackup() {
		t.Fatalf("Expected backup file to exist")
	}

	// Deliberately corrupt profiles.json
	profilesFile := paths.ProfilesFile()
	if err := os.WriteFile(profilesFile, []byte("{corrupted json here..."), 0600); err != nil {
		t.Fatalf("Failed to write corrupted file: %v", err)
	}

	// Load should fail with ErrCorruptedProfiles
	_, err := store.Load()
	if !errors.Is(err, ErrCorruptedProfiles) {
		t.Fatalf("Expected ErrCorruptedProfiles, got: %v", err)
	}

	// Restore from backup
	if err := store.RestoreBackup(); err != nil {
		t.Fatalf("RestoreBackup failed: %v", err)
	}

	// Load should succeed now
	recovered, err := store.Load()
	if err != nil {
		t.Fatalf("Load after restore failed: %v", err)
	}
	if len(recovered.Profiles) != 1 {
		t.Fatalf("Expected 1 recovered profile, got %d", len(recovered.Profiles))
	}
}

func TestStore_FilePermissions(t *testing.T) {
	tempDir := t.TempDir()
	paths := config.NewPathsWithBase(tempDir)
	store := NewStore(paths)

	data := &StoreData{
		Version:  CurrentProfileSchemaVersion,
		Profiles: []Profile{},
	}

	if err := store.Save(data); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	info, err := os.Stat(paths.ProfilesFile())
	if err != nil {
		t.Fatalf("Stat failed: %v", err)
	}

	// Check user-only permissions (0600) on Unix
	if perm := info.Mode().Perm(); perm != 0600 {
		t.Errorf("Expected 0600 permissions, got %04o", perm)
	}

	// Check profiles directory permissions (0700)
	dirInfo, err := os.Stat(filepath.Dir(paths.ProfilesFile()))
	if err != nil {
		t.Fatalf("Stat directory failed: %v", err)
	}
	if perm := dirInfo.Mode().Perm(); perm != 0700 {
		t.Errorf("Expected 0700 permissions on home dir, got %04o", perm)
	}
}
