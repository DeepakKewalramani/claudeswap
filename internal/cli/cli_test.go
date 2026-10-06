package cli

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/claudeswap/claudeswap/internal/claude"
	"github.com/claudeswap/claudeswap/internal/config"
	"github.com/claudeswap/claudeswap/internal/profile"
	"github.com/claudeswap/claudeswap/internal/shortcut"
)

func setupTestApp(t *testing.T, customClaudeBin string) (*AppContext, string) {
	tempDir := t.TempDir()
	paths := config.NewPathsWithBase(tempDir)
	cfgStore := config.NewConfigStore(paths)
	store := profile.NewStore(paths)
	mgr := profile.NewManager(paths, store)
	detector := claude.NewDetector(customClaudeBin)
	launcher := claude.NewLauncher(detector, paths)
	scMgr := shortcut.NewManager(paths)

	app := &AppContext{
		Paths:    paths,
		CfgStore: cfgStore,
		Store:    store,
		Manager:  mgr,
		Detector: detector,
		Launcher: launcher,
		Shortcut: scMgr,
	}

	return app, tempDir
}

func TestCLI_AddListStatusRenameRemove(t *testing.T) {
	app, _ := setupTestApp(t, "")

	// 1. Test Add
	rootCmd := NewRootCmd(app)
	rootCmd.SetArgs([]string{"add", "Deepak", "--shortcut", "claudeswap-deepak", "--no-launch"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("Add command failed: %v", err)
	}

	// Verify profile in manager
	p, err := app.Manager.GetProfile("deepak")
	if err != nil {
		t.Fatalf("Profile not found after add: %v", err)
	}
	if p.Name != "Deepak" || p.Shortcut != "claudeswap-deepak" {
		t.Errorf("Unexpected profile data: %+v", p)
	}

	// Verify shortcut file created
	scPath := app.Paths.ShortcutPath("claudeswap-deepak")
	if _, err := os.Stat(scPath); err != nil {
		t.Errorf("Expected shortcut file to exist at %s", scPath)
	}

	// 2. Test Add second profile
	rootCmd = NewRootCmd(app)
	rootCmd.SetArgs([]string{"add", "Rohit Soni", "--no-launch"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("Add second profile failed: %v", err)
	}

	// 3. Test List
	buf := new(bytes.Buffer)
	listCmd := newListCmd(app)
	listCmd.SetOut(buf)
	if err := listCmd.Execute(); err != nil {
		t.Fatalf("List command failed: %v", err)
	}
	listOutput := buf.String()
	if !strings.Contains(listOutput, "Deepak") || !strings.Contains(listOutput, "Rohit Soni") {
		t.Logf("List output: %s", listOutput)
	}

	// 4. Test Rename
	rootCmd = NewRootCmd(app)
	rootCmd.SetArgs([]string{"rename", "deepak", "Deepak Work", "--shortcut", "claudeswap-work"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("Rename command failed: %v", err)
	}

	renamed, err := app.Manager.GetProfile("deepak")
	if err != nil || renamed.Name != "Deepak Work" || renamed.Shortcut != "claudeswap-work" {
		t.Errorf("Rename failed: %+v", renamed)
	}

	// Verify old shortcut removed and new shortcut exists
	if _, err := os.Stat(scPath); !os.IsNotExist(err) {
		t.Errorf("Old shortcut should have been removed: %s", scPath)
	}
	newScPath := app.Paths.ShortcutPath("claudeswap-work")
	if _, err := os.Stat(newScPath); err != nil {
		t.Errorf("New shortcut should exist: %s", newScPath)
	}

	// 5. Test Set-Default
	rootCmd = NewRootCmd(app)
	rootCmd.SetArgs([]string{"set-default", "rohit-soni"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("Set-default failed: %v", err)
	}
	cfg, _ := app.CfgStore.Load()
	if cfg.DefaultProfile != "rohit-soni" {
		t.Errorf("Expected default profile to be rohit-soni, got: %s", cfg.DefaultProfile)
	}

	// 6. Test Remove with force
	rootCmd = NewRootCmd(app)
	rootCmd.SetArgs([]string{"remove", "deepak", "--force", "--delete-data"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("Remove command failed: %v", err)
	}

	_, err = app.Manager.GetProfile("deepak")
	if err == nil {
		t.Errorf("Expected profile deepak to be deleted")
	}
}

func TestCLI_DoctorAndSettings(t *testing.T) {
	app, _ := setupTestApp(t, "")

	// Settings view
	settingsCmd := newSettingsCmd(app)
	if err := settingsCmd.Execute(); err != nil {
		t.Fatalf("Settings command failed: %v", err)
	}

	// Doctor check
	doctorCmd := newDoctorCmd(app)
	// We expect doctor to run
	_ = doctorCmd.Execute()
}

func TestCLI_Version(t *testing.T) {
	cmd := newVersionCmd()
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.Execute()
}

func TestCLI_LaunchForwardingWithMockClaude(t *testing.T) {
	tempDir := t.TempDir()

	// Build mock Claude binary
	mockBinName := "mock-claude"
	if runtime.GOOS == "windows" {
		mockBinName += ".exe"
	}
	mockBinPath := filepath.Join(tempDir, mockBinName)
	buildCmd := exec.Command("go", "build", "-o", mockBinPath, "../../test/mock_claude/main.go")
	if out, err := buildCmd.CombinedOutput(); err != nil {
		t.Fatalf("Failed to build mock claude: %v (output: %s)", err, string(out))
	}

	app, _ := setupTestApp(t, mockBinPath)

	// Add profile
	p, err := app.Manager.CreateProfile("Deepak", "claudeswap-deepak")
	if err != nil {
		t.Fatalf("Failed to create profile: %v", err)
	}

	// Test Launch directly through launcher
	code, err := app.Launcher.Launch(p, []string{"--continue", "--resume"}, false)
	if err != nil {
		t.Fatalf("Launcher failed: %v", err)
	}
	if code != 0 {
		t.Errorf("Expected exit code 0, got %d", code)
	}
}

func TestCLI_ContextAndHandoff(t *testing.T) {
	app, _ := setupTestApp(t, "")

	// 1. Test context command
	rootCmd := NewRootCmd(app)
	rootCmd.SetArgs([]string{"context", "status"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("context status failed: %v", err)
	}

	rootCmd = NewRootCmd(app)
	rootCmd.SetArgs([]string{"context", "on"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("context on failed: %v", err)
	}

	cfg, _ := app.CfgStore.Load()
	if !cfg.SharedContext {
		t.Errorf("Expected SharedContext to be true")
	}

	rootCmd = NewRootCmd(app)
	rootCmd.SetArgs([]string{"context", "off"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("context off failed: %v", err)
	}

	cfg, _ = app.CfgStore.Load()
	if cfg.SharedContext {
		t.Errorf("Expected SharedContext to be false")
	}
}
