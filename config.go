package migrator

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Config represents the configuration for migrations
type Config struct {
	Directory string   `yaml:"directory"`
	Files     []string `yaml:"files"`
}

// LoadConfig reads and parses the configuration file
func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &Config{Directory: ".", Files: []string{}}, nil
		}
		return nil, err
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("invalid config format: %w", err)
	}

	return &cfg, nil
}

// SaveConfig writes the configuration to a file
func SaveConfig(path string, cfg *Config) error {
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}
