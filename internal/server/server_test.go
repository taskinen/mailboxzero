package server

import (
	"bytes"
	"encoding/json"
	"errors"
	"github.com/taskinen/mailboxzero/internal/config"
	"github.com/taskinen/mailboxzero/internal/jmap"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
)

// setupTestServer creates a test server with mock JMAP client
func setupTestServer(t *testing.T) *Server {
	t.Helper()

	// Create a minimal config
	cfg := &config.Config{
		Server: struct {
			Port int    `yaml:"port"`
			Host string `yaml:"host"`
		}{
			Port: 8080,
			Host: "localhost",
		},
		DryRun:            true,
		DefaultSimilarity: 75,
		MockMode:          true,
	}

	// Use mock JMAP client
	mockClient := jmap.NewMockClient()

	// Create temporary template for testing
	templateContent := `<!DOCTYPE html>
<html>
<head><title>Test</title></head>
<body>
<h1>Mailbox Zero</h1>
{{if .DryRun}}<div>DRY RUN MODE</div>{{end}}
</body>
</html>`

	// Create temp directory with proper web/templates structure
	tmpDir := t.TempDir()
	templatePath := tmpDir + "/web/templates"
	if err := os.MkdirAll(templatePath, 0755); err != nil {
		t.Fatalf("Failed to create template directory: %v", err)
	}

	templateFile := templatePath + "/index.html"
	if err := os.WriteFile(templateFile, []byte(templateContent), 0644); err != nil {
		t.Fatalf("Failed to write template file: %v", err)
	}

	// Change working directory temporarily
	oldWd, _ := os.Getwd()
	os.Chdir(tmpDir)
	t.Cleanup(func() { os.Chdir(oldWd) })

	server, err := New(cfg, mockClient)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	return server
}

func TestNew(t *testing.T) {
	cfg := &config.Config{
		Server: struct {
			Port int    `yaml:"port"`
			Host string `yaml:"host"`
		}{
			Port: 8080,
			Host: "localhost",
		},
		DryRun:            true,
		DefaultSimilarity: 75,
	}

	mockClient := jmap.NewMockClient()

	// Create temporary template with proper structure
	tmpDir := t.TempDir()
	os.MkdirAll(tmpDir+"/web/templates", 0755)
	os.WriteFile(tmpDir+"/web/templates/index.html", []byte("<html></html>"), 0644)

	oldWd, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(oldWd)

	server, err := New(cfg, mockClient)
	if err != nil {
		t.Errorf("New() unexpected error = %v", err)
	}

	if server == nil {
		t.Fatal("New() returned nil server")
	}

	if server.config != cfg {
		t.Error("New() did not set config correctly")
	}

	if server.jmapClient != mockClient {
		t.Error("New() did not set jmapClient correctly")
	}

	if server.templates == nil {
		t.Error("New() did not load templates")
	}
}

func TestNew_TemplateError(t *testing.T) {
	cfg := &config.Config{}
	mockClient := jmap.NewMockClient()

	// Don't create any template files
	tmpDir := t.TempDir()
	oldWd, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(oldWd)

	_, err := New(cfg, mockClient)
	if err == nil {
		t.Error("New() expected error for missing templates but got none")
	}
}

func TestHandleGetEmails(t *testing.T) {
	server := setupTestServer(t)

	tests := []struct {
		name           string
		query          string
		wantStatusCode int
	}{
		{
			name:           "get emails without query params",
			query:          "",
			wantStatusCode: http.StatusOK,
		},
		{
			name:           "get emails with limit",
			query:          "?limit=10",
			wantStatusCode: http.StatusOK,
		},
		{
			name:           "get emails with offset",
			query:          "?offset=5",
			wantStatusCode: http.StatusOK,
		},
		{
			name:           "get emails with limit and offset",
			query:          "?limit=10&offset=5",
			wantStatusCode: http.StatusOK,
		},
		{
			name:           "get emails with invalid limit",
			query:          "?limit=invalid",
			wantStatusCode: http.StatusOK, // Should use default
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/api/emails"+tt.query, nil)
			w := httptest.NewRecorder()

			server.handleGetEmails(w, req)

			if w.Code != tt.wantStatusCode {
				t.Errorf("handleGetEmails() status = %v, want %v", w.Code, tt.wantStatusCode)
			}

			if tt.wantStatusCode == http.StatusOK {
				var response jmap.InboxInfo
				if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
					t.Errorf("handleGetEmails() failed to decode response: %v", err)
				}

				if response.Emails == nil {
					t.Error("handleGetEmails() response.Emails is nil")
				}

				if response.TotalCount < 0 {
					t.Errorf("handleGetEmails() response.TotalCount = %v, want >= 0", response.TotalCount)
				}
			}
		})
	}
}

