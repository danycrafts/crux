package agentdata

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

// DiscoverGemini parses Gemini CLI's local data.
func DiscoverGemini() *DiscoveredAgent {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil
	}
	dataDir := filepath.Join(home, ".gemini")
	if _, err := os.Stat(dataDir); os.IsNotExist(err) {
		return nil
	}

	agent := &DiscoveredAgent{
		ID:           "gemini-cli",
		Name:         "Gemini CLI",
		Command:      "gemini",
		Provider:     "google",
		DataDir:      dataDir,
		Capabilities: []string{"code_edit", "shell", "mcp", "extensions", "skills", "hooks"},
		Config:       make(map[string]interface{}),
	}

	// Parse settings.json for MCP servers
	settingsPath := filepath.Join(dataDir, "settings.json")
	if data, err := os.ReadFile(settingsPath); err == nil {
		var settings struct {
			MCPServers map[string]struct {
				URL     string            `json:"url"`
				Type    string            `json:"type"`
				Headers map[string]string `json:"headers"`
				Trust   bool              `json:"trust"`
			} `json:"mcpServers"`
			Security struct {
				Auth struct {
					SelectedType string `json:"selectedType"`
				} `json:"auth"`
			} `json:"security"`
			IDE struct {
				Enabled bool `json:"enabled"`
			} `json:"ide"`
		}
		_ = json.Unmarshal(data, &settings)
		agent.Config["settings"] = settings
		for name, srv := range settings.MCPServers {
			transport := srv.Type
			if transport == "" {
				transport = "http"
			}
			agent.MCPServers = append(agent.MCPServers, MCPServer{
				Name:      name,
				Transport: transport,
				URL:       srv.URL,
			})
		}
	}

	// Parse state.json
	statePath := filepath.Join(dataDir, "state.json")
	if data, err := os.ReadFile(statePath); err == nil {
		var state map[string]interface{}
		_ = json.Unmarshal(data, &state)
		agent.Config["state"] = state
	}

	// Parse projects.json
	projectsPath := filepath.Join(dataDir, "projects.json")
	if data, err := os.ReadFile(projectsPath); err == nil {
		var projects struct {
			Projects map[string]string `json:"projects"`
		}
		_ = json.Unmarshal(data, &projects)
		for path, name := range projects.Projects {
			agent.Projects = append(agent.Projects, AgentProject{
				Name: name,
				Path: path,
			})
		}
	}

	// Parse history directories for project roots
	historyDir := filepath.Join(dataDir, "history")
	entries, _ := os.ReadDir(historyDir)
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		projectRootFile := filepath.Join(historyDir, entry.Name(), ".project_root")
		if data, err := os.ReadFile(projectRootFile); err == nil {
			path := strings.TrimSpace(string(data))
			// Check if we already have this project
			found := false
			for i := range agent.Projects {
				if agent.Projects[i].Path == path {
					found = true
					break
				}
			}
			if !found {
				agent.Projects = append(agent.Projects, AgentProject{
					Name: entry.Name(),
					Path: path,
				})
			}
		}
	}

	// Gemini doesn't store local session counts in accessible files
	agent.Stats.TotalSessions = len(agent.Projects)

	return agent
}
