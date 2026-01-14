package cmd

import (
	"context"
	"encoding/base64"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	mcpserver "github.com/mark3labs/mcp-go/server"

	"github.com/teemow/inboxfewer/internal/mcp/oauth"
	"github.com/teemow/inboxfewer/internal/resources"
	"github.com/teemow/inboxfewer/internal/server"
	"github.com/teemow/inboxfewer/internal/tools/calendar_tools"
	"github.com/teemow/inboxfewer/internal/tools/docs_tools"
	"github.com/teemow/inboxfewer/internal/tools/drive_tools"
	"github.com/teemow/inboxfewer/internal/tools/gmail_tools"
	"github.com/teemow/inboxfewer/internal/tools/meet_tools"
	"github.com/teemow/inboxfewer/internal/tools/tasks_tools"
)

// OAuthSecurityConfig holds OAuth security settings
type OAuthSecurityConfig struct {
	AllowPublicClientRegistration bool
	RegistrationAccessToken       string
	AllowInsecureAuthWithoutState bool
	MaxClientsPerIP               int
	EncryptionKey                 []byte

	// Interstitial page branding
	InterstitialLogoURL            string
	InterstitialLogoAlt            string
	InterstitialTitle              string
	InterstitialMessage            string
	InterstitialButtonText         string
	InterstitialPrimaryColor       string
	InterstitialBackgroundGradient string

	// Redirect URI Security (mcp-oauth v0.2.30+)
	DisableProductionMode              bool
	AllowLocalhostRedirectURIs         bool
	AllowPrivateIPRedirectURIs         bool
	AllowLinkLocalRedirectURIs         bool
	DisableDNSValidation               bool
	DisableDNSValidationStrict         bool
	DisableAuthorizationTimeValidation bool

	// Trusted scheme registration for Cursor/VSCode compatibility (mcp-oauth v0.2.30+)
	TrustedPublicRegistrationSchemes []string
	DisableStrictSchemeMatching      bool

	// CIMD (Client ID Metadata Documents) per MCP 2025-11-25 (mcp-oauth v0.2.30+)
	EnableCIMD bool

	// TLS/HTTPS support
	TLSCertFile string
	TLSKeyFile  string

	// Storage configuration (mcp-oauth v0.2.30+)
	Storage OAuthStorageConfig
}

// OAuthStorageConfig holds OAuth token storage backend configuration
type OAuthStorageConfig struct {
	// Type is the storage backend type: "memory" or "valkey" (default: "memory")
	Type string

	// Valkey configuration (used when Type is "valkey")
	Valkey ValkeyStorageConfig
}

// ValkeyStorageConfig holds configuration for Valkey storage backend
type ValkeyStorageConfig struct {
	// URL is the Valkey server address (e.g., "valkey.namespace.svc:6379")
	URL string

	// Password is the optional password for Valkey authentication
	Password string

	// TLSEnabled enables TLS for Valkey connections
	TLSEnabled bool

	// KeyPrefix is the prefix for all Valkey keys (default: "mcp:")
	KeyPrefix string

	// DB is the Valkey database number (default: 0)
	DB int
}

