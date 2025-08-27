package jmap

import (
	"fmt"
	"math/rand"
	"time"
)

// MockClient implements the JMAP client interface but returns sample data
type MockClient struct {
	sampleEmails []Email
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

// Authenticate always succeeds for mock client
func (m *MockClient) Authenticate() error {
	return nil
}

// GetPrimaryAccount returns a mock account ID
func (m *MockClient) GetPrimaryAccount() string {
	return "mock-account-123"
}

// GetMailboxes returns mock mailboxes
func (m *MockClient) GetMailboxes() ([]Mailbox, error) {
	return []Mailbox{
		{
			ID:   "inbox-123",
			Name: "Inbox",
			Role: "inbox",
		},
		{
			ID:   "archive-456",
			Name: "Archive",
			Role: "archive",
		},
	}, nil
}

// GetInboxEmails returns the sample emails that haven't been archived
func (m *MockClient) GetInboxEmails(limit int) ([]Email, error) {
	return m.GetInboxEmailsPaginated(limit, 0)
}

// GetInboxEmailsPaginated returns paginated sample emails that haven't been archived
func (m *MockClient) GetInboxEmailsPaginated(limit, offset int) ([]Email, error) {
	var inboxEmails []Email
	for _, email := range m.sampleEmails {
		if !m.archivedIDs[email.ID] {
			inboxEmails = append(inboxEmails, email)
		}
	}

	// Apply pagination
	start := offset
	if start >= len(inboxEmails) {
		return []Email{}, nil
	}
	
	end := start + limit
	if end > len(inboxEmails) {
		end = len(inboxEmails)
	}

	return inboxEmails[start:end], nil
}

// GetInboxEmailsWithCount returns sample emails with total count
func (m *MockClient) GetInboxEmailsWithCount(limit int) (*InboxInfo, error) {
	return m.GetInboxEmailsWithCountPaginated(limit, 0)
}

// GetInboxEmailsWithCountPaginated returns paginated sample emails with total count
func (m *MockClient) GetInboxEmailsWithCountPaginated(limit, offset int) (*InboxInfo, error) {
	// Count all non-archived emails
	totalCount := 0
	for _, email := range m.sampleEmails {
		if !m.archivedIDs[email.ID] {
			totalCount++
		}
	}

	emails, err := m.GetInboxEmailsPaginated(limit, offset)
	if err != nil {
		return nil, err
	}

	return &InboxInfo{
		Emails:     emails,
		TotalCount: totalCount,
	}, nil
}

// ArchiveEmails simulates archiving by marking emails as archived
func (m *MockClient) ArchiveEmails(emailIDs []string, dryRun bool) error {
	if dryRun {
		fmt.Printf("[MOCK DRY RUN] Would archive %d emails: %v\n", len(emailIDs), emailIDs)
		return nil
	}
	
	fmt.Printf("[MOCK MODE] Archiving %d emails: %v\n", len(emailIDs), emailIDs)
	for _, id := range emailIDs {
		m.archivedIDs[id] = true
	}
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
			email := Email{
				ID:       fmt.Sprintf("email-%d-%d", i, j),
				Subject:  baseSubject,
				From:     []EmailAddress{{Email: sender, Name: extractNameFromEmail(sender)}},
				Preview:  baseContent,
				ReceivedAt: baseTime.Add(time.Duration(i*24+j*6) * time.Hour),
				BodyValues: map[string]BodyValue{
					"text": {Value: baseContent + " This is additional content for the email body."},
				},
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
	uniqueEmails := []Email{
		{
			ID:       "unique-1",
			Subject:  "Welcome to our platform!",
			From:     []EmailAddress{{Email: "welcome@newservice.com", Name: "New Service"}},
			Preview:  "Thanks for signing up! Here's how to get started.",
			ReceivedAt: baseTime.Add(48 * time.Hour),
			BodyValues: map[string]BodyValue{
				"text": {Value: "Welcome! We're excited to have you on board."},
			},
		},
		{
			ID:       "unique-2",
			Subject:  "Conference invitation",
			From:     []EmailAddress{{Email: "events@techconf.com", Name: "Tech Conference"}},
			Preview:  "You're invited to speak at our upcoming conference.",
			ReceivedAt: baseTime.Add(72 * time.Hour),
			BodyValues: map[string]BodyValue{
				"text": {Value: "We'd love to have you present at our conference."},
			},
		},
	}
	
	m.sampleEmails = append(m.sampleEmails, uniqueEmails...)
}

// extractNameFromEmail creates a display name from an email address
func extractNameFromEmail(email string) string {
	names := map[string]string{
		"notifications@github.com": "GitHub",
		"support@stripe.com":       "Stripe Support",
		"noreply@amazon.com":       "Amazon",
		"alerts@uptime.com":        "Uptime Alerts",
		"newsletter@techcrunch.com": "TechCrunch",
		"billing@digitalocean.com": "DigitalOcean",
		"security@google.com":      "Google Security",
		"team@slack.com":           "Slack",
		"updates@docker.com":       "Docker",
		"info@mailchimp.com":       "Mailchimp",
		"welcome@newservice.com":   "New Service",
		"events@techconf.com":      "Tech Conference",
	}
	
	if name, ok := names[email]; ok {
		return name
	}
	return email
}