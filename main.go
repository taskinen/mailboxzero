// Mailbox Zero is a local web app that cleans up a Fastmail inbox: it finds
// groups of similar emails over JMAP and archives them.
//
// Install it with:
//
//	go install github.com/taskinen/mailboxzero@latest
//
// It reads its configuration from ~/.config/mailboxzero/config.yaml
// ($XDG_CONFIG_HOME/mailboxzero/config.yaml when that is set), or from the
// file given with -config. Create the file with
//
//	mkdir -p ~/.config/mailboxzero
//	mailboxzero -example-config > ~/.config/mailboxzero/config.yaml
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
	"path/filepath"

	"github.com/taskinen/mailboxzero/internal/config"
	"github.com/taskinen/mailboxzero/internal/jmap"
	"github.com/taskinen/mailboxzero/internal/server"
	"github.com/taskinen/mailboxzero/internal/version"
)

// exampleConfig is config.yaml.example, built in so that an installed binary
// can write it out with -example-config.
//
//go:embed config.yaml.example
var exampleConfig []byte

func main() {
	var configPath string
	var showVersion, printExampleConfig bool
	flag.StringVar(&configPath, "config", defaultConfigPath(), "Path to configuration file")
	flag.BoolVar(&showVersion, "version", false, "Print the version and exit")
	flag.BoolVar(&printExampleConfig, "example-config", false, "Print an example configuration file and exit")
	flag.Parse()

	if showVersion {
		fmt.Println(version.String())
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
	log.Printf("Using config file %s", configPath)

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

	log.Printf("Starting %s", version.String())
	if err := srv.Start(); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}

// defaultConfigPath returns where the configuration file is read from when
// -config is not given: mailboxzero/config.yaml in $XDG_CONFIG_HOME, or in
// ~/.config when that is not set (on every OS, macOS and Windows included).
func defaultConfigPath() string {
	dir := os.Getenv("XDG_CONFIG_HOME")
	// The XDG Base Directory spec says to ignore a relative path.
	if !filepath.IsAbs(dir) {
		home, err := os.UserHomeDir()
		if err != nil {
			return "config.yaml"
		}
		dir = filepath.Join(home, ".config")
	}
	return filepath.Join(dir, "mailboxzero", "config.yaml")
}

// loadConfig loads the configuration file, and says how to create one when
// it does not exist, as on the first run of an installed binary.
func loadConfig(path string) (*config.Config, error) {
	cfg, err := config.Load(path)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, fmt.Errorf("config file %s not found: create it with `mkdir -p %s && mailboxzero -example-config > %s` "+
			"and add your Fastmail API token, or give another file with -config: %w", path, filepath.Dir(path), path, err)
	}
	return cfg, err
}
