# inboxfewer

Archives Gmail threads for closed GitHub issues and pull requests.

## Features

- **Gmail Integration**: Automatically archives emails related to closed GitHub issues and PRs
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

### Gmail OAuth

On first run, you'll be prompted to authenticate with Gmail. The OAuth token will be cached at:
- Linux/Unix: `~/.cache/inboxfewer/gmail.token`
- macOS: `~/Library/Caches/inboxfewer/gmail.token`
- Windows: `%TEMP%/inboxfewer/gmail.token`

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
│   │   ├── classifier.go  # Thread classification
│   │   └── types.go       # GitHub issue/PR types
│   ├── server/            # MCP server context
│   │   └── context.go     # Server context management
│   └── tools/             # MCP tool implementations
│       └── gmail_tools/   # Gmail-related MCP tools
│           └── tools.go
├── main.go                # Application entry point
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
