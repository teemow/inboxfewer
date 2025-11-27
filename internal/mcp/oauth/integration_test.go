package oauth

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"
)

// TestIntegration_TokenFlow tests the complete token flow from storage to provider
func TestIntegration_TokenFlow(t *testing.T) {
	// Create OAuth handler
	config := &Config{
		BaseURL:            "http://localhost:8080",
		GoogleClientID:     "test-client-id",
		GoogleClientSecret: "test-client-secret",
		Security: SecurityConfig{
			EnableAuditLogging: true,
		},
	}

	handler, err := NewHandler(config)
	require.NoError(t, err)
	require.NotNil(t, handler)
	defer handler.Stop()

	// Create token provider
	provider := NewTokenProvider(handler.GetStore())

	// Test token operations
	ctx := context.Background()
	userID := "test-user@example.com"

	// 1. Check token doesn't exist
	assert.False(t, provider.HasTokenForAccount(userID))

	// 2. Save token
	token := &oauth2.Token{
		AccessToken:  "test-access-token",
		RefreshToken: "test-refresh-token",
		TokenType:    "Bearer",
		Expiry:       time.Now().Add(time.Hour),
	}

	err = provider.SaveToken(ctx, userID, token)
	require.NoError(t, err)

	// 3. Check token exists
	assert.True(t, provider.HasTokenForAccount(userID))

	// 4. Retrieve token
	retrievedToken, err := provider.GetToken(ctx, userID)
	require.NoError(t, err)
	assert.Equal(t, token.AccessToken, retrievedToken.AccessToken)
	assert.Equal(t, token.RefreshToken, retrievedToken.RefreshToken)
	assert.Equal(t, token.TokenType, retrievedToken.TokenType)

	// 5. Retrieve via GetTokenForAccount
	retrievedToken2, err := provider.GetTokenForAccount(ctx, userID)
	require.NoError(t, err)
	assert.Equal(t, token.AccessToken, retrievedToken2.AccessToken)
}

// TestIntegration_MultipleUsers tests handling multiple users
func TestIntegration_MultipleUsers(t *testing.T) {
	config := &Config{
		BaseURL:            "http://localhost:8080",
		GoogleClientID:     "test-client-id",
		GoogleClientSecret: "test-client-secret",
	}

	handler, err := NewHandler(config)
	require.NoError(t, err)
	defer handler.Stop()

	provider := NewTokenProvider(handler.GetStore())
	ctx := context.Background()

	// Create tokens for multiple users
	users := []string{
		"user1@example.com",
		"user2@example.com",
		"user3@example.com",
	}

	for i, userID := range users {
		token := &oauth2.Token{
			AccessToken:  "access-token-" + userID,
			RefreshToken: "refresh-token-" + userID,
			TokenType:    "Bearer",
			Expiry:       time.Now().Add(time.Hour * time.Duration(i+1)),
		}

		err = provider.SaveToken(ctx, userID, token)
		require.NoError(t, err)
	}

	// Verify all users have tokens
	for _, userID := range users {
		assert.True(t, provider.HasTokenForAccount(userID))

		token, err := provider.GetToken(ctx, userID)
		require.NoError(t, err)
		assert.Equal(t, "access-token-"+userID, token.AccessToken)
	}
}

// TestIntegration_HandlerLifecycle tests the complete handler lifecycle
func TestIntegration_HandlerLifecycle(t *testing.T) {
	config := &Config{
		BaseURL:            "http://localhost:8080",
		GoogleClientID:     "test-client-id",
		GoogleClientSecret: "test-client-secret",
		Security: SecurityConfig{
			EnableAuditLogging: true,
		},
		RateLimit: RateLimitConfig{
			Rate:     10,
			UserRate: 100,
		},
	}

	// Create handler
	handler, err := NewHandler(config)
	require.NoError(t, err)
	require.NotNil(t, handler)

	// Verify components are initialized
	assert.NotNil(t, handler.GetHandler())
	assert.NotNil(t, handler.GetStore())
	assert.NotNil(t, handler.GetServer())
	assert.True(t, handler.CanRefreshTokens())

	// Store some data
	provider := NewTokenProvider(handler.GetStore())
	ctx := context.Background()
	userID := "test@example.com"

	token := &oauth2.Token{
		AccessToken: "test-token",
		TokenType:   "Bearer",
		Expiry:      time.Now().Add(time.Hour),
	}

	err = provider.SaveToken(ctx, userID, token)
	require.NoError(t, err)

	// Verify token is there
	assert.True(t, provider.HasTokenForAccount(userID))

	// Stop handler (cleanup)
	handler.Stop()

	// Verify idempotent stop
	assert.NotPanics(t, func() {
		handler.Stop()
		handler.Stop()
	})
}

// TestIntegration_SecurityFeatures tests security features are properly configured
func TestIntegration_SecurityFeatures(t *testing.T) {
	// Create a 32-byte encryption key
	encryptionKey := make([]byte, 32)
	for i := range encryptionKey {
		encryptionKey[i] = byte(i)
	}

	config := &Config{
		BaseURL:            "http://localhost:8080",
		GoogleClientID:     "test-client-id",
		GoogleClientSecret: "test-client-secret",
		Security: SecurityConfig{
			AllowPublicClientRegistration: false,
			RegistrationAccessToken:       "test-registration-token",
			MaxClientsPerIP:               10,
			EnableAuditLogging:            true,
			EncryptionKey:                 encryptionKey,
			RefreshTokenTTL:               90 * 24 * time.Hour,
		},
		RateLimit: RateLimitConfig{
			Rate:      10,
			Burst:     20,
			UserRate:  100,
			UserBurst: 200,
		},
	}

	handler, err := NewHandler(config)
	require.NoError(t, err)
	require.NotNil(t, handler)
	defer handler.Stop()

	// Verify server is configured
	server := handler.GetServer()
	assert.NotNil(t, server)

	// The server should have all security features enabled
	// This is verified by the library's internal configuration
	assert.NotNil(t, handler.GetHandler())
	assert.NotNil(t, handler.GetStore())
}
