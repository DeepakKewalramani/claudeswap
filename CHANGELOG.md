# Changelog

All notable changes to this project will be documented in this file.
The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

---

## [1.0.0] - 2026-10-06

### Initial Release of ClaudeSwap
- **Core CLI & Profile Management:**
  - `claudeswap add`, `list`, `status`, `launch`, `remove`, `rename`, `shortcut`, `set-default`
  - Direct profile shortcuts (e.g. `claudeswap-deepak`, `claudeswap-rohit`) with full argument forwarding
  - Process isolation using `CLAUDE_CONFIG_DIR`
  - Safe status detection without credential inspection
- **Shared Memory & Account Swapping:**
  - `claudeswap context [on|off|status]` for unifying chat history across accounts
  - `claudeswap switch <profile>` (and `claudeswap handoff`) for zero-friction account rotation when API rate limits are reached
- **Chat Import & Export:**
  - `claudeswap chat export` to portable `.tar.gz` archives or readable `.md`
  - `claudeswap chat import` with path-traversal security verification
  - `claudeswap chat list` session inspector
- **Interactive TUI:**
  - Modern Bubble Tea + Lip Gloss dashboard with keyboard navigation
- **Self-Healing Storage:**
  - Atomic JSON persistence and automatic backup recovery
- **Diagnostics:**
  - `claudeswap doctor` system diagnostics and automated shell PATH integration
- **Cross-Platform:**
  - Native builds for macOS (`darwin/arm64`, `darwin/amd64`), Linux (`linux/arm64`, `linux/amd64`), and Windows (`windows/arm64`, `windows/amd64`)
