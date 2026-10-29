# ClaudeSwap

<p align="center">
  <strong>Instant Account Swapper, Multi-Session Matrix & Shared Memory Continuity for Claude Code</strong>
</p>

<p align="center">
  <a href="LICENSE"><img src="https://img.shields.io/badge/license-MIT-blue.svg?style=flat-square" alt="License"></a>
  <a href="https://golang.org"><img src="https://img.shields.io/badge/go-1.22%2B-00ADD8.svg?style=flat-square" alt="Go Version"></a>
  <a href="#"><img src="https://img.shields.io/badge/version-1.1.0-FF5722.svg?style=flat-square" alt="Version"></a>
  <a href="#"><img src="https://img.shields.io/badge/platform-macOS%20%7C%20Linux%20%7C%20Windows-lightgrey.svg?style=flat-square" alt="Platforms"></a>
  <a href="#"><img src="https://img.shields.io/badge/security-zero--credential%20storage-brightgreen.svg?style=flat-square" alt="Security"></a>
</p>

---

**ClaudeSwap** is a developer tool written in Go that lets you manage, isolate, and instantly switch between multiple Claude Code profiles with **zero downtime, shared memory continuity, folder-wise session exploration, and multi-agent supervision**.

```
  ┌──────────────────┐       Limit Exhausted?       ┌──────────────────┐
  │  Claude Profile  │ ───────────────────────────> │  Claude Profile  │
  │     [Work]       │    claudeswap handoff        │    [Personal]    │
  └─────────┬────────┘                              └─────────┬────────┘
            │                                                 │
            └───────────────>  Unified Context <──────────────┘
                              (~/.claudeswap/shared)
                              • Full Chat Transcripts
                              • Project Folder Memory
                              • File Edit Caches
                              • Command History
```

> **🛡️ Strict Zero-Credential Boundary:**  
> ClaudeSwap operates with an inviolable security model. It **never** reads, stores, intercepts, or logs Claude tokens, OAuth cookies, passwords, or keychain secrets. Authentication remains 100% native and isolated within each Claude Code profile.

---

## ⚡ The Problems ClaudeSwap Solves

| Challenge | Without ClaudeSwap | With ClaudeSwap |
|---|---|---|
| **Rate Limit / Quota Exhaustion** | Workflow stops for 4+ hours, or you manually log out, re-authenticate, and lose your 50-turn conversation context. | Type `claudeswap handoff` and resume the exact conversation under another account in **under 2 seconds**. |
| **Multi-Repo Parallel Coding** | 5 terminal tabs with background agents hitting hidden permission prompts and process lock conflicts (`daemon.lock`). | `claudeswap matrix` provides a live mission-control dashboard supervising 20+ parallel sessions with instant 1-click attach. |
| **Scattered Past Sessions** | Sessions buried across unreadable hashed folders in `~/.claude/projects/`. | `claudeswap sessions` groups your work folder-wise with deep telemetry, token micro-gauges, and 1–9 tab switching. |
| **Context Fracturing** | Each account starts with zero memory of previous architectures and file edits. | `claudeswap context on` pools transferable project memory and history without sharing credentials. |

---

## 🖥️ Terminal Experience

### 1. Folder-Wise Session Explorer & Telemetry HUD (`claudeswap sessions`)

Browse all active and historical sessions grouped by project workspace, inspect real-time token consumption, and review execution trails before resuming:

```text
 ┌── ClaudeSwap Explorer ──┬── [1] payment-service ──┬── [2] web-dashboard ──┬── Profile: Work (✓ Ready) ── Memory: ON ──┐
 │                                                                                                                       │
 │  WORKSPACES (4)         │  SESSIONS (3)                  │  TELEMETRY INSPECTOR                                      │
 │                         │                                │                                                           │
 │  ❯ payment-service  (3) │  ❯ refactor: auth middleware   │  id:       a8f10b2c                                       │
 │    web-dashboard    (2) │    2m ago • 4 turns • 42k tok  │  model:    claude-3-7-sonnet                              │
 │    worker-daemon    (1) │    [████████░░░░░░░░░░] 42%    │  status:   active                                         │
 │    mobile-client    (0) │                                │  tokens:   42,150 [████████░░░░] 42%                      │
 │                         │    feat: stripe webhooks       │  speed:    58 tok/s                                       │
 │                         │    1h ago • 12 turns • 88k tok │  cost:     $0.14                                          │
 │                         │    [██████████████░░░░] 74%    │                                                           │
 │                         │                                │  thinking: analyzing webhook idempotency...               │
 │                         │    fix: connection timeout     │  trail:    Read file -> Edit file -> Run tests            │
 │                         │    yesterday • 6 turns • 18k   │  files:    middleware/auth.go (+24/-3)                    │
 │                         │                                │            server.go (+12/-1)                             │
 ├─────────────────────────┴────────────────────────────────┴───────────────────────────────────────────────────────────┤
 │ Tab: Switch Pane • ↑↓: Navigate • Enter: Open • f: Fork • s: Cycle Profile • m: Toggle Memory • q: Exit             │
 └──────────────────────────────────────────────────────────────────────────────────────────────────────────────────────┘
```

