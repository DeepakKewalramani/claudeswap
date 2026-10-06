package chat

import (
	"os"
	"path/filepath"
	"testing"
)

func TestChatExportAndImport(t *testing.T) {
	tempDir := t.TempDir()

	// 1. Create simulated project and sessions
	srcDir := filepath.Join(tempDir, "src")
	projDir := filepath.Join(srcDir, "projects", "my-repo")
	sessDir := filepath.Join(srcDir, "sessions")
	_ = os.MkdirAll(projDir, 0700)
	_ = os.MkdirAll(sessDir, 0700)

	// Create sample session JSONL
	jsonlContent := `{"role":"user","content":"Hello world"}
{"role":"assistant","content":"Hi there! How can I help you today?"}
`
	jsonlPath := filepath.Join(projDir, "session-abc.jsonl")
	if err := os.WriteFile(jsonlPath, []byte(jsonlContent), 0600); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	// 2. Test FindSessions
	sessions, err := FindSessions(srcDir)
	if err != nil {
		t.Fatalf("FindSessions failed: %v", err)
	}
	if len(sessions) != 1 {
		t.Fatalf("Expected 1 session, found %d", len(sessions))
	}
	if sessions[0].ID != "session-abc" || sessions[0].Project != "my-repo" {
		t.Errorf("Unexpected session metadata: %+v", sessions[0])
	}

	// 3. Test ExportArchive
	archivePath := filepath.Join(tempDir, "export.tar.gz")
	if err := ExportArchive(srcDir, archivePath); err != nil {
		t.Fatalf("ExportArchive failed: %v", err)
	}

	info, err := os.Stat(archivePath)
	if err != nil || info.Size() == 0 {
		t.Fatalf("Export archive is missing or empty")
	}

	// 4. Test ImportArchive
	destDir := filepath.Join(tempDir, "dest")
	count, err := ImportArchive(archivePath, destDir)
	if err != nil {
		t.Fatalf("ImportArchive failed: %v", err)
	}
	if count < 1 {
		t.Errorf("Expected at least 1 imported file, got %d", count)
	}

	// Verify imported file content matches
	importedJsonl := filepath.Join(destDir, "projects", "my-repo", "session-abc.jsonl")
	data, err := os.ReadFile(importedJsonl)
	if err != nil {
		t.Fatalf("Failed to read imported jsonl: %v", err)
	}
	if string(data) != jsonlContent {
		t.Errorf("Imported content mismatch: %q", string(data))
	}

	// 5. Test ExportSessionsToMarkdown
	mdOutDir := filepath.Join(tempDir, "markdown_export")
	mdCount, err := ExportSessionsToMarkdown(srcDir, mdOutDir)
	if err != nil {
		t.Fatalf("ExportSessionsToMarkdown failed: %v", err)
	}
	if mdCount != 1 {
		t.Errorf("Expected 1 markdown exported, got %d", mdCount)
	}

	mdFile := filepath.Join(mdOutDir, "my-repo-session-abc.md")
	mdData, err := os.ReadFile(mdFile)
	if err != nil {
		t.Fatalf("Failed to read markdown file: %v", err)
	}
	if len(mdData) == 0 {
		t.Errorf("Markdown file is empty")
	}
}
