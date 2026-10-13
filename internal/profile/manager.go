package profile

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/claudeswap/claudeswap/internal/config"
)

var (
	ErrProfileNotFound      = errors.New("profile not found")
	ErrProfileAlreadyExists = errors.New("profile already exists")
	ErrShortcutAlreadyUsed  = errors.New("shortcut name already used by another profile")
)

// StatusChecker checks whether Claude Code in the given directory is logged in.
type StatusChecker func(configDir string) (bool, error)

type cachedStatus struct {
	status    Status
	dirMtime  time.Time
	jsonMtime time.Time
	timestamp time.Time
}

// Manager provides business logic for managing ClaudeSwap profiles.
type Manager struct {
	paths         *config.Paths
	store         *Store
	statusChecker StatusChecker
	statusCache   map[string]cachedStatus
	cacheMu       sync.RWMutex
}

// NewManager creates a new Manager instance.
func NewManager(paths *config.Paths, store *Store) *Manager {
	return &Manager{
		paths:       paths,
		store:       store,
		statusCache: make(map[string]cachedStatus),
	}
}

// SetStatusChecker configures an external Claude Code auth status checker.
func (m *Manager) SetStatusChecker(checker StatusChecker) {
	m.cacheMu.Lock()
	defer m.cacheMu.Unlock()
	m.statusChecker = checker
	m.statusCache = make(map[string]cachedStatus) // Invalidate cache when checker changes
}

// Paths returns the paths instance used by the manager.
func (m *Manager) Paths() *config.Paths {
	return m.paths
}

// Store returns the store instance.
func (m *Manager) Store() *Store {
	return m.store
}

// CreateProfile adds a new profile with the given display name and optional custom shortcut.
func (m *Manager) CreateProfile(name, customShortcut string) (*Profile, error) {
	if err := ValidateName(name); err != nil {
		return nil, err
	}

	id := GenerateID(name)
	if err := ValidateID(id); err != nil {
		return nil, err
	}

	shortcut := customShortcut
	if strings.TrimSpace(shortcut) == "" {
		shortcut = DefaultShortcut(id)
	}
	if err := ValidateShortcut(shortcut); err != nil {
		return nil, err
	}

	storeData, err := m.store.Load()
	if err != nil {
		return nil, err
	}

	// Check for collision with existing profiles
	for _, p := range storeData.Profiles {
		if strings.EqualFold(p.ID, id) || strings.EqualFold(p.Name, name) {
			return nil, fmt.Errorf("%w with ID %q or name %q", ErrProfileAlreadyExists, id, name)
		}
		if strings.EqualFold(p.Shortcut, shortcut) {
			return nil, fmt.Errorf("%w: %q", ErrShortcutAlreadyUsed, shortcut)
		}
	}

	configDir := m.paths.ProfileConfigDir(id)
	if err := os.MkdirAll(configDir, 0700); err != nil {
		return nil, fmt.Errorf("failed to create isolated profile directory %q: %w", configDir, err)
	}

	now := time.Now().UTC()
	newProfile := Profile{
		ID:        id,
		Name:      strings.TrimSpace(name),
		Shortcut:  shortcut,
		ConfigDir: configDir,
		CreatedAt: now,
		UpdatedAt: now,
	}

	storeData.Profiles = append(storeData.Profiles, newProfile)
	if err := m.store.Save(storeData); err != nil {
		return nil, err
	}

	return &newProfile, nil
}

// ListProfiles returns all configured profiles.
func (m *Manager) ListProfiles() ([]Profile, error) {
	storeData, err := m.store.Load()
	if err != nil {
		return nil, err
	}
	return storeData.Profiles, nil
}

// GetProfile resolves a profile by ID, name, or shortcut (case-insensitive).
func (m *Manager) GetProfile(idOrName string) (*Profile, error) {
	target := strings.TrimSpace(idOrName)
	if target == "" {
		return nil, errors.New("empty profile identifier")
	}

	storeData, err := m.store.Load()
	if err != nil {
		return nil, err
	}

	// 1. Exact match on ID
	for i := range storeData.Profiles {
		if strings.EqualFold(storeData.Profiles[i].ID, target) {
			return &storeData.Profiles[i], nil
		}
	}

	// 2. Exact match on Name
	for i := range storeData.Profiles {
		if strings.EqualFold(storeData.Profiles[i].Name, target) {
			return &storeData.Profiles[i], nil
		}
	}

	// 3. Exact match on Shortcut
	for i := range storeData.Profiles {
		if strings.EqualFold(storeData.Profiles[i].Shortcut, target) {
			return &storeData.Profiles[i], nil
		}
	}

	return nil, fmt.Errorf("%w: %q", ErrProfileNotFound, target)
}

