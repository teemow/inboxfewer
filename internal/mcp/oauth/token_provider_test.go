package oauth

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"

	"github.com/giantswarm/mcp-oauth/storage/memory"

	"github.com/teemow/inboxfewer/internal/instrumentation"
)

// mockMetricsRecorder implements MetricsRecorder for testing
type mockMetricsRecorder struct {
	tokenRefreshCalls []string
}

func (m *mockMetricsRecorder) RecordOAuthTokenRefresh(ctx context.Context, result string) {
	m.tokenRefreshCalls = append(m.tokenRefreshCalls, result)
}

func TestTokenProvider(t *testing.T) {
	// Create storage
	store := memory.New()
	defer store.Stop()

	// Create token provider
	provider := NewTokenProvider(store)
	require.NotNil(t, provider)

	ctx := context.Background()
	userID := "test-user@example.com"

	// Test saving and getting a token
	token := &oauth2.Token{
		AccessToken:  "test-access-token",
		RefreshToken: "test-refresh-token",
		TokenType:    "Bearer",
		Expiry:       time.Now().Add(time.Hour),
	}

	err := provider.SaveToken(ctx, userID, token)
	require.NoError(t, err)

	// Retrieve the token
	retrievedToken, err := provider.GetToken(ctx, userID)
	require.NoError(t, err)
	assert.Equal(t, token.AccessToken, retrievedToken.AccessToken)
	assert.Equal(t, token.RefreshToken, retrievedToken.RefreshToken)
	assert.Equal(t, token.TokenType, retrievedToken.TokenType)
}

func TestTokenProvider_NonExistentUser(t *testing.T) {
	// Create storage
	store := memory.New()
	defer store.Stop()

	// Create token provider
	provider := NewTokenProvider(store)

	ctx := context.Background()

	// Try to get token for non-existent user
	_, err := provider.GetToken(ctx, "nonexistent@example.com")
	assert.Error(t, err)
}

func TestTokenProvider_HasTokenForAccount(t *testing.T) {
	// Create storage
	store := memory.New()
	defer store.Stop()

	// Create token provider
	provider := NewTokenProvider(store)

	ctx := context.Background()
	userID := "test-user@example.com"

	// Initially should return false
	assert.False(t, provider.HasTokenForAccount(userID))

	// Save a token
	token := &oauth2.Token{
		AccessToken:  "test-access-token",
		RefreshToken: "test-refresh-token",
		TokenType:    "Bearer",
		Expiry:       time.Now().Add(time.Hour),
	}
	err := provider.SaveToken(ctx, userID, token)
	require.NoError(t, err)

	// Now should return true
	assert.True(t, provider.HasTokenForAccount(userID))
}

func TestGetUserFromContext(t *testing.T) {
	// Test with context that has no user info
	ctx := context.Background()
	user, ok := GetUserFromContext(ctx)
	assert.False(t, ok)
	assert.Nil(t, user)

	// Test with context that has user info
	userInfo := &UserInfo{
		ID:            "user-123",
		Email:         "test@example.com",
		EmailVerified: true,
		Name:          "Test User",
	}
	ctxWithUser := ContextWithUserInfo(ctx, userInfo)

	retrievedUser, ok := GetUserFromContext(ctxWithUser)
	assert.True(t, ok)
	assert.NotNil(t, retrievedUser)
	assert.Equal(t, userInfo.Email, retrievedUser.Email)
	assert.Equal(t, userInfo.ID, retrievedUser.ID)
	assert.Equal(t, userInfo.Name, retrievedUser.Name)
}

func TestContextWithUserInfo(t *testing.T) {
	ctx := context.Background()

	// Test with valid user info
	userInfo := &UserInfo{
		ID:            "user-456",
		Email:         "another@example.com",
		EmailVerified: true,
	}
	ctxWithUser := ContextWithUserInfo(ctx, userInfo)

	// Verify the user info can be retrieved
	retrieved, ok := GetUserFromContext(ctxWithUser)
	assert.True(t, ok)
	assert.Equal(t, userInfo.Email, retrieved.Email)

	// Test with nil user info
	// When nil is passed, the context key is set but the value is nil.
	// The expected behavior is that retrievedNil should be nil.
	ctxWithNil := ContextWithUserInfo(ctx, nil)
	retrievedNil, _ := GetUserFromContext(ctxWithNil)
	assert.Nil(t, retrievedNil, "GetUserFromContext should return nil when ContextWithUserInfo was called with nil")
}

func TestTokenProviderWithMetrics_Success(t *testing.T) {
	// Create storage
	store := memory.New()
	defer store.Stop()

	// Create mock metrics recorder
	metrics := &mockMetricsRecorder{}

	// Create token provider with metrics
	provider := NewTokenProviderWithMetrics(store, metrics)
	require.NotNil(t, provider)

	ctx := context.Background()
	userID := "test-user@example.com"

	// Save a valid (non-expired) token
	token := &oauth2.Token{
		AccessToken:  "test-access-token",
		RefreshToken: "test-refresh-token",
		TokenType:    "Bearer",
		Expiry:       time.Now().Add(time.Hour),
	}
	err := provider.SaveToken(ctx, userID, token)
	require.NoError(t, err)

	// Get the token - should record success
	_, err = provider.GetToken(ctx, userID)
	require.NoError(t, err)

	// Verify metrics were recorded
	require.Len(t, metrics.tokenRefreshCalls, 1)
	assert.Equal(t, instrumentation.OAuthResultSuccess, metrics.tokenRefreshCalls[0])
}

