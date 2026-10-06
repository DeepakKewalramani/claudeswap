# ClaudeSwap

<p align="center">
  <strong>Instant Account Swapper & Memory Continuity for Claude Code</strong>
</p>

<p align="center">
  <a href="LICENSE"><img src="https://img.shields.io/badge/license-MIT-blue.svg" alt="License"></a>
  <a href="https://golang.org"><img src="https://img.shields.io/badge/go-1.22%2B-00ADD8.svg" alt="Go Version"></a>
  <a href="#"><img src="https://img.shields.io/badge/platform-macOS%20%7C%20Linux%20%7C%20Windows-lightgrey.svg" alt="Platforms"></a>
  <a href="#"><img src="https://img.shields.io/badge/security-zero--credential%20storage-brightgreen.svg" alt="Security"></a>
</p>

---

**ClaudeSwap** (`claudeswap`) is a fast, secure, cross-platform developer CLI that lets you manage and instantly swap between multiple Claude Code accounts with **unified context, shared session memory, and zero downtime**.

When your current Claude account hits rate limits or quota, ClaudeSwap seamlessly rotates your active conversation to your next account so you can continue coding without explaining context all over again.

> **Security Guarantee:**
> ClaudeSwap operates with a strict zero-credential boundary. It does not store, read, copy, or manage Claude passwords, OTPs, access tokens, refresh tokens, or other authentication secrets. Authentication remains entirely native to Claude Code.

---

## ✨ Features

- **⚡ Instant Account Swapping (`claudeswap switch`):** Swap active Claude accounts on the fly with an interactive prompt or direct argument.
- **🔄 Quota Limit Rotation (`claudeswap handoff`):** When one account hits token or rate limits, ClaudeSwap passes the conversation baton to your next account and resumes with Claude Code's `--resume` flag.
- **🧠 Continuous Memory & Shared Context:** Keep project sessions, prompt history, and execution context synced across all your profiles with `claudeswap context on`.
- **🛡️ Strict Security Boundary:** Zero credential storage or token handling. Claude Code authenticates natively and securely.
- **📦 Process Isolation:** Fully separated configuration directories (`CLAUDE_CONFIG_DIR`) ensure sessions, memory, and settings never collide.
- **👥 Concurrent Profiles:** Run multiple Claude Code instances under different accounts simultaneously without lock contention.
- **🎯 Direct Shortcuts:** Launch accounts directly with `claudeswap-<name>` (e.g. `claudeswap-deepak`, `claudeswap-rohit`).
- **💾 Chat Export & Import:** Export chats to portable `.tar.gz` archives or readable `.md` files; import sessions into any profile or machine.
- **🖥️ Interactive Terminal UI:** Built with Bubble Tea & Lip Gloss for a terminal UI with keyboard navigation and instant status indicators.
- **🩹 Self-Healing Storage:** Atomic writes with automatic backup creation and corruption recovery.
- **🩺 Built-in Diagnostics:** `claudeswap doctor` checks filesystem integrity, CLI availability, PATH configuration, and shell hooks.

---

## 🚀 Installation

### Option 1: One-Line Installer (macOS & Linux)

```bash
curl -fsSL https://raw.githubusercontent.com/claudeswap/claudeswap/main/install.sh | sh
```

### Option 2: Go Install (Any Platform)

```bash
go install github.com/claudeswap/claudeswap/cmd/claudeswap@latest
```

### Option 3: Windows PowerShell

```powershell
irm https://raw.githubusercontent.com/claudeswap/claudeswap/main/install.ps1 | iex
```

### Option 4: Build from Source

```bash
git clone https://github.com/claudeswap/claudeswap.git
cd claudeswap
make install
```

After installation, ensure your shell PATH includes ClaudeSwap:
```bash
claudeswap doctor --install-shell
source ~/.zshrc    # or ~/.bashrc
```

---

## 🕹️ Everyday Developer Experience

### 1. Launch a Profile Directly
```bash
# Launch Deepak's Claude Code profile
claudeswap-deepak

# Launch Rohit's Claude Code profile
claudeswap-rohit

# Launch with flags forwarded to Claude Code unchanged
claudeswap-deepak --continue
claudeswap-rohit --resume
```

### 2. Switch Accounts Instantly
```bash
# Interactive menu: pick an account and continue current conversation
claudeswap switch

# Or switch directly to a specific profile
claudeswap switch rohit
```

### 3. Open the Interactive Dashboard
```bash
claudeswap
```

```text
╭────────────────────────────────────────────────╮
│                 ClaudeSwap                     │
│      Claude Profile Manager & Swapper          │
├────────────────────────────────────────────────┤
│                                                │
│  Profiles                                      │
│                                                │
│  ❯ Deepak             ✓ Ready                  │
│    Rohit Soni         ✓ Ready                  │
│    Personal           ✓ Ready                  │
│                                                │
│  ────────────────────────────────────────────  │
│                                                │
│    Add profile                                 │
│    Settings                                    │
│    Diagnostics                                 │
│    Exit                                        │
│                                                │
├────────────────────────────────────────────────┤
│ ↑↓ Navigate   Enter Select   a Add   q Quit    │
╰────────────────────────────────────────────────╯
```

