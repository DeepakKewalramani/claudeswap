package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/claudeswap/claudeswap/internal/claude"

	"github.com/spf13/cobra"
)

func newContextCmd(app *AppContext) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "context [on|off|status]",
		Short: "Manage shared memory and conversation context across profiles",
		Long: `Shared Context allows multiple Claude Code profiles (e.g. Deepak and Rohit)
to share conversation sessions, project history, and command history while keeping
their authentication credentials strictly separate.

When an account hits rate limits, switching to another profile with shared context
allows Claude Code to resume the exact same conversation seamlessly.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := app.CfgStore.Load()
			if err != nil {
				fmt.Fprintf(os.Stderr, "✗ Error reading config: %v\n", err)
				os.Exit(ExitConfigError)
			}

			if len(args) == 0 || args[0] == "status" {
				return showContextStatus(app, cfg.SharedContext)
			}

			action := strings.ToLower(args[0])
			switch action {
			case "on", "enable", "true":
				cfg.SharedContext = true
				if err := app.CfgStore.Save(cfg); err != nil {
					fmt.Fprintf(os.Stderr, "✗ Error saving config: %v\n", err)
					os.Exit(ExitConfigError)
				}

				// Synchronize existing profiles
				profiles, _ := app.Manager.ListProfiles()
				for _, p := range profiles {
					_ = claude.SyncSharedContext(app.Paths, p.ConfigDir, true)
				}

				fmt.Println("✓ Shared Context ENABLED across all profiles.")
				fmt.Printf("  • Shared directory: %s\n", app.Paths.SharedDir())
				fmt.Println("  • Sessions, project memory, and command history are now unified.")
				fmt.Println("  • Account authentication credentials remain 100% isolated.")
				fmt.Println("\nTip: When an account hits rate limits, switch profiles seamlessly with:")
				fmt.Println("  claudeswap handoff <other-profile>")

			case "off", "disable", "false":
				cfg.SharedContext = false
				if err := app.CfgStore.Save(cfg); err != nil {
					fmt.Fprintf(os.Stderr, "✗ Error saving config: %v\n", err)
					os.Exit(ExitConfigError)
				}

				// Detach shared symlinks
				profiles, _ := app.Manager.ListProfiles()
				for _, p := range profiles {
					_ = claude.SyncSharedContext(app.Paths, p.ConfigDir, false)
				}

				fmt.Println("✓ Shared Context DISABLED.")
				fmt.Println("  • All profiles now maintain completely independent conversation history.")

			default:
				fmt.Fprintf(os.Stderr, "Unknown action %q. Use 'on', 'off', or 'status'.\n", action)
				os.Exit(ExitInvalidUsage)
			}

			return nil
		},
	}

	return cmd
}

func showContextStatus(app *AppContext, enabled bool) error {
	fmt.Println("ClaudeSwap Memory & Context Status")
	fmt.Println("=================================")
	if enabled {
		fmt.Println("Status:           ✓ ENABLED (Shared across accounts)")
		fmt.Printf("Shared Base:      %s\n", app.Paths.SharedDir())
		fmt.Println("\nLinked components:")
		fmt.Println("  • sessions/      (Active and past conversation transcripts)")
		fmt.Println("  • projects/      (Project memory and state mappings)")
		fmt.Println("  • file-history/  (File edit caches)")
		fmt.Println("  • plans/         (Active planning state)")
		fmt.Println("  • history.jsonl  (Command history)")
		fmt.Println("\nSecurity boundary:")
		fmt.Println("  • Authentication credentials remain strictly isolated per account.")
	} else {
		fmt.Println("Status:           ○ DISABLED (Isolated per account)")
		fmt.Println("\nTo enable shared conversation sessions between profiles, run:")
		fmt.Println("  claudeswap context on")
	}

	return nil
}
