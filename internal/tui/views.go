package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/claudeswap/claudeswap/internal/profile"
)

func (m Model) View() string {
	if m.Quitting {
		return ""
	}

	contentWidth := m.width - 6
	if contentWidth < 46 {
		contentWidth = 46
	} else if contentWidth > 72 {
		contentWidth = 72
	}

	var b strings.Builder

	// Header section
	title := centerText("ClaudeSwap", contentWidth)
	subtitle := centerText("Claude Profile Manager", contentWidth)

	b.WriteString(m.styles.Title.Render(title) + "\n")
	b.WriteString(m.styles.Subtitle.Render(subtitle) + "\n\n")

	// Profiles section
	b.WriteString(m.styles.Subtitle.Render("Profiles") + "\n\n")

	if len(m.profiles) == 0 {
		b.WriteString("  No profiles configured yet.\n\n")
	} else {
		nameColWidth := contentWidth - 18
		if nameColWidth < 20 {
			nameColWidth = 20
		}

		for i, p := range m.profiles {
			isCursor := !m.inActions && m.cursor == i

			prefix := "  "
			if isCursor {
				prefix = "❯ "
			}

			nameStr := p.Profile.Name
			maxLen := nameColWidth - 3
			if len(nameStr) > maxLen {
				nameStr = nameStr[:maxLen-3] + "..."
			}

			statusBadge := formatStatusBadge(m.styles, p.Status)

			leftPart := fmt.Sprintf("%s%-*s", prefix, nameColWidth, nameStr)
			row := leftPart + statusBadge

			if isCursor {
				b.WriteString(m.styles.ActiveItem.Render(row) + "\n")
			} else {
				b.WriteString(m.styles.InactiveItem.Render(row) + "\n")
			}
		}
		b.WriteString("\n")
	}

	// Divider
	b.WriteString(m.styles.Divider.Render(strings.Repeat("─", contentWidth)) + "\n\n")

	// Actions section
	for i, action := range m.actions {
		isCursor := m.inActions && m.actionCursor == i
		prefix := "  "
		if isCursor {
			prefix = "❯ "
		}

		var line string
		switch action {
		case ActionToggleMemory:
			var stateBadge string
			if m.sharedMemory {
				stateBadge = m.styles.ReadyBadge.Render("[ON  • Shared]")
			} else {
				stateBadge = m.styles.LoggedOutBadge.Render("[OFF • Isolated]")
			}
			line = fmt.Sprintf("%sMemory State:  %s  (m)", prefix, stateBadge)
		case ActionSessionManager:
			line = prefix + "Session Explorer (Sidebar) (w)"
		case ActionAddProfile:
			line = prefix + "Add Profile         (a)"
		case ActionHandoff:
			line = prefix + "Switch / Handoff    (h)"
		case ActionRemove:
			line = prefix + "Delete Profile      (r)"
		case ActionSettings:
			line = prefix + "Settings            (s)"
		case ActionDoctor:
			line = prefix + "Diagnostics         (d)"
		case ActionExit:
			line = prefix + "Exit                (q)"
		default:
			line = prefix + string(action)
		}

		if isCursor {
			b.WriteString(m.styles.SelectedAction.Render(line) + "\n")
		} else {
			b.WriteString(m.styles.ActionItem.Render(line) + "\n")
		}
	}

	b.WriteString("\n" + m.styles.Divider.Render(strings.Repeat("─", contentWidth)) + "\n")
	footer := centerText("↑↓ Move • Enter Select • m Memory • w Sessions • a Add • h Handoff • q Quit", contentWidth)
	b.WriteString(m.styles.Footer.Render(footer))

	renderedBox := m.styles.Box.Render(b.String())
	if m.width > 0 && m.height > 0 {
		return lipgloss.Place(
			m.width,
			m.height,
			lipgloss.Center,
			lipgloss.Center,
			renderedBox,
		)
	}

	return renderedBox + "\n"
}

func formatStatusBadge(styles UIStyles, status profile.Status) string {
	switch status {
	case profile.StatusReady:
		return styles.ReadyBadge.Render("✓ Ready")
	case profile.StatusSessionOut:
		return styles.WarnBadge.Render("⚠ Session out")
	case profile.StatusLoggedOut, profile.StatusLoginRequired:
		return styles.LoggedOutBadge.Render("○ Logged out")
	default:
		return styles.UnknownBadge.Render("? Unknown")
	}
}

func centerText(s string, width int) string {
	if len(s) >= width {
		return s
	}
	leftPad := (width - len(s)) / 2
	rightPad := width - len(s) - leftPad
	return strings.Repeat(" ", leftPad) + s + strings.Repeat(" ", rightPad)
}
