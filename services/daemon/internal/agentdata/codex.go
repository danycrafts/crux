package agentdata

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// DiscoverCodex parses Codex CLI's local data.
func DiscoverCodex() *DiscoveredAgent {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil
	}
	dataDir := filepath.Join(home, ".codex")
	if _, err := os.Stat(dataDir); os.IsNotExist(err) {
		return nil
	}

	agent := &DiscoveredAgent{
		ID:           "codex",
		Name:         "Codex CLI",
		Command:      "codex",
		Provider:     "openai",
		DataDir:      dataDir,
		Capabilities: []string{"code_edit", "shell", "mcp", "plugins", "sandbox", "skills"},
		Config:       make(map[string]interface{}),
	}

	// Version
	versionPath := filepath.Join(dataDir, "version.json")
	if data, err := os.ReadFile(versionPath); err == nil {
		var v struct {
			LatestVersion string `json:"latest_version"`
		}
		_ = json.Unmarshal(data, &v)
		agent.Version = v.LatestVersion
	}

	// Config.toml as raw text (too complex to parse without toml lib)
	configPath := filepath.Join(dataDir, "config.toml")
	if data, err := os.ReadFile(configPath); err == nil {
		agent.Config["config_toml"] = string(data)
		// Extract MCP servers via simple string parsing
		lines := strings.Split(string(data), "\n")
		inMCP := false
		var currentMCP *MCPServer
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "[mcp_servers.") {
				inMCP = true
				name := strings.TrimPrefix(line, "[mcp_servers.")
				name = strings.TrimSuffix(name, "]")
				currentMCP = &MCPServer{Name: name, Transport: "stdio"}
			} else if strings.HasPrefix(line, "[") && inMCP {
				inMCP = false
				if currentMCP != nil {
					agent.MCPServers = append(agent.MCPServers, *currentMCP)
					currentMCP = nil
				}
			}
			if inMCP && currentMCP != nil {
				if strings.HasPrefix(line, "command = ") {
					currentMCP.Command = strings.Trim(strings.TrimPrefix(line, "command = "), `"`)
				}
				if strings.HasPrefix(line, "args = ") {
					// Simple args extraction
					argStr := strings.TrimPrefix(line, "args = ")
					argStr = strings.Trim(argStr, "[]")
					parts := strings.Split(argStr, ",")
					for _, p := range parts {
						p = strings.TrimSpace(p)
						p = strings.Trim(p, `"`)
						if p != "" {
							currentMCP.Args = append(currentMCP.Args, p)
						}
					}
				}
			}
		}
		if currentMCP != nil {
			agent.MCPServers = append(agent.MCPServers, *currentMCP)
		}
	}

	// Parse history.jsonl
	historyPath := filepath.Join(dataDir, "history.jsonl")
	if f, err := os.Open(historyPath); err == nil {
		defer f.Close()
		scanner := bufio.NewScanner(f)
		sessionMap := make(map[string]*AgentSession)
		for scanner.Scan() {
			var entry struct {
				SessionID string `json:"session_id"`
				Ts        int64  `json:"ts"`
				Text      string `json:"text"`
			}
			_ = json.Unmarshal(scanner.Bytes(), &entry)
			if entry.SessionID == "" {
				continue
			}
			sess, ok := sessionMap[entry.SessionID]
			if !ok {
				sess = &AgentSession{
					ID:     entry.SessionID,
					Status: "unknown",
				}
				sessionMap[entry.SessionID] = sess
			}
			ts := time.Unix(entry.Ts, 0)
			if sess.StartedAt.IsZero() || ts.Before(sess.StartedAt) {
				sess.StartedAt = ts
			}
			if sess.EndedAt == nil || ts.After(*sess.EndedAt) {
				sess.EndedAt = &ts
			}
			agent.Stats.TotalMessages++
		}
		for _, sess := range sessionMap {
			agent.Sessions = append(agent.Sessions, *sess)
		}
		agent.Stats.TotalSessions = len(agent.Sessions)
	}

	// Parse session files from sessions/YYYY/MM/DD/*.jsonl
	sessionsDir := filepath.Join(dataDir, "sessions")
	_ = filepath.Walk(sessionsDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || !strings.HasSuffix(path, ".jsonl") {
			return nil
		}
		// Extract session ID from filename
		base := filepath.Base(path)
		// Format: rollout-YYYY-MM-DDTHH-MM-SS-<uuid>.jsonl
		parts := strings.Split(base, "-")
		if len(parts) >= 2 {
			// The UUID is the last part before .jsonl
			uuidPart := strings.TrimSuffix(parts[len(parts)-1], ".jsonl")
			if len(parts) >= 8 {
				// Reconstruct UUID from parts
				uuidPart = parts[len(parts)-5] + "-" + parts[len(parts)-4] + "-" + parts[len(parts)-3] + "-" + parts[len(parts)-2] + "-" + parts[len(parts)-1]
				uuidPart = strings.TrimSuffix(uuidPart, ".jsonl")
			}
			// Parse first line for metadata
			if f, err := os.Open(path); err == nil {
				scanner := bufio.NewScanner(f)
				if scanner.Scan() {
					var meta struct {
						Timestamp string `json:"timestamp"`
						Type      string `json:"type"`
						Payload   struct {
							ID       string `json:"id"`
							CWD      string `json:"cwd"`
							Model    string `json:"model"`
							Source   string `json:"source"`
						} `json:"payload"`
					}
					_ = json.Unmarshal(scanner.Bytes(), &meta)
					if meta.Payload.CWD != "" {
						found := false
						for i := range agent.Sessions {
							if agent.Sessions[i].ID == meta.Payload.ID {
								agent.Sessions[i].RepoPath = meta.Payload.CWD
								found = true
								break
							}
						}
						if !found {
							started, _ := time.Parse(time.RFC3339, meta.Timestamp)
							agent.Sessions = append(agent.Sessions, AgentSession{
								ID:        meta.Payload.ID,
								RepoPath:  meta.Payload.CWD,
								StartedAt: started,
								Status:    "completed",
							})
						}
					}
				}
				f.Close()
			}
		}
		return nil
	})

	return agent
}
