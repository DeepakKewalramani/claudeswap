package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

// Version of ClaudeSwap
const Version = "1.0.0"

func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Display ClaudeSwap version",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("ClaudeSwap v%s\n", Version)
		},
	}
}
