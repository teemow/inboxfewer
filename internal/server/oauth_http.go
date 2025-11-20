package server

import (
	"context"
	"fmt"
	"net/http"
	"time"

	mcpserver "github.com/mark3labs/mcp-go/server"
	"github.com/teemow/inboxfewer/internal/mcp/oauth"
)

// OAuthHTTPServer wraps an MCP server with OAuth 2.1 authentication
// It implements RFC 9728 Protected Resource Metadata for MCP clients to discover
// Google as the authorization server
type OAuthHTTPServer struct {
	mcpServer    *mcpserver.MCPServer
	oauthHandler *oauth.Handler
	httpServer   *http.Server
	serverType   string // "sse" or "streamable-http"
}

// NewOAuthHTTPServer creates a new OAuth-enabled HTTP server for MCP
func NewOAuthHTTPServer(mcpServer *mcpserver.MCPServer, serverType string, baseURL string) (*OAuthHTTPServer, error) {
	// Create OAuth handler with Google as the authorization server
	oauthConfig := &oauth.Config{
		Resource: baseURL,
		SupportedScopes: []string{
			"https://www.googleapis.com/auth/gmail.readonly",
			"https://www.googleapis.com/auth/gmail.modify",
			"https://www.googleapis.com/auth/gmail.send",
			"https://www.googleapis.com/auth/gmail.settings.basic",
			"https://www.googleapis.com/auth/documents.readonly",
			"https://www.googleapis.com/auth/drive",
			"https://www.googleapis.com/auth/calendar",
			"https://www.googleapis.com/auth/meetings.space.readonly",
			"https://www.googleapis.com/auth/tasks",
		},
	}

	oauthHandler, err := oauth.NewHandler(oauthConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create OAuth handler: %w", err)
	}

	return &OAuthHTTPServer{
		mcpServer:    mcpServer,
		oauthHandler: oauthHandler,
		serverType:   serverType,
	}, nil
}

// Start starts the OAuth-enabled HTTP server
func (s *OAuthHTTPServer) Start(addr string) error {
	mux := http.NewServeMux()

	// Register OAuth endpoints
	// Protected Resource Metadata endpoint (RFC 9728)
	// This tells MCP clients where to find the authorization server (Google)
	mux.HandleFunc("/.well-known/oauth-protected-resource", s.oauthHandler.ServeProtectedResourceMetadata)

	// Register MCP endpoints based on server type
	switch s.serverType {
	case "sse":
		// Create SSE server
		sseServer := mcpserver.NewSSEServer(s.mcpServer,
			mcpserver.WithSSEEndpoint("/sse"),
			mcpserver.WithMessageEndpoint("/message"),
		)

		// Wrap SSE endpoints with OAuth middleware
		mux.Handle("/sse", s.oauthHandler.ValidateGoogleToken(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Get the underlying SSE handler
			// Note: This is a simplified approach. We may need to access the internal handler differently
			sseServer.ServeHTTP(w, r)
		})))

		mux.Handle("/message", s.oauthHandler.ValidateGoogleToken(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			sseServer.ServeHTTP(w, r)
		})))

	case "streamable-http":
		// Create Streamable HTTP server
		httpServer := mcpserver.NewStreamableHTTPServer(s.mcpServer,
			mcpserver.WithEndpointPath("/mcp"),
		)

		// Wrap MCP endpoint with OAuth middleware
		mux.Handle("/mcp", s.oauthHandler.ValidateGoogleToken(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			httpServer.ServeHTTP(w, r)
		})))

	default:
		return fmt.Errorf("unsupported server type: %s", s.serverType)
	}

	// Create HTTP server
	s.httpServer = &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	// Start server
	return s.httpServer.ListenAndServe()
}

// Shutdown gracefully shuts down the server
func (s *OAuthHTTPServer) Shutdown(ctx context.Context) error {
	if s.httpServer != nil {
		return s.httpServer.Shutdown(ctx)
	}
	return nil
}

// GetOAuthHandler returns the OAuth handler for testing or direct access
func (s *OAuthHTTPServer) GetOAuthHandler() *oauth.Handler {
	return s.oauthHandler
}
