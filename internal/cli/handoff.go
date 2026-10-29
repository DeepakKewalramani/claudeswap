package cli

import (
	"fmt"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/claudeswap/claudeswap/internal/claude"
	"github.com/claudeswap/claudeswap/internal/profile"
	"github.com/claudeswap/claudeswap/internal/tui"

	"github.com/spf13/cobra"
	"golang.org/x/term"
)

func newHandoffCmd(app *AppContext) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "handoff [target-profile] [extra-args...]",
		Aliases: []string{"switch", "sw"},
		Short:   "Switch active conversation or hand off to another profile when limits are reached",
		Long: `Seamlessly transfer your current conversation context to another profile.
This is designed for when an account reaches its rate or token limits or when you want
to switch accounts: it enables shared context and launches the target profile with Claude
Code's --resume flag, picking up the conversation exactly where you left off with the new
account's quota.`,
		DisableFlagParsing: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 && (args[0] == "--help" || args[0] == "-h" || args[0] == "help") {
				return cmd.Help()
			}

			var p *profile.Profile
			var extraArgs []string

			if len(args) > 0 {
				target := args[0]
				extraArgs = args[1:]
				resolved, err := app.Manager.GetProfile(target)
				if err != nil {
					fmt.Fprintf(os.Stderr, "✗ Profile %q was not found.\n", target)
					printAvailableProfiles(app)
					os.Exit(ExitProfileNotFound)
				}
				p = resolved
			} else {
				// Interactive selection
				selected, proceed := PromptHandoffMenu(app, nil)
				if !proceed || selected == nil {
					return nil
				}
				p = selected
			}

			// Ensure shared context is active
			cfg, _ := app.CfgStore.Load()
			if cfg != nil && !cfg.SharedContext {
				cfg.SharedContext = true
				_ = app.CfgStore.Save(cfg)
			}

			// Ensure shared context directories are synchronized
			_ = claude.SyncSharedContext(app.Paths, p.ConfigDir, true)

			fmt.Printf("✓ Handing off active conversation to profile %q (%s)...\n", p.Name, p.ID)
			fmt.Println("✓ Resuming session under new account quota...")
			fmt.Println()

			// Prepare args with --resume
			resumeArgs := []string{"--resume"}
			resumeArgs = append(resumeArgs, extraArgs...)

			status := app.Manager.GetStatus(p)
			if status == profile.StatusLoginRequired {
				fmt.Printf("⚠ Note: Profile %q may require authentication first.\n\n", p.Name)
			}

			code, err := app.Launcher.Launch(p, resumeArgs, true)
			if err != nil {
				fmt.Fprintf(os.Stderr, "✗ Error launching profile: %v\n", err)
				os.Exit(code)
			}

			os.Exit(code)
			return nil
		},
	}

	return cmd
}

// PromptHandoffMenu displays an interactive prompt asking whether to continue with another profile.
func PromptHandoffMenu(app *AppContext, currentProfile *profile.Profile) (*profile.Profile, bool) {
	if !term.IsTerminal(int(os.Stdin.Fd())) {
		return nil, false
	}

	allProfiles, err := app.Manager.ListProfiles()
	if err != nil || len(allProfiles) == 0 {
		return nil, false
	}

	// Filter other available profiles
	var candidates []profile.Profile
	for _, p := range allProfiles {
		if currentProfile == nil || !strings.EqualFold(p.ID, currentProfile.ID) {
			candidates = append(candidates, p)
		}
	}

	if len(candidates) == 0 {
		return nil, false
	}

	app.Manager.PreloadStatuses(candidates)
	var items []handoffItem
	for _, c := range candidates {
		status := app.Manager.GetStatus(&c)
		var statusBadge string
		switch status {
		case profile.StatusReady:
			statusBadge = "✓ Ready"
		case profile.StatusSessionOut:
			statusBadge = "⚠ Session out"
		case profile.StatusLoggedOut, profile.StatusLoginRequired:
			statusBadge = "○ Logged out"
		default:
			statusBadge = "? Unknown"
		}
		items = append(items, handoffItem{
			profile: c,
			badge:   statusBadge,
		})
	}

	m := handoffPickerModel{
		candidates:     items,
		currentProfile: currentProfile,
		styles:         tui.DefaultStyles(),
	}

	p := tea.NewProgram(m)
	finalModel, err := p.Run()
	if err != nil {
		return nil, false
	}

	if picker, ok := finalModel.(handoffPickerModel); ok && picker.selected != nil {
		return picker.selected, true
	}

	return nil, false
}

type handoffItem struct {
	profile profile.Profile
	badge   string
}

type handoffPickerModel struct {
	candidates     []handoffItem
	cursor         int
	selected       *profile.Profile
	currentProfile *profile.Profile
	styles         tui.UIStyles
}

func (m handoffPickerModel) Init() tea.Cmd { return nil }

func (m handoffPickerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q", "esc":
			return m, tea.Quit
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.candidates)-1 {
				m.cursor++
			}
		case "enter":
			if len(m.candidates) > 0 {
				m.selected = &m.candidates[m.cursor].profile
			}
			return m, tea.Quit
		default:
			if len(msg.Runes) == 1 && msg.Runes[0] >= '1' && msg.Runes[0] <= '9' {
				idx := int(msg.Runes[0] - '1')
				if idx < len(m.candidates) {
					m.selected = &m.candidates[idx].profile
					return m, tea.Quit
				}
			}
			for _, item := range m.candidates {
				if strings.EqualFold(item.profile.Shortcut, msg.String()) {
					m.selected = &item.profile
					return m, tea.Quit
				}
			}
		}
	}
	return m, nil
}

func (m handoffPickerModel) View() string {
	var b strings.Builder
	b.WriteString("\n")
	if m.currentProfile != nil {
		b.WriteString(m.styles.WarnBadge.Render(fmt.Sprintf(" Claude Code session ended for %s ", m.currentProfile.Name)) + "\n")
		b.WriteString(m.styles.TextMuted.Render("Select profile to resume active conversation:") + "\n\n")
	} else {
		b.WriteString(m.styles.Title.Render(" Select profile to resume active conversation: ") + "\n\n")
	}

	for i, c := range m.candidates {
		prefix := "  "
		if i == m.cursor {
			prefix = "❯ "
		}
		itemStr := fmt.Sprintf("[%d] %-18s (%s)  %s", i+1, c.profile.Name, c.profile.Shortcut, c.badge)
		if i == m.cursor {
			b.WriteString(m.styles.ActiveItem.Render(prefix+itemStr) + "\n")
		} else {
			b.WriteString(prefix + itemStr + "\n")
		}
	}

	b.WriteString("\n" + m.styles.TextMuted.Render("Press 1-9 or Enter to select • q to exit") + "\n")
	return b.String()
}
