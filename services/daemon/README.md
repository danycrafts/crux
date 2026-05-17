# cruxd — Crux Daemon

The local control plane for Crux. Manages agent processes via PTY, records sessions to SQLite, and exposes an HTTP API for the CLI and dashboard.

## Installation

### One-liner (part of Crux bundle)

```bash
curl -fsSL https://raw.githubusercontent.com/danycrafts/crux/main/scripts/install.sh | bash
```

### Manual

Download the appropriate binary for your platform from the [Releases](https://github.com/danycrafts/crux/releases) page:

- `cruxd_linux_amd64`
- `cruxd_linux_arm64`
- `cruxd_darwin_amd64`
- `cruxd_darwin_arm64`
- `cruxd_windows_amd64.exe`
- `cruxd_windows_arm64.exe`

Place it in your `PATH` (e.g., `/usr/local/bin` or `C:\Users\<You>\AppData\Local\Crux\bin`).

## Configuration

On first start, `cruxd` creates a default config at:

- **macOS/Linux:** `~/.crux/crux.yaml`
- **Windows:** `%LOCALAPPDATA%\Crux\crux.yaml`

Example:

```yaml
api_port: 8080
data_dir: /home/user/.crux
agents:
  claude-code:
    type: cli
    command: claude
    capabilities:
      - code_edit
      - shell
      - mcp
mcp:
  port: 3000
  servers:
    filesystem:
      transport: stdio
      command: npx
      args:
        - -y
        - "@modelcontextprotocol/server-filesystem"
policies:
  deny:
    - email.send
  require_approval:
    - github.pr.merge
logging:
  level: info
  format: text
  file: /home/user/.crux/logs/cruxd.log
  max_size_mb: 100
  max_backups: 3
  max_age_days: 30
```

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `CRUX_API_PORT` | `8080` | HTTP API port |

## Logging

Logs are written to `~/.crux/logs/cruxd.log` by default with rotation:
- Max size: 100 MB per file
- Max backups: 3 files
- Max age: 30 days
- Compression enabled

Set `logging.level` to `debug`, `info`, `warn`, or `error`.

## Usage

```bash
# Start the daemon
cruxd

# Or via CLI wrapper
crux daemon start

# Stop
crux daemon stop
```

## API Endpoints

| Method | Path | Description |
|--------|------|-------------|
| GET | `/health` | Health check |
| POST | `/discover` | Discover agents and MCP servers |
| GET | `/agents` | List registered agents |
| POST | `/agents/{id}/run` | Start an agent session |
| POST | `/sessions/{id}/input` | Send input to a session (HTTP) |
| GET | `/sessions/{id}/attach` | **WebSocket** attach to PTY |
| POST | `/sessions/{id}/resize` | Resize PTY terminal |
| POST | `/sessions/{id}/stop` | Stop a running session |
| GET | `/sessions` | List sessions |
| GET | `/sessions/{id}` | Get session details |
| GET | `/sessions/{id}/logs` | Get transcript |
| GET | `/sessions/{id}/events` | Get events |
| POST | `/sessions/{id}/continue` | Continue with another agent |
| GET | `/mcp/servers` | List MCP servers |
| POST | `/mcp/generate` | Generate agentgateway config |
| GET | `/mcp/policy` | Get policies |
| POST | `/mcp/policy` | Update policies |
| GET | `/stats` | Aggregate stats |

## Data Storage

- **SQLite:** `~/.crux/crux.db`
- **Transcripts:** stored in the database
- **Gateway configs:** `~/.crux/gateway/`

## Uninstall

```bash
rm -f /usr/local/bin/cruxd
rm -rf ~/.crux
```

On Windows:

```powershell
Remove-Item -Recurse -Force "$env:LOCALAPPDATA\Crux"
```

## Build from Source

```bash
cd services/daemon
go build -o cruxd ./cmd/cruxd
```
