package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoad(t *testing.T) {
	tests := []struct {
		name        string
		configYAML  string
		wantErr     bool
		errContains string
	}{
		{
			name: "valid config",
			configYAML: `
server:
  port: 8080
  host: localhost
jmap:
  endpoint: https://api.fastmail.com/jmap/session
  api_token: test-token
dry_run: true
default_similarity: 75
`,
			wantErr: false,
		},
		{
			name: "valid config in mock mode",
			configYAML: `
server:
  port: 8080
  host: localhost
jmap:
  endpoint: ""
  api_token: ""
dry_run: true
default_similarity: 75
mock_mode: true
`,
			wantErr: false,
		},
		{
			name: "missing jmap endpoint",
			configYAML: `
server:
  port: 8080
  host: localhost
jmap:
  api_token: test-token
dry_run: true
default_similarity: 75
`,
			wantErr:     true,
			errContains: "JMAP endpoint is required",
		},
		{
			name: "missing jmap api token",
			configYAML: `
server:
  port: 8080
  host: localhost
jmap:
  endpoint: https://api.fastmail.com/jmap/session
dry_run: true
default_similarity: 75
`,
			wantErr:     true,
			errContains: "JMAP API token is required",
		},
		{
			name: "invalid port - negative",
			configYAML: `
server:
  port: -1
  host: localhost
jmap:
  endpoint: https://api.fastmail.com/jmap/session
  api_token: test-token
dry_run: true
default_similarity: 75
`,
			wantErr:     true,
			errContains: "invalid server port",
		},
		{
			name: "invalid port - too high",
			configYAML: `
server:
  port: 99999
  host: localhost
jmap:
  endpoint: https://api.fastmail.com/jmap/session
  api_token: test-token
dry_run: true
default_similarity: 75
`,
			wantErr:     true,
			errContains: "invalid server port",
		},
		{
			name: "invalid similarity - negative",
			configYAML: `
server:
  port: 8080
  host: localhost
jmap:
  endpoint: https://api.fastmail.com/jmap/session
  api_token: test-token
dry_run: true
default_similarity: -10
`,
			wantErr:     true,
			errContains: "default similarity must be between 0 and 100",
		},
		{
			name: "invalid similarity - over 100",
			configYAML: `
server:
  port: 8080
  host: localhost
jmap:
  endpoint: https://api.fastmail.com/jmap/session
  api_token: test-token
dry_run: true
default_similarity: 150
`,
			wantErr:     true,
			errContains: "default similarity must be between 0 and 100",
		},
		{
			name: "invalid YAML",
			configYAML: `
server:
  port: 8080
  host: localhost
  invalid yaml here: [
`,
			wantErr:     true,
			errContains: "failed to parse config file",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary config file
			tmpDir := t.TempDir()
			configPath := filepath.Join(tmpDir, "config.yaml")

			err := os.WriteFile(configPath, []byte(tt.configYAML), 0644)
			if err != nil {
				t.Fatalf("Failed to write test config: %v", err)
			}

			// Test Load function
			cfg, err := Load(configPath)

			if tt.wantErr {
				if err == nil {
					t.Errorf("Load() expected error but got none")
				} else if tt.errContains != "" && !contains(err.Error(), tt.errContains) {
					t.Errorf("Load() error = %v, want error containing %q", err, tt.errContains)
				}
			} else {
				if err != nil {
					t.Errorf("Load() unexpected error = %v", err)
				}
				if cfg == nil {
					t.Errorf("Load() returned nil config")
				}
			}
		})
	}
}

func TestLoadNonexistentFile(t *testing.T) {
	_, err := Load("/nonexistent/path/config.yaml")
	if err == nil {
		t.Error("Load() expected error for nonexistent file but got none")
	}
	if !contains(err.Error(), "failed to read config file") {
		t.Errorf("Load() error = %v, want error containing 'failed to read config file'", err)
	}
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name        string
		config      Config
		wantErr     bool
		errContains string
	}{
		{
			name: "valid config",
			config: Config{
				Server: struct {
					Port int    `yaml:"port"`
					Host string `yaml:"host"`
				}{
					Port: 8080,
					Host: "localhost",
				},
				JMAP: struct {
					Endpoint string `yaml:"endpoint"`
					APIToken string `yaml:"api_token"`
				}{
					Endpoint: "https://api.fastmail.com/jmap/session",
					APIToken: "test-token",
				},
				DryRun:            true,
				DefaultSimilarity: 75,
				MockMode:          false,
			},
			wantErr: false,
		},
		{
			name: "valid config with mock mode",
			config: Config{
				Server: struct {
					Port int    `yaml:"port"`
					Host string `yaml:"host"`
				}{
					Port: 8080,
					Host: "localhost",
				},
				JMAP: struct {
					Endpoint string `yaml:"endpoint"`
					APIToken string `yaml:"api_token"`
				}{
					Endpoint: "",
					APIToken: "",
				},
				DryRun:            true,
				DefaultSimilarity: 75,
				MockMode:          true,
			},
			wantErr: false,
		},
		{
			name: "missing jmap endpoint without mock mode",
			config: Config{
				Server: struct {
					Port int    `yaml:"port"`
					Host string `yaml:"host"`
				}{
					Port: 8080,
					Host: "localhost",
				},
				JMAP: struct {
					Endpoint string `yaml:"endpoint"`
					APIToken string `yaml:"api_token"`
				}{
					Endpoint: "",
					APIToken: "test-token",
				},
				DryRun:            true,
				DefaultSimilarity: 75,
				MockMode:          false,
			},
			wantErr:     true,
			errContains: "JMAP endpoint is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.validate()

			if tt.wantErr {
				if err == nil {
					t.Errorf("validate() expected error but got none")
				} else if tt.errContains != "" && !contains(err.Error(), tt.errContains) {
					t.Errorf("validate() error = %v, want error containing %q", err, tt.errContains)
				}
			} else {
				if err != nil {
					t.Errorf("validate() unexpected error = %v", err)
				}
			}
		})
	}
}

func TestGetServerAddr(t *testing.T) {
	tests := []struct {
		name string
		host string
		port int
		want string
	}{
		{
			name: "localhost with port 8080",
			host: "localhost",
			port: 8080,
			want: "localhost:8080",
		},
		{
			name: "0.0.0.0 with port 3000",
			host: "0.0.0.0",
			port: 3000,
			want: "0.0.0.0:3000",
		},
		{
			name: "empty host with port 8080",
			host: "",
			port: 8080,
			want: ":8080",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{
				Server: struct {
					Port int    `yaml:"port"`
					Host string `yaml:"host"`
				}{
					Port: tt.port,
					Host: tt.host,
				},
			}

			got := cfg.GetServerAddr()
			if got != tt.want {
				t.Errorf("GetServerAddr() = %v, want %v", got, tt.want)
			}
		})
	}
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && stringContains(s, substr)))
}

func stringContains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
