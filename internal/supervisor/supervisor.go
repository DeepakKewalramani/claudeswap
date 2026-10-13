package supervisor

import (
	"bufio"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/claudeswap/claudeswap/internal/claude"
	"github.com/claudeswap/claudeswap/internal/config"
	"github.com/claudeswap/claudeswap/internal/profile"
	"github.com/claudeswap/claudeswap/internal/session"
)

// StartOptions contains parameters for launching a supervised background Claude session.
type StartOptions struct {
	TaskName     string
	ProjectDir   string
	Profile      *profile.Profile
	InitialQuery string
	SharedMemory bool
}

// Supervisor manages a matrix of background Claude Code processes.
type Supervisor struct {
	paths    *config.Paths
	detector *claude.Detector
	sessMgr  *session.Manager
	stateDir string
	mu       sync.Mutex
}

// NewSupervisor initializes a background session supervisor.
func NewSupervisor(paths *config.Paths, detector *claude.Detector, sessMgr *session.Manager) *Supervisor {
	baseDir := paths.HomeDir()
	stateDir := filepath.Join(baseDir, "supervisor")
	_ = os.MkdirAll(filepath.Join(stateDir, "logs"), 0700)

	return &Supervisor{
		paths:    paths,
		detector: detector,
		sessMgr:  sessMgr,
		stateDir: stateDir,
	}
}

func (s *Supervisor) registryPath() string {
	return filepath.Join(s.stateDir, "sessions.json")
}

func (s *Supervisor) logPath(sessionID string) string {
	return filepath.Join(s.stateDir, "logs", sessionID+".log")
}

// ListSessions returns all tracked background sessions with updated live status.
func (s *Supervisor) ListSessions() ([]ManagedSession, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	reg, err := s.loadRegistry()
	if err != nil {
		reg = &Registry{}
	}

	// Update live status for each session
	changed := false
	now := time.Now()

	for i := range reg.Sessions {
		sess := &reg.Sessions[i]
		if sess.PID > 0 && (sess.Status == StateRunning || sess.Status == StateWaitingInput) {
			alive := isProcessAlive(sess.PID)
			if !alive {
				sess.Status = StateCompleted
				sess.UpdatedAt = now
				changed = true
			} else {
				// Process is alive, check latest transcript activity to distinguish running vs waiting
				activity, waiting := s.inspectProcessActivity(sess)
				if activity != "" {
					sess.LastActivity = activity
				}
				if waiting {
					sess.Status = StateWaitingInput
				} else {
					sess.Status = StateRunning
				}
				sess.UpdatedAt = now
			}
		}
	}

	if changed {
		_ = s.saveRegistry(reg)
	}

	return reg.Sessions, nil
}

// StartSession launches a new background Claude Code process managed by the supervisor.
func (s *Supervisor) StartSession(opts StartOptions) (*ManagedSession, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if opts.Profile == nil {
		return nil, fmt.Errorf("profile cannot be nil")
	}

	claudePath, err := s.detector.FindExecutable()
	if err != nil {
		return nil, fmt.Errorf("claude executable not found: %w", err)
	}

	sessionID := generateShortUUID()
	logFile := s.logPath(sessionID)

	lf, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
	if err != nil {
		return nil, fmt.Errorf("failed to open session log: %w", err)
	}

	// Build arguments
	var args []string
	if opts.InitialQuery != "" {
		args = append(args, opts.InitialQuery)
	}

	cmd := exec.Command(claudePath, args...)
	cmd.Dir = opts.ProjectDir
	cmd.Env = claude.BuildEnvironment(opts.Profile.ConfigDir)
	cmd.Stdout = lf
	cmd.Stderr = lf

	// Detach child process from current process group so it lives independently
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true,
	}

	if err := cmd.Start(); err != nil {
		_ = lf.Close()
		return nil, fmt.Errorf("failed to spawn background session: %w", err)
	}

	taskName := opts.TaskName
	if taskName == "" {
		taskName = "task-" + sessionID[:6]
	}

	projName := filepath.Base(opts.ProjectDir)
	if projName == "" || projName == "." {
		projName = "workspace"
	}

	managed := ManagedSession{
		ID:           sessionID,
		TaskName:     taskName,
		ProjectName:  projName,
		ProjectDir:   opts.ProjectDir,
		ProfileName:  opts.Profile.Name,
		PID:          cmd.Process.Pid,
		Status:       StateRunning,
		LastActivity: "Initializing session...",
		InitialQuery: opts.InitialQuery,
		LogPath:      logFile,
		StartedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	// Monitor child in background goroutine to record exit
	go func(pid int, id string) {
		_ = cmd.Wait()
		_ = lf.Close()
		s.markSessionCompleted(id, cmd.ProcessState.ExitCode())
	}(cmd.Process.Pid, sessionID)

	// Save to registry
	reg, _ := s.loadRegistry()
	if reg == nil {
		reg = &Registry{}
	}
	reg.Sessions = append([]ManagedSession{managed}, reg.Sessions...)
	_ = s.saveRegistry(reg)

	return &managed, nil
}

