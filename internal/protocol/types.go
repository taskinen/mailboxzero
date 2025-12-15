// Package protocol provides protocol-agnostic data structures for email operations.
package protocol

import "time"

// Email represents a protocol-agnostic email message.
// This structure unifies email representation across different protocols (JMAP, IMAP).
type Email struct {
	ID            string         `json:"id"`
	Subject       string         `json:"subject"`
	From          []EmailAddress `json:"from"`
	To            []EmailAddress `json:"to"`
	Cc            []EmailAddress `json:"cc"`
	Bcc           []EmailAddress `json:"bcc"`
	ReplyTo       []EmailAddress `json:"replyTo"`
	ReceivedAt    time.Time      `json:"receivedAt"`
	SentAt        time.Time      `json:"sentAt"`
	Preview       string         `json:"preview"`
	HasAttachment bool           `json:"hasAttachment"`
	Size          int            `json:"size"`

	// Simplified body content (vs JMAP's complex BodyValues map + TextBody/HTMLBody arrays)
	BodyText string `json:"bodyText"` // Plain text body content
	BodyHTML string `json:"bodyHTML"` // HTML body content

	// Generic flags (e.g., "seen", "flagged", "draft")
	// Replaces JMAP's Keywords map[string]bool
	Flags []string `json:"flags"`

	// Protocol-specific data can be stored here if needed
	// For example, JMAP might store ThreadID, IMAP might store UID
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// EmailAddress represents an email address with optional display name.
type EmailAddress struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

// InboxInfo contains inbox emails with pagination metadata.
type InboxInfo struct {
	Emails     []Email `json:"emails"`
	TotalCount int     `json:"totalCount"`
}
