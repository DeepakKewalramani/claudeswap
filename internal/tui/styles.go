package tui

import (
	"os"

	"github.com/charmbracelet/lipgloss"
)

// UIStyles encapsulates all styling for the ClaudeSwap TUI.
type UIStyles struct {
	Box           lipgloss.Style
	Title         lipgloss.Style
	Subtitle      lipgloss.Style
	ActiveItem    lipgloss.Style
	InactiveItem  lipgloss.Style
	ReadyBadge    lipgloss.Style
	WarnBadge     lipgloss.Style
	UnknownBadge  lipgloss.Style
	Footer        lipgloss.Style
	Divider       lipgloss.Style
	ActionItem    lipgloss.Style
	SelectedAction lipgloss.Style
}

// DefaultStyles returns modern, polished styles respecting NO_COLOR.
func DefaultStyles() UIStyles {
	noColor := os.Getenv("NO_COLOR") != ""

	if noColor {
		return UIStyles{
			Box:           lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).Padding(0, 1),
			Title:         lipgloss.NewStyle().Bold(true),
			Subtitle:      lipgloss.NewStyle(),
			ActiveItem:    lipgloss.NewStyle().Bold(true),
			InactiveItem:  lipgloss.NewStyle(),
			ReadyBadge:    lipgloss.NewStyle(),
			WarnBadge:     lipgloss.NewStyle(),
			UnknownBadge:  lipgloss.NewStyle(),
			Footer:        lipgloss.NewStyle(),
			Divider:       lipgloss.NewStyle(),
			ActionItem:    lipgloss.NewStyle(),
			SelectedAction: lipgloss.NewStyle().Bold(true),
		}
	}

	primaryColor := lipgloss.Color("#7D56F4")
	accentColor := lipgloss.Color("#04B575")
	warnColor := lipgloss.Color("#FFAE00")
	mutedColor := lipgloss.Color("#888888")
	subtleBorder := lipgloss.Color("#3C3C3C")

	return UIStyles{
		Box: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(subtleBorder).
			Padding(1, 2),
		Title: lipgloss.NewStyle().
			Bold(true).
			Foreground(primaryColor),
		Subtitle: lipgloss.NewStyle().
			Foreground(mutedColor),
		ActiveItem: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FFFFFF")),
		InactiveItem: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#B0B0B0")),
		ReadyBadge: lipgloss.NewStyle().
			Bold(true).
			Foreground(accentColor),
		WarnBadge: lipgloss.NewStyle().
			Bold(true).
			Foreground(warnColor),
		UnknownBadge: lipgloss.NewStyle().
			Foreground(mutedColor),
		Footer: lipgloss.NewStyle().
			Foreground(mutedColor),
		Divider: lipgloss.NewStyle().
			Foreground(subtleBorder),
		ActionItem: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#CCCCCC")),
		SelectedAction: lipgloss.NewStyle().
			Bold(true).
			Foreground(primaryColor),
	}
}
