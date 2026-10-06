package shortcut

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/claudeswap/claudeswap/internal/config"
	"github.com/claudeswap/claudeswap/internal/profile"
)

// Manager manages direct executable shortcuts in ~/.claudeswap/bin.
type Manager struct {
	paths *config.Paths
}

// NewManager creates a new shortcut Manager.
func NewManager(paths *config.Paths) *Manager {
	return &Manager{paths: paths}
}

// CreateShortcut creates or updates the direct shortcut launcher script for a profile.
func (m *Manager) CreateShortcut(p *profile.Profile) (string, error) {
	if p == nil {
		return "", fmt.Errorf("profile cannot be nil")
	}

	if err := m.paths.EnsureDirectories(); err != nil {
		return "", err
	}

	shortcutName := p.Shortcut
	if shortcutName == "" {
		shortcutName = profile.DefaultShortcut(p.ID)
	}

	targetPath := m.paths.ShortcutPath(shortcutName)
	bin := resolveBinary()

	var content string
	if runtime.GOOS == "windows" {
		if !strings.HasSuffix(targetPath, ".cmd") && !strings.HasSuffix(targetPath, ".bat") {
			targetPath += ".cmd"
		}
		// Windows CMD batch script forwarding all arguments (%*)
		content = fmt.Sprintf("@echo off\r\n\"%s\" launch %s %%*\r\n", bin, p.ID)
	} else {
		// Unix POSIX shell script forwarding all arguments ("$@")
		content = fmt.Sprintf("#!/usr/bin/env sh\nexec \"%s\" launch %s \"$@\"\n", bin, p.ID)
	}

	if err := os.WriteFile(targetPath, []byte(content), 0755); err != nil {
		return "", fmt.Errorf("failed to write shortcut at %q: %w", targetPath, err)
	}

	// Also opportunistically link into standard PATH locations if writable on Unix
	if runtime.GOOS != "windows" {
		candidates := []string{"/usr/local/bin"}
		if homeDir, err := os.UserHomeDir(); err == nil {
			candidates = append(candidates, filepath.Join(homeDir, ".local", "bin"))
		}
		for _, dir := range candidates {
			if info, err := os.Stat(dir); err == nil && info.IsDir() {
				linkPath := filepath.Join(dir, shortcutName)
				_ = os.Remove(linkPath)
				_ = os.Symlink(targetPath, linkPath)
			}
		}
	}

	return targetPath, nil
}

// RemoveShortcut deletes the shortcut launcher script.
func (m *Manager) RemoveShortcut(shortcutName string) error {
	if shortcutName == "" {
		return nil
	}

	targetPath := m.paths.ShortcutPath(shortcutName)
	_ = os.Remove(targetPath)

	if runtime.GOOS == "windows" {
		_ = os.Remove(targetPath + ".cmd")
		_ = os.Remove(targetPath + ".bat")
	} else {
		candidates := []string{"/usr/local/bin"}
		if homeDir, err := os.UserHomeDir(); err == nil {
			candidates = append(candidates, filepath.Join(homeDir, ".local", "bin"))
		}
		for _, dir := range candidates {
			_ = os.Remove(filepath.Join(dir, shortcutName))
		}
	}

	return nil
}

// RecreateShortcut removes old and creates new shortcut.
func (m *Manager) RecreateShortcut(p *profile.Profile, oldShortcut string) (string, error) {
	if oldShortcut != "" && oldShortcut != p.Shortcut {
		_ = m.RemoveShortcut(oldShortcut)
	}
	return m.CreateShortcut(p)
}

func resolveBinary() string {
	execPath, err := os.Executable()
	if err == nil {
		// Verify this is not a temporary test binary or go-run cache
		if !strings.Contains(execPath, "go-build") && !strings.Contains(execPath, "/tmp/") && !strings.Contains(execPath, "\\Temp\\") {
			return execPath
		}
	}
	return "claudeswap"
}
