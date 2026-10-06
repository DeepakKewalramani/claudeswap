# Contributing to ClaudeSwap

Thank you for your interest in contributing to **ClaudeSwap**! We welcome bug reports, feature requests, documentation improvements, and pull requests.

---

## Architecture Principles

When contributing code, please keep these non-negotiable principles in mind:

1. **Authentication Boundary:** ClaudeSwap NEVER handles, parses, stores, transmits, or extracts Claude tokens, passwords, cookies, or OTPs. Authentication is strictly delegated to Claude Code.
2. **Local-First:** ClaudeSwap has no remote telemetry, external analytics, or cloud sync.
3. **Cross-Platform:** Code must function identically across macOS, Linux, and Windows. Use Go standard library path utilities (`filepath.Join`, `filepath.Clean`).
4. **Safety & Security:** Every user-supplied input (profile names, shortcuts, archive files) must be strictly validated against path traversal (`../`) and shell injection.

---

## Development Setup

### Prerequisites
- Go 1.22+
- `make`

### Building & Testing

```bash
# Clone the repository
git clone https://github.com/claudeswap/claudeswap.git
cd claudeswap

# Run test suite
make test

# Run linter
make vet

# Build local binary (claudeswap)
make build
```

---

## Submitting Pull Requests

1. Fork the repository and create your feature branch: `git checkout -b feature/my-feature`.
2. Write clean code with tests covering new functionality.
3. Ensure `make test` and `make vet` pass with 0 errors.
4. Commit with clear, descriptive commit messages.
5. Push to your fork and submit a Pull Request!
