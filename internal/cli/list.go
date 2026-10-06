package cli

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/claudeswap/claudeswap/internal/profile"

	"github.com/spf13/cobra"
)

func newListCmd(app *AppContext) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all configured Claude profiles",
		RunE: func(cmd *cobra.Command, args []string) error {
			profiles, err := app.Manager.ListProfiles()
			if err != nil {
				fmt.Fprintf(os.Stderr, "✗ Error loading profiles: %v\n", err)
				os.Exit(ExitConfigError)
			}

			if len(profiles) == 0 {
				fmt.Println("No profiles configured yet.")
				fmt.Println("\nRun 'claudeswap add' to create your first profile.")
				return nil
			}

			fmt.Println("ClaudeSwap Profiles")
			fmt.Println()

			w := tabwriter.NewWriter(os.Stdout, 0, 0, 4, ' ', 0)
			fmt.Fprintln(w, "NAME\tSHORTCUT\tSTATUS")
			fmt.Fprintln(w, "----\t--------\t------")

			for _, p := range profiles {
				status := app.Manager.GetStatus(&p)
				var statusStr string
				switch status {
				case profile.StatusReady:
					statusStr = "✓ Ready"
				case profile.StatusLoginRequired:
					statusStr = "⚠ Login required"
				default:
					statusStr = "? Unknown"
				}
				fmt.Fprintf(w, "%s\t%s\t%s\n", p.Name, p.Shortcut, statusStr)
			}
			w.Flush()

			return nil
		},
	}
}
