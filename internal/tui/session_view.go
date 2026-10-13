package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/claudeswap/claudeswap/internal/session"
)

// SessionBrowserResult is returned when the session browser exits.
type SessionBrowserResult struct {
	Action  string // "resume", "handoff", "delete", "back"
	Session *session.Session
	Group   *session.ProjectGroup
}

// SessionBrowserModel provides a two-column responsive TUI with a folder sidebar and session list.
type SessionBrowserModel struct {
	groups           []session.ProjectGroup
	activeGroupIdx   int
	activeSessionIdx int
	focusSidebar     bool // true = sidebar has focus, false = session list has focus
	width            int
	height           int
	styles           UIStyles
	Quitting         bool
	ConfirmDelete    bool
	Result           *SessionBrowserResult
}

// NewSessionBrowserModel initializes the session explorer TUI.
func NewSessionBrowserModel(groups []session.ProjectGroup) SessionBrowserModel {
	return SessionBrowserModel{
		groups:       groups,
		focusSidebar: true,
		styles:       DefaultStyles(),
	}
}

func (m SessionBrowserModel) Init() tea.Cmd {
	return nil
}

func (m SessionBrowserModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		if m.ConfirmDelete {
			switch strings.ToLower(msg.String()) {
			case "y":
				m.ConfirmDelete = false
				if curSess := m.currentSession(); curSess != nil {
					m.Result = &SessionBrowserResult{
						Action:  "delete",
						Session: curSess,
						Group:   m.currentGroup(),
					}
					return m, tea.Quit
				}
			case "n", "esc", "q":
				m.ConfirmDelete = false
			}
			return m, nil
		}

		switch msg.String() {
		case "ctrl+c", "q", "esc":
			m.Result = &SessionBrowserResult{Action: "back"}
			m.Quitting = true
			return m, tea.Quit

		case "tab", "right", "l":
			if m.focusSidebar && len(m.groups) > 0 {
				m.focusSidebar = false
			} else {
				m.focusSidebar = true
			}

		case "left", "h":
			if !m.focusSidebar {
				m.focusSidebar = true
			}

		case "up", "k":
			if m.focusSidebar {
				if m.activeGroupIdx > 0 {
					m.activeGroupIdx--
					m.activeSessionIdx = 0
				}
			} else {
				if m.activeSessionIdx > 0 {
					m.activeSessionIdx--
				}
			}

		case "down", "j":
			if m.focusSidebar {
				if m.activeGroupIdx < len(m.groups)-1 {
					m.activeGroupIdx++
					m.activeSessionIdx = 0
				}
			} else {
				curGroup := m.currentGroup()
				if curGroup != nil && m.activeSessionIdx < len(curGroup.Sessions)-1 {
					m.activeSessionIdx++
				}
			}

		case "enter":
			curSess := m.currentSession()
			if curSess != nil {
				m.Result = &SessionBrowserResult{
					Action:  "resume",
					Session: curSess,
					Group:   m.currentGroup(),
				}
				return m, tea.Quit
			}

		case "s": // Switch / Handoff
			curSess := m.currentSession()
			if curSess != nil {
				m.Result = &SessionBrowserResult{
					Action:  "handoff",
					Session: curSess,
					Group:   m.currentGroup(),
				}
				return m, tea.Quit
			}

		case "d", "x": // Delete
			if m.currentSession() != nil {
				m.ConfirmDelete = true
			}
		}
	}

	return m, nil
}

func (m SessionBrowserModel) currentGroup() *session.ProjectGroup {
	if len(m.groups) == 0 || m.activeGroupIdx >= len(m.groups) {
		return nil
	}
	return &m.groups[m.activeGroupIdx]
}

func (m SessionBrowserModel) currentSession() *session.Session {
	g := m.currentGroup()
	if g == nil || len(g.Sessions) == 0 || m.activeSessionIdx >= len(g.Sessions) {
		return nil
	}
	return &g.Sessions[m.activeSessionIdx]
}

