// Package imap provides IMAP (Internet Message Access Protocol) implementation
// for the mailboxzero email client using go-imap v1.
package imap

import (
	"crypto/tls"
	"fmt"
	"log"
	"strconv"
	"strings"

	"mailboxzero/internal/protocol"

	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/client"
)

// Client implements protocol.EmailClient for IMAP servers
type Client struct {
	host          string
	port          int
	username      string
	password      string
	useTLS        bool
	archiveFolder string

	client        *client.Client
	authenticated bool
	inboxName     string
	totalMessages uint32
}

// NewClient creates a new IMAP client
func NewClient(host string, port int, username, password string, useTLS bool, archiveFolder string) *Client {
	return &Client{
		host:          host,
		port:          port,
		username:      username,
		password:      password,
		useTLS:        useTLS,
		archiveFolder: archiveFolder,
		inboxName:     "INBOX",
	}
}

// Authenticate connects to the IMAP server and authenticates
func (c *Client) Authenticate() error {
	addr := fmt.Sprintf("%s:%d", c.host, c.port)

	var imapClient *client.Client
	var err error

	// Connect with TLS
	if c.useTLS {
		tlsConfig := &tls.Config{
			ServerName: c.host,
		}
		imapClient, err = client.DialTLS(addr, tlsConfig)
		if err != nil {
			return fmt.Errorf("failed to connect to IMAP server: %w", err)
		}
	} else {
		// Plain connection (not recommended)
		imapClient, err = client.Dial(addr)
		if err != nil {
			return fmt.Errorf("failed to connect to IMAP server: %w", err)
		}
	}

	c.client = imapClient

	// Authenticate
	if err := c.client.Login(c.username, c.password); err != nil {
		return fmt.Errorf("failed to authenticate: %w", err)
	}

	c.authenticated = true
	log.Printf("Successfully authenticated to IMAP server: %s", c.host)

	return nil
}

// GetInboxEmailsWithCountPaginated retrieves emails from inbox with pagination
func (c *Client) GetInboxEmailsWithCountPaginated(limit, offset int) (*protocol.InboxInfo, error) {
	if !c.authenticated {
		return nil, fmt.Errorf("not authenticated")
	}

	// Select INBOX
	mbox, err := c.client.Select(c.inboxName, false)
	if err != nil {
		return nil, fmt.Errorf("failed to select inbox: %w", err)
	}

	c.totalMessages = mbox.Messages

	// Calculate message range for pagination (reverse chronological order)
	// Most recent emails first
	if c.totalMessages == 0 {
		return &protocol.InboxInfo{
			Emails:     []protocol.Email{},
			TotalCount: 0,
		}, nil
	}

	// Convert offset to sequence number (from newest)
	startSeq := c.totalMessages - uint32(offset)
	if startSeq < 1 || offset >= int(c.totalMessages) {
		// Offset beyond available messages
		return &protocol.InboxInfo{
			Emails:     []protocol.Email{},
			TotalCount: int(c.totalMessages),
		}, nil
	}

	endSeq := startSeq - uint32(limit) + 1
	if endSeq < 1 {
		endSeq = 1
	}

	// Build sequence set (endSeq:startSeq for reverse order)
	seqSet := new(imap.SeqSet)
	seqSet.AddRange(endSeq, startSeq)

	// Fetch email data
	messages := make(chan *imap.Message, 10)
	done := make(chan error, 1)

	go func() {
		done <- c.client.Fetch(seqSet, []imap.FetchItem{
			imap.FetchEnvelope,
			imap.FetchFlags,
			imap.FetchInternalDate,
			imap.FetchUid,
			imap.FetchBodyStructure,
		}, messages)
	}()

	// Collect emails
	emails := []protocol.Email{}
	for msg := range messages {
		email := c.convertIMAPMessage(msg)
		emails = append(emails, email)
	}

	if err := <-done; err != nil {
		return nil, fmt.Errorf("fetch error: %w", err)
	}

	return &protocol.InboxInfo{
		Emails:     emails,
		TotalCount: int(c.totalMessages),
	}, nil
}

// GetInboxEmails retrieves emails from inbox (for compatibility)
func (c *Client) GetInboxEmails(limit int) ([]protocol.Email, error) {
	info, err := c.GetInboxEmailsWithCountPaginated(limit, 0)
	if err != nil {
		return nil, err
	}
	return info.Emails, nil
}

