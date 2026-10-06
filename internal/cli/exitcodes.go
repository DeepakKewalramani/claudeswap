package cli

// Exit codes conforming to the ClaudeSwap specification:
// 0 = success
// 1 = general error
// 2 = invalid usage
// 3 = profile not found
// 4 = Claude Code not found
// 5 = configuration error
// 6 = shortcut error
// 7 = authentication required
const (
	ExitSuccess         = 0
	ExitGeneralError    = 1
	ExitInvalidUsage    = 2
	ExitProfileNotFound = 3
	ExitClaudeNotFound  = 4
	ExitConfigError     = 5
	ExitShortcutError   = 6
	ExitAuthRequired    = 7
)
