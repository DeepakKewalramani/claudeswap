package tui

import (
	"fmt"
	"strings"

	"github.com/claudeswap/claudeswap/internal/profile"
)

func (m Model) View() string {
	if m.Quitting {
		return ""
	}

	const contentWidth = 50

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
		for i, p := range m.profiles {
			isCursor := !m.inActions && m.cursor == i

			prefix := "  "
			if isCursor {
				prefix = "❯ "
			}

			nameStr := p.Profile.Name
			if len(nameStr) > 20 {
				nameStr = nameStr[:17] + "..."
			}

			statusBadge := formatStatusBadge(m.styles, p.Status)

			// Format with fixed column spacing
			leftPart := fmt.Sprintf("%s%-22s", prefix, nameStr)
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

		line := prefix + string(action)
		if isCursor {
			b.WriteString(m.styles.SelectedAction.Render(line) + "\n")
		} else {
			b.WriteString(m.styles.ActionItem.Render(line) + "\n")
		}
	}

	b.WriteString("\n" + m.styles.Divider.Render(strings.Repeat("─", contentWidth)) + "\n")
	footer := centerText("↑↓ Navigate   Enter Select   a Add   q Quit", contentWidth)
	b.WriteString(m.styles.Footer.Render(footer))

	return m.styles.Box.Render(b.String()) + "\n"
}

func formatStatusBadge(styles UIStyles, status profile.Status) string {
	switch status {
	case profile.StatusReady:
		return styles.ReadyBadge.Render("✓ Ready")
	case profile.StatusLoginRequired:
		return styles.WarnBadge.Render("⚠ Login required")
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
