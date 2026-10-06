package claude

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

var ErrClaudeNotFound = errors.New("Claude Code executable not found")

// Detector detects the Claude Code executable on the host system.
type Detector struct {
	configuredPath string
}

// NewDetector creates a new Claude detector.
func NewDetector(configuredPath string) *Detector {
	return &Detector{configuredPath: configuredPath}
}

// FindExecutable resolves the absolute path to the Claude Code CLI.
func (d *Detector) FindExecutable() (string, error) {
	// 1. Check environment variable override (CLAUDE_BIN)
	if envBin := os.Getenv("CLAUDE_BIN"); envBin != "" {
		if path, err := verifyExecutable(envBin); err == nil {
			return path, nil
		}
	}

	// 2. Check configured path from AppConfig
	if d.configuredPath != "" {
		if path, err := verifyExecutable(d.configuredPath); err == nil {
			return path, nil
		}
	}

	// 3. Search in system PATH
	binName := "claude"
	if runtime.GOOS == "windows" {
		binName = "claude.cmd"
	}
	if path, err := exec.LookPath(binName); err == nil {
		if abs, err := filepath.Abs(path); err == nil {
			return abs, nil
		}
		return path, nil
	}

	if runtime.GOOS == "windows" {
		// Also try claude.exe
		if path, err := exec.LookPath("claude.exe"); err == nil {
			return filepath.Abs(path)
		}
	}

	// 4. Check known standard locations
	homeDir, _ := os.UserHomeDir()
	candidatePaths := getStandardLocations(homeDir)
	for _, cand := range candidatePaths {
		if path, err := verifyExecutable(cand); err == nil {
			return path, nil
		}
	}

	return "", ErrClaudeNotFound
}

func getStandardLocations(homeDir string) []string {
	var candidates []string
	if homeDir != "" {
		if runtime.GOOS == "windows" {
			appData := os.Getenv("APPDATA")
			localAppData := os.Getenv("LOCALAPPDATA")
			if appData != "" {
				candidates = append(candidates, filepath.Join(appData, "npm", "claude.cmd"))
			}
			if localAppData != "" {
				candidates = append(candidates, filepath.Join(localAppData, "Programs", "claude", "claude.exe"))
			}
		} else {
			candidates = append(candidates,
				filepath.Join(homeDir, ".local", "bin", "claude"),
				filepath.Join(homeDir, ".npm-global", "bin", "claude"),
				filepath.Join(homeDir, ".cargo", "bin", "claude"),
				"/usr/local/bin/claude",
				"/opt/homebrew/bin/claude",
			)
		}
	}
	return candidates
}

func verifyExecutable(path string) (string, error) {
	cleanPath := strings.TrimSpace(path)
	if cleanPath == "" {
		return "", errors.New("empty path")
	}

	info, err := os.Stat(cleanPath)
	if err != nil {
		return "", err
	}
	if info.IsDir() {
		return "", fmt.Errorf("%q is a directory, not an executable", cleanPath)
	}

	// On Unix, ensure execute bits are set
	if runtime.GOOS != "windows" {
		if info.Mode()&0111 == 0 {
			return "", fmt.Errorf("%q is not executable", cleanPath)
		}
	}

	abs, err := filepath.Abs(cleanPath)
	if err != nil {
		return cleanPath, nil
	}
	return abs, nil
}

// GetVersion executes `claude -v` and returns the version string.
func (d *Detector) GetVersion(claudePath string) (string, error) {
	cmd := exec.Command(claudePath, "-v")
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}
