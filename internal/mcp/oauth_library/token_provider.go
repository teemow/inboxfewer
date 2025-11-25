package oauth_library

import (
	"context"

	"golang.org/x/oauth2"

	"github.com/giantswarm/mcp-oauth/storage"
)

// TokenProvider implements the server.TokenProvider interface using the mcp-oauth library's storage.
// It bridges the mcp-oauth storage with our existing server context that needs token access.
type TokenProvider struct {
	store storage.TokenStore
}

// NewTokenProvider creates a new token provider from an mcp-oauth TokenStore.
func NewTokenProvider(store storage.TokenStore) *TokenProvider {
	return &TokenProvider{
		store: store,
	}
}

// GetToken retrieves a Google OAuth token for the given user ID.
// This implements the server.TokenProvider interface.
func (p *TokenProvider) GetToken(ctx context.Context, userID string) (*oauth2.Token, error) {
	return p.store.GetToken(ctx, userID)
}

// GetTokenForAccount retrieves a Google OAuth token for the specified account.
// This implements the google.TokenProvider interface (account is typically an email address).
func (p *TokenProvider) GetTokenForAccount(ctx context.Context, account string) (*oauth2.Token, error) {
	return p.store.GetToken(ctx, account)
}

// HasTokenForAccount checks if a token exists for the specified account.
// This implements the google.TokenProvider interface.
func (p *TokenProvider) HasTokenForAccount(account string) bool {
	ctx := context.Background()
	_, err := p.store.GetToken(ctx, account)
	return err == nil
}

// SaveToken saves a Google OAuth token for the given user ID.
// This is used when tokens are refreshed or initially acquired.
func (p *TokenProvider) SaveToken(ctx context.Context, userID string, token *oauth2.Token) error {
	return p.store.SaveToken(ctx, userID, token)
}
