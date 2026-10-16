package main

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/taskinen/mailboxzero/internal/config"
)

// The example that -example-config prints must become a working config once
// the user adds a token, and must start in dry run mode.
func TestExampleConfig(t *testing.T) {
	example := string(exampleConfig)
	if !strings.Contains(example, `api_token: ""`) {
		t.Fatalf("example config has no empty api_token to fill in:\n%s", example)
	}

	path := filepath.Join(t.TempDir(), "config.yaml")
	filled := strings.Replace(example, `api_token: ""`, `api_token: "test-token"`, 1)
	if err := os.WriteFile(path, []byte(filled), 0o600); err != nil {
		t.Fatal(err)
	}

	cfg, err := config.Load(path)
	if err != nil {
		t.Fatalf("config.Load(example with a token) failed: %v", err)
	}
	if !cfg.DryRun {
		t.Error("example config has dry_run off, want on")
	}
}

func TestDefaultConfigPath(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	tests := []struct {
		name          string
		xdgConfigHome string
		want          string
	}{
		{name: "XDG_CONFIG_HOME not set", want: filepath.Join(home, ".config", "mailboxzero", "config.yaml")},
		{name: "XDG_CONFIG_HOME set", xdgConfigHome: "/xdg/config", want: filepath.Join("/xdg/config", "mailboxzero", "config.yaml")},
		// The XDG Base Directory spec says to ignore a relative path.
		{name: "XDG_CONFIG_HOME relative", xdgConfigHome: "relative/config", want: filepath.Join(home, ".config", "mailboxzero", "config.yaml")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("XDG_CONFIG_HOME", tt.xdgConfigHome)
			if got := defaultConfigPath(); got != tt.want {
				t.Errorf("defaultConfigPath() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestLoadConfig_MissingFile(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "mailboxzero")
	path := filepath.Join(dir, "config.yaml")

	_, err := loadConfig(path)
	if err == nil {
		t.Fatal("loadConfig(missing file) succeeded, want an error")
	}
	if !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("error %q does not wrap fs.ErrNotExist", err)
	}
	// The default folder may not exist yet either.
	for _, want := range []string{path, "mkdir -p " + dir, "mailboxzero -example-config > " + path} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error %q does not mention %q", err, want)
		}
	}
}

func TestLoadConfig_InvalidFileIsNotMissing(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte("server: [not yaml"), 0o600); err != nil {
		t.Fatal(err)
	}

	_, err := loadConfig(path)
	if err == nil {
		t.Fatal("loadConfig(invalid YAML) succeeded, want an error")
	}
	if strings.Contains(err.Error(), "-example-config") {
		t.Errorf("error %q suggests creating a file that exists", err)
	}
}
