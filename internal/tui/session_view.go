package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/claudeswap/claudeswap/internal/profile"
	"github.com/claudeswap/claudeswap/internal/session"
)

// SessionBrowserResult is returned when the session browser exits.
type SessionBrowserResult struct {
	Action  string // "resume", "new", "switch_profile", "toggle_memory", "handoff", "delete", "back"
	Session *session.Session
	Group   *session.ProjectGroup
	Profile *profile.Profile
}

// SessionBrowserModel provides a two/three-column responsive TUI with a folder sidebar, session list, and activity inspector.
type SessionBrowserModel struct {
	groups           []session.ProjectGroup
	activeProfile    *profile.Profile
	sharedMemory     bool
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

// NewSessionBrowserModel initializes the session explorer TUI with profile and memory state.
func NewSessionBrowserModel(groups []session.ProjectGroup, activeProfile *profile.Profile, sharedMemory bool) SessionBrowserModel {
	return SessionBrowserModel{
		groups:        groups,
		activeProfile: activeProfile,
		sharedMemory:  sharedMemory,
		focusSidebar:  true,
		styles:        DefaultStyles(),
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
						Profile: m.activeProfile,
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
			m.Result = &SessionBrowserResult{Action: "back", Profile: m.activeProfile}
			m.Quitting = true
			return m, tea.Quit

		case "tab":
			if m.focusSidebar && len(m.groups) > 0 {
				m.focusSidebar = false
			} else {
				m.focusSidebar = true
			}

		case "right", "l":
			if m.focusSidebar && len(m.groups) > 0 {
				m.focusSidebar = false
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
					Profile: m.activeProfile,
				}
				return m, tea.Quit
			} else if m.focusSidebar && len(m.groups) > 0 {
				// Switch focus into session list
				m.focusSidebar = false
			}

		case "n": // New session in current group
			m.Result = &SessionBrowserResult{
				Action:  "new",
				Group:   m.currentGroup(),
				Profile: m.activeProfile,
			}
			return m, tea.Quit

		case "s", "p": // Switch profile
			m.Result = &SessionBrowserResult{
				Action:  "switch_profile",
				Session: m.currentSession(),
				Group:   m.currentGroup(),
				Profile: m.activeProfile,
			}
			return m, tea.Quit

		case "m": // Toggle shared memory
			m.sharedMemory = !m.sharedMemory
			m.Result = &SessionBrowserResult{
				Action:  "toggle_memory",
				Session: m.currentSession(),
				Group:   m.currentGroup(),
				Profile: m.activeProfile,
			}
			return m, tea.Quit

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
		totalWidth = 96
	}
	if totalWidth < 60 {
		totalWidth = 60
	}

	// Header line with active profile and memory state
	var header strings.Builder
	titleStr := " ClaudeSwap • Session Explorer "
	header.WriteString(m.styles.Title.Render(titleStr))

	if m.activeProfile != nil {
		profBadge := m.styles.ReadyBadge.Render(fmt.Sprintf(" Profile: %s ", m.activeProfile.Name))
		header.WriteString(" " + profBadge)
	}

	var memBadge string
	if m.sharedMemory {
		memBadge = m.styles.ReadyBadge.Render(" Memory: ON (Shared) ")
	} else {
		memBadge = m.styles.LoggedOutBadge.Render(" Memory: OFF (Isolated) ")
	}
	header.WriteString(" " + memBadge + "\n\n")

	showThreeColumns := totalWidth >= 96

	var sidebarWidth, centerWidth, inspectorWidth int
	if showThreeColumns {
		sidebarWidth = 24
		inspectorWidth = 32
		centerWidth = totalWidth - sidebarWidth - inspectorWidth - 6
		if centerWidth < 36 {
			centerWidth = 36
		}
	} else {
		sidebarWidth = totalWidth * 30 / 100
		if sidebarWidth < 22 {
			sidebarWidth = 22
		} else if sidebarWidth > 30 {
			sidebarWidth = 30
		}
		centerWidth = totalWidth - sidebarWidth - 4
		if centerWidth < 34 {
			centerWidth = 34
		}
	}

	// 1. Render Left Sidebar (Project Folders)
	var sb strings.Builder
	sidebarTitle := fmt.Sprintf("📁 Folders (%d)", len(m.groups))
	sb.WriteString(m.styles.Subtitle.Render(sidebarTitle) + "\n\n")

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

	// 2. Render Center Panel (Sessions List)
	var cb strings.Builder
	curGroup := m.currentGroup()

	if curGroup == nil || len(curGroup.Sessions) == 0 {
		cb.WriteString(m.styles.Subtitle.Render("No sessions in this project") + "\n\n")
		cb.WriteString("Press [n] to start a new Claude session\nin this folder.\n")
	} else {
		groupHeader := fmt.Sprintf("Sessions in %s", curGroup.Name)
		if len(groupHeader) > centerWidth-4 {
			groupHeader = groupHeader[:centerWidth-7] + "..."
		}
		cb.WriteString(m.styles.Subtitle.Render(groupHeader) + "\n\n")

		// Header columns
		hdr := fmt.Sprintf("  %-9s %-12s %-8s %s", "SESSION", "BRANCH", "ACTIVE", "TOPIC")
		if len(hdr) > centerWidth-2 {
			hdr = hdr[:centerWidth-2]
		}
		cb.WriteString(m.styles.Footer.Render(hdr) + "\n")
		cb.WriteString(m.styles.Divider.Render(strings.Repeat("─", centerWidth-2)) + "\n")

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
			if len(branch) > 10 {
				branch = branch[:8] + ".."
			}

			relTime := formatRelativeTime(s.UpdatedAt)
			title := s.Title
			maxTitleLen := centerWidth - 36
			if maxTitleLen > 10 && len(title) > maxTitleLen {
				title = title[:maxTitleLen-3] + "..."
			}

			row := fmt.Sprintf("%s%-9s %-12s %-8s %s", prefix, shortID, branch, relTime, title)
			if len(row) > centerWidth-2 {
				row = row[:centerWidth-2]
			}

			if isCursor {
				cb.WriteString(m.styles.ActiveItem.Render(row) + "\n")
			} else {
				cb.WriteString(m.styles.InactiveItem.Render(row) + "\n")
			}
		}

		// When width is narrow, show compact activity summary at bottom of center panel
		if !showThreeColumns {
			curSess := m.currentSession()
			if curSess != nil && (curSess.LastActivity != "" || len(curSess.WebSources) > 0) {
				cb.WriteString("\n" + m.styles.Divider.Render(strings.Repeat("─", centerWidth-2)) + "\n")
				if curSess.LastActivity != "" {
					cb.WriteString(fmt.Sprintf("⚡ %s\n", curSess.LastActivity))
				}
				if len(curSess.WebSources) > 0 {
					cb.WriteString(fmt.Sprintf("🔗 Sources: %s\n", curSess.WebSources[0]))
				}
			}
		}
	}
	centerBox := m.styles.Box.Width(centerWidth).Render(cb.String())

	// 3. Render Right Inspector Panel (Activity & Web Sources)
	var content string
	if showThreeColumns {
		var ib strings.Builder
		ib.WriteString(m.styles.Subtitle.Render("🔍 Live Inspector") + "\n\n")

		curSess := m.currentSession()
		if curSess == nil {
			ib.WriteString("Select a session to\ninspect details.\n")
		} else {
			ib.WriteString(fmt.Sprintf("🆔 %s\n", curSess.ID[:8]))
			if curSess.GitBranch != "" {
				ib.WriteString(fmt.Sprintf("🌿 %s\n", curSess.GitBranch))
			}
			if curSess.ProjectCWD != "" {
				cwdDisplay := curSess.ProjectCWD
				if len(cwdDisplay) > inspectorWidth-4 {
					cwdDisplay = "..." + cwdDisplay[len(cwdDisplay)-(inspectorWidth-7):]
				}
				ib.WriteString(fmt.Sprintf("📂 %s\n\n", cwdDisplay))
			} else {
				ib.WriteString("\n")
			}

			// Last Activity / Current Tool
			ib.WriteString(m.styles.Footer.Render("⚡ ACTIVITY:") + "\n")
			if curSess.LastActivity != "" {
				act := curSess.LastActivity
				if len(act) > inspectorWidth-2 {
					act = act[:inspectorWidth-5] + "..."
				}
				ib.WriteString(" " + act + "\n\n")
			} else {
				ib.WriteString(" Idle / Completed\n\n")
			}

			// Visited Web Source Links
			ib.WriteString(m.styles.Footer.Render("🔗 WEB SOURCES:") + "\n")
			if len(curSess.WebSources) > 0 {
				for _, src := range curSess.WebSources {
					dispSrc := src
					dispSrc = strings.TrimPrefix(dispSrc, "https://")
					dispSrc = strings.TrimPrefix(dispSrc, "http://")
					if len(dispSrc) > inspectorWidth-4 {
						dispSrc = dispSrc[:inspectorWidth-7] + "..."
					}
					ib.WriteString(fmt.Sprintf(" • %s\n", dispSrc))
				}
			} else {
				ib.WriteString(" None recorded\n")
			}

			// Topic Preview
			if curSess.Title != "" {
				ib.WriteString("\n" + m.styles.Footer.Render("💬 PROMPT:") + "\n")
				p := curSess.Title
				if len(p) > inspectorWidth-2 {
					p = p[:inspectorWidth-5] + "..."
				}
				ib.WriteString(" " + p + "\n")
			}
		}

		inspectorBox := m.styles.Box.Width(inspectorWidth).Render(ib.String())
		content = lipgloss.JoinHorizontal(lipgloss.Top, sidebarBox, " ", centerBox, " ", inspectorBox)
	} else {
		content = lipgloss.JoinHorizontal(lipgloss.Top, sidebarBox, " ", centerBox)
	}

	// Bottom bar / Hotkeys
	var footerText string
	if m.ConfirmDelete {
		curS := m.currentSession()
		footerText = fmt.Sprintf("⚠ PERMANENTLY DELETE session %s? Press [y] to confirm, [n] to cancel", curS.ID[:8])
	} else {
		footerText = "Tab: Switch Pane • Enter: Resume • n: New Session • s: Switch Profile • m: Memory • d: Delete • q: Back"
	}
	footer := m.styles.Footer.Render(centerText(footerText, totalWidth))

	fullView := header.String() + content + "\n\n" + footer

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
