package chat

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ExportSessionsToMarkdown converts all session .jsonl files in srcDir to readable .md files in destDir.
func ExportSessionsToMarkdown(srcDir, destDir string) (int, error) {
	sessions, err := FindSessions(srcDir)
	if err != nil {
		return 0, err
	}

	if err := os.MkdirAll(destDir, 0755); err != nil {
		return 0, err
	}

	converted := 0
	for _, sess := range sessions {
		outFile := filepath.Join(destDir, fmt.Sprintf("%s-%s.md", sess.Project, sess.ID))
		err := convertJSONLToMarkdown(sess.FilePath, outFile, sess)
		if err == nil {
			converted++
		}
	}

	return converted, nil
}

func convertJSONLToMarkdown(jsonlPath, mdPath string, meta SessionSummary) error {
	f, err := os.Open(jsonlPath)
	if err != nil {
		return err
	}
	defer f.Close()

	out, err := os.Create(mdPath)
	if err != nil {
		return err
	}
	defer out.Close()

	// Write Markdown Header
	fmt.Fprintf(out, "# Claude Code Session: %s\n\n", meta.ID)
	fmt.Fprintf(out, "- **Project:** %s\n", meta.Project)
	fmt.Fprintf(out, "- **Last Modified:** %s\n", meta.ModTime)
	fmt.Fprintf(out, "\n---\n\n")

	scanner := bufio.NewScanner(f)
	buf := make([]byte, 1024*1024) // 1MB buffer for long LLM outputs
	scanner.Buffer(buf, 10*1024*1024)

	msgIndex := 1
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		var raw map[string]interface{}
		if err := json.Unmarshal(line, &raw); err != nil {
			continue
		}

		// Try extracting role & content
		role := "Message"
		if r, ok := raw["role"].(string); ok {
			role = strings.Title(r)
		} else if t, ok := raw["type"].(string); ok {
			role = strings.Title(t)
		}

		content := extractContent(raw)
		if strings.TrimSpace(content) != "" {
			fmt.Fprintf(out, "### %d. %s\n\n", msgIndex, role)
			fmt.Fprintf(out, "%s\n\n---\n\n", strings.TrimSpace(content))
			msgIndex++
		}
	}

	return scanner.Err()
}

func extractContent(raw map[string]interface{}) string {
	// 1. Direct "content" string
	if c, ok := raw["content"].(string); ok {
		return c
	}

	// 2. Direct "text" string
	if t, ok := raw["text"].(string); ok {
		return t
	}

	// 3. Nested "message" object
	if msg, ok := raw["message"].(map[string]interface{}); ok {
		if c, ok := msg["content"].(string); ok {
			return c
		}
		if text, ok := msg["text"].(string); ok {
			return text
		}
	}

	// 4. Array of content blocks
	if arr, ok := raw["content"].([]interface{}); ok {
		var parts []string
		for _, item := range arr {
			if m, ok := item.(map[string]interface{}); ok {
				if text, ok := m["text"].(string); ok {
					parts = append(parts, text)
				}
			}
		}
		if len(parts) > 0 {
			return strings.Join(parts, "\n\n")
		}
	}

	return ""
}
