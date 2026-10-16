# Mailbox Zero - Claude Development Documentation

## Project Status: ✅ COMPLETED

**Last Updated:** October 11, 2026 (README refresh)
**Version:** 1.0
**Status:** Production Ready (with dry run safety)

## Project Overview

Mailbox Zero is a Go-based web application that helps users clean up their Fastmail inbox by finding and archiving similar emails using the JMAP protocol. The application provides a dual-pane interface with advanced fuzzy matching capabilities and comprehensive safety features.

## Architecture

### Tech Stack
- **Backend:** Go 1.26+ with Gorilla Mux router
- **Frontend:** Vanilla HTML5, CSS3, JavaScript (ES6+)
- **Email Protocol:** JMAP (JSON Meta Application Protocol)
- **Email Provider:** Fastmail
- **Configuration:** YAML
- **Templates:** Go HTML templates, built into the binary with `embed` together with the static files

### Project Structure
```
mailboxzero/
├── main.go                     # Application entry point
├── go.mod / go.sum             # Go module dependencies (gorilla/mux, yaml.v3)
├── config.yaml.example         # Configuration template
├── config-mock.yaml.example    # Mock-mode configuration template
├── .gitignore                  # Git ignore rules
├── LICENSE                     # MIT license
├── README.md                   # User documentation
├── CLAUDE.md                   # Development documentation (this file)
├── .github/
│   ├── dependabot.yml          # Weekly dependency update PRs: Go modules, GitHub Actions, e2e npm
│   └── workflows/
│       └── test.yml            # CI: gofmt check, `go test -race`, then `-coverprofile` without `-race` (together they are slow), Playwright e2e tests. Only the test job saves the Go build cache (the e2e job restores it), and `-count=1` keeps every test running
├── e2e/                        # Playwright end-to-end tests, run against the server in mock mode
│   ├── package.json            # @playwright/test dev dependency (plus package-lock.json)
│   ├── playwright.config.js    # Starts the server with config-e2e.yaml and drives the installed Chrome
│   ├── config-e2e.yaml         # Mock mode with dry_run: false on port 18080
│   └── tests/                  # *.spec.js browser tests
├── internal/
│   ├── config/
│   │   ├── config.go           # Configuration loading and validation
│   │   └── config_test.go
│   ├── jmap/
│   │   ├── client.go           # JMAPClient interface + real Client transport
│   │   ├── email.go            # Email/Mailbox types + Archive/Unarchive operations
│   │   ├── mock.go             # MockClient with built-in sample data for mock mode
│   │   ├── client_test.go      # Authenticate and GetMailboxes against the fake server
│   │   ├── fakeserver_test.go  # Fake JMAP server (session + API) for testing the real Client over HTTP
│   │   ├── jmap_test.go
│   │   └── mock_test.go
│   ├── server/
│   │   ├── server.go           # HTTP server and API handlers
│   │   └── server_test.go
│   └── similarity/
│       ├── similarity.go       # Fuzzy matching algorithms
│       ├── candidates.go       # Index of the email pairs that could reach the threshold
│       ├── candidates_test.go  # Checks the fast grouping against comparing every pair
│       └── similarity_test.go
└── web/
    ├── web.go                  # Embeds templates/ and static/ into the binary
    ├── templates/
    │   └── index.html          # Main application template
    └── static/
        ├── style.css           # Application styles
        └── app.js              # Frontend JavaScript
```

### Core Components

#### 1. Configuration System (`internal/config/`)
- **File:** `config.go`
- **Purpose:** Manages YAML configuration loading and validation
- **Features:**
  - Server port/host configuration
  - JMAP endpoint and credentials
  - Dry run safety toggle
  - Default similarity threshold

