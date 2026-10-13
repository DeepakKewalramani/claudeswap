package tui

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/claudeswap/claudeswap/internal/supervisor"
)

func TestSupervisorModel_NavigationAndActions(t *testing.T) {
	sessions := []supervisor.ManagedSession{
		{
			ID:           "sess-1",
			TaskName:     "billing-v2",
			ProjectName:  "autopredict",
			ProjectDir:   "/Users/test/autopredict",
			ProfileName:  "ritik",
			PID:          1234,
			Status:       supervisor.StateRunning,
			LastActivity: "Bash: npm test",
			StartedAt:    time.Now().Add(-2 * time.Minute),
		},
		{
			ID:           "sess-2",
			TaskName:     "redis-cache",
			ProjectName:  "hero-data",
			ProjectDir:   "/Users/test/hero-data",
			ProfileName:  "work",
			PID:          5678,
			Status:       supervisor.StateWaitingInput,
			LastActivity: "Prompt: Confirm rm -rf",
			StartedAt:    time.Now().Add(-10 * time.Minute),
		},
	}

	mockLogs := func(id string, tail int) []string {
		return []string{"[output] building project...", "[output] tests running"}
	}

	m := NewSupervisorModel(sessions, mockLogs, func() []supervisor.ManagedSession {
		return sessions
	})

	// 1. Initial state
	if m.cursor != 0 {
		t.Fatalf("Expected cursor 0, got %d", m.cursor)
	}

	// 2. Navigate down
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = updated.(SupervisorModel)
	if m.cursor != 1 {
		t.Fatalf("Expected cursor 1, got %d", m.cursor)
	}

	// 3. Attach with Enter
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(SupervisorModel)
	if cmd == nil || m.Result == nil || m.Result.Action != "attach" {
		t.Fatalf("Expected attach action, got %+v", m.Result)
	}
	if m.Result.Session.ID != "sess-2" {
		t.Errorf("Expected attached session sess-2, got %s", m.Result.Session.ID)
	}

	// 4. Test 'k' kill confirmation
	m = NewSupervisorModel(sessions, mockLogs, nil)
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	m = updated.(SupervisorModel)
	if !m.ConfirmKill {
		t.Errorf("Expected ConfirmKill true")
	}

	// Confirm with 'y'
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	m = updated.(SupervisorModel)
	if m.Result == nil || m.Result.Action != "kill" {
		t.Fatalf("Expected kill action, got %+v", m.Result)
	}

	// 5. Test rendering
	m = NewSupervisorModel(sessions, mockLogs, nil)
	m.width = 110
	m.height = 30
	view := m.View()
	if !strings.Contains(view, "Multi-Session Mission Control") || !strings.Contains(view, "billing-v2") || !strings.Contains(view, "RUNNING") {
		t.Errorf("View missing key table elements:\n%s", view)
	}
	if !strings.Contains(view, "Live Output") || !strings.Contains(view, "building project") {
		t.Errorf("View missing live log drawer:\n%s", view)
	}
}
