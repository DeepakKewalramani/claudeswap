package session

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/claudeswap/claudeswap/internal/config"
	"github.com/claudeswap/claudeswap/internal/profile"
)

var xmlTagRegex = regexp.MustCompile(`<[^>]+>`)

// Manager discovers and manages Claude Code session files across profiles and shared context.
type Manager struct {
	paths      *config.Paths
	profStore  *profile.Store
}

// NewManager creates a new session Manager.
func NewManager(paths *config.Paths, profStore *profile.Store) *Manager {
	return &Manager{
		paths:     paths,
		profStore: profStore,
	}
}

// CollectProjectsDirs resolves all directories where Claude Code stores project sessions.
func (m *Manager) CollectProjectsDirs() []string {
	seen := make(map[string]bool)
	var dirs []string

	addDir := func(d string) {
		clean := filepath.Clean(d)
		if clean == "" || seen[clean] {
			return
		}
		if info, err := os.Stat(clean); err == nil && info.IsDir() {
			seen[clean] = true
			dirs = append(dirs, clean)
		}
	}

	// 1. Check shared context projects dir
	if m.paths != nil {
		addDir(filepath.Join(m.paths.SharedDir(), "projects"))
		addDir(filepath.Join(m.paths.HomeDir(), "projects"))
	}

	// 2. Check each configured profile's projects directory
	if m.profStore != nil {
		storeData, err := m.profStore.Load()
		if err == nil && storeData != nil {
			for _, p := range storeData.Profiles {
				if p.ConfigDir != "" {
					addDir(filepath.Join(p.ConfigDir, "projects"))
				}
			}
		}
	}

	// 3. Fallback to ~/.claude/projects or ~/.claudehub/shared/projects only if no project dirs were found
	if len(dirs) == 0 {
		userHome, _ := os.UserHomeDir()
		if userHome != "" {
			addDir(filepath.Join(userHome, ".claude", "projects"))
			addDir(filepath.Join(userHome, ".claudehub", "shared", "projects"))
		}
	}

	return dirs
}

// ScanProjects scans the filesystem and groups all Claude Code sessions folder-wise.
func (m *Manager) ScanProjects() ([]ProjectGroup, error) {
	searchDirs := m.CollectProjectsDirs()
	groupsMap := make(map[string]*ProjectGroup)
	seenSessions := make(map[string]bool)

	for _, base := range searchDirs {
		entries, err := os.ReadDir(base)
		if err != nil {
			continue
		}

		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}

			projectDir := filepath.Join(base, entry.Name())
			fileEntries, err := os.ReadDir(projectDir)
			if err != nil {
				continue
			}

			for _, fe := range fileEntries {
				if fe.IsDir() || !strings.HasSuffix(fe.Name(), ".jsonl") {
					continue
				}

				sessionID := strings.TrimSuffix(fe.Name(), ".jsonl")
				filePath := filepath.Join(projectDir, fe.Name())

				// Avoid duplicates if folders are symlinked across profiles
				if seenSessions[sessionID] {
					continue
				}
				seenSessions[sessionID] = true

				fi, err := fe.Info()
				if err != nil {
					continue
				}

				sess := parseSessionHeader(sessionID, filePath, fi)

				groupKey := sess.ProjectCWD
				if groupKey == "" {
					groupKey = sess.ProjectName
				}
				if groupKey == "" {
					groupKey = entry.Name()
				}

				group, ok := groupsMap[groupKey]
				if !ok {
					group = &ProjectGroup{
						Name: sess.ProjectName,
						CWD:  sess.ProjectCWD,
					}
					if group.Name == "" {
						group.Name = decodeProjectName(entry.Name())
					}
					groupsMap[groupKey] = group
				}

				group.Sessions = append(group.Sessions, sess)
			}
		}
	}

	// Sort sessions in each group by UpdatedAt descending
	var groups []ProjectGroup
	for _, g := range groupsMap {
		sort.Slice(g.Sessions, func(i, j int) bool {
			return g.Sessions[i].UpdatedAt.After(g.Sessions[j].UpdatedAt)
		})
		groups = append(groups, *g)
	}

	// Sort project groups by most recently active session
	sort.Slice(groups, func(i, j int) bool {
		var timeI, timeJ time.Time
		if len(groups[i].Sessions) > 0 {
			timeI = groups[i].Sessions[0].UpdatedAt
		}
		if len(groups[j].Sessions) > 0 {
			timeJ = groups[j].Sessions[0].UpdatedAt
		}
		return timeI.After(timeJ)
	})

	return groups, nil
}

