package agentdata

import (
	"time"
)

// DiscoveredAgent holds everything we learned about an installed agent.
type DiscoveredAgent struct {
	ID           string            `json:"id"`
	Name         string            `json:"name"`
	Version      string            `json:"version"`
	Command      string            `json:"command"`
	Provider     string            `json:"provider"`
	DataDir      string            `json:"data_dir"`
	Capabilities []string          `json:"capabilities"`
	Config       map[string]interface{} `json:"config"`
	Sessions     []AgentSession    `json:"sessions"`
	Stats        AgentStats        `json:"stats"`
	MCPServers   []MCPServer       `json:"mcp_servers"`
	Projects     []AgentProject     `json:"projects"`
}

// AgentSession is a normalized session from any agent.
type AgentSession struct {
	ID        string    `json:"id"`
	Project   string    `json:"project"`
	RepoPath  string    `json:"repo_path"`
	StartedAt time.Time `json:"started_at"`
	EndedAt   *time.Time `json:"ended_at,omitempty"`
	Status    string    `json:"status"`
	Summary   string    `json:"summary"`
}

// AgentStats holds usage stats.
type AgentStats struct {
	TotalSessions  int            `json:"total_sessions"`
	TotalMessages  int            `json:"total_messages"`
	TotalToolCalls int            `json:"total_tool_calls"`
	TokensByModel  map[string]int64 `json:"tokens_by_model"`
	DailyActivity  []DailyActivity `json:"daily_activity"`
}

// DailyActivity is per-day stats.
type DailyActivity struct {
	Date         string `json:"date"`
	MessageCount int    `json:"message_count"`
	SessionCount int    `json:"session_count"`
	ToolCallCount int   `json:"tool_call_count"`
}

// MCPServer is a discovered MCP server config.
type MCPServer struct {
	Name      string   `json:"name"`
	Transport string   `json:"transport"`
	Command   string   `json:"command,omitempty"`
	Args      []string `json:"args,omitempty"`
	URL       string   `json:"url,omitempty"`
}

// AgentProject is a project the agent knows about.
type AgentProject struct {
	Name     string `json:"name"`
	Path     string `json:"path"`
	Sessions int    `json:"sessions"`
}

// DiscoverAll scans all known agents and returns their data.
func DiscoverAll() []DiscoveredAgent {
	var out []DiscoveredAgent
	if a := DiscoverClaude(); a != nil {
		out = append(out, *a)
	}
	if a := DiscoverCodex(); a != nil {
		out = append(out, *a)
	}
	if a := DiscoverGemini(); a != nil {
		out = append(out, *a)
	}
	return out
}
