---
name: claudeswap-tui-dev
description: >-
  Use this skill when developing, refactoring, or polishing Bubble Tea and Lip Gloss
  Terminal User Interfaces (TUI) in ClaudeSwap (e.g., session explorer, supervisor matrix, HUD).
---

# ClaudeSwap TUI Development Guidelines

ClaudeSwap's TUI is designed with a minimal, responsive, high-density terminal interface inspired by modern hacker tools.

## 1. Visual & Styling Principles
- **Minimal Lowercase Typography:** Use understated labels (e.g., `model`, `trail`, `files`, `status`) to keep focus on actionable telemetry.
- **Zero Visual Clutter:** Avoid dense borders, heavy box drawing, and nested banners where clean single-line dividers suffice.
- **Micro-Gauges:** Render progress or token usage with compact bar indicators (e.g., `[████░░░░░░] 42%`).
- **Responsive Layout:**
  - Always handle `tea.WindowSizeMsg` in your `Update()` loop.
  - Dynamically recalculate column widths, sidebar widths, and list heights.
  - Truncate long strings cleanly using `mattn/go-runewidth` to prevent unintended line wrapping.

## 2. Standard Keybindings Pattern
Maintain consistent keyboard navigation across all TUI models:
- `Tab` / `Shift+Tab`: Switch pane focus (e.g., sidebar ↔ main list ↔ inspector).
- `Up` / `k`, `Down` / `j`: Move selection up and down.
- `1` – `9`: Instant switch to numbered tabs or session cards.
- `s` / `p`: Cycle active profile in-place without quitting.
- `m`: Instant toggle for shared memory.
- `Enter`: Open / resume session (or auto-fork if live).
- `f`: Fork session (`--fork-session`).
- `d`: Delete session (with inline confirmation prompt).
- `Esc` / `q`: Back or quit.

## 3. Terminal Safety & Raw Mode
- When reading raw terminal keystrokes outside Bubble Tea (e.g. quick confirmation menus):
  ```go
  if term.IsTerminal(int(os.Stdin.Fd())) {
      oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
      if err == nil {
          defer term.Restore(int(os.Stdin.Fd()), oldState)
      }
  }
  ```
- Always ensure `term.Restore()` is deferred immediately to prevent leaving the user's terminal in raw state.

