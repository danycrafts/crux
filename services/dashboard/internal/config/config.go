package config

import (
	"os"
	"path/filepath"
	"runtime"

	"github.com/danycrafts/crux/pkg/logger"
	"gopkg.in/yaml.v3"
)

// Config holds dashboard configuration.
type Config struct {
	Port            int           `yaml:"port"`
	APIURL          string        `yaml:"api_url"`
	RefreshInterval int           `yaml:"refresh_interval"`
	Logging         logger.Config `yaml:"logging"`
}

// Default returns default dashboard config.
func Default() *Config {
	return &Config{
		Port:            3001,
		APIURL:          "http://localhost:8080",
		RefreshInterval: 10,
		Logging:         logger.DefaultConfig(),
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
	if cfg.Port == 0 {
		cfg.Port = 3001
	}
	if cfg.APIURL == "" {
		cfg.APIURL = "http://localhost:8080"
	}
	if cfg.RefreshInterval == 0 {
		cfg.RefreshInterval = 10
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
		return filepath.Join(home, "AppData", "Local", "Crux", "dashboard.yaml")
	}
	return filepath.Join(home, ".crux", "dashboard.yaml")
}
