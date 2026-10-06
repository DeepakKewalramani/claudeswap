package cli

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/claudeswap/claudeswap/internal/profile"

	"github.com/spf13/cobra"
)

func newAddCmd(app *AppContext) *cobra.Command {
	var shortcutFlag string
	var noLaunchFlag bool

	cmd := &cobra.Command{
		Use:   "add [profile-name]",
		Short: "Add a new Claude Code profile",
		Long: `Add a new isolated Claude Code profile. ClaudeSwap initializes an isolated
configuration directory, generates a direct shortcut, and optionally launches Claude Code
so you can authenticate directly through Claude Code.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			var name string
			if len(args) > 0 {
				name = strings.Join(args, " ")
			}

			if strings.TrimSpace(name) == "" {
				return runInteractiveAdd(app)
			}

			return addProfile(app, name, shortcutFlag, noLaunchFlag)
		},
	}

	cmd.Flags().StringVarP(&shortcutFlag, "shortcut", "s", "", "Custom shortcut command name (e.g. claudeswap-work)")
	cmd.Flags().BoolVar(&noLaunchFlag, "no-launch", false, "Do not immediately launch Claude Code after profile creation")

	return cmd
}

func runInteractiveAdd(app *AppContext) error {
	reader := bufio.NewReader(os.Stdin)

	fmt.Println("\nAdd Claude Profile")
	fmt.Println("------------------")
	fmt.Print("Profile name: ")
	nameInput, err := reader.ReadString('\n')
	if err != nil {
		return err
	}
	name := strings.TrimSpace(nameInput)
	if err := profile.ValidateName(name); err != nil {
		fmt.Fprintf(os.Stderr, "✗ Error: %v\n", err)
		os.Exit(ExitInvalidUsage)
	}

	generatedID := profile.GenerateID(name)
	defaultSC := profile.DefaultShortcut(generatedID)

	fmt.Printf("Shortcut [%s]: ", defaultSC)
	shortcutInput, err := reader.ReadString('\n')
	if err != nil {
		return err
	}
	shortcut := strings.TrimSpace(shortcutInput)
	if shortcut == "" {
		shortcut = defaultSC
	}

	return addProfile(app, name, shortcut, false)
}

func addProfile(app *AppContext, name, customShortcut string, noLaunch bool) error {
	fmt.Printf("\n✓ Creating isolated Claude profile for %q...\n", name)
	p, err := app.Manager.CreateProfile(name, customShortcut)
	if err != nil {
		fmt.Fprintf(os.Stderr, "✗ Error creating profile: %v\n", err)
		os.Exit(ExitGeneralError)
	}

	scPath, err := app.Shortcut.CreateShortcut(p)
	if err != nil {
		fmt.Fprintf(os.Stderr, "⚠ Warning: Could not create shortcut: %v\n", err)
	} else {
		fmt.Printf("✓ Created direct shortcut: %s (%s)\n", p.Shortcut, scPath)
	}

	fmt.Println("✓ Profile created successfully")
	fmt.Printf("\nProfile:\n  %s\n", p.Name)
	fmt.Printf("Shortcut:\n  %s\n", p.Shortcut)
	fmt.Printf("Configuration:\n  %s\n", p.ConfigDir)
	fmt.Printf("\nLaunch directly with:\n  %s\n\n", p.Shortcut)

	if !noLaunch {
		fmt.Println("✓ Starting Claude Code...")
		fmt.Println("  (Claude Code will handle its own authentication)")
		fmt.Println()
		cfg, _ := app.CfgStore.Load()
		shared := cfg != nil && cfg.SharedContext
		code, err := app.Launcher.Launch(p, nil, shared)
		if err != nil {
			fmt.Fprintf(os.Stderr, "✗ Error launching Claude Code: %v\n", err)
			os.Exit(code)
		}
		os.Exit(code)
	}

	return nil
}
