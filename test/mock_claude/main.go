package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
)

func main() {
	args := os.Args[1:]

	// Version check
	if len(args) == 1 && (args[0] == "-v" || args[0] == "--version") {
		fmt.Println("claude-code-mock 1.0.0")
		return
	}

	configDir := os.Getenv("CLAUDE_CONFIG_DIR")
	fmt.Printf("MOCK_CLAUDE_STARTED\n")
	fmt.Printf("CLAUDE_CONFIG_DIR=%s\n", configDir)
	fmt.Printf("ARGS=%v\n", args)

	// Check if simulated login is requested
	for _, arg := range args {
		if arg == "--simulate-login" && configDir != "" {
			_ = os.MkdirAll(configDir, 0700)
			_ = os.WriteFile(filepath.Join(configDir, "config.json"), []byte(`{"mock":true}`), 0600)
			fmt.Printf("SIMULATED_LOGIN_SUCCESS\n")
		}
	}

	// Check if specific exit code is requested
	for i, arg := range args {
		if arg == "--exit-code" && i+1 < len(args) {
			if code, err := strconv.Atoi(args[i+1]); err == nil {
				os.Exit(code)
			}
		}
	}

	os.Exit(0)
}
