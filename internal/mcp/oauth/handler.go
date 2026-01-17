package oauth

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"log/slog"
	"os"
	"sync"
	"time"

	oauth "github.com/giantswarm/mcp-oauth"
	"github.com/giantswarm/mcp-oauth/providers/google"
	"github.com/giantswarm/mcp-oauth/security"
	oauthserver "github.com/giantswarm/mcp-oauth/server"
	"github.com/giantswarm/mcp-oauth/storage"
	"github.com/giantswarm/mcp-oauth/storage/memory"
	"github.com/giantswarm/mcp-oauth/storage/valkey"

	inboxgoogle "github.com/teemow/inboxfewer/internal/google"
)

const (
	// defaultBurstMultiplier is the default multiplier for burst size relative to rate
	// Burst = Rate * defaultBurstMultiplier
	defaultBurstMultiplier = 2
)

// Config holds the OAuth library handler configuration
// This maps our existing configuration to the mcp-oauth library configuration
type Config struct {
	// BaseURL is the MCP server base URL (e.g., https://mcp.example.com)
	BaseURL string

	// GoogleClientID is the Google OAuth Client ID
	GoogleClientID string

	// GoogleClientSecret is the Google OAuth Client Secret
	GoogleClientSecret string

	// Security settings (secure by default)
	Security SecurityConfig

	// RateLimit configuration
	RateLimit RateLimitConfig

	// Interstitial configures the OAuth success page for custom URL schemes (cursor://, vscode://, etc.)
	// If nil, uses the default mcp-oauth interstitial page
	Interstitial *InterstitialConfig

	// RedirectURISecurity configures security validation for redirect URIs
	// All options default to secure values in mcp-oauth
	RedirectURISecurity RedirectURISecurityConfig

	// TrustedPublicRegistrationSchemes lists URI schemes allowed for unauthenticated
	// client registration. Enables Cursor/VSCode without registration tokens.
	// Best suited for internal/development deployments due to platform-specific
	// limitations in custom URI scheme security. Schemes must conform to RFC 3986.
	TrustedPublicRegistrationSchemes []string

	// DisableStrictSchemeMatching allows mixed scheme clients to register without token
	DisableStrictSchemeMatching bool

	// EnableCIMD enables Client ID Metadata Documents per MCP 2025-11-25.
	// When enabled, clients can use HTTPS URLs as client identifiers.
	// Default: true (enabled for MCP 2025-11-25 compliance)
	EnableCIMD bool

	// CIMDAllowPrivateIPs allows CIMD metadata URLs that resolve to private IPs.
	// WARNING: Reduces SSRF protection. Only enable for internal/VPN deployments
	// where MCP servers legitimately communicate over private networks.
	// Default: false (blocked for security)
	CIMDAllowPrivateIPs bool

	// TrustedAudiences lists additional OAuth client IDs whose tokens are accepted.
	// This enables Single Sign-On (SSO) scenarios where an upstream aggregator (like muster)
	// forwards user tokens to downstream MCP servers. When the aggregator adds its
	// client_id to TrustedAudiences, downstream servers can accept these forwarded
	// tokens without requiring a separate authentication flow.
	//
	// Security: Tokens must still be from the configured issuer (Google/Dex) and
	// cryptographically signed. Only the audience claim is relaxed.
	//
	// Example: ["muster-client", "my-aggregator-client"]
	TrustedAudiences []string

	// Storage configures the token storage backend
	// Defaults to in-memory storage if not specified
	Storage StorageConfig

	// Logger for structured logging (optional, uses default if not provided)
	Logger *slog.Logger
}

// StorageType represents the type of token storage backend.
type StorageType string

const (
	// StorageTypeMemory uses in-memory storage (default, not recommended for production)
	StorageTypeMemory StorageType = "memory"
	// StorageTypeValkey uses Valkey (Redis-compatible) for persistent storage
	StorageTypeValkey StorageType = "valkey"
)

// StorageConfig holds configuration for OAuth token storage backend.
type StorageConfig struct {
	// Type is the storage backend type: "memory" or "valkey" (default: "memory")
	Type StorageType

	// Valkey configuration (used when Type is "valkey")
	Valkey ValkeyConfig
}

