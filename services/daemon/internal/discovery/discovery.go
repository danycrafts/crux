package discovery

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
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

// Discover searches PATH for supported agents.
func Discover(ctx context.Context) []KnownAgent {
	candidates := []struct {
		name         string
		commands     []string
		provider     string
		capabilities []string
		supportsMCP  bool
	}{
		{"Claude Code", []string{"claude"}, "anthropic", []string{"code_edit", "shell", "mcp", "repo_search"}, true},
		{"Codex CLI", []string{"codex"}, "openai", []string{"code_edit", "shell", "mcp"}, true},
		{"Gemini CLI", []string{"gemini"}, "google", []string{"code_edit", "shell"}, false},
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
	return hints
}

// MCPServerHint is a discovered MCP server configuration.
type MCPServerHint struct {
	Name      string   `json:"name"`
	Transport string   `json:"transport"`
	Command   string   `json:"command"`
	Args      []string `json:"args"`
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