func TestHandleFindSimilar(t *testing.T) {
	server := setupTestServer(t)

	// Get some emails first to use their IDs
	mockClient := server.jmapClient.(*jmap.MockClient)
	emails, _ := mockClient.GetInboxEmails(10)

	tests := []struct {
		name           string
		requestBody    interface{}
		wantStatusCode int
	}{
		{
			name: "find similar without specific email",
			requestBody: SimilarRequest{
				SimilarityThreshold: 75.0,
			},
			wantStatusCode: http.StatusOK,
		},
		{
			name: "find similar with specific email",
			requestBody: SimilarRequest{
				EmailID:             emails[0].ID,
				SimilarityThreshold: 75.0,
			},
			wantStatusCode: http.StatusOK,
		},
		{
			name: "find similar with nonexistent email",
			requestBody: SimilarRequest{
				EmailID:             "nonexistent-id",
				SimilarityThreshold: 75.0,
			},
			wantStatusCode: http.StatusNotFound,
		},
		{
			name: "find similar with low threshold",
			requestBody: SimilarRequest{
				SimilarityThreshold: 0.0,
			},
			wantStatusCode: http.StatusOK,
		},
		{
			name: "find similar with high threshold",
			requestBody: SimilarRequest{
				SimilarityThreshold: 99.0,
			},
			wantStatusCode: http.StatusOK,
		},
		{
			name:           "invalid request body",
			requestBody:    "invalid json",
			wantStatusCode: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var body []byte
			var err error

			if str, ok := tt.requestBody.(string); ok {
				body = []byte(str)
			} else {
				body, err = json.Marshal(tt.requestBody)
				if err != nil {
					t.Fatalf("Failed to marshal request: %v", err)
				}
			}

			req := httptest.NewRequest("POST", "/api/similar", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			server.handleFindSimilar(w, req)

			if w.Code != tt.wantStatusCode {
				t.Errorf("handleFindSimilar() status = %v, want %v", w.Code, tt.wantStatusCode)
			}

			if tt.wantStatusCode == http.StatusOK {
				var response []jmap.Email
				if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
					t.Errorf("handleFindSimilar() failed to decode response: %v", err)
				}
			}
		})
	}
}

// fixedInboxJMAPClient serves a fixed inbox. A nil emails slice mirrors the
// real JMAP client, which returns nil rather than an empty slice when the
// inbox has no emails.
type fixedInboxJMAPClient struct {
	jmap.JMAPClient
	emails []jmap.Email
}

func (c *fixedInboxJMAPClient) GetInboxEmails(limit int) ([]jmap.Email, error) {
	return c.emails, nil
}

func (c *fixedInboxJMAPClient) GetInboxEmailsWithCountPaginated(limit, offset int) (*jmap.InboxInfo, error) {
	return &jmap.InboxInfo{Emails: c.emails, TotalCount: len(c.emails)}, nil
}

func TestHandleGetEmails_EmptyInboxIsEmptyArray(t *testing.T) {
	server := &Server{
		config:     &config.Config{},
		jmapClient: &fixedInboxJMAPClient{JMAPClient: jmap.NewMockClient()},
	}

	req := httptest.NewRequest("GET", "/api/emails", nil)
	w := httptest.NewRecorder()
	server.handleGetEmails(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("handleGetEmails() status = %v, want %v", w.Code, http.StatusOK)
	}

	var response map[string]json.RawMessage
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("handleGetEmails() failed to decode response: %v", err)
	}
	if got := string(response["emails"]); got != "[]" {
		t.Errorf("handleGetEmails() emails = %s, want []", got)
	}
}

