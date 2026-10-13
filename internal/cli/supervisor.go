package cli

import (
	"fmt"
	"os"
	"text/tabwriter"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/claudeswap/claudeswap/internal/supervisor"
	"github.com/claudeswap/claudeswap/internal/tui"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

func newMatrixCmd(app *AppContext) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "matrix [subcommand]",
		Aliases: []string{"supervisor", "ps", "top"},
		Short:   "Multi-session background supervisor for 20+ parallel Claude sessions",
		Long: `Mission Control Matrix for managing large-scale multi-terminal Claude Code workflows.
Monitor up to 20 background Claude sessions, check live tool activities, see which sessions
are waiting for user input, and attach/detach with one keypress without losing progress.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 && term.IsTerminal(int(os.Stdin.Fd())) {
				return runSupervisorTUI(app)
			}
			return printSupervisorList(app)
		},
	}

	cmd.AddCommand(
		newMatrixListCmd(app),
		newMatrixSpawnCmd(app),
		newMatrixAttachCmd(app),
		newMatrixKillCmd(app),
		newMatrixLogsCmd(app),
	)

	return cmd
}

func newMatrixListCmd(app *AppContext) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all background Claude sessions",
		RunE: func(cmd *cobra.Command, args []string) error {
			return printSupervisorList(app)
		},
	}
}

func newMatrixSpawnCmd(app *AppContext) *cobra.Command {
	var profileName string
	var projectDir string
	var query string

	cmd := &cobra.Command{
		Use:   "spawn <task-name>",
		Short: "Spawn a new background Claude session",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			taskName := args[0]
			if projectDir == "" {
				projectDir, _ = os.Getwd()
			}

			p, err := resolveTargetProfile(app, profileName)
			if err != nil {
				return err
			}

			sess, err := app.Supervisor.StartSession(supervisor.StartOptions{
				TaskName:     taskName,
				ProjectDir:   projectDir,
				Profile:      p,
				InitialQuery: query,
			})
			if err != nil {
				return fmt.Errorf("failed to spawn background session: %w", err)
			}

			fmt.Printf("✓ Spawned background Claude session:\n")
			fmt.Printf("  Task:    %s\n", sess.TaskName)
			fmt.Printf("  ID:      %s\n", sess.ID)
			fmt.Printf("  PID:     %d\n", sess.PID)
			fmt.Printf("  Profile: %s\n", sess.ProfileName)
			fmt.Printf("  Folder:  %s\n", sess.ProjectDir)
			fmt.Printf("  Logs:    %s\n\n", sess.LogPath)
			fmt.Printf("Attach to it anytime using: claudeswap matrix attach %s\n", sess.ID)
			return nil
		},
	}

	cmd.Flags().StringVarP(&profileName, "profile", "p", "", "Profile to run the session under")
	cmd.Flags().StringVarP(&projectDir, "dir", "d", "", "Working directory for the session")
	cmd.Flags().StringVarP(&query, "query", "q", "", "Initial prompt query for Claude")

	return cmd
}

func newMatrixAttachCmd(app *AppContext) *cobra.Command {
	return &cobra.Command{
		Use:   "attach <session-id-or-task>",
		Short: "Attach interactive terminal to a background session",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			target := args[0]
			sessions, err := app.Supervisor.ListSessions()
			if err != nil {
				return err
			}

			var matched *supervisor.ManagedSession
			for _, s := range sessions {
				if s.ID == target || s.TaskName == target {
					matched = &s
					break
				}
			}

			if matched == nil {
				return fmt.Errorf("session %q not found", target)
			}

			return attachToManagedSession(app, matched)
		},
	}
}

func newMatrixKillCmd(app *AppContext) *cobra.Command {
	return &cobra.Command{
		Use:   "kill <session-id-or-task>",
		Short: "Terminate a background session",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			target := args[0]
			sessions, err := app.Supervisor.ListSessions()
			if err != nil {
				return err
			}

			targetID := target
			for _, s := range sessions {
				if s.TaskName == target {
					targetID = s.ID
					break
				}
			}

			if err := app.Supervisor.KillSession(targetID); err != nil {
				return err
			}
			fmt.Printf("✓ Terminated background session %s\n", target)
			return nil
		},
	}
}

func newMatrixLogsCmd(app *AppContext) *cobra.Command {
	var tailLines int

	cmd := &cobra.Command{
		Use:   "logs <session-id-or-task>",
		Short: "View logs from a background session",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			target := args[0]
			sessions, _ := app.Supervisor.ListSessions()
			targetID := target
			for _, s := range sessions {
				if s.TaskName == target {
					targetID = s.ID
					break
				}
			}

			logs, err := app.Supervisor.GetLogs(targetID, tailLines)
			if err != nil {
				return err
			}

			for _, l := range logs {
				fmt.Println(l)
			}
			return nil
		},
	}

	cmd.Flags().IntVarP(&tailLines, "tail", "n", 30, "Number of log lines to show")
	return cmd
}

func runSupervisorTUI(app *AppContext) error {
	for {
		sessions, err := app.Supervisor.ListSessions()
		if err != nil {
			return err
		}

		model := tui.NewSupervisorModel(
			sessions,
			func(id string, tail int) []string {
				logs, _ := app.Supervisor.GetLogs(id, tail)
				return logs
			},
			func() []supervisor.ManagedSession {
				s, _ := app.Supervisor.ListSessions()
				return s
			},
		)

		p := tea.NewProgram(model)
		finalModel, err := p.Run()
		if err != nil {
			return err
		}

		resModel, ok := finalModel.(tui.SupervisorModel)
		if !ok || resModel.Result == nil || resModel.Result.Action == "back" {
			return nil
		}

		res := resModel.Result
		switch res.Action {
		case "attach":
			if res.Session != nil {
				return attachToManagedSession(app, res.Session)
			}

		case "kill":
			if res.Session != nil {
				_ = app.Supervisor.KillSession(res.Session.ID)
			}
			continue

		case "new":
			// Prompt new background session spawn
			cwd, _ := os.Getwd()
			cfg, _ := app.CfgStore.Load()
			defaultProf := ""
			if cfg != nil {
				defaultProf = cfg.DefaultProfile
			}
			prof, err := resolveTargetProfile(app, defaultProf)
			if err != nil {
				return err
			}

			taskName := fmt.Sprintf("task-%d", len(sessions)+1)
			_, err = app.Supervisor.StartSession(supervisor.StartOptions{
				TaskName:   taskName,
				ProjectDir: cwd,
				Profile:    prof,
			})
			if err != nil {
				fmt.Fprintf(os.Stderr, "✗ Failed to spawn session: %v\n", err)
			}
			continue
		}

		return nil
	}
}

func attachToManagedSession(app *AppContext, sess *supervisor.ManagedSession) error {
	fmt.Printf("\n✓ Attaching to session %s (%s)...\n", sess.TaskName, sess.ID[:8])
	fmt.Printf("✓ Working Directory: %s\n\n", sess.ProjectDir)

	if info, err := os.Stat(sess.ProjectDir); err == nil && info.IsDir() {
		_ = os.Chdir(sess.ProjectDir)
	}

	p, err := resolveTargetProfile(app, sess.ProfileName)
	if err != nil {
		return err
	}

	cfg, _ := app.CfgStore.Load()
	shared := cfg != nil && cfg.SharedContext

	// Launch or resume Claude Code interactively
	args := []string{"--resume", sess.ID}
	code, err := app.Launcher.Launch(p, args, shared)
	if err != nil {
		// If --resume fails because session was not an internal ID, launch clean
		code, err = app.Launcher.Launch(p, nil, shared)
	}
	os.Exit(code)
	return nil
}

func printSupervisorList(app *AppContext) error {
	sessions, err := app.Supervisor.ListSessions()
	if err != nil {
		return err
	}

	if len(sessions) == 0 {
		fmt.Println("No background Claude sessions running.")
		fmt.Println("Spawn one using: claudeswap matrix spawn <task-name>")
		return nil
	}

	fmt.Println("ClaudeSwap Background Sessions Matrix")
	fmt.Println("=====================================")

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	fmt.Fprintln(w, "  #\tTASK\tPROJECT\tPID\tSTATUS\tPROFILE\tACTIVITY")
	fmt.Fprintln(w, "  -\t----\t-------\t---\t------\t-------\t--------")

	for i, s := range sessions {
		act := s.LastActivity
		if len(act) > 30 {
			act = act[:27] + "..."
		}
		fmt.Fprintf(w, "  %02d\t%s\t%s\t%d\t%s\t%s\t%s\n",
			i+1, s.TaskName, s.ProjectName, s.PID, s.Status, s.ProfileName, act)
	}
	w.Flush()
	fmt.Println()
	return nil
}
