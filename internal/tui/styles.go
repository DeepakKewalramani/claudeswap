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
	InactiveItem   lipgloss.Style
	ReadyBadge     lipgloss.Style
	WarnBadge      lipgloss.Style
	LoggedOutBadge lipgloss.Style
	UnknownBadge   lipgloss.Style
	Footer         lipgloss.Style
	Divider        lipgloss.Style
	ActionItem     lipgloss.Style
	SelectedAction lipgloss.Style
	ActiveBox      lipgloss.Style
	InactiveBox    lipgloss.Style
	InspectorBox   lipgloss.Style
	TabActive      lipgloss.Style
	TabInactive    lipgloss.Style
	BranchBadge    lipgloss.Style
	HashBadge      lipgloss.Style
	TextMuted      lipgloss.Style
	GadgetHeader   lipgloss.Style
	BarFilled      lipgloss.Style
	BarWarn        lipgloss.Style
	BarDanger      lipgloss.Style
	BarEmpty       lipgloss.Style
}

// DefaultStyles returns modern, polished styles respecting NO_COLOR.
func DefaultStyles() UIStyles {
	noColor := os.Getenv("NO_COLOR") != ""

	if noColor {
		return UIStyles{
			Box:            lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).Padding(0, 1),
			Title:          lipgloss.NewStyle().Bold(true),
			Subtitle:       lipgloss.NewStyle(),
			ActiveItem:     lipgloss.NewStyle().Bold(true),
			InactiveItem:   lipgloss.NewStyle(),
			ReadyBadge:     lipgloss.NewStyle(),
			WarnBadge:      lipgloss.NewStyle(),
			LoggedOutBadge: lipgloss.NewStyle(),
			UnknownBadge:   lipgloss.NewStyle(),
			Footer:         lipgloss.NewStyle(),
			Divider:        lipgloss.NewStyle(),
			ActionItem:     lipgloss.NewStyle(),
			SelectedAction: lipgloss.NewStyle().Bold(true),
			ActiveBox:      lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).Padding(0, 1),
			InactiveBox:    lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).Padding(0, 1),
			InspectorBox:   lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).Padding(0, 1),
			TabActive:      lipgloss.NewStyle().Bold(true),
			TabInactive:    lipgloss.NewStyle(),
			BranchBadge:    lipgloss.NewStyle(),
			HashBadge:      lipgloss.NewStyle(),
			TextMuted:      lipgloss.NewStyle(),
			GadgetHeader:   lipgloss.NewStyle().Bold(true),
			BarFilled:      lipgloss.NewStyle(),
			BarWarn:        lipgloss.NewStyle(),
			BarDanger:      lipgloss.NewStyle(),
			BarEmpty:       lipgloss.NewStyle(),
		}
	}

	primaryColor := lipgloss.Color("#7D56F4")
	accentColor := lipgloss.Color("#04B575")
	warnColor := lipgloss.Color("#FFAE00")
	loggedOutColor := lipgloss.Color("#E06C75")
	mutedColor := lipgloss.Color("#888888")
	subtleBorder := lipgloss.Color("#3B4252")
	activeBorder := lipgloss.Color("#8172F8")
	inspectorBorder := lipgloss.Color("#4C566A")

	return UIStyles{
		Box: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(subtleBorder).
			Padding(0, 1),
		ActiveBox: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(activeBorder).
			Padding(0, 1),
		InactiveBox: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(subtleBorder).
			Padding(0, 1),
		InspectorBox: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(inspectorBorder).
			Padding(0, 1),
		TabActive: lipgloss.NewStyle().
			Background(primaryColor).
			Foreground(lipgloss.Color("#FFFFFF")).
			Bold(true).
			Padding(0, 1),
		TabInactive: lipgloss.NewStyle().
			Background(lipgloss.Color("#2E3440")).
			Foreground(lipgloss.Color("#D8DEE9")).
			Padding(0, 1),
		BranchBadge: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#88C0D0")),
		HashBadge: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#EBCB8B")),
		TextMuted: lipgloss.NewStyle().
			Foreground(mutedColor),
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
		LoggedOutBadge: lipgloss.NewStyle().
			Bold(true).
			Foreground(loggedOutColor),
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
		GadgetHeader: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#88C0D0")),
		BarFilled: lipgloss.NewStyle().
			Foreground(accentColor),
		BarWarn: lipgloss.NewStyle().
			Foreground(warnColor),
		BarDanger: lipgloss.NewStyle().
			Foreground(loggedOutColor),
		BarEmpty: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#4C566A")),
	}
}
