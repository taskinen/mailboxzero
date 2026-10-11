package jmap

import (
	"encoding/json"
	"testing"
	"time"
)

func TestGetString(t *testing.T) {
	tests := []struct {
		name string
		data map[string]interface{}
		key  string
		want string
	}{
		{
			name: "string value exists",
			data: map[string]interface{}{"key": "value"},
			key:  "key",
			want: "value",
		},
		{
			name: "key doesn't exist",
			data: map[string]interface{}{"other": "value"},
			key:  "key",
			want: "",
		},
		{
			name: "value is not a string",
			data: map[string]interface{}{"key": 123},
			key:  "key",
			want: "",
		},
		{
			name: "empty map",
			data: map[string]interface{}{},
			key:  "key",
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getString(tt.data, tt.key)
			if got != tt.want {
				t.Errorf("getString() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetInt(t *testing.T) {
	tests := []struct {
		name string
		data map[string]interface{}
		key  string
		want int
	}{
		{
			name: "float64 value",
			data: map[string]interface{}{"key": float64(123)},
			key:  "key",
			want: 123,
		},
		{
			name: "int value",
			data: map[string]interface{}{"key": 123},
			key:  "key",
			want: 123,
		},
		{
			name: "key doesn't exist",
			data: map[string]interface{}{"other": 123},
			key:  "key",
			want: 0,
		},
		{
			name: "value is not a number",
			data: map[string]interface{}{"key": "string"},
			key:  "key",
			want: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getInt(tt.data, tt.key)
			if got != tt.want {
				t.Errorf("getInt() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetBool(t *testing.T) {
	tests := []struct {
		name string
		data map[string]interface{}
		key  string
		want bool
	}{
		{
			name: "bool true value",
			data: map[string]interface{}{"key": true},
			key:  "key",
			want: true,
		},
		{
			name: "bool false value",
			data: map[string]interface{}{"key": false},
			key:  "key",
			want: false,
		},
		{
			name: "key doesn't exist",
			data: map[string]interface{}{"other": true},
			key:  "key",
			want: false,
		},
		{
			name: "value is not a bool",
			data: map[string]interface{}{"key": "true"},
			key:  "key",
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getBool(tt.data, tt.key)
			if got != tt.want {
				t.Errorf("getBool() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseEmail(t *testing.T) {
	tests := []struct {
		name string
		data map[string]interface{}
		want Email
	}{
		{
			name: "basic email data",
			data: map[string]interface{}{
				"id":      "test-id-123",
				"subject": "Test Subject",
				"preview": "Test preview text",
			},
			want: Email{
				ID:      "test-id-123",
				Subject: "Test Subject",
				Preview: "Test preview text",
			},
		},
		{
			name: "email with from address",
			data: map[string]interface{}{
				"id":      "test-id-456",
				"subject": "Test Subject",
				"from": []interface{}{
					map[string]interface{}{
						"name":  "Test User",
						"email": "test@example.com",
					},
				},
			},
			want: Email{
				ID:      "test-id-456",
				Subject: "Test Subject",
				From: []EmailAddress{
					{Name: "Test User", Email: "test@example.com"},
				},
			},
		},
		{
			name: "email with receivedAt",
			data: map[string]interface{}{
				"id":         "test-id-789",
				"receivedAt": "2023-01-01T12:00:00Z",
			},
			want: Email{
				ID:         "test-id-789",
				ReceivedAt: time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC),
			},
		},
		{
			name: "email with body values",
			data: map[string]interface{}{
				"id": "test-id-body",
				"bodyValues": map[string]interface{}{
					"text": map[string]interface{}{
						"value":             "Body text content",
						"isEncodingProblem": false,
						"isTruncated":       true,
					},
				},
			},
			want: Email{
				ID: "test-id-body",
				BodyValues: map[string]BodyValue{
					"text": {
						Value:             "Body text content",
						IsEncodingProblem: false,
						IsTruncated:       true,
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseEmail(tt.data)

			if got.ID != tt.want.ID {
				t.Errorf("parseEmail().ID = %v, want %v", got.ID, tt.want.ID)
			}
			if got.Subject != tt.want.Subject {
				t.Errorf("parseEmail().Subject = %v, want %v", got.Subject, tt.want.Subject)
			}
			if got.Preview != tt.want.Preview {
				t.Errorf("parseEmail().Preview = %v, want %v", got.Preview, tt.want.Preview)
			}
			if len(tt.want.From) > 0 {
				if len(got.From) != len(tt.want.From) {
					t.Errorf("parseEmail().From length = %v, want %v", len(got.From), len(tt.want.From))
				} else {
					if got.From[0].Email != tt.want.From[0].Email {
						t.Errorf("parseEmail().From[0].Email = %v, want %v", got.From[0].Email, tt.want.From[0].Email)
					}
				}
			}
		})
	}
}

func TestNewClient(t *testing.T) {
	endpoint := "https://api.example.com/jmap/session"
	apiToken := "test-token"

	client := NewClient(endpoint, apiToken)

	if client == nil {
		t.Fatal("NewClient() returned nil")
	}
	if client.endpoint != endpoint {
		t.Errorf("NewClient().endpoint = %v, want %v", client.endpoint, endpoint)
	}
	if client.apiToken != apiToken {
		t.Errorf("NewClient().apiToken = %v, want %v", client.apiToken, apiToken)
	}
	if client.httpClient == nil {
		t.Error("NewClient().httpClient is nil")
	}
	if client.session != nil {
		t.Error("NewClient().session should be nil before authentication")
	}
}

func TestClient_GetPrimaryAccount(t *testing.T) {
	tests := []struct {
		name    string
		session *Session
		want    string
	}{
		{
			name: "valid session with mail account",
			session: &Session{
				PrimaryAccounts: map[string]string{
					"urn:ietf:params:jmap:mail": "account-123",
				},
			},
			want: "account-123",
		},
		{
			name: "session without mail account",
			session: &Session{
				PrimaryAccounts: map[string]string{
					"urn:ietf:params:jmap:contacts": "account-456",
				},
			},
			want: "",
		},
		{
			name:    "nil session",
			session: nil,
			want:    "",
		},
		{
			name: "empty primary accounts",
			session: &Session{
				PrimaryAccounts: map[string]string{},
			},
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &Client{session: tt.session}
			got := client.GetPrimaryAccount()
			if got != tt.want {
				t.Errorf("GetPrimaryAccount() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestExtractNameFromEmail(t *testing.T) {
	tests := []struct {
		name  string
		email string
		want  string
	}{
		{
			name:  "github email",
			email: "notifications@github.com",
			want:  "GitHub",
		},
		{
			name:  "stripe email",
			email: "support@stripe.com",
			want:  "Stripe Support",
		},
		{
			name:  "unknown email",
			email: "unknown@example.com",
			want:  "unknown@example.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractNameFromEmail(tt.email)
			if got != tt.want {
				t.Errorf("extractNameFromEmail() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestClient_MakeRequest_Unauthenticated(t *testing.T) {
	client := NewClient("https://api.example.com/jmap/session", "test-token")

	// Try to make a request without authenticating first
	methodCalls := []MethodCall{
		{"Email/get", map[string]interface{}{"accountId": "test"}, "0"},
	}

	_, err := client.makeRequest(methodCalls)
	if err == nil {
		t.Error("makeRequest() should fail when client not authenticated")
	}
	if err.Error() != "client not authenticated" {
		t.Errorf("makeRequest() error = %v, want 'client not authenticated'", err)
	}
}

// TestClient_InboxPage_SingleMailboxLookup checks that loading an inbox page
// looks the mailboxes up only once: one HTTP request for Mailbox/get and one
// for Email/query plus Email/get.
func TestClient_InboxPage_SingleMailboxLookup(t *testing.T) {
	tests := []struct {
		name      string
		fetch     func(c *Client) ([]Email, int, error)
		wantTotal int
	}{
		{
			name: "GetInboxEmailsWithCountPaginated",
			fetch: func(c *Client) ([]Email, int, error) {
				info, err := c.GetInboxEmailsWithCountPaginated(2, 40)
				if err != nil {
					return nil, 0, err
				}
				return info.Emails, info.TotalCount, nil
			},
			wantTotal: 42,
		},
		{
			name: "GetInboxEmailsPaginated",
			fetch: func(c *Client) ([]Email, int, error) {
				emails, err := c.GetInboxEmailsPaginated(2, 40)
				return emails, 0, err
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fake, c := newFakeJMAPServer(t, map[string]jmapMethodHandler{
				"Mailbox/get": mailboxGet(testInbox, testArchive),
				"Email/query": func(map[string]interface{}) map[string]interface{} {
					return map[string]interface{}{"accountId": "acc", "ids": []string{"e1", "e2"}}
				},
				"Email/get": func(map[string]interface{}) map[string]interface{} {
					return map[string]interface{}{"accountId": "acc", "list": []interface{}{
						map[string]interface{}{
							"id":         "e1",
							"subject":    "First",
							"from":       []interface{}{map[string]interface{}{"name": "Alice", "email": "alice@example.com"}},
							"receivedAt": "2026-10-01T09:00:00Z",
							"preview":    "Hello",
						},
						map[string]interface{}{"id": "e2", "subject": "Second"},
					}}
				},
			})

			emails, total, err := tt.fetch(c)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			gotJSON, _ := json.Marshal(fake.requestMethods())
			if want := `[["Mailbox/get"],["Email/query","Email/get"]]`; string(gotJSON) != want {
				t.Errorf("JMAP requests = %s, want %s", gotJSON, want)
			}

			queries := fake.callArgs("Email/query")
			if len(queries) != 1 {
				t.Fatalf("got %d Email/query calls, want 1", len(queries))
			}
			args := queries[0]
			filter, _ := json.Marshal(args["filter"])
			if string(filter) != `{"inMailbox":"mb-inbox"}` {
				t.Errorf("Email/query filter = %s, want inbox filter", filter)
			}
			if args["limit"] != float64(2) || args["position"] != float64(40) {
				t.Errorf("Email/query limit/position = %v/%v, want 2/40", args["limit"], args["position"])
			}

			if total != tt.wantTotal {
				t.Errorf("total = %d, want %d", total, tt.wantTotal)
			}
			if len(emails) != 2 {
				t.Fatalf("got %d emails, want 2", len(emails))
			}
			if emails[0].ID != "e1" || emails[0].Subject != "First" || emails[0].From[0].Email != "alice@example.com" {
				t.Errorf("first email = %+v, want e1 from alice@example.com", emails[0])
			}
		})
	}
}

func TestClient_GetInboxEmails(t *testing.T) {
	// Test with mock client
	mockClient := NewMockClient()
	emails, err := mockClient.GetInboxEmails(10)

	if err != nil {
		t.Errorf("GetInboxEmails() unexpected error = %v", err)
	}

	if len(emails) > 10 {
		t.Errorf("GetInboxEmails() returned %d emails, want at most 10", len(emails))
	}
}

func TestClient_GetInboxEmailsWithCount(t *testing.T) {
	// Test with mock client
	mockClient := NewMockClient()
	info, err := mockClient.GetInboxEmailsWithCount(5)

	if err != nil {
		t.Errorf("GetInboxEmailsWithCount() unexpected error = %v", err)
	}

	if info == nil {
		t.Fatal("GetInboxEmailsWithCount() returned nil")
	}

	if len(info.Emails) > 5 {
		t.Errorf("GetInboxEmailsWithCount() returned %d emails, want at most 5", len(info.Emails))
	}

	if info.TotalCount < len(info.Emails) {
		t.Errorf("GetInboxEmailsWithCount() TotalCount = %d, but returned %d emails",
			info.TotalCount, len(info.Emails))
	}
}

func TestClient_ArchiveEmails_DryRun(t *testing.T) {
	mockClient := NewMockClient()

	// Test dry run
	err := mockClient.ArchiveEmails([]string{"email-0-0"}, true)
	if err != nil {
		t.Errorf("ArchiveEmails() dry run unexpected error = %v", err)
	}

	// Verify email wasn't actually archived
	emails, _ := mockClient.GetInboxEmails(100)
	found := false
	for _, email := range emails {
		if email.ID == "email-0-0" {
			found = true
			break
		}
	}
	if !found {
		t.Error("ArchiveEmails() in dry run mode should not actually archive emails")
	}
}

func TestClient_ArchiveEmails_Real(t *testing.T) {
	mockClient := NewMockClient()

	// Get initial count
	initialInfo, _ := mockClient.GetInboxEmailsWithCount(100)
	initialCount := initialInfo.TotalCount

	// Archive an email
	err := mockClient.ArchiveEmails([]string{"email-0-0"}, false)
	if err != nil {
		t.Errorf("ArchiveEmails() unexpected error = %v", err)
	}

	// Verify email was archived
	afterInfo, _ := mockClient.GetInboxEmailsWithCount(100)
	if afterInfo.TotalCount != initialCount-1 {
		t.Errorf("ArchiveEmails() inbox count = %d, want %d",
			afterInfo.TotalCount, initialCount-1)
	}
}

// TestClient_MoveEmails_PatchesOnlyInboxAndArchive checks that archiving and
// unarchiving patch just the inbox and archive memberships, so any other
// mailbox the email belongs to (a Fastmail label) is left untouched.
func TestClient_MoveEmails_PatchesOnlyInboxAndArchive(t *testing.T) {
	tests := []struct {
		name      string
		mailboxes []map[string]interface{}
		move      func(c *Client, ids []string) error
		wantPatch string
		wantErr   string
	}{
		{
			name:      "archive",
			mailboxes: []map[string]interface{}{testInbox, testArchive, testLabel},
			move:      func(c *Client, ids []string) error { return c.ArchiveEmails(ids, false) },
			wantPatch: `{"mailboxIds/mb-archive":true,"mailboxIds/mb-inbox":null}`,
		},
		{
			name:      "unarchive",
			mailboxes: []map[string]interface{}{testInbox, testArchive, testLabel},
			move:      func(c *Client, ids []string) error { return c.UnarchiveEmails(ids, false) },
			wantPatch: `{"mailboxIds/mb-archive":null,"mailboxIds/mb-inbox":true}`,
		},
		{
			name:      "unarchive without archive mailbox",
			mailboxes: []map[string]interface{}{testInbox, testLabel},
			move:      func(c *Client, ids []string) error { return c.UnarchiveEmails(ids, false) },
			wantErr:   "archive folder not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fake, c := newFakeJMAPServer(t, map[string]jmapMethodHandler{
				"Mailbox/get": mailboxGet(tt.mailboxes...),
				"Email/set":   emailSetAllUpdated,
			})

			err := tt.move(c, []string{"e1", "e2"})
			sets := fake.callArgs("Email/set")
			if tt.wantErr != "" {
				if err == nil || err.Error() != tt.wantErr {
					t.Fatalf("error = %v, want %q", err, tt.wantErr)
				}
				if len(sets) != 0 {
					t.Error("Email/set was sent even though the move could not be made")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(sets) != 1 {
				t.Fatalf("got %d Email/set calls, want 1", len(sets))
			}

			update, _ := sets[0]["update"].(map[string]interface{})
			if len(update) != 2 {
				t.Fatalf("Email/set updated %d emails, want 2", len(update))
			}
			for _, id := range []string{"e1", "e2"} {
				got, _ := json.Marshal(update[id])
				if string(got) != tt.wantPatch {
					t.Errorf("patch for %s = %s, want %s", id, got, tt.wantPatch)
				}
			}
		})
	}
}

func TestParseEmail_ComplexStructures(t *testing.T) {
	data := map[string]interface{}{
		"id":      "complex-email",
		"subject": "Test Subject",
		"textBody": []interface{}{
			map[string]interface{}{
				"partId": "text-part-1",
				"type":   "text/plain",
			},
			map[string]interface{}{
				"partId": "text-part-2",
				"type":   "text/plain",
			},
		},
		"htmlBody": []interface{}{
			map[string]interface{}{
				"partId": "html-part-1",
				"type":   "text/html",
			},
		},
		"bodyValues": map[string]interface{}{
			"text-part-1": map[string]interface{}{
				"value":             "First text part",
				"isEncodingProblem": false,
				"isTruncated":       false,
			},
			"text-part-2": map[string]interface{}{
				"value":             "Second text part",
				"isEncodingProblem": true,
				"isTruncated":       true,
			},
		},
	}

	email := parseEmail(data)

	if email.ID != "complex-email" {
		t.Errorf("parseEmail().ID = %v, want 'complex-email'", email.ID)
	}

	if len(email.TextBody) != 2 {
		t.Errorf("parseEmail() TextBody length = %d, want 2", len(email.TextBody))
	}

	if len(email.HTMLBody) != 1 {
		t.Errorf("parseEmail() HTMLBody length = %d, want 1", len(email.HTMLBody))
	}

	if len(email.BodyValues) != 2 {
		t.Errorf("parseEmail() BodyValues length = %d, want 2", len(email.BodyValues))
	}

	// Check specific body value
	if bodyVal, ok := email.BodyValues["text-part-2"]; ok {
		if bodyVal.Value != "Second text part" {
			t.Errorf("parseEmail() BodyValue.Value = %v, want 'Second text part'", bodyVal.Value)
		}
		if !bodyVal.IsEncodingProblem {
			t.Error("parseEmail() BodyValue.IsEncodingProblem should be true")
		}
		if !bodyVal.IsTruncated {
			t.Error("parseEmail() BodyValue.IsTruncated should be true")
		}
	} else {
		t.Error("parseEmail() should have body value for 'text-part-2'")
	}
}

func TestParseEmail_MissingFields(t *testing.T) {
	// Test with minimal data
	data := map[string]interface{}{
		"id": "minimal-email",
	}

	email := parseEmail(data)

	if email.ID != "minimal-email" {
		t.Errorf("parseEmail().ID = %v, want 'minimal-email'", email.ID)
	}
	if email.Subject != "" {
		t.Errorf("parseEmail().Subject = %v, want empty string", email.Subject)
	}
	if len(email.From) != 0 {
		t.Errorf("parseEmail().From length = %d, want 0", len(email.From))
	}
	if email.ReceivedAt.IsZero() {
		// This is expected for missing receivedAt
	}
}

func TestParseEmail_InvalidReceivedAt(t *testing.T) {
	data := map[string]interface{}{
		"id":         "test-id",
		"receivedAt": "invalid-date-format",
	}

	email := parseEmail(data)

	// Should handle invalid date gracefully
	if !email.ReceivedAt.IsZero() {
		t.Error("parseEmail() should have zero time for invalid receivedAt")
	}
}

func TestInboxInfo(t *testing.T) {
	info := &InboxInfo{
		Emails: []Email{
			{ID: "1", Subject: "Test 1"},
			{ID: "2", Subject: "Test 2"},
		},
		TotalCount: 10,
	}

	if len(info.Emails) != 2 {
		t.Errorf("InboxInfo.Emails length = %d, want 2", len(info.Emails))
	}
	if info.TotalCount != 10 {
		t.Errorf("InboxInfo.TotalCount = %d, want 10", info.TotalCount)
	}
}

func TestCheckSetResponse(t *testing.T) {
	setResponse := func(name string, args map[string]interface{}) *Response {
		return &Response{MethodResponses: [][]interface{}{{name, args, "0"}}}
	}

	tests := []struct {
		name    string
		resp    *Response
		wantErr string
	}{
		{
			name: "all updated",
			resp: setResponse("Email/set", map[string]interface{}{
				"updated":    map[string]interface{}{"e1": nil, "e2": nil},
				"notUpdated": nil,
			}),
		},
		{
			name: "empty notUpdated",
			resp: setResponse("Email/set", map[string]interface{}{
				"updated":    map[string]interface{}{"e1": nil},
				"notUpdated": map[string]interface{}{},
			}),
		},
		{
			name: "some emails rejected",
			resp: setResponse("Email/set", map[string]interface{}{
				"updated": map[string]interface{}{"e2": nil},
				"notUpdated": map[string]interface{}{
					"e3": map[string]interface{}{"type": "forbidden"},
					"e1": map[string]interface{}{"type": "notFound", "description": "no such email"},
				},
			}),
			wantErr: "server rejected 2 email(s): e1 (notFound: no such email), e3 (forbidden)",
		},
		{
			name:    "rejection without type",
			resp:    setResponse("Email/set", map[string]interface{}{"notUpdated": map[string]interface{}{"e1": nil}}),
			wantErr: "server rejected 1 email(s): e1 (unknown error)",
		},
		{
			name: "method error",
			resp: setResponse("error", map[string]interface{}{
				"type":        "invalidArguments",
				"description": "bad patch",
			}),
			wantErr: "server rejected Email/set: invalidArguments: bad patch",
		},
		{
			name:    "unexpected method response",
			resp:    setResponse("Mailbox/get", map[string]interface{}{}),
			wantErr: `unexpected response to Email/set: "Mailbox/get"`,
		},
		{
			name:    "no method responses",
			resp:    &Response{},
			wantErr: "invalid Email/set response format",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := checkSetResponse(tt.resp)
			if tt.wantErr == "" {
				if err != nil {
					t.Errorf("checkSetResponse() unexpected error = %v", err)
				}
				return
			}
			if err == nil || err.Error() != tt.wantErr {
				t.Errorf("checkSetResponse() error = %v, want %q", err, tt.wantErr)
			}
		})
	}
}

// TestClient_MoveEmails_ReportsRejectedEmails checks that archive and
// unarchive report emails the server refused to move instead of succeeding.
func TestClient_MoveEmails_ReportsRejectedEmails(t *testing.T) {
	_, c := newFakeJMAPServer(t, map[string]jmapMethodHandler{
		"Mailbox/get": mailboxGet(testInbox, testArchive),
		"Email/set": func(map[string]interface{}) map[string]interface{} {
			return map[string]interface{}{
				"accountId": "acc",
				"updated":   map[string]interface{}{"e1": nil},
				"notUpdated": map[string]interface{}{
					"e2": map[string]interface{}{"type": "notFound", "description": "no such email"},
				},
			}
		},
	})

	tests := []struct {
		name    string
		move    func(ids []string) error
		wantErr string
	}{
		{
			name:    "archive",
			move:    func(ids []string) error { return c.ArchiveEmails(ids, false) },
			wantErr: "server rejected 1 email(s): e2 (notFound: no such email)",
		},
		{
			name:    "unarchive",
			move:    func(ids []string) error { return c.UnarchiveEmails(ids, false) },
			wantErr: "server rejected 1 email(s): e2 (notFound: no such email)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.move([]string{"e1", "e2"})
			if err == nil || err.Error() != tt.wantErr {
				t.Errorf("error = %v, want %q", err, tt.wantErr)
			}
		})
	}
}
