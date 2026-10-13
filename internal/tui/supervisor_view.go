package tui

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/claudeswap/claudeswap/internal/supervisor"
)

// SupervisorResult is returned when exiting the supervisor matrix view.
type SupervisorResult struct {
	Action  string // "attach", "new", "kill", "back"
	Session *supervisor.ManagedSession
}

type tickMsg time.Time

// SupervisorModel provides a live terminal matrix for up to 20 background Claude sessions.
type SupervisorModel struct {
	sessions      []supervisor.ManagedSession
	cursor        int
	width         int
	height        int
	styles        UIStyles
	Quitting      bool
	ConfirmKill   bool
	Result        *SupervisorResult
	logsGetter    func(sessionID string, tailLines int) []string
	refreshGetter func() []supervisor.ManagedSession
}

// NewSupervisorModel creates the live supervisor dashboard.
func NewSupervisorModel(
	sessions []supervisor.ManagedSession,
	logsGetter func(sessionID string, tailLines int) []string,
	refreshGetter func() []supervisor.ManagedSession,
) SupervisorModel {
	return SupervisorModel{
		sessions:      sessions,
		styles:        DefaultStyles(),
		logsGetter:    logsGetter,
		refreshGetter: refreshGetter,
	}
}

func (m SupervisorModel) Init() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func (m SupervisorModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tickMsg:
		if m.refreshGetter != nil {
			m.sessions = m.refreshGetter()
		}
		return m, tea.Tick(time.Second, func(t time.Time) tea.Msg {
			return tickMsg(t)
		})

	case tea.KeyMsg:
		if m.ConfirmKill {
			switch strings.ToLower(msg.String()) {
			case "y":
				m.ConfirmKill = false
				if cur := m.currentSession(); cur != nil {
					m.Result = &SupervisorResult{Action: "kill", Session: cur}
					return m, tea.Quit
				}
			case "n", "esc", "q":
				m.ConfirmKill = false
			}
			return m, nil
		}

		switch msg.String() {
		case "ctrl+c", "q", "esc":
			m.Result = &SupervisorResult{Action: "back"}
			m.Quitting = true
			return m, tea.Quit

		case "up":
			if m.cursor > 0 {
				m.cursor--
			}

		case "down":
			if m.cursor < len(m.sessions)-1 {
				m.cursor++
			}

		case "enter", "a": // Attach to session
			cur := m.currentSession()
			if cur != nil {
				m.Result = &SupervisorResult{Action: "attach", Session: cur}
				return m, tea.Quit
			}

		case "n": // Spawn new background session
			m.Result = &SupervisorResult{Action: "new"}
			return m, tea.Quit

		case "k", "x": // Kill session
			if m.currentSession() != nil {
				m.ConfirmKill = true
			}

		case "r": // Manual refresh
			if m.refreshGetter != nil {
				m.sessions = m.refreshGetter()
			}
		}
	}

	return m, nil
}

func (m SupervisorModel) currentSession() *supervisor.ManagedSession {
	if len(m.sessions) == 0 || m.cursor >= len(m.sessions) {
		return nil
	}
	return &m.sessions[m.cursor]
}