// ValkeyConfig holds configuration for Valkey storage backend.
type ValkeyConfig struct {
	// URL is the Valkey server address (e.g., "valkey.namespace.svc:6379")
	URL string

	// Password is the optional password for Valkey authentication
	Password string

	// TLSEnabled enables TLS for Valkey connections
	TLSEnabled bool

	// TLSCAFile is the path to a custom CA certificate file for TLS verification.
	// Use this when Valkey uses certificates signed by a private CA.
	// If empty, the system CA pool is used.
	TLSCAFile string

	// KeyPrefix is the prefix for all Valkey keys (default: "mcp:")
	KeyPrefix string

	// DB is the Valkey database number (default: 0)
	DB int
}

// InterstitialConfig configures the OAuth success interstitial page branding
type InterstitialConfig struct {
	// LogoURL is an optional URL to a logo image (must be HTTPS)
	// Leave empty to use the default animated checkmark icon
	LogoURL string

	// LogoAlt is the alt text for the logo image (for accessibility)
	LogoAlt string

	// Title replaces the "Authorization Successful" heading
	// Example: "Connected to Inboxfewer"
	Title string

	// Message replaces the default success message
	// Use {{.AppName}} placeholder for the application name
	Message string

	// ButtonText replaces the "Open [AppName]" button text
	// Use {{.AppName}} placeholder for the application name
	ButtonText string

	// PrimaryColor is the primary brand color (CSS color value)
	// Example: "#667eea" or "rgb(102, 126, 234)"
	PrimaryColor string

	// BackgroundGradient is the body background CSS value
	// Example: "linear-gradient(135deg, #1e3a5f 0%, #2d5a87 100%)"
	BackgroundGradient string

	// CustomCSS allows additional CSS customization (injected into <style> tag)
	// WARNING: Be careful with CSS injection - avoid user-provided values
	CustomCSS string
}

// SecurityConfig holds OAuth security settings (secure by default)
type SecurityConfig struct {
	// AllowPublicClientRegistration allows unauthenticated dynamic client registration
	// WARNING: This can lead to DoS attacks via unlimited client registration
	// Default: false (authentication REQUIRED for security)
	AllowPublicClientRegistration bool

	// RegistrationAccessToken is the token required for client registration
	// Only checked if AllowPublicClientRegistration is false
	RegistrationAccessToken string

	// AllowInsecureAuthWithoutState allows authorization requests without state parameter
	// WARNING: Disabling this weakens CSRF protection
	// Default: false (state is REQUIRED for security)
	AllowInsecureAuthWithoutState bool

	// MaxClientsPerIP limits the number of clients that can be registered per IP
	// Prevents DoS attacks via mass client registration
	// Default: 10
	MaxClientsPerIP int

	// EncryptionKey is the AES-256 key for encrypting tokens at rest (32 bytes)
	// If nil or empty, tokens are stored unencrypted in memory
	// For production, provide a 32-byte key from secure storage (KMS, Vault, etc.)
	EncryptionKey []byte

	// EnableAuditLogging enables comprehensive security audit logging
	// Logs authentication events, token operations, and security violations
	// Default: true (enabled for security)
	EnableAuditLogging bool

	// RefreshTokenTTL is the time-to-live for refresh tokens
	// Recommended: 30-90 days for security vs usability balance
	// Default: 90 days
	RefreshTokenTTL time.Duration
}

// RateLimitConfig holds rate limiting configuration
type RateLimitConfig struct {
	// Rate is the number of requests per second allowed per IP (0 = no limit)
	Rate int

	// Burst is the maximum burst size allowed per IP
	Burst int

	// UserRate is the number of requests per second allowed per authenticated user (0 = no limit)
	UserRate int

	// UserBurst is the maximum burst size allowed per authenticated user
	UserBurst int

	// TrustProxy indicates whether to trust X-Forwarded-For and X-Real-IP headers
	// Only set to true if the server is behind a trusted proxy
	// Default: false (secure by default)
	TrustProxy bool
}

// RedirectURISecurityConfig holds configuration for redirect URI security validation.
// All options default to secure values in mcp-oauth. Use Disable* flags to opt-out.
type RedirectURISecurityConfig struct {
	// DisableProductionMode disables strict HTTPS/private IP enforcement
	DisableProductionMode bool

	// AllowLocalhostRedirectURIs allows http://localhost for native apps (RFC 8252)
	AllowLocalhostRedirectURIs bool

	// AllowPrivateIPRedirectURIs allows private IP addresses in redirect URIs
	AllowPrivateIPRedirectURIs bool

	// AllowLinkLocalRedirectURIs allows link-local addresses (169.254.x.x)
	AllowLinkLocalRedirectURIs bool

	// DisableDNSValidation disables hostname resolution checks
	DisableDNSValidation bool

	// DisableDNSValidationStrict disables fail-closed DNS validation
	DisableDNSValidationStrict bool

	// DisableAuthorizationTimeValidation disables redirect URI checks at auth time
	DisableAuthorizationTimeValidation bool
}