#### 2. JMAP Client (`internal/jmap/`)
- **Files:** `client.go`, `email.go`, `mock.go`
- **Purpose:** Handles communication with Fastmail's JMAP API
- **Features:**
  - `JMAPClient` interface implemented by both the real `Client` and the in-memory `MockClient` (selected via `mock_mode` in config)
  - Session authentication with Bearer tokens
  - Mailbox discovery (inbox, archive): the ids are looked up once and remembered (`mailboxIDs`), so an inbox page is one request (Mailbox/get for the inbox count, Email/query, Email/get) and an archive or undo is one Email/set. A failed request forgets them, and an account without an archive mailbox is looked up again each time
  - Email querying without bodies (summaries with JMAP's `preview`, `emailSummaryProperties`), including a paginated `GetInboxEmailsWithCountPaginated` used by the `/api/emails` endpoint; `GetEmail` fetches one email with its text and HTML bodies on demand for the hover preview, returning `ErrEmailNotFound` for an unknown id
  - Safe archive operations: archive and unarchive send an Email/set patch that changes only the inbox and archive memberships (`mailboxIds/<id>`), so any other mailbox or label the email is in is kept
  - Comprehensive error handling

#### 3. Similarity Engine (`internal/similarity/`)
- **Files:** `similarity.go`, `candidates.go`
- **Purpose:** Fuzzy matching tuned for newsletter/notification clustering
- **Algorithm:**
  - **Sender Similarity (50% weight):** Structured ladder, not Levenshtein. Same full address → 1.0; same domain → 0.8 (suppressed for shared-ESP domains like `sendgrid.net` unless local parts also match); same registrable root domain (e.g. `mail.google.com` ↔ `accounts.google.com`) → 0.7; same non-generic display name → 0.6.
  - **Subject Similarity (30% weight):** `max(tokenJaccard, levenshtein)` over a normalized subject that has reply/forward prefixes (`Re:`, `Fwd:`, `[list]`) stripped. Token Jaccard filters a small stop-word set (`weekly`, `newsletter`, `update`, etc.) and requires both sides to have ≥2 surviving tokens before being credited; otherwise the score falls back to Levenshtein alone.
  - **Body Similarity (20% weight):** Jaccard similarity over normalized word tokens (length ≥ 3) extracted from JMAP's `preview` text (listings no longer fetch bodies; a body is only used if a preview is missing).
- **Features:**
  - Per-email features (`senderAddr`, `senderDomain`, `senderRoot`, `senderName`, `subjectNorm`, `subjectRunes`, `subjectTokens`, `bodyTokens`) precomputed once per call so the inner pairwise compare avoids re-parsing senders or re-tokenizing strings
  - Threshold-aware short-circuit: cheap-first ordering (sender → body → subject) bails out as soon as the partial score plus the maximum remaining contribution falls below `threshold`. The subject's Levenshtein distance is skipped when the length difference already rules it out, and stops early once it exceeds what the subject can afford; the score is still summed sender + subject + body, so results are unchanged
  - Single-link cluster expansion in `groupSimilarFeatures`: a candidate joins a cluster if it matches ANY existing member, not just the seed — important when the seed is an atypical member of an obvious group. The expansion is breadth-first, so each member is compared once with each unclustered email
  - Candidate index (`candidateIndex` in `candidates.go`): grouping only compares emails that could reach the threshold. Above 50% that means sharing a sender key (address, domain, root domain or display name, mirroring `senderSimilarity`); between 30% and 50% also sharing one of the rarest body tokens (prefix filtering on the body Jaccard that a pair from unrelated senders needs); at 30% and below every pair. `candidates_test.go` checks on random inboxes that the groups are identical to comparing every pair with the original scoring
  - String normalization (lowercase, punctuation-to-space, collapsed whitespace)
  - Public-suffix awareness for a small set of multi-label suffixes (`co.uk`, `github.io`, etc.)
  - Group-based and individual email matching

#### 4. Web Server (`internal/server/`)
- **File:** `server.go`
- **Purpose:** HTTP server with RESTful API endpoints
- **Endpoints:**
  - `GET /` - Main application interface
  - `GET /api/emails` - Fetch inbox emails
  - `GET /api/emails/{id}` - Fetch one email with its bodies (hover preview)
  - `POST /api/similar` - Find similar emails with threshold
  - `POST /api/archive` - Archive selected emails
  - `POST /api/unarchive` - Move previously archived emails back to the inbox (powers the Undo button)
  - `POST /api/clear` - Clear results
- **Features:**
  - Template rendering with data injection, from the files `web/web.go` embeds (the binary runs from any directory)
  - JSON API responses
  - Error handling and logging
  - Static file serving
  - `Handler()` builds the router behind the request checks described under Security Considerations (testable end to end); `Start()` serves it through `newHTTPServer`, which sets timeouts: 10 s to read headers, 30 s to read the request, 2 min to write the response (Find Similar fetches up to 10,000 emails in JMAP requests of up to 1000, each with a 30 s client timeout, then groups them), 2 min idle
  - Find Similar searches the newest 10,000 inbox emails (`maxSimilarityEmails`), which the client fetches in pages of up to 1000 (or the server's `maxObjectsInGet`). A targeted email outside them (older, or newer than the cache) is fetched by itself with `GetEmail`
  - Inbox cache for `/api/similar`: that fetch is cached in-process for 60 seconds (`inboxCacheTTL`). A successful archive removes the archived emails from the cache (a dry run leaves it), so successive "Find Similar" / "Archive & Find Next" calls don't re-pull from JMAP; a failed archive (JMAP can reject some emails while moving the rest) and any unarchive invalidate it

#### 5. Frontend Interface (`web/`)
- **Template:** `index.html` - Responsive dual-pane layout
- **Styles:** `style.css` - Modern CSS with mobile responsiveness
- **JavaScript:** `app.js` - Single-page application logic
- **Embedding:** `web.go` builds `templates/` and `static/` into the binary, so after editing them restart the server (`go run` rebuilds)
- **Features:**
  - Real-time similarity threshold adjustment
  - Email selection and multi-select capabilities
  - Modal confirmation dialogs
  - Async API communication. Inbox and Find Similar requests are numbered (`inboxRequest`, `similarRequest` in `app.js`), and an answer to a request that a newer one replaced (or that Clear Results cancelled) is ignored, since answers can arrive out of order
  - Responsive design for mobile/desktop

## Key Features Implemented

### ✅ Safety Features
1. **Dry Run Mode:** Default enabled, prevents actual email modifications. After an archive the status pill says "Dry run: would have archived N e-mail(s)" (amber, `dry-run` class) and offers no Undo, since nothing moved
2. **Archive Only:** Never deletes emails, only moves to archive folder
3. **Confirmation Dialogs:** Required before the standard "Archive Selected" write operation (the opt-in "Archive & Find Next" button intentionally skips it for rapid sweeping)
4. **Visual Warnings:** Clear UI indicators when in dry run mode
5. **API Token Authentication:** Secure authentication using Fastmail API tokens
6. **Undo Archive:** After every successful archive, a status pill with an "Undo" button appears in the top-right of the action bar. Undo POSTs the last-archived IDs to `/api/unarchive`, which moves them back to the inbox. Undo then reloads the inbox, so the restored emails and the inbox count come back right away (a failed Undo reloads it too, since some emails may have moved back); the similar-emails pane is left as it is until the next Find Similar. After Undo, the status text changes to "Unarchived N e-mail(s)" with a glow animation (`undoGlow` in `style.css`).
7. **Flagged Emails Stay Unselected:** In a group of similar emails, every email starts selected except those flagged in Fastmail (`$flagged` keyword), which show a ⚑ marker; unread emails (no `$seen`) show a blue dot. Select All still includes flagged emails

### ✅ Core Functionality
1. **Dual-Pane Interface:** Inbox (left) and similar emails (right)
2. **Smart Similarity Matching:** Multi-factor fuzzy algorithm
3. **Adjustable Threshold:** 0-100% similarity slider with real-time updates
4. **Email Selection:** Individual and bulk selection with checkboxes
5. **Archive Operations:** Bulk archive with JMAP email movement
6. **Rapid Archive Loop:** "Archive & Find Next" button archives the selected similar emails without confirmation and immediately re-runs the same similarity search (uses `archiveAndFindNext()` in `app.js`)
7. **Clear Results:** Reset functionality for multiple searches
8. **Individual Email Targeting:** Select specific email to find matches. The selection is cleared once that email is archived (`clearArchivedSelection()` in `app.js`), so the next Find Similar searches the whole inbox

### ✅ User Experience
1. **Responsive Design:** Mobile and desktop optimized
2. **Loading States:** Clear feedback during async operations
3. **Error Handling:** User-friendly error messages. A failed archive or Undo shows the server's reason (each email JMAP rejected, and why) in a red status pill instead of an alert, and the inbox reloads because some emails may have moved (`showArchiveError()` in `app.js`)
4. **Keyboard:** Shortcuts `F` (Find Similar), `A` (Archive & Find Next), `U` (Undo), `R` (Refresh), `C` (Clear Results) and `Escape` (close the preview or the archive dialog), handled by `handleShortcut()` in `app.js`. They are ignored in a drop-down, with Ctrl/Cmd/Alt, and while the archive dialog is open. The buttons show them in `title` and `aria-keyshortcuts`. Inbox rows are focusable buttons (`aria-pressed` when chosen): Enter or Space chooses one, and ArrowUp/ArrowDown move between inbox rows or between the checkboxes of the similar emails
5. **Performance:** Efficient API calls and client-side caching
6. **Empty Inbox:** An empty inbox shows "Inbox zero!" instead of a blank list, pagination never shows "Page 1 of 0", and a last page that was emptied in the meantime falls back to the new last page (`loadEmails()` in `app.js`)

## Configuration Details

### Required Settings
```yaml
server:
  port: 8080                    # Default web server port
  host: "localhost"             # Server binding host: localhost, 127.0.0.1 or ::1 only

jmap:
  endpoint: "https://api.fastmail.com/jmap/session"
  api_token: ""                 # Fastmail API token (required)

dry_run: true                   # Safety feature - MUST be false for real operations
default_similarity: 60          # Default similarity percentage (0-100)
```

### Security Considerations
- **Localhost Only:** The web UI has no login, so `validate()` in `internal/config/config.go` rejects any `server.host` that `IsLoopbackHost` does not accept (`localhost`, 127.0.0.0/8, `::1`), including empty and `0.0.0.0`. The app deliberately provides no authentication or HTTPS; remote use goes through an SSH tunnel
- **Browser Protections:** `Handler()` wraps the router in `localhostOnly` (403 unless the `Host` header is a loopback name, which stops DNS rebinding) and Go's `http.CrossOriginProtection` (403 for cross-origin `POST`s via `Sec-Fetch-Site`/`Origin`, which stops cross-site request forgery). `e2e/tests/cross-site.spec.js` checks the latter in a real browser
- **API Tokens:** Use Fastmail API tokens for secure authentication
- **Local Processing:** All similarity calculations happen locally
- **Minimal Permissions:** Only requires read access to inbox and write to archive
- **No External Services:** No data sent to third-party services

## API Endpoints

### GET /api/emails
- **Purpose:** Retrieve inbox emails (paginated)
- **Query Parameters:** `limit` (default 100, capped at 500, the largest page size the UI offers) and `offset` (default 0)
- **Response:** JSON object `{ "emails": [...], "totalCount": N }` (`InboxInfo`)
- **Email Fields:** ID, subject, from, preview, receivedAt, keywords (`$seen`, `$flagged`) (no bodies: `bodyValues` is `null`; see `GET /api/emails/{id}`)

### GET /api/emails/{id}
- **Purpose:** Fetch one email with its text and HTML bodies, for the hover preview. Listings and `/api/similar` leave bodies out, and `app.js` (`loadPreviewBody()`) fetches them here the first time an email is previewed
- **Response:** JSON email object including `bodyValues`, `textBody` and `htmlBody`; 404 if there is no such email

### POST /api/similar
- **Purpose:** Find similar emails
- **Request Body:**
  ```json
  {
    "similarityThreshold": 60.0,
    "emailId": "optional-specific-email-id"
  }
  ```
- **Searches:** the newest 10,000 inbox emails. A targeted `emailId` outside them is fetched by itself; HTTP 404 if it no longer exists
- **Response:** JSON array of matching email objects; an empty array `[]` (never `null`) when nothing matches

### POST /api/archive
- **Purpose:** Archive selected emails
- **Request Body:**
  ```json
  {
    "emailIds": ["id1", "id2", "id3"]
  }
  ```
- **Response:** Success confirmation with dry run status. If JMAP rejects the Email/set call or any of the emails (`notUpdated`), HTTP 500 with a message naming each rejected email and the reason

### POST /api/unarchive
- **Purpose:** Move previously archived emails back to the inbox (Undo)
- **Request Body:**
  ```json
  {
    "emailIds": ["id1", "id2", "id3"]
  }
  ```
- **Response:** Success confirmation with dry run status (same shape as `/api/archive`, including the HTTP 500 for rejected emails)

### POST /api/clear
- **Purpose:** Clear similarity results
- **Response:** Success confirmation

## Development Commands

### Setup and Dependencies
```bash
# Initialize Go modules
go mod download
go mod tidy

# Create configuration file from example
cp config.yaml.example config.yaml
# Edit config.yaml with your Fastmail API token

# Run the application
go run main.go

# Run with custom config
go run main.go -config custom-config.yaml

# Build for production (a single binary that includes the web page)
go build -o mailboxzero main.go
```

### Mock Mode for Testing

For development and testing purposes, you can run the application in mock mode:

```bash
# Copy the mock configuration
cp config-mock.yaml.example config.yaml

# Run in mock mode - no Fastmail credentials needed
go run main.go
```

**Mock Mode Features:**
- Uses realistic sample email data (41 emails: 10 senders with 3, 4 or 5 messages each, plus 2 unique)
- No real JMAP connection required
- Sample emails include groups of similar messages for testing similarity matching
- Simulates archiving operations without affecting real emails
- Perfect for development, testing, and demonstrations
- The same 41 emails (senders, subjects, content, IDs) on every run; only the dates move with today so the inbox always looks recent

**Mock Configuration:**
```yaml
server:
  port: 8080
  host: "localhost"

jmap:
  endpoint: ""      # Not required in mock mode
  api_token: ""     # Not required in mock mode

dry_run: true       # Keep enabled for safety
default_similarity: 60
mock_mode: true     # Enable mock mode
```

### Testing Commands
```bash
# Quick start with mock data (no Fastmail account required)
go run main.go -config config-mock.yaml.example

# Run all tests
go test ./...

# Run tests with verbose output
go test ./... -v

# Run with race detection
go test -race ./...

# Run tests with coverage
go test ./... -cover

# Generate coverage report
go test ./... -coverprofile=coverage.out
go tool cover -html=coverage.out

# Run benchmarks
go test ./... -bench=.

# Run the end-to-end browser tests (requires Node.js and Google Chrome)
cd e2e && npm ci && npx playwright test

# Lint the code (requires golangci-lint)
golangci-lint run
```

## Deployment Instructions

### Production Deployment
1. **Build Binary:** the page and static files are built in, so the binary runs from any directory
   ```bash
   go build -o mailboxzero main.go
   ```

2. **Create Production Config:**
   ```bash
   cp config.yaml.example config.yaml
   ```
   
   Then edit `config.yaml`:
   ```yaml
   server:
     port: 8080
     host: "localhost"  # Only loopback is accepted; see Remote Access below
   jmap:
     api_token: "production-api-token"
   dry_run: false      # Enable real operations
   ```

3. **Run Binary:**
   ```bash
   ./mailboxzero -config production-config.yaml
   ```

### Remote Access
The server only listens on localhost and has no login. To use it from another machine, forward the port over SSH and open http://localhost:8080 locally:
```bash
ssh -L 8080:localhost:8080 user@server
```

### Docker Deployment (Future Enhancement)
The server only binds to loopback, so a published container port (`-p 8080:8080`) cannot reach it. Run such a container with host networking (`--network host`, Linux) instead.
```dockerfile
FROM golang:1.26-alpine AS builder
WORKDIR /app
COPY . .
RUN go build -o mailboxzero main.go

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/mailboxzero .
CMD ["./mailboxzero"]
```

## Known Limitations and Future Enhancements

### Current Limitations
1. **Single Account:** Only supports one Fastmail account at a time
2. **Memory Usage:** Loads all emails into memory for processing
3. **No Persistent Storage:** No database for tracking operations
4. **Bearer Token Authentication:** Uses API tokens instead of OAuth2
5. **Local Access Only:** No login or HTTPS; the server only binds to loopback, so remote use needs an SSH tunnel

### Potential Enhancements
1. **Multi-Account Support:** Support multiple email accounts
2. **Database Integration:** SQLite for operation history and caching
3. **Advanced Filters:** Date ranges, sender whitelist, size limits
4. **Batch Operations:** Process large inboxes in chunks
5. **Email Preview:** Full email content preview before archiving
6. **Statistics Dashboard:** Email cleanup metrics and reports
7. **API Rate Limiting:** Respect JMAP API rate limits
8. **OAuth2 Support:** Modern authentication flow

## Troubleshooting Guide

### Common Issues

#### Authentication Failures
- **Symptom:** "Failed to authenticate" error
- **Solutions:**
  1. Verify Fastmail API token is correct
  2. Generate new API token in Fastmail settings (Settings → Privacy & Security → Integrations)
  3. Ensure JMAP is enabled in account settings
  4. Check network connectivity to api.fastmail.com

#### No Emails Found
- **Symptom:** Empty inbox or no similar emails found
- **Solutions:**
  1. Verify emails exist in Fastmail inbox
  2. Lower similarity threshold (try 50% or lower)
  3. Check email content has sufficient text for matching
  4. Verify mailbox permissions

#### UI Not Loading
- **Symptom:** Blank page or JavaScript errors
- **Solutions:**
  1. Check browser console for errors
  2. Verify static files are served correctly
  3. Clear browser cache
  4. Check server logs for template errors

### Debug Mode
Enable verbose logging by modifying main.go:
```go
log.SetFlags(log.LstdFlags | log.Lshortfile)
```

## Code Quality Standards

### Go Best Practices Followed
1. **Package Organization:** Clear internal package structure
2. **Error Handling:** Comprehensive error wrapping and logging
3. **Interface Design:** Clean separation of concerns
4. **Memory Management:** Efficient string operations and minimal allocations
5. **Concurrency Safety:** Thread-safe operations where needed
6. **Comprehensive Testing:** Full unit test coverage with table-driven tests

### Test Coverage
The project includes comprehensive unit tests for all packages:
- **Config Package:** Configuration loading, validation, and error handling
- **JMAP Package:** Data parsing, mock client functionality, helper functions, and the real Client's requests and responses via a fake JMAP server (`newFakeJMAPServer` in `fakeserver_test.go`), including authentication against its session resource and bearer-token check
- **Similarity Package:** Fuzzy matching algorithms, Levenshtein distance, email similarity
- **Server Package:** HTTP handlers, API endpoints, and request/response handling

Tests follow Go best practices:
- Table-driven test design for multiple scenarios
- Clear test naming and organization
- Use of test helpers and fixtures
- Mock clients for external dependencies
- Benchmark tests for performance-critical functions

### Code Style
- **Naming:** Clear, descriptive variable and function names
- **Documentation:** Comprehensive comments and documentation
- **Formatting:** Standard `gofmt` formatting
- **Imports:** Organized standard, external, and internal imports
- **Error Messages:** User-friendly error messages with context

## Maintenance Notes

### Regular Maintenance Tasks
1. **Dependency Updates:** Dependabot (`.github/dependabot.yml`) opens weekly PRs for Go modules, GitHub Actions and the e2e Playwright package; review and merge them once CI passes. `go-junit-report`, which CI uses for the test report, is pinned in `test.yml` (`GO_JUNIT_REPORT_VERSION`, with its binary cached by version) and is updated by hand
2. **Security Patches:** Monitor for security vulnerabilities
3. **Performance Monitoring:** Track API response times
4. **Log Analysis:** Review error patterns and usage metrics

### Backup Considerations
- **Configuration Files:** The config.yaml file contains credentials and should NOT be committed to version control (already in .gitignore)
- **User Data:** No persistent user data to backup
- **Application State:** Stateless application, no backup needed
- **Template Files:** config.yaml.example should be committed as a template

---

**Note:** This application is designed with safety as the primary concern. The dry run mode should remain enabled during initial testing, and all operations should be thoroughly tested before enabling real email modifications.