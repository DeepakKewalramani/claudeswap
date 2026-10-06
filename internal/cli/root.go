package cli

import (
	"fmt"
	"os"

	"github.com/claudeswap/claudeswap/internal/claude"
	"github.com/claudeswap/claudeswap/internal/config"
	"github.com/claudeswap/claudeswap/internal/profile"
	"github.com/claudeswap/claudeswap/internal/shortcut"
	"github.com/claudeswap/claudeswap/internal/tui"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

// AppContext holds shared dependencies for all commands.
type AppContext struct {
	Paths    *config.Paths
	CfgStore *config.ConfigStore
	Store    *profile.Store
	Manager  *profile.Manager
	Detector *claude.Detector
	Launcher *claude.Launcher
	Shortcut *shortcut.Manager
}

// NewAppContext initializes the shared application dependencies.
func NewAppContext() (*AppContext, error) {
	paths, err := config.NewPaths()
	if err != nil {
		return nil, err
	}

	cfgStore := config.NewConfigStore(paths)
	store := profile.NewStore(paths)
	mgr := profile.NewManager(paths, store)

	cfg, _ := cfgStore.Load()
	var configuredClaude string
	if cfg != nil {
		configuredClaude = cfg.ClaudeBinaryPath
	}

	detector := claude.NewDetector(configuredClaude)
	launcher := claude.NewLauncher(detector, paths)
	scMgr := shortcut.NewManager(paths)

	return &AppContext{
		Paths:    paths,
		CfgStore: cfgStore,
		Store:    store,
		Manager:  mgr,
		Detector: detector,
		Launcher: launcher,
		Shortcut: scMgr,
	}, nil
}

// NewRootCmd creates the primary claudeswap command.
func NewRootCmd(app *AppContext) *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "claudeswap",
		Short: "ClaudeSwap - Seamless Claude Code Account Swapper & Memory Continuity",
		Long: `ClaudeSwap allows developers to manage and instantly swap between multiple independent
Claude Code profiles with unified context and shared session memory.

When an account reaches rate limits or quota, ClaudeSwap seamlessly transfers the active
conversation to your next account so you never lose flow. ClaudeSwap never stores or touches
your credentials.`,
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			// If arguments provided or non-interactive terminal, show help
			isTerminal := term.IsTerminal(int(os.Stdin.Fd())) && term.IsTerminal(int(os.Stdout.Fd()))
			if !isTerminal {
				return cmd.Help()
			}

			// Interactive TUI
			return runTUI(app)
		},
	}

	rootCmd.AddCommand(
		newAddCmd(app),
		newListCmd(app),
		newStatusCmd(app),
		newLaunchCmd(app),
		newHandoffCmd(app),
		newContextCmd(app),
		newChatCmd(app),
		newRemoveCmd(app),
		newRenameCmd(app),
		newShortcutCmd(app),
		newDoctorCmd(app),
		newSettingsCmd(app),
		newSetDefaultCmd(app),
		newVersionCmd(),
	)

	return rootCmd
}

func runTUI(app *AppContext) error {
	profiles, err := app.Manager.ListProfiles()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading profiles: %v\n", err)
		return err
	}

	var items []tui.ProfileWithStatus
	for _, p := range profiles {
		status := app.Manager.GetStatus(&p)
		items = append(items, tui.ProfileWithStatus{
			Profile: p,
			Status:  status,
		})
	}

	m := tui.NewModel(items)
	p := tea.NewProgram(m)
	finalModel, err := p.Run()
	if err != nil {
		return err
	}

	// Process any selected action after TUI exits
	if model, ok := finalModel.(tui.Model); ok && model.SelectedResult != nil {
		res := model.SelectedResult
		switch res.Type {
		case "launch":
			if res.TargetProfile != nil {
				fmt.Printf("Launching %s...\n\n", res.TargetProfile.Name)
				cfg, _ := app.CfgStore.Load()
				shared := cfg != nil && cfg.SharedContext
				code, err := app.Launcher.Launch(res.TargetProfile, nil, shared)
				if err != nil {
					fmt.Fprintf(os.Stderr, "Error launching profile: %v\n", err)
				}
				os.Exit(code)
			}
		case "add":
			return runInteractiveAdd(app)
		case "doctor":
			return runDoctorCheck(app)
		case "settings":
			return runSettingsView(app)
		}
	}

	return nil
}
