---
name: claudeswap-release
description: >-
  Use this skill whenever preparing, bumping, verifying, or publishing a new release
  of ClaudeSwap. Guides version synchronization, testing, and git release tagging.
---

# ClaudeSwap Release Runbook

Follow this step-by-step procedure to prepare and publish a release according to Semantic Versioning (`MAJOR.MINOR.PATCH`).

## 1. Determine the Version Bump
- **PATCH** (`x.y.Z`): Backward-compatible bug fixes or internal refactoring.
- **MINOR** (`x.Y.0`): Backward-compatible new features (e.g., new commands, UI components, supervisor upgrades).
- **MAJOR** (`X.0.0`): Breaking changes to CLI syntax, storage schema, or configuration format.

## 2. Synchronize Version Declarations
Update the version string across all 5 required files in lockstep:

1. `internal/cli/version.go`:
   ```go
   const Version = "<NEW_VERSION>"
   ```
2. `Makefile`:
   ```makefile
   VERSION=<NEW_VERSION>
   ```
3. `install.sh`:
   ```bash
   VERSION="<NEW_VERSION>"
   ```
4. `install.ps1`:
   ```powershell
   $Version = "<NEW_VERSION>"
   ```
5. `CHANGELOG.md`:
   Add a new section under `## [<NEW_VERSION>] - YYYY-MM-DD` following Keep a Changelog conventions (`Added`, `Changed`, `Fixed`, `Removed`).

## 3. Mandatory Pre-Release Verification
Run all automated checks:

```bash
# 1. Run all unit and integration tests
go test -v ./...

# 2. Static analysis
go vet ./...

# 3. Test compilation
go build -o /dev/null ./cmd/claudeswap

# 4. Cross-compilation smoke checks
GOOS=linux GOARCH=amd64 go build -o /dev/null ./cmd/claudeswap
GOOS=windows GOARCH=amd64 go build -o /dev/null ./cmd/claudeswap
```

## 4. Rebuild Local Binary
Update the local installed binary so the local environment stays current:
```bash
go build -ldflags="-s -w" -o /Users/tequity/.claudehub/bin/claudeswap ./cmd/claudeswap
/Users/tequity/.claudehub/bin/claudeswap version
```

## 5. Git Commit & Tagging
Stage, commit, and tag the release:
```bash
git add internal/cli/version.go Makefile install.sh install.ps1 CHANGELOG.md
git commit -m "chore(release): bump version to v<NEW_VERSION>"
git tag -a "v<NEW_VERSION>" -m "Release v<NEW_VERSION>"
```

