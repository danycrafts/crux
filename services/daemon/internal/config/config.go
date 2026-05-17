package config

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"gopkg.in/yaml.v3"
)

// Config is the daemon configuration.
type Config struct {
	APIPort       int            `yaml:"api_port"`
	DataDir       string         `yaml:"data_dir"`
	AgentRegistry *AgentRegistry `yaml:"agents,omitempty"`
	MCP           *MCPConfig     `yaml:"mcp,omitempty"`
	Policies      *Policies      `yaml:"policies,omitempty"`
}

// AgentRegistry holds known agents.
type AgentRegistry struct {
	Agents map[string]AgentDef `yaml:"agents"`
}

// AgentDef defines a single agent.
type AgentDef struct {
	Type         string   `yaml:"type"`
	Command      string   `yaml:"command"`
	Path         string   `yaml:"path,omitempty"`
	Capabilities []string `yaml:"capabilities,omitempty"`
	Owner        string   `yaml:"owner,omitempty"`
	Provider     string   `yaml:"provider,omitempty"`
	SupportsMCP  bool     `yaml:"supports_mcp,omitempty"`
}

// MCPConfig holds gateway configuration.
type MCPConfig struct {
	Port    int                  `yaml:"port"`
	Servers map[string]MCPServer `yaml:"servers,omitempty"`
}

// MCPServer defines an MCP server target.
type MCPServer struct {
	Transport string   `yaml:"transport"`
	Command   string   `yaml:"command,omitempty"`
	Args      []string `yaml:"args,omitempty"`
	URL       string   `yaml:"url,omitempty"`
}

// Policies holds tool policies.
type Policies struct {
	Deny            []string `yaml:"deny,omitempty"`
	RequireApproval []string `yaml:"require_approval,omitempty"`
	Allow           []string `yaml:"allow,omitempty"`
}

// Default returns the default configuration.
func Default() *Config {
	dir := defaultDataDir()
	return &Config{
		APIPort: 8080,
		DataDir: dir,
		AgentRegistry: &AgentRegistry{
			Agents: make(map[string]AgentDef),
		},
		MCP: &MCPConfig{
			Port:    3000,
			Servers: make(map[string]MCPServer),
		},
		Policies: &Policies{},
	}
}

// Load reads configuration from path or creates defaults.
func Load(path string) (*Config, error) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		cfg := Default()
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return nil, err
		}
		if err := cfg.Save(path); err != nil {
			return nil, err
		}
		return cfg, nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	if cfg.DataDir == "" {
		cfg.DataDir = defaultDataDir()
	}
	if cfg.APIPort == 0 {
		cfg.APIPort = 8080
	}
	if cfg.AgentRegistry == nil {
		cfg.AgentRegistry = &AgentRegistry{Agents: make(map[string]AgentDef)}
	}
	if cfg.MCP == nil {
		cfg.MCP = &MCPConfig{Port: 3000, Servers: make(map[string]MCPServer)}
	}
	if cfg.Policies == nil {
		cfg.Policies = &Policies{}
	}
	return &cfg, nil
}

// Save writes configuration to disk.
func (c *Config) Save(path string) error {
	data, err := yaml.Marshal(c)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o600)
}

func defaultDataDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		home = "."
	}
	if runtime.GOOS == "windows" {
		return filepath.Join(home, "AppData", "Local", "Crux")
	}
	return filepath.Join(home, ".crux")
}

// ConfigPath returns the default configuration file path.
func ConfigPath() string {
	return filepath.Join(defaultDataDir(), "crux.yaml")
}

// PIDPath returns the daemon PID file path.
func PIDPath() string {
	return filepath.Join(defaultDataDir(), "cruxd.pid")
}

// DBPath returns the SQLite database path.
func DBPath() string {
	return filepath.Join(defaultDataDir(), "crux.db")
}

// EnsureDirs creates the data directory structure.
func (c *Config) EnsureDirs() error {
	for _, sub := range []string{"", "transcripts", "gateway"} {
		p := filepath.Join(c.DataDir, sub)
		if err := os.MkdirAll(p, 0o755); err != nil {
			return fmt.Errorf("mkdir %s: %w", p, err)
		}
	}
	return nil
}
