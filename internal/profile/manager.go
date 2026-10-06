package profile

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/claudeswap/claudeswap/internal/config"
)

var (
	ErrProfileNotFound      = errors.New("profile not found")
	ErrProfileAlreadyExists = errors.New("profile already exists")
	ErrShortcutAlreadyUsed  = errors.New("shortcut name already used by another profile")
)

// Manager provides business logic for managing ClaudeSwap profiles.
type Manager struct {
	paths *config.Paths
	store *Store
}

// NewManager creates a new Manager instance.
func NewManager(paths *config.Paths, store *Store) *Manager {
	return &Manager{
		paths: paths,
		store: store,
	}
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

// GetStatus checks profile status in a strictly safe manner.
// It NEVER reads tokens, credentials, or secrets.
// It checks whether the isolated directory exists and contains data generated by Claude Code.
func (m *Manager) GetStatus(p *Profile) Status {
	if p.ConfigDir == "" {
		return StatusUnknown
	}

	info, err := os.Stat(p.ConfigDir)
	if err != nil {
		if os.IsNotExist(err) {
			return StatusLoginRequired
		}
		return StatusUnknown
	}

	if !info.IsDir() {
		return StatusUnknown
	}

	entries, err := os.ReadDir(p.ConfigDir)
	if err != nil {
		return StatusUnknown
	}

	// If empty, user hasn't launched or logged into Claude Code with this profile yet
	if len(entries) == 0 {
		return StatusLoginRequired
	}

	return StatusReady
}
