package config

import (
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"

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
	MockMode          bool `yaml:"mock_mode"`
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

	// The web UI has no login, so it must only be reachable from this
	// machine. An empty host or 0.0.0.0 would listen on every interface.
	if !IsLoopbackHost(c.Server.Host) {
		return fmt.Errorf("server host %q is not allowed: use localhost, 127.0.0.1 or ::1 "+
			"(Mailbox Zero has no login, so it only listens on this machine; "+
			"use an SSH tunnel for remote access)", c.Server.Host)
	}

	// In mock mode, JMAP credentials are not required
	if !c.MockMode {
		if c.JMAP.Endpoint == "" {
			return fmt.Errorf("JMAP endpoint is required")
		}

		if c.JMAP.APIToken == "" {
			return fmt.Errorf("JMAP API token is required")
		}
	}

	if c.DefaultSimilarity < 0 || c.DefaultSimilarity > 100 {
		return fmt.Errorf("default similarity must be between 0 and 100")
	}

	return nil
}

func (c *Config) GetServerAddr() string {
	return net.JoinHostPort(c.Server.Host, strconv.Itoa(c.Server.Port))
}

// IsLoopbackHost reports whether host names the local machine only:
// "localhost" or a loopback IP address (127.0.0.0/8, ::1). host is a bare
// host name or IP, without a port or IPv6 brackets.
func IsLoopbackHost(host string) bool {
	if strings.EqualFold(host, "localhost") {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}
