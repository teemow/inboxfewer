package oauth

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"

	"github.com/giantswarm/mcp-oauth/storage/memory"
)

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

	// Note: Testing the positive case (context with user info) would require
	// setting up the full OAuth middleware flow, which is covered in integration tests
}
