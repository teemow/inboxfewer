package oauth

import (
	"context"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"golang.org/x/oauth2"

	"github.com/giantswarm/mcp-oauth/storage"

	"github.com/teemow/inboxfewer/internal/instrumentation"
)

const (
	// SSOAccessTokenHeader is the HTTP header name for forwarded Google access tokens.
	// When SSO token forwarding is enabled, the upstream aggregator (e.g., muster) forwards
	// the user's Google access token in this header alongside the ID token in the
	// Authorization header.
	//
	// The ID token proves identity (validated via TrustedAudiences/JWKS),
	// while the access token provides Google API access with the required scopes.
	SSOAccessTokenHeader = "X-Google-Access-Token"

	// SSORefreshTokenHeader is the optional HTTP header name for forwarded Google refresh tokens.
	// If provided, enables automatic token refresh for long-running sessions.
	SSORefreshTokenHeader = "X-Google-Refresh-Token"

	// SSOTokenExpiryHeader is the optional HTTP header name for the access token expiry time.
	// Expected format: RFC3339 (e.g., "2024-01-20T15:04:05Z")
	// If not provided, a default expiry of 1 hour is assumed.
	SSOTokenExpiryHeader = "X-Google-Token-Expiry"

	// defaultAccessTokenExpiry is the default expiry duration for access tokens
	// when no expiry header is provided. Google access tokens typically expire in 1 hour.
	defaultAccessTokenExpiry = 1 * time.Hour

	// tokenStoreTimeout is the timeout for storing tokens in the token store.
	tokenStoreTimeout = 5 * time.Second
)

// SSOMetricsRecorder is an interface for recording SSO token injection metrics.
// This allows the middleware to record metrics without directly depending on the full Metrics type.
type SSOMetricsRecorder interface {
	RecordSSOTokenInjection(ctx context.Context, result string)
}

// SSOMiddlewareConfig holds configuration for the SSO access token middleware.
type SSOMiddlewareConfig struct {
	// Store is the token store to save forwarded access tokens
	Store storage.TokenStore

	// Logger for audit and debug logging (optional, uses slog.Default if nil)
	Logger *slog.Logger

	// Metrics for recording SSO token injection metrics (optional)
	Metrics SSOMetricsRecorder
}

// SSOAccessTokenMiddleware creates middleware that extracts and stores forwarded Google access tokens.
// This middleware should wrap handlers that are already protected by OAuth validation.
//
// When SSO token forwarding is enabled:
//  1. The upstream aggregator (e.g., muster) validates the user with Google OAuth
//  2. The aggregator forwards the ID token in the Authorization header (validated by TrustedAudiences)
//  3. The aggregator forwards the access token in X-Google-Access-Token header
//  4. This middleware extracts the access token, stores it for Google API calls, and injects it into context
//
// The middleware processes the access token if:
//   - The user was authenticated (user info present in context from OAuth middleware)
//   - The X-Google-Access-Token header is present and non-empty
//
// Token Handling:
//   - SSO flow (userInfo.IsSSO() == true): Token is injected into context AND stored
//   - Non-SSO flow: Token is stored only (for compatibility)
//
// Parameters:
//   - store: The token store to save forwarded access tokens
//   - logger: Logger for audit and debug logging (optional, uses slog.Default if nil)
func SSOAccessTokenMiddleware(store storage.TokenStore, logger *slog.Logger) func(http.Handler) http.Handler {
	return SSOAccessTokenMiddlewareWithConfig(&SSOMiddlewareConfig{
		Store:  store,
		Logger: logger,
	})
}