// KillSession terminates a running background session.
func (s *Supervisor) KillSession(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	reg, err := s.loadRegistry()
	if err != nil {
		return err
	}

	for i := range reg.Sessions {
		sess := &reg.Sessions[i]
		if sess.ID == id {
			if sess.PID > 0 && isProcessAlive(sess.PID) {
				_ = syscall.Kill(-sess.PID, syscall.SIGTERM)
				time.Sleep(50 * time.Millisecond)
				if isProcessAlive(sess.PID) {
					_ = syscall.Kill(-sess.PID, syscall.SIGKILL)
				}
			}
			sess.Status = StateStopped
			sess.UpdatedAt = time.Now()
			return s.saveRegistry(reg)
		}
	}

	return fmt.Errorf("session %q not found", id)
}

// GetLogs returns the most recent lines from the session's log file.
func (s *Supervisor) GetLogs(sessionID string, tailLines int) ([]string, error) {
	logFile := s.logPath(sessionID)
	f, err := os.Open(logFile)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{"(No log output recorded yet)"}, nil
		}
		return nil, err
	}
	defer f.Close()

	var lines []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	if len(lines) <= tailLines {
		return lines, nil
	}
	return lines[len(lines)-tailLines:], nil
}

func (s *Supervisor) markSessionCompleted(id string, exitCode int) {
	s.mu.Lock()
	defer s.mu.Unlock()

	reg, err := s.loadRegistry()
	if err != nil {
		return
	}

	for i := range reg.Sessions {
		sess := &reg.Sessions[i]
		if sess.ID == id {
			if exitCode == 0 {
				sess.Status = StateCompleted
			} else {
				sess.Status = StateFailed
			}
			sess.ExitCode = exitCode
			sess.UpdatedAt = time.Now()
			_ = s.saveRegistry(reg)
			break
		}
	}
}

func (s *Supervisor) inspectProcessActivity(sess *ManagedSession) (string, bool) {
	// 1. Inspect recent tail of the session log
	logFile := sess.LogPath
	if logFile == "" {
		logFile = s.logPath(sess.ID)
	}

	f, err := os.Open(logFile)
	if err != nil {
		return "", false
	}
	defer f.Close()

	// Read last 4KB
	stat, err := f.Stat()
	if err != nil || stat.Size() == 0 {
		return "Running in background", false
	}

	offset := int64(0)
	if stat.Size() > 4096 {
		offset = stat.Size() - 4096
	}
	_, _ = f.Seek(offset, io.SeekStart)

	content, err := io.ReadAll(f)
	if err != nil || len(content) == 0 {
		return "Active", false
	}

	text := string(content)
	lines := strings.Split(strings.TrimSpace(text), "\n")
	if len(lines) == 0 {
		return "Active", false
	}

	lastLine := strings.TrimSpace(lines[len(lines)-1])

	// Check if Claude is waiting for user prompt or permission
	isWaiting := false
	if strings.Contains(text, "?") || strings.Contains(text, "[y/n]") || strings.Contains(text, "Allow") || strings.Contains(text, "press enter") {
		isWaiting = true
	}

	if len(lastLine) > 50 {
		lastLine = lastLine[:47] + "..."
	}

	return lastLine, isWaiting
}

func (s *Supervisor) loadRegistry() (*Registry, error) {
	p := s.registryPath()
	data, err := os.ReadFile(p)
	if err != nil {
		return nil, err
	}

	var reg Registry
	if err := json.Unmarshal(data, &reg); err != nil {
		return nil, err
	}
	return &reg, nil
}

func (s *Supervisor) saveRegistry(reg *Registry) error {
	data, err := json.MarshalIndent(reg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.registryPath(), data, 0600)
}

func isProcessAlive(pid int) bool {
	if pid <= 0 {
		return false
	}
	process, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	// Signal 0 verifies existence without killing
	err = process.Signal(syscall.Signal(0))
	return err == nil
}

func generateShortUUID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
