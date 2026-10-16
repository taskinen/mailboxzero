package main

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"strings"
	"testing"

	"github.com/taskinen/mailboxzero/internal/config"
)

func TestVersionFrom(t *testing.T) {
	platform := runtime.GOOS + "/" + runtime.GOARCH

	tests := []struct {
		name string
		info *debug.BuildInfo
		ok   bool
		want string
	}{
		{
			name: "release",
			info: &debug.BuildInfo{GoVersion: "go1.26.9", Main: debug.Module{Version: "v1.0.0"}},
			ok:   true,
			want: "mailboxzero v1.0.0 (go1.26.9, " + platform + ")",
		},
		{
			name: "build from a checkout between releases",
			info: &debug.BuildInfo{GoVersion: "go1.26.9", Main: debug.Module{Version: "v1.0.1-0.20261016120000-abcdef123456"}},
			ok:   true,
			want: "mailboxzero v1.0.1-0.20261016120000-abcdef123456 (go1.26.9, " + platform + ")",
		},
		{
			name: "build with uncommitted changes",
			info: &debug.BuildInfo{GoVersion: "go1.26.9", Main: debug.Module{Version: "v1.0.0+dirty"}},
			ok:   true,
			want: "mailboxzero v1.0.0+dirty (go1.26.9, " + platform + ")",
		},
		{
			name: "go run",
			info: &debug.BuildInfo{GoVersion: "go1.26.9", Main: debug.Module{Version: "(devel)"}},
			ok:   true,
			want: "mailboxzero (devel) (go1.26.9, " + platform + ")",
		},
		{
			name: "no module version",
			info: &debug.BuildInfo{GoVersion: "go1.26.9"},
			ok:   true,
			want: "mailboxzero (devel) (go1.26.9, " + platform + ")",
		},
		{
			name: "no build info",
			ok:   false,
			want: "mailboxzero (devel) (" + runtime.Version() + ", " + platform + ")",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := versionFrom(tt.info, tt.ok); got != tt.want {
				t.Errorf("versionFrom() = %q, want %q", got, tt.want)
			}
		})
	}
}

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

func TestLoadConfig_MissingFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")

	_, err := loadConfig(path)
	if err == nil {
		t.Fatal("loadConfig(missing file) succeeded, want an error")
	}
	if !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("error %q does not wrap fs.ErrNotExist", err)
	}
	for _, want := range []string{path, "mailboxzero -example-config > " + path} {
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