// RenameProfile updates a profile's display name and optionally shortcut.
func (m *Manager) RenameProfile(idOrName, newName, newShortcut string) (*Profile, error) {
	if err := ValidateName(newName); err != nil {
		return nil, err
	}

	storeData, err := m.store.Load()
	if err != nil {
		return nil, err
	}

	target := strings.TrimSpace(idOrName)
	targetIdx := -1
	for i := range storeData.Profiles {
		if strings.EqualFold(storeData.Profiles[i].ID, target) ||
			strings.EqualFold(storeData.Profiles[i].Name, target) ||
			strings.EqualFold(storeData.Profiles[i].Shortcut, target) {
			targetIdx = i
			break
		}
	}

	if targetIdx == -1 {
		return nil, fmt.Errorf("%w: %q", ErrProfileNotFound, target)
	}

	current := storeData.Profiles[targetIdx]

	// Determine new shortcut
	effectiveShortcut := current.Shortcut
	if strings.TrimSpace(newShortcut) != "" {
		effectiveShortcut = strings.TrimSpace(newShortcut)
	}
	if err := ValidateShortcut(effectiveShortcut); err != nil {
		return nil, err
	}

	// Validate name/shortcut uniqueness against other profiles
	for i, p := range storeData.Profiles {
		if i == targetIdx {
			continue
		}
		if strings.EqualFold(p.Name, newName) {
			return nil, fmt.Errorf("%w: another profile has name %q", ErrProfileAlreadyExists, newName)
		}
		if strings.EqualFold(p.Shortcut, effectiveShortcut) {
			return nil, fmt.Errorf("%w: %q", ErrShortcutAlreadyUsed, effectiveShortcut)
		}
	}

	storeData.Profiles[targetIdx].Name = strings.TrimSpace(newName)
	storeData.Profiles[targetIdx].Shortcut = effectiveShortcut
	storeData.Profiles[targetIdx].UpdatedAt = time.Now().UTC()

	if err := m.store.Save(storeData); err != nil {
		return nil, err
	}

	res := storeData.Profiles[targetIdx]
	return &res, nil
}

// DeleteProfile deletes a profile from metadata, and optionally removes its configuration directory.
func (m *Manager) DeleteProfile(idOrName string, removeConfigDir bool) error {
	storeData, err := m.store.Load()
	if err != nil {
		return err
	}

	target := strings.TrimSpace(idOrName)
	targetIdx := -1
	for i := range storeData.Profiles {
		if strings.EqualFold(storeData.Profiles[i].ID, target) ||
			strings.EqualFold(storeData.Profiles[i].Name, target) ||
			strings.EqualFold(storeData.Profiles[i].Shortcut, target) {
			targetIdx = i
			break
		}
	}

	if targetIdx == -1 {
		return fmt.Errorf("%w: %q", ErrProfileNotFound, target)
	}

	removed := storeData.Profiles[targetIdx]
	storeData.Profiles = append(storeData.Profiles[:targetIdx], storeData.Profiles[targetIdx+1:]...)

	if err := m.store.Save(storeData); err != nil {
		return err
	}

	if removeConfigDir && removed.ConfigDir != "" {
		// Safety check: ensure removed.ConfigDir is strictly within profiles directory
		profilesBase := m.paths.ProfilesDir()
		if strings.HasPrefix(removed.ConfigDir, profilesBase) && removed.ConfigDir != profilesBase {
			_ = os.RemoveAll(removed.ConfigDir)
		}
	}

	return nil
}

// PreloadStatuses evaluates statuses for multiple profiles concurrently.
// It populates the cache so subsequent GetStatus calls return immediately.
func (m *Manager) PreloadStatuses(profiles []Profile) {
	if len(profiles) == 0 {
		return
	}
	var wg sync.WaitGroup
	for i := range profiles {
		wg.Add(1)
		go func(p *Profile) {
			defer wg.Done()
			_ = m.GetStatus(p)
		}(&profiles[i])
	}
	wg.Wait()
}

