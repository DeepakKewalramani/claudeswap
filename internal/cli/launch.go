package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/claudeswap/claudeswap/internal/profile"

	"github.com/spf13/cobra"
	"golang.org/x/term"
)

func newLaunchCmd(app *AppContext) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "launch [profile] [claude-args...]",
		Short: "Launch Claude Code using a specific profile",
		Long: `Launch Claude Code in the isolated environment of the specified profile.
All flags and arguments are forwarded directly to Claude Code unchanged.
If no profile is specified, the default profile is used.`,
		DisableFlagParsing: true, // Allow passing arbitrary Claude Code flags directly (e.g. --continue, -r)
		RunE: func(cmd *cobra.Command, rawArgs []string) error {
			var profileTarget string
			var claudeArgs []string

			// If user asked for help explicitly
			if len(rawArgs) == 1 && (rawArgs[0] == "-h" || rawArgs[0] == "--help" || rawArgs[0] == "help") {
				return cmd.Help()
			}

			// Check if first arg is a profile name or a flag
			if len(rawArgs) > 0 && !strings.HasPrefix(rawArgs[0], "-") {
				profileTarget = rawArgs[0]
				claudeArgs = rawArgs[1:]
			} else {
				// No profile provided or first arg is a flag, check default profile
				claudeArgs = rawArgs
				cfg, err := app.CfgStore.Load()
				if err == nil && cfg != nil && cfg.DefaultProfile != "" {
					profileTarget = cfg.DefaultProfile
				}
			}

			if profileTarget == "" {
				fmt.Fprintln(os.Stderr, "✗ No profile specified and no default profile configured.")
				fmt.Fprintln(os.Stderr, "\nUsage: claudeswap launch <profile> [flags...]")
				printAvailableProfiles(app)
				os.Exit(ExitInvalidUsage)
			}

			p, err := app.Manager.GetProfile(profileTarget)
			if err != nil {
				fmt.Fprintf(os.Stderr, "✗ Profile %q was not found.\n", profileTarget)
				printAvailableProfiles(app)
				os.Exit(ExitProfileNotFound)
			}

			status := app.Manager.GetStatus(p)
			if status == profile.StatusLoginRequired {
				fmt.Printf("⚠ This profile may require authentication.\n\nProfile:\n  %s\n\nStarting Claude Code...\n", p.Name)
			}

			cfg, _ := app.CfgStore.Load()
			shared := cfg != nil && cfg.SharedContext

			for {
				code, err := app.Launcher.Launch(p, claudeArgs, shared)
				if err != nil {
					fmt.Fprintf(os.Stderr, "✗ Error launching Claude Code: %v\n", err)
					os.Exit(code)
				}

				if cfg == nil || !cfg.IsAutoHandoffEnabled() || !term.IsTerminal(int(os.Stdin.Fd())) {
					os.Exit(code)
				}

				nextProfile, proceed := PromptHandoffMenu(app, p)
				if !proceed || nextProfile == nil {
					os.Exit(code)
				}

				// Hand off to next profile and resume session seamlessly!
				p = nextProfile
				claudeArgs = []string{"--resume"}
				shared = true
				fmt.Printf("\n✓ Handing off active conversation to profile %q (%s)...\n", p.Name, p.ID)
				fmt.Println("✓ Resuming session under new account quota...")
			}
		},
	}

	return cmd
}
