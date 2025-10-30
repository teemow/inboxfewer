# inboxfewer

Archives Gmail threads for closed GitHub issues and pull requests.

## Features

- **Gmail Integration**: Automatically archives emails related to closed GitHub issues and PRs
- **Google Docs Integration**: Extract and retrieve Google Docs content from email messages, with full support for multi-tab documents (Oct 2024 feature)
- **MCP Server**: Provides Model Context Protocol server for AI assistant integration
- **Multiple Transports**: Supports stdio, SSE, and streamable HTTP transports
- **Flexible Usage**: Can run as a CLI tool or as an MCP server

## Installation

```bash
go install github.com/teemow/inboxfewer@latest
```

## Configuration

### GitHub Token

Create a file at `~/keys/github-inboxfewer.token` with two space-separated values:
```
<github-username> <github-personal-access-token>
```

### Google Services OAuth

On first run, you'll be prompted to authenticate with Google services (Gmail, Google Docs, Google Drive). A single unified OAuth token is used for all Google services and will be cached at:
- Linux/Unix: `~/.cache/inboxfewer/google.token`
- macOS: `~/Library/Caches/inboxfewer/google.token`
- Windows: `%TEMP%/inboxfewer/google.token`

**Note:** The OAuth token provides access to Gmail, Google Docs, and Google Drive APIs with the following scopes:
- Gmail: Read and modify messages (for archiving)
- Google Docs: Read document content
- Google Drive: Read file metadata

## Usage

### CLI Mode (Cleanup)

Archive Gmail threads related to closed GitHub issues/PRs:

```bash
# Run cleanup (default command)
inboxfewer

# Or explicitly
inboxfewer cleanup
```

### MCP Server Mode

Start the MCP server to provide Gmail/GitHub tools for AI assistants:

#### Standard I/O (Default)
```bash
inboxfewer serve
# or
inboxfewer serve --transport stdio
```

#### Server-Sent Events (SSE)
```bash
inboxfewer serve --transport sse --http-addr :8080
```

The SSE server will expose:
- SSE endpoint: `http://localhost:8080/sse`
- Message endpoint: `http://localhost:8080/message`

#### Streamable HTTP
```bash
inboxfewer serve --transport streamable-http --http-addr :8080
```

The HTTP server will expose:
- HTTP endpoint: `http://localhost:8080/mcp`

### Options

```bash
--debug           Enable debug logging
--transport       Transport type: stdio, sse, or streamable-http (default: stdio)
--http-addr       HTTP server address for sse/http transports (default: :8080)
```

## MCP Server Tools

When running as an MCP server, the following tools are available:

### OAuth Authentication Flow

Before using any Google services (Gmail, Docs, Drive), you need to authenticate:

1. **Check if authenticated:** The server will automatically check for an existing token
2. **Get authorization URL:** If not authenticated, use `google_get_auth_url` to get the OAuth URL
3. **Authorize access:** Visit the URL in your browser and grant permissions
4. **Save the code:** Copy the authorization code and use `google_save_auth_code` to save it
5. **Use the tools:** All Google-related tools will now work with the saved token

The token is stored in `~/.cache/inboxfewer/google.token` and provides access to all Google APIs (Gmail, Docs, Drive).

### Gmail Tools

#### `gmail_list_threads`
List Gmail threads matching a query.

**Arguments:**
- `query` (required): Gmail search query (e.g., 'in:inbox', 'from:user@example.com')
- `maxResults` (optional): Maximum number of results (default: 10)

#### `gmail_archive_thread`
Archive a Gmail thread by removing it from the inbox.

**Arguments:**
- `threadId` (required): The ID of the thread to archive

#### `gmail_classify_thread`
Classify a Gmail thread to determine if it's related to GitHub issues or PRs.

**Arguments:**
- `threadId` (required): The ID of the thread to classify

#### `gmail_check_stale`
Check if a Gmail thread is stale (GitHub issue/PR is closed).

**Arguments:**
- `threadId` (required): The ID of the thread to check

#### `gmail_archive_stale_threads`
Archive all Gmail threads in inbox that are related to closed GitHub issues/PRs.

**Arguments:**
- `query` (optional): Gmail search query (default: 'in:inbox')

#### `gmail_list_attachments`
List all attachments in a Gmail message.

**Arguments:**
- `messageId` (required): The ID of the Gmail message

**Returns:** JSON array of attachment metadata including attachmentId, filename, mimeType, size, and human-readable size.

#### `gmail_get_attachment`
Get the content of an attachment from a Gmail message.

