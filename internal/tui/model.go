package tui

import (
	"github.com/claudeswap/claudeswap/internal/profile"

	tea "github.com/charmbracelet/bubbletea"
)

// ActionItem represents a selectable menu option.
type ActionItem string

const (
	ActionAddProfile ActionItem = "Add profile"
	ActionSettings   ActionItem = "Settings"
	ActionDoctor     ActionItem = "Diagnostics"
	ActionExit       ActionItem = "Exit"
)

var defaultActions = []ActionItem{
	ActionAddProfile,
	ActionSettings,
	ActionDoctor,
	ActionExit,
}

// ResultAction is returned by the TUI to the CLI to execute after TUI terminates.
type ResultAction struct {
	Type          string
	TargetProfile *profile.Profile
	Message       string
}

// ProfileWithStatus wraps a profile with its display status.
type ProfileWithStatus struct {
	Profile profile.Profile
	Status  profile.Status
}

// Model represents the Bubble Tea state for the main ClaudeSwap interactive interface.
type Model struct {
	profiles      []ProfileWithStatus
	actions       []ActionItem
	cursor        int
	inActions     bool
	actionCursor  int
	width         int
	height        int
	styles        UIStyles
	Quitting      bool
	SelectedResult *ResultAction
}

// NewModel creates an initialized TUI Model.
func NewModel(profiles []ProfileWithStatus) Model {
	return Model{
		profiles: profiles,
		actions:  defaultActions,
		styles:   DefaultStyles(),
	}
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			m.Quitting = true
			return m, tea.Quit

		case "up", "k":
			if m.inActions {
				if m.actionCursor > 0 {
					m.actionCursor--
				} else if len(m.profiles) > 0 {
					m.inActions = false
					m.cursor = len(m.profiles) - 1
				}
			} else {
				if m.cursor > 0 {
					m.cursor--
				}
			}

		case "down", "j":
			if !m.inActions {
				if m.cursor < len(m.profiles)-1 {
					m.cursor++
				} else {
					m.inActions = true
					m.actionCursor = 0
				}
			} else {
				if m.actionCursor < len(m.actions)-1 {
					m.actionCursor++
				}
			}

		case "tab":
			if len(m.profiles) > 0 {
				m.inActions = !m.inActions
			}

		case "enter":
			if !m.inActions && len(m.profiles) > 0 {
				selected := m.profiles[m.cursor].Profile
				m.SelectedResult = &ResultAction{
					Type:          "launch",
					TargetProfile: &selected,
				}
				return m, tea.Quit
			} else if m.inActions {
				action := m.actions[m.actionCursor]
				switch action {
				case ActionAddProfile:
					m.SelectedResult = &ResultAction{Type: "add"}
					return m, tea.Quit
				case ActionSettings:
					m.SelectedResult = &ResultAction{Type: "settings"}
					return m, tea.Quit
				case ActionDoctor:
					m.SelectedResult = &ResultAction{Type: "doctor"}
					return m, tea.Quit
				case ActionExit:
					m.Quitting = true
					return m, tea.Quit
				}
			}

		case "a":
			m.SelectedResult = &ResultAction{Type: "add"}
			return m, tea.Quit

		case "d":
			m.SelectedResult = &ResultAction{Type: "doctor"}
			return m, tea.Quit

		case "s":
			m.SelectedResult = &ResultAction{Type: "settings"}
			return m, tea.Quit
		}
	}

	return m, nil
}
