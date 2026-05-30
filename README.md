# Mailbox Zero - Email Cleanup Helper

[![Test Suite](https://github.com/taskinen/mailboxzero/actions/workflows/test.yml/badge.svg)](https://github.com/taskinen/mailboxzero/actions/workflows/test.yml)
[![Go Version](https://img.shields.io/badge/Go-1.26+-00ADD8?style=flat&logo=go)](https://go.dev/)
[![Go Report Card](https://goreportcard.com/badge/github.com/taskinen/mailboxzero)](https://goreportcard.com/report/github.com/taskinen/mailboxzero)
[![Made with Go](https://img.shields.io/badge/Made%20with-Go-1f425f.svg)](https://go.dev/)

A Go-based web application that helps you clean up your Fastmail inbox by finding and archiving similar emails using JMAP protocol.

<img width="1457" height="1169" alt="SCR-20250827-psxl" src="https://github.com/user-attachments/assets/2f1fe630-5ba7-4c8a-a610-6f32388654b5" />

## Features

- **Safe Operations**: Built-in dry run mode prevents accidental changes
- **Dual-pane Interface**: View inbox on left, grouped similar emails on right
- **Smart Similarity Matching**: Fuzzy matching based on subject, sender, and email content
- **Adjustable Similarity Threshold**: Fine-tune matching with a percentage slider
- **Selective Archiving**: Choose which emails to archive with confirmation dialog
- **Rapid Archive Loop**: "Archive & Find Next" button archives the current selection without confirmation and immediately re-runs the similarity search
- **Undo Archive**: After each successful archive a status pill with an "Undo" button appears in the top-right of the action bar to restore the just-archived messages
- **Individual Email Selection**: Select specific emails to find similar matches
- **Mock Mode**: Run against built-in sample data — no Fastmail credentials required

## Safety Features

- **DRY RUN MODE**: All write operations are disabled by default
- **Archive Only**: The only write operation is moving emails to archive (never deletes)
- **Confirmation Dialog**: Required before archiving via "Archive Selected" (the opt-in "Archive & Find Next" button intentionally skips it)
- **Undo Archive**: An "Undo" button appears after each successful archive and moves the affected messages back to the inbox via `POST /api/unarchive`
- **Visual Warnings**: Clear indication when in dry run mode

## Setup

### Prerequisites

- Go 1.26 or later
- Fastmail account with JMAP access
- Fastmail API token (generated from account settings)

### Installation

1. Clone/download the project
2. Install dependencies:
   ```bash
   go mod download
   ```
3. Create your configuration file:
   ```bash
   cp config.yaml.example config.yaml
   ```

4. Edit `config.yaml` with your Fastmail API token:
   ```yaml
   jmap:
     endpoint: "https://api.fastmail.com/jmap/session"
     api_token: "your-api-token-here"
   
   # IMPORTANT: Set to false only when ready for real changes
   dry_run: true
   ```

### Getting Fastmail API Token

1. Log into your Fastmail account
2. Go to Settings → Privacy & Security → Integrations
3. Click "New API Token"
4. Set the scope to "Mail" access
5. Generate the token and copy it to your config file

### Running the Application

1. Start the server:
   ```bash
   go run main.go
   ```

2. Open your browser to: http://localhost:8080

3. The application will display a warning banner when in dry run mode

### Mock Mode (no Fastmail account required)

For development, testing, or demos you can run against built-in sample data instead of a real Fastmail inbox:

```bash
cp config-mock.yaml.example config.yaml
go run main.go
```

Mock mode is enabled by setting `mock_mode: true` in the config; when on, the JMAP endpoint and API token are not required. Keep `dry_run: true` here too — archive operations simulate against the in-memory sample emails.

## Usage

### Basic Workflow

1. **Load Inbox**: The left pane shows your current inbox emails
2. **Find Similar Emails**: 
   - Click "Find Similar Emails" to find all similar email groups
   - Or select a specific email and click "Find Similar Emails" to find matches for that email
3. **Adjust Similarity**: Use the percentage slider to fine-tune matching sensitivity
4. **Review Matches**: Similar emails appear in the right pane with checkboxes
5. **Select for Archiving**: Choose which emails to archive (all selected by default)
6. **Archive**: Click "Archive Selected" and confirm to move emails to archive folder, or click "Archive & Find Next" to archive immediately without confirmation and automatically search for the next batch of similar emails

### Key Features

- **Similarity Slider**: Adjust from 0-100% to control how strict the matching should be
- **Select All/None**: Quickly select or deselect all found similar emails
- **Individual Selection**: Click on specific emails to select/deselect them
- **Clear Results**: Remove all results from the right pane to start fresh

### Enabling Real Changes

**⚠️ WARNING**: Only disable dry run mode when you're confident the application works correctly with your mailbox.

1. Edit `config.yaml`:
   ```yaml
   dry_run: false
   ```

2. Restart the application
3. The warning banner will disappear
4. Archive operations will now actually move emails

## Configuration Options

```yaml
server:
  port: 8080              # Web server port
  host: "localhost"       # Web server host

jmap:
  endpoint: "https://api.fastmail.com/jmap/session"
  api_token: ""           # Your Fastmail API token

dry_run: true             # Safety feature - set to false to enable changes
default_similarity: 60    # Default similarity percentage (0-100)
mock_mode: false          # Set to true to use built-in sample data (no JMAP needed)
```

## How Similarity Matching Works

The application uses weighted scoring tuned for newsletter and notification clustering:

- **Sender Similarity (50%)** — Structured ladder, not raw string distance. Same full address → 1.0; same domain → 0.8 (suppressed for shared-ESP domains like `sendgrid.net` unless local parts also match); same registrable root domain (e.g. `mail.google.com` ↔ `accounts.google.com`) → 0.7; same specific display name → 0.6.
- **Subject Similarity (30%)** — `max(token Jaccard, Levenshtein)` over a normalized subject with reply/forward prefixes (`Re:`, `Fwd:`, `[list]`) stripped. Common newsletter stop words (`weekly`, `newsletter`, `update`, etc.) are filtered before Jaccard.
- **Content Similarity (20%)** — Jaccard overlap of normalized word tokens (length ≥ 3) from preview/body.

Clusters are formed by single-link expansion: an email joins a group if it matches **any** existing member at the threshold, not just the seed — so a sibling that diverges from the group's first email but matches another member still gets pulled in.

## Security Considerations

- **API Tokens**: Use Fastmail API tokens for secure authentication
- **Local Only**: All processing happens locally - no data sent to external servers
- **Read-Heavy**: Only reads email data, minimal write operations
- **Archive Only**: Never deletes emails, only moves them to archive

## Troubleshooting

### Authentication Errors
- Verify your Fastmail API token is correct
- Ensure JMAP is enabled in your Fastmail account
- Check network connectivity

### No Emails Found
- Verify you have emails in your inbox
- Check if your account has the expected mailbox structure

### Similarity Issues
- Try adjusting the similarity threshold
- Some emails may have very little content to compare
- Ensure emails have sufficient text in subject/preview

## Development

The project structure:
```
mailboxzero/
├── main.go                 # Application entry point
├── config.yaml            # Configuration file
├── internal/
│   ├── config/            # Configuration handling
│   ├── jmap/              # JMAP client implementation
│   ├── server/            # Web server and API handlers
│   └── similarity/        # Email similarity algorithms
└── web/
    ├── templates/         # HTML templates
    └── static/           # CSS and JavaScript files
```

### Running Tests

The project includes comprehensive unit tests for all packages:

```bash
# Run all tests
go test ./...

# Run tests with verbose output
go test ./... -v

# Run tests with coverage
go test ./... -cover

# Generate coverage report
go test ./... -coverprofile=coverage.out
go tool cover -html=coverage.out
```

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.
