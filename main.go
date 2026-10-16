// Mailbox Zero is a local web app that cleans up a Fastmail inbox: it finds
// groups of similar emails over JMAP and archives them.
//
// Install it with:
//
//	go install github.com/taskinen/mailboxzero@latest
//
// Then create a configuration file with
//
//	mailboxzero -example-config > config.yaml
//
// add a Fastmail API token to it, run mailboxzero and open
// http://localhost:8080.
package main

import (
	_ "embed"
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"os"
	"runtime"
	"runtime/debug"

	"github.com/taskinen/mailboxzero/internal/config"
	"github.com/taskinen/mailboxzero/internal/jmap"
	"github.com/taskinen/mailboxzero/internal/server"
)

// exampleConfig is config.yaml.example, built in so that an installed binary
// can write it out with -example-config.
//
//go:embed config.yaml.example
var exampleConfig []byte

func main() {
	var configPath string
	var showVersion, printExampleConfig bool
	flag.StringVar(&configPath, "config", "config.yaml", "Path to configuration file")
	flag.BoolVar(&showVersion, "version", false, "Print the version and exit")
	flag.BoolVar(&printExampleConfig, "example-config", false, "Print an example configuration file and exit")
	flag.Parse()

	if showVersion {
		fmt.Println(versionString())
		return
	}
	if printExampleConfig {
		if _, err := os.Stdout.Write(exampleConfig); err != nil {
			log.Fatalf("Failed to write the example config: %v", err)
		}
		return
	}

	cfg, err := loadConfig(configPath)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	var jmapClient jmap.JMAPClient

	if cfg.MockMode {
		log.Println("Starting in MOCK MODE - using sample data")
		jmapClient = jmap.NewMockClient()
	} else {
		log.Println("Connecting to Fastmail JMAP server...")
		realClient := jmap.NewClient(cfg.JMAP.Endpoint, cfg.JMAP.APIToken)

		log.Println("Authenticating with JMAP server...")
		if err := realClient.Authenticate(); err != nil {
			log.Fatalf("Failed to authenticate: %v", err)
		}
		log.Println("Authentication successful!")

		jmapClient = realClient
	}

	srv, err := server.New(cfg, jmapClient)
	if err != nil {
		log.Fatalf("Failed to create server: %v", err)
	}

	log.Printf("Starting %s", versionString())
	if err := srv.Start(); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}

// loadConfig loads the configuration file, and says how to create one when
// it does not exist, as on the first run of an installed binary.
func loadConfig(path string) (*config.Config, error) {
	cfg, err := config.Load(path)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, fmt.Errorf("config file %s not found: create one with `mailboxzero -example-config > %s` "+
			"and add your Fastmail API token: %w", path, path, err)
	}
	return cfg, err
}

// versionString describes the running binary, from the build information Go
// records in it.
func versionString() string {
	info, ok := debug.ReadBuildInfo()
	return versionFrom(info, ok)
}

// versionFrom returns "mailboxzero <version> (<Go version>, <OS>/<arch>)".
// The version is the release tag for go install ...@vX.Y.Z and for release
// builds, a pseudo-version for a go build in a checkout between releases
// (with +dirty for uncommitted changes), and "(devel)" for go run.
func versionFrom(info *debug.BuildInfo, ok bool) string {
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
