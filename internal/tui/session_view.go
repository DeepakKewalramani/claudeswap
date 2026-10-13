package tui

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
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
	showInspector    bool // true = display right telemetry inspector
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
		showInspector: true,
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

		case "i", "ctrl+s": // Toggle inspector sidebar
			m.showInspector = !m.showInspector

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

// cleanOneLine removes all newlines and collapses whitespace, truncating to maxLen.
func cleanOneLine(s string, maxLen int) string {
	fields := strings.Fields(s)
	clean := strings.Join(fields, " ")
	if maxLen <= 0 {
		return ""
	}
	runes := []rune(clean)
	if len(runes) > maxLen {
		if maxLen > 3 {
			return string(runes[:maxLen-3]) + "..."
		}
		return string(runes[:maxLen])
	}
	return clean
}

func (m SessionBrowserModel) renderMicroBar(ratio float64, width int) string {
	if width < 8 {
		width = 8
	}
	if ratio > 1.0 {
		ratio = 1.0
	}
	if ratio < 0 {
		ratio = 0
	}
	filledLen := int(ratio * float64(width))
	if filledLen > width {
		filledLen = width
	}
	emptyLen := width - filledLen

	filledStr := strings.Repeat("▋", filledLen)
	emptyStr := strings.Repeat("·", emptyLen)

	var filledStyled string
	if ratio >= 0.85 {
		filledStyled = m.styles.BarDanger.Render(filledStr)
	} else if ratio >= 0.70 {
		filledStyled = m.styles.BarWarn.Render(filledStr)
	} else {
		filledStyled = m.styles.BarFilled.Render(filledStr)
	}
	return filledStyled + m.styles.BarEmpty.Render(emptyStr)
}

func (m SessionBrowserModel) renderHeader() string {
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
				tb.WriteString(m.styles.TabActive.Render(label) + " ")
			} else {
				tb.WriteString(m.styles.TabInactive.Render(label) + " ")
			}
		}
		tb.WriteString(m.styles.Footer.Render("• [+] New (n) • [x] Matrix (20)"))
		header.WriteString(tb.String() + "\n")
	}

	return header.String()
}

func (m SessionBrowserModel) renderSidebar(width, height int) string {
	innerWidth := width - 4
	if innerWidth < 10 {
		innerWidth = 10
	}
	maxLines := height - 4
	if maxLines < 1 {
		maxLines = 1
	}

	var sb strings.Builder
	title := fmt.Sprintf("📁 Folders (%d)", len(m.groups))
	sb.WriteString(m.styles.Subtitle.Render(cleanOneLine(title, innerWidth)) + "\n\n")

	if len(m.groups) == 0 {
		sb.WriteString(m.styles.TextMuted.Render(cleanOneLine("No projects found.", innerWidth)) + "\n")
	} else {
		startIdx := 0
		if m.activeGroupIdx >= maxLines {
			startIdx = m.activeGroupIdx - maxLines + 1
		}
		endIdx := startIdx + maxLines
		if endIdx > len(m.groups) {
			endIdx = len(m.groups)
			if endIdx-maxLines >= 0 {
				startIdx = endIdx - maxLines
			}
		}

		for i := startIdx; i < endIdx; i++ {
			g := m.groups[i]
			isCursor := m.focusSidebar && m.activeGroupIdx == i
			isSelected := !m.focusSidebar && m.activeGroupIdx == i

			prefix := "  "
			if isCursor {
				prefix = "❯ "
			} else if isSelected {
				prefix = "● "
			}

			countStr := fmt.Sprintf("(%d)", len(g.Sessions))
			availName := innerWidth - 2 - len(countStr) - 1
			if availName < 3 {
				availName = 3
			}
			name := cleanOneLine(g.Name, availName)
			line := fmt.Sprintf("%s%-*s %s", prefix, availName, name, countStr)
			if len([]rune(line)) > innerWidth {
				line = string([]rune(line)[:innerWidth])
			}

			if isCursor {
				sb.WriteString(m.styles.ActiveItem.Render(line) + "\n")
			} else if isSelected {
				sb.WriteString(m.styles.SelectedAction.Render(line) + "\n")
			} else {
				sb.WriteString(m.styles.InactiveItem.Render(line) + "\n")
			}
		}
	}

	style := m.styles.InactiveBox
	if m.focusSidebar {
		style = m.styles.ActiveBox
	}
	return style.Width(width).Height(height).Render(sb.String())
}