**Arguments:**
- `messageId` (required): The ID of the Gmail message
- `attachmentId` (required): The ID of the attachment
- `encoding` (optional): Encoding format - 'base64' (default) or 'text'

**Returns:** Attachment content in the specified encoding.

**Note:** Use 'text' encoding for text-based attachments (.txt, .ics, .csv, etc.) and 'base64' for binary files (.pdf, .png, .zip, etc.).

**Security:** Attachments are limited to 25MB in size.

#### `gmail_get_message_body`
Extract text or HTML body from a Gmail message.

**Arguments:**
- `messageId` (required): The ID of the Gmail message
- `format` (optional): Body format - 'text' (default) or 'html'

**Returns:** Message body content in the specified format.

**Use Case:** Useful for extracting Google Docs/Drive links from email bodies, since Google Meet notes are typically shared as links rather than attachments.

#### `gmail_extract_doc_links`
Extract Google Docs/Drive links from a Gmail message.

**Arguments:**
- `messageId` (required): The ID of the Gmail message
- `format` (optional): Body format to search - 'text' (default) or 'html'

**Returns:** JSON array of Google Docs/Drive links found in the message, including documentId, url, and type (document, spreadsheet, presentation, or drive).

**Use Case:** Extracts Google Docs, Sheets, Slides, and Drive file links from email bodies. Particularly useful for finding meeting notes shared via Google Docs links.

### Google OAuth Tools

#### `google_get_auth_url`
Get the OAuth authorization URL for Google services.

**Arguments:** None

**Returns:** Authorization URL that the user should visit to grant access to Gmail, Google Docs, and Google Drive.

**Use Case:** When the OAuth token is missing or expired, use this to get the authorization URL. After visiting the URL and authorizing access, use `google_save_auth_code` with the provided code.

#### `google_save_auth_code`
Save the OAuth authorization code to complete authentication.

**Arguments:**
- `authCode` (required): The authorization code obtained from the Google OAuth flow

**Returns:** Success message indicating the token has been saved.

**Use Case:** After visiting the authorization URL from `google_get_auth_url`, Google provides an authorization code. Pass this code to complete the OAuth flow and save the access token.

### Google Docs Tools

#### `docs_get_document`
Get Google Docs content by document ID.

**Arguments:**
- `documentId` (required): The ID of the Google Doc (extracted from URL)
- `format` (optional): Output format - 'markdown' (default), 'text', or 'json'

**Returns:** Document content in the specified format. Markdown format preserves headings, lists, formatting, and links. **Fully supports documents with multiple tabs** (introduced October 2024) - all tabs and nested child tabs are automatically fetched and included in the output.

**OAuth:** Uses the unified Google OAuth token (see Configuration section above). If not already authenticated, you'll be prompted to authorize access.

**Use Case:** Retrieve the actual content of Google Meet notes, shared documents, or any Google Doc accessible to your account. Works seamlessly with both legacy single-tab documents and new multi-tab documents.

#### `docs_get_document_metadata`
Get metadata about a Google Doc or Drive file.

**Arguments:**
- `documentId` (required): The ID of the Google Doc or Drive file

**Returns:** JSON with document metadata including id, name, mimeType, createdTime, modifiedTime, size, and owners.

**Use Case:** Get information about a document without downloading its full content.

### Workflow Example: Extracting Meeting Notes

```bash
# 1. Find emails with Google Docs links
gmail_list_threads(query: "meeting notes")

# 2. Extract doc links from an email
gmail_extract_doc_links(messageId: "msg123")
# Returns: [{"documentId": "1ABC...", "url": "https://docs.google.com/...", "type": "document"}]

# 3. Fetch the document content
docs_get_document(documentId: "1ABC...", format: "markdown")
# Returns the meeting notes in Markdown format
```

## MCP Server Configuration

### Using with Claude Desktop

Add to your Claude Desktop configuration (`~/Library/Application Support/Claude/claude_desktop_config.json` on macOS):

```json
{
  "mcpServers": {
    "inboxfewer": {
      "command": "/path/to/inboxfewer",
      "args": ["serve"]
    }
  }
}
```

### Using with Other MCP Clients

For SSE or HTTP transports, configure your MCP client to connect to:
- SSE: `http://localhost:8080/sse` (with message endpoint at `/message`)
- HTTP: `http://localhost:8080/mcp`

## Development

### Quick Start

```bash
# Clone the repository
git clone https://github.com/teemow/inboxfewer.git
cd inboxfewer

# Build the project
make build

# Run tests
make test

# See all available targets
make help
```

### Debugging