func TestHandleFindSimilar_NoMatchesIsEmptyArray(t *testing.T) {
	tests := []struct {
		name   string
		emails []jmap.Email
	}{
		{
			name:   "empty inbox",
			emails: nil,
		},
		{
			name: "no similar emails",
			emails: []jmap.Email{
				{
					ID:      "tax",
					Subject: "Quarterly tax filing reminder",
					From:    []jmap.EmailAddress{{Name: "Tax Office", Email: "filing@tax.example"}},
					Preview: "Your filing is due next month.",
				},
				{
					ID:      "photos",
					Subject: "Photos from the lake trip",
					From:    []jmap.EmailAddress{{Name: "Alex", Email: "alex@friends.example"}},
					Preview: "Here are the pictures I promised.",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := &Server{
				config:     &config.Config{},
				jmapClient: &fixedInboxJMAPClient{JMAPClient: jmap.NewMockClient(), emails: tt.emails},
			}

			body, _ := json.Marshal(SimilarRequest{SimilarityThreshold: 60.0})
			req := httptest.NewRequest("POST", "/api/similar", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			server.handleFindSimilar(w, req)

			if w.Code != http.StatusOK {
				t.Fatalf("handleFindSimilar() status = %v, want %v", w.Code, http.StatusOK)
			}
			if got := strings.TrimSpace(w.Body.String()); got != "[]" {
				t.Errorf("handleFindSimilar() body = %s, want []", got)
			}
		})
	}
}

func TestHandleArchive(t *testing.T) {
	server := setupTestServer(t)

	// Get some emails first to use their IDs
	mockClient := server.jmapClient.(*jmap.MockClient)
	emails, _ := mockClient.GetInboxEmails(10)

	tests := []struct {
		name           string
		requestBody    interface{}
		wantStatusCode int
	}{
		{
			name: "archive single email",
			requestBody: ArchiveRequest{
				EmailIDs: []string{emails[0].ID},
			},
			wantStatusCode: http.StatusOK,
		},
		{
			name: "archive multiple emails",
			requestBody: ArchiveRequest{
				EmailIDs: []string{emails[1].ID, emails[2].ID},
			},
			wantStatusCode: http.StatusOK,
		},
		{
			name: "archive empty list",
			requestBody: ArchiveRequest{
				EmailIDs: []string{},
			},
			wantStatusCode: http.StatusBadRequest,
		},
		{
			name:           "invalid request body",
			requestBody:    "invalid json",
			wantStatusCode: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var body []byte
			var err error

			if str, ok := tt.requestBody.(string); ok {
				body = []byte(str)
			} else {
				body, err = json.Marshal(tt.requestBody)
				if err != nil {
					t.Fatalf("Failed to marshal request: %v", err)
				}
			}

			req := httptest.NewRequest("POST", "/api/archive", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			server.handleArchive(w, req)

			if w.Code != tt.wantStatusCode {
				t.Errorf("handleArchive() status = %v, want %v", w.Code, tt.wantStatusCode)
			}

			if tt.wantStatusCode == http.StatusOK {
				var response map[string]interface{}
				if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
					t.Errorf("handleArchive() failed to decode response: %v", err)
				}

				if success, ok := response["success"].(bool); !ok || !success {
					t.Error("handleArchive() response should have success=true")
				}

				if dryRun, ok := response["dryRun"].(bool); !ok {
					t.Error("handleArchive() response should have dryRun field")
				} else if dryRun != server.config.DryRun {
					t.Errorf("handleArchive() dryRun = %v, want %v", dryRun, server.config.DryRun)
				}
			}
		})
	}
}

func TestHandleUnarchive(t *testing.T) {
	server := setupTestServer(t)

	mockClient := server.jmapClient.(*jmap.MockClient)
	emails, _ := mockClient.GetInboxEmails(10)

	tests := []struct {
		name           string
		requestBody    interface{}
		wantStatusCode int
	}{
		{
			name: "unarchive single email",
			requestBody: ArchiveRequest{
				EmailIDs: []string{emails[0].ID},
			},
			wantStatusCode: http.StatusOK,
		},
		{
			name: "unarchive multiple emails",
			requestBody: ArchiveRequest{
				EmailIDs: []string{emails[1].ID, emails[2].ID},
			},
			wantStatusCode: http.StatusOK,
		},
		{
			name: "unarchive empty list",
			requestBody: ArchiveRequest{
				EmailIDs: []string{},
			},
			wantStatusCode: http.StatusBadRequest,
		},
		{
			name:           "invalid request body",
			requestBody:    "invalid json",
			wantStatusCode: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var body []byte
			var err error

			if str, ok := tt.requestBody.(string); ok {
				body = []byte(str)
			} else {
				body, err = json.Marshal(tt.requestBody)
				if err != nil {
					t.Fatalf("Failed to marshal request: %v", err)
				}
			}

			req := httptest.NewRequest("POST", "/api/unarchive", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			server.handleUnarchive(w, req)

			if w.Code != tt.wantStatusCode {
				t.Errorf("handleUnarchive() status = %v, want %v", w.Code, tt.wantStatusCode)
			}

			if tt.wantStatusCode == http.StatusOK {
				var response map[string]interface{}
				if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
					t.Errorf("handleUnarchive() failed to decode response: %v", err)
				}

				if success, ok := response["success"].(bool); !ok || !success {
					t.Error("handleUnarchive() response should have success=true")
				}

				if dryRun, ok := response["dryRun"].(bool); !ok {
					t.Error("handleUnarchive() response should have dryRun field")
				} else if dryRun != server.config.DryRun {
					t.Errorf("handleUnarchive() dryRun = %v, want %v", dryRun, server.config.DryRun)
				}
			}
		})
	}
}

