package claude

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/claudeswap/claudeswap/internal/config"
	"github.com/claudeswap/claudeswap/internal/profile"
)

func TestLauncher_WithMockClaude(t *testing.T) {
	tempDir := t.TempDir()
	paths := config.NewPathsWithBase(tempDir)

	// Build mock Claude binary
	mockBinName := "mock-claude"
	if runtime.GOOS == "windows" {
		mockBinName += ".exe"
	}
	mockBinPath := filepath.Join(tempDir, mockBinName)
	buildCmd := exec.Command("go", "build", "-o", mockBinPath, "../../test/mock_claude/main.go")
	out, err := buildCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to build mock claude: %v (output: %s)", err, string(out))
	}

	detector := NewDetector(mockBinPath)
	launcher := NewLauncher(detector, paths)

	profileDir := filepath.Join(tempDir, "profiles", "deepak")
	p := &profile.Profile{
		ID:        "deepak",
		Name:      "Deepak",
		Shortcut:  "claudeswap-deepak",
		ConfigDir: profileDir,
	}

	// Test 1: Successful launch with arguments forwarding and simulated login
	exitCode, err := launcher.Launch(p, []string{"--continue", "--simulate-login"}, false)
	if err != nil {
		t.Fatalf("Launch failed: %v", err)
	}
	if exitCode != 0 {
		t.Errorf("Expected exit code 0, got %d", exitCode)
	}

	// Verify profile config directory exists
	if info, err := os.Stat(profileDir); err != nil || !info.IsDir() {
		t.Errorf("Expected profile config directory to be created: %v", err)
	}

	// Verify simulated login created config.json
	cfgFile := filepath.Join(profileDir, "config.json")
	if _, err := os.Stat(cfgFile); err != nil {
		t.Errorf("Expected simulated login config file to exist: %v", err)
	}

	// Test 2: Process exit code forwarding
	exitCode, err = launcher.Launch(p, []string{"--exit-code", "42"}, false)
	if err != nil {
		t.Fatalf("Launch unexpectedly returned Go error: %v", err)
	}
	if exitCode != 42 {
		t.Errorf("Expected exit code 42, got %d", exitCode)
	}
}

func TestLauncher_ConcurrentProfiles(t *testing.T) {
	tempDir := t.TempDir()
	paths := config.NewPathsWithBase(tempDir)

	mockBinPath := filepath.Join(tempDir, "mock-claude")
	buildCmd := exec.Command("go", "build", "-o", mockBinPath, "../../test/mock_claude/main.go")
	if out, err := buildCmd.CombinedOutput(); err != nil {
		t.Fatalf("Failed to build mock claude: %v (%s)", err, string(out))
	}

	detector := NewDetector(mockBinPath)
	launcher := NewLauncher(detector, paths)

	p1 := &profile.Profile{
		ID:        "deepak",
		Name:      "Deepak",
		Shortcut:  "claudeswap-deepak",
		ConfigDir: filepath.Join(tempDir, "profiles", "deepak"),
	}
	p2 := &profile.Profile{
		ID:        "rohit",
		Name:      "Rohit",
		Shortcut:  "claudeswap-rohit",
		ConfigDir: filepath.Join(tempDir, "profiles", "rohit"),
	}

	done := make(chan bool, 2)
	errChan := make(chan error, 2)

	go func() {
		code, err := launcher.Launch(p1, []string{"--continue"}, false)
		if err != nil {
			errChan <- err
			return
		}
		if code != 0 {
			errChan <- os.ErrInvalid
			return
		}
		done <- true
	}()

	go func() {
		code, err := launcher.Launch(p2, []string{"--resume"}, false)
		if err != nil {
			errChan <- err
			return
		}
		if code != 0 {
			errChan <- os.ErrInvalid
			return
		}
		done <- true
	}()

	for i := 0; i < 2; i++ {
		select {
		case <-done:
		case err := <-errChan:
			t.Fatalf("Concurrent launch failed: %v", err)
		}
	}
}

func TestLauncher_SharedContext(t *testing.T) {
	tempDir := t.TempDir()
	paths := config.NewPathsWithBase(tempDir)

	mockBinPath := filepath.Join(tempDir, "mock-claude")
	buildCmd := exec.Command("go", "build", "-o", mockBinPath, "../../test/mock_claude/main.go")
	if out, err := buildCmd.CombinedOutput(); err != nil {
		t.Fatalf("Failed to build mock claude: %v (%s)", err, string(out))
	}

	detector := NewDetector(mockBinPath)
	launcher := NewLauncher(detector, paths)

	p1 := &profile.Profile{
		ID:        "deepak",
		Name:      "Deepak",
		Shortcut:  "claudeswap-deepak",
		ConfigDir: filepath.Join(tempDir, "profiles", "deepak"),
	}
	p2 := &profile.Profile{
		ID:        "rohit",
		Name:      "Rohit",
		Shortcut:  "claudeswap-rohit",
		ConfigDir: filepath.Join(tempDir, "profiles", "rohit"),
	}

	// Launch Deepak with shared context
	_, err := launcher.Launch(p1, []string{"--simulate-login"}, true)
	if err != nil {
		t.Fatalf("Launch p1 failed: %v", err)
	}

	// Verify sessions directory is a symlink to shared
	p1Sessions := filepath.Join(p1.ConfigDir, "sessions")
	fi, err := os.Lstat(p1Sessions)
	if err != nil || fi.Mode()&os.ModeSymlink == 0 {
		t.Errorf("Expected %s to be a symlink to shared/sessions", p1Sessions)
	}

	// Create a dummy session in Deepak's sessions
	sessionFile := filepath.Join(p1Sessions, "session-123.json")
	if err := os.WriteFile(sessionFile, []byte(`{"session":"deepak-123"}`), 0600); err != nil {
		t.Fatalf("Failed to write session file: %v", err)
	}

	// Launch Rohit with shared context
	_, err = launcher.Launch(p2, []string{"--continue"}, true)
	if err != nil {
		t.Fatalf("Launch p2 failed: %v", err)
	}

	// Verify Rohit sees Deepak's session!
	p2SessionFile := filepath.Join(p2.ConfigDir, "sessions", "session-123.json")
	data, err := os.ReadFile(p2SessionFile)
	if err != nil {
		t.Fatalf("Rohit cannot see Deepak's session: %v", err)
	}
	if string(data) != `{"session":"deepak-123"}` {
		t.Errorf("Session content mismatch: %s", string(data))
	}
}
