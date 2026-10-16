// Package version describes the running binary from the build information
// Go records in every binary: the module version, which is the release tag
// for a release and for go install ...@vX.Y.Z, and for a build in a Git
// checkout the commit it was built from. Nothing is set at build time.
package version

import (
	"fmt"
	"regexp"
	"runtime"
	"runtime/debug"
	"strings"
)

// String returns "mailboxzero <version> (<Go version>, <OS>/<arch>)", as
// printed by -version and logged at startup.
func String() string {
	info, ok := debug.ReadBuildInfo()
	return describe(info, ok)
}

// Short returns the label the web page shows: the release version
// ("v1.0.0"), otherwise the commit the binary was built from ("59b1469",
// with "-dirty" for uncommitted changes), or "dev" when Go recorded neither,
// as for go run.
func Short() string {
	info, ok := debug.ReadBuildInfo()
	return short(info, ok)
}

// describe is String for the given build information. The version is the
// module version Go recorded: the tag, a pseudo-version for a build between
// releases (with +dirty for uncommitted changes), or "(devel)".
func describe(info *debug.BuildInfo, ok bool) string {
	version := "(devel)"
	goVersion := runtime.Version()
	if ok {
		if info.Main.Version != "" {
			version = info.Main.Version
		}
		if info.GoVersion != "" {
			goVersion = info.GoVersion
		}
	}
	return fmt.Sprintf("mailboxzero %s (%s, %s/%s)", version, goVersion, runtime.GOOS, runtime.GOARCH)
}

// pseudoVersion matches the end of a Go pseudo-version, the version Go
// gives a commit that is not tagged, such as
// v0.0.0-20261016172523-59b1469d2f82 or v1.0.1-0.20261016172523-59b1469d2f82.
// Its last part is the start of the commit hash.
var pseudoVersion = regexp.MustCompile(`[-.]\d{14}-([0-9a-f]{12})(\+dirty)?$`)

// short is Short for the given build information.
func short(info *debug.BuildInfo, ok bool) string {
	if !ok {
		return "dev"
	}
	version := info.Main.Version
	if version != "" && version != "(devel)" && !pseudoVersion.MatchString(version) && !strings.HasSuffix(version, "+dirty") {
		return version
	}

	// go build in a checkout records the commit and whether files were
	// changed.
	var revision string
	var modified bool
	for _, setting := range info.Settings {
		switch setting.Key {
		case "vcs.revision":
			revision = setting.Value
		case "vcs.modified":
			modified = setting.Value == "true"
		}
	}
	if revision != "" {
		label := revision[:min(7, len(revision))]
		if modified {
			label += "-dirty"
		}
		return label
	}

	// go install of a commit records no vcs settings, but its
	// pseudo-version names the commit.
	if match := pseudoVersion.FindStringSubmatch(version); match != nil {
		return match[1][:7]
	}
	return "dev"
}
