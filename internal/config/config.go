package config

import (
	"fmt"
	"mailboxzero/internal/protocol"
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Server ServerConfig `yaml:"server"`

	// Protocol selection: "jmap" or "imap"
	Protocol string `yaml:"protocol"`

	// Protocol-specific configurations
	JMAP JMAPConfig `yaml:"jmap"`
	IMAP IMAPConfig `yaml:"imap"`

	DryRun            bool `yaml:"dry_run"`
	DefaultSimilarity int  `yaml:"default_similarity"`
	MockMode          bool `yaml:"mock_mode"`
}

type ServerConfig struct {
	Port int    `yaml:"port"`
	Host string `yaml:"host"`
}

type JMAPConfig struct {
	Endpoint string `yaml:"endpoint"`
	APIToken string `yaml:"api_token"`
}

type IMAPConfig struct {
	Host          string `yaml:"host"`
	Port          int    `yaml:"port"`
	Username      string `yaml:"username"`
	Password      string `yaml:"password"`
	UseTLS        bool   `yaml:"use_tls"`
	ArchiveFolder string `yaml:"archive_folder"`
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

	// Default to JMAP for backward compatibility
	if config.Protocol == "" {
		config.Protocol = "jmap"
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

	// Validate protocol selection
	if c.Protocol != "jmap" && c.Protocol != "imap" {
		return fmt.Errorf("invalid protocol: %s (must be 'jmap' or 'imap')", c.Protocol)
	}

	// In mock mode, protocol credentials are not required
	if !c.MockMode {
		switch c.Protocol {
		case "jmap":
			if err := c.validateJMAP(); err != nil {
				return err
			}
		case "imap":
			if err := c.validateIMAP(); err != nil {
				return err
			}
		}
	}

	if c.DefaultSimilarity < 0 || c.DefaultSimilarity > 100 {
		return fmt.Errorf("default similarity must be between 0 and 100")
	}

	return nil
}

func (c *Config) validateJMAP() error {
	if c.JMAP.Endpoint == "" {
		return fmt.Errorf("JMAP endpoint is required")
	}
	if c.JMAP.APIToken == "" {
		return fmt.Errorf("JMAP API token is required")
	}
	return nil
}

func (c *Config) validateIMAP() error {
	if c.IMAP.Host == "" {
		return fmt.Errorf("IMAP host is required")
	}
	if c.IMAP.Port <= 0 || c.IMAP.Port > 65535 {
		return fmt.Errorf("invalid IMAP port: %d", c.IMAP.Port)
	}
	if c.IMAP.Username == "" {
		return fmt.Errorf("IMAP username is required")
	}
	if c.IMAP.Password == "" {
		return fmt.Errorf("IMAP password is required")
	}
	// ArchiveFolder is optional, will default to "Archive" if not set
	if c.IMAP.ArchiveFolder == "" {
		c.IMAP.ArchiveFolder = "Archive"
	}
	return nil
}

func (c *Config) GetServerAddr() string {
	return fmt.Sprintf("%s:%d", c.Server.Host, c.Server.Port)
}

func (c *Config) GetProtocolType() protocol.ProtocolType {
	return protocol.ProtocolType(c.Protocol)
}
