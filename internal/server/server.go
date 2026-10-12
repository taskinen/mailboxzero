package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"log"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/taskinen/mailboxzero/internal/config"
	"github.com/taskinen/mailboxzero/internal/jmap"
	"github.com/taskinen/mailboxzero/internal/similarity"
	"github.com/taskinen/mailboxzero/web"

	"github.com/gorilla/mux"
)

const inboxCacheTTL = 60 * time.Second

// maxSimilarityEmails caps how many inbox emails Find Similar searches,
// newest first, to bound the time one search takes on a very large inbox.
// A targeted email outside them is fetched by itself.
const maxSimilarityEmails = 10000

// maxEmailsPageSize caps the limit parameter of /api/emails. It matches the
// largest page size the UI offers; anything bigger only makes one JMAP
// request fetch (and the server hold) an unbounded number of emails.
const maxEmailsPageSize = 500

// HTTP server timeouts, so slow or stalled clients cannot hold connections
// open indefinitely. Find Similar fetches up to maxSimilarityEmails emails in
// JMAP requests of up to 1000 (usually well under a second each, with a
// 30-second client timeout) and then groups them, so the write timeout
// leaves room for that.
const (
	readHeaderTimeout = 10 * time.Second
	readTimeout       = 30 * time.Second
	writeTimeout      = 2 * time.Minute
	idleTimeout       = 2 * time.Minute
)

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
}

func New(cfg *config.Config, jmapClient jmap.JMAPClient) (*Server, error) {
	templates, err := template.ParseFS(web.Templates, "*.html")
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
	addr := s.config.GetServerAddr()
	log.Printf("Server starting on http://%s", addr)
	log.Printf("DRY RUN MODE: %v", s.config.DryRun)

	return newHTTPServer(addr, s.Handler()).ListenAndServe()
}

// Handler returns the HTTP handler that serves the UI, static files and API.
//
// Listening on localhost only keeps other machines out, but every website
// open in the user's browser can still send requests here. Two checks run
// before the router:
//   - localhostOnly refuses requests addressed to any other host name, which
//     stops DNS rebinding (a foreign domain re-pointed at 127.0.0.1).
//   - CrossOriginProtection refuses cross-origin POSTs, so another site cannot
//     make the browser archive or unarchive emails.
func (s *Server) Handler() http.Handler {
	r := mux.NewRouter()

	r.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServerFS(web.Static)))

	r.HandleFunc("/", s.handleIndex).Methods("GET")
	r.HandleFunc("/api/emails", s.handleGetEmails).Methods("GET")
	r.HandleFunc("/api/emails/{id}", s.handleGetEmail).Methods("GET")
	r.HandleFunc("/api/similar", s.handleFindSimilar).Methods("POST")
	r.HandleFunc("/api/archive", s.handleArchive).Methods("POST")
	r.HandleFunc("/api/unarchive", s.handleUnarchive).Methods("POST")
	r.HandleFunc("/api/clear", s.handleClear).Methods("POST")

	return localhostOnly(http.NewCrossOriginProtection().Handler(r))
}

