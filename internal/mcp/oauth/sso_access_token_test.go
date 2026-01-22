package oauth

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"

	mcpoauth "github.com/giantswarm/mcp-oauth"
	"github.com/giantswarm/mcp-oauth/providers"
	"github.com/giantswarm/mcp-oauth/storage/memory"
)

func TestSSOAccessTokenMiddleware_NoUser(t *testing.T) {
	// Test that requests without authenticated user pass through without storing tokens
	store := memory.New()
	defer store.Stop()

	handler := SSOAccessTokenMiddleware(store, nil)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/mcp", nil)
	req.Header.Set(SSOAccessTokenHeader, "test-access-token")

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestSSOAccessTokenMiddleware_NoAccessToken(t *testing.T) {
	// Test that requests without X-Google-Access-Token header pass through normally
	store := memory.New()
	defer store.Stop()

	handler := SSOAccessTokenMiddleware(store, nil)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Create request with authenticated user context but no access token header
	req := httptest.NewRequest(http.MethodGet, "/mcp", nil)
	userInfo := &providers.UserInfo{
		Email: "test@example.com",
		Name:  "Test User",
	}
	req = req.WithContext(mcpoauth.ContextWithUserInfo(req.Context(), userInfo))

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	// Token should not be stored
	_, err := store.GetToken(req.Context(), "test@example.com")
	assert.Error(t, err)
}

func TestSSOAccessTokenMiddleware_StoresAccessToken(t *testing.T) {
	// Test that valid SSO access tokens are stored correctly
	store := memory.New()
	defer store.Stop()

	var handlerCalled bool
	handler := SSOAccessTokenMiddleware(store, nil)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		w.WriteHeader(http.StatusOK)
	}))

	// Create request with authenticated user context and access token header
	req := httptest.NewRequest(http.MethodGet, "/mcp", nil)
	req.Header.Set(SSOAccessTokenHeader, "forwarded-access-token")

	userInfo := &providers.UserInfo{
		Email: "sso-user@example.com",
		Name:  "SSO User",
	}
	req = req.WithContext(mcpoauth.ContextWithUserInfo(req.Context(), userInfo))

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.True(t, handlerCalled)

	// Token should be stored
	token, err := store.GetToken(req.Context(), "sso-user@example.com")
	require.NoError(t, err)
	assert.Equal(t, "forwarded-access-token", token.AccessToken)
	assert.Equal(t, "Bearer", token.TokenType)
	// Expiry should be approximately 1 hour from now (default)
	assert.WithinDuration(t, time.Now().Add(1*time.Hour), token.Expiry, 5*time.Second)
}

func TestSSOAccessTokenMiddleware_WithRefreshToken(t *testing.T) {
	// Test that refresh tokens are stored when provided
	store := memory.New()
	defer store.Stop()

	handler := SSOAccessTokenMiddleware(store, nil)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/mcp", nil)
	req.Header.Set(SSOAccessTokenHeader, "access-token")
	req.Header.Set(SSORefreshTokenHeader, "refresh-token")

	userInfo := &providers.UserInfo{
		Email: "refresh-user@example.com",
		Name:  "Refresh User",
	}
	req = req.WithContext(mcpoauth.ContextWithUserInfo(req.Context(), userInfo))

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	token, err := store.GetToken(req.Context(), "refresh-user@example.com")
	require.NoError(t, err)
	assert.Equal(t, "access-token", token.AccessToken)
	assert.Equal(t, "refresh-token", token.RefreshToken)
}

func TestSSOAccessTokenMiddleware_WithExpiry(t *testing.T) {
	// Test that custom expiry times are respected
	store := memory.New()
	defer store.Stop()

	handler := SSOAccessTokenMiddleware(store, nil)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	expectedExpiry := time.Now().Add(2 * time.Hour).UTC()

	req := httptest.NewRequest(http.MethodGet, "/mcp", nil)
	req.Header.Set(SSOAccessTokenHeader, "access-token")
	req.Header.Set(SSOTokenExpiryHeader, expectedExpiry.Format(time.RFC3339))

	userInfo := &providers.UserInfo{
		Email: "expiry-user@example.com",
		Name:  "Expiry User",
	}
	req = req.WithContext(mcpoauth.ContextWithUserInfo(req.Context(), userInfo))

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	token, err := store.GetToken(req.Context(), "expiry-user@example.com")
	require.NoError(t, err)
	// Allow 1 second tolerance for parsing/storage
	assert.WithinDuration(t, expectedExpiry, token.Expiry, 1*time.Second)
}

