package google

import (
	"context"
	"fmt"
	"sync"
	"time"

	"golang.org/x/oauth2"
)

// NewTokenSourceForAccount returns a token source that follows the account's
// stored grant in the provider.
//
// A stored access token is never trusted as it is: the OAuth store keeps the
// per-user copy with an expiry extended for SSO forwarding (mcp-oauth's
// ProviderTokenTTL, 24 h by default) while Google's access tokens die after
// one hour, so a client built straight from that copy sends a dead token for
// the rest of the day. A grant that carries a refresh token therefore starts
// out expired: its first use refreshes, and the oauth2 client renews on
// Google's real schedule from then on.
//
// When a refresh fails (the grant was revoked), the source re-reads the
// provider once: a newer grant from a re-login replaces the revoked one, so a
// cached client recovers without a restart. A grant without a refresh token
// (an access token forwarded by an upstream) is used until it dies.
func NewTokenSourceForAccount(ctx context.Context, account string, provider TokenProvider) (oauth2.TokenSource, error) {
	return newTokenSourceForAccount(ctx, account, provider, getOAuthConfig())
}

func newTokenSourceForAccount(ctx context.Context, account string, provider TokenProvider, conf *oauth2.Config) (oauth2.TokenSource, error) {
	if provider == nil {
		return nil, fmt.Errorf("token provider cannot be nil")
	}
	s := &grantTokenSource{ctx: ctx, account: account, provider: provider, conf: conf}
	if _, err := s.reseed(); err != nil {
		return nil, err
	}
	return s, nil
}

// grantTokenSource is an oauth2.TokenSource over the provider's stored grant.
type grantTokenSource struct {
	ctx      context.Context
	account  string
	provider TokenProvider
	conf     *oauth2.Config

	mu      sync.Mutex
	grant   string             // fingerprint of the stored grant current was seeded from
	current oauth2.TokenSource // refreshing source for that grant
}

// Token returns a valid token for the account, refreshing through the stored
// grant's refresh token when needed.
func (s *grantTokenSource) Token() (*oauth2.Token, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	token, err := s.current.Token()
	if err == nil {
		return token, nil
	}

	// The grant may have been revoked; a re-login since then stored a new one.
	changed, reseedErr := s.reseed()
	if reseedErr != nil || !changed {
		return nil, err
	}
	return s.current.Token()
}

// reseed reads the stored grant and, when it differs from the one in use,
// replaces the refreshing source. It reports whether the grant changed.
// Callers other than the constructor hold s.mu.
func (s *grantTokenSource) reseed() (bool, error) {
	stored, err := s.provider.GetTokenForAccount(s.ctx, s.account)
	if err != nil {
		return false, fmt.Errorf("failed to get Google OAuth token for account %s: %w", s.account, err)
	}
	fingerprint := stored.RefreshToken + "\x00" + stored.AccessToken
	if s.current != nil && fingerprint == s.grant {
		return false, nil
	}
	s.current = s.conf.TokenSource(s.ctx, seedToken(stored))
	s.grant = fingerprint
	return true, nil
}

// seedToken returns the token the refreshing source starts from. A grant with
// a refresh token starts out expired so that its first use refreshes (the
// stored expiry is not Google's); one without is used as it is.
func seedToken(stored *oauth2.Token) *oauth2.Token {
	if stored.RefreshToken == "" {
		return stored
	}
	seed := *stored
	seed.Expiry = time.Unix(1, 0)
	return &seed
}