func TestHandleUnarchive_ErrorConditions(t *testing.T) {
	server := setupTestServer(t)

	tests := []struct {
		name           string
		requestBody    string
		wantStatusCode int
	}{
		{
			name:           "malformed JSON",
			requestBody:    "{invalid json",
			wantStatusCode: http.StatusBadRequest,
		},
		{
			name:           "null emailIds",
			requestBody:    `{"emailIds": null}`,
			wantStatusCode: http.StatusBadRequest,
		},
		{
			name:           "missing emailIds field",
			requestBody:    `{}`,
			wantStatusCode: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("POST", "/api/unarchive", strings.NewReader(tt.requestBody))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			server.handleUnarchive(w, req)

			if w.Code != tt.wantStatusCode {
				t.Errorf("handleUnarchive() status = %v, want %v", w.Code, tt.wantStatusCode)
			}
		})
	}
}

func TestHandleClear(t *testing.T) {
	server := setupTestServer(t)

	req := httptest.NewRequest("POST", "/api/clear", nil)
	w := httptest.NewRecorder()

	server.handleClear(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("handleClear() status = %v, want %v", w.Code, http.StatusOK)
	}

	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Errorf("handleClear() failed to decode response: %v", err)
	}

	if success, ok := response["success"].(bool); !ok || !success {
		t.Error("handleClear() response should have success=true")
	}
}

func TestHandleIndex(t *testing.T) {
	server := setupTestServer(t)

	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()

	server.handleIndex(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("handleIndex() status = %v, want %v", w.Code, http.StatusOK)
	}

	body := w.Body.String()
	if !strings.Contains(body, "Mailbox Zero") {
		t.Error("handleIndex() response should contain 'Mailbox Zero'")
	}

	// Check that DryRun mode indicator is present when enabled
	if server.config.DryRun && !strings.Contains(body, "DRY RUN MODE") {
		t.Error("handleIndex() response should indicate DRY RUN MODE when enabled")
	}
}

func TestPageData(t *testing.T) {
	data := PageData{
		DryRun:            true,
		DefaultSimilarity: 75,
	}

	if !data.DryRun {
		t.Error("PageData.DryRun not set correctly")
	}
	if data.DefaultSimilarity != 75 {
		t.Errorf("PageData.DefaultSimilarity = %v, want 75", data.DefaultSimilarity)
	}
}

func TestSimilarRequest(t *testing.T) {
	req := SimilarRequest{
		EmailID:             "test-email-id",
		SimilarityThreshold: 85.5,
	}

	if req.EmailID != "test-email-id" {
		t.Errorf("SimilarRequest.EmailID = %v, want 'test-email-id'", req.EmailID)
	}
	if req.SimilarityThreshold != 85.5 {
		t.Errorf("SimilarRequest.SimilarityThreshold = %v, want 85.5", req.SimilarityThreshold)
	}
}

func TestArchiveRequest(t *testing.T) {
	req := ArchiveRequest{
		EmailIDs: []string{"id1", "id2", "id3"},
	}

	if len(req.EmailIDs) != 3 {
		t.Errorf("ArchiveRequest.EmailIDs length = %v, want 3", len(req.EmailIDs))
	}
}

func TestHandleGetEmails_ParseErrors(t *testing.T) {
	server := setupTestServer(t)

	tests := []struct {
		name           string
		query          string
		wantStatusCode int
	}{
		{
			name:           "negative limit",
			query:          "?limit=-5",
			wantStatusCode: http.StatusOK, // Should use default
		},
		{
			name:           "zero limit",
			query:          "?limit=0",
			wantStatusCode: http.StatusOK, // Should use default
		},
		{
			name:           "negative offset",
			query:          "?offset=-5",
			wantStatusCode: http.StatusOK, // Should use default (0)
		},
		{
			name:           "very large limit",
			query:          "?limit=99999",
			wantStatusCode: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/api/emails"+tt.query, nil)
			w := httptest.NewRecorder()

			server.handleGetEmails(w, req)

			if w.Code != tt.wantStatusCode {
				t.Errorf("handleGetEmails() status = %v, want %v", w.Code, tt.wantStatusCode)
			}
		})
	}
}

// limitRecordingJMAPClient records the page size handleGetEmails asks for.
type limitRecordingJMAPClient struct {
	jmap.JMAPClient
	gotLimit int
}

func (c *limitRecordingJMAPClient) GetInboxEmailsWithCountPaginated(limit, offset int) (*jmap.InboxInfo, error) {
	c.gotLimit = limit
	return &jmap.InboxInfo{Emails: []jmap.Email{}}, nil
}

