package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

func newRenameCmd(app *AppContext) *cobra.Command {
	var shortcutFlag string

	cmd := &cobra.Command{
		Use:   "rename <profile> <new-name>",
		Short: "Rename a Claude profile",
		Long: `Update a profile's display name and optionally its shortcut command.
The underlying Claude configuration directory remains preserved.`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			target := args[0]
			newName := strings.TrimSpace(args[1])

			p, err := app.Manager.GetProfile(target)
			if err != nil {
				fmt.Fprintf(os.Stderr, "✗ Profile %q was not found.\n", target)
				printAvailableProfiles(app)
				os.Exit(ExitProfileNotFound)
			}

			oldShortcut := p.Shortcut
			updated, err := app.Manager.RenameProfile(p.ID, newName, shortcutFlag)
			if err != nil {
				fmt.Fprintf(os.Stderr, "✗ Error renaming profile: %v\n", err)
				os.Exit(ExitGeneralError)
			}

			// Recreate shortcut if changed
			if updated.Shortcut != oldShortcut {
				_, err = app.Shortcut.RecreateShortcut(updated, oldShortcut)
				if err != nil {
					fmt.Fprintf(os.Stderr, "⚠ Warning: Could not update shortcut: %v\n", err)
				} else {
					fmt.Printf("✓ Shortcut updated: %s -> %s\n", oldShortcut, updated.Shortcut)
				}
			}

			fmt.Printf("✓ Profile successfully renamed to %q\n", updated.Name)
			fmt.Printf("Shortcut: %s\n", updated.Shortcut)
			return nil
		},
	}

	cmd.Flags().StringVarP(&shortcutFlag, "shortcut", "s", "", "New shortcut command name")

	return cmd
}
