package agentdata

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// DiscoverClaude parses Claude Code's local data.
func DiscoverClaude() *DiscoveredAgent {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil
	}
	dataDir := filepath.Join(home, ".claude")
	if _, err := os.Stat(dataDir); os.IsNotExist(err) {
		return nil
	}

	agent := &DiscoveredAgent{
		ID:           "claude-code",
		Name:         "Claude Code",
		Command:      "claude",
		Provider:     "anthropic",
		DataDir:      dataDir,
		Capabilities: []string{"code_edit", "shell", "mcp", "repo_search", "skills", "plugins"},
		Config:       make(map[string]interface{}),
	}

	// Version from settings or fallback
	agent.Version = readClaudeVersion(dataDir)

	// Parse settings.json
	settingsPath := filepath.Join(dataDir, "settings.json")
	if data, err := os.ReadFile(settingsPath); err == nil {
		var settings map[string]interface{}
		_ = json.Unmarshal(data, &settings)
		agent.Config["settings"] = settings
		// Extract MCP servers from plugins if present
		if plugins, ok := settings["enabledPlugins"].(map[string]interface{}); ok {
			for name := range plugins {
				agent.MCPServers = append(agent.MCPServers, MCPServer{
					Name:      strings.Split(name, "@")[0],
					Transport: "plugin",
				})
			}
		}
	}

	// Parse stats-cache.json
	statsPath := filepath.Join(dataDir, "stats-cache.json")
	if data, err := os.ReadFile(statsPath); err == nil {
		var stats struct {
			Version         int `json:"version"`
			DailyActivity   []struct {
				Date          string `json:"date"`
				MessageCount  int    `json:"messageCount"`
				SessionCount  int    `json:"sessionCount"`
				ToolCallCount int    `json:"toolCallCount"`
			} `json:"dailyActivity"`
			DailyModelTokens []struct {
				Date          string            `json:"date"`
				TokensByModel map[string]int64  `json:"tokensByModel"`
			} `json:"dailyModelTokens"`
		}
		_ = json.Unmarshal(data, &stats)
		for _, d := range stats.DailyActivity {
			agent.Stats.DailyActivity = append(agent.Stats.DailyActivity, DailyActivity{
				Date:          d.Date,
				MessageCount:  d.MessageCount,
				SessionCount:  d.SessionCount,
				ToolCallCount: d.ToolCallCount,
			})
			agent.Stats.TotalMessages += d.MessageCount
			agent.Stats.TotalSessions += d.SessionCount
			agent.Stats.TotalToolCalls += d.ToolCallCount
		}
		agent.Stats.TokensByModel = make(map[string]int64)
		for _, d := range stats.DailyModelTokens {
			for model, tokens := range d.TokensByModel {
				agent.Stats.TokensByModel[model] += tokens
			}
		}
	}

	// Parse history.jsonl for sessions
	historyPath := filepath.Join(dataDir, "history.jsonl")
	if f, err := os.Open(historyPath); err == nil {
		defer f.Close()
		scanner := bufio.NewScanner(f)
		sessionMap := make(map[string]*AgentSession)
		for scanner.Scan() {
			var entry struct {
				Display   string `json:"display"`
				Timestamp int64  `json:"timestamp"`
				Project   string `json:"project"`
				SessionID string `json:"sessionId"`
			}
			_ = json.Unmarshal(scanner.Bytes(), &entry)
			if entry.SessionID == "" {
				continue
			}
			sess, ok := sessionMap[entry.SessionID]
			if !ok {
				sess = &AgentSession{
					ID:       entry.SessionID,
					Project:  filepath.Base(entry.Project),
					RepoPath: entry.Project,
					Status:   "unknown",
				}
				sessionMap[entry.SessionID] = sess
			}
			ts := time.UnixMilli(entry.Timestamp)
			if sess.StartedAt.IsZero() || ts.Before(sess.StartedAt) {
				sess.StartedAt = ts
			}
			if sess.EndedAt == nil || ts.After(*sess.EndedAt) {
				sess.EndedAt = &ts
			}
		}
		for _, sess := range sessionMap {
			agent.Sessions = append(agent.Sessions, *sess)
		}
	}

	// Parse active sessions from sessions/*.json
	sessionsDir := filepath.Join(dataDir, "sessions")
	entries, _ := os.ReadDir(sessionsDir)
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		data, _ := os.ReadFile(filepath.Join(sessionsDir, entry.Name()))
		var sessMeta struct {
			SessionID string `json:"sessionId"`
			CWD       string `json:"cwd"`
			StartedAt int64  `json:"startedAt"`
			Status    string `json:"status"`
			Version   string `json:"version"`
		}
		_ = json.Unmarshal(data, &sessMeta)
		if sessMeta.SessionID != "" {
			// Update existing or add new
			found := false
			for i := range agent.Sessions {
				if agent.Sessions[i].ID == sessMeta.SessionID {
					agent.Sessions[i].Status = sessMeta.Status
					agent.Sessions[i].RepoPath = sessMeta.CWD
					found = true
					break
				}
			}
			if !found {
				agent.Sessions = append(agent.Sessions, AgentSession{
					ID:        sessMeta.SessionID,
					RepoPath:  sessMeta.CWD,
					StartedAt: time.UnixMilli(sessMeta.StartedAt),
					Status:    sessMeta.Status,
				})
			}
		}
		if agent.Version == "" && sessMeta.Version != "" {
			agent.Version = sessMeta.Version
		}
	}

	// Parse projects
	projectsDir := filepath.Join(dataDir, "projects")
	projectEntries, _ := os.ReadDir(projectsDir)
	for _, pe := range projectEntries {
		if !pe.IsDir() {
			continue
		}
		projPath := filepath.Join(projectsDir, pe.Name())
		projectFiles, _ := os.ReadDir(projPath)
		sessionCount := 0
		for _, pf := range projectFiles {
			if strings.HasSuffix(pf.Name(), ".jsonl") {
				sessionCount++
			}
		}
		// Decode path from dir name
		decodedPath := decodeClaudeProjectName(pe.Name())
		agent.Projects = append(agent.Projects, AgentProject{
			Name:     filepath.Base(decodedPath),
			Path:     decodedPath,
			Sessions: sessionCount,
		})
	}

	return agent
}

func readClaudeVersion(dataDir string) string {
	// Try to get version from package or session files
	entries, _ := os.ReadDir(filepath.Join(dataDir, "sessions"))
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		data, _ := os.ReadFile(filepath.Join(dataDir, "sessions", e.Name()))
		var meta struct {
			Version string `json:"version"`
		}
		_ = json.Unmarshal(data, &meta)
		if meta.Version != "" {
			return meta.Version
		}
	}
	return "unknown"
}

func decodeClaudeProjectName(name string) string {
	// Names are like "-home-dany-Desktop-substrate"
	// Decode by replacing - with / and handling leading -
	parts := strings.Split(name, "-")
	var decoded []string
	for _, p := range parts {
		if p == "" {
			continue
		}
		decoded = append(decoded, p)
	}
	return "/" + strings.Join(decoded, "/")
}
