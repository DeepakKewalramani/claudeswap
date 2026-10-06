package profile

import (
	"time"
)

// CurrentProfileSchemaVersion defines the schema version for profiles.json.
const CurrentProfileSchemaVersion = 1

// Status represents the safe detected readiness of a profile.
type Status string

const (
	// StatusReady indicates the profile directory exists and has session/config state.
	StatusReady Status = "Ready"
	// StatusLoginRequired indicates the profile exists but no session or configuration has been initialized yet.
	StatusLoginRequired Status = "Login required"
	// StatusUnknown indicates status could not be determined without inspecting sensitive stores.
	StatusUnknown Status = "Unknown"
)

// Profile represents an isolated Claude Code configuration profile.
// CRITICAL SECURITY NOTE: Never include authentication secrets, tokens, OTPs, or passwords in this struct.
type Profile struct {
	ID            string    `json:"id"`
	Name          string    `json:"name"`
	Shortcut      string    `json:"shortcut"`
	ConfigDir     string    `json:"configDir"`
	SharedContext *bool     `json:"sharedContext,omitempty"`
	CreatedAt     time.Time `json:"createdAt"`
	UpdatedAt     time.Time `json:"updatedAt"`
}

// StoreData represents the root JSON schema for ~/.claudeswap/profiles.json.
type StoreData struct {
	Version  int       `json:"version"`
	Profiles []Profile `json:"profiles"`
}
