// Package imap provides IMAP mock implementation for testing
package imap

import (
	"fmt"
	"log"
	"time"

	"mailboxzero/internal/protocol"
)

// MockClient implements protocol.EmailClient for testing without a real IMAP server
type MockClient struct {
	sampleEmails []protocol.Email
	archivedIDs  map[string]bool
}

// NewMockClient creates a new mock IMAP client with sample data
func NewMockClient() *MockClient {
	return &MockClient{
		sampleEmails: generateSampleIMAPEmails(),
		archivedIDs:  make(map[string]bool),
	}
}

// Authenticate simulates authentication (always succeeds for mock)
func (m *MockClient) Authenticate() error {
	log.Println("Mock IMAP: Authentication successful")
	return nil
}

// GetInboxEmailsWithCountPaginated returns paginated sample emails
func (m *MockClient) GetInboxEmailsWithCountPaginated(limit, offset int) (*protocol.InboxInfo, error) {
	// Filter out archived emails
	inboxEmails := make([]protocol.Email, 0)
	for _, email := range m.sampleEmails {
		if !m.archivedIDs[email.ID] {
			inboxEmails = append(inboxEmails, email)
		}
	}

	totalCount := len(inboxEmails)

	// Apply pagination
	start := offset
	if start > totalCount {
		start = totalCount
	}

	end := start + limit
	if end > totalCount {
		end = totalCount
	}

	emails := inboxEmails[start:end]

	return &protocol.InboxInfo{
		Emails:     emails,
		TotalCount: totalCount,
	}, nil
}

// GetInboxEmails returns sample emails (for compatibility)
func (m *MockClient) GetInboxEmails(limit int) ([]protocol.Email, error) {
	info, err := m.GetInboxEmailsWithCountPaginated(limit, 0)
	if err != nil {
		return nil, err
	}
	return info.Emails, nil
}

// ArchiveEmails simulates archiving emails
func (m *MockClient) ArchiveEmails(emailIDs []string, dryRun bool) error {
	if dryRun {
		log.Printf("Mock IMAP DRY RUN: Would archive %d emails", len(emailIDs))
		return nil
	}

	for _, id := range emailIDs {
		m.archivedIDs[id] = true
	}

	log.Printf("Mock IMAP: Archived %d emails", len(emailIDs))
	return nil
}

// Close simulates closing the connection (no-op for mock)
func (m *MockClient) Close() error {
	log.Println("Mock IMAP: Connection closed")
	return nil
}