// SSOAccessTokenMiddlewareWithConfig creates middleware with full configuration including metrics.
// This is the preferred way to create the middleware when metrics are available.
func SSOAccessTokenMiddlewareWithConfig(config *SSOMiddlewareConfig) func(http.Handler) http.Handler {
	logger := config.Logger
	if logger == nil {
		logger = slog.Default()
	}

	// Helper to record metrics if configured
	recordMetric := func(ctx context.Context, result string) {
		if config.Metrics != nil {
			config.Metrics.RecordSSOTokenInjection(ctx, result)
		}
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()

			// Check if user is authenticated (set by OAuth ValidateToken middleware)
			userInfo, ok := GetUserFromContext(ctx)
			if !ok || userInfo == nil || userInfo.Email == "" {
				// User not authenticated - just pass through
				// The OAuth middleware will have already returned 401 if auth was required
				recordMetric(ctx, instrumentation.SSOInjectionResultNoUser)
				next.ServeHTTP(w, r)
				return
			}

			// Check for forwarded access token
			accessToken := r.Header.Get(SSOAccessTokenHeader)
			if accessToken == "" {
				// No forwarded access token - normal flow (user authenticated directly with inboxfewer)
				recordMetric(ctx, instrumentation.SSOInjectionResultNoToken)
				next.ServeHTTP(w, r)
				return
			}

			// Extract optional refresh token and expiry
			refreshToken := r.Header.Get(SSORefreshTokenHeader)
			expiry := parseTokenExpiry(r.Header.Get(SSOTokenExpiryHeader))

			// Build OAuth2 token
			token := &oauth2.Token{
				AccessToken:  accessToken,
				RefreshToken: refreshToken,
				TokenType:    "Bearer",
				Expiry:       expiry,
			}

			// Store the forwarded access token for this user
			storeCtx, cancel := context.WithTimeout(ctx, tokenStoreTimeout)
			storeErr := config.Store.SaveToken(storeCtx, userInfo.Email, token)
			cancel()

			if storeErr != nil {
				logger.Error("Failed to store forwarded SSO access token",
					"email", hashEmailForLog(userInfo.Email),
					"error", storeErr,
				)
				recordMetric(ctx, instrumentation.SSOInjectionResultStoreFailed)
				// Continue anyway - we can still inject the token into context
			} else {
				logger.Info("Stored forwarded SSO access token",
					"email", hashEmailForLog(userInfo.Email),
					"has_refresh_token", refreshToken != "",
					"expires_in", time.Until(expiry).Round(time.Second).String(),
					"is_sso", userInfo.IsSSO(),
				)
			}

			// Inject the access token into the request context for downstream use.
			// This allows MCP tools to access the Google token via GetGoogleAccessTokenFromContext()
			// without having to look it up from the store.
			//
			// This pattern mirrors mcp-kubernetes's ContextWithAccessToken approach,
			// enabling efficient token propagation through the request lifecycle.
			ctx = ContextWithGoogleAccessToken(ctx, accessToken)
			r = r.WithContext(ctx)

			// Use IsSSO() to detect SSO flow for metrics (mcp-oauth v0.2.43+)
			// This is more robust than just checking for header presence
			if userInfo.IsSSO() {
				logger.Debug("SSO token injection: using SSO-forwarded token",
					"email", hashEmailForLog(userInfo.Email))
				recordMetric(ctx, instrumentation.SSOInjectionResultSuccess)
			} else {
				// Non-SSO flow with access token header (unusual but supported)
				logger.Debug("SSO token injection: token stored for non-SSO user",
					"email", hashEmailForLog(userInfo.Email))
				if storeErr == nil {
					recordMetric(ctx, instrumentation.SSOInjectionResultStored)
				}
			}

			next.ServeHTTP(w, r)
		})
	}
}

// parseTokenExpiry parses the token expiry header value.
// Returns a default expiry of 1 hour from now if the value is empty or invalid.
func parseTokenExpiry(expiryStr string) time.Time {
	if expiryStr == "" {
		return time.Now().Add(defaultAccessTokenExpiry)
	}

	expiry, err := time.Parse(time.RFC3339, expiryStr)
	if err != nil {
		// Invalid format - use default
		return time.Now().Add(defaultAccessTokenExpiry)
	}

	return expiry
}

// hashEmailForLog returns a partially masked version of the email for logging.
// This prevents PII leakage in logs while still allowing correlation.
// Format: first 2 chars of local part + "***@" + full domain (e.g., "te***@example.com")
func hashEmailForLog(email string) string {
	if email == "" {
		return ""
	}

	// Short emails can't be meaningfully masked
	if len(email) <= 8 {
		return "***"
	}

	// Split email into local part and domain
	localPart, domain, found := strings.Cut(email, "@")
	if !found || localPart == "" || domain == "" {
		return "***"
	}

	// Show first 2 chars of local part and full domain
	if len(localPart) <= 2 {
		return localPart + "***@" + domain
	}
	return localPart[:2] + "***@" + domain
}

// WrapWithSSOAccessToken wraps an HTTP handler with SSO access token middleware.
// This is a convenience function that creates and applies the middleware.
func WrapWithSSOAccessToken(handler http.Handler, store storage.TokenStore, logger *slog.Logger) http.Handler {
	return SSOAccessTokenMiddleware(store, logger)(handler)
}

// WrapWithSSOAccessTokenAndMetrics wraps an HTTP handler with SSO access token middleware including metrics.
// This is the preferred way to wrap handlers when metrics are available.
func WrapWithSSOAccessTokenAndMetrics(handler http.Handler, store storage.TokenStore, logger *slog.Logger, metrics SSOMetricsRecorder) http.Handler {
	return SSOAccessTokenMiddlewareWithConfig(&SSOMiddlewareConfig{
		Store:   store,
		Logger:  logger,
		Metrics: metrics,
	})(handler)
}
