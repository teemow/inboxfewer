#!/bin/bash

# Start the inboxfewer MCP server for debugging
# Usage: ./scripts/start-mcp-server.sh [--restart]

set -e

HTTP_PORT="8080"
BINARY="./inboxfewer"

# Check for restart flag
RESTART=false
if [ "$1" = "--restart" ]; then
    RESTART=true
fi

# Build the binary
echo "Building inboxfewer..."
go build -o "$BINARY" .

# Kill existing server if restart requested or port is in use
if [ "$RESTART" = true ] || lsof -Pi :${HTTP_PORT} -sTCP:LISTEN -t >/dev/null 2>&1; then
    echo "Stopping existing server on port ${HTTP_PORT}..."
    lsof -ti :${HTTP_PORT} | xargs kill 2>/dev/null || true
    sleep 1
fi

# Start the server
echo "Starting inboxfewer MCP server on http://localhost:${HTTP_PORT}/mcp"
echo "Press Ctrl+C to stop"
echo ""
exec "$BINARY" serve --transport streamable-http --http-addr ":${HTTP_PORT}" --debug
