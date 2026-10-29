# ClaudeSwap Developer & Agent Guidelines

ClaudeSwap is a high-performance developer tool written in Go that enables developers to manage, isolate, and seamlessly switch between multiple independent Claude Code profiles with shared memory and unified context.

---

## 1. Core Architectural Invariants

### 1.1 Zero-Touch Credentials & Privacy Isolation
- **NEVER** read, inspect, parse, or log authentication tokens, session cookies, API keys, or credentials stored inside `.claude.json` or OS keychains.
- **NEVER** output, leak, or mention third-party project names, client names, external repository names, or private local directories in responses or documentation. Always use generic placeholders (e.g. `<project>`, `workspace-1`).
- Profile isolation is maintained strictly through the `CLAUDE_CONFIG_DIR` environment variable pointing to `~/.claudeswap/profiles/<profile-id>`.
- Any command that inspects profile status must do so via non-intrusive heuristics (e.g., checking for the presence of valid OAuth tokens without reading their values, checking file timestamps, or testing process exit codes).

### 1.2 Shared Context vs Isolation Invariant
- `~/.claudeswap/shared/` only stores transferable conversation artifacts:
  - `sessions/`: Transcripts and conversation state.
  - `projects/`: Folder-wise project memory and state mappings.
  - `file-history/`: File edit caches.
  - `plans/`: Active planning transcripts.
  - `history.jsonl`: Shell command execution history.
- When `sharedContext` is **ON**: These directories are symlinked from `~/.claudeswap/shared/` into each profile's config directory.
- When `sharedContext` is **OFF**: Symlinks are safely detached, creating independent local directories. The shared pool is preserved untouched.
- When toggling back **ON**: `mergeDirectories()` must merge any local sessions created while disconnected into the shared pool before recreating symlinks.

### 1.3 Live Session & Process Safety
- Active Claude Code sessions may hold process locks (`daemon.lock`).
- When launching, resuming, or attaching to a session that is already live or running in the background, prefer `--fork-session` to prevent lock contention and process aborts.
- Always forward terminal signals (`os.Interrupt`, `syscall.SIGTERM`, `syscall.SIGHUP`) to the child process so Claude Code handles interrupts cleanly.

### 1.4 Cross-Platform Compatibility
- Support macOS (`darwin`), Linux (`linux`), and Windows (`windows`) across `amd64` and `arm64`.
- Never hardcode `/` or `\` in filesystem paths; always use `filepath.Join()`, `filepath.Clean()`, and `filepath.ToSlash()`.
- On Windows, account for `.cmd` and `.exe` extensions when executing or locating binaries, and avoid POSIX-only file permission assumptions.

### 1.5 Dual-Interface Parity (TUI & CLI)
- Any feature exposed in the interactive Bubble Tea dashboard must have a corresponding non-interactive, scriptable CLI equivalent (e.g., `--no-launch`, `--json`, `--yes`).
- Interactive prompts must check `term.IsTerminal(int(os.Stdin.Fd()))` and fall back gracefully to standard buffered I/O when piped or run in CI/headless environments.

---

## 2. Strict Version Management & Release Discipline

Whenever features, improvements, or fixes are made:
1. Adhere strictly to **Semantic Versioning (MAJOR.MINOR.PATCH)**.
2. Synchronize all version declarations across the repository:
   - `internal/cli/version.go` (`const Version`)
   - `Makefile` (`VERSION =`)
   - `install.sh` (`VERSION =`)
   - `install.ps1` (`$Version =`)
   - `CHANGELOG.md` (`## [X.Y.Z] - YYYY-MM-DD`)
3. Never push or release without running `go test ./...` and `go vet ./...`.

