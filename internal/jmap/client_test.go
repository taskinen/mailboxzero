package jmap

import (
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
)

// TestClient_Authenticate logs in against the fake server's session resource
// and checks that the session it returns drives the following API calls.
func TestClient_Authenticate(t *testing.T) {
	fake, _ := newFakeJMAPServer(t, map[string]jmapMethodHandler{
		"Mailbox/get": mailboxGet(testInbox),
	})

	c := NewClient(fake.sessionURL(), fakeAPIToken)
	if err := c.Authenticate(); err != nil {
		t.Fatalf("Authenticate() unexpected error = %v", err)
	}

	if c.session.APIUrl != fake.url {
		t.Errorf("session APIUrl = %q, want %q", c.session.APIUrl, fake.url)
	}
	if c.session.Username != "test@example.com" {
		t.Errorf("session Username = %q, want test@example.com", c.session.Username)
	}
	if got := c.GetPrimaryAccount(); got != "acc" {
		t.Errorf("GetPrimaryAccount() = %q, want acc", got)
	}

	// The session's API URL and account are used for the next call.
	mailboxes, err := c.GetMailboxes()
	if err != nil {
		t.Fatalf("GetMailboxes() after Authenticate() unexpected error = %v", err)
	}
	if len(mailboxes) != 1 || mailboxes[0].ID != "mb-inbox" {
		t.Errorf("GetMailboxes() = %+v, want the inbox", mailboxes)
	}
	if args := fake.callArgs("Mailbox/get"); len(args) != 1 || args[0]["accountId"] != "acc" {
		t.Errorf("Mailbox/get args = %v, want accountId acc", args)
	}

	// Both the session request and the API request carry the token.
	auth := fake.authHeaders()
	if len(auth) != 2 {
		t.Fatalf("got %d HTTP requests, want 2 (session + API)", len(auth))
	}
	for i, header := range auth {
		if header != "Bearer "+fakeAPIToken {
			t.Errorf("request %d Authorization = %q, want Bearer %s", i, header, fakeAPIToken)
		}
	}
}

func TestClient_Authenticate_Errors(t *testing.T) {
	t.Run("rejected token", func(t *testing.T) {
		fake, _ := newFakeJMAPServer(t, nil)
		c := NewClient(fake.sessionURL(), "wrong-token")

		err := c.Authenticate()
		if err == nil || !strings.HasPrefix(err.Error(), "authentication failed: 401 - unauthorized") {
			t.Errorf("Authenticate() error = %v, want authentication failed: 401", err)
		}
		if c.GetPrimaryAccount() != "" {
			t.Error("a failed Authenticate() left a session behind")
		}
	})

	t.Run("invalid session document", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("not json"))
		}))
		defer srv.Close()

		err := NewClient(srv.URL, fakeAPIToken).Authenticate()
		if err == nil || !strings.HasPrefix(err.Error(), "failed to decode session") {
			t.Errorf("Authenticate() error = %v, want failed to decode session", err)
		}
	})

	t.Run("unreachable server", func(t *testing.T) {
		srv := httptest.NewServer(http.NotFoundHandler())
		url := srv.URL
		srv.Close()

		err := NewClient(url, fakeAPIToken).Authenticate()
		if err == nil || !strings.HasPrefix(err.Error(), "failed to get session") {
			t.Errorf("Authenticate() error = %v, want failed to get session", err)
		}
	})
}

func TestClient_GetMailboxes(t *testing.T) {
	inbox := map[string]interface{}{"id": "mb-inbox", "name": "Inbox", "role": "inbox", "totalEmails": 42, "unreadEmails": 5}
	_, c := newFakeJMAPServer(t, map[string]jmapMethodHandler{
		"Mailbox/get": mailboxGet(inbox, testArchive, testLabel),
	})

	mailboxes, err := c.GetMailboxes()
	if err != nil {
		t.Fatalf("GetMailboxes() unexpected error = %v", err)
	}

	want := []Mailbox{
		{ID: "mb-inbox", Name: "Inbox", Role: "inbox", TotalEmails: 42, UnreadEmails: 5},
		{ID: "mb-archive", Name: "Archive", Role: "archive", TotalEmails: 7},
		{ID: "mb-label", Name: "Receipts"}, // a label has no role
	}
	if !reflect.DeepEqual(mailboxes, want) {
		t.Errorf("GetMailboxes() = %+v, want %+v", mailboxes, want)
	}
}

func TestClient_GetMailboxes_Errors(t *testing.T) {
	t.Run("no primary account", func(t *testing.T) {
		c := &Client{session: &Session{}}
		_, err := c.GetMailboxes()
		if err == nil || err.Error() != "no primary account found" {
			t.Errorf("GetMailboxes() error = %v, want no primary account found", err)
		}
	})

	t.Run("response without a list", func(t *testing.T) {
		_, c := newFakeJMAPServer(t, map[string]jmapMethodHandler{
			"Mailbox/get": func(map[string]interface{}) map[string]interface{} {
				return map[string]interface{}{"accountId": "acc"}
			},
		})
		_, err := c.GetMailboxes()
		if err == nil || err.Error() != "invalid mailboxes data format" {
			t.Errorf("GetMailboxes() error = %v, want invalid mailboxes data format", err)
		}
	})

	t.Run("server error", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "boom", http.StatusInternalServerError)
		}))
		defer srv.Close()

		c := &Client{
			apiToken:   fakeAPIToken,
			httpClient: srv.Client(),
			session: &Session{
				APIUrl:          srv.URL,
				PrimaryAccounts: map[string]string{"urn:ietf:params:jmap:mail": "acc"},
			},
		}
		_, err := c.GetMailboxes()
		if err == nil || !strings.HasPrefix(err.Error(), "failed to get mailboxes: request failed: 500 - boom") {
			t.Errorf("GetMailboxes() error = %v, want request failed: 500", err)
		}
	})
}
