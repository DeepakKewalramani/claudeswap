package session

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/claudeswap/claudeswap/internal/config"
	"github.com/claudeswap/claudeswap/internal/profile"
)

func TestSessionManager_ScanAndGroup(t *testing.T) {
	tempDir := t.TempDir()
	paths := config.NewPathsWithBase(tempDir)
	profStore := profile.NewStore(paths)
	mgr := NewManager(paths, profStore)

	// Create mock shared projects directory
	sharedProjects := filepath.Join(paths.SharedDir(), "projects")
	proj1 := filepath.Join(sharedProjects, "-Users-test-myproject")
	if err := os.MkdirAll(proj1, 0700); err != nil {
		t.Fatalf("Failed to create mock project dir: %v", err)
	}

	// Create mock session jsonl
	sessFile := filepath.Join(proj1, "11112222-3333-4444-5555-666677778888.jsonl")
	content := `{"type":"mode","mode":"code","sessionId":"11112222-3333-4444-5555-666677778888"}
{"type":"user","cwd":"/Users/test/myproject","gitBranch":"feature-x","timestamp":"2026-10-13T10:00:00Z","message":{"content":"Implement user login API"}}
`
	if err := os.WriteFile(sessFile, []byte(content), 0600); err != nil {
		t.Fatalf("Failed to write mock session: %v", err)
	}

	groups, err := mgr.ScanProjects()
	if err != nil {
		t.Fatalf("ScanProjects failed: %v", err)
	}

	if len(groups) != 1 {
		t.Fatalf("Expected 1 project group, got %d", len(groups))
	}

	if groups[0].Name != "myproject" {
		t.Errorf("Expected project name 'myproject', got %q", groups[0].Name)
	}

	if len(groups[0].Sessions) != 1 {
		t.Fatalf("Expected 1 session in group, got %d", len(groups[0].Sessions))
	}

	sess := groups[0].Sessions[0]
	if sess.ID != "11112222-3333-4444-5555-666677778888" {
		t.Errorf("Unexpected session ID: %s", sess.ID)
	}
	if sess.GitBranch != "feature-x" {
		t.Errorf("Expected branch 'feature-x', got %q", sess.GitBranch)
	}
	if sess.Title != "Implement user login API" {
		t.Errorf("Expected title 'Implement user login API', got %q", sess.Title)
	}

	// Test find session
	found, _, err := mgr.FindSessionByID(groups, "11112222")
	if err != nil || found == nil {
		t.Fatalf("FindSessionByID failed: %v", err)
	}

	// Test delete session
	if err := mgr.DeleteSession(sess.FilePath); err != nil {
		t.Fatalf("DeleteSession failed: %v", err)
	}

	if _, err := os.Stat(sess.FilePath); !os.IsNotExist(err) {
		t.Errorf("Expected file to be deleted: %s", sess.FilePath)
	}
}

func TestSessionManager_CleanTitle(t *testing.T) {
	raw := "<pasted_content id=\"123\">Hello world</pasted_content> How are you?"
	cleaned := cleanTitle(raw)
	if cleaned != "Hello world How are you?" {
		t.Errorf("Unexpected cleaned title: %q", cleaned)
	}
}
