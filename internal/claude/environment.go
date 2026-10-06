package claude

import (
	"os"
	"strings"
)

// ConfigEnvVar is the primary environment variable recognized by Claude Code for isolated configuration.
// If Claude Code changes its mechanism in the future, only this integration package needs to be updated.
const ConfigEnvVar = "CLAUDE_CONFIG_DIR"

// BuildEnvironment takes the current process environment and overrides CLAUDE_CONFIG_DIR with the isolated profile directory.
func BuildEnvironment(configDir string) []string {
	var env []string
	prefix := ConfigEnvVar + "="

	for _, entry := range os.Environ() {
		// Filter out any inherited CLAUDE_CONFIG_DIR
		if strings.HasPrefix(entry, prefix) {
			continue
		}
		env = append(env, entry)
	}

	// Inject the isolated profile configuration directory
	env = append(env, prefix+configDir)
	return env
}
