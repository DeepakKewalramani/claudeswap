package shortcut

import (
	"os"
	"runtime"
	"strings"
	"testing"

	"github.com/claudeswap/claudeswap/internal/config"
	"github.com/claudeswap/claudeswap/internal/profile"
)

func TestShortcutManager(t *testing.T) {
	tempDir := t.TempDir()
	paths := config.NewPathsWithBase(tempDir)
	mgr := NewManager(paths)

	p := &profile.Profile{
		ID:       "deepak",
		Name:     "Deepak",
		Shortcut: "claudeswap-deepak",
	}

	// 1. Create shortcut
	path, err := mgr.CreateShortcut(p)
	if err != nil {
		t.Fatalf("CreateShortcut failed: %v", err)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat shortcut failed: %v", err)
	}

	if runtime.GOOS != "windows" {
		if perm := info.Mode().Perm(); perm&0111 == 0 {
			t.Errorf("Shortcut is missing execute permissions: %04o", perm)
		}
	}

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}

	if !strings.Contains(string(content), "launch deepak") {
		t.Errorf("Shortcut content missing expected launcher invocation: %s", string(content))
	}

	// 2. Recreate with new shortcut name
	p.Shortcut = "claudeswap-work"
	newPath, err := mgr.RecreateShortcut(p, "claudeswap-deepak")
	if err != nil {
		t.Fatalf("RecreateShortcut failed: %v", err)
	}

	// Old file should be gone
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Errorf("Old shortcut file should be removed: %s", path)
	}

	// New file should exist
	if _, err := os.Stat(newPath); err != nil {
		t.Errorf("New shortcut file should exist: %s", newPath)
	}

	// 3. Remove shortcut
	if err := mgr.RemoveShortcut("claudeswap-work"); err != nil {
		t.Fatalf("RemoveShortcut failed: %v", err)
	}

	if _, err := os.Stat(newPath); !os.IsNotExist(err) {
		t.Errorf("Shortcut file was not removed: %s", newPath)
	}
}
