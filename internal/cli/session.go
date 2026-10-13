package cli

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/claudeswap/claudeswap/internal/claude"
	"github.com/claudeswap/claudeswap/internal/profile"
	"github.com/claudeswap/claudeswap/internal/session"
	"github.com/claudeswap/claudeswap/internal/tui"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

func newSessionCmd(app *AppContext) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "sessions [subcommand]",
		Aliases: []string{"session", "s"},
		Short:   "Manage Claude Code sessions with folder-wise sidebar",
		Long: `Interactive Session Manager for Claude Code.
Browse sessions folder-wise with an interactive sidebar, resume conversations,
switch active sessions to another account quota, or clean up old transcripts.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				return cmd.Help()
			}

			// Interactive TTY: Launch the 2-column sidebar explorer
			if term.IsTerminal(int(os.Stdin.Fd())) && term.IsTerminal(int(os.Stdout.Fd())) {
				return runSessionBrowserTUI(app)
			}

			// Non-interactive fallback: list sessions
			return printSessionsList(app)
		},
	}

	cmd.AddCommand(
		newSessionListCmd(app),
		newSessionResumeCmd(app),
		newSessionDeleteCmd(app),
	)

	return cmd
}

func newSessionListCmd(app *AppContext) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all sessions grouped by project folder",
		RunE: func(cmd *cobra.Command, args []string) error {
			return printSessionsList(app)
		},
	}
}

func newSessionResumeCmd(app *AppContext) *cobra.Command {
	var profileFlag string

	cmd := &cobra.Command{
		Use:   "resume [session-id]",
		Short: "Resume a session under a specific profile",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			targetID := args[0]
			groups, err := app.SessMgr.ScanProjects()
			if err != nil {
				return err
			}

			sess, group, err := app.SessMgr.FindSessionByID(groups, targetID)
			if err != nil {
				fmt.Fprintf(os.Stderr, "✗ Error: %v\n", err)
				os.Exit(ExitInvalidUsage)
			}

			// Resolve target profile
			p, err := resolveTargetProfile(app, profileFlag)
			if err != nil {
				return err
			}

			return resumeSessionWithProfile(app, p, sess, group)
		},
	}

	cmd.Flags().StringVarP(&profileFlag, "profile", "p", "", "Profile to resume the session under")

	return cmd
}

func newSessionDeleteCmd(app *AppContext) *cobra.Command {
	return &cobra.Command{
		Use:     "delete <session-id>",
		Aliases: []string{"rm"},
		Short:   "Delete a session file",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			targetID := args[0]
			groups, err := app.SessMgr.ScanProjects()
			if err != nil {
				return err
			}

			sess, _, err := app.SessMgr.FindSessionByID(groups, targetID)
			if err != nil {
				fmt.Fprintf(os.Stderr, "✗ Error: %v\n", err)
				os.Exit(ExitInvalidUsage)
			}

			if err := app.SessMgr.DeleteSession(sess.FilePath); err != nil {
				fmt.Fprintf(os.Stderr, "✗ Failed to delete session: %v\n", err)
				os.Exit(ExitGeneralError)
			}

			fmt.Printf("✓ Deleted session %s (%s)\n", sess.ID[:8], sess.Title)
			return nil
		},
	}
}

func runSessionBrowserTUI(app *AppContext) error {
	cfg, _ := app.CfgStore.Load()
	var defaultProf string
	if cfg != nil {
		defaultProf = cfg.DefaultProfile
	}
	prof, err := resolveTargetProfile(app, defaultProf)
	if err != nil {
		return err
	}
	return runSessionBrowserTUIWithProfile(app, prof)
}

func runSessionBrowserTUIWithProfile(app *AppContext, prof *profile.Profile) error {
	for {
		groups, err := app.SessMgr.ScanProjects()
		if err != nil {
			fmt.Fprintf(os.Stderr, "✗ Error scanning sessions: %v\n", err)
			return err
		}

		cfg, _ := app.CfgStore.Load()
		shared := cfg != nil && cfg.SharedContext

		model := tui.NewSessionBrowserModel(groups, prof, shared)
		p := tea.NewProgram(model)
		finalModel, err := p.Run()
		if err != nil {
			return err
		}

		resModel, ok := finalModel.(tui.SessionBrowserModel)
		if !ok || resModel.Result == nil || resModel.Result.Action == "back" {
			return nil
		}

		res := resModel.Result
		switch res.Action {
		case "resume":
			_ = resumeSessionWithProfileInteractive(app, prof, res.Session, res.Group)
			continue

		case "new":
			_ = launchNewSessionWithProfileInteractive(app, prof, res.Group)
			continue

		case "matrix":
			_ = runSupervisorTUI(app)
			continue

		case "switch_profile":
			nextProf, proceed := PromptHandoffMenu(app, prof)
			if proceed && nextProf != nil {
				prof = nextProf
			}
			continue

		case "toggle_memory":
			if cfg != nil {
				cfg.SharedContext = !cfg.SharedContext
				_ = app.CfgStore.Save(cfg)
				profiles, _ := app.Manager.ListProfiles()
				for _, pr := range profiles {
					_ = claude.SyncSharedContext(app.Paths, pr.ConfigDir, cfg.SharedContext)
				}
			}
			continue

		case "handoff":
			nextProf, proceed := PromptHandoffMenu(app, prof)
			if proceed && nextProf != nil {
				return resumeSessionWithProfile(app, nextProf, res.Session, res.Group)
			}
			continue

		case "delete":
			if res.Session != nil {
				_ = app.SessMgr.DeleteSession(res.Session.FilePath)
				fmt.Printf("✓ Deleted session %s\n", res.Session.ID[:8])
			}
			continue
		}

		return nil
	}
}

func resumeSessionWithProfileInteractive(app *AppContext, p *profile.Profile, sess *session.Session, group *session.ProjectGroup) error {
	fmt.Printf("\n✓ Attaching to session %s (%s)\n", sess.ID[:8], sess.Title)
	fmt.Printf("✓ Profile: %s (%s)\n", p.Name, p.Shortcut)
	if sess.ProjectCWD != "" {
		fmt.Printf("✓ Directory: %s\n\n", sess.ProjectCWD)
		if info, err := os.Stat(sess.ProjectCWD); err == nil && info.IsDir() {
			_ = os.Chdir(sess.ProjectCWD)
		}
	}

	cfg, _ := app.CfgStore.Load()
	shared := cfg != nil && cfg.SharedContext

	resumeArgs := []string{"--resume", sess.ID}
	_, err := app.Launcher.Launch(p, resumeArgs, shared)
	if err != nil {
		fmt.Fprintf(os.Stderr, "✗ Error resuming session: %v\n", err)
	}
	return nil
}

func launchNewSessionWithProfileInteractive(app *AppContext, p *profile.Profile, group *session.ProjectGroup) error {
	folderName := "Current Folder"
	cwd := ""
	if group != nil {
		folderName = group.Name
		cwd = group.CWD
	}
	if cwd == "" {
		cwd, _ = os.Getwd()
	}

	fmt.Printf("\n✓ Starting new Claude Code session in %s\n", folderName)
	fmt.Printf("✓ Profile: %s (%s)\n", p.Name, p.Shortcut)
	fmt.Printf("✓ Directory: %s\n\n", cwd)

	if info, err := os.Stat(cwd); err == nil && info.IsDir() {
		_ = os.Chdir(cwd)
	}

	cfg, _ := app.CfgStore.Load()
	shared := cfg != nil && cfg.SharedContext

	_, err := app.Launcher.Launch(p, nil, shared)
	if err != nil {
		fmt.Fprintf(os.Stderr, "✗ Error launching Claude: %v\n", err)
	}
	return nil
}

func launchNewSessionWithProfile(app *AppContext, p *profile.Profile, group *session.ProjectGroup) error {
	folderName := "Current Folder"
	cwd := ""
	if group != nil {
		folderName = group.Name
		cwd = group.CWD
	}
	if cwd == "" {
		cwd, _ = os.Getwd()
	}

	fmt.Printf("\n✓ Starting new Claude Code session in %s\n", folderName)
	fmt.Printf("✓ Profile: %s (%s)\n", p.Name, p.Shortcut)
	fmt.Printf("✓ Directory: %s\n\n", cwd)

	if info, err := os.Stat(cwd); err == nil && info.IsDir() {
		_ = os.Chdir(cwd)
	}

	cfg, _ := app.CfgStore.Load()
	shared := cfg != nil && cfg.SharedContext

	code, err := app.Launcher.Launch(p, nil, shared)
	if err != nil {
		fmt.Fprintf(os.Stderr, "✗ Error launching Claude: %v\n", err)
	}
	os.Exit(code)
	return nil
}

func printSessionsList(app *AppContext) error {
	groups, err := app.SessMgr.ScanProjects()
	if err != nil {
		fmt.Fprintf(os.Stderr, "✗ Error loading sessions: %v\n", err)
		return err
	}

	if len(groups) == 0 {
		fmt.Println("No sessions found.")
		return nil
	}

	fmt.Println("Claude Code Sessions (Folder-Wise)")
	fmt.Println("=================================")

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	for _, g := range groups {
		fmt.Printf("\n📁 %s (%s) — %d session(s)\n", g.Name, g.CWD, len(g.Sessions))
		fmt.Fprintln(w, "  SESSION\tBRANCH\tACTIVE\tTITLE")
		fmt.Fprintln(w, "  -------\t------\t------\t-----")
		for _, s := range g.Sessions {
			shortID := s.ID
			if len(shortID) > 8 {
				shortID = shortID[:8]
			}
			branch := s.GitBranch
			if branch == "" {
				branch = "main"
			}
			title := s.Title
			if len(title) > 50 {
				title = title[:47] + "..."
			}
			fmt.Fprintf(w, "  %s\t%s\t%s\t%s\n", shortID, branch, s.UpdatedAt.Format("Jan 02 15:04"), title)
		}
		w.Flush()
	}

	return nil
}

func resolveTargetProfile(app *AppContext, profileName string) (*profile.Profile, error) {
	if profileName != "" {
		return app.Manager.GetProfile(profileName)
	}

	cfg, _ := app.CfgStore.Load()
	if cfg != nil && cfg.DefaultProfile != "" {
		if p, err := app.Manager.GetProfile(cfg.DefaultProfile); err == nil {
			return p, nil
		}
	}

	profiles, err := app.Manager.ListProfiles()
	if err != nil || len(profiles) == 0 {
		return nil, fmt.Errorf("no profiles configured in ClaudeSwap. Run 'claudeswap add' first")
	}

	return &profiles[0], nil
}

func resumeSessionWithProfile(app *AppContext, p *profile.Profile, sess *session.Session, group *session.ProjectGroup) error {
	fmt.Printf("\n✓ Resuming session %s (%s)\n", sess.ID[:8], sess.Title)
	fmt.Printf("✓ Profile: %s (%s)\n", p.Name, p.Shortcut)
	if sess.ProjectCWD != "" {
		fmt.Printf("✓ Directory: %s\n\n", sess.ProjectCWD)
		// Change working directory to the session's original folder if valid
		if info, err := os.Stat(sess.ProjectCWD); err == nil && info.IsDir() {
			_ = os.Chdir(sess.ProjectCWD)
		}
	}

	cfg, _ := app.CfgStore.Load()
	shared := cfg != nil && cfg.SharedContext

	resumeArgs := []string{"--resume", sess.ID}
	code, err := app.Launcher.Launch(p, resumeArgs, shared)
	if err != nil {
		fmt.Fprintf(os.Stderr, "✗ Error resuming session: %v\n", err)
	}
	os.Exit(code)
	return nil
}
