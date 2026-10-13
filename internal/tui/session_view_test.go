package tui

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/claudeswap/claudeswap/internal/profile"
	"github.com/claudeswap/claudeswap/internal/session"
)

func TestSessionBrowser_NavigationAndActions(t *testing.T) {
	testProf := &profile.Profile{
		ID:   "prof-1",
		Name: "ritik-test",
	}

	groups := []session.ProjectGroup{
		{
			Name: "autopredict",
			CWD:  "/Users/test/autopredict",
			Sessions: []session.Session{
				{
					ID:           "11112222-3333-4444-5555-666677778888",
					ProjectName:  "autopredict",
					ProjectCWD:   "/Users/test/autopredict",
					GitBranch:    "main",
					Title:        "Fix DNS issue",
					LastActivity: "WebSearch: Sarvam AI speech",
					WebSources:   []string{"https://docs.sarvam.ai/pricing"},
					UpdatedAt:    time.Now().Add(-10 * time.Minute),
				},
				{
					ID:          "aaaa2222-3333-4444-5555-666677778888",
					ProjectName: "autopredict",
					ProjectCWD:  "/Users/test/autopredict",
					GitBranch:   "dev",
					Title:       "Add auth tests",
					UpdatedAt:   time.Now().Add(-2 * time.Hour),
				},
			},
		},
		{
			Name: "claudeswap",
			CWD:  "/Users/test/claudeswap",
			Sessions: []session.Session{
				{
					ID:          "bbbb2222-3333-4444-5555-666677778888",
					ProjectName: "claudeswap",
					ProjectCWD:  "/Users/test/claudeswap",
					GitBranch:   "feature/sessions",
					Title:       "Implement session sidebar",
					UpdatedAt:   time.Now().Add(-5 * time.Minute),
				},
			},
		},
	}

	m := NewSessionBrowserModel(groups, testProf, true)

	// 1. Initial state: focus on sidebar
	if !m.focusSidebar || m.activeGroupIdx != 0 {
		t.Fatalf("Expected focus on sidebar at index 0")
	}

	// 2. Navigate down on sidebar to next project
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = updated.(SessionBrowserModel)
	if m.activeGroupIdx != 1 {
		t.Errorf("Expected activeGroupIdx 1, got %d", m.activeGroupIdx)
	}

	// 3. Navigate back up
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyUp})
	m = updated.(SessionBrowserModel)
	if m.activeGroupIdx != 0 {
		t.Errorf("Expected activeGroupIdx 0, got %d", m.activeGroupIdx)
	}

	// 4. Tab over to session list
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyTab})
	m = updated.(SessionBrowserModel)
	if m.focusSidebar {
		t.Errorf("Expected focus on session list after Tab")
	}

	// 5. Navigate down in session list
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = updated.(SessionBrowserModel)
	if m.activeSessionIdx != 1 {
		t.Errorf("Expected activeSessionIdx 1, got %d", m.activeSessionIdx)
	}

	// 6. Press Enter on selected session -> action "resume"
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(SessionBrowserModel)
	if cmd == nil || m.Result == nil || m.Result.Action != "resume" {
		t.Fatalf("Expected result action 'resume', got %+v", m.Result)
	}
	if m.Result.Session.ID != "aaaa2222-3333-4444-5555-666677778888" {
		t.Errorf("Unexpected resumed session ID: %s", m.Result.Session.ID)
	}

	// 7. Test 'n' for new session
	m = NewSessionBrowserModel(groups, testProf, true)
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	m = updated.(SessionBrowserModel)
	if m.Result == nil || m.Result.Action != "new" {
		t.Fatalf("Expected result action 'new', got %+v", m.Result)
	}

	// 8. Test 's' for switch profile
	m = NewSessionBrowserModel(groups, testProf, true)
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}})
	m = updated.(SessionBrowserModel)
	if m.Result == nil || m.Result.Action != "switch_profile" {
		t.Fatalf("Expected result action 'switch_profile', got %+v", m.Result)
	}

	// 9. Test 'd' delete confirmation
	m = NewSessionBrowserModel(groups, testProf, true)
	m.focusSidebar = false
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	m = updated.(SessionBrowserModel)
	if !m.ConfirmDelete {
		t.Errorf("Expected ConfirmDelete to be true")
	}

	// Confirm delete with 'y'
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	m = updated.(SessionBrowserModel)
	if m.Result == nil || m.Result.Action != "delete" {
		t.Fatalf("Expected result action 'delete', got %+v", m.Result)
	}

	// 10. Test Tab Switching with '2' and ']'
	m = NewSessionBrowserModel(groups, testProf, true)
	if len(m.tabs) != 2 {
		t.Fatalf("Expected 2 tabs, got %d", len(m.tabs))
	}
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'2'}})
	m = updated.(SessionBrowserModel)
	if m.activeTabIdx != 1 || m.activeGroupIdx != 1 {
		t.Errorf("Expected tab 2 selection, got activeTabIdx=%d, activeGroupIdx=%d", m.activeTabIdx, m.activeGroupIdx)
	}

	// Test 'x' for Mission Control matrix
	m = NewSessionBrowserModel(groups, testProf, true)
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
	m = updated.(SessionBrowserModel)
	if m.Result == nil || m.Result.Action != "matrix" {
		t.Fatalf("Expected result action 'matrix', got %+v", m.Result)
	}

	// 11. Verify 3-column wide view with live inspector, web sources, and tab strip
	m = NewSessionBrowserModel(groups, testProf, true)
	m.focusSidebar = false
	m.width = 110
	m.height = 30
	view := m.View()
	if !strings.Contains(view, "Folders") || !strings.Contains(view, "ritik-test") || !strings.Contains(strings.ToLower(view), "telemetry") {
		t.Errorf("View missing key 3-column elements:\n%s", view)
	}
	if !strings.Contains(view, "Tabs:") || !strings.Contains(view, "autopredict") {
		t.Errorf("View missing tabs strip:\n%s", view)
	}
	if !strings.Contains(view, "Sarvam AI") || !strings.Contains(view, "docs.sarvam.ai/pricing") {
		t.Errorf("View missing live activity and web sources inspector elements:\n%s", view)
	}
}
