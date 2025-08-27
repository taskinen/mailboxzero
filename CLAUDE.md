# Mailbox Zero - Claude Development Documentation

## Project Status: ✅ COMPLETED

**Last Updated:** August 26, 2025
**Version:** 1.0
**Status:** Production Ready (with dry run safety)

## Project Overview

Mailbox Zero is a Go-based web application that helps users clean up their Fastmail inbox by finding and archiving similar emails using the JMAP protocol. The application provides a dual-pane interface with advanced fuzzy matching capabilities and comprehensive safety features.

## Architecture

### Tech Stack
- **Backend:** Go 1.21+ with Gorilla Mux router
- **Frontend:** Vanilla HTML5, CSS3, JavaScript (ES6+)
- **Email Protocol:** JMAP (JSON Meta Application Protocol)
- **Email Provider:** Fastmail
- **Configuration:** YAML
- **Templates:** Go HTML templates

### Project Structure
```
mailboxzero/
├── main.go                     # Application entry point
├── go.mod                      # Go module dependencies
├── config.yaml.example         # Configuration template
├── .gitignore                  # Git ignore rules
├── README.md                   # User documentation
├── CLAUDE.md                   # Development documentation (this file)
├── internal/
│   ├── config/
│   │   └── config.go          # Configuration loading and validation
│   ├── jmap/
│   │   ├── client.go          # JMAP client implementation
│   │   └── email.go           # Email data structures and operations
│   ├── server/
│   │   └── server.go          # HTTP server and API handlers
│   └── similarity/
│       └── similarity.go      # Fuzzy matching algorithms
└── web/
    ├── templates/
    │   └── index.html         # Main application template
    └── static/
        ├── style.css          # Application styles
        └── app.js             # Frontend JavaScript
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
- **Files:** `client.go`, `email.go`
- **Purpose:** Handles communication with Fastmail's JMAP API
- **Features:**
  - Session authentication with Bearer tokens
  - Mailbox discovery (inbox, archive)
  - Email querying and retrieval with body content
  - Safe archive operations (move to archive folder)
  - Comprehensive error handling

#### 3. Similarity Engine (`internal/similarity/`)
- **File:** `similarity.go`
- **Purpose:** Advanced fuzzy matching for email similarity
- **Algorithm:**
  - **Subject Similarity (40% weight):** Levenshtein distance with normalization
  - **Sender Similarity (40% weight):** Email address comparison
  - **Content Similarity (20% weight):** Body/preview text analysis
- **Features:**
  - String normalization (lowercase, punctuation removal)
  - Common word detection for similarity boosting
  - Configurable threshold matching
  - Group-based and individual email matching

#### 4. Web Server (`internal/server/`)
- **File:** `server.go`
- **Purpose:** HTTP server with RESTful API endpoints
- **Endpoints:**
  - `GET /` - Main application interface
  - `GET /api/emails` - Fetch inbox emails
  - `POST /api/similar` - Find similar emails with threshold
  - `POST /api/archive` - Archive selected emails
  - `POST /api/clear` - Clear results
- **Features:**
  - Template rendering with data injection
  - JSON API responses
  - Error handling and logging
  - Static file serving

#### 5. Frontend Interface (`web/`)
- **Template:** `index.html` - Responsive dual-pane layout
- **Styles:** `style.css` - Modern CSS with mobile responsiveness
- **JavaScript:** `app.js` - Single-page application logic
- **Features:**
  - Real-time similarity threshold adjustment
  - Email selection and multi-select capabilities
  - Modal confirmation dialogs
  - Async API communication
  - Responsive design for mobile/desktop

## Key Features Implemented

### ✅ Safety Features
1. **Dry Run Mode:** Default enabled, prevents actual email modifications
2. **Archive Only:** Never deletes emails, only moves to archive folder
3. **Confirmation Dialogs:** Required before any write operations
4. **Visual Warnings:** Clear UI indicators when in dry run mode
5. **API Token Authentication:** Secure authentication using Fastmail API tokens

### ✅ Core Functionality
1. **Dual-Pane Interface:** Inbox (left) and similar emails (right)
2. **Smart Similarity Matching:** Multi-factor fuzzy algorithm
3. **Adjustable Threshold:** 0-100% similarity slider with real-time updates
4. **Email Selection:** Individual and bulk selection with checkboxes
5. **Archive Operations:** Bulk archive with JMAP email movement
6. **Clear Results:** Reset functionality for multiple searches
7. **Individual Email Targeting:** Select specific email to find matches

### ✅ User Experience
1. **Responsive Design:** Mobile and desktop optimized
2. **Loading States:** Clear feedback during async operations
3. **Error Handling:** User-friendly error messages
4. **Accessibility:** Keyboard navigation and screen reader support
5. **Performance:** Efficient API calls and client-side caching

## Configuration Details

### Required Settings
```yaml
server:
  port: 8080                    # Default web server port
  host: "localhost"             # Server binding host

