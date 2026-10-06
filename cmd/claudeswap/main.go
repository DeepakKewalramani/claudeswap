package main

import (
	"fmt"
	"os"

	"github.com/claudeswap/claudeswap/internal/cli"
)

func main() {
	app, err := cli.NewAppContext()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Fatal initialization error: %v\n", err)
		os.Exit(cli.ExitGeneralError)
	}

	rootCmd := cli.NewRootCmd(app)
	if err := rootCmd.Execute(); err != nil {
		os.Exit(cli.ExitGeneralError)
	}
}
