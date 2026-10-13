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

// SessionTab represents an open tab across workspaces/terminals.
type SessionTab struct {
	Index       int
	ProjectName string
	SessionID   string
	GitBranch   string
	StatusBadge string
	GroupIdx    int
	SessionIdx  int
}

// SessionBrowserModel provides a two/three-column responsive TUI with a folder sidebar, session list, and activity inspector.
type SessionBrowserModel struct {
	groups           []session.ProjectGroup
	tabs             []SessionTab
	activeTabIdx     int
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
	var tabs []SessionTab
	tabNum := 1
	for gi, g := range groups {
		if len(g.Sessions) > 0 && tabNum <= 8 {
			badge := "🟢"
			if time.Since(g.Sessions[0].UpdatedAt) > 2*time.Hour {
				badge = "⚪"
			}
			shortID := g.Sessions[0].ID
			if len(shortID) > 8 {
				shortID = shortID[:8]
			}
			tabs = append(tabs, SessionTab{
				Index:       tabNum,
				ProjectName: g.Name,
				SessionID:   shortID,
				GitBranch:   g.Sessions[0].GitBranch,
				StatusBadge: badge,
				GroupIdx:    gi,
				SessionIdx:  0,
			})
			tabNum++
		}
	}

	return SessionBrowserModel{
		groups:        groups,
		tabs:          tabs,
		activeTabIdx:  0,
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

		case "1", "2", "3", "4", "5", "6", "7", "8", "9":
			if len(msg.Runes) > 0 {
				tabIdx := int(msg.Runes[0] - '1')
				if tabIdx < len(m.tabs) {
					tab := m.tabs[tabIdx]
					m.activeGroupIdx = tab.GroupIdx
					m.activeSessionIdx = tab.SessionIdx
					m.activeTabIdx = tabIdx
					m.focusSidebar = false
				}
			}

		case "[":
			if len(m.tabs) > 0 {
				m.activeTabIdx = (m.activeTabIdx - 1 + len(m.tabs)) % len(m.tabs)
				tab := m.tabs[m.activeTabIdx]
				m.activeGroupIdx = tab.GroupIdx
				m.activeSessionIdx = tab.SessionIdx
				m.focusSidebar = false
			}

		case "]":
			if len(m.tabs) > 0 {
				m.activeTabIdx = (m.activeTabIdx + 1) % len(m.tabs)
				tab := m.tabs[m.activeTabIdx]
				m.activeGroupIdx = tab.GroupIdx
				m.activeSessionIdx = tab.SessionIdx
				m.focusSidebar = false
			}

		case "x": // Jump to 20-terminal Multi-Session Matrix
			m.Result = &SessionBrowserResult{
				Action:  "matrix",
				Session: m.currentSession(),
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

		case "d": // Delete session
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
	header.WriteString(" " + memBadge + "\n")

	// Render Tabs Strip
	if len(m.tabs) > 0 {
		var tb strings.Builder
		tb.WriteString(m.styles.Footer.Render("Tabs: "))
		for i, tab := range m.tabs {
			label := fmt.Sprintf("[%d] %s (%s) %s", tab.Index, tab.ProjectName, tab.SessionID, tab.StatusBadge)
			if i == m.activeTabIdx {
				tb.WriteString(m.styles.SelectedAction.Render(" "+label+" ") + " ")
			} else {
				tb.WriteString(m.styles.ActionItem.Render(" "+label+" ") + " ")
			}
		}
		tb.WriteString(m.styles.Footer.Render("• [+] New (n) • [x] Matrix (20)"))
		header.WriteString(tb.String() + "\n")
	}
	header.WriteString("\n")

	showThreeColumns := totalWidth >= 96

	var sidebarWidth, centerWidth, inspectorWidth int
	if showThreeColumns {
		sidebarWidth = 24
		inspectorWidth = 42
		centerWidth = totalWidth - sidebarWidth - inspectorWidth - 6
		if centerWidth < 34 {
			centerWidth = 34
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

	// 3. Render Right Inspector Panel (Deep Intelligence & Telemetry)
	var content string
	if showThreeColumns {
		var ib strings.Builder
		ib.WriteString(m.styles.Subtitle.Render("🔍 LIVE CLAUDE TELEMETRY") + "\n\n")

		curSess := m.currentSession()
		if curSess == nil {
			ib.WriteString("Select a session to inspect\nreal-time execution state.\n")
		} else {
			// 1. Model, Turns & Token Efficiency
			modelName := curSess.Model
			if modelName == "" {
				modelName = "claude-3-5-sonnet"
			}
			turnStr := fmt.Sprintf("Turn #%d", curSess.TurnCount)
			if curSess.TurnCount == 0 {
				turnStr = "Active"
			}
			tokenStr := formatTokenCount(curSess.TotalTokens)

			ib.WriteString(m.styles.Footer.Render("🧠 MODEL & TELEMETRY:") + "\n")
			ib.WriteString(fmt.Sprintf(" %s • %s\n", m.styles.ReadyBadge.Render(modelName), turnStr))
			if curSess.TotalTokens > 0 {
				ib.WriteString(fmt.Sprintf(" %s\n", tokenStr))
			}
			ib.WriteString("\n")

			// 2. Inner Monologue / Thinking
			if curSess.Thinking != "" {
				ib.WriteString(m.styles.Footer.Render("💭 INNER MONOLOGUE / THINKING:") + "\n")
				th := curSess.Thinking
				if len(th) > 180 {
					th = th[:177] + "..."
				}
				ib.WriteString(fmt.Sprintf(" “%s”\n\n", th))
			}

			// 3. Current Work / Where Claude is right now
			ib.WriteString(m.styles.Footer.Render("⚡ CURRENT WORK:") + "\n")
			act := curSess.LastActivity
			if act == "" {
				act = "Idle / Awaiting instructions"
			}
			if len(act) > inspectorWidth-4 {
				act = act[:inspectorWidth-7] + "..."
			}
			ib.WriteString(" " + act + "\n\n")

			// 4. Execution Journey (Breadcrumbs Timeline)
			if len(curSess.Timeline) > 0 {
				ib.WriteString(m.styles.Footer.Render("🛠 EXECUTION TRAIL (RECENT):") + "\n")
				for _, step := range curSess.Timeline {
					dispStep := step
					if len(dispStep) > inspectorWidth-4 {
						dispStep = dispStep[:inspectorWidth-7] + "..."
					}
					ib.WriteString(fmt.Sprintf("  %s\n", dispStep))
				}
				ib.WriteString("\n")
			}

			// 5. Files Touched
			if len(curSess.FilesTouched) > 0 {
				ib.WriteString(m.styles.Footer.Render(fmt.Sprintf("📁 FILES TOUCHED (%d):", len(curSess.FilesTouched))) + "\n")
				for _, f := range curSess.FilesTouched {
					dispFile := f
					if len(dispFile) > inspectorWidth-4 {
						dispFile = dispFile[:inspectorWidth-7] + "..."
					}
					ib.WriteString(fmt.Sprintf("  %s\n", dispFile))
				}
				ib.WriteString("\n")
			}

			// 6. Web Documentation & Sources
			if len(curSess.WebSources) > 0 {
				ib.WriteString(m.styles.Footer.Render("🔗 WEB SOURCES VISITED:") + "\n")
				for _, src := range curSess.WebSources {
					dispSrc := src
					dispSrc = strings.TrimPrefix(dispSrc, "https://")
					dispSrc = strings.TrimPrefix(dispSrc, "http://")
					if len(dispSrc) > inspectorWidth-4 {
						dispSrc = dispSrc[:inspectorWidth-7] + "..."
					}
					ib.WriteString(fmt.Sprintf("  • %s\n", dispSrc))
				}
				ib.WriteString("\n")
			}

			// 7. Git & Workspace Context
			ib.WriteString(m.styles.Footer.Render("🌿 CONTEXT:") + "\n")
			branch := curSess.GitBranch
			if branch == "" {
				branch = "main"
			}
			ib.WriteString(fmt.Sprintf("  Branch: %s\n", branch))
			if curSess.ProjectCWD != "" {
				cwd := curSess.ProjectCWD
				if len(cwd) > inspectorWidth-10 {
					cwd = "..." + cwd[len(cwd)-(inspectorWidth-13):]
				}
				ib.WriteString(fmt.Sprintf("  Path:   %s\n", cwd))
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
		footerText = "Tab: Pane • 1-9/[]: Tabs • Enter: Resume • n: New Tab • s: Switch Profile • x: Matrix • q: Back"
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

func formatTokenCount(n int64) string {
	if n >= 1_000_000 {
		return fmt.Sprintf("%.1fM cached", float64(n)/1_000_000)
	}
	if n >= 1_000 {
		return fmt.Sprintf("%.1fk cached", float64(n)/1_000)
	}
	return fmt.Sprintf("%d tokens", n)
}
