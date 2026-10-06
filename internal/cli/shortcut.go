package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func newShortcutCmd(app *AppContext) *cobra.Command {
	var allFlag bool

	cmd := &cobra.Command{
		Use:   "shortcut [profile]",
		Short: "Create or recreate direct profile launcher shortcuts",
		Long: `Create or recreate executable shortcuts in ~/.claudeswap/bin.
Direct shortcuts allow launching profiles directly from your shell
(e.g., 'claudeswap-deepak', 'claudeswap-rohit').`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if allFlag {
				profiles, err := app.Manager.ListProfiles()
				if err != nil {
					fmt.Fprintf(os.Stderr, "✗ Error reading profiles: %v\n", err)
					os.Exit(ExitConfigError)
				}
				if len(profiles) == 0 {
					fmt.Println("No profiles to create shortcuts for.")
					return nil
				}
				for _, p := range profiles {
					path, err := app.Shortcut.CreateShortcut(&p)
					if err != nil {
						fmt.Fprintf(os.Stderr, "✗ Failed creating shortcut for %s: %v\n", p.Name, err)
					} else {
						fmt.Printf("✓ Created shortcut %s -> %s\n", p.Shortcut, path)
					}
				}
				return nil
			}

			if len(args) == 0 {
				fmt.Fprintln(os.Stderr, "✗ Please specify a profile or use --all.")
				printAvailableProfiles(app)
				os.Exit(ExitInvalidUsage)
			}

			target := args[0]
			p, err := app.Manager.GetProfile(target)
			if err != nil {
				fmt.Fprintf(os.Stderr, "✗ Profile %q was not found.\n", target)
				printAvailableProfiles(app)
				os.Exit(ExitProfileNotFound)
			}

			path, err := app.Shortcut.CreateShortcut(p)
			if err != nil {
				fmt.Fprintf(os.Stderr, "✗ Failed creating shortcut: %v\n", err)
				os.Exit(ExitShortcutError)
			}

			fmt.Printf("✓ Shortcut created for %s:\n  Command: %s\n  Path:    %s\n", p.Name, p.Shortcut, path)
			return nil
		},
	}

	cmd.Flags().BoolVarP(&allFlag, "all", "a", false, "Recreate shortcuts for all configured profiles")

	return cmd
}
