# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.1.0] - 2026-03-16

### Added

- Interactive TUI chat with Bubbletea framework
- Streaming responses with real-time token-by-token display
- Thinking model support with toggle visibility (Ctrl+T)
- Multi-agent switching mid-conversation (Ctrl+A)
- Session management: browse, resume, and reset sessions
- Memory persistence via `/save` command
- Markdown rendering with Glamour (code blocks, lists, headings)
- Clickable OSC 8 terminal hyperlinks
- Animated thinking indicator with shimmer gradient
- Slash command autocomplete
- Auto-reconnect with exponential backoff
- Scrollable chat (mouse wheel, PgUp/PgDn, Ctrl+Home/End)
- Zero-config auto-discovery from `~/.openclaw/openclaw.json`
- Ed25519 device signing authentication
- Token-based authentication fallback
- Security hardening: ANSI/OSC sanitization, URL validation, secure key handling
- Debug logging opt-in with secure temp files
- Cross-platform builds (macOS, Linux, Windows on amd64/arm64)
