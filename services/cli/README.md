# crux — Crux CLI

The developer entry point for Crux Control. Discovers agents, runs managed sessions, and queries session history.

## Installation

### One-liner (part of Crux bundle)

```bash
curl -fsSL https://raw.githubusercontent.com/danycrafts/crux/main/scripts/install.sh | bash
```

### Manual

Download the appropriate binary for your platform from the [Releases](https://github.com/danycrafts/crux/releases) page:

- `crux_linux_amd64`
- `crux_linux_arm64`
- `crux_darwin_amd64`
- `crux_darwin_arm64`
- `crux_windows_amd64.exe`
- `crux_windows_arm64.exe`

Place it in your `PATH`.

## Configuration

The CLI reads environment variables only:

| Variable | Default | Description |
|----------|---------|-------------|
| `CRUX_API_URL` | `http://localhost:8080` | URL of the cruxd daemon |

## Interactive PTY

When `crux run` or `crux attach` is executed from a terminal, the CLI:
1. Opens a WebSocket to the daemon's PTY
2. Puts your local terminal into raw mode
3. Streams input/output bidirectionally
4. Propagates terminal resize events (Unix: SIGWINCH, Windows: polling)
5. Restores terminal settings on detach

Detach with `Ctrl-C` or by closing the terminal.

## Usage

```bash
# Initialize crux (starts daemon if needed)
crux init

# Discover installed agents
crux discover

# List registered agents
crux agents

# Run an agent in an interactive PTY session
crux run claude-code --repo ./my-app

# Attach to a running session
crux attach sess_123

# List sessions
crux sessions

# Show session transcript
crux logs sess_123

# Replay session output
crux replay sess_123

# Continue session with another agent
crux continue sess_123 --with gemini-cli

# MCP commands
crux mcp list
crux mcp generate
crux mcp policy
crux mcp policy apply

# Stats
crux stats

# Daemon management
crux daemon start
crux daemon stop
crux daemon status
```

## Uninstall

```bash
rm -f /usr/local/bin/crux
```

On Windows:

```powershell
Remove-Item "$env:LOCALAPPDATA\Crux\bin\crux.exe"
```

## Build from Source

```bash
cd services/cli
go build -o crux ./cmd/crux
```
