package cli

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/claudeswap/claudeswap/internal/claude"
	"github.com/claudeswap/claudeswap/internal/profile"

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

	fmt.Println()
	fmt.Println("────────────────────────────────────────────────────────────")
	if currentProfile != nil {
		fmt.Printf("Claude Code session ended for %s.\n\n", currentProfile.Name)
		fmt.Println("Reached account limits or want to continue with another account?")
	} else {
		fmt.Println("Select profile to hand off and resume active conversation:")
	}
	fmt.Println()

	for i, c := range candidates {
		status := app.Manager.GetStatus(&c)
		statusBadge := "✓ Ready"
		if status == profile.StatusLoginRequired {
			statusBadge = "⚠ Login required"
		}
		fmt.Printf("  [%d] %-20s (%s)  %s\n", i+1, c.Name, c.Shortcut, statusBadge)
	}
	fmt.Println("  [q] Exit to shell")
	fmt.Println()
	fmt.Print("Select account [1-", len(candidates), ", or Enter/q to exit]: ")

	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		return nil, false
	}

	choice := strings.TrimSpace(input)
	if choice == "" || strings.EqualFold(choice, "q") || strings.EqualFold(choice, "exit") {
		return nil, false
	}

	// Check if user entered a number
	if num, err := strconv.Atoi(choice); err == nil && num >= 1 && num <= len(candidates) {
		selected := candidates[num-1]
		return &selected, true
	}

	// Check if user entered a profile name or shortcut directly
	for _, c := range candidates {
		if strings.EqualFold(c.ID, choice) || strings.EqualFold(c.Name, choice) || strings.EqualFold(c.Shortcut, choice) {
			selected := c
			return &selected, true
		}
	}

	return nil, false
}