func TestSSOAccessTokenMiddleware_InvalidExpiry(t *testing.T) {
	// Test that invalid expiry format falls back to default
	store := memory.New()
	defer store.Stop()

	handler := SSOAccessTokenMiddleware(store, nil)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/mcp", nil)
	req.Header.Set(SSOAccessTokenHeader, "access-token")
	req.Header.Set(SSOTokenExpiryHeader, "invalid-date-format")

	userInfo := &providers.UserInfo{
		Email: "invalid-expiry@example.com",
		Name:  "Invalid Expiry User",
	}
	req = req.WithContext(mcpoauth.ContextWithUserInfo(req.Context(), userInfo))

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	token, err := store.GetToken(req.Context(), "invalid-expiry@example.com")
	require.NoError(t, err)
	// Should fall back to default ~1 hour expiry
	assert.WithinDuration(t, time.Now().Add(1*time.Hour), token.Expiry, 5*time.Second)
}

func TestSSOAccessTokenMiddleware_OverwritesExistingToken(t *testing.T) {
	// Test that new SSO tokens overwrite existing tokens
	store := memory.New()
	defer store.Stop()

	// Pre-store an existing token
	existingToken := &oauth2.Token{
		AccessToken: "old-access-token",
		TokenType:   "Bearer",
		Expiry:      time.Now().Add(30 * time.Minute),
	}
	err := store.SaveToken(context.Background(), "overwrite-user@example.com", existingToken)
	require.NoError(t, err)

	handler := SSOAccessTokenMiddleware(store, nil)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/mcp", nil)
	req.Header.Set(SSOAccessTokenHeader, "new-access-token")

	userInfo := &providers.UserInfo{
		Email: "overwrite-user@example.com",
		Name:  "Overwrite User",
	}
	req = req.WithContext(mcpoauth.ContextWithUserInfo(req.Context(), userInfo))

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	// Token should be updated with new value
	token, err := store.GetToken(req.Context(), "overwrite-user@example.com")
	require.NoError(t, err)
	assert.Equal(t, "new-access-token", token.AccessToken)
}

func TestParseTokenExpiry(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantNear time.Time
	}{
		{
			name:     "empty string uses default",
			input:    "",
			wantNear: time.Now().Add(1 * time.Hour),
		},
		{
			name:     "invalid format uses default",
			input:    "not-a-date",
			wantNear: time.Now().Add(1 * time.Hour),
		},
		{
			name:     "valid RFC3339",
			input:    "2024-01-20T15:04:05Z",
			wantNear: time.Date(2024, 1, 20, 15, 4, 5, 0, time.UTC),
		},
		{
			name:     "valid RFC3339 with timezone",
			input:    "2024-01-20T15:04:05+02:00",
			wantNear: time.Date(2024, 1, 20, 13, 4, 5, 0, time.UTC),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseTokenExpiry(tt.input)
			assert.WithinDuration(t, tt.wantNear, got, 5*time.Second)
		})
	}
}

