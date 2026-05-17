package gateway

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/danycrafts/crux/services/daemon/internal/config"
	"gopkg.in/yaml.v3"
)

// GenerateConfig writes an agentgateway-compatible MCP config.
func GenerateConfig(cfg *config.Config, outDir string) (string, error) {
	servers := make(map[string]ServerBlock)
	for name, s := range cfg.MCP.Servers {
		servers[name] = ServerBlock{
			Transport: s.Transport,
			Command:   s.Command,
			Args:      s.Args,
			URL:       s.URL,
		}
	}

	gw := GatewayConfig{
		Version: "1.0",
		MCP: MCPBlock{
			Port:    cfg.MCP.Port,
			Servers: servers,
		},
		Policies: PolicyBlock{
			Deny:            cfg.Policies.Deny,
			RequireApproval: cfg.Policies.RequireApproval,
			Allow:           cfg.Policies.Allow,
		},
	}

	data, err := yaml.Marshal(gw)
	if err != nil {
		return "", err
	}

	path := filepath.Join(outDir, "agentgateway.yaml")
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return "", err
	}
	return path, nil
}

// GatewayConfig is the top-level agentgateway config shape.
type GatewayConfig struct {
	Version  string     `yaml:"version"`
	MCP      MCPBlock   `yaml:"mcp"`
	Policies PolicyBlock `yaml:"policies,omitempty"`
}

// MCPBlock holds MCP gateway settings.
type MCPBlock struct {
	Port    int                   `yaml:"port"`
	Servers map[string]ServerBlock `yaml:"servers"`
}

// ServerBlock describes one MCP server target.
type ServerBlock struct {
	Transport string   `yaml:"transport"`
	Command   string   `yaml:"command,omitempty"`
	Args      []string `yaml:"args,omitempty"`
	URL       string   `yaml:"url,omitempty"`
}

// PolicyBlock holds simple policy lists.
type PolicyBlock struct {
	Deny            []string `yaml:"deny,omitempty"`
	RequireApproval []string `yaml:"require_approval,omitempty"`
	Allow           []string `yaml:"allow,omitempty"`
}

// ValidatePolicy checks if a tool is allowed under current policies.
func ValidatePolicy(tool string, p *config.Policies) (allowed bool, needsApproval bool, reason string) {
	for _, d := range p.Deny {
		if d == tool || d == "*" {
			return false, false, fmt.Sprintf("tool %q is denied by policy", tool)
		}
	}
	for _, a := range p.RequireApproval {
		if a == tool || a == "*" {
			return true, true, fmt.Sprintf("tool %q requires approval", tool)
		}
	}
	if len(p.Allow) == 0 {
		return true, false, ""
	}
	for _, a := range p.Allow {
		if a == tool || a == "*" {
			return true, false, ""
		}
	}
	return false, false, fmt.Sprintf("tool %q is not in allow list", tool)
}
