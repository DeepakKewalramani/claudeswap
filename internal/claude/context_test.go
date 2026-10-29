package claude

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/claudeswap/claudeswap/internal/config"
)

func TestSyncSharedContext_ToggleOnOff(t *testing.T) {
	tempDir := t.TempDir()
	paths := config.NewPathsWithBase(tempDir)
	profileConfigDir := filepath.Join(tempDir, "profiles", "test-user")
	if err := os.MkdirAll(profileConfigDir, 0700); err != nil {
		t.Fatalf("Failed to create profileConfigDir: %v", err)
	}

	// 1. Initial State: Enable shared context
	if err := SyncSharedContext(paths, profileConfigDir, true); err != nil {
		t.Fatalf("Failed to enable shared context: %v", err)
	}

	// Verify symlinks exist
	for _, dir := range sharedDirectories {
		p := filepath.Join(profileConfigDir, dir)
		fi, err := os.Lstat(p)
		if err != nil || fi.Mode()&os.ModeSymlink == 0 {
			t.Fatalf("Expected %s to be a symlink when enabled", dir)
		}
	}
	// Verify history symlink
	histPath := filepath.Join(profileConfigDir, "history.jsonl")
	fi, err := os.Lstat(histPath)
	if err != nil || fi.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("Expected history.jsonl to be a symlink when enabled")
	}

	// Write a shared session file
	sharedFile := filepath.Join(profileConfigDir, "sessions", "session-shared.json")
	if err := os.WriteFile(sharedFile, []byte("shared content"), 0600); err != nil {
		t.Fatalf("Failed to write to shared session: %v", err)
	}

	// 2. Disable shared context (isolate profile)
	if err := SyncSharedContext(paths, profileConfigDir, false); err != nil {
		t.Fatalf("Failed to disable shared context: %v", err)
	}

	// Verify symlinks are removed and real directories exist
	for _, dir := range sharedDirectories {
		p := filepath.Join(profileConfigDir, dir)
		fi, err := os.Lstat(p)
		if err != nil {
			t.Fatalf("Expected directory %s to exist when disabled: %v", dir, err)
		}
		if fi.Mode()&os.ModeSymlink != 0 {
			t.Fatalf("Expected %s to NOT be a symlink when disabled", dir)
		}
	}
	// Verify history.jsonl symlink is removed
	if _, err := os.Lstat(histPath); err == nil {
		t.Fatalf("Expected history.jsonl symlink to be removed when disabled")
	}

	// Verify the shared session file is preserved in the shared pool, but NOT in isolated profile
	isolatedFile := filepath.Join(profileConfigDir, "sessions", "session-shared.json")
	if _, err := os.Stat(isolatedFile); err == nil {
		t.Fatalf("Isolated profile sessions directory should be clean and not have shared file")
	}
	sharedPoolFile := filepath.Join(paths.SharedDir(), "sessions", "session-shared.json")
	if _, err := os.Stat(sharedPoolFile); err != nil {
		t.Fatalf("Shared pool file should still exist: %v", err)
	}

	// 3. Write a session while isolated
	localFile := filepath.Join(profileConfigDir, "sessions", "session-local.json")
	if err := os.WriteFile(localFile, []byte("local content"), 0600); err != nil {
		t.Fatalf("Failed to write local session: %v", err)
	}

	// 4. Re-enable shared context: should merge local sessions into shared pool and restore symlinks
	if err := SyncSharedContext(paths, profileConfigDir, true); err != nil {
		t.Fatalf("Failed to re-enable shared context: %v", err)
	}

	// Verify symlinks restored
	fi, err = os.Lstat(filepath.Join(profileConfigDir, "sessions"))
	if err != nil || fi.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("Expected sessions to be symlink again after re-enabling")
	}

	// Verify both old shared session and newly merged local session are in the shared pool!
	mergedLocal := filepath.Join(paths.SharedDir(), "sessions", "session-local.json")
	if _, err := os.Stat(mergedLocal); err != nil {
		t.Fatalf("Expected local session to be merged into shared pool: %v", err)
	}
	if _, err := os.Stat(sharedPoolFile); err != nil {
		t.Fatalf("Expected original shared session to still exist: %v", err)
	}
}

