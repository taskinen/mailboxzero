package jmap

import (
	"testing"
)

func TestNewMockClient(t *testing.T) {
	client := NewMockClient()

	if client == nil {
		t.Fatal("NewMockClient() returned nil")
	}

	if client.sampleEmails == nil {
		t.Error("NewMockClient() sampleEmails is nil")
	}

	if len(client.sampleEmails) == 0 {
		t.Error("NewMockClient() should generate sample emails")
	}

	if client.archivedIDs == nil {
		t.Error("NewMockClient() archivedIDs is nil")
	}
}

func TestMockClient_Authenticate(t *testing.T) {
	client := NewMockClient()
	err := client.Authenticate()

	if err != nil {
		t.Errorf("MockClient.Authenticate() unexpected error = %v", err)
	}
}

func TestMockClient_GetPrimaryAccount(t *testing.T) {
	client := NewMockClient()
	accountID := client.GetPrimaryAccount()

	if accountID == "" {
		t.Error("MockClient.GetPrimaryAccount() returned empty string")
	}
}

func TestMockClient_GetMailboxes(t *testing.T) {
	client := NewMockClient()
	mailboxes, err := client.GetMailboxes()

	if err != nil {
		t.Errorf("MockClient.GetMailboxes() unexpected error = %v", err)
	}

	if len(mailboxes) == 0 {
		t.Error("MockClient.GetMailboxes() returned no mailboxes")
	}

	// Check for inbox
	foundInbox := false
	foundArchive := false
	for _, mb := range mailboxes {
		if mb.Role == "inbox" {
			foundInbox = true
		}
		if mb.Role == "archive" {
			foundArchive = true
		}
	}

	if !foundInbox {
		t.Error("MockClient.GetMailboxes() did not return inbox")
	}
	if !foundArchive {
		t.Error("MockClient.GetMailboxes() did not return archive")
	}
}

func TestMockClient_GetInboxEmails(t *testing.T) {
	client := NewMockClient()
	totalEmails := len(client.sampleEmails)

	tests := []struct {
		name      string
		limit     int
		wantCount int
	}{
		{
			name:      "get all emails",
			limit:     100,
			wantCount: totalEmails,
		},
		{
			name:      "get limited emails",
			limit:     5,
			wantCount: 5,
		},
		{
			name:      "get zero emails",
			limit:     0,
			wantCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			emails, err := client.GetInboxEmails(tt.limit)
			if err != nil {
				t.Errorf("MockClient.GetInboxEmails() unexpected error = %v", err)
			}

			if len(emails) > tt.wantCount {
				t.Errorf("MockClient.GetInboxEmails() returned %d emails, want at most %d",
					len(emails), tt.wantCount)
			}
		})
	}
}

func TestMockClient_GetInboxEmailsPaginated(t *testing.T) {
	client := NewMockClient()

	tests := []struct {
		name    string
		limit   int
		offset  int
		wantErr bool
	}{
		{
			name:    "first page",
			limit:   10,
			offset:  0,
			wantErr: false,
		},
		{
			name:    "second page",
			limit:   10,
			offset:  10,
			wantErr: false,
		},
		{
			name:    "offset beyond emails",
			limit:   10,
			offset:  1000,
			wantErr: false, // Should return empty slice, not error
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			emails, err := client.GetInboxEmailsPaginated(tt.limit, tt.offset)

			if tt.wantErr {
				if err == nil {
					t.Error("MockClient.GetInboxEmailsPaginated() expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("MockClient.GetInboxEmailsPaginated() unexpected error = %v", err)
				}

				if len(emails) > tt.limit {
					t.Errorf("MockClient.GetInboxEmailsPaginated() returned %d emails, want at most %d",
						len(emails), tt.limit)
				}
			}
		})
	}
}

func TestMockClient_GetInboxEmailsWithCount(t *testing.T) {
	client := NewMockClient()

	info, err := client.GetInboxEmailsWithCount(10)
	if err != nil {
		t.Errorf("MockClient.GetInboxEmailsWithCount() unexpected error = %v", err)
	}

	if info == nil {
		t.Fatal("MockClient.GetInboxEmailsWithCount() returned nil")
	}

	if len(info.Emails) > 10 {
		t.Errorf("MockClient.GetInboxEmailsWithCount() returned %d emails, want at most 10",
			len(info.Emails))
	}

	if info.TotalCount <= 0 {
		t.Error("MockClient.GetInboxEmailsWithCount() TotalCount should be positive")
	}

	if info.TotalCount < len(info.Emails) {
		t.Errorf("MockClient.GetInboxEmailsWithCount() TotalCount = %d, but returned %d emails",
			info.TotalCount, len(info.Emails))
	}
}

