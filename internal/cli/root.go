package cli

import (
	"fmt"
	"os"

	"github.com/claudeswap/claudeswap/internal/claude"
	"github.com/claudeswap/claudeswap/internal/config"
	"github.com/claudeswap/claudeswap/internal/profile"
	"github.com/claudeswap/claudeswap/internal/session"
	"github.com/claudeswap/claudeswap/internal/shortcut"
	"github.com/claudeswap/claudeswap/internal/supervisor"
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
	Launcher   *claude.Launcher
	Shortcut   *shortcut.Manager
	SessMgr    *session.Manager
	Supervisor *supervisor.Supervisor
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
	mgr.SetStatusChecker(detector.CheckAuthStatus)
	launcher := claude.NewLauncher(detector, paths)
	scMgr := shortcut.NewManager(paths)
	sessMgr := session.NewManager(paths, store)
	sup := supervisor.NewSupervisor(paths, detector, sessMgr)

	return &AppContext{
		Paths:      paths,
		CfgStore:   cfgStore,
		Store:      store,
		Manager:    mgr,
		Detector:   detector,
		Launcher:   launcher,
		Shortcut:   scMgr,
		SessMgr:    sessMgr,
		Supervisor: sup,
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
		newSessionCmd(app),
		newMatrixCmd(app),
		newVersionCmd(),
	)

	return rootCmd
}

func runTUI(app *AppContext) error {
	for {
		cfg, _ := app.CfgStore.Load()
		shared := cfg != nil && cfg.SharedContext

		profiles, err := app.Manager.ListProfiles()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error loading profiles: %v\n", err)
			return err
		}

		app.Manager.PreloadStatuses(profiles)

		var items []tui.ProfileWithStatus
		for _, p := range profiles {
			status := app.Manager.GetStatus(&p)
			items = append(items, tui.ProfileWithStatus{
				Profile: p,
				Status:  status,
			})
		}

		m := tui.NewModel(items, shared)
		p := tea.NewProgram(m)
		finalModel, err := p.Run()
		if err != nil {
			return err
		}

		// Process any selected action after TUI exits
		if model, ok := finalModel.(tui.Model); ok && model.SelectedResult != nil {
			res := model.SelectedResult
			switch res.Type {
			case "toggle_memory":
				cfg, _ := app.CfgStore.Load()
				if cfg == nil {
					cfg = &config.AppConfig{}
				}
				cfg.SharedContext = !cfg.SharedContext
				_ = app.CfgStore.Save(cfg)
				for _, pr := range profiles {
					_ = claude.SyncSharedContext(app.Paths, pr.ConfigDir, cfg.SharedContext)
				}
				app.Manager.ClearStatusCache()
				continue

			case "launch":
				if res.TargetProfile != nil {
					_ = runSessionBrowserTUIWithProfile(app, res.TargetProfile)
					continue
				}
				return nil

			case "handoff":
				nextProfile, proceed := PromptHandoffMenu(app, nil)
				if proceed && nextProfile != nil {
					fmt.Printf("\n✓ Resuming session under profile %q (%s)...\n", nextProfile.Name, nextProfile.ID)
					code, err := app.Launcher.Launch(nextProfile, []string{"--resume"}, true)
					if err != nil {
						fmt.Fprintf(os.Stderr, "Error launching profile: %v\n", err)
					}
					os.Exit(code)
				}
				return nil

			case "remove":
				target := res.TargetProfile
				if target != nil {
					return PromptRemove(app, target)
				}
				return nil

			case "add":
				return runInteractiveAdd(app)

			case "session_manager":
				var targetProf *profile.Profile
				if len(profiles) > 0 {
					targetProf = &profiles[0]
				}
				_ = runSessionBrowserTUIWithProfile(app, targetProf)
				continue

			case "matrix":
				_ = runSupervisorTUI(app)
				continue

			case "doctor":
				return runDoctorCheck(app)

			case "settings":
				return runSettingsView(app)
			}
		}

		return nil
	}
}
