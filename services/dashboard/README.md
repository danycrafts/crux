# crux-dashboard — Crux Web Dashboard

A lightweight web dashboard for viewing agents, sessions, and stats. Serves a single-page app that polls the cruxd HTTP API.

## Installation

### One-liner (part of Crux bundle)

```bash
curl -fsSL https://raw.githubusercontent.com/danycrafts/crux/main/scripts/install.sh | bash
```

### Manual

Download the appropriate binary for your platform from the [Releases](https://github.com/danycrafts/crux/releases) page:

- `crux-dashboard_linux_amd64`
- `crux-dashboard_linux_arm64`
- `crux-dashboard_darwin_amd64`
- `crux-dashboard_darwin_arm64`
- `crux-dashboard_windows_amd64.exe`
- `crux-dashboard_windows_arm64.exe`

Place it in your `PATH`.

## Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| `CRUX_DASHBOARD_PORT` | `3001` | Dashboard HTTP port |
| `CRUX_API_URL` | `http://localhost:8080` | cruxd API URL |

## Usage

```bash
# Start the dashboard
crux-dashboard

# Open browser
open http://localhost:3001
```

## Features

- Real-time session overview
- Agent inventory table
- Aggregate stats (sessions, tool calls, cost)
- Transcript viewer
- Auto-refresh every 10 seconds

## Uninstall

```bash
rm -f /usr/local/bin/crux-dashboard
```

On Windows:

```powershell
Remove-Item "$env:LOCALAPPDATA\Crux\bin\crux-dashboard.exe"
```

## Build from Source

```bash
cd services/dashboard
go build -o crux-dashboard ./cmd/dashboard
```