// ArchiveEmails moves emails to the archive folder
func (c *Client) ArchiveEmails(emailIDs []string, dryRun bool) error {
	if !c.authenticated {
		return fmt.Errorf("not authenticated")
	}

	if len(emailIDs) == 0 {
		return nil
	}

	if dryRun {
		log.Printf("DRY RUN: Would archive %d emails to %s", len(emailIDs), c.archiveFolder)
		return nil
	}

	// Select INBOX
	if _, err := c.client.Select(c.inboxName, false); err != nil {
		return fmt.Errorf("failed to select inbox: %w", err)
	}

	// Parse email IDs and build UID set
	uidSet := new(imap.SeqSet)
	for _, emailID := range emailIDs {
		uid, err := parseEmailIDToUID(emailID)
		if err != nil {
			log.Printf("Warning: invalid email ID %s: %v", emailID, err)
			continue
		}
		uidSet.AddNum(uid)
	}

	if uidSet.Empty() {
		return fmt.Errorf("no valid email IDs to archive")
	}

	// Copy messages to archive folder
	if err := c.client.UidCopy(uidSet, c.archiveFolder); err != nil {
		return fmt.Errorf("failed to copy emails to archive: %w", err)
	}

	// Mark messages as deleted
	item := imap.FormatFlagsOp(imap.AddFlags, true)
	flags := []interface{}{imap.DeletedFlag}
	if err := c.client.UidStore(uidSet, item, flags, nil); err != nil {
		return fmt.Errorf("failed to mark emails as deleted: %w", err)
	}

	// Expunge deleted messages
	if err := c.client.Expunge(nil); err != nil {
		return fmt.Errorf("failed to expunge deleted emails: %w", err)
	}

	log.Printf("Successfully archived %d emails to %s", len(emailIDs), c.archiveFolder)
	return nil
}

// Close closes the IMAP connection
func (c *Client) Close() error {
	if c.client != nil {
		if err := c.client.Logout(); err != nil {
			log.Printf("Warning: logout error: %v", err)
		}
		c.authenticated = false
		log.Println("IMAP connection closed")
	}
	return nil
}

// convertIMAPMessage converts an IMAP message to protocol.Email
func (c *Client) convertIMAPMessage(msg *imap.Message) protocol.Email {
	email := protocol.Email{
		ID: formatEmailID(msg.Uid),
	}

	// Extract envelope data
	if msg.Envelope != nil {
		email.Subject = msg.Envelope.Subject
		email.From = convertAddresses(msg.Envelope.From)
		email.To = convertAddresses(msg.Envelope.To)
		email.Cc = convertAddresses(msg.Envelope.Cc)
		email.ReceivedAt = msg.Envelope.Date
	}

	// Use internal date as fallback
	if email.ReceivedAt.IsZero() && !msg.InternalDate.IsZero() {
		email.ReceivedAt = msg.InternalDate
	}

	// Extract flags
	if msg.Flags != nil {
		email.Flags = make([]string, len(msg.Flags))
		for i, flag := range msg.Flags {
			email.Flags[i] = flag
		}
	}

	// Check for attachments based on body structure
	if msg.BodyStructure != nil {
		email.HasAttachment = hasAttachments(msg.BodyStructure)
	}

	// Generate preview from subject and sender
	email.Preview = generatePreview(email.Subject, email.From)

	return email
}

// convertAddresses converts IMAP addresses to protocol.EmailAddress
func convertAddresses(addrs []*imap.Address) []protocol.EmailAddress {
	result := make([]protocol.EmailAddress, 0, len(addrs))
	for _, addr := range addrs {
		if addr == nil {
			continue
		}
		result = append(result, protocol.EmailAddress{
			Name:  addr.PersonalName,
			Email: addr.Address(),
		})
	}
	return result
}

// hasAttachments checks if the body structure indicates attachments
func hasAttachments(bs *imap.BodyStructure) bool {
	// Check if this part is an attachment
	if bs.Disposition == "attachment" {
		return true
	}

	// Check child parts for multipart messages
	for _, part := range bs.Parts {
		if hasAttachments(part) {
			return true
		}
	}

	return false
}

// generatePreview generates a preview from subject and sender
func generatePreview(subject string, from []protocol.EmailAddress) string {
	preview := subject
	if len(from) > 0 && from[0].Name != "" {
		preview = fmt.Sprintf("From: %s - %s", from[0].Name, subject)
	} else if len(from) > 0 && from[0].Email != "" {
		preview = fmt.Sprintf("From: %s - %s", from[0].Email, subject)
	}
	if len(preview) > 200 {
		preview = preview[:197] + "..."
	}
	return preview
}

// formatEmailID formats a UID as an email ID
func formatEmailID(uid uint32) string {
	return fmt.Sprintf("imap-%d", uid)
}

// parseEmailIDToUID parses an email ID to extract the UID
func parseEmailIDToUID(emailID string) (uint32, error) {
	if !strings.HasPrefix(emailID, "imap-") {
		return 0, fmt.Errorf("invalid IMAP email ID format: %s", emailID)
	}

	uidStr := strings.TrimPrefix(emailID, "imap-")
	uid, err := strconv.ParseUint(uidStr, 10, 32)
	if err != nil {
		return 0, fmt.Errorf("invalid UID in email ID %s: %w", emailID, err)
	}

	return uint32(uid), nil
}
