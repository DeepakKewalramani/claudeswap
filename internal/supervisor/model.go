package supervisor

import (
	"time"
)

// SessionState represents the operational status of a background Claude session.
type SessionState string

const (
	StateRunning      SessionState = "RUNNING"
	StateWaitingInput SessionState = "WAITING INPUT"
	StateCompleted    SessionState = "COMPLETED"
	StateFailed       SessionState = "FAILED"
	StateStopped      SessionState = "STOPPED"
)

// ManagedSession represents a tracked background Claude Code session.
type ManagedSession struct {
	ID           string       `json:"id"`
	TaskName     string       `json:"taskName"`
	ProjectName  string       `json:"projectName"`
	ProjectDir   string       `json:"projectDir"`
	GitBranch    string       `json:"gitBranch"`
	ProfileName  string       `json:"profileName"`
	PID          int          `json:"pid"`
	Status       SessionState `json:"status"`
	LastActivity string       `json:"lastActivity"`
	InitialQuery string       `json:"initialQuery"`
	LogPath      string       `json:"logPath"`
	StartedAt    time.Time    `json:"startedAt"`
	UpdatedAt    time.Time    `json:"updatedAt"`
	ExitCode     int          `json:"exitCode"`
}

// Registry stores all tracked supervisor sessions on disk.
type Registry struct {
	Sessions []ManagedSession `json:"sessions"`
}
