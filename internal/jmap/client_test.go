package jmap

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
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

// jsonOf returns v as compact JSON, for comparing request structures.
func jsonOf(v interface{}) string {
	b, _ := json.Marshal(v)
	return string(b)
}

// The inbox and archive ids are looked up once. After that, an inbox page is
// one request (that also asks for the inbox count), and an archive or undo is
// just the Email/set.
func TestClient_RemembersMailboxIDs(t *testing.T) {
	fake, c := newFakeJMAPServer(t, map[string]jmapMethodHandler{
		"Mailbox/get": mailboxGet(testInbox, testArchive, testLabel),
		"Email/query": func(map[string]interface{}) map[string]interface{} {
			return map[string]interface{}{"accountId": "acc", "ids": []string{"e1"}}
		},
		"Email/get": func(map[string]interface{}) map[string]interface{} {
			return map[string]interface{}{"accountId": "acc", "list": []interface{}{map[string]interface{}{"id": "e1"}}}
		},
		"Email/set": emailSetAllUpdated,
	})

	for i := 0; i < 2; i++ {
		info, err := c.GetInboxEmailsWithCountPaginated(10, 0)
		if err != nil {
			t.Fatalf("GetInboxEmailsWithCountPaginated() unexpected error = %v", err)
		}
		if info.TotalCount != 42 || len(info.Emails) != 1 {
			t.Errorf("page %d = %d emails of %d, want 1 of 42", i+1, len(info.Emails), info.TotalCount)
		}
	}
	if err := c.ArchiveEmails([]string{"e1"}, false); err != nil {
		t.Fatalf("ArchiveEmails() unexpected error = %v", err)
	}
	if err := c.UnarchiveEmails([]string{"e1"}, false); err != nil {
		t.Fatalf("UnarchiveEmails() unexpected error = %v", err)
	}

	page := `["Mailbox/get","Email/query","Email/get"]`
	want := `[["Mailbox/get"],` + page + `,` + page + `,["Email/set"],["Email/set"]]`
	if got := jsonOf(fake.requestMethods()); got != want {
		t.Errorf("JMAP requests = %s, want %s", got, want)
	}

	// Only the lookup lists every mailbox; the pages ask for the inbox count.
	gets := fake.callArgs("Mailbox/get")
	if _, ok := gets[0]["ids"]; ok {
		t.Errorf("lookup Mailbox/get = %s, want every mailbox", jsonOf(gets[0]))
	}
	for _, args := range gets[1:] {
		if jsonOf(args["ids"]) != `["mb-inbox"]` || jsonOf(args["properties"]) != `["totalEmails"]` {
			t.Errorf("page Mailbox/get = %s, want the inbox's totalEmails", jsonOf(args))
		}
	}
}

// A failed archive may come from a mailbox that was deleted or replaced, so
// the next call looks the ids up again.
func TestClient_LooksUpMailboxIDsAgainAfterAFailure(t *testing.T) {
	var reject atomic.Bool
	reject.Store(true)
	fake, c := newFakeJMAPServer(t, map[string]jmapMethodHandler{
		"Mailbox/get": mailboxGet(testInbox, testArchive),
		"Email/set": func(args map[string]interface{}) map[string]interface{} {
			if reject.Load() {
				return map[string]interface{}{"accountId": "acc", "notUpdated": map[string]interface{}{
					"e1": map[string]interface{}{"type": "invalidProperties"},
				}}
			}
			return emailSetAllUpdated(args)
		},
	})

	if err := c.ArchiveEmails([]string{"e1"}, false); err == nil {
		t.Fatal("ArchiveEmails() succeeded, want the rejection")
	}
	reject.Store(false)
	for i := 0; i < 2; i++ {
		if err := c.ArchiveEmails([]string{"e1"}, false); err != nil {
			t.Fatalf("ArchiveEmails() unexpected error = %v", err)
		}
	}

	want := `[["Mailbox/get"],["Email/set"],["Mailbox/get"],["Email/set"],["Email/set"]]`
	if got := jsonOf(fake.requestMethods()); got != want {
		t.Errorf("JMAP requests = %s, want %s", got, want)
	}
}

// Without an archive mailbox nothing is remembered, so an archive created
// later is found.
func TestClient_FindsArchiveCreatedLater(t *testing.T) {
	var hasArchive atomic.Bool
	fake, c := newFakeJMAPServer(t, map[string]jmapMethodHandler{
		"Mailbox/get": func(args map[string]interface{}) map[string]interface{} {
			if hasArchive.Load() {
				return mailboxGet(testInbox, testArchive)(args)
			}
			return mailboxGet(testInbox)(args)
		},
		"Email/set": emailSetAllUpdated,
	})

	if err := c.ArchiveEmails([]string{"e1"}, false); err == nil || err.Error() != "archive folder not found" {
		t.Fatalf("ArchiveEmails() error = %v, want archive folder not found", err)
	}
	hasArchive.Store(true)
	if err := c.ArchiveEmails([]string{"e1"}, false); err != nil {
		t.Fatalf("ArchiveEmails() unexpected error = %v", err)
	}

	want := `[["Mailbox/get"],["Mailbox/get"],["Email/set"]]`
	if got := jsonOf(fake.requestMethods()); got != want {
		t.Errorf("JMAP requests = %s, want %s", got, want)
	}
}

// The HTTP server calls the client from concurrent handlers; run with -race.
func TestClient_MailboxIDsConcurrentUse(t *testing.T) {
	fake, c := newFakeJMAPServer(t, map[string]jmapMethodHandler{
		"Mailbox/get": mailboxGet(testInbox, testArchive),
		"Email/set":   emailSetAllUpdated,
	})

	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := c.ArchiveEmails([]string{"e1"}, false); err != nil {
				t.Errorf("ArchiveEmails() unexpected error = %v", err)
			}
		}()
	}
	wg.Wait()

	if lookups := len(fake.callArgs("Mailbox/get")); lookups != 1 {
		t.Errorf("mailboxes looked up %d times, want once", lookups)
	}
}
