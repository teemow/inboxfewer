package server

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"time"

	mcpserver "github.com/mark3labs/mcp-go/server"

	"github.com/teemow/inboxfewer/internal/mcp/oauth_library"
)

// OAuthConfig holds configuration for OAuth server creation
type OAuthConfig struct {
	BaseURL            string
	GoogleClientID     string
	GoogleClientSecret string
	DisableStreaming   bool

	// Security Settings (secure by default)
	// See oauth_library.Config for detailed documentation
	AllowPublicClientRegistration bool   // Default: false (requires registration token)
	RegistrationAccessToken       string // Required if AllowPublicClientRegistration=false
	AllowInsecureAuthWithoutState bool   // Default: false (state parameter required)
	MaxClientsPerIP               int    // Default: 10 (prevents DoS)
}

// OAuthHTTPServerLibrary wraps an MCP server with OAuth 2.1 authentication using the mcp-oauth library
type OAuthHTTPServerLibrary struct {
	mcpServer        *mcpserver.MCPServer
	oauthHandler     *oauth_library.Handler
	httpServer       *http.Server
	serverType       string // "streamable-http"
	disableStreaming bool
}

// buildOAuthLibraryConfig converts OAuthConfig to oauth_library.Config
// This eliminates code duplication between NewOAuthHTTPServerLibrary and CreateOAuthHandlerLibrary
func buildOAuthLibraryConfig(config OAuthConfig) *oauth_library.Config {
	return &oauth_library.Config{
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
}

// NewOAuthHTTPServerLibrary creates a new OAuth-enabled HTTP server using the mcp-oauth library
func NewOAuthHTTPServerLibrary(mcpServer *mcpserver.MCPServer, serverType string, config OAuthConfig) (*OAuthHTTPServerLibrary, error) {
	oauthHandler, err := oauth_library.NewHandler(buildOAuthLibraryConfig(config))
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
	return oauth_library.NewHandler(buildOAuthLibraryConfig(config))
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
