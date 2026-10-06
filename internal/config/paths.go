package config

import (
	"fmt"
	"os"
	"path/filepath"
)

// DefaultDirName is the base directory name inside the user's home directory.
const DefaultDirName = ".claudeswap"

// Paths manages all filesystem paths for ClaudeSwap.
type Paths struct {
	homeDir string
}

// NewPaths creates a new Paths instance, resolving the base directory.
// It checks CLAUDESWAP_HOME environment variable first, defaulting to ~/.claudeswap.
func NewPaths() (*Paths, error) {
	if custom := os.Getenv("CLAUDESWAP_HOME"); custom != "" {
		abs, err := filepath.Abs(custom)
		if err != nil {
			return nil, fmt.Errorf("invalid CLAUDESWAP_HOME path %q: %w", custom, err)
		}
		return &Paths{homeDir: abs}, nil
	}

	userHome, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("unable to determine user home directory: %w", err)
	}

	return &Paths{homeDir: filepath.Join(userHome, DefaultDirName)}, nil
}

// NewPathsWithBase creates a Paths instance using an explicit base directory (primarily for testing).
func NewPathsWithBase(baseDir string) *Paths {
	return &Paths{homeDir: baseDir}
}

// HomeDir returns the root ClaudeSwap directory (e.g., ~/.claudeswap).
func (p *Paths) HomeDir() string {
	return p.homeDir
}

// ConfigFile returns the path to config.json.
func (p *Paths) ConfigFile() string {
	return filepath.Join(p.homeDir, "config.json")
}

// ConfigBakFile returns the path to backups/config.json.bak.
func (p *Paths) ConfigBakFile() string {
	return filepath.Join(p.BackupsDir(), "config.json.bak")
}

// ProfilesFile returns the path to profiles.json.
func (p *Paths) ProfilesFile() string {
	return filepath.Join(p.homeDir, "profiles.json")
}

// ProfilesBakFile returns the path to backups/profiles.json.bak.
func (p *Paths) ProfilesBakFile() string {
	return filepath.Join(p.BackupsDir(), "profiles.json.bak")
}

// ProfilesDir returns the path to the profiles directory (e.g., ~/.claudeswap/profiles).
func (p *Paths) ProfilesDir() string {
	return filepath.Join(p.homeDir, "profiles")
}

// ProfileConfigDir returns the isolated Claude configuration directory for a profile.
func (p *Paths) ProfileConfigDir(profileID string) string {
	return filepath.Join(p.ProfilesDir(), profileID)
}

// BinDir returns the path to the shortcut launcher scripts directory (e.g., ~/.claudeswap/bin).
func (p *Paths) BinDir() string {
	return filepath.Join(p.homeDir, "bin")
}

// LogsDir returns the path to the logs directory.
func (p *Paths) LogsDir() string {
	return filepath.Join(p.homeDir, "logs")
}

// SharedDir returns the path to the shared context directory (e.g., ~/.claudeswap/shared).
func (p *Paths) SharedDir() string {
	return filepath.Join(p.homeDir, "shared")
}

// BackupsDir returns the path to the backups directory.
func (p *Paths) BackupsDir() string {
	return filepath.Join(p.homeDir, "backups")
}

// ShortcutPath returns the absolute path for a direct shortcut script/binary.
func (p *Paths) ShortcutPath(shortcutName string) string {
	return filepath.Join(p.BinDir(), shortcutName)
}

// EnsureDirectories ensures that all required directories exist with safe permissions.
func (p *Paths) EnsureDirectories() error {
	// Standard directories with user-only permissions (0700)
	dirs0700 := []string{
		p.homeDir,
		p.ProfilesDir(),
		p.SharedDir(),
		p.LogsDir(),
		p.BackupsDir(),
	}

	for _, dir := range dirs0700 {
		if err := os.MkdirAll(dir, 0700); err != nil {
			return fmt.Errorf("failed to create directory %q: %w", dir, err)
		}
		_ = os.Chmod(dir, 0700)
	}

	// Bin directory may need 0755 so shortcuts can be executed
	if err := os.MkdirAll(p.BinDir(), 0755); err != nil {
		return fmt.Errorf("failed to create bin directory %q: %w", p.BinDir(), err)
	}

	return nil
}