func TestHandleGetEmails_LimitIsCapped(t *testing.T) {
	tests := []struct {
		name      string
		query     string
		wantLimit int
	}{
		{name: "default", query: "", wantLimit: 100},
		{name: "small page", query: "?limit=50", wantLimit: 50},
		{name: "largest UI page", query: "?limit=500", wantLimit: 500},
		{name: "just over the cap", query: "?limit=501", wantLimit: 500},
		{name: "huge", query: "?limit=100000", wantLimit: 500},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &limitRecordingJMAPClient{JMAPClient: jmap.NewMockClient()}
			server := &Server{config: &config.Config{}, jmapClient: client}

			req := httptest.NewRequest("GET", "/api/emails"+tt.query, nil)
			w := httptest.NewRecorder()
			server.handleGetEmails(w, req)

			if w.Code != http.StatusOK {
				t.Fatalf("handleGetEmails() status = %v, want %v", w.Code, http.StatusOK)
			}
			if client.gotLimit != tt.wantLimit {
				t.Errorf("handleGetEmails() requested limit %d, want %d", client.gotLimit, tt.wantLimit)
			}
		})
	}
}

func TestHandleIndex_WithPageData(t *testing.T) {
	server := setupTestServer(t)

	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()

	server.handleIndex(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("handleIndex() status = %v, want %v", w.Code, http.StatusOK)
	}

	body := w.Body.String()

	// Check that the template was rendered with the page data
	if server.config.DryRun && !strings.Contains(body, "DRY RUN MODE") {
		t.Error("handleIndex() should render DryRun indicator when enabled")
	}

	// Verify it's HTML
	if !strings.Contains(body, "<html>") || !strings.Contains(body, "</html>") {
		t.Error("handleIndex() should return HTML content")
	}
}

func TestHandleFindSimilar_ErrorConditions(t *testing.T) {
	server := setupTestServer(t)

	tests := []struct {
		name           string
		requestBody    string
		contentType    string
		wantStatusCode int
	}{
		{
			name:           "malformed JSON",
			requestBody:    "{invalid json",
			contentType:    "application/json",
			wantStatusCode: http.StatusBadRequest,
		},
		{
			name:           "empty body",
			requestBody:    "",
			contentType:    "application/json",
			wantStatusCode: http.StatusBadRequest,
		},
		{
			name:           "wrong content type",
			requestBody:    `{"similarityThreshold": 75}`,
			contentType:    "text/plain",
			wantStatusCode: http.StatusOK, // JSON decoder doesn't check content-type
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("POST", "/api/similar", strings.NewReader(tt.requestBody))
			if tt.contentType != "" {
				req.Header.Set("Content-Type", tt.contentType)
			}
			w := httptest.NewRecorder()

			server.handleFindSimilar(w, req)

			if w.Code != tt.wantStatusCode {
				t.Errorf("handleFindSimilar() status = %v, want %v", w.Code, tt.wantStatusCode)
			}
		})
	}
}

func TestHandleArchive_ErrorConditions(t *testing.T) {
	server := setupTestServer(t)

	tests := []struct {
		name           string
		requestBody    string
		wantStatusCode int
	}{
		{
			name:           "malformed JSON",
			requestBody:    "{invalid json",
			wantStatusCode: http.StatusBadRequest,
		},
		{
			name:           "null emailIds",
			requestBody:    `{"emailIds": null}`,
			wantStatusCode: http.StatusBadRequest,
		},
		{
			name:           "missing emailIds field",
			requestBody:    `{}`,
			wantStatusCode: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("POST", "/api/archive", strings.NewReader(tt.requestBody))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			server.handleArchive(w, req)

			if w.Code != tt.wantStatusCode {
				t.Errorf("handleArchive() status = %v, want %v", w.Code, tt.wantStatusCode)
			}
		})
	}
}

func TestHandleFindSimilar_BoundaryValues(t *testing.T) {
	server := setupTestServer(t)

	tests := []struct {
		name      string
		threshold float64
	}{
		{
			name:      "zero threshold",
			threshold: 0.0,
		},
		{
			name:      "max threshold",
			threshold: 100.0,
		},
		{
			name:      "mid threshold",
			threshold: 50.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reqBody := SimilarRequest{
				SimilarityThreshold: tt.threshold,
			}

			body, _ := json.Marshal(reqBody)
			req := httptest.NewRequest("POST", "/api/similar", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			server.handleFindSimilar(w, req)

			if w.Code != http.StatusOK {
				t.Errorf("handleFindSimilar() with threshold %v status = %v, want %v",
					tt.threshold, w.Code, http.StatusOK)
			}
		})
	}
}

