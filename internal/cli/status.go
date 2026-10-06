package cli

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/claudeswap/claudeswap/internal/profile"

	"github.com/spf13/cobra"
)

func newStatusCmd(app *AppContext) *cobra.Command {
	return &cobra.Command{
		Use:   "status [profile]",
		Short: "Show status of Claude profiles",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				p, err := app.Manager.GetProfile(args[0])
				if err != nil {
					fmt.Fprintf(os.Stderr, "✗ Profile %q was not found.\n", args[0])
					printAvailableProfiles(app)
					os.Exit(ExitProfileNotFound)
				}

				status := app.Manager.GetStatus(p)
				fmt.Printf("Profile:       %s\n", p.Name)
				fmt.Printf("ID:            %s\n", p.ID)
				fmt.Printf("Shortcut:      %s\n", p.Shortcut)
				fmt.Printf("Status:        %s\n", formatStatus(status))
				fmt.Printf("Config Dir:    %s\n", p.ConfigDir)
				return nil
			}

			profiles, err := app.Manager.ListProfiles()
			if err != nil {
				fmt.Fprintf(os.Stderr, "✗ Error loading profiles: %v\n", err)
				os.Exit(ExitConfigError)
			}

			if len(profiles) == 0 {
				fmt.Println("No profiles configured yet.")
				return nil
			}

			w := tabwriter.NewWriter(os.Stdout, 0, 0, 4, ' ', 0)
			fmt.Fprintln(w, "NAME\tID\tSHORTCUT\tSTATUS")
			fmt.Fprintln(w, "----\t--\t--------\t------")
			for _, p := range profiles {
				status := app.Manager.GetStatus(&p)
				fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", p.Name, p.ID, p.Shortcut, formatStatus(status))
			}
			w.Flush()
			return nil
		},
	}
}

func formatStatus(s profile.Status) string {
	switch s {
	case profile.StatusReady:
		return "✓ Ready"
	case profile.StatusLoginRequired:
		return "⚠ Login required"
	default:
		return "? Unknown"
	}
}

func printAvailableProfiles(app *AppContext) {
	profiles, _ := app.Manager.ListProfiles()
	if len(profiles) > 0 {
		fmt.Println("\nAvailable profiles:")
		for _, p := range profiles {
			fmt.Printf("  %s (%s)\n", p.ID, p.Name)
		}
	}
}
