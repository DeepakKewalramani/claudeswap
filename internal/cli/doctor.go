package cli

import (
	"fmt"
	"os"

	"github.com/claudeswap/claudeswap/internal/profile"
	"github.com/claudeswap/claudeswap/internal/shortcut"

	"github.com/spf13/cobra"
)

func newDoctorCmd(app *AppContext) *cobra.Command {
	var fixShellFlag bool

	cmd := &cobra.Command{
		Use:   "doctor",
		Short: "Run system and configuration diagnostics",
		Long: `Verify that ClaudeSwap, its directories, profile database, Claude Code CLI,
and shell PATH integration are properly configured.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if fixShellFlag {
				msg, err := shortcut.InstallShellIntegration(app.Paths)
				if err != nil {
					fmt.Fprintf(os.Stderr, "✗ Failed to update shell integration: %v\n", err)
					return err
				}
				fmt.Println(msg)
				return nil
			}

			return runDoctorCheck(app)
		},
	}

	cmd.Flags().BoolVar(&fixShellFlag, "install-shell", false, "Safely append ClaudeSwap bin directory to your shell configuration file")

	return cmd
}

func runDoctorCheck(app *AppContext) error {
	fmt.Println("ClaudeSwap Diagnostics")
	fmt.Println("=====================")

	hasCriticalIssues := false

	// 1. Check directories
	if err := app.Paths.EnsureDirectories(); err != nil {
		fmt.Printf("✗ ClaudeSwap directories: failed to create (%v)\n", err)
		hasCriticalIssues = true
	} else {
		fmt.Printf("✓ ClaudeSwap installation: %s\n", app.Paths.HomeDir())
	}

	// 2. Check profile database
	storeData, err := app.Store.Load()
	if err != nil {
		fmt.Printf("✗ Profile database: corrupted or unreadable (%v)\n", err)
		if app.Store.HasBackup() {
			fmt.Printf("  ➜ Backup available at %s. Run 'claudeswap settings --restore' to recover.\n", app.Paths.ProfilesBakFile())
		}
		hasCriticalIssues = true
	} else {
		fmt.Printf("✓ Profile database: OK (%d profiles, schema v%d)\n", len(storeData.Profiles), storeData.Version)
	}

	// 3. Check AppConfig
	_, err = app.CfgStore.Load()
	if err != nil {
		fmt.Printf("✗ Configuration: corrupted or unreadable (%v)\n", err)
		hasCriticalIssues = true
	} else {
		fmt.Println("✓ Configuration: OK")
	}

	// 4. Check Claude Code CLI detection
	claudePath, err := app.Detector.FindExecutable()
	if err != nil {
		fmt.Println("✗ Claude Code detected: NOT FOUND")
		fmt.Println("  ➜ Claude Code was not found on your system.")
		fmt.Println("  ➜ Please install Claude Code and ensure it is available in your PATH,")
		fmt.Println("  ➜ or configure its path using: claudeswap settings --claude-path <path>")
		hasCriticalIssues = true
	} else {
		version, vErr := app.Detector.GetVersion(claudePath)
		if vErr == nil && version != "" {
			fmt.Printf("✓ Claude Code detected: %s (v%s)\n", claudePath, version)
		} else {
			fmt.Printf("✓ Claude Code detected: %s\n", claudePath)
		}
	}

	// 5. Check Shortcut bin directory & PATH
	binDir := app.Paths.BinDir()
	shellInfo := shortcut.DetectShell(app.Paths)

	fmt.Printf("✓ Shortcut directory: %s\n", binDir)
	if shellInfo.BinDirInPath {
		fmt.Println("✓ PATH: Shortcut directory is active in PATH")
	} else {
		fmt.Println("⚠ PATH: Shortcut directory is NOT in your current PATH")
		fmt.Printf("  ➜ Detected shell: %s\n", shellInfo.ShellType)
		if shellInfo.ConfigFile != "" {
			fmt.Printf("  ➜ Add with: claudeswap doctor --install-shell\n")
			fmt.Printf("  ➜ Or manually add to %s:\n      %s\n", shellInfo.ConfigFile, shellInfo.IntegrationLine)
		}
	}

	// 6. Profiles breakdown
	if storeData != nil && len(storeData.Profiles) > 0 {
		fmt.Println("\nProfiles:")
		for _, p := range storeData.Profiles {
			status := app.Manager.GetStatus(&p)
			switch status {
			case profile.StatusReady:
				fmt.Printf("  ✓ %-20s (ready)\n", p.Name)
			case profile.StatusLoginRequired:
				fmt.Printf("  ⚠ %-20s (requires authentication)\n", p.Name)
			default:
				fmt.Printf("  ? %-20s (unknown status)\n", p.Name)
			}
		}
	}

	fmt.Println()
	if hasCriticalIssues {
		fmt.Println("⚠ Issues were detected that may prevent ClaudeSwap from functioning.")
	} else {
		fmt.Println("✓ No critical problems detected.")
	}

	return nil
}
