package claude

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"syscall"

	"github.com/claudeswap/claudeswap/internal/config"
	"github.com/claudeswap/claudeswap/internal/profile"
)

// Launcher executes Claude Code with isolated profile environments.
type Launcher struct {
	detector *Detector
	paths    *config.Paths
}

// NewLauncher creates a new Launcher.
func NewLauncher(detector *Detector, paths *config.Paths) *Launcher {
	return &Launcher{
		detector: detector,
		paths:    paths,
	}
}

// Launch executes Claude Code for the specified profile, forwarding all arguments and terminal streams.
// Returns the exit code of the Claude Code process.
func (l *Launcher) Launch(p *profile.Profile, args []string, sharedContext bool) (int, error) {
	if p == nil {
		return 1, errors.New("cannot launch nil profile")
	}

	claudePath, err := l.detector.FindExecutable()
	if err != nil {
		return 4, fmt.Errorf("%w: please install Claude Code and ensure it is in your PATH or configured", ErrClaudeNotFound)
	}

	// Ensure profile config directory exists with secure user-only permissions (0700)
	if err := os.MkdirAll(p.ConfigDir, 0700); err != nil {
		return 5, fmt.Errorf("failed to initialize profile directory %q: %w", p.ConfigDir, err)
	}

	// Synchronize shared session/project context if enabled
	shouldShare := sharedContext
	if p.SharedContext != nil {
		shouldShare = *p.SharedContext
	}
	if l.paths != nil {
		_ = SyncSharedContext(l.paths, p.ConfigDir, shouldShare)
	}

	cmd := exec.Command(claudePath, args...)
	cmd.Env = BuildEnvironment(p.ConfigDir)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Setup signal forwarding so Ctrl+C or terminal resize gracefully reaches Claude Code
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM, syscall.SIGHUP)
	defer signal.Stop(sigChan)

	if err := cmd.Start(); err != nil {
		return 1, fmt.Errorf("failed to start Claude Code: %w", err)
	}

	// Forward signals to child process
	go func() {
		for sig := range sigChan {
			if cmd.Process != nil {
				_ = cmd.Process.Signal(sig)
			}
		}
	}()

	err = cmd.Wait()
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return exitErr.ExitCode(), nil
		}
		return 1, err
	}

	return 0, nil
}
