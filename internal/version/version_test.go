package version

import (
	"runtime"
	"runtime/debug"
	"testing"
)

func checkout(revision, modified string) []debug.BuildSetting {
	return []debug.BuildSetting{
		{Key: "vcs", Value: "git"},
		{Key: "vcs.revision", Value: revision},
		{Key: "vcs.modified", Value: modified},
	}
}

const revision = "59b1469d2f82a1b2c3d4e5f60718293a4b5c6d7e"

func TestDescribe(t *testing.T) {
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
			if got := describe(tt.info, tt.ok); got != tt.want {
				t.Errorf("describe() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestShort(t *testing.T) {
	tests := []struct {
		name string
		info *debug.BuildInfo
		ok   bool
		want string
	}{
		{
			name: "release from go install",
			info: &debug.BuildInfo{Main: debug.Module{Version: "v1.0.0"}},
			ok:   true,
			want: "v1.0.0",
		},
		{
			name: "release built at its tag",
			info: &debug.BuildInfo{Main: debug.Module{Version: "v1.2.0"}, Settings: checkout(revision, "false")},
			ok:   true,
			want: "v1.2.0",
		},
		{
			name: "pre-release",
			info: &debug.BuildInfo{Main: debug.Module{Version: "v1.1.0-rc.1"}},
			ok:   true,
			want: "v1.1.0-rc.1",
		},
		{
			name: "build from a checkout between releases",
			info: &debug.BuildInfo{
				Main:     debug.Module{Version: "v1.0.1-0.20261016172523-59b1469d2f82"},
				Settings: checkout(revision, "false"),
			},
			ok:   true,
			want: "59b1469",
		},
		{
			name: "build with uncommitted changes",
			info: &debug.BuildInfo{
				Main:     debug.Module{Version: "v1.0.1-0.20261016172523-59b1469d2f82+dirty"},
				Settings: checkout(revision, "true"),
			},
			ok:   true,
			want: "59b1469-dirty",
		},
		{
			name: "build at a tag with uncommitted changes",
			info: &debug.BuildInfo{Main: debug.Module{Version: "v1.0.0+dirty"}, Settings: checkout(revision, "true")},
			ok:   true,
			want: "59b1469-dirty",
		},
		{
			// go install of a commit: a pseudo-version, but no vcs settings.
			name: "go install of a commit",
			info: &debug.BuildInfo{Main: debug.Module{Version: "v0.0.0-20261016172523-59b1469d2f82"}},
			ok:   true,
			want: "59b1469",
		},
		{
			name: "pseudo-version after a pre-release",
			info: &debug.BuildInfo{Main: debug.Module{Version: "v1.1.0-rc.1.0.20261016172523-59b1469d2f82"}},
			ok:   true,
			want: "59b1469",
		},
		{
			name: "go run",
			info: &debug.BuildInfo{Main: debug.Module{Version: "(devel)"}},
			ok:   true,
			want: "dev",
		},
		{
			name: "no build info",
			ok:   false,
			want: "dev",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := short(tt.info, tt.ok); got != tt.want {
				t.Errorf("short() = %q, want %q", got, tt.want)
			}
		})
	}
}
