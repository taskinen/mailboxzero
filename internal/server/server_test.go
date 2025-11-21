package server

import (
	"bytes"
	"encoding/json"
	"mailboxzero/internal/config"
	"mailboxzero/internal/jmap"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
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
		Emails:            []jmap.Email{},
		GroupedEmails:     []jmap.Email{},
		SelectedEmailID:   "test-id",
	}

	if !data.DryRun {
		t.Error("PageData.DryRun not set correctly")
	}
	if data.DefaultSimilarity != 75 {
		t.Errorf("PageData.DefaultSimilarity = %v, want 75", data.DefaultSimilarity)
	}
	if data.SelectedEmailID != "test-id" {
		t.Errorf("PageData.SelectedEmailID = %v, want 'test-id'", data.SelectedEmailID)
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