---

### 2. Multi-Session Supervisor Matrix (`claudeswap matrix`)

Supervise up to 20+ parallel Claude Code background sessions across your repositories in a live status matrix:

```text
 ┌── ClaudeSwap Supervisor Matrix ────────────────────────────── [Active: 3/4] ── Refresh: 1s ──┐
 │                                                                                              │
 │  [1] payment-service                 [2] web-dashboard                                       │
 │  ● RUNNING                           ○ IDLE                                                  │
 │  Session: a8f10b2c (refactor auth)   Session: f3c21a9d (navbar polish)                       │
 │  Model:   claude-3-7-sonnet          Model:   claude-3-5-haiku                               │
 │  Tokens:  [████████░░] 42k (42%)     Tokens:  [████░░░░░░] 18k (18%)                         │
 │  Trail:   running unit tests...      Trail:   awaiting user prompt                           │
 │                                                                                              │
 │  [3] worker-daemon                   [4] mobile-client                                       │
 │  ⚡ WAITING FOR PERMISSION           ○ COMPLETED                                             │
 │  Session: 7b19de33 (sqs consumer)    Session: 2e88a104 (asset optimization)                  │
 │  Model:   claude-3-7-sonnet          Model:   claude-3-5-sonnet                              │
 │  Tokens:  [████████████░░] 78k       Tokens:  [████████████████] 110k                        │
 │  Action:  needs Bash permission      Trail:   finished in 4m 12s                             │
 ├──────────────────────────────────────────────────────────────────────────────────────────────┤
 │ 1-4: Attach/Fork Session • r: Force Refresh • p: Switch Profile • q: Back to Shell          │
 └──────────────────────────────────────────────────────────────────────────────────────────────┘
```

---

### 3. Interactive Profile Dashboard (`claudeswap`)

Manage profiles, inspect quota statuses, and toggle shared context from a clean Bubble Tea interface:

```text
 ╭────────────────────────────────────────────────╮
 │                 ClaudeSwap                     │
 │      Claude Profile Manager & Swapper          │
 ├────────────────────────────────────────────────┤
 │                                                │
 │  Profiles                                      │
 │                                                │
 │  ❯ Work Account        ✓ Ready                 │
 │    Personal Account    ✓ Ready                 │
 │    Consulting Team     ○ Logged out            │
 │                                                │
 │  ────────────────────────────────────────────  │
 │                                                │
 │    Add profile                                 │
 │    Session Explorer (TUI)                      │
 │    Supervisor Matrix                           │
 │    Settings & Memory                           │
 │    Doctor Diagnostics                          │
 │                                                │
 ├────────────────────────────────────────────────┤
 │ ↑↓ Navigate   Enter Select   m Memory   q Quit │
 ╰────────────────────────────────────────────────╯
```

---

## 🚀 Installation

### Option 1: Quick Install Script (macOS & Linux)

```bash
curl -fsSL https://raw.githubusercontent.com/DeepakKewalramani/claudeswap/main/install.sh | sh
```

### Option 2: Go Install (Any Platform)

```bash
go install github.com/DeepakKewalramani/claudeswap/cmd/claudeswap@latest
```

### Option 3: Windows PowerShell

```powershell
irm https://raw.githubusercontent.com/DeepakKewalramani/claudeswap/main/install.ps1 | iex
```

### Option 4: Build from Source

```bash
git clone https://github.com/DeepakKewalramani/claudeswap.git
cd claudeswap
make install
```

Configure shell autocompletion and shortcuts:
```bash
claudeswap doctor --install-shell
source ~/.zshrc    # or ~/.bashrc
```

---

## 💡 Real-World Use Cases & Examples

### Scenario 1: The Mid-Sprint Limit Handoff
You're 60 turns into building a complex feature when Claude Code hits the hourly usage cap:

```bash
# 1. Start the work under your primary profile
claudeswap-work

# 2. [Claude outputs]: "Rate limit reached until 4:00 PM"

# 3. Rotate conversation to your personal account quota seamlessly:
claudeswap handoff personal
```
**What happens behind the scenes:**
1. ClaudeSwap verifies shared context is enabled.
2. It syncs the transcript to `~/.claudeswap/shared/sessions`.
3. It immediately launches Claude Code under the `personal` account using `--resume <session-id>`.
4. Claude resumes with the full conversation history intact.