// DeleteSession permanently deletes a session jsonl file and its associated data directory.
func (m *Manager) DeleteSession(filePath string) error {
	if filePath == "" {
		return fmt.Errorf("empty session path")
	}

	if err := os.Remove(filePath); err != nil && !os.IsNotExist(err) {
		return err
	}

	// Also remove artifact folder if it exists (same name without .jsonl)
	dirPath := strings.TrimSuffix(filePath, ".jsonl")
	if info, err := os.Stat(dirPath); err == nil && info.IsDir() {
		_ = os.RemoveAll(dirPath)
	}

	return nil
}

// FindSessionByID searches across all groups for a session matching the UUID or prefix.
func (m *Manager) FindSessionByID(groups []ProjectGroup, target string) (*Session, *ProjectGroup, error) {
	trimmed := strings.TrimSpace(target)
	if trimmed == "" {
		return nil, nil, fmt.Errorf("session ID cannot be empty")
	}

	for _, g := range groups {
		for _, s := range g.Sessions {
			if strings.EqualFold(s.ID, trimmed) || strings.HasPrefix(strings.ToLower(s.ID), strings.ToLower(trimmed)) {
				return &s, &g, nil
			}
		}
	}

	return nil, nil, fmt.Errorf("session %q not found", target)
}

func parseSessionHeader(sessionID, filePath string, fi os.FileInfo) Session {
	sess := Session{
		ID:        sessionID,
		FilePath:  filePath,
		UpdatedAt: fi.ModTime(),
		SizeBytes: fi.Size(),
		Title:     "Session " + truncateID(sessionID),
	}

	f, err := os.Open(filePath)
	if err != nil {
		return sess
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	lineCount := 0

	for scanner.Scan() && lineCount < 40 {
		lineCount++
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		var entry struct {
			Type      string `json:"type"`
			CWD       string `json:"cwd"`
			GitBranch string `json:"gitBranch"`
			Timestamp string `json:"timestamp"`
			Message   any    `json:"message"`
		}

		if err := json.Unmarshal(line, &entry); err != nil {
			continue
		}

		if entry.CWD != "" && sess.ProjectCWD == "" {
			sess.ProjectCWD = entry.CWD
			sess.ProjectName = filepath.Base(entry.CWD)
		}
		if entry.GitBranch != "" && sess.GitBranch == "" {
			sess.GitBranch = entry.GitBranch
		}

		if entry.Type == "user" && entry.Message != nil {
			title := extractMessageText(entry.Message)
			if title != "" {
				sess.Title = cleanTitle(title)
				break
			}
		}
	}

	return sess
}

func extractMessageText(raw any) string {
	switch v := raw.(type) {
	case string:
		return v
	case map[string]any:
		if c, ok := v["content"].(string); ok {
			return c
		}
		if parts, ok := v["content"].([]any); ok {
			var sb strings.Builder
			for _, p := range parts {
				if m, ok := p.(map[string]any); ok {
					if t, ok := m["text"].(string); ok {
						sb.WriteString(t)
						sb.WriteString(" ")
					}
				}
			}
			return sb.String()
		}
	}
	return ""
}

func cleanTitle(s string) string {
	// Strip XML tags like <pasted_content> or <local-command-caveat>
	clean := xmlTagRegex.ReplaceAllString(s, " ")
	clean = strings.Join(strings.Fields(clean), " ")
	if len(clean) > 80 {
		return clean[:77] + "..."
	}
	return clean
}

func decodeProjectName(encoded string) string {
	clean := strings.TrimPrefix(encoded, "-")
	parts := strings.Split(clean, "-")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return encoded
}

func truncateID(id string) string {
	if len(id) > 8 {
		return id[:8]
	}
	return id
}