// valkeyCloser is a function type for closing Valkey connections
type valkeyCloser func()

// Handler wraps the mcp-oauth library components for integration with inboxfewer
type Handler struct {
	server          *oauth.Server
	handler         *oauth.Handler
	tokenStore      storage.TokenStore
	memoryStore     *memory.Store // Only set when using memory storage
	closeValkey     valkeyCloser  // Function to close Valkey store
	ipRateLimiter   *security.RateLimiter
	userRateLimiter *security.RateLimiter
	clientRegRL     *security.ClientRegistrationRateLimiter
	stopOnce        sync.Once // Ensures Stop() is truly idempotent even under concurrent calls
}

// NewHandler creates a new OAuth handler using the mcp-oauth library
func NewHandler(config *Config) (*Handler, error) {
	// Set default logger if not provided
	logger := config.Logger
	if logger == nil {
		logger = slog.Default()
	}

	// Create Google provider with our scopes
	redirectURL := config.BaseURL + "/oauth/callback"
	provider, err := google.NewProvider(&google.Config{
		ClientID:     config.GoogleClientID,
		ClientSecret: config.GoogleClientSecret,
		RedirectURL:  redirectURL,
		Scopes:       inboxgoogle.DefaultOAuthScopes,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create Google provider: %w", err)
	}

	// Create storage backend based on configuration
	var tokenStore storage.TokenStore
	var clientStore storage.ClientStore
	var flowStore storage.FlowStore
	var memStore *memory.Store
	var closeValkeyFn valkeyCloser

	switch config.Storage.Type {
	case StorageTypeValkey:
		if config.Storage.Valkey.URL == "" {
			return nil, fmt.Errorf("valkey URL is required when using valkey storage")
		}

		// Configure Valkey storage
		valkeyConfig := valkey.Config{
			Address:   config.Storage.Valkey.URL,
			Password:  config.Storage.Valkey.Password,
			DB:        config.Storage.Valkey.DB,
			KeyPrefix: config.Storage.Valkey.KeyPrefix,
			Logger:    logger,
		}

		// Configure TLS if enabled
		if config.Storage.Valkey.TLSEnabled {
			tlsConfig := &tls.Config{
				MinVersion: tls.VersionTLS12,
			}

			// Load custom CA certificate if provided
			if config.Storage.Valkey.TLSCAFile != "" {
				caCert, err := os.ReadFile(config.Storage.Valkey.TLSCAFile)
				if err != nil {
					return nil, fmt.Errorf("failed to read Valkey TLS CA certificate: %w", err)
				}

				caCertPool := x509.NewCertPool()
				if !caCertPool.AppendCertsFromPEM(caCert) {
					return nil, fmt.Errorf("failed to parse Valkey TLS CA certificate")
				}

				tlsConfig.RootCAs = caCertPool
				logger.Info("Using custom CA certificate for Valkey TLS", "caFile", config.Storage.Valkey.TLSCAFile)
			}

			valkeyConfig.TLS = tlsConfig
		}

		// Set default key prefix if not specified
		if valkeyConfig.KeyPrefix == "" {
			valkeyConfig.KeyPrefix = valkey.DefaultKeyPrefix
		}

		valkeyStore, err := valkey.New(valkeyConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to create Valkey storage: %w", err)
		}

		// Set up encryption for Valkey store if key is provided
		if len(config.Security.EncryptionKey) > 0 {
			encryptor, err := security.NewEncryptor(config.Security.EncryptionKey)
			if err != nil {
				// Close the Valkey store on error to release resources
				valkeyStore.Close()
				return nil, fmt.Errorf("failed to create encryptor for Valkey storage: %w", err)
			}
			valkeyStore.SetEncryptor(encryptor)
			logger.Info("Token encryption at rest enabled for Valkey storage (AES-256-GCM)")
		}

		// Valkey store implements all required interfaces
		tokenStore = valkeyStore
		clientStore = valkeyStore
		flowStore = valkeyStore
		closeValkeyFn = valkeyStore.Close // Store Close function for cleanup
		logger.Info("Using Valkey storage backend", "address", config.Storage.Valkey.URL, "tls", config.Storage.Valkey.TLSEnabled)

	case StorageTypeMemory, "":
		// Use memory storage (default)
		memStore = memory.New()
		tokenStore = memStore
		clientStore = memStore
		flowStore = memStore
		if config.Storage.Type == "" {
			logger.Info("Using in-memory storage backend (default)")
		} else {
			logger.Info("Using in-memory storage backend")
		}

	default:
		return nil, fmt.Errorf("unsupported storage type: %s (supported: memory, valkey)", config.Storage.Type)
	}

	// Set default refresh token TTL if not specified
	refreshTokenTTL := config.Security.RefreshTokenTTL
	if refreshTokenTTL == 0 {
		refreshTokenTTL = 90 * 24 * time.Hour // 90 days default
	}

	// Set default max clients per IP if not specified
	maxClientsPerIP := config.Security.MaxClientsPerIP
	if maxClientsPerIP == 0 {
		maxClientsPerIP = 10
	}

	// Create server configuration
	serverConfig := &oauthserver.Config{
		Issuer:                        config.BaseURL,
		RefreshTokenTTL:               int64(refreshTokenTTL.Seconds()),
		AllowRefreshTokenRotation:     true,  // OAuth 2.1 best practice
		RequirePKCE:                   true,  // OAuth 2.1 requirement
		AllowPKCEPlain:                false, // Only S256, not plain
		AllowPublicClientRegistration: config.Security.AllowPublicClientRegistration,
		RegistrationAccessToken:       config.Security.RegistrationAccessToken,
		AllowNoStateParameter:         config.Security.AllowInsecureAuthWithoutState,
		MaxClientsPerIP:               maxClientsPerIP,
		TrustProxy:                    config.RateLimit.TrustProxy,
		TokenRefreshThreshold:         300, // 5 minutes proactive refresh
		ClockSkewGracePeriod:          5,   // 5 seconds clock skew tolerance

		// Enable Client ID Metadata Documents (CIMD) per MCP 2025-11-25
		// Allows clients to use HTTPS URLs as client identifiers
		EnableClientIDMetadataDocuments: config.EnableCIMD,

		// Allow CIMD metadata URLs that resolve to private IPs (mcp-oauth v0.2.33+)
		AllowPrivateIPClientMetadata: config.CIMDAllowPrivateIPs,

		// Trusted scheme registration for Cursor/VSCode compatibility
		// Allows unauthenticated registration for clients using these schemes only
		TrustedPublicRegistrationSchemes: config.TrustedPublicRegistrationSchemes,
		DisableStrictSchemeMatching:      config.DisableStrictSchemeMatching,

		// Redirect URI Security Configuration
		// mcp-oauth defaults to secure values; we pass explicit disable flags
		DisableProductionMode:              config.RedirectURISecurity.DisableProductionMode,
		AllowLocalhostRedirectURIs:         config.RedirectURISecurity.AllowLocalhostRedirectURIs,
		AllowPrivateIPRedirectURIs:         config.RedirectURISecurity.AllowPrivateIPRedirectURIs,
		AllowLinkLocalRedirectURIs:         config.RedirectURISecurity.AllowLinkLocalRedirectURIs,
		DisableDNSValidation:               config.RedirectURISecurity.DisableDNSValidation,
		DisableDNSValidationStrict:         config.RedirectURISecurity.DisableDNSValidationStrict,
		DisableAuthorizationTimeValidation: config.RedirectURISecurity.DisableAuthorizationTimeValidation,

		// SSO token forwarding (mcp-oauth v0.2.38+)
		// Accept tokens with audiences from trusted upstream aggregators
		TrustedAudiences: config.TrustedAudiences,
	}

	// Configure interstitial page branding if provided
	if config.Interstitial != nil {
		serverConfig.Interstitial = &oauthserver.InterstitialConfig{
			Branding: &oauthserver.InterstitialBranding{
				LogoURL:            config.Interstitial.LogoURL,
				LogoAlt:            config.Interstitial.LogoAlt,
				Title:              config.Interstitial.Title,
				Message:            config.Interstitial.Message,
				ButtonText:         config.Interstitial.ButtonText,
				PrimaryColor:       config.Interstitial.PrimaryColor,
				BackgroundGradient: config.Interstitial.BackgroundGradient,
				CustomCSS:          config.Interstitial.CustomCSS,
			},
		}
	}

	// Create OAuth server
	server, err := oauth.NewServer(
		provider,
		tokenStore,  // TokenStore
		clientStore, // ClientStore
		flowStore,   // FlowStore
		serverConfig,
		logger,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create OAuth server: %w", err)
	}

	// Set up encryption if key provided (only for memory storage; Valkey encryption is set above)
	if len(config.Security.EncryptionKey) > 0 && config.Storage.Type != StorageTypeValkey {
		encryptor, err := security.NewEncryptor(config.Security.EncryptionKey)
		if err != nil {
			return nil, fmt.Errorf("failed to create encryptor: %w", err)
		}
		server.SetEncryptor(encryptor)
		logger.Info("Token encryption at rest enabled (AES-256-GCM)")
	}

	// Set up audit logging if enabled
	if config.Security.EnableAuditLogging {
		auditor := security.NewAuditor(logger, true)
		server.SetAuditor(auditor)
		logger.Info("Security audit logging enabled")
	}

	// Set up IP-based rate limiting if configured
	var ipRateLimiter *security.RateLimiter
	if config.RateLimit.Rate > 0 {
		burst := config.RateLimit.Burst
		if burst == 0 {
			burst = config.RateLimit.Rate * defaultBurstMultiplier
		}
		ipRateLimiter = security.NewRateLimiter(config.RateLimit.Rate, burst, logger)
		server.SetRateLimiter(ipRateLimiter)
		logger.Info("IP-based rate limiting enabled",
			"rate", config.RateLimit.Rate,
			"burst", burst)
	}

	// Set up user-based rate limiting if configured
	var userRateLimiter *security.RateLimiter
	if config.RateLimit.UserRate > 0 {
		burst := config.RateLimit.UserBurst
		if burst == 0 {
			burst = config.RateLimit.UserRate * defaultBurstMultiplier
		}
		userRateLimiter = security.NewRateLimiter(config.RateLimit.UserRate, burst, logger)
		server.SetUserRateLimiter(userRateLimiter)
		logger.Info("User-based rate limiting enabled",
			"rate", config.RateLimit.UserRate,
			"burst", burst)
	}

	// Set up client registration rate limiting with configured maxClientsPerIP
	clientRegRL := security.NewClientRegistrationRateLimiterWithConfig(
		maxClientsPerIP,
		security.DefaultRegistrationWindow,
		security.DefaultMaxRegistrationEntries,
		logger,
	)
	server.SetClientRegistrationRateLimiter(clientRegRL)
	logger.Info("Client registration rate limiting enabled",
		"maxClientsPerIP", maxClientsPerIP,
		"window", security.DefaultRegistrationWindow)

	// Create HTTP handler
	handler := oauth.NewHandler(server, logger)

	return &Handler{
		server:          server,
		handler:         handler,
		tokenStore:      tokenStore,
		memoryStore:     memStore,
		closeValkey:     closeValkeyFn,
		ipRateLimiter:   ipRateLimiter,
		userRateLimiter: userRateLimiter,
		clientRegRL:     clientRegRL,
	}, nil
}

// GetHandler returns the underlying mcp-oauth handler for HTTP routing
func (h *Handler) GetHandler() *oauth.Handler {
	return h.handler
}

// GetStore returns the underlying storage for token provider integration
func (h *Handler) GetStore() storage.TokenStore {
	return h.tokenStore
}

// GetServer returns the underlying OAuth server
func (h *Handler) GetServer() *oauth.Server {
	return h.server
}

// CanRefreshTokens returns true if the handler can refresh tokens
// This checks if Google OAuth credentials are properly configured
func (h *Handler) CanRefreshTokens() bool {
	// The library's Google provider always supports refresh if configured
	return true
}

// Stop gracefully stops all background services (rate limiters, storage cleanup, etc.)
// This method is idempotent and safe to call concurrently from multiple goroutines.
func (h *Handler) Stop() {
	h.stopOnce.Do(func() {
		if h.memoryStore != nil {
			h.memoryStore.Stop()
		}
		if h.closeValkey != nil {
			h.closeValkey()
		}
		if h.ipRateLimiter != nil {
			h.ipRateLimiter.Stop()
		}
		if h.userRateLimiter != nil {
			h.userRateLimiter.Stop()
		}
		if h.clientRegRL != nil {
			h.clientRegRL.Stop()
		}
	})
}
