package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"text/tabwriter"
	"time"

	"github.com/claudeswap/claudeswap/internal/chat"

	"github.com/spf13/cobra"
)

func newChatCmd(app *AppContext) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "chat",
		Short: "Export, import, and inspect Claude Code chat sessions",
		Long: `Manage chat history and sessions.
Export conversations to a portable archive (.tar.gz) or readable Markdown (.md),
or import previously exported sessions into any profile or shared context.`,
	}

	cmd.AddCommand(
		newChatExportCmd(app),
		newChatImportCmd(app),
		newChatListCmd(app),
	)

	return cmd
}

func newChatExportCmd(app *AppContext) *cobra.Command {
	var outputPath string
	var format string

	cmd := &cobra.Command{
		Use:   "export [profile]",
		Short: "Export chat sessions to an archive or Markdown files",
		Long: `Export conversation transcripts and project memory into a compressed
archive (.tar.gz) or readable Markdown files (.md).
If no profile is specified, sessions from the active shared memory or default profile are used.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			srcDir, label, err := resolveChatDir(app, args)
			if err != nil {
				return err
			}

			if format == "md" || format == "markdown" {
				if outputPath == "" {
					outputPath = fmt.Sprintf("claudeswap-chat-export-%s", time.Now().Format("20060102-150405"))
				}
				count, err := chat.ExportSessionsToMarkdown(srcDir, outputPath)
				if err != nil {
					fmt.Fprintf(os.Stderr, "✗ Export failed: %v\n", err)
					os.Exit(ExitGeneralError)
				}
				fmt.Printf("✓ Exported %d chat sessions as Markdown to: %s/\n", count, outputPath)
				return nil
			}

			// Default: tar.gz archive
			if outputPath == "" {
				outputPath = fmt.Sprintf("claudeswap-chat-%s-%s.tar.gz", label, time.Now().Format("20060102-150405"))
			}

			err = chat.ExportArchive(srcDir, outputPath)
			if err != nil {
				fmt.Fprintf(os.Stderr, "✗ Export failed: %v\n", err)
				os.Exit(ExitGeneralError)
			}

			absPath, _ := filepath.Abs(outputPath)
			fmt.Printf("✓ Successfully exported chat sessions from %s to:\n  %s\n", label, absPath)
			return nil
		},
	}

	cmd.Flags().StringVarP(&outputPath, "output", "o", "", "Destination file or folder for export")
	cmd.Flags().StringVarP(&format, "format", "f", "tar", "Export format: 'tar' (.tar.gz archive) or 'md' (Markdown files)")

	return cmd
}

func newChatImportCmd(app *AppContext) *cobra.Command {
	var targetProfile string

	cmd := &cobra.Command{
		Use:   "import <archive-file>",
		Short: "Import chat sessions from an archive (.tar.gz)",
		Long: `Extract conversation sessions into a profile or shared memory.
Once imported, Claude Code's --resume or 'claudeswap handoff' can resume the conversations directly.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			archivePath := args[0]
			if _, err := os.Stat(archivePath); err != nil {
				fmt.Fprintf(os.Stderr, "✗ Archive file %q not found.\n", archivePath)
				os.Exit(ExitInvalidUsage)
			}

			var destDir string
			var label string

			if targetProfile != "" {
				p, err := app.Manager.GetProfile(targetProfile)
				if err != nil {
					fmt.Fprintf(os.Stderr, "✗ Profile %q was not found.\n", targetProfile)
					os.Exit(ExitProfileNotFound)
				}
				destDir = p.ConfigDir
				label = fmt.Sprintf("profile %q", p.Name)
			} else {
				cfg, _ := app.CfgStore.Load()
				if cfg != nil && cfg.SharedContext {
					destDir = app.Paths.SharedDir()
					label = "shared context (all profiles)"
				} else if cfg != nil && cfg.DefaultProfile != "" {
					p, _ := app.Manager.GetProfile(cfg.DefaultProfile)
					if p != nil {
						destDir = p.ConfigDir
						label = fmt.Sprintf("default profile %q", p.Name)
					}
				}
				if destDir == "" {
					destDir = app.Paths.SharedDir()
					label = "shared directory"
				}
			}

			count, err := chat.ImportArchive(archivePath, destDir)
			if err != nil {
				fmt.Fprintf(os.Stderr, "✗ Import failed: %v\n", err)
				os.Exit(ExitGeneralError)
			}

			fmt.Printf("✓ Successfully imported %d session files into %s.\n", count, label)
			fmt.Println("\nTo resume an imported session, run:")
			fmt.Println("  claudeswap launch <profile> --resume")
			return nil
		},
	}

	cmd.Flags().StringVarP(&targetProfile, "profile", "p", "", "Target profile for imported chats (defaults to shared memory)")

	return cmd
}

func newChatListCmd(app *AppContext) *cobra.Command {
	return &cobra.Command{
		Use:   "list [profile]",
		Short: "List discovered chat sessions",
		RunE: func(cmd *cobra.Command, args []string) error {
			srcDir, label, err := resolveChatDir(app, args)
			if err != nil {
				return err
			}

			sessions, err := chat.FindSessions(srcDir)
			if err != nil {
				fmt.Fprintf(os.Stderr, "✗ Error scanning sessions: %v\n", err)
				os.Exit(ExitGeneralError)
			}

			if len(sessions) == 0 {
				fmt.Printf("No chat sessions found in %s.\n", label)
				return nil
			}

			fmt.Printf("Claude Code Chat Sessions (%s)\n\n", label)

			w := tabwriter.NewWriter(os.Stdout, 0, 0, 4, ' ', 0)
			fmt.Fprintln(w, "SESSION ID\tPROJECT\tSIZE\tLAST MODIFIED")
			fmt.Fprintln(w, "----------\t-------\t----\t-------------")

			for _, s := range sessions {
				sizeStr := formatBytes(s.SizeBytes)
				fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", s.ID, s.Project, sizeStr, s.ModTime)
			}
			w.Flush()

			return nil
		},
	}
}

func resolveChatDir(app *AppContext, args []string) (string, string, error) {
	if len(args) > 0 {
		p, err := app.Manager.GetProfile(args[0])
		if err != nil {
			fmt.Fprintf(os.Stderr, "✗ Profile %q was not found.\n", args[0])
			return "", "", err
		}
		return p.ConfigDir, p.ID, nil
	}

	cfg, _ := app.CfgStore.Load()
	if cfg != nil && cfg.SharedContext {
		return app.Paths.SharedDir(), "shared", nil
	}

	if cfg != nil && cfg.DefaultProfile != "" {
		p, err := app.Manager.GetProfile(cfg.DefaultProfile)
		if err == nil {
			return p.ConfigDir, p.ID, nil
		}
	}

	// Fallback to shared dir or first profile
	profiles, _ := app.Manager.ListProfiles()
	if len(profiles) > 0 {
		return profiles[0].ConfigDir, profiles[0].ID, nil
	}

	return app.Paths.SharedDir(), "shared", nil
}

func formatBytes(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(b)/float64(div), "KMGTPE"[exp])
}
