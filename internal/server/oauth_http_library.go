package server

import (
	"context"
	"fmt"
	"net/http"
	"time"

	mcpserver "github.com/mark3labs/mcp-go/server"

	"github.com/teemow/inboxfewer/internal/mcp/oauth_library"
)

// OAuthHTTPServerLibrary wraps an MCP server with OAuth 2.1 authentication using the mcp-oauth library
type OAuthHTTPServerLibrary struct {
	mcpServer        *mcpserver.MCPServer
	oauthHandler     *oauth_library.Handler
	httpServer       *http.Server
	serverType       string // "streamable-http"
	disableStreaming bool
}

// NewOAuthHTTPServerLibrary creates a new OAuth-enabled HTTP server using the mcp-oauth library
func NewOAuthHTTPServerLibrary(mcpServer *mcpserver.MCPServer, serverType string, config OAuthConfig) (*OAuthHTTPServerLibrary, error) {
	// Create OAuth handler using the library
	libraryConfig := &oauth_library.Config{
		BaseURL:            config.BaseURL,
		GoogleClientID:     config.GoogleClientID,
		GoogleClientSecret: config.GoogleClientSecret,
		Security: oauth_library.SecurityConfig{
			AllowPublicClientRegistration: config.AllowPublicClientRegistration,
			RegistrationAccessToken:       config.RegistrationAccessToken,
			AllowInsecureAuthWithoutState: config.AllowInsecureAuthWithoutState,
			MaxClientsPerIP:               config.MaxClientsPerIP,
			EnableAuditLogging:            true, // Always enable audit logging
		},
		RateLimit: oauth_library.RateLimitConfig{
			Rate:      10,  // 10 req/sec per IP
			Burst:     20,  // Allow burst of 20
			UserRate:  100, // 100 req/sec per authenticated user
			UserBurst: 200, // Allow burst of 200
		},
	}

	oauthHandler, err := oauth_library.NewHandler(libraryConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create OAuth handler: %w", err)
	}

	return &OAuthHTTPServerLibrary{
		mcpServer:        mcpServer,
		oauthHandler:     oauthHandler,
		serverType:       serverType,
		disableStreaming: config.DisableStreaming,
	}, nil
}

// CreateOAuthHandlerLibrary creates an OAuth handler using the library for use with HTTP transport
// This allows creating the handler before the server to inject the token provider
func CreateOAuthHandlerLibrary(config OAuthConfig) (*oauth_library.Handler, error) {
	libraryConfig := &oauth_library.Config{
		BaseURL:            config.BaseURL,
		GoogleClientID:     config.GoogleClientID,
		GoogleClientSecret: config.GoogleClientSecret,
		Security: oauth_library.SecurityConfig{
			AllowPublicClientRegistration: config.AllowPublicClientRegistration,
			RegistrationAccessToken:       config.RegistrationAccessToken,
			AllowInsecureAuthWithoutState: config.AllowInsecureAuthWithoutState,
			MaxClientsPerIP:               config.MaxClientsPerIP,
			EnableAuditLogging:            true, // Always enable audit logging
		},
		RateLimit: oauth_library.RateLimitConfig{
			Rate:      10,  // 10 req/sec per IP
			Burst:     20,  // Allow burst of 20
			UserRate:  100, // 100 req/sec per authenticated user
			UserBurst: 200, // Allow burst of 200
		},
	}

	return oauth_library.NewHandler(libraryConfig)
}

// NewOAuthHTTPServerLibraryWithHandler creates a new OAuth-enabled HTTP server with an existing handler
func NewOAuthHTTPServerLibraryWithHandler(mcpServer *mcpserver.MCPServer, serverType string, oauthHandler *oauth_library.Handler, disableStreaming bool) (*OAuthHTTPServerLibrary, error) {
	return &OAuthHTTPServerLibrary{
		mcpServer:        mcpServer,
		oauthHandler:     oauthHandler,
		serverType:       serverType,
		disableStreaming: disableStreaming,
	}, nil
}

// Start starts the OAuth-enabled HTTP server
func (s *OAuthHTTPServerLibrary) Start(addr string) error {
	// Validate HTTPS requirement for OAuth 2.1
	baseURL := s.oauthHandler.GetServer().Config.Issuer
	if err := validateHTTPSRequirement(baseURL); err != nil {
		return err
	}

	mux := http.NewServeMux()

	// Get the library's HTTP handler
	libHandler := s.oauthHandler.GetHandler()

	// ========== OAuth 2.1 Endpoints ==========

	// Protected Resource Metadata endpoint (RFC 9728)
	mux.HandleFunc("/.well-known/oauth-protected-resource", libHandler.ServeProtectedResourceMetadata)

	// Authorization Server Metadata endpoint (RFC 8414)
	mux.HandleFunc("/.well-known/oauth-authorization-server", libHandler.ServeAuthorizationServerMetadata)

	// Dynamic Client Registration endpoint (RFC 7591)
	mux.HandleFunc("/oauth/register", libHandler.ServeClientRegistration)

	// OAuth Authorization endpoint
	mux.HandleFunc("/oauth/authorize", libHandler.ServeAuthorization)

	// OAuth Token endpoint
	mux.HandleFunc("/oauth/token", libHandler.ServeToken)

	// OAuth Callback endpoint (from provider)
	mux.HandleFunc("/oauth/callback", libHandler.ServeCallback)

	// Token Revocation endpoint (RFC 7009)
	mux.HandleFunc("/oauth/revoke", libHandler.ServeTokenRevocation)

	// Token Introspection endpoint (RFC 7662) - new feature from library
	mux.HandleFunc("/oauth/introspect", libHandler.ServeTokenIntrospection)

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

		// Wrap MCP endpoint with OAuth middleware
		mcpHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			httpServer.ServeHTTP(w, r)
		})
		mux.Handle("/mcp", libHandler.ValidateToken(mcpHandler))

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
func (s *OAuthHTTPServerLibrary) Shutdown(ctx context.Context) error {
	// Stop the OAuth handler's background services
	if s.oauthHandler != nil {
		s.oauthHandler.Stop()
	}

	// Shutdown HTTP server
	if s.httpServer != nil {
		return s.httpServer.Shutdown(ctx)
	}
	return nil
}

// GetOAuthHandler returns the OAuth handler for testing or direct access
func (s *OAuthHTTPServerLibrary) GetOAuthHandler() *oauth_library.Handler {
	return s.oauthHandler
}
