package tui

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/claudeswap/claudeswap/internal/session"
)

func TestSessionBrowser_NavigationAndActions(t *testing.T) {
	groups := []session.ProjectGroup{
		{
			Name: "autopredict",
			CWD:  "/Users/test/autopredict",
			Sessions: []session.Session{
				{
					ID:          "11112222-3333-4444-5555-666677778888",
					ProjectName: "autopredict",
					ProjectCWD:  "/Users/test/autopredict",
					GitBranch:   "main",
					Title:       "Fix DNS issue",
					UpdatedAt:   time.Now().Add(-10 * time.Minute),
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

	m := NewSessionBrowserModel(groups)

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

	// 7. Test 's' for handoff
	m = NewSessionBrowserModel(groups)
	m.focusSidebar = false
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}})
	m = updated.(SessionBrowserModel)
	if m.Result == nil || m.Result.Action != "handoff" {
		t.Fatalf("Expected result action 'handoff', got %+v", m.Result)
	}

	// 8. Test 'd' delete confirmation
	m = NewSessionBrowserModel(groups)
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

	// 9. Verify responsive two-column view
	m = NewSessionBrowserModel(groups)
	m.width = 90
	m.height = 30
	view := m.View()
	if !strings.Contains(view, "Folders") || !strings.Contains(view, "autopredict") || !strings.Contains(view, "Fix DNS issue") {
		t.Errorf("View missing key sidebar elements:\n%s", view)
	}
}
