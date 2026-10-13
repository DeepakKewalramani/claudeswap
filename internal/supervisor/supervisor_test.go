package supervisor

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/claudeswap/claudeswap/internal/config"
	"github.com/claudeswap/claudeswap/internal/profile"
)

func TestSupervisor_RegistryAndLifecycle(t *testing.T) {
	tmpDir := t.TempDir()
	paths := config.NewPathsWithBase(tmpDir)

	sup := NewSupervisor(paths, nil, nil)

	// 1. Initial list should be empty
	sessions, err := sup.ListSessions()
	if err != nil {
		t.Fatalf("ListSessions failed: %v", err)
	}
	if len(sessions) != 0 {
		t.Errorf("Expected 0 sessions, got %d", len(sessions))
	}

	// 2. Mock a managed session directly in registry
	reg := &Registry{
		Sessions: []ManagedSession{
			{
				ID:          "mock-sess-1",
				TaskName:    "billing-v2",
				ProjectName: "autopredict",
				ProjectDir:  tmpDir,
				ProfileName: "ritik-mathur",
				PID:         99999999, // Non-existent PID
				Status:      StateRunning,
				StartedAt:   time.Now().Add(-5 * time.Minute),
				UpdatedAt:   time.Now(),
			},
		},
	}
	if err := sup.saveRegistry(reg); err != nil {
		t.Fatalf("saveRegistry failed: %v", err)
	}

	// 3. List sessions: supervisor should detect that PID 99999999 is dead and mark as COMPLETED
	sessions, err = sup.ListSessions()
	if err != nil {
		t.Fatalf("ListSessions failed: %v", err)
	}
	if len(sessions) != 1 {
		t.Fatalf("Expected 1 session, got %d", len(sessions))
	}
	if sessions[0].Status != StateCompleted {
		t.Errorf("Expected status to update to COMPLETED, got %s", sessions[0].Status)
	}

	// 4. Test KillSession on dead session marks StateStopped
	if err := sup.KillSession("mock-sess-1"); err != nil {
		t.Fatalf("KillSession failed: %v", err)
	}
	sessions, _ = sup.ListSessions()
	if sessions[0].Status != StateStopped {
		t.Errorf("Expected status STOPPED, got %s", sessions[0].Status)
	}

	// 5. Test log writing and reading
	logFile := filepath.Join(tmpDir, "supervisor", "logs", "mock-sess-1.log")
	_ = os.WriteFile(logFile, []byte("line 1\nline 2\nline 3\n"), 0600)

	logs, err := sup.GetLogs("mock-sess-1", 2)
	if err != nil {
		t.Fatalf("GetLogs failed: %v", err)
	}
	if len(logs) != 2 || logs[1] != "line 3" {
		t.Errorf("Unexpected log lines: %v", logs)
	}
}

func TestSupervisor_Validation(t *testing.T) {
	tmpDir := t.TempDir()
	paths := config.NewPathsWithBase(tmpDir)
	sup := NewSupervisor(paths, nil, nil)

	// Starting with nil profile should error cleanly
	_, err := sup.StartSession(StartOptions{
		Profile: nil,
	})
	if err == nil {
		t.Error("Expected error when starting with nil profile")
	}

	// Kill non-existent session
	err = sup.KillSession("does-not-exist")
	if err == nil {
		t.Error("Expected error when killing non-existent session")
	}
}

var _ = profile.Profile{}