func TestHashEmailForLog(t *testing.T) {
	tests := []struct {
		name  string
		email string
		want  string
	}{
		{
			name:  "empty email",
			email: "",
			want:  "",
		},
		{
			name:  "short email",
			email: "a@b.com",
			want:  "***",
		},
		{
			name:  "normal email",
			email: "testuser@example.com",
			want:  "te***@example.com",
		},
		{
			name:  "short local part",
			email: "ab@example.com",
			want:  "ab***@example.com",
		},
		{
			name:  "no at sign",
			email: "invalidemail",
			want:  "***",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := hashEmailForLog(tt.email)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestWrapWithSSOAccessToken(t *testing.T) {
	// Test the convenience wrapper function
	store := memory.New()
	defer store.Stop()

	var handlerCalled bool
	innerHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		w.WriteHeader(http.StatusOK)
	})

	wrapped := WrapWithSSOAccessToken(innerHandler, store, nil)
	require.NotNil(t, wrapped)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()

	wrapped.ServeHTTP(rec, req)

	assert.True(t, handlerCalled)
	assert.Equal(t, http.StatusOK, rec.Code)
}

// TestMiddlewareChainOrdering_Integration tests the correct middleware chain ordering.
// This is a critical integration test that verifies:
//   - ValidateToken middleware runs BEFORE SSOAccessToken middleware
//   - SSOAccessToken can see user info set by ValidateToken
//   - Access tokens are correctly stored when both middlewares are properly ordered
//
// The correct chain is: ValidateToken -> SSOAccessToken -> handler
// NOT: SSOAccessToken -> ValidateToken -> handler (which would fail)
func TestMiddlewareChainOrdering_Integration(t *testing.T) {
	store := memory.New()
	defer store.Stop()

	// Track what happened in each layer
	var (
		mcpHandlerCalled  bool
		userSeenInHandler string
	)

	// The innermost handler (MCP endpoint)
	mcpHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mcpHandlerCalled = true
		// Check if user info is available
		if userInfo, ok := GetUserFromContext(r.Context()); ok && userInfo != nil {
			userSeenInHandler = userInfo.Email
		}
		w.WriteHeader(http.StatusOK)
	})

	// Simulate ValidateToken middleware - sets user info in context
	// In production, this validates the JWT and extracts user info
	simulatedValidateToken := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Simulate successful token validation - set user info in context
			userInfo := &providers.UserInfo{
				Email: "integration-test@example.com",
				Name:  "Integration Test User",
			}
			ctx := mcpoauth.ContextWithUserInfo(r.Context(), userInfo)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}

	// Build the middleware chain in the CORRECT order:
	// Request -> ValidateToken -> SSOAccessToken -> mcpHandler
	//
	// Wrapping order (inside-out):
	// 1. SSO wraps mcpHandler
	// 2. ValidateToken wraps SSO
	ssoHandler := WrapWithSSOAccessToken(mcpHandler, store, nil)
	validatedHandler := simulatedValidateToken(ssoHandler)

	// Create request with SSO headers
	req := httptest.NewRequest(http.MethodGet, "/mcp", nil)
	req.Header.Set(SSOAccessTokenHeader, "integration-test-access-token")
	req.Header.Set(SSORefreshTokenHeader, "integration-test-refresh-token")

	rec := httptest.NewRecorder()
	validatedHandler.ServeHTTP(rec, req)

	// Verify the request completed successfully
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.True(t, mcpHandlerCalled, "MCP handler should have been called")
	assert.Equal(t, "integration-test@example.com", userSeenInHandler, "User info should be visible in handler")

	// Verify the access token was stored correctly
	token, err := store.GetToken(context.Background(), "integration-test@example.com")
	require.NoError(t, err, "Access token should have been stored")
	assert.Equal(t, "integration-test-access-token", token.AccessToken)
	assert.Equal(t, "integration-test-refresh-token", token.RefreshToken)
}

// TestMiddlewareChainOrdering_WrongOrder verifies that the WRONG middleware order fails to store tokens.
// This test documents the bug that was fixed: if SSOAccessToken runs before ValidateToken,
// no user info is available and tokens are not stored.
func TestMiddlewareChainOrdering_WrongOrder(t *testing.T) {
	store := memory.New()
	defer store.Stop()

	var mcpHandlerCalled bool

	mcpHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mcpHandlerCalled = true
		w.WriteHeader(http.StatusOK)
	})

	// Simulate ValidateToken middleware
	simulatedValidateToken := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userInfo := &providers.UserInfo{
				Email: "wrong-order-user@example.com",
				Name:  "Wrong Order User",
			}
			ctx := mcpoauth.ContextWithUserInfo(r.Context(), userInfo)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}

	// Build the middleware chain in the WRONG order:
	// Request -> SSOAccessToken -> ValidateToken -> mcpHandler
	//
	// This is wrong because SSOAccessToken runs BEFORE user info is set!
	validatedHandler := simulatedValidateToken(mcpHandler)
	ssoHandler := WrapWithSSOAccessToken(validatedHandler, store, nil) // WRONG: SSO wraps validated

	req := httptest.NewRequest(http.MethodGet, "/mcp", nil)
	req.Header.Set(SSOAccessTokenHeader, "should-not-be-stored")

	rec := httptest.NewRecorder()
	ssoHandler.ServeHTTP(rec, req)

	// Request still succeeds (passes through)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.True(t, mcpHandlerCalled)

	// But token was NOT stored because user info wasn't available when SSO middleware ran
	_, err := store.GetToken(context.Background(), "wrong-order-user@example.com")
	assert.Error(t, err, "Token should NOT be stored with wrong middleware order")
}