func (m SessionBrowserModel) renderCenterPanel(width, height int, showThreeColumns bool) string {
	innerWidth := width - 4
	if innerWidth < 20 {
		innerWidth = 20
	}
	var cb strings.Builder
	curGroup := m.currentGroup()

	if curGroup == nil || len(curGroup.Sessions) == 0 {
		cb.WriteString(m.styles.Subtitle.Render("No sessions in this project") + "\n\n")
		cb.WriteString(m.styles.TextMuted.Render("Press [n] to start a new Claude session\nin this folder.") + "\n")
	} else {
		groupHeader := fmt.Sprintf("Sessions in %s", curGroup.Name)
		cb.WriteString(m.styles.Subtitle.Render(cleanOneLine(groupHeader, innerWidth)) + "\n\n")

		// Columns: SESSION (8), BRANCH (10), ACTIVE (8), TOPIC (remaining)
		availTopic := innerWidth - 31
		if availTopic < 6 {
			availTopic = 6
		}

		hdr := fmt.Sprintf("  %-8s %-10s %-8s %s", "SESSION", "BRANCH", "ACTIVE", "TOPIC")
		if len([]rune(hdr)) > innerWidth {
			hdr = string([]rune(hdr)[:innerWidth])
		}
		cb.WriteString(m.styles.Footer.Render(hdr) + "\n")
		cb.WriteString(m.styles.Divider.Render(strings.Repeat("─", innerWidth)) + "\n")

		maxRows := height - 6
		if maxRows < 1 {
			maxRows = 1
		}

		startIdx := 0
		if m.activeSessionIdx >= maxRows {
			startIdx = m.activeSessionIdx - maxRows + 1
		}
		endIdx := startIdx + maxRows
		if endIdx > len(curGroup.Sessions) {
			endIdx = len(curGroup.Sessions)
			if endIdx-maxRows >= 0 {
				startIdx = endIdx - maxRows
			}
		}

		for i := startIdx; i < endIdx; i++ {
			s := curGroup.Sessions[i]
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
			branch = cleanOneLine(branch, 10)

			relTime := cleanOneLine(formatRelativeTime(s.UpdatedAt), 8)
			topic := cleanOneLine(s.Title, availTopic)

			row := fmt.Sprintf("%s%-8s %-10s %-8s %s", prefix, shortID, branch, relTime, topic)
			if len([]rune(row)) > innerWidth {
				row = string([]rune(row)[:innerWidth])
			}

			if isCursor {
				cb.WriteString(m.styles.ActiveItem.Render(row) + "\n")
			} else {
				cb.WriteString(m.styles.InactiveItem.Render(row) + "\n")
			}
		}

		// When width is narrow, show compact activity at bottom if extra space remains
		if !showThreeColumns {
			curSess := m.currentSession()
			if curSess != nil && (curSess.LastActivity != "" || len(curSess.WebSources) > 0) {
				cb.WriteString("\n" + m.styles.Divider.Render(strings.Repeat("─", innerWidth)) + "\n")
				if curSess.LastActivity != "" {
					act := cleanOneLine("⚡ "+curSess.LastActivity, innerWidth)
					cb.WriteString(m.styles.TextMuted.Render(act) + "\n")
				}
				if len(curSess.WebSources) > 0 {
					src := cleanOneLine("🔗 "+curSess.WebSources[0], innerWidth)
					cb.WriteString(m.styles.TextMuted.Render(src) + "\n")
				}
			}
		}
	}

	style := m.styles.InactiveBox
	if !m.focusSidebar {
		style = m.styles.ActiveBox
	}
	return style.Width(width).Height(height).Render(cb.String())
}

