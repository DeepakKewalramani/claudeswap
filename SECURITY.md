# Security Policy

## Security Model & Boundary Guarantee

ClaudeSwap is built with a **strict security-first boundary**:

> **Core Guarantee:**
> ClaudeSwap does not implement, store, copy, or manage Claude authentication secrets, passwords, OTPs, access tokens, or refresh tokens. Authentication is strictly delegated to Claude Code.

ClaudeSwap operates exclusively as a **configuration orchestrator**:
1. It resolves an isolated profile directory (`~/.claudeswap/profiles/<id>/`).
2. It sets the official `CLAUDE_CONFIG_DIR` environment variable.
3. It starts Claude Code via direct `os/exec` (avoiding shell command string construction).
4. Claude Code performs native authentication and stores secrets directly in its own isolated state.

ClaudeSwap metadata files (`config.json`, `profiles.json`) only store display names, IDs, and directory paths. They never store secret keys or tokens.

---

## Reporting a Vulnerability

If you discover a potential security vulnerability or boundary breach in ClaudeSwap, please report it responsibly by contacting the maintainers or opening a private security advisory on GitHub.

Please do not disclose security vulnerabilities publicly until they have been addressed.
