// Package server provides the MCP server context, session management,
// and OAuth-enabled HTTP server for the inboxfewer application.
//
// # Key Components
//
// ServerContext manages Google API clients with lazy initialization and caching.
// It supports multiple accounts and can use different token providers:
//   - FileTokenProvider: For STDIO transport, reads tokens from disk
//   - OAuth TokenProvider: For HTTP/SSE transport, manages tokens via OAuth flow
//
// OAuthHTTPServer wraps an MCP server with OAuth 2.1 authentication:
//   - Authorization Server Metadata (RFC 8414)
//   - Protected Resource Metadata (RFC 9728)
//   - Dynamic Client Registration (RFC 7591)
//   - Token Revocation (RFC 7009)
//   - Token Introspection (RFC 7662)
//
// SessionIDManager handles multi-account session tracking for HTTP transport.
// It maps Bearer tokens to Google accounts, enabling multiple users to share
// a single MCP server instance.
//
// # Security Features
//
// The OAuth server includes security-focused defaults:
//   - HTTPS required for production (localhost exempt for development)
//   - PKCE required (OAuth 2.1 compliance)
//   - State parameter required for CSRF protection
//   - Rate limiting per IP and per authenticated user
//   - Optional token encryption at rest (AES-256-GCM)
//   - Security headers on all HTTP responses
//   - Audit logging for authentication events
package server
