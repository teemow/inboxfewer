// Package google provides OAuth2 authentication and token management for Google APIs.
//
// This package handles both file-based token storage (for STDIO transport) and
// OAuth store-based token management (for HTTP/SSE transports with OAuth authentication).
//
// The TokenProvider interface allows different token sources to be plugged in,
// enabling seamless integration between MCP OAuth authentication and Google API access.
package google
