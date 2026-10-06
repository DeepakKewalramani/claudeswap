package config

import (
	"os"
	"testing"
)

func TestConfigStore(t *testing.T) {
	tempDir := t.TempDir()
	paths := NewPathsWithBase(tempDir)
	store := NewConfigStore(paths)

	// Load non-existent
	cfg, err := store.Load()
	if err != nil {
		t.Fatalf("Load on non-existent config failed: %v", err)
	}
	if cfg.Version != CurrentConfigVersion {
		t.Errorf("Expected version %d, got %d", CurrentConfigVersion, cfg.Version)
	}

	// 1. Save valid initial config
	cfg.DefaultProfile = "deepak"
	cfg.ClaudeBinaryPath = "/usr/local/bin/claude"
	if err := store.Save(cfg); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	// 2. Save second config so backup is created with the first config
	cfg.DefaultProfile = "deepak-work"
	if err := store.Save(cfg); err != nil {
		t.Fatalf("Second save failed: %v", err)
	}

	// Verify current loaded
	loaded, err := store.Load()
	if err != nil {
		t.Fatalf("Re-load failed: %v", err)
	}
	if loaded.DefaultProfile != "deepak-work" {
		t.Errorf("Expected deepak-work, got: %s", loaded.DefaultProfile)
	}

	// 3. Deliberately corrupt config.json
	cfgPath := paths.ConfigFile()
	if err := os.WriteFile(cfgPath, []byte("broken json"), 0600); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	// Verify Load fails with ErrCorruptedConfig
	_, err = store.Load()
	if err == nil {
		t.Fatalf("Expected error loading corrupted config, got nil")
	}

	// 4. Restore from backup
	if err := store.RestoreBackup(); err != nil {
		t.Fatalf("RestoreBackup failed: %v", err)
	}

	restored, err := store.Load()
	if err != nil {
		t.Fatalf("Load after restore failed: %v", err)
	}
	if restored.DefaultProfile != "deepak" {
		t.Errorf("Expected restored profile to be 'deepak', got %q", restored.DefaultProfile)
	}
}
