package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func newSettingsCmd(app *AppContext) *cobra.Command {
	var claudePathFlag string
	var restoreBackupFlag bool
	var enableSharedContextFlag bool
	var disableSharedContextFlag bool
	var enableAutoHandoffFlag bool
	var disableAutoHandoffFlag bool

	cmd := &cobra.Command{
		Use:   "settings",
		Short: "View and manage ClaudeSwap settings",
		Long: `Inspect current configuration, set custom Claude Code binary path,
manage shared memory context across accounts, or recover from profile database backups.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if restoreBackupFlag {
				if err := app.Store.RestoreBackup(); err != nil {
					fmt.Fprintf(os.Stderr, "✗ Failed to restore profile backup: %v\n", err)
					os.Exit(ExitConfigError)
				}
				fmt.Println("✓ Successfully restored profile database from backup.")
				return nil
			}

			if enableSharedContextFlag {
				cfg, _ := app.CfgStore.Load()
				cfg.SharedContext = true
				_ = app.CfgStore.Save(cfg)
				fmt.Println("✓ Shared context enabled across all profiles.")
				return nil
			}

			if disableSharedContextFlag {
				cfg, _ := app.CfgStore.Load()
				cfg.SharedContext = false
				_ = app.CfgStore.Save(cfg)
				fmt.Println("✓ Shared context disabled.")
				return nil
			}

			if enableAutoHandoffFlag {
				cfg, _ := app.CfgStore.Load()
				t := true
				cfg.AutoHandoff = &t
				_ = app.CfgStore.Save(cfg)
				fmt.Println("✓ Auto-handoff prompt enabled on session exit.")
				return nil
			}

			if disableAutoHandoffFlag {
				cfg, _ := app.CfgStore.Load()
				f := false
				cfg.AutoHandoff = &f
				_ = app.CfgStore.Save(cfg)
				fmt.Println("✓ Auto-handoff prompt disabled.")
				return nil
			}

			if claudePathFlag != "" {
				cfg, err := app.CfgStore.Load()
				if err != nil {
					fmt.Fprintf(os.Stderr, "✗ Error reading config: %v\n", err)
					os.Exit(ExitConfigError)
				}
				cfg.ClaudeBinaryPath = claudePathFlag
				if err := app.CfgStore.Save(cfg); err != nil {
					fmt.Fprintf(os.Stderr, "✗ Failed to save config: %v\n", err)
					os.Exit(ExitConfigError)
				}
				fmt.Printf("✓ Configured Claude binary path: %s\n", claudePathFlag)
				return nil
			}

			return runSettingsView(app)
		},
	}

	cmd.Flags().StringVar(&claudePathFlag, "claude-path", "", "Set custom Claude Code binary path")
	cmd.Flags().BoolVar(&restoreBackupFlag, "restore", false, "Restore profile database from backup (profiles.json.bak)")
	cmd.Flags().BoolVar(&enableSharedContextFlag, "enable-shared-context", false, "Share conversation history and memory across all profiles")
	cmd.Flags().BoolVar(&disableSharedContextFlag, "disable-shared-context", false, "Disable shared conversation history across profiles")
	cmd.Flags().BoolVar(&enableAutoHandoffFlag, "enable-auto-handoff", false, "Prompt to hand off active session to another account when finished")
	cmd.Flags().BoolVar(&disableAutoHandoffFlag, "disable-auto-handoff", false, "Do not prompt for account handoff on session exit")

	return cmd
}

func runSettingsView(app *AppContext) error {
	cfg, err := app.CfgStore.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "✗ Warning: Could not read config file: %v\n", err)
	}

	fmt.Println("ClaudeSwap Settings")
	fmt.Println("==================")
	fmt.Printf("Home directory:     %s\n", app.Paths.HomeDir())
	fmt.Printf("Config file:        %s\n", app.Paths.ConfigFile())
	fmt.Printf("Profiles file:      %s\n", app.Paths.ProfilesFile())
	fmt.Printf("Shared directory:   %s\n", app.Paths.SharedDir())
	fmt.Printf("Bin directory:      %s\n", app.Paths.BinDir())
	fmt.Printf("Backups directory:  %s\n", app.Paths.BackupsDir())

	if cfg != nil {
		if cfg.SharedContext {
			fmt.Println("Shared Context:     ✓ Enabled (sessions & projects shared)")
		} else {
			fmt.Println("Shared Context:     ○ Disabled (isolated sessions)")
		}

		if cfg.IsAutoHandoffEnabled() {
			fmt.Println("Auto-Handoff:       ✓ Enabled (prompts next account on session end)")
		} else {
			fmt.Println("Auto-Handoff:       ○ Disabled")
		}
		if cfg.DefaultProfile != "" {
			fmt.Printf("Default profile:    %s\n", cfg.DefaultProfile)
		} else {
			fmt.Println("Default profile:    (none)")
		}

		if cfg.ClaudeBinaryPath != "" {
			fmt.Printf("Claude binary:      %s (custom)\n", cfg.ClaudeBinaryPath)
		} else {
			detected, err := app.Detector.FindExecutable()
			if err == nil {
				fmt.Printf("Claude binary:      %s (auto-detected)\n", detected)
			} else {
				fmt.Println("Claude binary:      (not found)")
			}
		}
	}

	hasBak := app.Store.HasBackup()
	fmt.Printf("Backup available:   %t\n", hasBak)

	return nil
}