// generateSampleIMAPEmails creates realistic sample email data
func generateSampleIMAPEmails() []protocol.Email {
	baseTime := time.Now().Add(-24 * 30 * time.Hour) // 30 days ago
	emails := []protocol.Email{}

	// Gmail-style newsletters
	for i := 0; i < 5; i++ {
		emails = append(emails, protocol.Email{
			ID:      fmt.Sprintf("imap-%d", 1000+i),
			Subject: fmt.Sprintf("Gmail Newsletter #%d - Tips and Updates", i+1),
			From: []protocol.EmailAddress{
				{Name: "Gmail Team", Email: "no-reply@google.com"},
			},
			To: []protocol.EmailAddress{
				{Name: "You", Email: "user@example.com"},
			},
			ReceivedAt:    baseTime.Add(time.Duration(i) * 24 * time.Hour),
			Preview:       "Discover new features and tips for Gmail...",
			HasAttachment: false,
			BodyText:      "Check out these new Gmail features and productivity tips.",
			Flags:         []string{"\\Seen"},
		})
	}

	// GitHub notifications
	for i := 0; i < 8; i++ {
		emails = append(emails, protocol.Email{
			ID:      fmt.Sprintf("imap-%d", 2000+i),
			Subject: fmt.Sprintf("[GitHub] Issue #%d was updated", 100+i),
			From: []protocol.EmailAddress{
				{Name: "GitHub", Email: "notifications@github.com"},
			},
			To: []protocol.EmailAddress{
				{Name: "You", Email: "user@example.com"},
			},
			ReceivedAt:    baseTime.Add(time.Duration(i*2) * 24 * time.Hour),
			Preview:       "A new comment was added to your issue...",
			HasAttachment: false,
			BodyText:      "Someone commented on your issue. View it on GitHub.",
			Flags:         []string{},
		})
	}

	// LinkedIn connection requests
	for i := 0; i < 6; i++ {
		emails = append(emails, protocol.Email{
			ID:      fmt.Sprintf("imap-%d", 3000+i),
			Subject: fmt.Sprintf("Person %d wants to connect on LinkedIn", i+1),
			From: []protocol.EmailAddress{
				{Name: "LinkedIn", Email: "invitations@linkedin.com"},
			},
			To: []protocol.EmailAddress{
				{Name: "You", Email: "user@example.com"},
			},
			ReceivedAt:    baseTime.Add(time.Duration(i*3) * 24 * time.Hour),
			Preview:       "Accept this invitation to expand your network...",
			HasAttachment: false,
			BodyText:      "You have a new connection request on LinkedIn.",
			Flags:         []string{"\\Seen"},
		})
	}

	// Stack Overflow notifications
	for i := 0; i < 4; i++ {
		emails = append(emails, protocol.Email{
			ID:      fmt.Sprintf("imap-%d", 4000+i),
			Subject: "New answers to your Stack Overflow question",
			From: []protocol.EmailAddress{
				{Name: "Stack Overflow", Email: "do-not-reply@stackoverflow.com"},
			},
			To: []protocol.EmailAddress{
				{Name: "You", Email: "user@example.com"},
			},
			ReceivedAt:    baseTime.Add(time.Duration(i*4) * 24 * time.Hour),
			Preview:       "Your question has received new answers...",
			HasAttachment: false,
			BodyText:      "Check out the new answers to your programming question.",
			Flags:         []string{},
		})
	}

	// Amazon order confirmations
	for i := 0; i < 3; i++ {
		emails = append(emails, protocol.Email{
			ID:      fmt.Sprintf("imap-%d", 5000+i),
			Subject: fmt.Sprintf("Your Amazon.com order #%d has shipped", 100+i),
			From: []protocol.EmailAddress{
				{Name: "Amazon.com", Email: "ship-confirm@amazon.com"},
			},
			To: []protocol.EmailAddress{
				{Name: "You", Email: "user@example.com"},
			},
			ReceivedAt:    baseTime.Add(time.Duration(i*7) * 24 * time.Hour),
			Preview:       "Your package is on the way...",
			HasAttachment: false,
			BodyText:      "Track your shipment and view order details.",
			Flags:         []string{"\\Seen", "\\Flagged"},
		})
	}

	// Medium digest
	for i := 0; i < 4; i++ {
		emails = append(emails, protocol.Email{
			ID:      fmt.Sprintf("imap-%d", 6000+i),
			Subject: "Daily Digest from Medium - Top Stories",
			From: []protocol.EmailAddress{
				{Name: "Medium Daily Digest", Email: "noreply@medium.com"},
			},
			To: []protocol.EmailAddress{
				{Name: "You", Email: "user@example.com"},
			},
			ReceivedAt:    baseTime.Add(time.Duration(i*5) * 24 * time.Hour),
			Preview:       "Here are today's most recommended stories...",
			HasAttachment: false,
			BodyText:      "Discover the best stories from writers you follow.",
			Flags:         []string{},
		})
	}

	// Slack notifications
	for i := 0; i < 7; i++ {
		emails = append(emails, protocol.Email{
			ID:      fmt.Sprintf("imap-%d", 7000+i),
			Subject: fmt.Sprintf("[@channel] New message in #%s", []string{"general", "random", "dev"}[i%3]),
			From: []protocol.EmailAddress{
				{Name: "Slack", Email: "feedback@slack.com"},
			},
			To: []protocol.EmailAddress{
				{Name: "You", Email: "user@example.com"},
			},
			ReceivedAt:    baseTime.Add(time.Duration(i) * 12 * time.Hour),
			Preview:       "You have new messages in your Slack workspace...",
			HasAttachment: false,
			BodyText:      "Check your Slack workspace for new activity.",
			Flags:         []string{"\\Recent"},
		})
	}

	// Work emails
	emails = append(emails, protocol.Email{
		ID:      "imap-8001",
		Subject: "Important: Team meeting tomorrow",
		From: []protocol.EmailAddress{
			{Name: "Boss Person", Email: "boss@company.com"},
		},
		To: []protocol.EmailAddress{
			{Name: "You", Email: "user@example.com"},
		},
		Cc: []protocol.EmailAddress{
			{Name: "Team", Email: "team@company.com"},
		},
		ReceivedAt:    baseTime.Add(10 * 24 * time.Hour),
		Preview:       "Don't forget about our important team meeting...",
		HasAttachment: true,
		BodyText:      "Please review the attached agenda for tomorrow's meeting.",
		BodyHTML:      "<p>Please review the attached agenda for tomorrow's meeting.</p>",
		Flags:         []string{"\\Seen", "\\Flagged"},
	})

	// Personal email
	emails = append(emails, protocol.Email{
		ID:      "imap-9001",
		Subject: "Re: Weekend plans",
		From: []protocol.EmailAddress{
			{Name: "Friend Name", Email: "friend@personal.com"},
		},
		To: []protocol.EmailAddress{
			{Name: "You", Email: "user@example.com"},
		},
		ReceivedAt:    baseTime.Add(15 * 24 * time.Hour),
		Preview:       "Sounds great! Let's meet at 2pm...",
		HasAttachment: false,
		BodyText:      "Looking forward to catching up this weekend!",
		Flags:         []string{"\\Seen"},
	})

	return emails
}
