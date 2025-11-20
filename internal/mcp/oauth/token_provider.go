package oauth

import (
	"context"
	"fmt"

	"golang.org/x/oauth2"
)

// TokenProvider implements google.TokenProvider using the OAuth store
// This allows Google API clients to use tokens from OAuth authentication
type TokenProvider struct {
	store *Store
}

// NewTokenProvider creates a new OAuth-based token provider
func NewTokenProvider(store *Store) *TokenProvider {
	return &TokenProvider{
		store: store,
	}
}

// GetTokenForAccount retrieves a Google OAuth token from the store
func (p *TokenProvider) GetTokenForAccount(ctx context.Context, account string) (*oauth2.Token, error) {
	token, err := p.store.GetGoogleToken(account)
	if err != nil {
		return nil, fmt.Errorf("no Google OAuth token found for account %s. Please authenticate with Google through your MCP client", account)
	}
	return token, nil
}

// HasTokenForAccount checks if a token exists in the store for the specified account
func (p *TokenProvider) HasTokenForAccount(account string) bool {
	_, err := p.store.GetGoogleToken(account)
	return err == nil
}

