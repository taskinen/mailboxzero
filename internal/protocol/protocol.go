// Package protocol provides the generic email client interface that all
// email protocol implementations (JMAP, IMAP, etc.) must satisfy.
package protocol

// EmailClient is the generic interface that all email protocol implementations must satisfy.
// This abstraction allows the application to work with JMAP, IMAP, or any future protocol
// without changing the core application logic.
type EmailClient interface {
	// Authenticate establishes a connection and authenticates with the email server.
	// Returns an error if authentication fails.
	Authenticate() error

	// GetInboxEmailsWithCountPaginated retrieves emails from the inbox with pagination support.
	// limit: maximum number of emails to retrieve
	// offset: number of emails to skip (for pagination)
	// Returns inbox information with emails and total count, or an error.
	GetInboxEmailsWithCountPaginated(limit, offset int) (*InboxInfo, error)

	// GetInboxEmails retrieves emails from the inbox without pagination.
	// This is a convenience method that typically calls GetInboxEmailsWithCountPaginated with offset=0.
	// limit: maximum number of emails to retrieve
	// Returns a list of emails or an error.
	GetInboxEmails(limit int) ([]Email, error)

	// ArchiveEmails moves the specified emails to the archive folder.
	// emailIDs: list of email identifiers to archive
	// dryRun: if true, simulates the operation without making actual changes
	// Returns an error if the operation fails.
	ArchiveEmails(emailIDs []string, dryRun bool) error

	// Close releases any resources held by the client (connections, etc.)
	// Should be called when the client is no longer needed.
	Close() error
}

// ProtocolType identifies the email protocol being used.
type ProtocolType string

const (
	// ProtocolJMAP represents the JMAP protocol (JSON Meta Application Protocol)
	ProtocolJMAP ProtocolType = "jmap"

	// ProtocolIMAP represents the IMAP protocol (Internet Message Access Protocol)
	ProtocolIMAP ProtocolType = "imap"
)
