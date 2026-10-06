package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func newSetDefaultCmd(app *AppContext) *cobra.Command {
	return &cobra.Command{
		Use:   "set-default <profile>",
		Short: "Set the default profile for launches",
		Long: `Sets the default profile to be launched when running 'claudeswap launch'
without arguments.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			target := args[0]
			p, err := app.Manager.GetProfile(target)
			if err != nil {
				fmt.Fprintf(os.Stderr, "✗ Profile %q was not found.\n", target)
				printAvailableProfiles(app)
				os.Exit(ExitProfileNotFound)
			}

			cfg, err := app.CfgStore.Load()
			if err != nil {
				fmt.Fprintf(os.Stderr, "✗ Error reading configuration: %v\n", err)
				os.Exit(ExitConfigError)
			}

			cfg.DefaultProfile = p.ID
			if err := app.CfgStore.Save(cfg); err != nil {
				fmt.Fprintf(os.Stderr, "✗ Failed to save configuration: %v\n", err)
				os.Exit(ExitConfigError)
			}

			fmt.Printf("✓ Default profile set to %q (%s).\n", p.Name, p.ID)
			return nil
		},
	}
}
