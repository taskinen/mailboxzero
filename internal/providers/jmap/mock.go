package jmap

import (
	"fmt"
	"mailboxzero/internal/protocol"
	"math/rand"
	"time"
)

// MockClient implements protocol.EmailClient but returns sample data
type MockClient struct {
	sampleEmails []protocol.Email
	archivedIDs  map[string]bool
}

// NewMockClient creates a new mock JMAP client with sample data
func NewMockClient() *MockClient {
	mock := &MockClient{
		archivedIDs: make(map[string]bool),
	}
	mock.generateSampleEmails()
	return mock
}

// Authenticate implements protocol.EmailClient - always succeeds for mock client
func (m *MockClient) Authenticate() error {
	return nil
}

// GetInboxEmails implements protocol.EmailClient - returns sample emails that haven't been archived
func (m *MockClient) GetInboxEmails(limit int) ([]protocol.Email, error) {
	info, err := m.GetInboxEmailsWithCountPaginated(limit, 0)
	if err != nil {
		return nil, err
	}
	return info.Emails, nil
}

// GetInboxEmailsWithCountPaginated implements protocol.EmailClient
func (m *MockClient) GetInboxEmailsWithCountPaginated(limit, offset int) (*protocol.InboxInfo, error) {
	var inboxEmails []protocol.Email
	for _, email := range m.sampleEmails {
		if !m.archivedIDs[email.ID] {
			inboxEmails = append(inboxEmails, email)
		}
	}

	totalCount := len(inboxEmails)

	// Apply pagination
	start := offset
	if start >= len(inboxEmails) {
		return &protocol.InboxInfo{
			Emails:     []protocol.Email{},
			TotalCount: totalCount,
		}, nil
	}

	end := start + limit
	if end > len(inboxEmails) {
		end = len(inboxEmails)
	}

	return &protocol.InboxInfo{
		Emails:     inboxEmails[start:end],
		TotalCount: totalCount,
	}, nil
}

// ArchiveEmails implements protocol.EmailClient - simulates archiving by marking emails as archived
func (m *MockClient) ArchiveEmails(emailIDs []string, dryRun bool) error {
	if dryRun {
		fmt.Printf("[JMAP MOCK DRY RUN] Would archive %d emails: %v\n", len(emailIDs), emailIDs)
		return nil
	}

	fmt.Printf("[JMAP MOCK MODE] Archiving %d emails: %v\n", len(emailIDs), emailIDs)
	for _, id := range emailIDs {
		m.archivedIDs[id] = true
	}
	return nil
}

// Close implements protocol.EmailClient - nothing to close for mock
func (m *MockClient) Close() error {
	return nil
}

// generateSampleEmails creates realistic sample email data
func (m *MockClient) generateSampleEmails() {
	senders := []string{
		"notifications@github.com",
		"support@stripe.com",
		"noreply@amazon.com",
		"alerts@uptime.com",
		"newsletter@techcrunch.com",
		"billing@digitalocean.com",
		"security@google.com",
		"team@slack.com",
		"updates@docker.com",
		"info@mailchimp.com",
	}

	subjects := []string{
		"Weekly deployment summary",
		"Payment confirmation",
		"Your order has been shipped",
		"Service alert: downtime detected",
		"This week in tech news",
		"Monthly billing statement",
		"Security alert: new sign-in",
		"Daily digest from your team",
		"New Docker image available",
		"Campaign performance report",
	}

	contents := []string{
		"This is your weekly summary of deployments and system status.",
		"Thank you for your payment. Your invoice has been processed.",
		"Great news! Your order is on its way and will arrive soon.",
		"We've detected unusual activity and wanted to notify you immediately.",
		"Here are the most important tech stories from this week.",
		"Your monthly statement is now available for review.",
		"We noticed a new sign-in to your account from an unknown device.",
		"Here's what your team has been working on today.",
		"A new version of your favorite Docker image is ready to use.",
		"See how your latest email campaign performed with detailed analytics.",
	}

	// Create similar email groups
	baseTime := time.Now().AddDate(0, 0, -30)

	for i := 0; i < len(senders); i++ {
		sender := senders[i]
		baseSubject := subjects[i]
		baseContent := contents[i]

		// Create 3-5 similar emails for each sender
		numSimilar := 3 + rand.Intn(3)
		for j := 0; j < numSimilar; j++ {
			email := protocol.Email{
				ID:         fmt.Sprintf("email-%d-%d", i, j),
				Subject:    baseSubject,
				From:       []protocol.EmailAddress{{Email: sender, Name: extractNameFromEmail(sender)}},
				Preview:    baseContent,
				ReceivedAt: baseTime.Add(time.Duration(i*24+j*6) * time.Hour),
				BodyText:   baseContent + " This is additional content for the email body.",
			}

			// Add slight variations to subjects for some emails
			if j > 0 {
				variations := []string{
					" - Follow up",
					" - Updated",
					" - Reminder",
					" #" + fmt.Sprintf("%d", j+1),
				}
				email.Subject += variations[j%len(variations)]
			}

			m.sampleEmails = append(m.sampleEmails, email)
		}
	}

	// Add some unique emails
	uniqueEmails := []protocol.Email{
		{
			ID:         "unique-1",
			Subject:    "Welcome to our platform!",
			From:       []protocol.EmailAddress{{Email: "welcome@newservice.com", Name: "New Service"}},
			Preview:    "Thanks for signing up! Here's how to get started.",
			ReceivedAt: baseTime.Add(48 * time.Hour),
			BodyText:   "Welcome! We're excited to have you on board.",
		},
		{
			ID:         "unique-2",
			Subject:    "Conference invitation",
			From:       []protocol.EmailAddress{{Email: "events@techconf.com", Name: "Tech Conference"}},
			Preview:    "You're invited to speak at our upcoming conference.",
			ReceivedAt: baseTime.Add(72 * time.Hour),
			BodyText:   "We'd love to have you present at our conference.",
		},
	}

	m.sampleEmails = append(m.sampleEmails, uniqueEmails...)
}

// extractNameFromEmail creates a display name from an email address
func extractNameFromEmail(email string) string {
	names := map[string]string{
		"notifications@github.com":  "GitHub",
		"support@stripe.com":        "Stripe Support",
		"noreply@amazon.com":        "Amazon",
		"alerts@uptime.com":         "Uptime Alerts",
		"newsletter@techcrunch.com": "TechCrunch",
		"billing@digitalocean.com":  "DigitalOcean",
		"security@google.com":       "Google Security",
		"team@slack.com":            "Slack",
		"updates@docker.com":        "Docker",
		"info@mailchimp.com":        "Mailchimp",
		"welcome@newservice.com":    "New Service",
		"events@techconf.com":       "Tech Conference",
	}

	if name, ok := names[email]; ok {
		return name
	}
	return email
}
