package main

import (
	"flag"
	"fmt"
	"log"
	"mailboxzero/internal/config"
	"mailboxzero/internal/protocol"
	"mailboxzero/internal/providers/imap"
	"mailboxzero/internal/providers/jmap"
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

	// Create email client based on configuration
	emailClient, err := createEmailClient(cfg)
	if err != nil {
		log.Fatalf("Failed to create email client: %v", err)
	}
	defer emailClient.Close()

	// Create and start server
	srv, err := server.New(cfg, emailClient)
	if err != nil {
		log.Fatalf("Failed to create server: %v", err)
	}

	log.Printf("Starting Mailbox Zero...")
	if err := srv.Start(); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}

// createEmailClient is a factory function that creates the appropriate email client
// based on the configuration (mock mode, protocol type).
func createEmailClient(cfg *config.Config) (protocol.EmailClient, error) {
	if cfg.MockMode {
		log.Println("Starting in MOCK MODE - using sample data")

		// Create mock client based on protocol
		switch cfg.GetProtocolType() {
		case protocol.ProtocolJMAP:
			return jmap.NewMockClient(), nil
		case protocol.ProtocolIMAP:
			log.Println("Using IMAP mock client")
			return imap.NewMockClient(), nil
		default:
			return jmap.NewMockClient(), nil // Default to JMAP mock
		}
	}

	// Create real client based on protocol
	switch cfg.GetProtocolType() {
	case protocol.ProtocolJMAP:
		log.Println("Connecting to JMAP server...")
		client := jmap.NewClient(cfg.JMAP.Endpoint, cfg.JMAP.APIToken)

		log.Println("Authenticating with JMAP server...")
		if err := client.Authenticate(); err != nil {
			return nil, err
		}
		log.Println("JMAP authentication successful!")

		return client, nil

	case protocol.ProtocolIMAP:
		log.Println("Connecting to IMAP server...")
		client := imap.NewClient(
			cfg.IMAP.Host,
			cfg.IMAP.Port,
			cfg.IMAP.Username,
			cfg.IMAP.Password,
			cfg.IMAP.UseTLS,
			cfg.IMAP.ArchiveFolder,
		)

		log.Println("Authenticating with IMAP server...")
		if err := client.Authenticate(); err != nil {
			return nil, err
		}
		log.Println("IMAP authentication successful!")

		return client, nil

	default:
		return nil, fmt.Errorf("unsupported protocol: %s", cfg.Protocol)
	}
}
