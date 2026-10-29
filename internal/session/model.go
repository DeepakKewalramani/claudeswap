package session

import (
	"time"
)

// Session represents a Claude Code conversation session.
type Session struct {
	ID           string    `json:"id"`
	ProjectName  string    `json:"projectName"`
	ProjectCWD   string    `json:"projectCwd"`
	FilePath     string    `json:"filePath"`
	GitBranch    string    `json:"gitBranch"`
	Title        string    `json:"title"`
	UpdatedAt    time.Time `json:"updatedAt"`
	SizeBytes    int64     `json:"sizeBytes"`
	MessageCount int       `json:"messageCount"`
	LastActivity string    `json:"lastActivity"`
	WebSources   []string  `json:"webSources"`

	// Deep Live Intelligence & Telemetry
	Model        string   `json:"model"`
	Thinking     string   `json:"thinking"`
	FilesTouched []string `json:"filesTouched"`
	Timeline     []string `json:"timeline"`
	TotalTokens  int64    `json:"totalTokens"`
	TurnCount    int      `json:"turnCount"`

	// Live Claude Desktop & Background State
	IsLive     bool   `json:"isLive"`
	LiveStatus string `json:"liveStatus"` // "waiting", "idle", "active"
	WaitingFor string `json:"waitingFor"` // "permission prompt"
	Entrypoint string `json:"entrypoint"` // "claude-desktop", "cli"
	PID        int    `json:"pid"`
}

// ProjectGroup aggregates all Claude Code sessions for a specific project directory.
type ProjectGroup struct {
	Name     string    `json:"name"`
	CWD      string    `json:"cwd"`
	Sessions []Session `json:"sessions"`
}
