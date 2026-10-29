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
	"sync"
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

	// 3. ALWAYS check default ~/.claude/projects and ~/.claudehub / ~/.claudeswap shared projects in production
	userHome, _ := os.UserHomeDir()
	isProd := userHome != "" && (m.paths == nil || filepath.Dir(m.paths.HomeDir()) == userHome)
	if isProd {
		addDir(filepath.Join(userHome, ".claude", "projects"))
		addDir(filepath.Join(userHome, ".claudehub", "shared", "projects"))
		addDir(filepath.Join(userHome, ".claudeswap", "shared", "projects"))
	}

	return dirs
}

// ScanProjects scans the filesystem and groups all Claude Code sessions folder-wise.
func (m *Manager) ScanProjects() ([]ProjectGroup, error) {
	searchDirs := m.CollectProjectsDirs()
	groupsMap := make(map[string]*ProjectGroup)
	seenSessions := make(map[string]bool)

	userHome, _ := os.UserHomeDir()
	isProd := userHome != "" && (m.paths == nil || filepath.Dir(m.paths.HomeDir()) == userHome)
	liveClaudeSessions := loadLiveClaudeSessions()

	var repoCache sync.Map

	// Ensure active working directory is always registered in groupsMap in production
	if isProd {
		if cwd, err := os.Getwd(); err == nil && cwd != "" {
			name := resolveGitRepoNameWithCache(cwd, &repoCache)
			if name == "" {
				name = filepath.Base(cwd)
			}
			groupsMap[cwd] = &ProjectGroup{
				Name: name,
				CWD:  cwd,
			}
		}
	}

	type projectTarget struct {
		base      string
		entryName string
	}
	var targets []projectTarget
	for _, base := range searchDirs {
		entries, err := os.ReadDir(base)
		if err != nil {
			continue
		}

		for _, entry := range entries {
			if entry.IsDir() {
				targets = append(targets, projectTarget{base: base, entryName: entry.Name()})
			}
		}
	}

	var mu sync.Mutex
	var wg sync.WaitGroup

	for _, target := range targets {
		wg.Add(1)
		go func(t projectTarget) {
			defer wg.Done()

			projectDir := filepath.Join(t.base, t.entryName)
			fileEntries, err := os.ReadDir(projectDir)
			if err != nil {
				return
			}

			type sessionFileCandidate struct {
				sessionID string
				filePath  string
				fi        os.FileInfo
			}

			var candidates []sessionFileCandidate
			for _, fe := range fileEntries {
				if fe.IsDir() || !strings.HasSuffix(fe.Name(), ".jsonl") {
					continue
				}

				sessionID := strings.TrimSuffix(fe.Name(), ".jsonl")
				mu.Lock()
				alreadySeen := seenSessions[sessionID]
				if !alreadySeen {
					seenSessions[sessionID] = true
				}
				mu.Unlock()
				if alreadySeen {
					continue
				}

				fi, err := fe.Info()
				if err != nil {
					continue
				}

				candidates = append(candidates, sessionFileCandidate{
					sessionID: sessionID,
					filePath:  filepath.Join(projectDir, fe.Name()),
					fi:        fi,
				})
			}

			if len(candidates) == 0 {
				return
			}

			// Sort newest to oldest so we only deep-parse sessions relevant to current view
			sort.Slice(candidates, func(i, j int) bool {
				return candidates[i].fi.ModTime().After(candidates[j].fi.ModTime())
			})

			defaultProjectName := decodeProjectName(t.entryName)
			defaultCWD := ""

			localSessions := make([]Session, 0, len(candidates))
			for idx, cand := range candidates {
				var sess Session
				if idx < 20 {
					// Deep parse top 20 most recent sessions (covers full screen + tabs)
					sess = parseSessionHeader(cand.sessionID, cand.filePath, cand.fi, liveClaudeSessions, &repoCache)
					if defaultCWD == "" && sess.ProjectCWD != "" {
						defaultCWD = sess.ProjectCWD
						if repo := resolveGitRepoNameWithCache(defaultCWD, &repoCache); repo != "" {
							defaultProjectName = repo
						}
					}
				} else {
					// Instant lightweight parse for older sessions (zero disk I/O for jsonl)
					sess = parseSessionLight(cand.sessionID, cand.filePath, cand.fi, liveClaudeSessions)
					if defaultCWD != "" {
						sess.ProjectCWD = defaultCWD
						sess.ProjectName = defaultProjectName
					}
				}
				localSessions = append(localSessions, sess)
			}

			mu.Lock()
			defer mu.Unlock()
			for _, sess := range localSessions {
				groupKey := sess.ProjectCWD
				if groupKey == "" {
					groupKey = defaultCWD
				}
				if groupKey == "" {
					groupKey = sess.ProjectName
				}
				if groupKey == "" {
					groupKey = t.entryName
				}

				group, ok := groupsMap[groupKey]
				if !ok {
					groupName := sess.ProjectName
					if groupName == "" && sess.ProjectCWD != "" {
						groupName = resolveGitRepoNameWithCache(sess.ProjectCWD, &repoCache)
					}
					if groupName == "" {
						groupName = defaultProjectName
					}
					group = &ProjectGroup{
						Name: groupName,
						CWD:  sess.ProjectCWD,
					}
					groupsMap[groupKey] = group
				} else if group.Name == "" || group.Name == filepath.Base(group.CWD) {
					if repo := resolveGitRepoNameWithCache(group.CWD, &repoCache); repo != "" {
						group.Name = repo
					}
				}

				group.Sessions = append(group.Sessions, sess)
			}
		}(target)
	}

	wg.Wait()

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

// LiveSessionInfo captures active daemon and desktop state for a Claude Code session.
type LiveSessionInfo struct {
	PID        int    `json:"pid"`
	SessionID  string `json:"sessionId"`
	CWD        string `json:"cwd"`
	Name       string `json:"name"`
	Status     string `json:"status"`
	WaitingFor string `json:"waitingFor"`
	Entrypoint string `json:"entrypoint"`
}

func loadLiveClaudeSessions() map[string]LiveSessionInfo {
	liveMap := make(map[string]LiveSessionInfo)
	home, err := os.UserHomeDir()
	if err != nil {
		return liveMap
	}
	sessionsDir := filepath.Join(home, ".claude", "sessions")
	entries, err := os.ReadDir(sessionsDir)
	if err != nil {
		return liveMap
	}

	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		filePath := filepath.Join(sessionsDir, e.Name())
		data, err := os.ReadFile(filePath)
		if err != nil {
			continue
		}
		var info LiveSessionInfo
		if err := json.Unmarshal(data, &info); err == nil && info.SessionID != "" {
			liveMap[info.SessionID] = info
		}
	}
	return liveMap
}

