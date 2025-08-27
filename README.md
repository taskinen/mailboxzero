# Mailbox Zero - Email Cleanup Helper

A Go-based web application that helps you clean up your Fastmail inbox by finding and archiving similar emails using JMAP protocol.

## Features

- **Safe Operations**: Built-in dry run mode prevents accidental changes
- **Dual-pane Interface**: View inbox on left, grouped similar emails on right
- **Smart Similarity Matching**: Fuzzy matching based on subject, sender, and email content
- **Adjustable Similarity Threshold**: Fine-tune matching with a percentage slider
- **Selective Archiving**: Choose which emails to archive with confirmation dialog
- **Individual Email Selection**: Select specific emails to find similar matches

## Safety Features

- **DRY RUN MODE**: All write operations are disabled by default
- **Archive Only**: The only write operation is moving emails to archive (never deletes)
- **Confirmation Dialog**: Requires confirmation before archiving
- **Visual Warnings**: Clear indication when in dry run mode

## Setup

### Prerequisites

- Go 1.21 or later
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

## Usage

### Basic Workflow

1. **Load Inbox**: The left pane shows your current inbox emails
2. **Find Similar Emails**: 
   - Click "Find Similar Emails" to find all similar email groups
   - Or select a specific email and click "Find Similar Emails" to find matches for that email
3. **Adjust Similarity**: Use the percentage slider to fine-tune matching sensitivity
4. **Review Matches**: Similar emails appear in the right pane with checkboxes
5. **Select for Archiving**: Choose which emails to archive (all selected by default)
6. **Archive**: Click "Archive Selected" and confirm to move emails to archive folder

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
default_similarity: 75    # Default similarity percentage (0-100)
```

## How Similarity Matching Works

The application uses fuzzy matching with weighted scoring:

- **Subject Similarity** (40%): Compares email subjects using Levenshtein distance
- **Sender Similarity** (40%): Compares sender email addresses
- **Content Similarity** (20%): Compares email preview/body content

Additional boosters:
- Common words in subjects increase similarity
- Normalized text (lowercase, punctuation removed) for better matching

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

## License

This project is provided as-is for educational and personal use.