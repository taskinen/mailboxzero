package server

import (
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/taskinen/mailboxzero/internal/config"
	"github.com/taskinen/mailboxzero/internal/jmap"
	"github.com/taskinen/mailboxzero/internal/similarity"

	"github.com/gorilla/mux"
)

const inboxCacheTTL = 60 * time.Second

type Server struct {
	config     *config.Config
	jmapClient jmap.JMAPClient
	templates  *template.Template

	inboxMu       sync.Mutex
	inboxCache    []jmap.Email
	inboxCachedAt time.Time
}

type PageData struct {
	DryRun            bool
	DefaultSimilarity int
	Emails            []jmap.Email
	GroupedEmails     []jmap.Email
	SelectedEmailID   string
}

func New(cfg *config.Config, jmapClient jmap.JMAPClient) (*Server, error) {
	templates, err := template.ParseGlob("web/templates/*.html")
	if err != nil {
		return nil, fmt.Errorf("failed to parse templates: %w", err)
	}

	return &Server{
		config:     cfg,
		jmapClient: jmapClient,
		templates:  templates,
	}, nil
}

func (s *Server) Start() error {
	r := mux.NewRouter()

	r.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(http.Dir("web/static/"))))

	r.HandleFunc("/", s.handleIndex).Methods("GET")
	r.HandleFunc("/api/emails", s.handleGetEmails).Methods("GET")
	r.HandleFunc("/api/similar", s.handleFindSimilar).Methods("POST")
	r.HandleFunc("/api/archive", s.handleArchive).Methods("POST")
	r.HandleFunc("/api/unarchive", s.handleUnarchive).Methods("POST")
	r.HandleFunc("/api/clear", s.handleClear).Methods("POST")

	addr := s.config.GetServerAddr()
	log.Printf("Server starting on http://%s", addr)
	log.Printf("DRY RUN MODE: %v", s.config.DryRun)

	return http.ListenAndServe(addr, r)
}

func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	data := PageData{
		DryRun:            s.config.DryRun,
		DefaultSimilarity: s.config.DefaultSimilarity,
	}

	if err := s.templates.ExecuteTemplate(w, "index.html", data); err != nil {
		http.Error(w, "Failed to render template", http.StatusInternalServerError)
		log.Printf("Template error: %v", err)
	}
}

func (s *Server) handleGetEmails(w http.ResponseWriter, r *http.Request) {
	limit := 100
	offset := 0

	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}

	if offsetStr := r.URL.Query().Get("offset"); offsetStr != "" {
		if o, err := strconv.Atoi(offsetStr); err == nil && o >= 0 {
			offset = o
		}
	}

	inboxInfo, err := s.jmapClient.GetInboxEmailsWithCountPaginated(limit, offset)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get emails: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(inboxInfo); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

type SimilarRequest struct {
	EmailID             string  `json:"emailId,omitempty"`
	SimilarityThreshold float64 `json:"similarityThreshold"`
}

func (s *Server) handleFindSimilar(w http.ResponseWriter, r *http.Request) {
	var req SimilarRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	emails, err := s.getInboxForSimilarity()
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get emails: %v", err), http.StatusInternalServerError)
		return
	}

	var similarEmails []jmap.Email
	if req.EmailID != "" {
		var targetEmail *jmap.Email
		for _, email := range emails {
			if email.ID == req.EmailID {
				targetEmail = &email
				break
			}
		}

		if targetEmail == nil {
			http.Error(w, "Target email not found", http.StatusNotFound)
			return
		}

		similarEmails = similarity.FindSimilarToEmail(*targetEmail, emails, req.SimilarityThreshold/100.0)
	} else {
		similarEmails = similarity.FindSimilarEmails(emails, req.SimilarityThreshold/100.0)
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(similarEmails); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

type ArchiveRequest struct {
	EmailIDs []string `json:"emailIds"`
}

func (s *Server) handleArchive(w http.ResponseWriter, r *http.Request) {
	var req ArchiveRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if len(req.EmailIDs) == 0 {
		http.Error(w, "No emails to archive", http.StatusBadRequest)
		return
	}

	if err := s.jmapClient.ArchiveEmails(req.EmailIDs, s.config.DryRun); err != nil {
		http.Error(w, fmt.Sprintf("Failed to archive emails: %v", err), http.StatusInternalServerError)
		return
	}
	s.invalidateInboxCache()

	response := map[string]interface{}{
		"success": true,
		"message": fmt.Sprintf("Successfully archived %d emails", len(req.EmailIDs)),
		"dryRun":  s.config.DryRun,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (s *Server) handleUnarchive(w http.ResponseWriter, r *http.Request) {
	var req ArchiveRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if len(req.EmailIDs) == 0 {
		http.Error(w, "No emails to unarchive", http.StatusBadRequest)
		return
	}

	if err := s.jmapClient.UnarchiveEmails(req.EmailIDs, s.config.DryRun); err != nil {
		http.Error(w, fmt.Sprintf("Failed to unarchive emails: %v", err), http.StatusInternalServerError)
		return
	}
	s.invalidateInboxCache()

	response := map[string]interface{}{
		"success": true,
		"message": fmt.Sprintf("Successfully unarchived %d emails", len(req.EmailIDs)),
		"dryRun":  s.config.DryRun,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (s *Server) handleClear(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"success": true})
}

// getInboxForSimilarity returns the cached inbox slice used by /api/similar.
// The cache avoids re-fetching 1000 emails over JMAP on every click of "Find
// Similar" or every iteration of the rapid "Archive & Find Next" sweep.
// handleArchive and handleUnarchive invalidate it so the next call sees the
// post-archive state.
func (s *Server) getInboxForSimilarity() ([]jmap.Email, error) {
	s.inboxMu.Lock()
	if s.inboxCache != nil && time.Since(s.inboxCachedAt) < inboxCacheTTL {
		cached := s.inboxCache
		s.inboxMu.Unlock()
		return cached, nil
	}
	s.inboxMu.Unlock()

	emails, err := s.jmapClient.GetInboxEmails(1000)
	if err != nil {
		return nil, err
	}

	s.inboxMu.Lock()
	s.inboxCache = emails
	s.inboxCachedAt = time.Now()
	s.inboxMu.Unlock()
	return emails, nil
}

func (s *Server) invalidateInboxCache() {
	s.inboxMu.Lock()
	s.inboxCache = nil
	s.inboxCachedAt = time.Time{}
	s.inboxMu.Unlock()
}