func readCustomTitle(filePath, sessionID string) string {
	dir := filepath.Dir(filePath)
	ctPath := filepath.Join(dir, sessionID, "custom-title.json")
	if data, err := os.ReadFile(ctPath); err == nil {
		var ct struct {
			CustomTitle string `json:"customTitle"`
		}
		if err := json.Unmarshal(data, &ct); err == nil && strings.TrimSpace(ct.CustomTitle) != "" {
			return strings.TrimSpace(ct.CustomTitle)
		}
	}
	return ""
}

func resolveGitRepoName(dir string) string {
	if dir == "" {
		return ""
	}
	gitConfigFile := filepath.Join(dir, ".git", "config")
	data, err := os.ReadFile(gitConfigFile)
	if err != nil {
		gitDotFile := filepath.Join(dir, ".git")
		if content, readErr := os.ReadFile(gitDotFile); readErr == nil {
			line := strings.TrimSpace(string(content))
			if strings.HasPrefix(line, "gitdir:") {
				gitDir := strings.TrimSpace(strings.TrimPrefix(line, "gitdir:"))
				if !filepath.IsAbs(gitDir) {
					gitDir = filepath.Join(dir, gitDir)
				}
				data, _ = os.ReadFile(filepath.Join(gitDir, "config"))
			}
		}
	}

	if len(data) > 0 {
		lines := strings.Split(string(data), "\n")
		inOrigin := false
		for _, l := range lines {
			trimmed := strings.TrimSpace(l)
			if strings.HasPrefix(trimmed, "[remote ") {
				if strings.Contains(trimmed, "\"origin\"") || strings.Contains(trimmed, "\"tequity\"") {
					inOrigin = true
				} else {
					inOrigin = false
				}
				continue
			}
			if inOrigin && strings.HasPrefix(trimmed, "url =") {
				urlVal := strings.TrimSpace(strings.TrimPrefix(trimmed, "url ="))
				urlVal = strings.TrimSuffix(urlVal, ".git")
				parts := strings.Split(urlVal, "/")
				if len(parts) > 0 {
					repoName := parts[len(parts)-1]
					if repoName != "" {
						return repoName
					}
				}
			}
		}
	}

	// Also check ~/Library/Application Support/Claude/git-worktrees.json on macOS
	home, _ := os.UserHomeDir()
	if home != "" {
		wtPath := filepath.Join(home, "Library", "Application Support", "Claude", "git-worktrees.json")
		if wtData, err := os.ReadFile(wtPath); err == nil {
			var wt struct {
				OriginUrls map[string]struct {
					URL string `json:"url"`
				} `json:"originUrls"`
			}
			if err := json.Unmarshal(wtData, &wt); err == nil {
				key := strings.ToLower(dir)
				if entry, ok := wt.OriginUrls[key]; ok && entry.URL != "" {
					u := strings.TrimSuffix(entry.URL, ".git")
					parts := strings.Split(u, "/")
					if len(parts) > 0 && parts[len(parts)-1] != "" {
						return parts[len(parts)-1]
					}
				}
			}
		}
	}

	return filepath.Base(dir)
}

