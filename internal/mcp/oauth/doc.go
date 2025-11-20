// Package oauth provides OAuth 2.1 authentication for the MCP server.
//
// This package implements OAuth 2.1 authentication according to the Model Context Protocol
// (MCP) specification dated 2025-06-18. It provides secure authentication for MCP server
// endpoints, ensuring that only authorized clients can access the server's capabilities.
//
// Key Features:
//   - OAuth 2.1 compliance with PKCE (Proof Key for Code Exchange)
//   - Dynamic Client Registration (RFC 7591)
//   - Authorization Server Metadata discovery (RFC 8414)
//   - Resource Indicators (RFC 8707) for audience binding
//   - Secure in-memory token storage
//   - Token expiration and validation
//   - Support for both confidential and public clients
//
// The package is designed to be used by the MCP server's HTTP and SSE transports
// to add OAuth protection to their endpoints.
//
// Example usage:
//
//	// Create OAuth handler
//	handler, err := oauth.NewHandler(config)
//	if err != nil {
//		log.Fatal(err)
//	}
//
//	// Protect MCP endpoints
//	http.Handle("/mcp", handler.ValidateToken(mcpHandler))
package oauth
