package tui

import (
	"strings"
	"testing"

	"github.com/claudeswap/claudeswap/internal/profile"

	tea "github.com/charmbracelet/bubbletea"
)

func TestTUI_ModelNavigationAndSelection(t *testing.T) {
	profiles := []ProfileWithStatus{
		{
			Profile: profile.Profile{ID: "deepak", Name: "Deepak", Shortcut: "claudeswap-deepak"},
			Status:  profile.StatusReady,
		},
		{
			Profile: profile.Profile{ID: "rohit", Name: "Rohit Soni", Shortcut: "claudeswap-rohit"},
			Status:  profile.StatusLoginRequired,
		},
	}

	m := NewModel(profiles)

	// 1. Initial state
	if m.cursor != 0 || m.inActions {
		t.Fatalf("Expected cursor at 0 and not in actions")
	}

	// 2. Navigate down to second profile
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = updated.(Model)
	if m.cursor != 1 || m.inActions {
		t.Errorf("Expected cursor at 1, got %d (inActions: %v)", m.cursor, m.inActions)
	}

	// 3. Press Enter on second profile -> should select launch for Rohit
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(Model)
	if cmd == nil {
		t.Errorf("Expected tea.Quit cmd, got nil")
	}
	if m.SelectedResult == nil || m.SelectedResult.Type != "launch" || m.SelectedResult.TargetProfile.ID != "rohit" {
		t.Errorf("Unexpected selected result: %+v", m.SelectedResult)
	}

	// 4. Test quick key 'a' for add
	m = NewModel(profiles)
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	m = updated.(Model)
	if m.SelectedResult == nil || m.SelectedResult.Type != "add" {
		t.Errorf("Expected result 'add', got %+v", m.SelectedResult)
	}

	// 5. Test quick key 'd' for doctor
	m = NewModel(profiles)
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	m = updated.(Model)
	if m.SelectedResult == nil || m.SelectedResult.Type != "doctor" {
		t.Errorf("Expected result 'doctor', got %+v", m.SelectedResult)
	}

	// 6. Test 'q' to quit
	m = NewModel(profiles)
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	m = updated.(Model)
	if !m.Quitting {
		t.Errorf("Expected m.Quitting to be true")
	}

	// 7. Verify view output renders correctly
	m = NewModel(profiles)
	view := m.View()
	if !strings.Contains(view, "ClaudeSwap") || !strings.Contains(view, "Deepak") || !strings.Contains(view, "Rohit Soni") {
		t.Errorf("View missing key elements:\n%s", view)
	}
}
