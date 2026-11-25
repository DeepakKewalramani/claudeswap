package claude

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/DeepakKewalramani/claudeswap/internal/config"
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


func TestSyncSharedContext_KeepsConflictingProfileFiles(t *testing.T) {
	tempDir := t.TempDir()
	paths := config.NewPathsWithBase(tempDir)
	profileConfigDir := filepath.Join(tempDir, "profiles", "work")

	sharedFile := filepath.Join(paths.SharedDir(), "projects", "repo", "abc.jsonl")
	profileFile := filepath.Join(profileConfigDir, "projects", "repo", "abc.jsonl")
	for path, content := range map[string]string{sharedFile: "shared version", profileFile: "profile version"} {
		if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0600); err != nil {
			t.Fatal(err)
		}
	}

	if err := SyncSharedContext(paths, profileConfigDir, true); err != nil {
		t.Fatalf("enable failed: %v", err)
	}

	if got, _ := os.ReadFile(sharedFile); string(got) != "shared version" {
		t.Fatalf("shared file was overwritten: %q", got)
	}
	conflict := sharedFile + ".conflict-work"
	got, err := os.ReadFile(conflict)
	if err != nil || string(got) != "profile version" {
		t.Fatalf("expected profile's conflicting file preserved at %s, got %q (%v)", conflict, got, err)
	}
}

func TestSyncSharedContext_AbortsWithoutDeletingWhenMergeFails(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("permission checks do not apply to root")
	}
	tempDir := t.TempDir()
	paths := config.NewPathsWithBase(tempDir)
	profileConfigDir := filepath.Join(tempDir, "profiles", "work")

	unreadable := filepath.Join(profileConfigDir, "sessions", "locked.json")
	if err := os.MkdirAll(filepath.Dir(unreadable), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(unreadable, []byte("important"), 0000); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(unreadable, 0600) })

	if err := SyncSharedContext(paths, profileConfigDir, true); err == nil {
		t.Fatal("expected an error when a profile file cannot be merged")
	}

	fi, err := os.Lstat(filepath.Join(profileConfigDir, "sessions"))
	if err != nil || !fi.IsDir() || fi.Mode()&os.ModeSymlink != 0 {
		t.Fatalf("profile sessions directory must be left in place after a failed merge")
	}
	if _, err := os.Lstat(unreadable); err != nil {
		t.Fatalf("unmerged profile file was deleted: %v", err)
	}
}

func TestSyncSharedContext_MergesExistingHistory(t *testing.T) {
	tempDir := t.TempDir()
	paths := config.NewPathsWithBase(tempDir)
	profileConfigDir := filepath.Join(tempDir, "profiles", "work")
	if err := os.MkdirAll(profileConfigDir, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(paths.SharedDir(), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(paths.SharedDir(), "history.jsonl"), []byte("{\"a\":1}\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(profileConfigDir, "history.jsonl"), []byte("{\"b\":2}"), 0600); err != nil {
		t.Fatal(err)
	}

	if err := SyncSharedContext(paths, profileConfigDir, true); err != nil {
		t.Fatalf("enable failed: %v", err)
	}

	got, err := os.ReadFile(filepath.Join(profileConfigDir, "history.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "{\"a\":1}\n{\"b\":2}\n" {
		t.Fatalf("expected profile history appended to shared history, got %q", got)
	}
}