func newServeCmd() *cobra.Command {
	var (
		debugMode          bool
		transport          string
		httpAddr           string
		yolo               bool
		googleClientID     string
		googleClientSecret string
		disableStreaming   bool
		baseURL            string
		// OAuth Security Settings
		allowPublicClientRegistration bool
		registrationAccessToken       string
		allowInsecureAuthWithoutState bool
		maxClientsPerIP               int
		encryptionKey                 string
		// Redirect URI Security Settings (mcp-oauth v0.2.30+)
		disableProductionMode              bool
		allowLocalhostRedirectURIs         bool
		allowPrivateIPRedirectURIs         bool
		allowLinkLocalRedirectURIs         bool
		disableDNSValidation               bool
		disableDNSValidationStrict         bool
		disableAuthorizationTimeValidation bool
		// Trusted scheme registration for Cursor/VSCode (mcp-oauth v0.2.30+)
		trustedPublicRegistrationSchemes []string
		disableStrictSchemeMatching      bool
		// CIMD (Client ID Metadata Documents) per MCP 2025-11-25 (mcp-oauth v0.2.30+)
		enableCIMD bool
		// TLS/HTTPS support
		tlsCertFile string
		tlsKeyFile  string
		// OAuth storage options (mcp-oauth v0.2.30+)
		oauthStorageType string
		valkeyURL        string
		valkeyPassword   string
		valkeyTLS        bool
		valkeyKeyPrefix  string
		valkeyDB         int
	)

	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Start the MCP server",
		Long: `Start the Model Context Protocol (MCP) server to provide Gmail and GitHub
integration tools for AI assistants.

Supports multiple transport types:
  - stdio: Standard input/output (default)
  - streamable-http: Streamable HTTP transport

Safety Mode:
  By default, the server operates in read-only mode, providing only safe operations.
  Use --yolo to enable write operations (email sending, file deletion, etc.)

OAuth Configuration:
  HTTP Transport:
    Base URL (required for deployed instances):
      --base-url https://your-domain.com OR MCP_BASE_URL env var
      Auto-detected for localhost (development only)
    
    Token Refresh (required):
      --google-client-id and --google-client-secret flags
      OR GOOGLE_CLIENT_ID and GOOGLE_CLIENT_SECRET env vars
      Required for OAuth proxy mode and automatic token refresh

  STDIO Transport:
    Token Refresh (optional):
      GOOGLE_STDIO_CLIENT_ID and GOOGLE_STDIO_CLIENT_SECRET env vars
      OR GOOGLE_CLIENT_ID and GOOGLE_CLIENT_SECRET env vars (fallback)
      Without these, token refresh will fail when tokens expire (~1 hour).`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Parse encryption key from base64 if provided
			var encKeyBytes []byte
			if encryptionKey != "" {
				decoded, err := base64.StdEncoding.DecodeString(encryptionKey)
				if err != nil {
					return fmt.Errorf("invalid encryption key (must be base64 encoded): %w", err)
				}
				if len(decoded) != 32 {
					return fmt.Errorf("encryption key must be exactly 32 bytes (got %d bytes)", len(decoded))
				}
				encKeyBytes = decoded
			}

			// Build storage config from flags/env
			storageConfig := OAuthStorageConfig{
				Type: oauthStorageType,
				Valkey: ValkeyStorageConfig{
					URL:        valkeyURL,
					Password:   valkeyPassword,
					TLSEnabled: valkeyTLS,
					KeyPrefix:  valkeyKeyPrefix,
					DB:         valkeyDB,
				},
			}

			// Load storage config from environment variables if not set via flags
			loadOAuthStorageEnvVars(cmd, &storageConfig)

			// Load TLS paths from environment if not provided via flags
			if tlsCertFile == "" {
				tlsCertFile = os.Getenv("TLS_CERT_FILE")
			}
			if tlsKeyFile == "" {
				tlsKeyFile = os.Getenv("TLS_KEY_FILE")
			}

			securityConfig := OAuthSecurityConfig{
				AllowPublicClientRegistration: allowPublicClientRegistration,
				RegistrationAccessToken:       registrationAccessToken,
				AllowInsecureAuthWithoutState: allowInsecureAuthWithoutState,
				MaxClientsPerIP:               maxClientsPerIP,
				EncryptionKey:                 encKeyBytes,
				// Redirect URI Security (mcp-oauth v0.2.30+)
				DisableProductionMode:              disableProductionMode,
				AllowLocalhostRedirectURIs:         allowLocalhostRedirectURIs,
				AllowPrivateIPRedirectURIs:         allowPrivateIPRedirectURIs,
				AllowLinkLocalRedirectURIs:         allowLinkLocalRedirectURIs,
				DisableDNSValidation:               disableDNSValidation,
				DisableDNSValidationStrict:         disableDNSValidationStrict,
				DisableAuthorizationTimeValidation: disableAuthorizationTimeValidation,
				// Trusted scheme registration (mcp-oauth v0.2.30+)
				TrustedPublicRegistrationSchemes: trustedPublicRegistrationSchemes,
				DisableStrictSchemeMatching:      disableStrictSchemeMatching,
				// CIMD (mcp-oauth v0.2.30+)
				EnableCIMD: enableCIMD,
				// TLS support
				TLSCertFile: tlsCertFile,
				TLSKeyFile:  tlsKeyFile,
				// Storage configuration
				Storage: storageConfig,
			}
			return runServe(transport, debugMode, httpAddr, yolo, googleClientID, googleClientSecret, disableStreaming, baseURL, securityConfig)
		},
	}

	cmd.Flags().BoolVar(&debugMode, "debug", false, "Enable debug logging")
	cmd.Flags().StringVar(&transport, "transport", "stdio", "Transport type: stdio or streamable-http")
	cmd.Flags().StringVar(&httpAddr, "http-addr", ":8080", "HTTP server address (for streamable-http transport)")
	cmd.Flags().BoolVar(&yolo, "yolo", false, "Enable write operations (email sending, file deletion, etc.). Default is read-only mode.")
	cmd.Flags().StringVar(&googleClientID, "google-client-id", "", "Google OAuth Client ID for automatic token refresh (HTTP transport only). Can also use GOOGLE_CLIENT_ID env var.")
	cmd.Flags().StringVar(&googleClientSecret, "google-client-secret", "", "Google OAuth Client Secret for automatic token refresh (HTTP transport only). Can also use GOOGLE_CLIENT_SECRET env var.")
	cmd.Flags().BoolVar(&disableStreaming, "disable-streaming", false, "Disable streaming for HTTP transport (for compatibility with certain clients)")
	cmd.Flags().StringVar(&baseURL, "base-url", "", "Public base URL for OAuth (HTTP transport only). Required for deployed instances. Can also use MCP_BASE_URL env var. Example: https://mcp.example.com")

	// OAuth Security Settings (HTTP transport only)
	cmd.Flags().BoolVar(&allowPublicClientRegistration, "oauth-allow-public-registration", false, "WARNING: Allow unauthenticated client registration (NOT recommended for production). Can also use MCP_OAUTH_ALLOW_PUBLIC_REGISTRATION env var. Default: false (secure)")
	cmd.Flags().StringVar(&registrationAccessToken, "oauth-registration-token", "", "Registration access token required for client registration when public registration is disabled. Can also use MCP_OAUTH_REGISTRATION_TOKEN env var.")
	cmd.Flags().StringVar(&encryptionKey, "oauth-encryption-key", "", "AES-256 encryption key for token storage at rest (32 bytes, base64 encoded). REQUIRED for production. Can also use MCP_OAUTH_ENCRYPTION_KEY env var. Generate with: openssl rand -base64 32")
	cmd.Flags().BoolVar(&allowInsecureAuthWithoutState, "oauth-allow-no-state", false, "WARNING: Allow authorization without state parameter (weakens CSRF protection). Can also use MCP_OAUTH_ALLOW_NO_STATE env var. Default: false (secure)")
	cmd.Flags().IntVar(&maxClientsPerIP, "oauth-max-clients-per-ip", 10, "Maximum number of clients that can be registered per IP address (prevents DoS). Can also use MCP_OAUTH_MAX_CLIENTS_PER_IP env var. Default: 10")

	// TLS flags for HTTPS support
	cmd.Flags().StringVar(&tlsCertFile, "tls-cert-file", "", "Path to TLS certificate file (PEM format). If provided with --tls-key-file, enables HTTPS. Can also use TLS_CERT_FILE env var.")
	cmd.Flags().StringVar(&tlsKeyFile, "tls-key-file", "", "Path to TLS private key file (PEM format). If provided with --tls-cert-file, enables HTTPS. Can also use TLS_KEY_FILE env var.")

	// OAuth storage flags
	cmd.Flags().StringVar(&oauthStorageType, "oauth-storage-type", "memory", "OAuth token storage type: memory or valkey. Can also use OAUTH_STORAGE_TYPE env var.")
	cmd.Flags().StringVar(&valkeyURL, "valkey-url", "", "Valkey server address (e.g., valkey.namespace.svc:6379). Can also use VALKEY_URL env var.")
	cmd.Flags().StringVar(&valkeyPassword, "valkey-password", "", "Valkey authentication password. Can also use VALKEY_PASSWORD env var.")
	cmd.Flags().BoolVar(&valkeyTLS, "valkey-tls", false, "Enable TLS for Valkey connections. Can also use VALKEY_TLS_ENABLED env var.")
	cmd.Flags().StringVar(&valkeyKeyPrefix, "valkey-key-prefix", "mcp:", "Prefix for all Valkey keys. Can also use VALKEY_KEY_PREFIX env var.")
	cmd.Flags().IntVar(&valkeyDB, "valkey-db", 0, "Valkey database number. Can also use VALKEY_DB env var.")

	// Redirect URI Security Settings (mcp-oauth v0.2.30+)
	cmd.Flags().BoolVar(&disableProductionMode, "oauth-disable-production-mode", false, "WARNING: Disable production mode security (allows HTTP, private IPs in redirect URIs). Significantly weakens security.")
	cmd.Flags().BoolVar(&allowLocalhostRedirectURIs, "oauth-allow-localhost-redirect-uris", false, "Allow http://localhost redirect URIs for native apps (RFC 8252)")
	cmd.Flags().BoolVar(&allowPrivateIPRedirectURIs, "oauth-allow-private-ip-redirect-uris", false, "WARNING: Allow private IP addresses (10.x, 172.16.x, 192.168.x) in redirect URIs. SSRF risk.")
	cmd.Flags().BoolVar(&allowLinkLocalRedirectURIs, "oauth-allow-link-local-redirect-uris", false, "WARNING: Allow link-local addresses (169.254.x.x) in redirect URIs. Cloud metadata SSRF risk.")
	cmd.Flags().BoolVar(&disableDNSValidation, "oauth-disable-dns-validation", false, "WARNING: Disable DNS validation of redirect URI hostnames. Allows DNS rebinding attacks.")
	cmd.Flags().BoolVar(&disableDNSValidationStrict, "oauth-disable-dns-validation-strict", false, "WARNING: Disable fail-closed DNS validation (allow registration on DNS failures).")
	cmd.Flags().BoolVar(&disableAuthorizationTimeValidation, "oauth-disable-authorization-time-validation", false, "WARNING: Disable redirect URI validation at authorization time. Allows TOCTOU attacks.")

	// Trusted scheme registration for Cursor/VSCode compatibility (mcp-oauth v0.2.30+)
	cmd.Flags().StringSliceVar(&trustedPublicRegistrationSchemes, "oauth-trusted-schemes", nil, "URI schemes allowed for unauthenticated client registration (e.g., cursor,vscode). Best for internal/dev deployments.")
	cmd.Flags().BoolVar(&disableStrictSchemeMatching, "oauth-disable-strict-scheme-matching", false, "WARNING: Allow mixed redirect URI schemes with trusted scheme registration. Reduces security.")

	// CIMD (Client ID Metadata Documents) per MCP 2025-11-25 (mcp-oauth v0.2.30+)
	cmd.Flags().BoolVar(&enableCIMD, "oauth-enable-cimd", true, "Enable Client ID Metadata Documents (CIMD) per MCP 2025-11-25. Allows clients to use HTTPS URLs as client identifiers. Can also use MCP_OAUTH_ENABLE_CIMD env var.")

	return cmd
}