// localhostOnly serves only requests whose Host header names this machine.
// The server listens on loopback only, so any other Host means a browser
// reached it through a rebound DNS name.
func localhostOnly(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		host, _, err := net.SplitHostPort(r.Host)
		if err != nil {
			// No port: the Host is a bare name or a bracketed IPv6 address.
			host = strings.TrimSuffix(strings.TrimPrefix(r.Host, "["), "]")
		}
		if !config.IsLoopbackHost(host) {
			http.Error(w, "Forbidden: requests must be addressed to localhost", http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// newHTTPServer returns the http.Server for addr and h, with timeouts set.
func newHTTPServer(addr string, h http.Handler) *http.Server {
	return &http.Server{
		Addr:              addr,
		Handler:           h,
		ReadHeaderTimeout: readHeaderTimeout,
		ReadTimeout:       readTimeout,
		WriteTimeout:      writeTimeout,
		IdleTimeout:       idleTimeout,
	}
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
			limit = min(l, maxEmailsPageSize)
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
	// Encode an empty inbox as [] rather than null; the frontend iterates it.
	if inboxInfo.Emails == nil {
		inboxInfo.Emails = []jmap.Email{}
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(inboxInfo); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

// handleGetEmail returns one email with its bodies. Inbox listings leave the
// bodies out, so the UI fetches them here when it previews an email.
func (s *Server) handleGetEmail(w http.ResponseWriter, r *http.Request) {
	email, err := s.jmapClient.GetEmail(mux.Vars(r)["id"])
	if errors.Is(err, jmap.ErrEmailNotFound) {
		http.Error(w, "Email not found", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get email: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(email); err != nil {
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

		// The target is missing when it is older than the emails searched,
		// or newer than the cached inbox: fetch it by itself.
		if targetEmail == nil {
			targetEmail, err = s.jmapClient.GetEmail(req.EmailID)
			if errors.Is(err, jmap.ErrEmailNotFound) {
				http.Error(w, "Target email not found", http.StatusNotFound)
				return
			}
			if err != nil {
				http.Error(w, fmt.Sprintf("Failed to get target email: %v", err), http.StatusInternalServerError)
				return
			}
		}

		similarEmails = similarity.FindSimilarToEmail(*targetEmail, emails, req.SimilarityThreshold/100.0)
	} else {
		similarEmails = similarity.FindSimilarEmails(emails, req.SimilarityThreshold/100.0)
	}
	// No matches must encode as [] rather than null; the frontend reads .length.
	if similarEmails == nil {
		similarEmails = []jmap.Email{}
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
		// When the server rejects only some of the emails, the others have
		// still moved, so the cached inbox is out of date.
		s.invalidateInboxCache()
		log.Printf("Archive failed: %v", err)
		http.Error(w, fmt.Sprintf("Failed to archive emails: %v", err), http.StatusInternalServerError)
		return
	}
	// The archived emails have left the inbox. Dropping them from the cache
	// saves fetching the whole inbox again for the next search. A dry run
	// moved nothing.
	if !s.config.DryRun {
		s.removeFromInboxCache(req.EmailIDs)
	}

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

	err := s.jmapClient.UnarchiveEmails(req.EmailIDs, s.config.DryRun)
	// Invalidate even on failure: when the server rejects only some of the
	// emails, the others have still moved.
	s.invalidateInboxCache()
	if err != nil {
		log.Printf("Unarchive failed: %v", err)
		http.Error(w, fmt.Sprintf("Failed to unarchive emails: %v", err), http.StatusInternalServerError)
		return
	}

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
// The cache avoids re-fetching up to maxSimilarityEmails emails over JMAP on
// every click of "Find Similar" or every iteration of the rapid "Archive &
// Find Next" sweep. handleArchive drops the emails it archived from it, and
// handleUnarchive invalidates it, so the next call sees the post-archive
// state.
func (s *Server) getInboxForSimilarity() ([]jmap.Email, error) {
	s.inboxMu.Lock()
	if s.inboxCache != nil && time.Since(s.inboxCachedAt) < inboxCacheTTL {
		cached := s.inboxCache
		s.inboxMu.Unlock()
		return cached, nil
	}
	s.inboxMu.Unlock()

	emails, err := s.jmapClient.GetInboxEmails(maxSimilarityEmails)
	if err != nil {
		return nil, err
	}

	s.inboxMu.Lock()
	s.inboxCache = emails
	s.inboxCachedAt = time.Now()
	s.inboxMu.Unlock()
	return emails, nil
}

// removeFromInboxCache drops archived emails from the cached inbox. The
// cache keeps its age, so new mail still shows up after inboxCacheTTL.
func (s *Server) removeFromInboxCache(emailIDs []string) {
	archived := make(map[string]bool, len(emailIDs))
	for _, id := range emailIDs {
		archived[id] = true
	}

	s.inboxMu.Lock()
	defer s.inboxMu.Unlock()
	if s.inboxCache == nil {
		return
	}
	// Build a new slice: callers may still be reading the old one.
	kept := make([]jmap.Email, 0, len(s.inboxCache))
	for _, email := range s.inboxCache {
		if !archived[email.ID] {
			kept = append(kept, email)
		}
	}
	s.inboxCache = kept
}

func (s *Server) invalidateInboxCache() {
	s.inboxMu.Lock()
	s.inboxCache = nil
	s.inboxCachedAt = time.Time{}
	s.inboxMu.Unlock()
}