func TestTokenProviderWithMetrics_Expired(t *testing.T) {
	// Create storage
	store := memory.New()
	defer store.Stop()

	// Create mock metrics recorder
	metrics := &mockMetricsRecorder{}

	// Create token provider with metrics
	provider := NewTokenProviderWithMetrics(store, metrics)

	ctx := context.Background()
	userID := "test-user@example.com"

	// Save an expired token
	token := &oauth2.Token{
		AccessToken:  "test-access-token",
		RefreshToken: "test-refresh-token",
		TokenType:    "Bearer",
		Expiry:       time.Now().Add(-time.Hour), // Expired 1 hour ago
	}
	err := provider.SaveToken(ctx, userID, token)
	require.NoError(t, err)

	// Get the token - should record expired
	_, err = provider.GetToken(ctx, userID)
	require.NoError(t, err)

	// Verify metrics were recorded as expired
	require.Len(t, metrics.tokenRefreshCalls, 1)
	assert.Equal(t, instrumentation.OAuthResultExpired, metrics.tokenRefreshCalls[0])
}

func TestTokenProviderWithMetrics_Failure(t *testing.T) {
	// Create storage
	store := memory.New()
	defer store.Stop()

	// Create mock metrics recorder
	metrics := &mockMetricsRecorder{}

	// Create token provider with metrics
	provider := NewTokenProviderWithMetrics(store, metrics)

	ctx := context.Background()

	// Try to get a non-existent token - should record failure
	_, err := provider.GetToken(ctx, "nonexistent@example.com")
	require.Error(t, err)

	// Verify metrics were recorded as failure
	require.Len(t, metrics.tokenRefreshCalls, 1)
	assert.Equal(t, instrumentation.OAuthResultFailure, metrics.tokenRefreshCalls[0])
}

func TestTokenProviderWithMetrics_SetMetrics(t *testing.T) {
	// Create storage
	store := memory.New()
	defer store.Stop()

	// Create token provider without metrics
	provider := NewTokenProvider(store)

	ctx := context.Background()
	userID := "test-user@example.com"

	// Save a valid token
	token := &oauth2.Token{
		AccessToken:  "test-access-token",
		RefreshToken: "test-refresh-token",
		TokenType:    "Bearer",
		Expiry:       time.Now().Add(time.Hour),
	}
	err := provider.SaveToken(ctx, userID, token)
	require.NoError(t, err)

	// Get token without metrics - should not record anything
	_, err = provider.GetToken(ctx, userID)
	require.NoError(t, err)

	// Set metrics
	metrics := &mockMetricsRecorder{}
	provider.SetMetrics(metrics)

	// Get token again - now should record metrics
	_, err = provider.GetToken(ctx, userID)
	require.NoError(t, err)

	// Verify metrics were recorded
	require.Len(t, metrics.tokenRefreshCalls, 1)
	assert.Equal(t, instrumentation.OAuthResultSuccess, metrics.tokenRefreshCalls[0])
}

func TestTokenProviderWithMetrics_HasTokenForAccountNoMetrics(t *testing.T) {
	// Create storage
	store := memory.New()
	defer store.Stop()

	// Create mock metrics recorder
	metrics := &mockMetricsRecorder{}

	// Create token provider with metrics
	provider := NewTokenProviderWithMetrics(store, metrics)

	ctx := context.Background()
	userID := "test-user@example.com"

	// Save a valid token
	token := &oauth2.Token{
		AccessToken:  "test-access-token",
		RefreshToken: "test-refresh-token",
		TokenType:    "Bearer",
		Expiry:       time.Now().Add(time.Hour),
	}
	err := provider.SaveToken(ctx, userID, token)
	require.NoError(t, err)

	// Check token existence - should NOT record metrics
	// (existence checks shouldn't count as token refresh operations)
	exists := provider.HasTokenForAccount(userID)
	assert.True(t, exists)

	// Verify no metrics were recorded
	assert.Len(t, metrics.tokenRefreshCalls, 0)
}

func TestTokenProviderWithMetrics_GetTokenForAccount(t *testing.T) {
	// Create storage
	store := memory.New()
	defer store.Stop()

	// Create mock metrics recorder
	metrics := &mockMetricsRecorder{}

	// Create token provider with metrics
	provider := NewTokenProviderWithMetrics(store, metrics)

	ctx := context.Background()
	userID := "test-user@example.com"

	// Save a valid token
	token := &oauth2.Token{
		AccessToken:  "test-access-token",
		RefreshToken: "test-refresh-token",
		TokenType:    "Bearer",
		Expiry:       time.Now().Add(time.Hour),
	}
	err := provider.SaveToken(ctx, userID, token)
	require.NoError(t, err)

	// Use GetTokenForAccount (the google.TokenProvider interface method)
	_, err = provider.GetTokenForAccount(ctx, userID)
	require.NoError(t, err)

	// Verify metrics were recorded (GetTokenForAccount delegates to GetToken)
	require.Len(t, metrics.tokenRefreshCalls, 1)
	assert.Equal(t, instrumentation.OAuthResultSuccess, metrics.tokenRefreshCalls[0])
}