func runServe(transport string, debugMode bool, httpAddr string, yolo bool, googleClientID, googleClientSecret string, disableStreaming bool, baseURL string, securityConfig OAuthSecurityConfig) error {
	// Setup graceful shutdown
	shutdownCtx, cancel := signal.NotifyContext(context.Background(),
		os.Interrupt, syscall.SIGTERM)
	defer cancel()

	// Read GitHub config (optional for serve mode - will use empty strings if not available)
	// Users can authenticate via OAuth for MCP server usage
	if err := readGithubConfig(); err != nil {
		// Log warning but continue - GitHub config is optional for MCP server
		if transport != "stdio" {
			log.Printf("Warning: GitHub config not found (this is OK for MCP server): %v", err)
		}
		// Set empty values - server will work without GitHub integration
		githubUser = ""
		githubToken = ""
	}

	// Get Google OAuth credentials from environment if not provided via flags
	if googleClientID == "" {
		googleClientID = os.Getenv("GOOGLE_CLIENT_ID")
	}
	if googleClientSecret == "" {
		googleClientSecret = os.Getenv("GOOGLE_CLIENT_SECRET")
	}

	// Get OAuth security settings from environment if not provided via flags
	if !securityConfig.AllowPublicClientRegistration && os.Getenv("MCP_OAUTH_ALLOW_PUBLIC_REGISTRATION") == "true" {
		securityConfig.AllowPublicClientRegistration = true
	}
	if securityConfig.RegistrationAccessToken == "" {
		securityConfig.RegistrationAccessToken = os.Getenv("MCP_OAUTH_REGISTRATION_TOKEN")
	}
	if len(securityConfig.EncryptionKey) == 0 {
		if encKeyStr := os.Getenv("MCP_OAUTH_ENCRYPTION_KEY"); encKeyStr != "" {
			decoded, err := base64.StdEncoding.DecodeString(encKeyStr)
			if err != nil {
				log.Printf("Warning: Invalid encryption key in MCP_OAUTH_ENCRYPTION_KEY (must be base64): %v", err)
			} else if len(decoded) != 32 {
				log.Printf("Warning: Invalid encryption key length in MCP_OAUTH_ENCRYPTION_KEY (must be 32 bytes, got %d)", len(decoded))
			} else {
				securityConfig.EncryptionKey = decoded
			}
		}
	}
	if !securityConfig.AllowInsecureAuthWithoutState && os.Getenv("MCP_OAUTH_ALLOW_NO_STATE") == "true" {
		securityConfig.AllowInsecureAuthWithoutState = true
	}
	if securityConfig.MaxClientsPerIP == 0 {
		if envMax := os.Getenv("MCP_OAUTH_MAX_CLIENTS_PER_IP"); envMax != "" {
			var maxClients int
			if _, err := fmt.Sscanf(envMax, "%d", &maxClients); err == nil && maxClients > 0 {
				securityConfig.MaxClientsPerIP = maxClients
			}
		}
		// If still 0, use default of 10
		if securityConfig.MaxClientsPerIP == 0 {
			securityConfig.MaxClientsPerIP = 10
		}
	}

	// Parse redirect URI security settings from environment variables (mcp-oauth v0.2.30+)
	if !securityConfig.DisableProductionMode && os.Getenv("MCP_OAUTH_DISABLE_PRODUCTION_MODE") == "true" {
		securityConfig.DisableProductionMode = true
	}
	if !securityConfig.AllowLocalhostRedirectURIs && os.Getenv("MCP_OAUTH_ALLOW_LOCALHOST_REDIRECT_URIS") == "true" {
		securityConfig.AllowLocalhostRedirectURIs = true
	}
	if !securityConfig.AllowPrivateIPRedirectURIs && os.Getenv("MCP_OAUTH_ALLOW_PRIVATE_IP_REDIRECT_URIS") == "true" {
		securityConfig.AllowPrivateIPRedirectURIs = true
	}
	if !securityConfig.AllowLinkLocalRedirectURIs && os.Getenv("MCP_OAUTH_ALLOW_LINK_LOCAL_REDIRECT_URIS") == "true" {
		securityConfig.AllowLinkLocalRedirectURIs = true
	}
	if !securityConfig.DisableDNSValidation && os.Getenv("MCP_OAUTH_DISABLE_DNS_VALIDATION") == "true" {
		securityConfig.DisableDNSValidation = true
	}
	if !securityConfig.DisableDNSValidationStrict && os.Getenv("MCP_OAUTH_DISABLE_DNS_VALIDATION_STRICT") == "true" {
		securityConfig.DisableDNSValidationStrict = true
	}
	if !securityConfig.DisableAuthorizationTimeValidation && os.Getenv("MCP_OAUTH_DISABLE_AUTHORIZATION_TIME_VALIDATION") == "true" {
		securityConfig.DisableAuthorizationTimeValidation = true
	}

	// Parse trusted scheme registration settings from environment variables (mcp-oauth v0.2.30+)
	if len(securityConfig.TrustedPublicRegistrationSchemes) == 0 {
		if schemes := os.Getenv("MCP_OAUTH_TRUSTED_SCHEMES"); schemes != "" {
			securityConfig.TrustedPublicRegistrationSchemes = strings.Split(schemes, ",")
		}
	}
	if !securityConfig.DisableStrictSchemeMatching && os.Getenv("MCP_OAUTH_DISABLE_STRICT_SCHEME_MATCHING") == "true" {
		securityConfig.DisableStrictSchemeMatching = true
	}

	// Parse CIMD setting from environment variable (mcp-oauth v0.2.30+)
	// Default to true (enabled) per MCP 2025-11-25 specification
	if !securityConfig.EnableCIMD {
		if os.Getenv("MCP_OAUTH_ENABLE_CIMD") != "false" {
			// Default to true if not explicitly disabled
			securityConfig.EnableCIMD = true
		}
	}

	// Parse interstitial page branding from environment variables
	if logoURL := os.Getenv("MCP_INTERSTITIAL_LOGO_URL"); logoURL != "" {
		securityConfig.InterstitialLogoURL = logoURL
	}
	if logoAlt := os.Getenv("MCP_INTERSTITIAL_LOGO_ALT"); logoAlt != "" {
		securityConfig.InterstitialLogoAlt = logoAlt
	}
	if title := os.Getenv("MCP_INTERSTITIAL_TITLE"); title != "" {
		securityConfig.InterstitialTitle = title
	}
	if message := os.Getenv("MCP_INTERSTITIAL_MESSAGE"); message != "" {
		securityConfig.InterstitialMessage = message
	}
	if buttonText := os.Getenv("MCP_INTERSTITIAL_BUTTON_TEXT"); buttonText != "" {
		securityConfig.InterstitialButtonText = buttonText
	}
	if primaryColor := os.Getenv("MCP_INTERSTITIAL_PRIMARY_COLOR"); primaryColor != "" {
		securityConfig.InterstitialPrimaryColor = primaryColor
	}
	if bgGradient := os.Getenv("MCP_INTERSTITIAL_BACKGROUND_GRADIENT"); bgGradient != "" {
		securityConfig.InterstitialBackgroundGradient = bgGradient
	}

	// Create server context (will be recreated for HTTP with OAuth token provider)
	serverContext, err := server.NewServerContext(shutdownCtx, githubUser, githubToken)
	if err != nil {
		return fmt.Errorf("failed to create server context: %w", err)
	}
	defer func() {
		if err := serverContext.Shutdown(); err != nil {
			if transport != "stdio" {
				log.Printf("Error during server context shutdown: %v", err)
			}
		}
	}()

	// Create MCP server
	// Note: mcp.Implementation has Title field but WithTitle() ServerOption not available in v0.43.0
	mcpSrv := mcpserver.NewMCPServer("inboxfewer", version,
		mcpserver.WithToolCapabilities(true),
		mcpserver.WithResourceCapabilities(false, false), // Subscribe and listChanged
	)

	// readOnly is the inverse of yolo
	readOnly := !yolo

	// Log the mode for visibility (only for non-stdio transports)
	if transport != "stdio" {
		if readOnly {
			log.Println("Starting server in READ-ONLY mode (use --yolo to enable write operations)")
		} else {
			log.Println("Starting server with WRITE operations enabled (--yolo flag is set)")
		}
	}

	// Register all tools and resources
	if err := registerAllTools(mcpSrv, serverContext, readOnly); err != nil {
		return err
	}

	// Start the appropriate server based on transport type
	switch transport {
	case "stdio":
		return runStdioServer(mcpSrv)
	case "streamable-http":
		fmt.Printf("Starting inboxfewer MCP server with %s transport...\n", transport)
		return runStreamableHTTPServer(mcpSrv, serverContext, httpAddr, shutdownCtx, debugMode, googleClientID, googleClientSecret, readOnly, disableStreaming, baseURL, securityConfig)
	default:
		return fmt.Errorf("unsupported transport type: %s (supported: stdio, streamable-http)", transport)
	}
}

