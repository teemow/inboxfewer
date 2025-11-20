package google

import (
	"context"
	"fmt"

	"golang.org/x/oauth2"
)

// TokenProvider is an interface for providing OAuth tokens for Google APIs
// This abstraction allows different token sources (file-based, OAuth store, etc.)
type TokenProvider interface {
	// GetTokenForAccount retrieves an OAuth token for the specified account
	GetTokenForAccount(ctx context.Context, account string) (*oauth2.Token, error)

	// HasTokenForAccount checks if a token exists for the specified account
	HasTokenForAccount(account string) bool
}

// FileTokenProvider provides tokens from disk files (for STDIO transport)
type FileTokenProvider struct{}

// NewFileTokenProvider creates a new file-based token provider
func NewFileTokenProvider() *FileTokenProvider {
	return &FileTokenProvider{}
}

// GetTokenForAccount retrieves a token from disk for the specified account
func (p *FileTokenProvider) GetTokenForAccount(ctx context.Context, account string) (*oauth2.Token, error) {
	ts, err := GetTokenSourceForAccount(ctx, account)
	if err != nil {
		return nil, err
	}

	token, err := ts.Token()
	if err != nil {
		return nil, fmt.Errorf("failed to get token from file: %w", err)
	}

	return token, nil
}

// HasTokenForAccount checks if a token file exists for the specified account
func (p *FileTokenProvider) HasTokenForAccount(account string) bool {
	return HasTokenForAccount(account)
}