func TestHandleGetEmails_JSONEncoding(t *testing.T) {
	server := setupTestServer(t)

	req := httptest.NewRequest("GET", "/api/emails?limit=5", nil)
	w := httptest.NewRecorder()

	server.handleGetEmails(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("handleGetEmails() status = %v, want %v", w.Code, http.StatusOK)
	}

	contentType := w.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("handleGetEmails() Content-Type = %v, want application/json", contentType)
	}

	// Verify response is valid JSON
	var response jmap.InboxInfo
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Errorf("handleGetEmails() returned invalid JSON: %v", err)
	}
}

func TestHandleClear_JSONResponse(t *testing.T) {
	server := setupTestServer(t)

	req := httptest.NewRequest("POST", "/api/clear", nil)
	w := httptest.NewRecorder()

	server.handleClear(w, req)

	contentType := w.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("handleClear() Content-Type = %v, want application/json", contentType)
	}
}

func TestServer_ConfigValues(t *testing.T) {
	server := setupTestServer(t)

	if !server.config.DryRun {
		t.Error("Test server should have DryRun enabled")
	}

	if server.config.DefaultSimilarity != 75 {
		t.Errorf("Test server DefaultSimilarity = %v, want 75", server.config.DefaultSimilarity)
	}
}

// countingJMAPClient wraps a JMAPClient and counts how many times the
// full 1000-email inbox fetch used by handleFindSimilar is invoked.
type countingJMAPClient struct {
	jmap.JMAPClient
	getInboxCalls int
}

func (c *countingJMAPClient) GetInboxEmails(limit int) ([]jmap.Email, error) {
	c.getInboxCalls++
	return c.JMAPClient.GetInboxEmails(limit)
}

func setupCountingTestServer(t *testing.T) (*Server, *countingJMAPClient) {
	t.Helper()

	cfg := &config.Config{
		Server: struct {
			Port int    `yaml:"port"`
			Host string `yaml:"host"`
		}{Port: 8080, Host: "localhost"},
		DryRun:            true,
		DefaultSimilarity: 75,
		MockMode:          true,
	}

	counter := &countingJMAPClient{JMAPClient: jmap.NewMockClient()}

	tmpDir := t.TempDir()
	templatePath := tmpDir + "/web/templates"
	if err := os.MkdirAll(templatePath, 0755); err != nil {
		t.Fatalf("Failed to create template directory: %v", err)
	}
	if err := os.WriteFile(templatePath+"/index.html", []byte("<html></html>"), 0644); err != nil {
		t.Fatalf("Failed to write template file: %v", err)
	}

	oldWd, _ := os.Getwd()
	os.Chdir(tmpDir)
	t.Cleanup(func() { os.Chdir(oldWd) })

	server, err := New(cfg, counter)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	return server, counter
}