---

### Scenario 2: Supervising Multiple Repos Simultaneously
You have several Claude Code sessions running across different microservices:

```bash
# Open the Supervisor Matrix
claudeswap matrix
```
- See which sessions are actively computing, idle, or blocked waiting for a shell execution confirmation.
- Press **`1`**, **`2`**, or **`3`** to jump directly into any running session.
- If a background session is currently locked, ClaudeSwap auto-branches a copy with `--fork-session` so you never crash active processes.

---

### Scenario 3: Shared Memory vs Strict Account Isolation
Easily toggle between unified context (for solo devs) and strict isolation (for client work):

```bash
# Enable unified memory (transcripts, project memory, command history shared)
claudeswap context on

# Isolate all accounts into separate silos (ideal for NDAs or distinct clients)
claudeswap context off

# Check current status and active symlinks
claudeswap context status
```

---

### Scenario 4: Fast Direct Shortcuts
ClaudeSwap automatically generates dedicated launcher binaries in your `$PATH` for each profile:

```bash
# Launch specific profiles with any Claude Code flag forwarded transparently:
claudeswap-work
claudeswap-personal --continue
claudeswap-consulting --resume a8f10b2c
```

---

### Scenario 5: Backup & Export Chat Transcripts
Save and share conversation sessions as portable archives or human-readable Markdown:

```bash
# Export active sessions into a portable .tar.gz archive
claudeswap chat export --output ./workspace-chats.tar.gz

# Export sessions as clean Markdown notes
claudeswap chat export --format md --output ./meeting-notes/

# Import an archive on another machine
claudeswap chat import ./workspace-chats.tar.gz
```

---

## 📖 Command Reference

| Command | Description |
|---|---|
| `claudeswap` | Opens the interactive Bubble Tea dashboard |
| `claudeswap sessions` | Full-screen folder-wise session explorer & telemetry HUD |
| `claudeswap matrix` | Multi-session supervisor matrix for parallel Claude tasks |
| `claudeswap handoff [profile]` | Seamlessly transfer active conversation to another account quota |
| `claudeswap agents [flags]` | Inspect, attach to, or stop Claude background agents |
| `claudeswap launch <profile> [flags]` | Launch Claude Code isolated under a specific profile |
| `claudeswap context [on\|off\|status]` | Enable or disable unified conversation memory across profiles |
| `claudeswap add [name]` | Register a new isolated profile and generate shortcuts |
| `claudeswap list` | Display all configured profiles and operational statuses |
| `claudeswap status [profile]` | Inspect profile configuration and directory health |
| `claudeswap chat export [profile]` | Export chat sessions to `.tar.gz` archive or Markdown notes |
| `claudeswap chat import <archive>` | Import conversation sessions into local profiles |
| `claudeswap chat list [profile]` | Quick CLI list of active and past conversation logs |
| `claudeswap remove <profile>` | Remove a profile (with option to preserve configuration files) |
| `claudeswap rename <profile> <new>` | Rename profile and update direct shortcut launcher |
| `claudeswap set-default <profile>` | Set default profile for launches |
| `claudeswap doctor` | System health check (binary paths, permissions, shell setup) |
| `claudeswap version` | Display current ClaudeSwap version |

---

## 🔒 Security Architecture

ClaudeSwap acts purely as an **environment coordinator**, never an authentication agent:

```
┌────────────────────────────────────────────────────────┐
│                      ClaudeSwap                        │
│                                                        │
│  • Profile Registry (`~/.claudeswap/profiles.json`)    │
│  • Isolated Dirs (`~/.claudeswap/profiles/<id>/`)      │
│  • Direct Shortcuts (`~/.claudeswap/bin/`)             │
│  • Shared Context Symlinks (`~/.claudeswap/shared/`)   │
│  • Multi-Session Matrix & Telemetry HUD                │
└──────────────────────────┬─────────────────────────────┘
                           │ Injects CLAUDE_CONFIG_DIR
                           ↓
┌────────────────────────────────────────────────────────┐
│                     Claude Code                        │
│                                                        │
│  • Native OAuth Handshake & Token Storage              │
│  • Local Secrets (`.claude.json`)                      │
│  • LLM Interaction & Agent Tools                       │
└────────────────────────────────────────────────────────┘
```

