package profile

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

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

	// Newly created profile has empty directory -> StatusLoggedOut
	status := mgr.GetStatus(p)
	if status != StatusLoggedOut && status != StatusLoginRequired {
		t.Errorf("Expected StatusLoggedOut for new empty profile, got %v", status)
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

	// Test SessionOut detection when statusChecker reports logged out
	mgr.SetStatusChecker(func(configDir string) (bool, error) {
		return false, nil
	})
	status = mgr.GetStatus(p)
	if status != StatusSessionOut {
		t.Errorf("Expected StatusSessionOut when account exists but loggedIn is false, got %v", status)
	}
}

func TestManager_StatusLoggedOut_WithSharedContextOrFiles(t *testing.T) {
	tempDir := t.TempDir()
	paths := config.NewPathsWithBase(tempDir)
	store := NewStore(paths)
	mgr := NewManager(paths, store)

	p, err := mgr.CreateProfile("Jane Doe", "")
	if err != nil {
		t.Fatalf("CreateProfile failed: %v", err)
	}

	// Simulate profile directory having entries (e.g. shared folders, cache) without auth
	_ = os.MkdirAll(filepath.Join(p.ConfigDir, "sessions"), 0700)
	_ = os.MkdirAll(filepath.Join(p.ConfigDir, "projects"), 0700)
	_ = os.WriteFile(filepath.Join(p.ConfigDir, ".claude.json"), []byte(`{"firstStartTime":"2026-10-13"}`), 0600)

	status := mgr.GetStatus(p)
	if status != StatusLoggedOut {
		t.Errorf("Expected StatusLoggedOut even when directory has folders/files but no account, got %v", status)
	}
}

func TestManager_StatusSessionOut_WithDaemonStatus(t *testing.T) {
	tempDir := t.TempDir()
	paths := config.NewPathsWithBase(tempDir)
	store := NewStore(paths)
	mgr := NewManager(paths, store)

	p, err := mgr.CreateProfile("Alice Smith", "")
	if err != nil {
		t.Fatalf("CreateProfile failed: %v", err)
	}

	// Write .claude.json with oauthAccount
	claudeJSON := filepath.Join(p.ConfigDir, ".claude.json")
	_ = os.WriteFile(claudeJSON, []byte(`{"oauthAccount":{"accountUuid":"123","emailAddress":"alice@test.com"}}`), 0600)

	// Write daemon-auth-status.json with auth_required
	daemonJSON := filepath.Join(p.ConfigDir, "daemon-auth-status.json")
	_ = os.WriteFile(daemonJSON, []byte(`{"status":"auth_required"}`), 0600)

	// Touch daemonJSON so mtime is after claudeJSON
	now := time.Now().Add(time.Second)
	_ = os.Chtimes(daemonJSON, now, now)

	status := mgr.GetStatus(p)
	if status != StatusSessionOut {
		t.Errorf("Expected StatusSessionOut when daemon indicates auth_required, got %v", status)
	}
}

func TestManager_PreloadStatuses(t *testing.T) {
	tempDir := t.TempDir()
	paths := config.NewPathsWithBase(tempDir)
	store := NewStore(paths)
	mgr := NewManager(paths, store)

	p1, _ := mgr.CreateProfile("User1", "")
	p2, _ := mgr.CreateProfile("User2", "")

	profiles := []Profile{*p1, *p2}
	mgr.PreloadStatuses(profiles)

	if s := mgr.GetStatus(p1); s != StatusLoggedOut {
		t.Errorf("Expected p1 StatusLoggedOut, got %v", s)
	}
	if s := mgr.GetStatus(p2); s != StatusLoggedOut {
		t.Errorf("Expected p2 StatusLoggedOut, got %v", s)
	}
}
