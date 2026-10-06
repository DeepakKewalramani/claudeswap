package profile

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"unicode"
)

var (
	// idRegex strictly permits lowercase letters, numbers, and single hyphens between characters.
	idRegex = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)

	// shortcutRegex permits letters, numbers, hyphens, and underscores.
	shortcutRegex = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)

	// reservedIDs are directory/file names reserved by ClaudeSwap or the OS.
	reservedIDs = map[string]bool{
		"bin":      true,
		"config":   true,
		"logs":     true,
		"backups":  true,
		"profiles": true,
		"default":  true,
		"null":     true,
		"con":      true,
		"prn":      true,
		"aux":      true,
	}
)

var (
	ErrEmptyProfileName    = errors.New("profile name cannot be empty")
	ErrProfileNameTooLong  = errors.New("profile name cannot exceed 64 characters")
	ErrInvalidProfileID    = errors.New("profile ID must contain only lowercase alphanumeric characters and hyphens")
	ErrReservedProfileID   = errors.New("profile ID is a reserved name")
	ErrPathTraversal       = errors.New("profile ID contains invalid or path traversal characters")
	ErrInvalidShortcutName = errors.New("shortcut name must contain only alphanumeric characters, hyphens, or underscores")
)

// GenerateID produces a sanitized, url/filesystem-safe ID from a profile display name.
// Examples:
//
//	"Deepak" -> "deepak"
//	"Rohit Soni" -> "rohit-soni"
//	"Hero Work!" -> "hero-work"
func GenerateID(name string) string {
	name = strings.TrimSpace(name)
	var sb strings.Builder

	lastWasHyphen := false
	for _, r := range name {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			sb.WriteRune(unicode.ToLower(r))
			lastWasHyphen = false
		} else if unicode.IsSpace(r) || r == '-' || r == '_' || r == '.' {
			if sb.Len() > 0 && !lastWasHyphen {
				sb.WriteRune('-')
				lastWasHyphen = true
			}
		}
	}

	result := strings.Trim(sb.String(), "-")
	if result == "" {
		return "profile"
	}
	return result
}

// ValidateName checks that a profile name is valid.
func ValidateName(name string) error {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return ErrEmptyProfileName
	}
	if len(trimmed) > 64 {
		return ErrProfileNameTooLong
	}

	// Reject control characters or newlines
	for _, r := range trimmed {
		if unicode.IsControl(r) {
			return errors.New("profile name cannot contain control characters")
		}
	}
	return nil
}

// ValidateID strictly validates a profile ID against path traversal, reserved words, and invalid characters.
func ValidateID(id string) error {
	id = strings.TrimSpace(id)
	if id == "" {
		return errors.New("profile ID cannot be empty")
	}

	// Explicit check for path traversal patterns
	if strings.Contains(id, "..") || strings.Contains(id, "/") || strings.Contains(id, "\\") {
		return fmt.Errorf("%w: %q", ErrPathTraversal, id)
	}

	if reservedIDs[strings.ToLower(id)] {
		return fmt.Errorf("%w: %q", ErrReservedProfileID, id)
	}

	if !idRegex.MatchString(id) {
		return fmt.Errorf("%w: %q", ErrInvalidProfileID, id)
	}

	return nil
}

// DefaultShortcut returns the standard shortcut command name for a given profile ID.
func DefaultShortcut(id string) string {
	return "claudeswap-" + id
}

// ValidateShortcut validates a shortcut name.
func ValidateShortcut(shortcut string) error {
	shortcut = strings.TrimSpace(shortcut)
	if shortcut == "" {
		return errors.New("shortcut name cannot be empty")
	}

	if strings.Contains(shortcut, "/") || strings.Contains(shortcut, "\\") || strings.Contains(shortcut, "..") {
		return fmt.Errorf("%w: %q", ErrPathTraversal, shortcut)
	}

	if !shortcutRegex.MatchString(shortcut) {
		return fmt.Errorf("%w: %q", ErrInvalidShortcutName, shortcut)
	}

	return nil
}
