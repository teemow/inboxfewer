package oauth

import (
	"fmt"
	"log/slog"
	"time"

	oauth "github.com/giantswarm/mcp-oauth"
	"github.com/giantswarm/mcp-oauth/providers/google"
	"github.com/giantswarm/mcp-oauth/security"
	"github.com/giantswarm/mcp-oauth/storage/memory"

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

	// Logger for structured logging (optional, uses default if not provided)
	Logger *slog.Logger
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

// Handler wraps the mcp-oauth library components for integration with inboxfewer
type Handler struct {
	server          *oauth.Server
	handler         *oauth.Handler
	store           *memory.Store
	ipRateLimiter   *security.RateLimiter
	userRateLimiter *security.RateLimiter
	clientRegRL     *security.ClientRegistrationRateLimiter
	stopped         bool // Track whether Stop has been called
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

	// Create memory storage
	store := memory.New()

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
	serverConfig := &oauth.ServerConfig{
		Issuer:                        config.BaseURL,
		RefreshTokenTTL:               int64(refreshTokenTTL.Seconds()),
		AllowRefreshTokenRotation:     true,  // OAuth 2.1 best practice
		RequirePKCE:                   true,  // OAuth 2.1 requirement
		AllowPKCEPlain:                false, // Only S256, not plain
		AllowPublicClientRegistration: config.Security.AllowPublicClientRegistration,
		RegistrationAccessToken:       config.Security.RegistrationAccessToken,
		MaxClientsPerIP:               maxClientsPerIP,
		TrustProxy:                    config.RateLimit.TrustProxy,
		TokenRefreshThreshold:         300, // 5 minutes proactive refresh
		ClockSkewGracePeriod:          5,   // 5 seconds clock skew tolerance
	}

	// Create OAuth server
	server, err := oauth.NewServer(
		provider,
		store, // TokenStore
		store, // ClientStore
		store, // FlowStore
		serverConfig,
		logger,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create OAuth server: %w", err)
	}

	// Set up encryption if key provided
	if len(config.Security.EncryptionKey) > 0 {
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

	// Set up client registration rate limiting
	clientRegRL := security.NewClientRegistrationRateLimiter(logger)
	server.SetClientRegistrationRateLimiter(clientRegRL)

	// Create HTTP handler
	handler := oauth.NewHandler(server, logger)

	return &Handler{
		server:          server,
		handler:         handler,
		store:           store,
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
func (h *Handler) GetStore() *memory.Store {
	return h.store
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
// This method is idempotent and can be safely called multiple times.
func (h *Handler) Stop() {
	if h.stopped {
		return // Already stopped
	}
	h.stopped = true

	if h.store != nil {
		h.store.Stop()
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
}