func runStdioServer(mcpSrv *mcpserver.MCPServer) error {
	serverDone := make(chan error, 1)
	go func() {
		defer close(serverDone)
		if err := mcpserver.ServeStdio(mcpSrv); err != nil {
			serverDone <- err
		}
	}()

	err := <-serverDone
	if err != nil {
		return fmt.Errorf("server stopped with error: %w", err)
	}
	return nil
}

// registerAllTools registers all MCP tools and resources
// Extracted to avoid duplication in serve.go
func registerAllTools(mcpSrv *mcpserver.MCPServer, ctx *server.ServerContext, readOnly bool) error {
	// Define all tool registrations
	type toolRegistration struct {
		name     string
		register func() error
	}

	registrations := []toolRegistration{
		{
			name: "Gmail",
			register: func() error {
				return gmail_tools.RegisterGmailTools(mcpSrv, ctx, readOnly)
			},
		},
		{
			name: "Docs",
			register: func() error {
				return docs_tools.RegisterDocsTools(mcpSrv, ctx)
			},
		},
		{
			name: "Drive",
			register: func() error {
				return drive_tools.RegisterDriveTools(mcpSrv, ctx, readOnly)
			},
		},
		{
			name: "Calendar",
			register: func() error {
				return calendar_tools.RegisterCalendarTools(mcpSrv, ctx, readOnly)
			},
		},
		{
			name: "Meet",
			register: func() error {
				return meet_tools.RegisterMeetTools(mcpSrv, ctx, readOnly)
			},
		},
		{
			name: "Tasks",
			register: func() error {
				return tasks_tools.RegisterTasksTools(mcpSrv, ctx, readOnly)
			},
		},
		{
			name: "User Resources",
			register: func() error {
				return resources.RegisterUserResources(mcpSrv, ctx)
			},
		},
	}

	// Register all tools
	for _, reg := range registrations {
		if err := reg.register(); err != nil {
			return fmt.Errorf("failed to register %s: %w", reg.name, err)
		}
	}

	return nil
}

