package main

import (
	"flag"
	"log"
	"mailboxzero/internal/config"
	"mailboxzero/internal/jmap"
	"mailboxzero/internal/server"
)

func main() {
	var configPath string
	flag.StringVar(&configPath, "config", "config.yaml", "Path to configuration file")
	flag.Parse()

	cfg, err := config.Load(configPath)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	jmapClient := jmap.NewClient(cfg.JMAP.Endpoint, cfg.JMAP.APIToken)

	log.Println("Authenticating with JMAP server...")
	if err := jmapClient.Authenticate(); err != nil {
		log.Fatalf("Failed to authenticate: %v", err)
	}
	log.Println("Authentication successful!")

	srv, err := server.New(cfg, jmapClient)
	if err != nil {
		log.Fatalf("Failed to create server: %v", err)
	}

	log.Printf("Starting Mailbox Zero...")
	if err := srv.Start(); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
