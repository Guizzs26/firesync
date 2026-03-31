package config

import (
	"fmt"
	"os"
	"time"

	"go.yaml.in/yaml/v2"
)

type Config struct {
	NodeID       string        `yaml:"node_id"`
	PollInterval time.Duration `yaml:"poll_interval"`
	Source       DBConfig      `yaml:"source"`
	Destination  DBConfig      `yaml:"destination"`
	Tables       []string      `yaml:"tables"`
}

type DBConfig struct {
	Type     string `yaml:"type"`
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	Database string `yaml:"database"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
	URL      string `yaml:"url"`
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read file: %v", err)
	}

	expandedData := os.ExpandEnv(string(data))
	var cfg Config
	if err := yaml.Unmarshal([]byte(expandedData), &cfg); err != nil {
		return nil, fmt.Errorf("unmarshal yaml: %v", err)
	}

	if cfg.PollInterval == 0 {
		cfg.PollInterval = 5 * time.Second
	}

	return &cfg, nil
}
