package jmap

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
)

// jmapMethodHandler answers one JMAP method call with its result object.
type jmapMethodHandler func(args map[string]interface{}) map[string]interface{}

// jmapCall is one method call the client sent to the fake server.
type jmapCall struct {
	name string
	args map[string]interface{}
}

// fakeJMAPServer speaks just enough JMAP over HTTP to test the real Client.
// Each test registers a handler per method name; the server records every
// request so tests can assert on what the client sent.
type fakeJMAPServer struct {
	t        *testing.T
	handlers map[string]jmapMethodHandler

	// mu guards requests, which the server's handler goroutine appends to
	// while the test reads it.
	mu       sync.Mutex
	requests [][]jmapCall // the method calls of each HTTP request, in order
}

// newFakeJMAPServer starts a fake JMAP server that answers the given methods
// and returns it with a Client already authenticated against it. Calls to any
// other method fail the test.
func newFakeJMAPServer(t *testing.T, handlers map[string]jmapMethodHandler) (*fakeJMAPServer, *Client) {
	t.Helper()

	f := &fakeJMAPServer{t: t, handlers: handlers}
	srv := httptest.NewServer(http.HandlerFunc(f.serveHTTP))
	t.Cleanup(srv.Close)

	client := &Client{
		apiToken:   "test-token",
		httpClient: srv.Client(),
		session: &Session{
			APIUrl:          srv.URL,
			PrimaryAccounts: map[string]string{"urn:ietf:params:jmap:mail": "acc"},
		},
	}
	return f, client
}

func (f *fakeJMAPServer) serveHTTP(w http.ResponseWriter, r *http.Request) {
	var req struct {
		MethodCalls [][]interface{} `json:"methodCalls"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		f.t.Errorf("failed to decode JMAP request: %v", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	calls := make([]jmapCall, 0, len(req.MethodCalls))
	responses := make([]interface{}, 0, len(req.MethodCalls))
	for _, call := range req.MethodCalls {
		name, _ := call[0].(string)
		args, _ := call[1].(map[string]interface{})
		calls = append(calls, jmapCall{name: name, args: args})

		handler, ok := f.handlers[name]
		if !ok {
			f.t.Errorf("unexpected JMAP method %s", name)
			responses = append(responses, []interface{}{"error", map[string]interface{}{"type": "unknownMethod"}, call[2]})
			continue
		}
		responses = append(responses, []interface{}{name, handler(args), call[2]})
	}

	f.mu.Lock()
	f.requests = append(f.requests, calls)
	f.mu.Unlock()

	json.NewEncoder(w).Encode(map[string]interface{}{"methodResponses": responses})
}

// requestMethods returns the method names of each HTTP request received so
// far, e.g. [["Mailbox/get"], ["Email/query", "Email/get"]].
func (f *fakeJMAPServer) requestMethods() [][]string {
	f.mu.Lock()
	defer f.mu.Unlock()

	methods := make([][]string, len(f.requests))
	for i, calls := range f.requests {
		for _, call := range calls {
			methods[i] = append(methods[i], call.name)
		}
	}
	return methods
}

// callArgs returns the arguments of every call to method, in the order the
// client sent them.
func (f *fakeJMAPServer) callArgs(method string) []map[string]interface{} {
	f.mu.Lock()
	defer f.mu.Unlock()

	var args []map[string]interface{}
	for _, calls := range f.requests {
		for _, call := range calls {
			if call.name == method {
				args = append(args, call.args)
			}
		}
	}
	return args
}

// Mailboxes for Mailbox/get responses.
var (
	testInbox   = map[string]interface{}{"id": "mb-inbox", "name": "Inbox", "role": "inbox", "totalEmails": 42}
	testArchive = map[string]interface{}{"id": "mb-archive", "name": "Archive", "role": "archive", "totalEmails": 7}
	testLabel   = map[string]interface{}{"id": "mb-label", "name": "Receipts"} // a Fastmail label: no role
)

// mailboxGet returns a Mailbox/get handler that lists the given mailboxes.
func mailboxGet(mailboxes ...map[string]interface{}) jmapMethodHandler {
	return func(map[string]interface{}) map[string]interface{} {
		return map[string]interface{}{"accountId": "acc", "list": mailboxes}
	}
}

// emailSetAllUpdated is an Email/set handler that accepts every update.
func emailSetAllUpdated(args map[string]interface{}) map[string]interface{} {
	update, _ := args["update"].(map[string]interface{})
	updated := make(map[string]interface{}, len(update))
	for id := range update {
		updated[id] = nil
	}
	return map[string]interface{}{"accountId": "acc", "updated": updated, "notUpdated": nil}
}
