package shortcut

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/claudeswap/claudeswap/internal/config"
)

// ShellType represents the detected shell.
type ShellType string

const (
	ShellZsh        ShellType = "zsh"
	ShellBash       ShellType = "bash"
	ShellFish       ShellType = "fish"
	ShellPowerShell ShellType = "powershell"
	ShellUnknown    ShellType = "unknown"
)

// ShellInfo holds diagnostics about shell integration.
type ShellInfo struct {
	ShellType      ShellType
	BinDirInPath   bool
	ConfigFile     string
	AlreadyInFile  bool
	IntegrationLine string
}

// DetectShell detects the user's active shell and checks if ~/.claudeswap/bin is in PATH.
func DetectShell(paths *config.Paths) ShellInfo {
	info := ShellInfo{
		ShellType:    ShellUnknown,
		BinDirInPath: isBinInPATH(paths.BinDir()),
	}

	shellEnv := os.Getenv("SHELL")
	homeDir, _ := os.UserHomeDir()

	if runtime.GOOS == "windows" {
		info.ShellType = ShellPowerShell
		info.IntegrationLine = fmt.Sprintf(`$env:PATH = "%s;" + $env:PATH`, paths.BinDir())
		return info
	}

	switch {
	case strings.Contains(shellEnv, "zsh"):
		info.ShellType = ShellZsh
		info.ConfigFile = filepath.Join(homeDir, ".zshrc")
		info.IntegrationLine = fmt.Sprintf(`export PATH="%s:$PATH"`, paths.BinDir())
	case strings.Contains(shellEnv, "bash"):
		info.ShellType = ShellBash
		info.ConfigFile = filepath.Join(homeDir, ".bashrc")
		if runtime.GOOS == "darwin" {
			// On macOS, bash login shells typically use ~/.bash_profile
			profilePath := filepath.Join(homeDir, ".bash_profile")
			if _, err := os.Stat(profilePath); err == nil {
				info.ConfigFile = profilePath
			}
		}
		info.IntegrationLine = fmt.Sprintf(`export PATH="%s:$PATH"`, paths.BinDir())
	case strings.Contains(shellEnv, "fish"):
		info.ShellType = ShellFish
		info.ConfigFile = filepath.Join(homeDir, ".config", "fish", "config.fish")
		info.IntegrationLine = fmt.Sprintf(`fish_add_path "%s"`, paths.BinDir())
	default:
		info.ConfigFile = filepath.Join(homeDir, ".profile")
		info.IntegrationLine = fmt.Sprintf(`export PATH="%s:$PATH"`, paths.BinDir())
	}

	if info.ConfigFile != "" {
		if content, err := os.ReadFile(info.ConfigFile); err == nil {
			info.AlreadyInFile = strings.Contains(string(content), paths.BinDir())
		}
	}

	return info
}

func isBinInPATH(binDir string) bool {
	pathEnv := os.Getenv("PATH")
	for _, p := range filepath.SplitList(pathEnv) {
		cleanP := filepath.Clean(p)
		if cleanP == filepath.Clean(binDir) {
			return true
		}
	}
	return false
}

// InstallShellIntegration safely adds the ClaudeSwap bin directory to the user's shell config.
// It creates a backup before editing and ensures no duplicate lines are added.
func InstallShellIntegration(paths *config.Paths) (string, error) {
	info := DetectShell(paths)
	if info.BinDirInPath && info.AlreadyInFile {
		return "PATH already includes ClaudeSwap bin directory", nil
	}

	if info.ConfigFile == "" {
		return "", fmt.Errorf("could not determine shell config file for shell %q", info.ShellType)
	}

	// Read existing content
	var existing string
	if data, err := os.ReadFile(info.ConfigFile); err == nil {
		existing = string(data)
		if strings.Contains(existing, paths.BinDir()) {
			return "ClaudeSwap path is already present in " + info.ConfigFile, nil
		}

		// Backup the original config file
		bakFile := info.ConfigFile + ".claudeswap.bak"
		_ = os.WriteFile(bakFile, data, 0600)
	}

	block := fmt.Sprintf("\n# Added by ClaudeSwap\n%s\n", info.IntegrationLine)

	f, err := os.OpenFile(info.ConfigFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
	if err != nil {
		return "", fmt.Errorf("failed to open %s for writing: %w", info.ConfigFile, err)
	}
	defer f.Close()

	if _, err := f.WriteString(block); err != nil {
		return "", fmt.Errorf("failed to append to %s: %w", info.ConfigFile, err)
	}

	return fmt.Sprintf("Successfully added ClaudeSwap to %s (backup at %s.claudeswap.bak)", info.ConfigFile, info.ConfigFile), nil
}
