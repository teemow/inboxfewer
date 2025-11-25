// Package oauth provides adapters for integrating the github.com/giantswarm/mcp-oauth
// library with the inboxfewer MCP server.
//
// This package bridges the mcp-oauth library with our existing server architecture,
// providing token provider integration and configuration mapping.
//
// Dependency Security Note:
// This package depends on github.com/giantswarm/mcp-oauth v0.1.26 for OAuth 2.1 implementation.
// The library provides: PKCE enforcement, refresh token rotation, rate limiting, and audit logging.
// Security posture: Actively maintained, implements OAuth 2.1 specification.
// Action required: Monitor https://github.com/giantswarm/mcp-oauth for security updates.
package oauth
