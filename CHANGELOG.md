# Changelog

All notable changes to this project will be documented in this file.
The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

---

## [1.1.0] - 2026-10-29

### Added
- **Multi-Session Supervisor Matrix:**
  - `claudeswap matrix`: Mission control dashboard to supervise 20+ concurrent Claude Code sessions.
  - Real-time status cards, activity gauges, active process detection, and instant session jump.
- **Folder-Wise Session Explorer & Deep Telemetry HUD:**
  - `claudeswap sessions [list|resume|delete]`: Full-screen browser with left folder navigation and interactive session cards.
  - Multi-session tab strip (1–9 keys) for instant tab switching.
  - Deep telemetry HUD displaying live thinking state, model specs, token micro-gauges, and files touched.
  - Auto-forking (`--fork-session`) support on running background instances to prevent process lock contention.
  - In-place profile cycling (`s` / `p`) and instant memory toggling (`m`).
- **Background Agents CLI:**
  - `claudeswap agents`: Direct CLI subcommand to attach, inspect, or stop background agents.
- **Enhanced Account Handoff & Profile Removal:**
  - Interactive Bubble Tea menu for `claudeswap handoff` and quick confirmation for profile removal.
- **Architecture Guidelines & Skill Toolkits:**
  - Project rules (`AGENTS.md`, `GEMINI.md`) and automation skills (`claudeswap-release`, `claudeswap-profile-sync`, `claudeswap-tui-dev`).

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
