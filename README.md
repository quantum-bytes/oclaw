# oclaw

A terminal UI for [OpenClaw](https://github.com/openclaw/openclaw) — interactive agent chat with streaming, session persistence, and multi-agent switching.

Built because the built-in `openclaw tui` is broken. `oclaw` connects directly to the gateway WebSocket, giving you a fast, reliable, interactive chat experience that doesn't time out.

## Features

- **Interactive chat** — persistent sessions, no timeouts
- **Streaming responses** — real-time token-by-token display
- **Thinking model support** — see reasoning with toggle (Ctrl+T)
- **Multi-agent switching** — switch between agents mid-conversation (Ctrl+A)
- **Session management** — browse and resume sessions (Ctrl+S)
- **Markdown rendering** — code blocks, lists, formatting in the terminal
- **Auto-reconnect** — exponential backoff, never drops your session
- **Zero config** — auto-discovers gateway from `~/.openclaw/openclaw.json`

## Install

```bash
go install github.com/quantum-bytes/oclaw@latest
```

Or download a binary from [Releases](https://github.com/quantum-bytes/oclaw/releases).

## Usage

```bash
# Launch with auto-discovered config
oclaw

# Connect to specific gateway
oclaw --url ws://localhost:39421 --token mytoken

# Start with a specific agent
oclaw --agent quasar
```

## Keybindings

| Key | Action |
|-----|--------|
| `Enter` | Send message |
| `Shift+Enter` | New line |
| `Ctrl+C` | Abort response / quit |
| `Ctrl+D` | Quit |
| `Ctrl+A` | Switch agent |
| `Ctrl+S` | Browse sessions |
| `Ctrl+N` | New session |
| `Ctrl+T` | Toggle thinking text |
| `Ctrl+/` | Help |
| `Esc` | Close overlay |

## Slash Commands

| Command | Action |
|---------|--------|
| `/agent <id>` | Switch to agent |
| `/session` | Browse sessions |
| `/new` | Reset session |
| `/think <level>` | Set thinking level |
| `/help` | Show help |
| `/quit` | Quit |

## Configuration

`oclaw` reads configuration from multiple sources (highest priority first):

1. CLI flags (`--url`, `--token`, `--agent`)
2. Environment variables (`OPENCLAW_GATEWAY_URL`, `OPENCLAW_GATEWAY_TOKEN`, `OPENCLAW_AGENT`)
3. `~/.openclaw/openclaw.json` (gateway URL, token, agent list)
4. Defaults (`ws://127.0.0.1:39421`, token `ollama`)

## Architecture

```
oclaw ──WebSocket──► OpenClaw Gateway ──► Model APIs (Gemini, GPT, Ollama)
                          │
                          ▼
                    Session Store (JSONL)
```

`oclaw` talks directly to the gateway via the OpenClaw WebSocket RPC protocol. No CLI wrapper, no shell-out, no timeouts.

## Building

```bash
git clone https://github.com/quantum-bytes/oclaw
cd oclaw
make build
./oclaw
```

## License

MIT