func (m SupervisorModel) View() string {
	if m.Quitting {
		return ""
	}

	totalWidth := m.width
	if totalWidth <= 0 {
		totalWidth = 96
	}
	if totalWidth < 70 {
		totalWidth = 70
	}

	var b strings.Builder

	// 1. Header
	headerTitle := " ClaudeSwap • Multi-Session Mission Control [Matrix] "
	activeCount := 0
	for _, s := range m.sessions {
		if s.Status == supervisor.StateRunning || s.Status == supervisor.StateWaitingInput {
			activeCount++
		}
	}
	slotBadge := m.styles.ReadyBadge.Render(fmt.Sprintf(" Active Slots: %d / 20 ", activeCount))
	refreshBadge := m.styles.LoggedOutBadge.Render(" Live Refresher: 1s ")

	b.WriteString(m.styles.Title.Render(headerTitle) + " " + slotBadge + " " + refreshBadge + "\n\n")

	actWidth := totalWidth - 72
	if actWidth < 20 {
		actWidth = 20
	}
	hdr := fmt.Sprintf("  %-3s %-16s %-14s %-12s %-16s %-*s %s",
		"#", "TASK NAME", "PROJECT", "PROFILE", "STATUS", actWidth, "CURRENT ACTIVITY", "UPTIME")
	if len(hdr) > totalWidth-2 {
		hdr = hdr[:totalWidth-2]
	}
	b.WriteString(m.styles.Footer.Render(hdr) + "\n")
	b.WriteString(m.styles.Divider.Render(strings.Repeat("─", totalWidth-2)) + "\n")

	// 3. Session Rows
	if len(m.sessions) == 0 {
		b.WriteString("\n  No background Claude sessions currently registered.\n")
		b.WriteString("  Press [n] to spawn a new background session in any project folder.\n\n")
	} else {
		for i, s := range m.sessions {
			isCursor := m.cursor == i
			prefix := "  "
			if isCursor {
				prefix = "❯ "
			}

			numStr := fmt.Sprintf("%02d", i+1)

			task := s.TaskName
			if len(task) > 15 {
				task = task[:13] + ".."
			}

			proj := s.ProjectName
			if len(proj) > 13 {
				proj = proj[:11] + ".."
			}

			prof := s.ProfileName
			if len(prof) > 11 {
				prof = prof[:9] + ".."
			}

			statusStr := renderStatusBadge(m.styles, s.Status)

			act := s.LastActivity
			if act == "" {
				act = "Idle"
			}
			if len(act) > actWidth-2 {
				act = act[:actWidth-5] + "..."
			}

			uptime := formatUptime(s.StartedAt)

			row := fmt.Sprintf("%s%-3s %-16s %-14s %-12s %-16s %-*s %s",
				prefix, numStr, task, proj, prof, statusStr, actWidth, act, uptime)
			if len(row) > totalWidth-2 {
				row = row[:totalWidth-2]
			}

			if isCursor {
				b.WriteString(m.styles.ActiveItem.Render(row) + "\n")
			} else {
				b.WriteString(m.styles.InactiveItem.Render(row) + "\n")
			}
		}
	}

	// 4. Live Log Drawer for Selected Session
	b.WriteString("\n" + m.styles.Divider.Render(strings.Repeat("─", totalWidth-2)) + "\n")
	cur := m.currentSession()
	if cur != nil && m.logsGetter != nil {
		drawerHdr := fmt.Sprintf("Live Output: %s (PID %d) • %s", cur.TaskName, cur.PID, cur.ProjectDir)
		b.WriteString(m.styles.Subtitle.Render(drawerHdr) + "\n")

		logs := m.logsGetter(cur.ID, 4)
		if len(logs) == 0 {
			b.WriteString(m.styles.Footer.Render("  (No logs emitted yet)") + "\n")
		} else {
			for _, l := range logs {
				if len(l) > totalWidth-4 {
					l = l[:totalWidth-7] + "..."
				}
				b.WriteString("  " + l + "\n")
			}
		}
	} else {
		b.WriteString(m.styles.Footer.Render("Select a session above to inspect live output streams.") + "\n")
	}

	// 5. Hotkeys Footer
	var footerText string
	if m.ConfirmKill {
		cur := m.currentSession()
		footerText = fmt.Sprintf("⚠ Terminate session %s (PID %d)? Press [y] to confirm, [n] to cancel", cur.TaskName, cur.PID)
	} else {
		footerText = "↑↓ Move • Enter/a: Attach • n: New Background Session • k: Terminate • r: Refresh • q: Detach"
	}
	b.WriteString("\n" + m.styles.Divider.Render(strings.Repeat("─", totalWidth-2)) + "\n")
	b.WriteString(m.styles.Footer.Render(centerText(footerText, totalWidth)))

	panelHeight := m.height - 4
	if m.height <= 0 || panelHeight < 16 {
		panelHeight = 22
	}
	rendered := m.styles.ActiveBox.Width(totalWidth).Height(panelHeight).Render(b.String())
	return rendered + "\n"
}

func renderStatusBadge(styles UIStyles, status supervisor.SessionState) string {
	switch status {
	case supervisor.StateRunning:
		return styles.ReadyBadge.Render("🟢 RUNNING")
	case supervisor.StateWaitingInput:
		return styles.WarnBadge.Render("🟡 WAITING INPUT")
	case supervisor.StateCompleted:
		return styles.Footer.Render("⚪ COMPLETED")
	case supervisor.StateFailed:
		return styles.LoggedOutBadge.Render("🔴 FAILED")
	case supervisor.StateStopped:
		return styles.LoggedOutBadge.Render("⏹ STOPPED")
	default:
		return string(status)
	}
}

func formatUptime(start time.Time) string {
	diff := time.Since(start)
	if diff < time.Minute {
		return fmt.Sprintf("%ds", int(diff.Seconds()))
	}
	if diff < time.Hour {
		return fmt.Sprintf("%dm %ds", int(diff.Minutes()), int(diff.Seconds())%60)
	}
	return fmt.Sprintf("%dh %dm", int(diff.Hours()), int(diff.Minutes())%60)
}
