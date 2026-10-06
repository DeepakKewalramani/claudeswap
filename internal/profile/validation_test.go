package profile

import (
	"errors"
	"strings"
	"testing"
)

func TestGenerateID(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"Single word", "Deepak", "deepak"},
		{"Two words", "Rohit Soni", "rohit-soni"},
		{"Special characters", "Hero & Work!", "hero-work"},
		{"Multiple spaces", "Deepak    Work", "deepak-work"},
		{"Hyphenated name", "Personal-Claude", "personal-claude"},
		{"Underscores", "hero_work_2", "hero-work-2"},
		{"Empty fallback", "   ", "profile"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GenerateID(tt.input)
			if got != tt.expected {
				t.Errorf("GenerateID(%q) = %q; want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestValidateID(t *testing.T) {
	validIDs := []string{
		"deepak",
		"rohit-soni",
		"personal",
		"hero-work",
		"dev-1",
	}

	for _, id := range validIDs {
		if err := ValidateID(id); err != nil {
			t.Errorf("ValidateID(%q) unexpectedly failed: %v", id, err)
		}
	}

	invalidIDs := []struct {
		id       string
		expected error
	}{
		{"", errors.New("empty")},
		{"../../evil", ErrPathTraversal},
		{"../something", ErrPathTraversal},
		{"deepak/sub", ErrPathTraversal},
		{"deepak\\sub", ErrPathTraversal},
		{"bin", ErrReservedProfileID},
		{"config", ErrReservedProfileID},
		{"logs", ErrReservedProfileID},
		{"backups", ErrReservedProfileID},
		{"profiles", ErrReservedProfileID},
		{"default", ErrReservedProfileID},
		{"Deepak", ErrInvalidProfileID}, // must be lowercase
		{"deepak*", ErrInvalidProfileID},
		{"-deepak", ErrInvalidProfileID},
		{"deepak-", ErrInvalidProfileID},
		{"deepak--work", ErrInvalidProfileID},
	}

	for _, tt := range invalidIDs {
		err := ValidateID(tt.id)
		if err == nil {
			t.Errorf("ValidateID(%q) should have failed but passed", tt.id)
		}
	}
}

func TestValidateName(t *testing.T) {
	if err := ValidateName("Deepak"); err != nil {
		t.Errorf("ValidateName('Deepak') failed: %v", err)
	}

	if err := ValidateName(""); err == nil {
		t.Errorf("ValidateName('') should have failed")
	}

	tooLong := strings.Repeat("a", 65)
	if err := ValidateName(tooLong); err == nil {
		t.Errorf("ValidateName(65 chars) should have failed")
	}

	withControlChar := "Deepak\x00Work"
	if err := ValidateName(withControlChar); err == nil {
		t.Errorf("ValidateName with control char should have failed")
	}
}

func TestValidateShortcut(t *testing.T) {
	valid := []string{"claudeswap-deepak", "deepak", "claude_rohit", "hero123"}
	for _, sc := range valid {
		if err := ValidateShortcut(sc); err != nil {
			t.Errorf("ValidateShortcut(%q) unexpectedly failed: %v", sc, err)
		}
	}

	invalid := []string{"", "claudeswap deepak", "claude/rohit", "../sc", "sc;rm -rf", "sc&test"}
	for _, sc := range invalid {
		if err := ValidateShortcut(sc); err == nil {
			t.Errorf("ValidateShortcut(%q) should have failed but passed", sc)
		}
	}
}
