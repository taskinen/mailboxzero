package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Server struct {
		Port int    `yaml:"port"`
		Host string `yaml:"host"`
	} `yaml:"server"`
	JMAP struct {
		Endpoint string `yaml:"endpoint"`
		APIToken string `yaml:"api_token"`
	} `yaml:"jmap"`
	DryRun            bool `yaml:"dry_run"`
	DefaultSimilarity int  `yaml:"default_similarity"`
}

func Load(configPath string) (*Config, error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	if err := config.validate(); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	return &config, nil
}

func (c *Config) validate() error {
	if c.Server.Port <= 0 || c.Server.Port > 65535 {
		return fmt.Errorf("invalid server port: %d", c.Server.Port)
	}

	if c.JMAP.Endpoint == "" {
		return fmt.Errorf("JMAP endpoint is required")
	}

	if c.JMAP.APIToken == "" {
		return fmt.Errorf("JMAP API token is required")
	}

	if c.DefaultSimilarity < 0 || c.DefaultSimilarity > 100 {
		return fmt.Errorf("default similarity must be between 0 and 100")
	}

	return nil
}

func (c *Config) GetServerAddr() string {
	return fmt.Sprintf("%s:%d", c.Server.Host, c.Server.Port)
}