func (m SessionBrowserModel) View() string {
	if m.Quitting {
		return ""
	}

	totalWidth := m.width
	if totalWidth <= 0 {
		totalWidth = 80
	}
	if totalWidth < 60 {
		totalWidth = 60
	}

	sidebarWidth := totalWidth * 30 / 100
	if sidebarWidth < 22 {
		sidebarWidth = 22
	} else if sidebarWidth > 32 {
		sidebarWidth = 32
	}

	mainWidth := totalWidth - sidebarWidth - 4
	if mainWidth < 34 {
		mainWidth = 34
	}

	// 1. Render Left Sidebar (Project Folders)
	var sb strings.Builder
	sidebarTitle := fmt.Sprintf("📁 Folders (%d)", len(m.groups))
	sb.WriteString(m.styles.Title.Render(sidebarTitle) + "\n\n")

	if len(m.groups) == 0 {
		sb.WriteString("  No projects found.\n")
	} else {
		for i, g := range m.groups {
			isCursor := m.focusSidebar && m.activeGroupIdx == i
			isSelected := !m.focusSidebar && m.activeGroupIdx == i

			prefix := "  "
			if isCursor {
				prefix = "❯ "
			} else if isSelected {
				prefix = "● "
			}

			countStr := fmt.Sprintf("(%d)", len(g.Sessions))
			maxNameLen := sidebarWidth - len(countStr) - 6
			name := g.Name
			if len(name) > maxNameLen && maxNameLen > 3 {
				name = name[:maxNameLen-3] + "..."
			}

			line := fmt.Sprintf("%s%-*s %s", prefix, maxNameLen, name, countStr)
			if isCursor {
				sb.WriteString(m.styles.ActiveItem.Render(line) + "\n")
			} else if isSelected {
				sb.WriteString(m.styles.SelectedAction.Render(line) + "\n")
			} else {
				sb.WriteString(m.styles.InactiveItem.Render(line) + "\n")
			}
		}
	}

	sidebarBox := m.styles.Box.Width(sidebarWidth).Render(sb.String())

	// 2. Render Right Panel (Sessions List)
	var mb strings.Builder
	curGroup := m.currentGroup()

	if curGroup == nil || len(curGroup.Sessions) == 0 {
		mb.WriteString(m.styles.Subtitle.Render("No active sessions in this project") + "\n\n")
		mb.WriteString("Start a conversation in this project using:\n  claude\n")
	} else {
		groupHeader := fmt.Sprintf("Sessions in %s (%s)", curGroup.Name, curGroup.CWD)
		if len(groupHeader) > mainWidth-4 {
			groupHeader = groupHeader[:mainWidth-7] + "..."
		}
		mb.WriteString(m.styles.Subtitle.Render(groupHeader) + "\n\n")

		// Header columns
		hdr := fmt.Sprintf("  %-9s %-15s %-10s %s", "SESSION", "BRANCH", "ACTIVE", "TOPIC / FIRST PROMPT")
		if len(hdr) > mainWidth-2 {
			hdr = hdr[:mainWidth-2]
		}
		mb.WriteString(m.styles.Footer.Render(hdr) + "\n")
		mb.WriteString(m.styles.Divider.Render(strings.Repeat("─", mainWidth-2)) + "\n")

		for i, s := range curGroup.Sessions {
			isCursor := !m.focusSidebar && m.activeSessionIdx == i

			prefix := "  "
			if isCursor {
				prefix = "❯ "
			}

			shortID := s.ID
			if len(shortID) > 8 {
				shortID = shortID[:8]
			}

			branch := s.GitBranch
			if branch == "" {
				branch = "main"
			}
			if len(branch) > 13 {
				branch = branch[:11] + ".."
			}

			relTime := formatRelativeTime(s.UpdatedAt)
			title := s.Title
			maxTitleLen := mainWidth - 40
			if maxTitleLen > 10 && len(title) > maxTitleLen {
				title = title[:maxTitleLen-3] + "..."
			}

			row := fmt.Sprintf("%s%-9s %-15s %-10s %s", prefix, shortID, branch, relTime, title)
			if len(row) > mainWidth-2 {
				row = row[:mainWidth-2]
			}

			if isCursor {
				mb.WriteString(m.styles.ActiveItem.Render(row) + "\n")
			} else {
				mb.WriteString(m.styles.InactiveItem.Render(row) + "\n")
			}
		}
	}

	mainBox := m.styles.Box.Width(mainWidth).Render(mb.String())

	// Join Left Sidebar and Right Panel horizontally
	content := lipgloss.JoinHorizontal(lipgloss.Top, sidebarBox, " ", mainBox)

	// Bottom bar / Hotkeys
	var footerText string
	if m.ConfirmDelete {
		curS := m.currentSession()
		footerText = fmt.Sprintf("⚠ PERMANENTLY DELETE session %s? Press [y] to confirm, [n] to cancel", curS.ID[:8])
	} else {
		footerText = "Tab/←→: Switch Pane • Enter: Resume • s: Switch Account • d: Delete • q/Esc: Back"
	}
	footer := m.styles.Footer.Render(centerText(footerText, totalWidth))

	fullView := content + "\n\n" + footer

	if m.width > 0 && m.height > 0 {
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, fullView)
	}

	return fullView + "\n"
}

func formatRelativeTime(t time.Time) string {
	diff := time.Since(t)
	switch {
	case diff < time.Minute:
		return "just now"
	case diff < time.Hour:
		return fmt.Sprintf("%dm ago", int(diff.Minutes()))
	case diff < 24*time.Hour:
		return fmt.Sprintf("%dh ago", int(diff.Hours()))
	case diff < 7*24*time.Hour:
		return fmt.Sprintf("%dd ago", int(diff.Hours()/24))
	default:
		return t.Format("Jan 02")
	}
}
