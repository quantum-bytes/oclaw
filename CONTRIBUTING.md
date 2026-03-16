# Contributing to oclaw

Thanks for your interest in contributing! Here's how to get started.

## Development Setup

```bash
git clone https://github.com/quantum-bytes/oclaw
cd oclaw
make build
```

Requirements: Go version matching `go.mod` (currently 1.26+)

## Building and Testing

```bash
make build      # Build binary
make test       # Run tests
make lint       # Run linter (requires golangci-lint)
make fmt        # Format code
make vet        # Run go vet
```

## Submitting Changes

1. Fork the repository
2. Create a feature branch (`git checkout -b feat/my-feature`)
3. Make your changes
4. Ensure `make fmt vet test` passes
5. Commit with a descriptive message (see below)
6. Push and open a Pull Request

## Commit Messages

Follow conventional commit style:

- `feat: add agent filtering` — new feature
- `fix: handle nil session` — bug fix
- `docs: update keybindings table` — documentation
- `refactor: extract message parser` — code restructure
- `test: add gateway client tests` — tests
- `chore: update dependencies` — maintenance

## Code Style

- Run `go fmt ./...` before committing
- Run `go vet ./...` to catch issues
- Keep functions focused and short
- Add tests for new functionality

## Reporting Issues

- Use the GitHub issue templates (bug report or feature request)
- Include your OS, terminal emulator, and oclaw version
- For bugs, include steps to reproduce and debug logs (`OCLAW_DEBUG=1`)

## Code Review

All PRs require review before merging. Reviewers will check:

- Correctness and test coverage
- Code style and readability
- Security implications (especially for input handling and auth)
