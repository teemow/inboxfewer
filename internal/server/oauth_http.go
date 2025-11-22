package server

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"time"

	mcpserver "github.com/mark3labs/mcp-go/server"
	"github.com/teemow/inboxfewer/internal/mcp/oauth"
)

// OAuthHTTPServer wraps an MCP server with OAuth 2.1 authentication
// It implements RFC 9728 Protected Resource Metadata for MCP clients to discover
// Google as the authorization server
type OAuthHTTPServer struct {
	mcpServer        *mcpserver.MCPServer
	oauthHandler     *oauth.Handler
	httpServer       *http.Server
	serverType       string // "streamable-http"
	disableStreaming bool
}

// OAuthConfig holds configuration for OAuth server creation
type OAuthConfig struct {
	BaseURL            string
	GoogleClientID     string
	GoogleClientSecret string
	DisableStreaming   bool
}

// NewOAuthHTTPServer creates a new OAuth-enabled HTTP server for MCP
func NewOAuthHTTPServer(mcpServer *mcpserver.MCPServer, serverType string, config OAuthConfig) (*OAuthHTTPServer, error) {
	// Create OAuth handler with Google as the authorization server
	oauthConfig := &oauth.Config{
		Resource: config.BaseURL,
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
		GoogleClientID:     config.GoogleClientID,
		GoogleClientSecret: config.GoogleClientSecret,
		RateLimitRate:      10,              // 10 requests per second per IP
		RateLimitBurst:     20,              // Allow burst of 20 requests
		CleanupInterval:    1 * time.Minute, // Cleanup expired tokens every minute
	}

	oauthHandler, err := oauth.NewHandler(oauthConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create OAuth handler: %w", err)
	}

	return &OAuthHTTPServer{
		mcpServer:        mcpServer,
		oauthHandler:     oauthHandler,
		serverType:       serverType,
		disableStreaming: config.DisableStreaming,
	}, nil
}

// CreateOAuthHandler creates an OAuth handler for use with HTTP transport
// This allows creating the handler before the server to inject the token provider
func CreateOAuthHandler(config OAuthConfig) (*oauth.Handler, error) {
	oauthConfig := &oauth.Config{
		Resource: config.BaseURL,
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
		GoogleClientID:     config.GoogleClientID,
		GoogleClientSecret: config.GoogleClientSecret,
		RateLimitRate:      10,              // 10 requests per second per IP
		RateLimitBurst:     20,              // Allow burst of 20 requests
		CleanupInterval:    1 * time.Minute, // Cleanup expired tokens every minute
	}

	return oauth.NewHandler(oauthConfig)
}

// NewOAuthHTTPServerWithHandler creates a new OAuth-enabled HTTP server with an existing handler
func NewOAuthHTTPServerWithHandler(mcpServer *mcpserver.MCPServer, serverType string, oauthHandler *oauth.Handler, disableStreaming bool) (*OAuthHTTPServer, error) {
	return &OAuthHTTPServer{
		mcpServer:        mcpServer,
		oauthHandler:     oauthHandler,
		serverType:       serverType,
		disableStreaming: disableStreaming,
	}, nil
}

// Start starts the OAuth-enabled HTTP server
func (s *OAuthHTTPServer) Start(addr string) error {
	// Validate HTTPS requirement for OAuth 2.1
	// Exception: localhost is allowed to use HTTP for development
	config := s.oauthHandler.GetConfig()
	baseURL := config.Resource
	if err := validateHTTPSRequirement(baseURL); err != nil {
		return err
	}

	mux := http.NewServeMux()

	// ========== OAuth 2.1 Endpoints ==========

	// Protected Resource Metadata endpoint (RFC 9728)
	// This tells MCP clients where to find the authorization server (inboxfewer)
	metadataHandler := http.HandlerFunc(s.oauthHandler.ServeProtectedResourceMetadata)
	mux.Handle("/.well-known/oauth-protected-resource", s.oauthHandler.RateLimitMiddleware(metadataHandler))

	// Authorization Server Metadata endpoint (RFC 8414)
	// This tells MCP clients about inboxfewer's OAuth endpoints
	authzMetadataHandler := http.HandlerFunc(s.oauthHandler.ServeAuthorizationServerMetadata)
	mux.Handle("/.well-known/oauth-authorization-server", s.oauthHandler.RateLimitMiddleware(authzMetadataHandler))

	// Dynamic Client Registration endpoint (RFC 7591)
	// Allows MCP clients to register without pre-configured credentials
	registrationHandler := http.HandlerFunc(s.oauthHandler.ServeDynamicClientRegistration)
	mux.Handle("/oauth/register", s.oauthHandler.RateLimitMiddleware(registrationHandler))

	// OAuth Authorization endpoint
	// MCP clients redirect users here to start OAuth flow
	authorizationHandler := http.HandlerFunc(s.oauthHandler.ServeAuthorization)
	mux.Handle("/oauth/authorize", s.oauthHandler.RateLimitMiddleware(authorizationHandler))

	// OAuth Token endpoint
	// MCP clients exchange authorization codes for access tokens
	tokenHandler := http.HandlerFunc(s.oauthHandler.ServeToken)
	mux.Handle("/oauth/token", s.oauthHandler.RateLimitMiddleware(tokenHandler))

	// Google OAuth Callback endpoint
	// Google redirects here after user authentication
	callbackHandler := http.HandlerFunc(s.oauthHandler.ServeGoogleCallback)
	mux.Handle("/oauth/google/callback", s.oauthHandler.RateLimitMiddleware(callbackHandler))

	// ========== MCP Endpoints ==========

	// Register MCP endpoints based on server type
	switch s.serverType {
	case "streamable-http":
		// Create Streamable HTTP server
		var httpServer http.Handler
		if s.disableStreaming {
			httpServer = mcpserver.NewStreamableHTTPServer(s.mcpServer,
				mcpserver.WithEndpointPath("/mcp"),
				mcpserver.WithDisableStreaming(true),
			)
		} else {
			httpServer = mcpserver.NewStreamableHTTPServer(s.mcpServer,
				mcpserver.WithEndpointPath("/mcp"),
			)
		}

		// Wrap MCP endpoint with rate limiting and OAuth middleware
		mcpHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			httpServer.ServeHTTP(w, r)
		})
		mux.Handle("/mcp", s.oauthHandler.RateLimitMiddleware(
			s.oauthHandler.ValidateGoogleToken(mcpHandler)))

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

// validateHTTPSRequirement ensures OAuth 2.1 HTTPS compliance
// Allows HTTP only for loopback addresses (localhost, 127.0.0.1, ::1)
func validateHTTPSRequirement(baseURL string) error {
	if baseURL == "" {
		return fmt.Errorf("base URL cannot be empty")
	}

	// Parse URL to properly validate scheme and host
	u, err := url.Parse(baseURL)
	if err != nil {
		return fmt.Errorf("invalid base URL: %w", err)
	}

	// Allow HTTP only for loopback addresses
	if u.Scheme == "http" {
		host := u.Hostname()
		if host != "localhost" && host != "127.0.0.1" && host != "::1" {
			return fmt.Errorf("OAuth 2.1 requires HTTPS for production (got: %s). Use HTTPS or localhost for development", baseURL)
		}
	} else if u.Scheme != "https" {
		return fmt.Errorf("invalid URL scheme: %s. Must be http (localhost only) or https", u.Scheme)
	}

	return nil
}
