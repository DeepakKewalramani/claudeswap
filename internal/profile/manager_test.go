package profile

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/claudeswap/claudeswap/internal/config"
)

func TestManager_CRUD(t *testing.T) {
	tempDir := t.TempDir()
	paths := config.NewPathsWithBase(tempDir)
	store := NewStore(paths)
	mgr := NewManager(paths, store)

	// 1. Create Profile
	p, err := mgr.CreateProfile("Deepak", "")
	if err != nil {
		t.Fatalf("CreateProfile failed: %v", err)
	}
	if p.ID != "deepak" || p.Shortcut != "claudeswap-deepak" {
		t.Errorf("Unexpected profile values: %+v", p)
	}

	// Verify isolated directory was created
	if info, err := os.Stat(p.ConfigDir); err != nil || !info.IsDir() {
		t.Fatalf("Config directory was not created: %v", err)
	}

	// 2. Reject duplicate creation
	_, err = mgr.CreateProfile("Deepak", "")
	if !errors.Is(err, ErrProfileAlreadyExists) {
		t.Errorf("Expected ErrProfileAlreadyExists, got %v", err)
	}

	// 3. Create second profile
	p2, err := mgr.CreateProfile("Rohit Soni", "claudeswap-rohit")
	if err != nil {
		t.Fatalf("CreateProfile 2 failed: %v", err)
	}
	if p2.ID != "rohit-soni" || p2.Shortcut != "claudeswap-rohit" {
		t.Errorf("Unexpected profile 2 values: %+v", p2)
	}

	// 4. List profiles
	list, err := mgr.ListProfiles()
	if err != nil {
		t.Fatalf("ListProfiles failed: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("Expected 2 profiles, got %d", len(list))
	}

	// 5. Get profile by ID, Name, Shortcut
	byID, err := mgr.GetProfile("deepak")
	if err != nil || byID.ID != "deepak" {
		t.Errorf("GetProfile by ID failed: %v", err)
	}

	byName, err := mgr.GetProfile("Deepak")
	if err != nil || byName.ID != "deepak" {
		t.Errorf("GetProfile by Name failed: %v", err)
	}

	byShortcut, err := mgr.GetProfile("claudeswap-deepak")
	if err != nil || byShortcut.ID != "deepak" {
		t.Errorf("GetProfile by claudeswap shortcut failed: %v", err)
	}

	// 6. Rename profile
	renamed, err := mgr.RenameProfile("deepak", "Deepak Work", "claudeswap-work")
	if err != nil {
		t.Fatalf("RenameProfile failed: %v", err)
	}
	if renamed.Name != "Deepak Work" || renamed.Shortcut != "claudeswap-work" {
		t.Errorf("Unexpected renamed values: %+v", renamed)
	}

	// 7. Delete profile
	err = mgr.DeleteProfile("rohit-soni", true)
	if err != nil {
		t.Fatalf("DeleteProfile failed: %v", err)
	}

	// Verify rohit-soni config directory removed
	if _, err := os.Stat(p2.ConfigDir); !os.IsNotExist(err) {
		t.Errorf("ConfigDir should have been removed: %s", p2.ConfigDir)
	}

	// List should now have 1
	remaining, err := mgr.ListProfiles()
	if err != nil || len(remaining) != 1 {
		t.Fatalf("Expected 1 profile remaining, got %d", len(remaining))
	}
}

func TestManager_SafeStatusDetection(t *testing.T) {
	tempDir := t.TempDir()
	paths := config.NewPathsWithBase(tempDir)
	store := NewStore(paths)
	mgr := NewManager(paths, store)

	p, err := mgr.CreateProfile("Hero Work", "")
	if err != nil {
		t.Fatalf("CreateProfile failed: %v", err)
	}

	// Newly created profile has empty directory -> StatusLoginRequired
	status := mgr.GetStatus(p)
	if status != StatusLoginRequired {
		t.Errorf("Expected StatusLoginRequired for new empty profile, got %v", status)
	}

	// Simulate Claude Code creating state files in the directory
	dummyFile := filepath.Join(p.ConfigDir, "config.json")
	if err := os.WriteFile(dummyFile, []byte(`{"version":1}`), 0600); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	// Now status should be Ready
	status = mgr.GetStatus(p)
	if status != StatusReady {
		t.Errorf("Expected StatusReady after Claude Code created files, got %v", status)
	}
}