func (m SessionBrowserModel) renderInspector(width, height int) string {
	innerWidth := width - 4
	if innerWidth < 20 {
		innerWidth = 20
	}
	var ib strings.Builder

	curSess := m.currentSession()
	if curSess == nil {
		ib.WriteString(m.styles.GadgetHeader.Render("telemetry") + "\n\n")
		ib.WriteString(m.styles.TextMuted.Render("  no session selected") + "\n")
		return m.styles.InspectorBox.Width(width).Height(height).Render(ib.String())
	}

	// 1. Header: "telemetry" on left, status on right
	titleStr := "telemetry"
	statusDiff := time.Since(curSess.UpdatedAt)
	var statusPill string
	if statusDiff < 10*time.Minute {
		statusPill = m.styles.ReadyBadge.Render("● active ") + m.styles.TextMuted.Render(formatRelativeTime(curSess.UpdatedAt))
	} else {
		statusPill = m.styles.TextMuted.Render("○ idle " + formatRelativeTime(curSess.UpdatedAt))
	}

	availPad := innerWidth - len(titleStr) - len([]rune(cleanOneLine(statusPill, innerWidth)))
	if availPad < 1 {
		availPad = 1
	}
	hdr := m.styles.GadgetHeader.Render(titleStr) + strings.Repeat(" ", availPad) + statusPill
	ib.WriteString(hdr + "\n")
	ib.WriteString(m.styles.Divider.Render(strings.Repeat("─", innerWidth)) + "\n\n")

	// 2. context section
	ib.WriteString(m.styles.GadgetHeader.Render("context") + "\n")
	modelName := curSess.Model
	if modelName == "" {
		modelName = "claude-3-5-sonnet"
	}
	turnStr := fmt.Sprintf("turn #%d", curSess.TurnCount)
	if curSess.TurnCount == 0 {
		turnStr = "active"
	}
	availModel := innerWidth - len(turnStr) - 4
	if availModel < 10 {
		availModel = 10
	}
	modelLine := fmt.Sprintf("  %-*s %s", availModel, cleanOneLine(modelName, availModel), m.styles.TextMuted.Render(turnStr))
	if len([]rune(modelLine)) > innerWidth {
		modelLine = string([]rune(modelLine)[:innerWidth])
	}
	ib.WriteString(modelLine + "\n")

	tokenMax := int64(200_000)
	if curSess.TotalTokens > tokenMax {
		tokenMax = curSess.TotalTokens
	}
	pct := float64(curSess.TotalTokens) / float64(tokenMax)
	pctStr := fmt.Sprintf("%.0f%%", pct*100)
	tokStr := formatTokenCount(curSess.TotalTokens) + " / " + formatTokenCount(tokenMax)
	availTok := innerWidth - len(pctStr) - 4
	if availTok < 10 {
		availTok = 10
	}
	tokLine := fmt.Sprintf("  %-*s %s", availTok, tokStr, pctStr)
	if len([]rune(tokLine)) > innerWidth {
		tokLine = string([]rune(tokLine)[:innerWidth])
	}
	ib.WriteString(tokLine + "\n")

	barWidth := innerWidth - 4
	if barWidth > 32 {
		barWidth = 32
	}
	ib.WriteString("  " + m.renderMicroBar(pct, barWidth) + "\n\n")

	// 3. activity section
	ib.WriteString(m.styles.GadgetHeader.Render("activity") + "\n")
	act := curSess.LastActivity
	if act == "" {
		act = "idle (awaiting instructions)"
	}
	act = cleanOneLine(act, innerWidth-4)
	ib.WriteString(fmt.Sprintf("  %s\n\n", act))

	// 4. thinking section (if present)
	if curSess.Thinking != "" {
		ib.WriteString(m.styles.GadgetHeader.Render("thinking") + "\n")
		th := cleanOneLine(curSess.Thinking, innerWidth-6)
		ib.WriteString(m.styles.TextMuted.Render(fmt.Sprintf("  “%s”", th)) + "\n\n")
	}

	// 5. trail section (if present)
	if len(curSess.Timeline) > 0 {
		ib.WriteString(m.styles.GadgetHeader.Render("trail") + "\n")
		count := 0
		for _, step := range curSess.Timeline {
			if count >= 3 {
				break
			}
			dispStep := cleanOneLine(step, innerWidth-6)
			if !strings.HasPrefix(dispStep, "✓") && !strings.HasPrefix(dispStep, "💻") {
				dispStep = "✓ " + dispStep
			}
			ib.WriteString(fmt.Sprintf("  %s\n", dispStep))
			count++
		}
		ib.WriteString("\n")
	}

	// 6. files section (if present)
	if len(curSess.FilesTouched) > 0 {
		ib.WriteString(m.styles.GadgetHeader.Render(fmt.Sprintf("files (%d)", len(curSess.FilesTouched))) + "\n")
		count := 0
		for _, f := range curSess.FilesTouched {
			if count >= 3 {
				break
			}
			dispFile := cleanOneLine(f, innerWidth-4)
			ib.WriteString(fmt.Sprintf("  %s\n", dispFile))
			count++
		}
		ib.WriteString("\n")
	}

	// 7. sources section (if present)
	if len(curSess.WebSources) > 0 {
		ib.WriteString(m.styles.GadgetHeader.Render("sources") + "\n")
		count := 0
		for _, src := range curSess.WebSources {
			if count >= 2 {
				break
			}
			dispSrc := strings.TrimPrefix(src, "https://")
			dispSrc = strings.TrimPrefix(dispSrc, "http://")
			dispSrc = cleanOneLine(dispSrc, innerWidth-4)
			ib.WriteString(fmt.Sprintf("  %s\n", dispSrc))
			count++
		}
		ib.WriteString("\n")
	}

	// 8. workspace section
	ib.WriteString(m.styles.GadgetHeader.Render("workspace") + "\n")
	branch := curSess.GitBranch
	if branch == "" {
		branch = "main"
	}
	branch = cleanOneLine(branch, innerWidth-12)
	ib.WriteString(fmt.Sprintf("  branch  %s\n", branch))
	if curSess.ProjectCWD != "" {
		cwd := curSess.ProjectCWD
		availPath := innerWidth - 12
		if len(cwd) > availPath && availPath > 5 {
			cwd = "..." + cwd[len(cwd)-(availPath-3):]
		}
		cwd = cleanOneLine(cwd, innerWidth-12)
		ib.WriteString(fmt.Sprintf("  path    %s\n", cwd))
	}

	return m.styles.InspectorBox.Width(width).Height(height).Render(ib.String())
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

	headerStr := m.renderHeader()
	headerLines := strings.Count(headerStr, "\n") + 1
	footerLines := 2

	panelHeight := m.height - headerLines - footerLines
	if m.height <= 0 || panelHeight < 14 {
		panelHeight = 20
	}

	showThreeColumns := totalWidth >= 92 && m.showInspector
	var sidebarWidth, centerWidth, inspectorWidth int

	if showThreeColumns {
		sidebarWidth = totalWidth * 22 / 100
		if sidebarWidth < 24 {
			sidebarWidth = 24
		} else if sidebarWidth > 34 {
			sidebarWidth = 34
		}

		inspectorWidth = totalWidth * 36 / 100
		if inspectorWidth < 38 {
			inspectorWidth = 38
		} else if inspectorWidth > 50 {
			inspectorWidth = 50
		}

		centerWidth = totalWidth - sidebarWidth - inspectorWidth - 2
		if centerWidth < 32 {
			centerWidth = 32
		}
	} else {
		sidebarWidth = totalWidth * 25 / 100
		if sidebarWidth < 22 {
			sidebarWidth = 22
		} else if sidebarWidth > 32 {
			sidebarWidth = 32
		}
		centerWidth = totalWidth - sidebarWidth - 1
		if centerWidth < 32 {
			centerWidth = 32
		}
	}

	sidebarBox := m.renderSidebar(sidebarWidth, panelHeight)
	centerBox := m.renderCenterPanel(centerWidth, panelHeight, showThreeColumns)

	var content string
	if showThreeColumns {
		inspectorBox := m.renderInspector(inspectorWidth, panelHeight)
		content = lipgloss.JoinHorizontal(lipgloss.Top, sidebarBox, " ", centerBox, " ", inspectorBox)
	} else {
		content = lipgloss.JoinHorizontal(lipgloss.Top, sidebarBox, " ", centerBox)
	}

	// Bottom bar / Hotkeys
	var footerText string
	if m.ConfirmDelete {
		curS := m.currentSession()
		sid := "session"
		if curS != nil && len(curS.ID) >= 8 {
			sid = curS.ID[:8]
		}
		footerText = fmt.Sprintf("⚠ PERMANENTLY DELETE session %s? Press [y] to confirm, [n] to cancel", sid)
	} else {
		footerText = "Tab: Pane • 1-9/[]: Tabs • i: Toggle Inspector • Enter: Resume • n: New • s: Profile • x: Matrix • q: Back"
	}
	footer := m.styles.Footer.Render(centerText(footerText, totalWidth))

	return headerStr + "\n" + content + "\n\n" + footer + "\n"
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