---

## 🔄 Rotating Accounts on Limit Exhaustion

If Deepak's account reaches its message/token limit in the middle of a coding task:

```bash
# Transfer active conversation to Rohit's quota
claudeswap switch rohit
```

Claude Code opens under **Rohit's account**, resumes Deepak's conversation transcript with `--resume`, and lets you continue coding without losing any context!

You can toggle shared memory mode anytime:
```bash
claudeswap context on      # Enable shared conversation history across accounts
claudeswap context off     # Keep conversation logs completely isolated
claudeswap context status  # Check current status
```

---

## 💾 Exporting & Importing Chats

### Export Chat Sessions:
```bash
# Export active sessions into a portable archive (.tar.gz)
claudeswap chat export --output my-chats.tar.gz

# Export sessions as human-readable Markdown files (.md)
claudeswap chat export --format md --output ./chat-notes/
```

### Import Chat Sessions:
```bash
# Import an archive into your profiles or shared memory
claudeswap chat import my-chats.tar.gz

# Resume the imported session immediately
claudeswap-rohit --resume
```

### List Discovered Sessions:
```bash
claudeswap chat list
```

---

## 📖 Command Reference

| Command | Description |
|---|---|
| `claudeswap` | Opens the interactive Terminal UI (TUI) |
| `claudeswap switch [target]` | Hand off conversation or switch active profile |
| `claudeswap add [name]` | Adds a new profile and sets up its shortcut |
| `claudeswap list` | Lists all configured profiles and statuses |
| `claudeswap status [profile]` | Displays profile status and directory details |
| `claudeswap launch <profile> [args...]` | Launches Claude Code with the specified profile |
| `claudeswap context [on\|off\|status]` | Enables or disables shared conversation sessions and memory |
| `claudeswap chat export [profile]` | Exports chat sessions to an archive (.tar.gz) or Markdown (.md) |
| `claudeswap chat import <archive>` | Imports chat sessions from an archive into a profile |
| `claudeswap chat list [profile]` | Lists active and past conversation sessions |
| `claudeswap remove <profile>` | Removes profile metadata (with optional file deletion) |
| `claudeswap rename <profile> <new-name>` | Renames a profile and updates shortcut |
| `claudeswap shortcut <profile> [--all]` | Generates or regenerates shortcut launcher scripts |
| `claudeswap set-default <profile>` | Sets the fallback profile for `claudeswap launch` |
| `claudeswap settings` | Displays configuration details, backup state, and memory settings |
| `claudeswap doctor` | Runs diagnostics and checks PATH & CLI health |
| `claudeswap version` | Displays the current version |

---

## 🔒 Security Model

ClaudeSwap operates strictly as an **execution environment coordinator**, never an authentication agent:

```
┌────────────────────────────────────────────────────────┐
│                      ClaudeSwap                        │
│                                                        │
│  • Profile Metadata (`~/.claudeswap/profiles.json`)    │
│  • Isolated Dirs (`~/.claudeswap/profiles/<id>/`)      │
│  • Launcher Scripts (`~/.claudeswap/bin/`)             │
│  • Interactive Terminal UI & Diagnostics               │
└──────────────────────────┬─────────────────────────────┘
                           │ Sets CLAUDE_CONFIG_DIR
                           ↓
┌────────────────────────────────────────────────────────┐
│                     Claude Code                        │
│                                                        │
│  • Authentication & OAuth Handshake                    │
│  • Secret Storage & Session State                      │
│  • LLM Interaction & Agent Tools                       │
└────────────────────────────────────────────────────────┘
```

ClaudeSwap **NEVER**:
- Asks for Claude passwords or OTPs
- Reads, parses, or extracts tokens from credential databases
- Stores, transmits, or logs tokens or cookies
- Intercepts or proxies Claude Code traffic
- Operates remote telemetry or cloud services (100% local-first)

---

## 📁 Filesystem Layout

ClaudeSwap organizes its state in `~/.claudeswap` (overrideable with `CLAUDESWAP_HOME`):

```
~/.claudeswap/
├── config.json          # Global app configuration (schema v1, mode 0600)
├── profiles.json        # Profile metadata registry (schema v1, mode 0600)
├── shared/              # Unified session context (when shared context is on)
│   ├── sessions/
│   ├── projects/
│   ├── plans/
│   └── history.jsonl
├── profiles/            # Isolated Claude Code directories
│   ├── deepak/          # Target for CLAUDE_CONFIG_DIR
│   └── rohit-soni/
├── bin/                 # Executable direct shortcuts
│   ├── claudeswap
│   ├── claudeswap-deepak
│   └── claudeswap-rohit
├── logs/                # Optional sanitized local logs
└── backups/             # Atomic backups of profiles and config
    ├── profiles.json.bak
    └── config.json.bak
```

---

## 🛠️ Development & Testing

```bash
# Run unit & integration tests
make test

# Run linter
make vet

# Build cross-platform releases in dist/
make release
```

---

## 📄 License

MIT License. See [LICENSE](LICENSE) for details.