func runStreamableHTTPServer(mcpSrv *mcpserver.MCPServer, oldServerContext *server.ServerContext, addr string, ctx context.Context, debugMode bool, googleClientID, googleClientSecret string, readOnly bool, disableStreaming bool, baseURL string, securityConfig OAuthSecurityConfig) error {
	// Create OAuth-enabled HTTP server
	// Base URL should be the full URL where the server is accessible
	// For development, use http://localhost:8080
	// For production, use the actual HTTPS URL

	// Determine base URL from flag, environment variable, or auto-detection
	if baseURL == "" {
		baseURL = os.Getenv("MCP_BASE_URL")
	}
	if baseURL == "" {
		// Fall back to auto-detection for local development
		baseURL = fmt.Sprintf("http://%s", addr)
		if addr[0] == ':' {
			baseURL = fmt.Sprintf("http://localhost%s", addr)
		}
		log.Printf("No base URL configured, using auto-detected: %s", baseURL)
		log.Printf("For deployed instances, set --base-url flag or MCP_BASE_URL env var")
	} else {
		log.Printf("Using configured base URL: %s", baseURL)
	}

	// Create OAuth handler
	oauthConfig := server.OAuthConfig{
		BaseURL:                       baseURL,
		GoogleClientID:                googleClientID,
		GoogleClientSecret:            googleClientSecret,
		DisableStreaming:              disableStreaming,
		DebugMode:                     debugMode,
		AllowPublicClientRegistration: securityConfig.AllowPublicClientRegistration,
		RegistrationAccessToken:       securityConfig.RegistrationAccessToken,
		AllowInsecureAuthWithoutState: securityConfig.AllowInsecureAuthWithoutState,
		MaxClientsPerIP:               securityConfig.MaxClientsPerIP,
		EncryptionKey:                 securityConfig.EncryptionKey,
		// Redirect URI Security (mcp-oauth v0.2.30+)
		RedirectURISecurity: oauth.RedirectURISecurityConfig{
			DisableProductionMode:              securityConfig.DisableProductionMode,
			AllowLocalhostRedirectURIs:         securityConfig.AllowLocalhostRedirectURIs,
			AllowPrivateIPRedirectURIs:         securityConfig.AllowPrivateIPRedirectURIs,
			AllowLinkLocalRedirectURIs:         securityConfig.AllowLinkLocalRedirectURIs,
			DisableDNSValidation:               securityConfig.DisableDNSValidation,
			DisableDNSValidationStrict:         securityConfig.DisableDNSValidationStrict,
			DisableAuthorizationTimeValidation: securityConfig.DisableAuthorizationTimeValidation,
		},
		// Trusted scheme registration (mcp-oauth v0.2.30+)
		TrustedPublicRegistrationSchemes: securityConfig.TrustedPublicRegistrationSchemes,
		DisableStrictSchemeMatching:      securityConfig.DisableStrictSchemeMatching,
		// CIMD (mcp-oauth v0.2.30+)
		EnableCIMD: securityConfig.EnableCIMD,
		// Storage configuration (mcp-oauth v0.2.30+)
		Storage: oauth.StorageConfig{
			Type: oauth.StorageType(securityConfig.Storage.Type),
			Valkey: oauth.ValkeyConfig{
				URL:        securityConfig.Storage.Valkey.URL,
				Password:   securityConfig.Storage.Valkey.Password,
				TLSEnabled: securityConfig.Storage.Valkey.TLSEnabled,
				KeyPrefix:  securityConfig.Storage.Valkey.KeyPrefix,
				DB:         securityConfig.Storage.Valkey.DB,
			},
		},
		// TLS configuration
		TLSCertFile: securityConfig.TLSCertFile,
		TLSKeyFile:  securityConfig.TLSKeyFile,
	}

	// Configure interstitial page branding if any env vars are set
	if securityConfig.InterstitialLogoURL != "" ||
		securityConfig.InterstitialLogoAlt != "" ||
		securityConfig.InterstitialTitle != "" ||
		securityConfig.InterstitialMessage != "" ||
		securityConfig.InterstitialButtonText != "" ||
		securityConfig.InterstitialPrimaryColor != "" ||
		securityConfig.InterstitialBackgroundGradient != "" {
		oauthConfig.Interstitial = &oauth.InterstitialConfig{
			LogoURL:            securityConfig.InterstitialLogoURL,
			LogoAlt:            securityConfig.InterstitialLogoAlt,
			Title:              securityConfig.InterstitialTitle,
			Message:            securityConfig.InterstitialMessage,
			ButtonText:         securityConfig.InterstitialButtonText,
			PrimaryColor:       securityConfig.InterstitialPrimaryColor,
			BackgroundGradient: securityConfig.InterstitialBackgroundGradient,
		}
	}

	oauthHandler, err := server.CreateOAuthHandler(oauthConfig)
	if err != nil {
		return fmt.Errorf("failed to create OAuth handler: %w", err)
	}
	defer oauthHandler.Stop() // Ensure cleanup

	// Create token provider from OAuth store
	tokenProvider := oauth.NewTokenProvider(oauthHandler.GetStore())

	// Recreate server context with OAuth token provider
	// This ensures Google API clients use tokens from OAuth authentication
	githubUser := oldServerContext.GithubUser()
	githubToken := oldServerContext.GithubToken()

	// Shutdown old context and create new one with OAuth token provider
	if err := oldServerContext.Shutdown(); err != nil {
		log.Printf("Warning: failed to shutdown old server context: %v", err)
	}

	serverContext, err := server.NewServerContextWithProvider(ctx, githubUser, githubToken, tokenProvider)
	if err != nil {
		return fmt.Errorf("failed to create server context with OAuth token provider: %w", err)
	}
	defer func() {
		if err := serverContext.Shutdown(); err != nil {
			log.Printf("Error during server context shutdown: %v", err)
		}
	}()

	// Re-register all tools with the new context
	if err := registerAllTools(mcpSrv, serverContext, readOnly); err != nil {
		return err
	}

	// Create OAuth server with existing handler
	oauthServer, err := server.NewOAuthHTTPServerWithHandlerAndTLS(mcpSrv, "streamable-http", oauthHandler, disableStreaming, securityConfig.TLSCertFile, securityConfig.TLSKeyFile)
	if err != nil {
		return fmt.Errorf("failed to create OAuth HTTP server: %w", err)
	}

	// Set up health checker for health check endpoints
	healthChecker := server.NewHealthChecker(serverContext)
	oauthServer.SetHealthChecker(healthChecker)

	fmt.Printf("Streamable HTTP server with Google OAuth authentication starting on %s\n", addr)
	fmt.Printf("  HTTP endpoint: /mcp\n")
	fmt.Printf("  Health endpoints: /healthz, /readyz\n")
	fmt.Printf("  OAuth metadata: /.well-known/oauth-protected-resource\n")
	fmt.Printf("  Authorization Server: %s\n", baseURL)

	if googleClientID != "" && googleClientSecret != "" {
		fmt.Println("\n✓ Automatic token refresh: ENABLED")
		fmt.Println("  Tokens will be refreshed automatically before expiration")
		fmt.Println("  Enhanced security features: proactive refresh, atomic operations, token families")
	} else {
		fmt.Println("\n⚠ Automatic token refresh: DISABLED")
		fmt.Println("  Users will need to re-authenticate when tokens expire (~1 hour)")
		fmt.Println("  To enable, provide --google-client-id and --google-client-secret")
	}

	fmt.Println("\nClients must authenticate with Google OAuth to access this server.")
	fmt.Println("The MCP client (e.g., Cursor, Claude Desktop) will handle the OAuth flow automatically.")

	serverDone := make(chan error, 1)
	go func() {
		defer close(serverDone)
		if err := oauthServer.Start(addr); err != nil {
			serverDone <- err
		}
	}()

	select {
	case <-ctx.Done():
		fmt.Println("Shutdown signal received, stopping HTTP server...")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		if err := oauthServer.Shutdown(shutdownCtx); err != nil {
			return fmt.Errorf("error shutting down HTTP server: %w", err)
		}
	case err := <-serverDone:
		if err != nil {
			return fmt.Errorf("HTTP server stopped with error: %w", err)
		}
		fmt.Println("HTTP server stopped normally")
	}

	fmt.Println("HTTP server gracefully stopped")
	return nil
}