jmap:
  endpoint: "https://api.fastmail.com/jmap/session"
  api_token: ""                 # Fastmail API token (required)

dry_run: true                   # Safety feature - MUST be false for real operations
default_similarity: 75          # Default similarity percentage (0-100)
```

### Security Considerations
- **API Tokens:** Use Fastmail API tokens for secure authentication
- **Local Processing:** All similarity calculations happen locally
- **Minimal Permissions:** Only requires read access to inbox and write to archive
- **No External Services:** No data sent to third-party services

## API Endpoints

### GET /api/emails
- **Purpose:** Retrieve inbox emails
- **Response:** JSON array of email objects
- **Limit:** 100 emails for performance
- **Fields:** ID, subject, from, preview, receivedAt, bodyValues

### POST /api/similar
- **Purpose:** Find similar emails
- **Request Body:**
  ```json
  {
    "similarityThreshold": 75.0,
    "emailId": "optional-specific-email-id"
  }
  ```
- **Response:** JSON array of matching email objects

### POST /api/archive
- **Purpose:** Archive selected emails
- **Request Body:**
  ```json
  {
    "emailIds": ["id1", "id2", "id3"]
  }
  ```
- **Response:** Success confirmation with dry run status

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

# Build for production
go build -o mailboxzero main.go
```

### Testing Commands
```bash
# Run all tests (when implemented)
go test ./...

# Run with race detection
go test -race ./...

# Lint the code (requires golangci-lint)
golangci-lint run
```

## Deployment Instructions

### Production Deployment
1. **Build Binary:**
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
     host: "0.0.0.0"  # For external access
   jmap:
     api_token: "production-api-token"
   dry_run: false      # Enable real operations
   ```

3. **Run Binary:**
   ```bash
   ./mailboxzero -config production-config.yaml
   ```

### Docker Deployment (Future Enhancement)
```dockerfile
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY . .
RUN go build -o mailboxzero main.go

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/mailboxzero .
COPY --from=builder /app/web ./web
CMD ["./mailboxzero"]
```

## Known Limitations and Future Enhancements

### Current Limitations
1. **Single Account:** Only supports one Fastmail account at a time
2. **Memory Usage:** Loads all emails into memory for processing
3. **No Persistent Storage:** No database for tracking operations
4. **Bearer Token Authentication:** Uses API tokens instead of OAuth2

### Potential Enhancements
1. **Multi-Account Support:** Support multiple email accounts
2. **Database Integration:** SQLite for operation history and caching
3. **Advanced Filters:** Date ranges, sender whitelist, size limits
4. **Batch Operations:** Process large inboxes in chunks
5. **Email Preview:** Full email content preview before archiving
6. **Undo Functionality:** Restore recently archived emails
7. **Statistics Dashboard:** Email cleanup metrics and reports
8. **API Rate Limiting:** Respect JMAP API rate limits
9. **OAuth2 Support:** Modern authentication flow

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
6. **Testing Ready:** Structured for unit test implementation

### Code Style
- **Naming:** Clear, descriptive variable and function names
- **Documentation:** Comprehensive comments and documentation
- **Formatting:** Standard `gofmt` formatting
- **Imports:** Organized standard, external, and internal imports
- **Error Messages:** User-friendly error messages with context

## Maintenance Notes

### Regular Maintenance Tasks
1. **Dependency Updates:** Keep Go modules up to date
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