func TestHandleFindSimilar_CachesInbox(t *testing.T) {
	server, counter := setupCountingTestServer(t)

	body, _ := json.Marshal(SimilarRequest{SimilarityThreshold: 75.0})
	for i := 0; i < 3; i++ {
		req := httptest.NewRequest("POST", "/api/similar", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		server.handleFindSimilar(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("call %d: status = %v, want 200", i, w.Code)
		}
	}

	if counter.getInboxCalls != 1 {
		t.Errorf("expected 1 inbox fetch across 3 calls, got %d", counter.getInboxCalls)
	}
}

func TestHandleArchive_InvalidatesInboxCache(t *testing.T) {
	server, counter := setupCountingTestServer(t)

	body, _ := json.Marshal(SimilarRequest{SimilarityThreshold: 75.0})
	req := httptest.NewRequest("POST", "/api/similar", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	server.handleFindSimilar(httptest.NewRecorder(), req)

	if counter.getInboxCalls != 1 {
		t.Fatalf("setup: expected 1 inbox fetch, got %d", counter.getInboxCalls)
	}

	// Archive an arbitrary id — under dry-run this is a no-op against the mock
	// but the handler still invalidates the cache.
	archiveBody, _ := json.Marshal(ArchiveRequest{EmailIDs: []string{"email-0-0"}})
	archiveReq := httptest.NewRequest("POST", "/api/archive", bytes.NewReader(archiveBody))
	archiveReq.Header.Set("Content-Type", "application/json")
	archiveW := httptest.NewRecorder()
	server.handleArchive(archiveW, archiveReq)
	if archiveW.Code != http.StatusOK {
		t.Fatalf("archive status = %v, want 200", archiveW.Code)
	}

	// Next /api/similar should re-fetch.
	req2 := httptest.NewRequest("POST", "/api/similar", bytes.NewReader(body))
	req2.Header.Set("Content-Type", "application/json")
	server.handleFindSimilar(httptest.NewRecorder(), req2)

	if counter.getInboxCalls != 2 {
		t.Errorf("expected 2 inbox fetches after archive, got %d", counter.getInboxCalls)
	}
}

func TestGetInboxForSimilarity_TTLExpiration(t *testing.T) {
	server, counter := setupCountingTestServer(t)

	body, _ := json.Marshal(SimilarRequest{SimilarityThreshold: 75.0})
	req := httptest.NewRequest("POST", "/api/similar", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	server.handleFindSimilar(httptest.NewRecorder(), req)

	if counter.getInboxCalls != 1 {
		t.Fatalf("setup: expected 1 inbox fetch, got %d", counter.getInboxCalls)
	}

	// Backdate the cache timestamp past the TTL window so the next call
	// must re-fetch instead of serving stale data indefinitely.
	server.inboxMu.Lock()
	server.inboxCachedAt = time.Now().Add(-2 * inboxCacheTTL)
	server.inboxMu.Unlock()

	req2 := httptest.NewRequest("POST", "/api/similar", bytes.NewReader(body))
	req2.Header.Set("Content-Type", "application/json")
	server.handleFindSimilar(httptest.NewRecorder(), req2)

	if counter.getInboxCalls != 2 {
		t.Errorf("expected 2 inbox fetches after TTL expiration, got %d", counter.getInboxCalls)
	}
}

// erroringJMAPClient always fails on GetInboxEmails; the embedded JMAPClient
// supplies stub implementations for every other interface method via the mock.
type erroringJMAPClient struct {
	jmap.JMAPClient
	calls int
	err   error
}

func (c *erroringJMAPClient) GetInboxEmails(limit int) ([]jmap.Email, error) {
	c.calls++
	return nil, c.err
}

func TestGetInboxForSimilarity_ErrorNotCached(t *testing.T) {
	cfg := &config.Config{
		Server: struct {
			Port int    `yaml:"port"`
			Host string `yaml:"host"`
		}{Port: 8080, Host: "localhost"},
		DryRun:            true,
		DefaultSimilarity: 75,
		MockMode:          true,
	}

	failing := &erroringJMAPClient{
		JMAPClient: jmap.NewMockClient(),
		err:        errors.New("jmap transient failure"),
	}

	tmpDir := t.TempDir()
	templatePath := tmpDir + "/web/templates"
	if err := os.MkdirAll(templatePath, 0755); err != nil {
		t.Fatalf("Failed to create template directory: %v", err)
	}
	if err := os.WriteFile(templatePath+"/index.html", []byte("<html></html>"), 0644); err != nil {
		t.Fatalf("Failed to write template file: %v", err)
	}

	oldWd, _ := os.Getwd()
	os.Chdir(tmpDir)
	t.Cleanup(func() { os.Chdir(oldWd) })

	server, err := New(cfg, failing)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	body, _ := json.Marshal(SimilarRequest{SimilarityThreshold: 75.0})

	// Both calls must propagate the JMAP error (HTTP 500) AND make their own
	// underlying fetch. Caching a nil/empty result on error would poison the
	// cache for the full TTL window after a single transient failure.
	for i := 0; i < 2; i++ {
		req := httptest.NewRequest("POST", "/api/similar", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		server.handleFindSimilar(w, req)
		if w.Code != http.StatusInternalServerError {
			t.Errorf("call %d: status = %v, want 500", i, w.Code)
		}
	}

	if failing.calls != 2 {
		t.Errorf("expected 2 underlying fetches across 2 failing /api/similar calls, got %d", failing.calls)
	}
}

func TestHandleUnarchive_InvalidatesInboxCache(t *testing.T) {
	server, counter := setupCountingTestServer(t)

	body, _ := json.Marshal(SimilarRequest{SimilarityThreshold: 75.0})
	req := httptest.NewRequest("POST", "/api/similar", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	server.handleFindSimilar(httptest.NewRecorder(), req)

	unarchiveBody, _ := json.Marshal(ArchiveRequest{EmailIDs: []string{"email-0-0"}})
	unarchiveReq := httptest.NewRequest("POST", "/api/unarchive", bytes.NewReader(unarchiveBody))
	unarchiveReq.Header.Set("Content-Type", "application/json")
	server.handleUnarchive(httptest.NewRecorder(), unarchiveReq)

	req2 := httptest.NewRequest("POST", "/api/similar", bytes.NewReader(body))
	req2.Header.Set("Content-Type", "application/json")
	server.handleFindSimilar(httptest.NewRecorder(), req2)

	if counter.getInboxCalls != 2 {
		t.Errorf("expected 2 inbox fetches after unarchive, got %d", counter.getInboxCalls)
	}
}

// failingMoveJMAPClient fails every archive and unarchive request.
type failingMoveJMAPClient struct {
	jmap.JMAPClient
	err error
}

func (c *failingMoveJMAPClient) ArchiveEmails(emailIDs []string, dryRun bool) error {
	return c.err
}

func (c *failingMoveJMAPClient) UnarchiveEmails(emailIDs []string, dryRun bool) error {
	return c.err
}

// A failed archive or unarchive may still have moved some of the emails
// (JMAP rejects emails individually), so the inbox cache must be dropped
// even when the handler returns an error.
func TestHandleMove_FailureInvalidatesInboxCache(t *testing.T) {
	tests := []struct {
		name     string
		handle   func(*Server, http.ResponseWriter, *http.Request)
		path     string
		wantBody string
	}{
		{
			name:     "archive",
			handle:   (*Server).handleArchive,
			path:     "/api/archive",
			wantBody: "Failed to archive emails: server rejected 1 email(s): e2 (notFound)",
		},
		{
			name:     "unarchive",
			handle:   (*Server).handleUnarchive,
			path:     "/api/unarchive",
			wantBody: "Failed to unarchive emails: server rejected 1 email(s): e2 (notFound)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := &Server{
				config: &config.Config{},
				jmapClient: &failingMoveJMAPClient{
					JMAPClient: jmap.NewMockClient(),
					err:        errors.New("server rejected 1 email(s): e2 (notFound)"),
				},
				inboxCache:    []jmap.Email{{ID: "e1"}, {ID: "e2"}},
				inboxCachedAt: time.Now(),
			}

			body, _ := json.Marshal(ArchiveRequest{EmailIDs: []string{"e1", "e2"}})
			req := httptest.NewRequest("POST", tt.path, bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			tt.handle(server, w, req)

			if w.Code != http.StatusInternalServerError {
				t.Errorf("status = %v, want %v", w.Code, http.StatusInternalServerError)
			}
			if got := strings.TrimSpace(w.Body.String()); got != tt.wantBody {
				t.Errorf("body = %q, want %q", got, tt.wantBody)
			}
			if server.inboxCache != nil {
				t.Error("inbox cache kept after a failed move; emails the server did move would still be listed")
			}
		})
	}
}

// TestHandler_Routes sends real HTTP requests through Handler(), the same
// handler Start() serves, to check routing and method matching end to end.
func TestHandler_Routes(t *testing.T) {
	server := setupTestServer(t)
	ts := httptest.NewServer(server.Handler())
	defer ts.Close()

	tests := []struct {
		method string
		path   string
		want   int
	}{
		{method: "GET", path: "/", want: http.StatusOK},
		{method: "GET", path: "/api/emails", want: http.StatusOK},
		{method: "POST", path: "/api/clear", want: http.StatusOK},
		{method: "GET", path: "/api/clear", want: http.StatusMethodNotAllowed},
		{method: "GET", path: "/no-such-page", want: http.StatusNotFound},
	}

	for _, tt := range tests {
		t.Run(tt.method+" "+tt.path, func(t *testing.T) {
			req, err := http.NewRequest(tt.method, ts.URL+tt.path, nil)
			if err != nil {
				t.Fatalf("failed to build request: %v", err)
			}
			resp, err := ts.Client().Do(req)
			if err != nil {
				t.Fatalf("request failed: %v", err)
			}
			resp.Body.Close()
			if resp.StatusCode != tt.want {
				t.Errorf("status = %d, want %d", resp.StatusCode, tt.want)
			}
		})
	}
}

func TestNewHTTPServer_Timeouts(t *testing.T) {
	handler := http.NewServeMux()
	srv := newHTTPServer("localhost:8080", handler)

	if srv.Addr != "localhost:8080" {
		t.Errorf("Addr = %q, want localhost:8080", srv.Addr)
	}
	if srv.Handler != handler {
		t.Error("Handler is not the handler passed in")
	}

	timeouts := []struct {
		name string
		got  time.Duration
		want time.Duration
	}{
		{name: "ReadHeaderTimeout", got: srv.ReadHeaderTimeout, want: readHeaderTimeout},
		{name: "ReadTimeout", got: srv.ReadTimeout, want: readTimeout},
		{name: "WriteTimeout", got: srv.WriteTimeout, want: writeTimeout},
		{name: "IdleTimeout", got: srv.IdleTimeout, want: idleTimeout},
	}
	for _, tt := range timeouts {
		if tt.got <= 0 || tt.got != tt.want {
			t.Errorf("%s = %v, want %v", tt.name, tt.got, tt.want)
		}
	}
}