// TestSSOAccessTokenMiddleware_InjectsIntoContext verifies that the access token
// is injected into the request context via ContextWithGoogleAccessToken.
func TestSSOAccessTokenMiddleware_InjectsIntoContext(t *testing.T) {
	store := memory.New()
	defer store.Stop()

	var capturedToken string
	var tokenFound bool

	handler := SSOAccessTokenMiddleware(store, nil)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check if token was injected into context
		capturedToken, tokenFound = GetGoogleAccessTokenFromContext(r.Context())
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/mcp", nil)
	req.Header.Set(SSOAccessTokenHeader, "context-injected-token")

	userInfo := &providers.UserInfo{
		Email: "context-user@example.com",
		Name:  "Context User",
	}
	req = req.WithContext(mcpoauth.ContextWithUserInfo(req.Context(), userInfo))

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.True(t, tokenFound, "Token should be found in context")
	assert.Equal(t, "context-injected-token", capturedToken)
}

// mockSSOMetricsRecorder tracks SSO token injection metrics for testing
type mockSSOMetricsRecorder struct {
	results []string
}

func (m *mockSSOMetricsRecorder) RecordSSOTokenInjection(ctx context.Context, result string) {
	m.results = append(m.results, result)
}

// TestSSOAccessTokenMiddleware_WithMetrics verifies that metrics are recorded correctly.
func TestSSOAccessTokenMiddleware_WithMetrics(t *testing.T) {
	store := memory.New()
	defer store.Stop()

	metrics := &mockSSOMetricsRecorder{}

	handler := SSOAccessTokenMiddlewareWithConfig(&SSOMiddlewareConfig{
		Store:   store,
		Logger:  nil,
		Metrics: metrics,
	})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	t.Run("records no_user when user not authenticated", func(t *testing.T) {
		metrics.results = nil
		req := httptest.NewRequest(http.MethodGet, "/mcp", nil)
		req.Header.Set(SSOAccessTokenHeader, "test-token")

		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		require.Len(t, metrics.results, 1)
		assert.Equal(t, "no_user", metrics.results[0])
	})

	t.Run("records no_token when header not present", func(t *testing.T) {
		metrics.results = nil
		req := httptest.NewRequest(http.MethodGet, "/mcp", nil)
		// No access token header

		userInfo := &providers.UserInfo{
			Email: "notoken-user@example.com",
			Name:  "No Token User",
		}
		req = req.WithContext(mcpoauth.ContextWithUserInfo(req.Context(), userInfo))

		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		require.Len(t, metrics.results, 1)
		assert.Equal(t, "no_token", metrics.results[0])
	})

	t.Run("records stored when token is stored for non-SSO user", func(t *testing.T) {
		metrics.results = nil
		req := httptest.NewRequest(http.MethodGet, "/mcp", nil)
		req.Header.Set(SSOAccessTokenHeader, "stored-token")

		// Non-SSO user (IsSSO() returns false by default)
		userInfo := &providers.UserInfo{
			Email: "stored-user@example.com",
			Name:  "Stored User",
		}
		req = req.WithContext(mcpoauth.ContextWithUserInfo(req.Context(), userInfo))

		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		require.Len(t, metrics.results, 1)
		assert.Equal(t, "stored", metrics.results[0])
	})

	t.Run("records sso_success when IsSSO is true", func(t *testing.T) {
		metrics.results = nil
		req := httptest.NewRequest(http.MethodGet, "/mcp", nil)
		req.Header.Set(SSOAccessTokenHeader, "sso-token")

		// SSO user - set TokenSource to make IsSSO() return true
		userInfo := &providers.UserInfo{
			Email:       "sso-user@example.com",
			Name:        "SSO User",
			TokenSource: "sso", // This makes IsSSO() return true
		}
		req = req.WithContext(mcpoauth.ContextWithUserInfo(req.Context(), userInfo))

		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		require.Len(t, metrics.results, 1)
		assert.Equal(t, "sso_success", metrics.results[0])
	})
}

// TestWrapWithSSOAccessTokenAndMetrics tests the metrics-enabled wrapper function.
func TestWrapWithSSOAccessTokenAndMetrics(t *testing.T) {
	store := memory.New()
	defer store.Stop()

	metrics := &mockSSOMetricsRecorder{}

	var handlerCalled bool
	innerHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		w.WriteHeader(http.StatusOK)
	})

	wrapped := WrapWithSSOAccessTokenAndMetrics(innerHandler, store, nil, metrics)
	require.NotNil(t, wrapped)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()

	wrapped.ServeHTTP(rec, req)

	assert.True(t, handlerCalled)
	assert.Equal(t, http.StatusOK, rec.Code)
	// Should record no_user since no user info in context
	require.Len(t, metrics.results, 1)
	assert.Equal(t, "no_user", metrics.results[0])
}
