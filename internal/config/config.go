package config

import (
	"os"

	"gopkg.in/yaml.v3"
)

// Config holds the migration configuration.
type Config struct {
	Directory string   `yaml:"directory"`
	Files     []string `yaml:"files"`
}

// Load reads a YAML configuration file from the given path.
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

// Save writes the configuration to a YAML file at the given path.
func Save(path string, cfg *Config) error {
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}
