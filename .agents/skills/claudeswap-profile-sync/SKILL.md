---
name: claudeswap-profile-sync
description: >-
  Use this skill to diagnose, repair, or inspect ClaudeSwap profile directories,
  shared context symlinks, broken session states, and handoff synchronization.
---

# ClaudeSwap Profile & Context Diagnostics Runbook

Use this runbook to inspect and resolve profile storage, context sharing, or session synchronization issues.

## 1. Quick Diagnostics
Run the built-in diagnostic suite:
```bash
claudeswap doctor
claudeswap context status
claudeswap list
```

## 2. Inspect Profile Storage Integrity
Check profile directories and symlinks:
```bash
# List all profile directories
ls -la ~/.claudeswap/profiles/

# Check shared pool components
ls -la ~/.claudeswap/shared/
```

Verify that for each profile with shared context active:
- `sessions/` -> symlink to `~/.claudeswap/shared/sessions`
- `projects/` -> symlink to `~/.claudeswap/shared/projects`
- `file-history/` -> symlink to `~/.claudeswap/shared/file-history`
- `plans/` -> symlink to `~/.claudeswap/shared/plans`
- `history.jsonl` -> symlink to `~/.claudeswap/shared/history.jsonl`
- `.claude.json` is a **regular file**, NOT a symlink.

## 3. Repairing Broken or Desynced Context
If a profile directory has dangling symlinks or missing folders:
1. Re-run context sync explicitly:
   ```bash
   claudeswap context on
   ```
2. If a specific profile is corrupted, inspect backup configurations:
   ```bash
   ls -la ~/.claudeswap/profiles/<profile-id>/backups/
   ```
3. To safely rebuild a profile's context symlinks programmatically without touching `.claude.json`:
   ```bash
   claudeswap context off
   claudeswap context on
   ```

## 4. Recovering Configuration Database
If `~/.claudeswap/profiles.json` or `config.json` is corrupted:
- ClaudeSwap maintains atomic write backups (`profiles.json.bak`, `config.json.bak`).
- Compare current file against `.bak` and restore if the primary file has invalid JSON syntax.

