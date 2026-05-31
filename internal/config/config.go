package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	BotToken         string         `yaml:"bot_token"`
	ChatID           int64          `yaml:"chat_id"`
	WatchDir         string         `yaml:"watch_dir"`
	MaxFileSizeBytes int64          `yaml:"max_file_size_bytes"`
	Mappings         map[string]int `yaml:"mappings"`
}

func Load(path string) (*Config, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open config: %w", err)
	}
	defer f.Close()

	var cfg Config
	if err := yaml.NewDecoder(f).Decode(&cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}

	if cfg.BotToken == "" {
		return nil, fmt.Errorf("bot_token is required")
	}
	if cfg.ChatID == 0 {
		return nil, fmt.Errorf("chat_id is required")
	}
	if cfg.WatchDir == "" {
		return nil, fmt.Errorf("watch_dir is required")
	}
	if cfg.MaxFileSizeBytes == 0 {
		cfg.MaxFileSizeBytes = 2147483648 // 2GB
	}

	return &cfg, nil
}
