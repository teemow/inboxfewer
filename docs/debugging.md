# Debugging the MCP Server

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

- `gmail_list_threads` - List Gmail threads
- `gmail_archive_thread` - Archive a thread
- `gmail_classify_thread` - Classify if GitHub-related
- `gmail_check_stale` - Check if stale (closed issue/PR)
- `gmail_archive_stale_threads` - Archive all stale threads

## REPL Examples

```javascript
// List tools
tools

// Get tool details
tool gmail_list_threads

// Execute tool
exec gmail_list_threads '{"query": "in:inbox", "maxResults": 5}'
```
