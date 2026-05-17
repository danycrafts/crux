# Crux Control

**Crux Control** is a vendor-neutral control plane for operating autonomous coding-agent fleets.

It discovers, governs, routes, observes, and coordinates coding agents across vendors, teams, machines, and environments.

## Quick Start

One-liner install:

### macOS / Linux

```bash
curl -fsSL https://raw.githubusercontent.com/danycrafts/crux/main/scripts/install.sh | bash
```

### Windows (PowerShell)

```powershell
irm https://raw.githubusercontent.com/danycrafts/crux/main/scripts/install.ps1 | iex
```

## Services

Crux is composed of three separate Go services:

| Service | Binary | Port | Purpose |
|---------|--------|------|---------|
| [CLI](./services/cli) | `crux` | — | Developer entry point. Discovers, runs, and manages agents. |
| [Daemon](./services/daemon) | `cruxd` | `8080` | Local control plane. PTY runner, SQLite store, HTTP API. |
| [Dashboard](./services/dashboard) | `crux-dashboard` | `3001` | Web UI for sessions, agents, and stats. |

## Architecture

```
  crux CLI
     |
     v
  cruxd (HTTP API)
     |
     +-- SQLite (sessions, agents, transcripts)
     +-- PTY Runner (claude, codex, gemini, ...)
     +-- MCP Gateway Config (agentgateway.yaml)
     |
     v
  crux-dashboard (Web UI)
```

## Supported Platforms

| OS | Architecture |
|----|-------------|
| macOS | amd64, arm64 |
| Linux (Ubuntu, Debian, Fedora, RHEL, Alpine, Arch) | amd64, arm64, arm |
| Windows 10/11 | amd64, arm64 |

## Build from Source

Requires Go 1.22+.

```bash
make build
make install
```

## Usage

```bash
# Initialize and start the daemon
crux init

# Discover installed agents
crux discover

# Run an agent in an interactive PTY
crux run claude-code --repo ./my-app

# Attach to a running session
crux attach sess_123

# List sessions
crux sessions

# View logs
crux logs sess_123

# Continue session with another agent
crux continue sess_123 --with gemini-cli

# MCP gateway
crux mcp list
crux mcp generate

# Web dashboard
crux-dashboard
```

## Uninstall

```bash
# macOS / Linux
rm -f /usr/local/bin/crux /usr/local/bin/cruxd /usr/local/bin/crux-dashboard
rm -rf ~/.crux

# Windows
Remove-Item -Recurse -Force "$env:LOCALAPPDATA\Crux"
```

## License

MIT
