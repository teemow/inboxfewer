# Debugging the MCP Server

This guide explains how to debug inboxfewer using [giantswarm/mcp-debug](https://github.com/giantswarm/mcp-debug), a powerful debugging tool for MCP servers.

## Prerequisites

Install mcp-debug:
```bash
go install github.com/giantswarm/mcp-debug@latest
```

## Quick Start

1. **Start the inboxfewer MCP server:**
   ```bash
   ./scripts/start-mcp-server.sh
   ```
   This runs on `http://localhost:8080/mcp`

2. **Use mcp-debug to connect:**
   ```bash
   # Interactive REPL
   mcp-debug --repl --endpoint http://localhost:8080/mcp
   
   # Or via Cursor - add to .cursor/mcp.json:
   # {
   #   "mcpServers": {
   #     "mcp-debug": {
   #       "command": "mcp-debug",
   #       "args": ["--mcp-server", "--endpoint", "http://localhost:8080/mcp"]
   #     }
   #   }
   # }
   ```

## Development Workflow

When making changes to inboxfewer:

```bash
# 1. Make your code changes
# 2. Rebuild and restart
./scripts/start-mcp-server.sh --restart
```

This will:
- Build the binary with `go build`
- Kill any running inboxfewer server
- Start the new server

## Available Tools

For a complete list of all available tools (Gmail, Calendar, Docs, Drive, Meet, Tasks, and OAuth), see [tools.md](tools.md).

**Quick examples:**
- `gmail_list_threads` - List Gmail threads
- `gmail_archive_thread` - Archive a thread
- `calendar_list_events` - List calendar events
- `docs_get_document` - Get Google Docs content
- `drive_list_files` - List files in Google Drive
- `meet_list_recordings` - List Meet recordings
- `tasks_list_tasks` - List tasks from Google Tasks

## REPL Examples

```javascript
// List tools
tools

// Get tool details
tool gmail_list_threads

// Execute tool
exec gmail_list_threads '{"query": "in:inbox", "maxResults": 5}'
```