To debug the MCP server with [mcp-debug](https://github.com/giantswarm/mcp-debug):

```bash
# Start the server
./scripts/start-mcp-server.sh

# In another terminal, use mcp-debug
mcp-debug --repl --endpoint http://localhost:8080/mcp
```

For development workflow (rebuild and restart):
```bash
./scripts/start-mcp-server.sh --restart
```

See [docs/debugging.md](docs/debugging.md) for details.

### Makefile Targets

The project includes a comprehensive Makefile with the following targets:

**Development:**
- `make build` - Build the binary
- `make install` - Install the binary to GOPATH/bin
- `make clean` - Clean build artifacts
- `make run` - Run the application

**Testing:**
- `make test` - Run tests
- `make test-coverage` - Run tests with coverage report
- `make vet` - Run go vet

**Code Quality:**
- `make fmt` - Run go fmt
- `make lint` - Run golangci-lint (requires golangci-lint installed)
- `make lint-yaml` - Run YAML linter (requires yamllint installed)
- `make tidy` - Run go mod tidy
- `make check` - Run all checks (fmt, vet, test, lint-yaml)

**Release:**
- `make release-dry-run` - Test the release process without publishing (requires goreleaser)
- `make release-local` - Create a release locally (requires goreleaser)

**Multi-platform Builds:**
- `make build-linux` - Build for Linux
- `make build-darwin` - Build for macOS
- `make build-windows` - Build for Windows
- `make build-all` - Build for all platforms

### Automated Releases

The project uses GitHub Actions to automatically create releases:

1. **CI Checks** (`ci.yaml`): Runs on every PR and push to master
   - Runs tests, linting, and formatting checks
   - Validates the release process with a dry-run

2. **Auto Release** (`auto-release.yaml`): Triggers on merged PRs to master
   - Automatically increments the patch version
   - Creates a git tag
   - Runs GoReleaser to build binaries for multiple platforms
   - Publishes a GitHub release with artifacts

Releases include pre-built binaries for:
- Linux (amd64, arm64)
- macOS/Darwin (amd64, arm64)
- Windows (amd64, arm64)

### Project Structure

```
inboxfewer/
├── cmd/                    # Command implementations
│   ├── root.go            # Root command
│   ├── cleanup.go         # Cleanup command (original functionality)
│   ├── serve.go           # MCP server command
│   └── version.go         # Version command
├── internal/
│   ├── gmail/             # Gmail client and utilities
│   │   ├── client.go      # Gmail API client
│   │   ├── attachments.go # Attachment retrieval
│   │   ├── doc_links.go   # Google Docs URL extraction
│   │   ├── classifier.go  # Thread classification
│   │   └── types.go       # GitHub issue/PR types
│   ├── docs/              # Google Docs client and utilities
│   │   ├── client.go      # Google Docs API client
│   │   ├── converter.go   # Document to Markdown/text conversion
│   │   ├── types.go       # Document metadata types
│   │   └── doc.go         # Package documentation
│   ├── google/            # Unified Google OAuth2 authentication
│   │   ├── oauth.go       # OAuth token management for all Google services
│   │   └── doc.go         # Package documentation
│   ├── github/            # GitHub types and utilities
│   │   └── types.go       # GitHub issue/PR types
│   ├── server/            # MCP server context
│   │   └── context.go     # Server context management
│   └── tools/             # MCP tool implementations
│       ├── google_tools/  # Google OAuth MCP tools
│       │   ├── tools.go   # OAuth authentication tools
│       │   └── doc.go     # Package documentation
│       ├── gmail_tools/   # Gmail-related MCP tools
│       │   ├── tools.go           # Thread tools
│       │   ├── attachment_tools.go # Attachment tools
│       │   └── doc.go             # Package documentation
│       └── docs_tools/    # Google Docs MCP tools
│           ├── tools.go   # Docs retrieval tools
│           └── doc.go     # Package documentation
├── docs/                  # Documentation
│   └── debugging.md       # Debugging guide
├── scripts/               # Utility scripts
│   └── start-mcp-server.sh # Development server script
├── .github/               # GitHub Actions workflows
│   └── workflows/
│       ├── ci.yaml        # Continuous integration
│       └── auto-release.yaml # Automated releases
├── main.go                # Application entry point
├── Makefile               # Build automation
├── go.mod                 # Go module definition
└── README.md              # This file
```

### Building

```bash
go build -o inboxfewer
```

### Testing

```bash
go test ./...
```

## License

See LICENSE file for details.

## Credits

Original concept and implementation by Brad Fitzpatrick.
MCP server integration added to provide AI assistant capabilities.

## Announcement

Original announcement: https://twitter.com/bradfitz/status/652973744302919680