func TestMockClient_GetInboxEmailsWithCountPaginated(t *testing.T) {
	client := NewMockClient()

	// Get first page
	info1, err := client.GetInboxEmailsWithCountPaginated(5, 0)
	if err != nil {
		t.Fatalf("MockClient.GetInboxEmailsWithCountPaginated() first page error = %v", err)
	}

	// Get second page
	info2, err := client.GetInboxEmailsWithCountPaginated(5, 5)
	if err != nil {
		t.Fatalf("MockClient.GetInboxEmailsWithCountPaginated() second page error = %v", err)
	}

	// Total count should be the same on both pages
	if info1.TotalCount != info2.TotalCount {
		t.Errorf("MockClient.GetInboxEmailsWithCountPaginated() TotalCount inconsistent: %d vs %d",
			info1.TotalCount, info2.TotalCount)
	}

	// Email IDs should be different between pages
	if len(info1.Emails) > 0 && len(info2.Emails) > 0 {
		if info1.Emails[0].ID == info2.Emails[0].ID {
			t.Error("MockClient.GetInboxEmailsWithCountPaginated() pages should return different emails")
		}
	}
}

func TestMockClient_ArchiveEmails(t *testing.T) {
	tests := []struct {
		name     string
		emailIDs []string
		dryRun   bool
		wantErr  bool
	}{
		{
			name:     "dry run archive",
			emailIDs: []string{"email-0-0", "email-0-1"},
			dryRun:   true,
			wantErr:  false,
		},
		{
			name:     "real archive",
			emailIDs: []string{"email-1-0", "email-1-1"},
			dryRun:   false,
			wantErr:  false,
		},
		{
			name:     "archive empty list",
			emailIDs: []string{},
			dryRun:   false,
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := NewMockClient()
			initialCount := len(client.sampleEmails)

			// Count non-archived emails before
			nonArchivedBefore := 0
			for _, email := range client.sampleEmails {
				if !client.archivedIDs[email.ID] {
					nonArchivedBefore++
				}
			}

			err := client.ArchiveEmails(tt.emailIDs, tt.dryRun)

			if tt.wantErr {
				if err == nil {
					t.Error("MockClient.ArchiveEmails() expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("MockClient.ArchiveEmails() unexpected error = %v", err)
				}

				// Check that total emails hasn't changed
				if len(client.sampleEmails) != initialCount {
					t.Errorf("MockClient.ArchiveEmails() changed total emails: %d -> %d",
						initialCount, len(client.sampleEmails))
				}

				// In dry run mode, emails should not be archived
				if tt.dryRun {
					for _, id := range tt.emailIDs {
						if client.archivedIDs[id] {
							t.Errorf("MockClient.ArchiveEmails() in dry run mode but email %s was archived", id)
						}
					}
				} else {
					// In real mode, emails should be marked as archived
					for _, id := range tt.emailIDs {
						if !client.archivedIDs[id] {
							t.Errorf("MockClient.ArchiveEmails() email %s should be archived", id)
						}
					}
				}
			}
		})
	}
}

func TestMockClient_ArchiveAndRetrieve(t *testing.T) {
	client := NewMockClient()

	// Get initial inbox count
	initialEmails, err := client.GetInboxEmails(100)
	if err != nil {
		t.Fatalf("Failed to get initial emails: %v", err)
	}
	initialCount := len(initialEmails)

	if initialCount < 2 {
		t.Fatal("Need at least 2 sample emails for this test")
	}

	// Archive some emails
	emailsToArchive := []string{initialEmails[0].ID, initialEmails[1].ID}
	err = client.ArchiveEmails(emailsToArchive, false)
	if err != nil {
		t.Fatalf("Failed to archive emails: %v", err)
	}

	// Get inbox emails again
	afterEmails, err := client.GetInboxEmails(100)
	if err != nil {
		t.Fatalf("Failed to get emails after archiving: %v", err)
	}

	// Should have fewer emails in inbox
	if len(afterEmails) != initialCount-2 {
		t.Errorf("After archiving 2 emails, inbox has %d emails, want %d",
			len(afterEmails), initialCount-2)
	}

	// Archived emails should not be in inbox
	for _, archivedID := range emailsToArchive {
		for _, email := range afterEmails {
			if email.ID == archivedID {
				t.Errorf("Archived email %s is still in inbox", archivedID)
			}
		}
	}
}

func TestMockClient_GenerateSampleEmails(t *testing.T) {
	client := NewMockClient()

	if len(client.sampleEmails) == 0 {
		t.Fatal("generateSampleEmails() should create emails")
	}

	// Check that emails have required fields
	for i, email := range client.sampleEmails {
		if email.ID == "" {
			t.Errorf("Email %d has empty ID", i)
		}
		if email.Subject == "" {
			t.Errorf("Email %d has empty Subject", i)
		}
		if len(email.From) == 0 {
			t.Errorf("Email %d has no From address", i)
		}
		if email.ReceivedAt.IsZero() {
			t.Errorf("Email %d has zero ReceivedAt time", i)
		}
	}

	// Check that we have emails from the same sender
	// (which indicates similar email groups)
	senderCount := make(map[string]int)
	for _, email := range client.sampleEmails {
		if len(email.From) > 0 {
			senderCount[email.From[0].Email]++
		}
	}

	// Should have at least one sender with multiple emails
	foundGroup := false
	for _, count := range senderCount {
		if count > 1 {
			foundGroup = true
			break
		}
	}

	if !foundGroup {
		t.Error("generateSampleEmails() should create groups of similar emails from same senders")
	}
}