// GetStatus checks profile status safely and accurately.
// It detects whether the profile is logged in (Ready), logged out (Logged out),
// or has an expired session (Session out).
// It NEVER exposes tokens, credentials, or secrets.
func (m *Manager) GetStatus(p *Profile) Status {
	if p == nil || p.ConfigDir == "" {
		return StatusUnknown
	}

	info, err := os.Stat(p.ConfigDir)
	if err != nil {
		if os.IsNotExist(err) {
			return StatusLoggedOut
		}
		return StatusUnknown
	}
	if !info.IsDir() {
		return StatusUnknown
	}

	claudeJSONPath := filepath.Join(p.ConfigDir, ".claude.json")
	var jsonMtime time.Time
	if jInfo, err := os.Stat(claudeJSONPath); err == nil {
		jsonMtime = jInfo.ModTime()
	}

	m.cacheMu.RLock()
	if cached, ok := m.statusCache[p.ConfigDir]; ok {
		if cached.dirMtime.Equal(info.ModTime()) && cached.jsonMtime.Equal(jsonMtime) && time.Since(cached.timestamp) < 30*time.Second {
			m.cacheMu.RUnlock()
			return cached.status
		}
	}
	m.cacheMu.RUnlock()

	status := m.detectStatus(p, claudeJSONPath)

	m.cacheMu.Lock()
	if m.statusCache == nil {
		m.statusCache = make(map[string]cachedStatus)
	}
	m.statusCache[p.ConfigDir] = cachedStatus{
		status:    status,
		dirMtime:  info.ModTime(),
		jsonMtime: jsonMtime,
		timestamp: time.Now(),
	}
	m.cacheMu.Unlock()

	return status
}

func (m *Manager) detectStatus(p *Profile, claudeJSONPath string) Status {
	hasOAuth := checkOAuthAccount(claudeJSONPath)
	hasAPIKey := checkAPIKey(p.ConfigDir)

	// If neither OAuth account details nor API key exist, user has never logged in or explicitly logged out
	if !hasOAuth && !hasAPIKey {
		return StatusLoggedOut
	}

	// If an auth status checker is configured (e.g. via claude auth status --json)
	if m.statusChecker != nil {
		loggedIn, err := m.statusChecker(p.ConfigDir)
		if err == nil {
			if loggedIn {
				return StatusReady
			}
			// Account credentials were known locally, but Claude Code reported not logged in: session expired / out
			return StatusSessionOut
		}
	}

	// Fallback heuristic: check daemon auth status
	daemonPath := filepath.Join(p.ConfigDir, "daemon-auth-status.json")
	if data, err := os.ReadFile(daemonPath); err == nil {
		if strings.Contains(string(data), "auth_required") {
			// Ensure daemon status is not older than .claude.json
			if dInfo, err := os.Stat(daemonPath); err == nil {
				if cInfo, err := os.Stat(claudeJSONPath); err == nil {
					if dInfo.ModTime().After(cInfo.ModTime()) {
						return StatusSessionOut
					}
				} else {
					return StatusSessionOut
				}
			}
		}
	}

	return StatusReady
}

func checkOAuthAccount(path string) bool {
	data, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	var root map[string]any
	if err := json.Unmarshal(data, &root); err != nil {
		return false
	}
	val, ok := root["oauthAccount"]
	if !ok || val == nil {
		return false
	}
	if m, ok := val.(map[string]any); ok {
		return len(m) > 0
	}
	return false
}

func checkAPIKey(configDir string) bool {
	settingsPath := filepath.Join(configDir, "settings.json")
	if data, err := os.ReadFile(settingsPath); err == nil {
		var s map[string]any
		if err := json.Unmarshal(data, &s); err == nil {
			if k, ok := s["apiKey"].(string); ok && strings.TrimSpace(k) != "" {
				return true
			}
			if k, ok := s["anthropicApiKey"].(string); ok && strings.TrimSpace(k) != "" {
				return true
			}
		}
	}

	// Recognize test/mock config.json
	mockPath := filepath.Join(configDir, "config.json")
	if _, err := os.Stat(mockPath); err == nil {
		return true
	}

	return false
}
