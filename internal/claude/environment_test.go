package claude

import (
	"os"
	"strings"
	"testing"
)

func TestBuildEnvironment(t *testing.T) {
	// Temporarily set a dummy CLAUDE_CONFIG_DIR in current environment
	orig := os.Getenv("CLAUDE_CONFIG_DIR")
	defer func() {
		if orig != "" {
			_ = os.Setenv("CLAUDE_CONFIG_DIR", orig)
		} else {
			_ = os.Unsetenv("CLAUDE_CONFIG_DIR")
		}
	}()
	_ = os.Setenv("CLAUDE_CONFIG_DIR", "/some/inherited/path")

	targetDir := "/Users/test/.claudeswap/profiles/deepak"
	env := BuildEnvironment(targetDir)

	count := 0
	for _, entry := range env {
		if strings.HasPrefix(entry, "CLAUDE_CONFIG_DIR=") {
			count++
			if entry != "CLAUDE_CONFIG_DIR="+targetDir {
				t.Errorf("Unexpected CLAUDE_CONFIG_DIR entry: %q, want %q", entry, "CLAUDE_CONFIG_DIR="+targetDir)
			}
		}
	}

	if count != 1 {
		t.Errorf("Expected exactly 1 CLAUDE_CONFIG_DIR entry in environment, found %d", count)
	}
}
