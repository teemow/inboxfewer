package google

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"golang.org/x/oauth2"
)

// fakeProvider is a TokenProvider whose stored grant the test can swap.
type fakeProvider struct {
	mu    sync.Mutex
	token *oauth2.Token
	err   error
}

func (p *fakeProvider) GetTokenForAccount(_ context.Context, _ string) (*oauth2.Token, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.err != nil {
		return nil, p.err
	}
	return p.token, nil
}

func (p *fakeProvider) HasTokenForAccount(_ string) bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.token != nil
}

func (p *fakeProvider) set(token *oauth2.Token) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.token = token
}

// tokenEndpoint is a fake Google token endpoint that answers refresh requests
// per refresh token: a refresh token mapped to "" is revoked (invalid_grant).
type tokenEndpoint struct {
	mu       sync.Mutex
	answers  map[string]string // refresh token -> access token it mints
	requests []string          // refresh tokens seen
}

func (e *tokenEndpoint) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	rt := r.Form.Get("refresh_token")
	e.requests = append(e.requests, rt)
	w.Header().Set("Content-Type", "application/json")
	access, ok := e.answers[rt]
	if !ok || access == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid_grant"})
		return
	}
	_ = json.NewEncoder(w).Encode(map[string]any{
		"access_token": access,
		"token_type":   "Bearer",
		"expires_in":   3600,
	})
}

func (e *tokenEndpoint) calls() int {
	e.mu.Lock()
	defer e.mu.Unlock()
	return len(e.requests)
}

func newTestSource(t *testing.T, provider *fakeProvider, endpoint *tokenEndpoint) oauth2.TokenSource {
	t.Helper()
	srv := httptest.NewServer(endpoint)
	t.Cleanup(srv.Close)
	conf := &oauth2.Config{
		ClientID:     "client",
		ClientSecret: "secret",
		Endpoint:     oauth2.Endpoint{TokenURL: srv.URL + "/token", AuthStyle: oauth2.AuthStyleInParams},
	}
	source, err := newTokenSourceForAccount(context.Background(), "user@example.com", provider, conf)
	if err != nil {
		t.Fatalf("newTokenSourceForAccount: %v", err)
	}
	return source
}

func TestGrantTokenSource_RefreshesStoredGrantOnFirstUse(t *testing.T) {
	// The stored copy claims a whole day of validity; Google's token is long dead.
	provider := &fakeProvider{token: &oauth2.Token{
		AccessToken:  "stale",
		RefreshToken: "rt-1",
		TokenType:    "Bearer",
		Expiry:       time.Now().Add(24 * time.Hour),
	}}
	endpoint := &tokenEndpoint{answers: map[string]string{"rt-1": "fresh-1"}}
	source := newTestSource(t, provider, endpoint)

	token, err := source.Token()
	if err != nil {
		t.Fatalf("first Token(): %v", err)
	}
	if token.AccessToken != "fresh-1" {
		t.Fatalf("first Token() = %q, want the refreshed token", token.AccessToken)
	}
	if endpoint.calls() != 1 {
		t.Fatalf("token endpoint calls = %d, want 1", endpoint.calls())
	}

	again, err := source.Token()
	if err != nil {
		t.Fatalf("second Token(): %v", err)
	}
	if again.AccessToken != "fresh-1" || endpoint.calls() != 1 {
		t.Fatalf("second Token() = %q after %d endpoint calls, want the cached token and 1 call", again.AccessToken, endpoint.calls())
	}
}

func TestGrantTokenSource_ReseedsFromReloginAfterRevocation(t *testing.T) {
	provider := &fakeProvider{token: &oauth2.Token{AccessToken: "stale", RefreshToken: "rt-1", TokenType: "Bearer"}}
	endpoint := &tokenEndpoint{answers: map[string]string{"rt-1": "", "rt-2": "fresh-2"}}
	source := newTestSource(t, provider, endpoint)

	if _, err := source.Token(); err == nil {
		t.Fatal("Token() with a revoked grant and no re-login succeeded, want an error")
	}

	// The person signed in again: the store now holds a new grant.
	provider.set(&oauth2.Token{AccessToken: "stale-2", RefreshToken: "rt-2", TokenType: "Bearer"})

	token, err := source.Token()
	if err != nil {
		t.Fatalf("Token() after re-login: %v", err)
	}
	if token.AccessToken != "fresh-2" {
		t.Fatalf("Token() after re-login = %q, want the token refreshed from the new grant", token.AccessToken)
	}
}

func TestGrantTokenSource_AccessTokenOnlyGrantIsUsedAsIs(t *testing.T) {
	provider := &fakeProvider{token: &oauth2.Token{
		AccessToken: "forwarded",
		TokenType:   "Bearer",
		Expiry:      time.Now().Add(time.Hour),
	}}
	endpoint := &tokenEndpoint{answers: map[string]string{}}
	source := newTestSource(t, provider, endpoint)

	token, err := source.Token()
	if err != nil {
		t.Fatalf("Token(): %v", err)
	}
	if token.AccessToken != "forwarded" || endpoint.calls() != 0 {
		t.Fatalf("Token() = %q after %d endpoint calls, want the forwarded token and no call", token.AccessToken, endpoint.calls())
	}
}

func TestNewTokenSourceForAccount_NoStoredGrant(t *testing.T) {
	provider := &fakeProvider{err: errors.New("token not found")}
	_, err := newTokenSourceForAccount(context.Background(), "user@example.com", provider, &oauth2.Config{})
	if err == nil {
		t.Fatal("newTokenSourceForAccount without a stored grant succeeded, want an error")
	}
	if _, err := newTokenSourceForAccount(context.Background(), "user@example.com", nil, &oauth2.Config{}); err == nil {
		t.Fatal("newTokenSourceForAccount with a nil provider succeeded, want an error")
	}
}