ClaudeSwap **NEVER**:
- Accesses, copies, or logs tokens in `.claude.json` or OS keychains.
- Proxies or intercepts network traffic.
- Transmits data to external servers (100% local-first and offline-capable).

---

## 🔒 Security & Bug-Free Architecture

ClaudeSwap is engineered with security-first invariants verified by automated fuzzing and `-race` detection:

1. **Zero Credential Touch:** ClaudeSwap **never** parses, opens, or logs OAuth tokens, session cookies, or secrets inside `.claude.json` or system keychains. Profile switching is managed purely via `CLAUDE_CONFIG_DIR`.
2. **Zip-Slip & Path Traversal Immunity:** Archive importers (`chat import`) strictly validate canonical paths with `filepath.Clean` and reject entries with `..`, absolute paths, or destinations outside the target directory.
3. **Command Injection Immunity:** Subprocesses (`claude`) are spawned using direct argv slices (`exec.Command(path, args...)`), completely bypassing shell interpreters (`bash -c`, `sh -c`).
4. **Race-Free Concurrency:** All shared memory operations, session scanning, and supervisor state pools are fully thread-safe and verified with `go test -race ./...`.
5. **Atomic Storage & Auto-Healing:** All metadata files (`profiles.json`, `config.json`) use atomic file write-and-rename mechanics (`.tmp` -> `os.Rename`) with automatic `.bak` self-healing on corruption.
6. **Strict POSIX Permissions:** All created folders are restricted to `0700` and config files to `0600` (readable and writable strictly by the current OS user).

---

## 🔍 Frequently Asked Questions (Search & Troubleshooting)

<details>
<summary><strong>Q: What should I do when Claude Code says "Rate limit reached until X:XX"?</strong></summary>
<br>

Run:
```bash
claudeswap handoff
```
Or specify your secondary profile directly:
```bash
claudeswap handoff <secondary-profile>
```
ClaudeSwap automatically verifies that shared context is active, syncs the conversation transcript, and opens Claude Code with `--resume <session-id>` under the secondary profile's quota.
</details>

<details>
<summary><strong>Q: How do I share project memory across accounts without leaking credentials?</strong></summary>
<br>

Run:
```bash
claudeswap context on
```
This safely symlinks non-credential directories (`sessions/`, `projects/`, `file-history/`, `plans/`, and `history.jsonl`) to `~/.claudeswap/shared/`. Your `.claude.json` tokens stay 100% private to each profile folder.
</details>

<details>
<summary><strong>Q: How do I avoid "daemon.lock" conflicts when running multiple sessions?</strong></summary>
<br>

When resuming an active session or inspecting background workers, run:
```bash
claudeswap sessions resume <session-id> --fork
```
Or in the visual TUI (`claudeswap sessions` or `claudeswap matrix`), press **`f`** to auto-branch the session using Claude Code's `--fork-session` flag.
</details>

<details>
<summary><strong>Q: How does ClaudeSwap isolate profiles under the hood?</strong></summary>
<br>

Claude Code natively reads the environment variable `CLAUDE_CONFIG_DIR`. ClaudeSwap assigns a unique directory for each profile (`~/.claudeswap/profiles/<profile-id>`). Each profile has its own isolated OAuth authentication, settings, and cache.
</details>

<details>
<summary><strong>Q: Can I run multiple profiles concurrently in separate terminals?</strong></summary>
<br>

Yes! Because each profile has its own `CLAUDE_CONFIG_DIR`, you can run `claudeswap-work` in Terminal 1 and `claudeswap-personal` in Terminal 2 simultaneously without any process lock collisions.
</details>

---

## 📁 Filesystem Layout

```
~/.claudeswap/
├── config.json          # Global app configuration (mode 0600)
├── profiles.json        # Profile metadata registry (mode 0600)
├── shared/              # Unified session context pool
│   ├── sessions/        # Conversation transcripts
│   ├── projects/        # Workspace memory mappings
│   ├── file-history/    # Edit caches
│   ├── plans/           # Active plans
│   └── history.jsonl    # Unified command history
├── profiles/            # Isolated Claude Code directories
│   ├── work/            # Target for CLAUDE_CONFIG_DIR
│   └── personal/
├── bin/                 # Generated executable shortcuts
│   ├── claudeswap
│   ├── claudeswap-work
│   └── claudeswap-personal
└── backups/             # Atomic backups of registry and configuration
    ├── profiles.json.bak
    └── config.json.bak
```

---

## 🛠️ Testing & Quality

```bash
# Run unit & integration test suites
go test -v ./...

# Run static analysis
go vet ./...

# Build cross-platform release binaries
make release
```

---

## 📄 License

MIT License. See [LICENSE](LICENSE) for details.
