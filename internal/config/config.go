package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type Config struct {
	Token  string `json:"token"`
	Cookie string `json:"cookie"`
}

func configDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".slack-cli")
}

func configPath() string {
	return filepath.Join(configDir(), "config.json")
}

func Load() (*Config, error) {
	data, err := os.ReadFile(configPath())
	if err != nil {
		return nil, fmt.Errorf("no config found — run 'slack-cli auth setup' first")
	}
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("invalid config file: %w", err)
	}
	return &cfg, nil
}

func Save(cfg *Config) error {
	if err := os.MkdirAll(configDir(), 0700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(configPath(), data, 0600)
}
