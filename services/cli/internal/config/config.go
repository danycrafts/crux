package config

import (
	"os"
	"path/filepath"
	"runtime"

	"github.com/danycrafts/crux/pkg/logger"
	"gopkg.in/yaml.v3"
)

// Config holds CLI configuration.
type Config struct {
	APIURL        string        `yaml:"api_url"`
	DefaultAgent  string        `yaml:"default_agent"`
	DefaultRepo   string        `yaml:"default_repo"`
	OutputFormat  string        `yaml:"output_format"`
	Logging       logger.Config `yaml:"logging"`
}

// Default returns default CLI config.
func Default() *Config {
	return &Config{
		APIURL:       "http://localhost:8080",
		DefaultAgent: "claude-code",
		DefaultRepo:  ".",
		OutputFormat: "table",
		Logging:      logger.DefaultConfig(),
	}
}

// Load reads or creates config.
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
	if cfg.APIURL == "" {
		cfg.APIURL = "http://localhost:8080"
	}
	if cfg.OutputFormat == "" {
		cfg.OutputFormat = "table"
	}
	return &cfg, nil
}

// Save writes config.
func (c *Config) Save(path string) error {
	data, err := yaml.Marshal(c)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o600)
}

// Path returns default config path.
func Path() string {
	home, _ := os.UserHomeDir()
	if runtime.GOOS == "windows" {
		return filepath.Join(home, "AppData", "Local", "Crux", "cli.yaml")
	}
	return filepath.Join(home, ".crux", "cli.yaml")
}