// loadOAuthStorageEnvVars loads OAuth storage configuration from environment variables.
// Environment variables only override flag values when the flag was not explicitly set.
// The cmd parameter is used to check if flags were explicitly set by the user.
func loadOAuthStorageEnvVars(cmd *cobra.Command, config *OAuthStorageConfig) {
	// Storage type - env var only applies if flag was not explicitly set
	if !cmd.Flags().Changed("oauth-storage-type") {
		if storageType := os.Getenv("OAUTH_STORAGE_TYPE"); storageType != "" {
			config.Type = storageType
		}
	}

	// Valkey URL - env var only applies if flag was not explicitly set
	if !cmd.Flags().Changed("valkey-url") {
		if url := os.Getenv("VALKEY_URL"); url != "" && config.Valkey.URL == "" {
			config.Valkey.URL = url
		}
	}

	// Valkey Password - env var only applies if flag was not explicitly set
	if !cmd.Flags().Changed("valkey-password") {
		if password := os.Getenv("VALKEY_PASSWORD"); password != "" && config.Valkey.Password == "" {
			config.Valkey.Password = password
		}
	}

	// Valkey Key Prefix - env var only applies if flag was not explicitly set
	if !cmd.Flags().Changed("valkey-key-prefix") {
		if keyPrefix := os.Getenv("VALKEY_KEY_PREFIX"); keyPrefix != "" && config.Valkey.KeyPrefix == "" {
			config.Valkey.KeyPrefix = keyPrefix
		}
	}

	// Valkey TLS - env var only applies if flag was not explicitly set
	if !cmd.Flags().Changed("valkey-tls") {
		if os.Getenv("VALKEY_TLS_ENABLED") == "true" {
			config.Valkey.TLSEnabled = true
		}
	}

	// Valkey DB - env var only applies if flag was not explicitly set
	if !cmd.Flags().Changed("valkey-db") {
		if dbStr := os.Getenv("VALKEY_DB"); dbStr != "" {
			var db int
			if _, err := fmt.Sscanf(dbStr, "%d", &db); err == nil {
				config.Valkey.DB = db
			}
		}
	}
}
