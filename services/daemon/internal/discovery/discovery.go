package discovery

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/danycrafts/crux/services/daemon/internal/agentdata"
)

// KnownAgent represents a discovered CLI agent.
type KnownAgent struct {
	Name         string   `json:"name"`
	Command      string   `json:"command"`
	Path         string   `json:"path"`
	Provider     string   `json:"provider"`
	Capabilities []string `json:"capabilities"`
	SupportsMCP  bool     `json:"supports_mcp"`
}

// Discover searches PATH for supported agents and parses their local data.
func Discover(ctx context.Context) []KnownAgent {
	candidates := []struct {
		name         string
		commands     []string
		provider     string
		capabilities []string
		supportsMCP  bool
	}{
		{"Claude Code", []string{"claude"}, "anthropic", []string{"code_edit", "shell", "mcp", "repo_search", "skills", "plugins"}, true},
		{"Codex CLI", []string{"codex"}, "openai", []string{"code_edit", "shell", "mcp", "plugins", "sandbox", "skills"}, true},
		{"Gemini CLI", []string{"gemini"}, "google", []string{"code_edit", "shell", "mcp", "extensions", "skills", "hooks"}, false},
		{"OpenCode", []string{"opencode"}, "community", []string{"code_edit", "shell", "mcp"}, true},
		{"Aider", []string{"aider"}, "community", []string{"code_edit", "shell", "repo_search"}, false},
		{"Continue", []string{"continue"}, "community", []string{"code_edit"}, false},
	}

	var found []KnownAgent
	for _, c := range candidates {
		for _, cmd := range c.commands {
			path, err := exec.LookPath(cmd)
			if err != nil {
				continue
			}
			found = append(found, KnownAgent{
				Name:         c.name,
				Command:      cmd,
				Path:         path,
				Provider:     c.provider,
				Capabilities: c.capabilities,
				SupportsMCP:  c.supportsMCP,
			})
			break
		}
	}
	return found
}

// DiscoverAllAgentData parses local data from all installed agents.
func DiscoverAllAgentData() []agentdata.DiscoveredAgent {
	return agentdata.DiscoverAll()
}

// DiscoverMCP attempts to find common MCP server configs or executables.
func DiscoverMCP(ctx context.Context) []MCPServerHint {
	var hints []MCPServerHint
	// Common npx-based servers
	servers := []struct {
		name string
		pkg  string
	}{
		{"filesystem", "@modelcontextprotocol/server-filesystem"},
		{"github", "@modelcontextprotocol/server-github"},
		{"postgres", "@modelcontextprotocol/server-postgres"},
		{"sqlite", "@modelcontextprotocol/server-sqlite"},
	}
	for _, s := range servers {
		hints = append(hints, MCPServerHint{
			Name:      s.name,
			Transport: "stdio",
			Command:   "npx",
			Args:      []string{"-y", s.pkg},
		})
	}

	// Also discover from agent configs
	for _, agent := range agentdata.DiscoverAll() {
		for _, mcp := range agent.MCPServers {
			if mcp.Command != "" || mcp.URL != "" {
				hints = append(hints, MCPServerHint{
					Name:      mcp.Name,
					Transport: mcp.Transport,
					Command:   mcp.Command,
					Args:      mcp.Args,
					URL:       mcp.URL,
				})
			}
		}
	}

	return hints
}

// MCPServerHint is a discovered MCP server configuration.
type MCPServerHint struct {
	Name      string   `json:"name"`
	Transport string   `json:"transport"`
	Command   string   `json:"command"`
	Args      []string `json:"args"`
	URL       string   `json:"url"`
}

// AgentCommand resolves the full command path for an agent ID.
func AgentCommand(agentID string) (string, []string, error) {
	m := map[string][]string{
		"claude-code": {"claude"},
		"codex":       {"codex"},
		"gemini-cli":  {"gemini"},
		"opencode":    {"opencode"},
		"aider":       {"aider"},
	}
	parts, ok := m[agentID]
	if !ok {
		return "", nil, fmt.Errorf("unknown agent %q", agentID)
	}
	path, err := exec.LookPath(parts[0])
	if err != nil {
		return "", nil, fmt.Errorf("agent %q not found in PATH: %w", agentID, err)
	}
	return path, parts[1:], nil
}

// WorkingDir returns the working directory for a repo flag.
func WorkingDir(repo string) string {
	if repo == "" || repo == "." {
		wd, _ := exec.Command("pwd").Output()
		return strings.TrimSpace(string(wd))
	}
	abs, _ := filepath.Abs(repo)
	if abs == "" {
		return repo
	}
	return abs
}

// FindProjectLevelConfigs scans for agent configs in project directories.
func FindProjectLevelConfigs(root string) map[string][]string {
	result := make(map[string][]string)
	_ = filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil || !info.IsDir() {
			return nil
		}
		// Check for agent-specific dirs
		for _, dir := range []string{".claude", ".codex", ".gemini", ".aider"} {
			if _, err := os.Stat(filepath.Join(path, dir)); err == nil {
				agent := strings.TrimPrefix(dir, ".")
				result[agent] = append(result[agent], path)
			}
		}
		return nil
	})
	return result
}