func resolveGitRepoNameWithCache(dir string, cache *sync.Map) string {
	if dir == "" {
		return ""
	}
	if cache != nil {
		if val, ok := cache.Load(dir); ok {
			return val.(string)
		}
	}
	val := resolveGitRepoName(dir)
	if cache != nil {
		cache.Store(dir, val)
	}
	return val
}

func parseSessionLight(sessionID, filePath string, fi os.FileInfo, liveClaudeSessions map[string]LiveSessionInfo) Session {
	sess := Session{
		ID:        sessionID,
		FilePath:  filePath,
		UpdatedAt: fi.ModTime(),
		SizeBytes: fi.Size(),
		Title:     "Session " + truncateID(sessionID),
	}

	if live, ok := liveClaudeSessions[sessionID]; ok {
		sess.IsLive = true
		sess.LiveStatus = live.Status
		sess.WaitingFor = live.WaitingFor
		sess.Entrypoint = live.Entrypoint
		sess.PID = live.PID
		if live.Name != "" {
			sess.Title = cleanTitle(live.Name)
		}
	}

	return sess
}

func parseSessionHeader(sessionID, filePath string, fi os.FileInfo, liveClaudeSessions map[string]LiveSessionInfo, repoCache *sync.Map) Session {
	sess := Session{
		ID:        sessionID,
		FilePath:  filePath,
		UpdatedAt: fi.ModTime(),
		SizeBytes: fi.Size(),
		Title:     "Session " + truncateID(sessionID),
	}

	// 1. Direct custom-title.json from Claude Code
	if ct := readCustomTitle(filePath, sessionID); ct != "" {
		sess.Title = cleanTitle(ct)
	}

	// 2. Live Claude Desktop / Daemon status
	if live, ok := liveClaudeSessions[sessionID]; ok {
		sess.IsLive = true
		sess.LiveStatus = live.Status
		sess.WaitingFor = live.WaitingFor
		sess.Entrypoint = live.Entrypoint
		sess.PID = live.PID
		if (sess.Title == "" || sess.Title == "Session "+truncateID(sessionID)) && live.Name != "" {
			sess.Title = cleanTitle(live.Name)
		}
	}

	f, err := os.Open(filePath)
	if err != nil {
		return sess
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	buf := make([]byte, 64*1024)
	scanner.Buffer(buf, 1024*1024)

	lineCount := 0
	seenSources := make(map[string]bool)
	seenFiles := make(map[string]bool)

	// Scanning up to 40 lines is plenty to get cwd, gitBranch, user prompt, and model!
	for scanner.Scan() && lineCount < 40 {
		lineCount++
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		var raw map[string]any
		if err := json.Unmarshal(line, &raw); err != nil {
			continue
		}

		if cwd, ok := raw["cwd"].(string); ok && cwd != "" && sess.ProjectCWD == "" {
			sess.ProjectCWD = cwd
			repo := resolveGitRepoNameWithCache(cwd, repoCache)
			if repo != "" {
				sess.ProjectName = repo
			} else {
				sess.ProjectName = filepath.Base(cwd)
			}
		}
		if branch, ok := raw["gitBranch"].(string); ok && branch != "" && sess.GitBranch == "" {
			sess.GitBranch = branch
		}

		// Prioritize real customTitle, agentName, and aiTitle from Claude Code
		if ct, ok := raw["customTitle"].(string); ok && strings.TrimSpace(ct) != "" {
			sess.Title = cleanTitle(ct)
		} else if an, ok := raw["agentName"].(string); ok && strings.TrimSpace(an) != "" {
			sess.Title = cleanTitle(an)
		} else if at, ok := raw["aiTitle"].(string); ok && strings.TrimSpace(at) != "" && !strings.HasPrefix(sess.Title, "custom:") {
			sess.Title = cleanTitle(at)
		} else if slug, ok := raw["slug"].(string); ok && strings.TrimSpace(slug) != "" && sess.Title == "Session "+truncateID(sessionID) {
			sess.Title = cleanTitle(slug)
		} else if sess.Title == "Session "+truncateID(sessionID) {
			// Title extraction from first real human user prompt (skip tool_result)
			entryType, _ := raw["type"].(string)
			if entryType == "user" {
				isToolResult := false
				if msg, ok := raw["message"].(map[string]any); ok {
					if content, ok := msg["content"].([]any); ok {
						for _, item := range content {
							if itemMap, ok := item.(map[string]any); ok {
								if itemMap["type"] == "tool_result" {
									isToolResult = true
									break
								}
							}
						}
					}
				}
				if !isToolResult {
					if msg := raw["message"]; msg != nil {
						title := extractMessageText(msg)
						if title != "" {
							sess.Title = cleanTitle(title)
						}
					}
				}
			}
		}

		// Deep Tool & Telemetry extraction
		extractTelemetry(raw, &sess, seenSources, seenFiles)

		// Early exit if core metadata and title are found
		if sess.ProjectCWD != "" && sess.GitBranch != "" && sess.Title != "Session "+truncateID(sessionID) && lineCount >= 10 {
			break
		}
	}

	return sess
}

func extractTelemetry(raw map[string]any, sess *Session, seenSources map[string]bool, seenFiles map[string]bool) {
	entryType, _ := raw["type"].(string)
	if entryType == "user" || entryType == "assistant" {
		sess.TurnCount++
	}

	// Direct tool_use
	if entryType == "tool_use" {
		recordTool(raw, sess, seenSources, seenFiles)
		return
	}

	// Message content
	if msg, ok := raw["message"].(map[string]any); ok {
		if m, ok := msg["model"].(string); ok && m != "" && sess.Model == "" {
			sess.Model = m
		}
		if usage, ok := msg["usage"].(map[string]any); ok {
			var total int64
			if r, ok := usage["cache_read_input_tokens"].(float64); ok {
				total += int64(r)
			}
			if in, ok := usage["input_tokens"].(float64); ok {
				total += int64(in)
			}
			if out, ok := usage["output_tokens"].(float64); ok {
				total += int64(out)
			}
			if total > 0 {
				sess.TotalTokens = total
			}
		}

		if content, ok := msg["content"].([]any); ok {
			for _, item := range content {
				if itemMap, ok := item.(map[string]any); ok {
					itType, _ := itemMap["type"].(string)
					if itType == "thinking" {
						if th, ok := itemMap["thinking"].(string); ok {
							trimmed := strings.TrimSpace(th)
							if trimmed != "" {
								sess.Thinking = trimmed
							}
						}
					} else if itType == "tool_use" {
						recordTool(itemMap, sess, seenSources, seenFiles)
					}
				}
			}
		}
	}
}

func recordTool(toolMap map[string]any, sess *Session, seenSources map[string]bool, seenFiles map[string]bool) {
	name, _ := toolMap["name"].(string)
	input, _ := toolMap["input"].(map[string]any)
	if input == nil {
		return
	}

	var stepSummary string

	switch name {
	case "WebSearch":
		if q, ok := input["query"].(string); ok && q != "" {
			cleanQ := cleanTitle(q)
			sess.LastActivity = "WebSearch: " + cleanQ
			stepSummary = "🌐 Search: " + cleanQ
		}
	case "WebFetch":
		if u, ok := input["url"].(string); ok && u != "" {
			sess.LastActivity = "WebFetch: " + u
			stepSummary = "🔗 Fetch: " + u
			if !seenSources[u] && len(sess.WebSources) < 6 {
				seenSources[u] = true
				sess.WebSources = append(sess.WebSources, u)
			}
		}
	case "Bash":
		if cmd, ok := input["command"].(string); ok && cmd != "" {
			trimmed := strings.TrimSpace(cmd)
			if len(trimmed) > 40 {
				trimmed = trimmed[:37] + "..."
			}
			sess.LastActivity = "Bash: " + trimmed
			stepSummary = "💻 Bash: " + trimmed
		}
	case "Read":
		if path, ok := input["file_path"].(string); ok && path != "" {
			base := filepath.Base(path)
			sess.LastActivity = "Read: " + base
			stepSummary = "📄 Read: " + base
			if !seenFiles[base] && len(sess.FilesTouched) < 8 {
				seenFiles[base] = true
				sess.FilesTouched = append(sess.FilesTouched, "📄 "+base)
			}
		}
	case "Write", "Edit":
		if path, ok := input["file_path"].(string); ok && path != "" {
			base := filepath.Base(path)
			sess.LastActivity = name + ": " + base
			stepSummary = "✏️ Write: " + base
			if !seenFiles[base] && len(sess.FilesTouched) < 8 {
				seenFiles[base] = true
				sess.FilesTouched = append(sess.FilesTouched, "✏️ "+base)
			}
		}
	}

	if stepSummary != "" {
		sess.Timeline = append(sess.Timeline, stepSummary)
		if len(sess.Timeline) > 5 {
			sess.Timeline = sess.Timeline[len(sess.Timeline)-5:]
		}
	}
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
	if clean == "" {
		return "root"
	}

	userHome, _ := os.UserHomeDir()
	if userHome != "" {
		homePrefix := strings.TrimPrefix(strings.ReplaceAll(userHome, "/", "-"), "-")
		if strings.HasPrefix(clean, homePrefix) {
			rel := strings.TrimPrefix(clean, homePrefix)
			rel = strings.TrimPrefix(rel, "-")
			for _, parent := range []string{"Projects-", "Documents-Projects-", "Documents-", "Downloads-", "Desktop-"} {
				if strings.HasPrefix(rel, parent) {
					return strings.TrimPrefix(rel, parent)
				}
			}
			if rel != "" {
				return rel
			}
			return filepath.Base(userHome)
		}
	}

	if idx := strings.Index(clean, "Projects-"); idx != -1 {
		return clean[idx+len("Projects-"):]
	}

	if idx := strings.LastIndex(clean, "--"); idx != -1 {
		return clean[idx+2:]
	}

	return clean
}

func truncateID(id string) string {
	if len(id) > 8 {
		return id[:8]
	}
	return id
}
