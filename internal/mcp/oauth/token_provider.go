package oauth

import (
	"context"
	"time"

	"golang.org/x/oauth2"

	oauth "github.com/giantswarm/mcp-oauth"
	"github.com/giantswarm/mcp-oauth/providers"
	"github.com/giantswarm/mcp-oauth/storage"

	"github.com/teemow/inboxfewer/internal/instrumentation"
)

// contextKey is a custom type for context keys to avoid collisions.
// Using a custom type instead of a plain string prevents key collisions
// with other packages that might use the same string key in the context.
type contextKey string

const (
	// googleAccessTokenKey is the context key for storing the user's Google access token.
	// This token is used for downstream Google API authentication.
	//nolint:gosec // G101 false positive - this is a context key name, not a credential
	googleAccessTokenKey contextKey = "google_access_token"
)

// TokenStore is a type alias for the mcp-oauth TokenStore interface.
// This allows external packages to use the interface without importing mcp-oauth directly.
type TokenStore = storage.TokenStore

// MetricsRecorder is an interface for recording OAuth-related metrics.
// This allows the token provider to record metrics without directly depending on the full Metrics type.
type MetricsRecorder interface {
	RecordOAuthTokenRefresh(ctx context.Context, result string)
}

// TokenProvider implements the server.TokenProvider interface using the mcp-oauth library's storage.
// It bridges the mcp-oauth storage with our existing server context that needs token access.
type TokenProvider struct {
	store   storage.TokenStore
	metrics MetricsRecorder
}

// NewTokenProvider creates a new token provider from an mcp-oauth TokenStore.
func NewTokenProvider(store storage.TokenStore) *TokenProvider {
	return &TokenProvider{
		store: store,
	}
}

// NewTokenProviderWithMetrics creates a new token provider with metrics recording.
// The metrics recorder will be called when tokens are retrieved, allowing tracking of
// token refresh operations (success/failure/expired).
func NewTokenProviderWithMetrics(store storage.TokenStore, metrics MetricsRecorder) *TokenProvider {
	return &TokenProvider{
		store:   store,
		metrics: metrics,
	}
}

// SetMetrics sets the metrics recorder for the token provider.
// This allows setting metrics after creation, useful for dependency injection.
func (p *TokenProvider) SetMetrics(metrics MetricsRecorder) {
	p.metrics = metrics
}

// GetToken retrieves a Google OAuth token for the given user ID.
// This implements the server.TokenProvider interface.
func (p *TokenProvider) GetToken(ctx context.Context, userID string) (*oauth2.Token, error) {
	token, err := p.store.GetToken(ctx, userID)

	// Record metrics if configured
	if p.metrics != nil {
		if err != nil {
			p.metrics.RecordOAuthTokenRefresh(ctx, instrumentation.OAuthResultFailure)
		} else if token != nil && token.Expiry.Before(time.Now()) {
			// Token was retrieved but is expired (will need refresh by the OAuth library)
			p.metrics.RecordOAuthTokenRefresh(ctx, instrumentation.OAuthResultExpired)
		} else {
			p.metrics.RecordOAuthTokenRefresh(ctx, instrumentation.OAuthResultSuccess)
		}
	}

	return token, err
}

// GetTokenForAccount retrieves a Google OAuth token for the specified account.
// This implements the google.TokenProvider interface (account is typically an email address).
func (p *TokenProvider) GetTokenForAccount(ctx context.Context, account string) (*oauth2.Token, error) {
	return p.GetToken(ctx, account)
}

// HasTokenForAccount checks if a token exists for the specified account.
// This implements the google.TokenProvider interface.
func (p *TokenProvider) HasTokenForAccount(account string) bool {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	// Use store directly to avoid recording metrics for existence checks
	_, err := p.store.GetToken(ctx, account)
	return err == nil
}

// SaveToken saves a Google OAuth token for the given user ID.
// This is used when tokens are refreshed or initially acquired.
func (p *TokenProvider) SaveToken(ctx context.Context, userID string, token *oauth2.Token) error {
	return p.store.SaveToken(ctx, userID, token)
}

// UserInfo represents Google user information.
// This is a convenience wrapper around the library's providers.UserInfo type.
type UserInfo = providers.UserInfo

// GetUserFromContext retrieves the authenticated user info from the request context.
// This is set by the OAuth middleware after validating the Bearer token.
// Returns the user info and true if present, or nil and false if not authenticated.
func GetUserFromContext(ctx context.Context) (*UserInfo, bool) {
	return oauth.UserInfoFromContext(ctx)
}

// ContextWithUserInfo creates a context with the given user info.
// This is useful for testing code that depends on authenticated user context.
func ContextWithUserInfo(ctx context.Context, userInfo *UserInfo) context.Context {
	return oauth.ContextWithUserInfo(ctx, userInfo)
}

// ContextWithGoogleAccessToken creates a context with the given Google access token.
// This is used to propagate the user's Google access token through the request context
// for downstream Google API authentication in MCP tools.
//
// This pattern mirrors mcp-kubernetes's ContextWithAccessToken, but is specifically
// for Google access tokens rather than Kubernetes ID tokens.
func ContextWithGoogleAccessToken(ctx context.Context, accessToken string) context.Context {
	return context.WithValue(ctx, googleAccessTokenKey, accessToken)
}

// GetGoogleAccessTokenFromContext retrieves the Google access token from the context.
// This returns the user's Google access token that can be used for Google API calls.
// Returns the access token and true if present, or empty string and false if not available.
//
// Usage in tool handlers:
//
//	token, ok := oauth.GetGoogleAccessTokenFromContext(ctx)
//	if !ok {
//	    return nil, fmt.Errorf("no Google access token available")
//	}
//	// Use token for Google API calls
func GetGoogleAccessTokenFromContext(ctx context.Context) (string, bool) {
	token, ok := ctx.Value(googleAccessTokenKey).(string)
	return token, ok && token != ""
}
