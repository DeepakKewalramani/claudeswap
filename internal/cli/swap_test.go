package cli

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/claudeswap/claudeswap/internal/claude"
	"github.com/claudeswap/claudeswap/internal/profile"
	"github.com/claudeswap/claudeswap/internal/session"
)

func TestE2E_FullAccountSwapAndSessionMemoryLifecycle(t *testing.T) {
	tempDir := t.TempDir()

	// 1. Setup mock Claude binary
	mockBinPath := filepath.Join(tempDir, "mock-claude")
	buildCmd := exec.Command("go", "build", "-o", mockBinPath, "../../test/mock_claude/main.go")
	if out, err := buildCmd.CombinedOutput(); err != nil {
		t.Fatalf("Failed to build mock claude: %v (output: %s)", err, string(out))
	}

	app, _ := setupTestApp(t, mockBinPath)

	// 2. Create Profile 1 (Work) and Profile 2 (Personal)
	p1, err := app.Manager.CreateProfile("Work Account", "claudeswap-work")
	if err != nil {
		t.Fatalf("Failed to create p1: %v", err)
	}
	p2, err := app.Manager.CreateProfile("Personal Account", "claudeswap-personal")
	if err != nil {
		t.Fatalf("Failed to create p2: %v", err)
	}

	// Create separate simulated credentials in both profiles
	p1Creds := `{"oauth_token":"SECRET_WORK_TOKEN_111","account":"work@example.com"}`
	p2Creds := `{"oauth_token":"SECRET_PERSONAL_TOKEN_222","account":"personal@example.com"}`
	if err := os.WriteFile(filepath.Join(p1.ConfigDir, ".claude.json"), []byte(p1Creds), 0600); err != nil {
		t.Fatalf("Failed to write p1 creds: %v", err)
	}
	if err := os.WriteFile(filepath.Join(p2.ConfigDir, ".claude.json"), []byte(p2Creds), 0600); err != nil {
		t.Fatalf("Failed to write p2 creds: %v", err)
	}

	// 3. Enable Shared Context across profiles
	cfg, _ := app.CfgStore.Load()
	cfg.SharedContext = true
	if err := app.CfgStore.Save(cfg); err != nil {
		t.Fatalf("Failed to save config: %v", err)
	}
	if err := claude.SyncSharedContext(app.Paths, p1.ConfigDir, true); err != nil {
		t.Fatalf("Failed to sync shared context for p1: %v", err)
	}
	if err := claude.SyncSharedContext(app.Paths, p2.ConfigDir, true); err != nil {
		t.Fatalf("Failed to sync shared context for p2: %v", err)
	}

	// Verify both profiles symlink to the shared pool
	for _, p := range []*profile.Profile{p1, p2} {
		sessDir := filepath.Join(p.ConfigDir, "sessions")
		fi, err := os.Lstat(sessDir)
		if err != nil || fi.Mode()&os.ModeSymlink == 0 {
			t.Fatalf("Expected %s to be a symlink to shared pool", sessDir)
		}
	}

	// 4. Simulate Work profile generating a session
	sessID := "sess-payment-engine-9988"
	sessPayload := `{"sessionId":"` + sessID + `","title":"Refactor Payment Engine","cwd":"/projects/payment-service"}`
	sharedSessFile := filepath.Join(p1.ConfigDir, "sessions", sessID+".json")
	if err := os.WriteFile(sharedSessFile, []byte(sessPayload), 0600); err != nil {
		t.Fatalf("Failed to write session file in p1: %v", err)
	}

	// Also simulate project transcript
	projectDir := filepath.Join(p1.ConfigDir, "projects", "payment-service")
	_ = os.MkdirAll(projectDir, 0700)
	transcriptFile := filepath.Join(projectDir, sessID+".jsonl")
	transcriptContent := `{"type":"user","message":"Refactor stripe auth"}
{"type":"assistant","message":"Auth refactored successfully"}`
	if err := os.WriteFile(transcriptFile, []byte(transcriptContent), 0600); err != nil {
		t.Fatalf("Failed to write transcript file: %v", err)
	}

	// 5. Test Swapping / Handoff to Personal Profile
	// Verify Personal profile can immediately read the exact same session file
	p2SessFile := filepath.Join(p2.ConfigDir, "sessions", sessID+".json")
	p2Data, err := os.ReadFile(p2SessFile)
	if err != nil {
		t.Fatalf("Personal profile failed to read session from shared memory: %v", err)
	}
	if !strings.Contains(string(p2Data), sessID) {
		t.Fatalf("Personal profile sees incorrect session data: %s", string(p2Data))
	}

	// Launch Personal profile with --resume
	code, err := app.Launcher.Launch(p2, []string{"--resume", sessID}, true)
	if err != nil || code != 0 {
		t.Fatalf("Launch p2 failed: code=%d err=%v", code, err)
	}

	// 6. Test Session Manager Discovery
	sessMgr := session.NewManager(app.Paths, app.Store)
	groups, err := sessMgr.ScanProjects()
	if err != nil {
		t.Fatalf("ScanProjects failed: %v", err)
	}
	if len(groups) == 0 {
		t.Fatalf("Expected to discover project groups, got 0")
	}

	// 7. Test Toggling Shared Context OFF (Strict Isolation)
	cfg.SharedContext = false
	_ = app.CfgStore.Save(cfg)
	_ = claude.SyncSharedContext(app.Paths, p1.ConfigDir, false)
	_ = claude.SyncSharedContext(app.Paths, p2.ConfigDir, false)

	// Verify symlinks are detached
	for _, p := range []*profile.Profile{p1, p2} {
		sessDir := filepath.Join(p.ConfigDir, "sessions")
		fi, err := os.Lstat(sessDir)
		if err != nil || fi.Mode()&os.ModeSymlink != 0 {
			t.Fatalf("Expected %s to NOT be a symlink after disabling context", sessDir)
		}
	}

	// Write private session in Work profile
	privateSess := "sess-secret-work-111"
	_ = os.WriteFile(filepath.Join(p1.ConfigDir, "sessions", privateSess+".json"), []byte("private"), 0600)

	// Verify Personal profile DOES NOT see the private session
	if _, err := os.Stat(filepath.Join(p2.ConfigDir, "sessions", privateSess+".json")); err == nil {
		t.Fatalf("Personal profile should NOT see private session when context is OFF")
	}

	// 8. Test Toggling Context Back ON (Smart Auto-Merge)
	cfg.SharedContext = true
	_ = app.CfgStore.Save(cfg)
	_ = claude.SyncSharedContext(app.Paths, p1.ConfigDir, true)
	_ = claude.SyncSharedContext(app.Paths, p2.ConfigDir, true)

	// Verify private session was safely merged into the shared pool
	sharedMerged := filepath.Join(app.Paths.SharedDir(), "sessions", privateSess+".json")
	if _, err := os.Stat(sharedMerged); err != nil {
		t.Fatalf("Expected private session to be merged into shared pool: %v", err)
	}

	// 9. SECURITY INVARIANT CHECK: Zero-Touch Credentials
	// Verify .claude.json was never symlinked, overwritten, or leaked
	p1CredsAfter, _ := os.ReadFile(filepath.Join(p1.ConfigDir, ".claude.json"))
	p2CredsAfter, _ := os.ReadFile(filepath.Join(p2.ConfigDir, ".claude.json"))

	if string(p1CredsAfter) != p1Creds {
		t.Fatalf("CRITICAL SECURITY FAILURE: Profile 1 credentials modified or corrupted!")
	}
	if string(p2CredsAfter) != p2Creds {
		t.Fatalf("CRITICAL SECURITY FAILURE: Profile 2 credentials modified or corrupted!")
	}
	if _, err := os.Stat(filepath.Join(app.Paths.SharedDir(), ".claude.json")); err == nil {
		t.Fatalf("CRITICAL SECURITY FAILURE: .claude.json found in shared directory!")
	}
}
