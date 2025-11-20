// Package oauth provides OAuth 2.1 authentication for the MCP server.
//
// This package implements OAuth 2.1 authentication according to the Model Context Protocol
// (MCP) specification dated 2025-06-18. The MCP server acts as an OAuth 2.1 Resource Server,
// using Google as the Authorization Server for secure authentication.
//
// Architecture:
//   - MCP Server: OAuth 2.1 Resource Server (validates Google tokens)
//   - Google: OAuth 2.1 Authorization Server (issues tokens, handles user auth)
//   - MCP Client: OAuth 2.1 Client (handles OAuth flow, includes tokens in requests)
//
// Key Features:
//   - Protected Resource Metadata (RFC 9728) - advertises Google as authorization server
//   - Bearer token validation via Google's userinfo endpoint
//   - Automatic token refresh (when Google OAuth credentials provided)
//   - Rate limiting per IP address with token bucket algorithm
//   - Secure in-memory token caching
//   - Token expiration handling with automatic cleanup
//   - Integration with Google APIs (Gmail, Drive, Calendar, etc.)
//
// The package is designed to be used by the MCP server's HTTP and SSE transports
// to add OAuth protection to their endpoints.
//
// Example usage:
//
//	// Create OAuth handler with Google as authorization server
//	handler, err := oauth.NewHandler(&oauth.Config{
//		Resource: "https://mcp.example.com",
//		SupportedScopes: []string{
//			"https://www.googleapis.com/auth/gmail.readonly",
//			// ... other Google scopes
//		},
//		GoogleClientID:     os.Getenv("GOOGLE_CLIENT_ID"),
//		GoogleClientSecret: os.Getenv("GOOGLE_CLIENT_SECRET"),
//		RateLimitRate:      10,
//		RateLimitBurst:     20,
//	})
//	if err != nil {
//		log.Fatal(err)
//	}
//
//	// Protect MCP endpoints
//	http.Handle("/mcp", handler.ValidateGoogleToken(mcpHandler))
//
//	// Serve Protected Resource Metadata
//	http.HandleFunc("/.well-known/oauth-protected-resource",
//		handler.ServeProtectedResourceMetadata)
package oauth
