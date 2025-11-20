package oauth

import (
	"fmt"
	"sync"
	"time"

	"golang.org/x/oauth2"
)

// Store manages OAuth clients, tokens, and authorization codes in memory
type Store struct {
	mu                 sync.RWMutex
	clients            map[string]*ClientInfo
	tokens             map[string]*Token // access token -> Token
	refreshTokens      map[string]*Token // refresh token -> Token
	authorizationCodes map[string]*AuthorizationCode
	googleTokens       map[string]*oauth2.Token   // user email -> Google token
	googleUserInfo     map[string]*GoogleUserInfo // user email -> Google user info
	cleanupInterval    time.Duration              // How often to cleanup expired tokens
}

// NewStore creates a new in-memory OAuth store with default cleanup interval
func NewStore() *Store {
	return NewStoreWithInterval(1 * time.Minute)
}

// NewStoreWithInterval creates a new in-memory OAuth store with custom cleanup interval
func NewStoreWithInterval(cleanupInterval time.Duration) *Store {
	s := &Store{
		clients:            make(map[string]*ClientInfo),
		tokens:             make(map[string]*Token),
		refreshTokens:      make(map[string]*Token),
		authorizationCodes: make(map[string]*AuthorizationCode),
		googleTokens:       make(map[string]*oauth2.Token),
		googleUserInfo:     make(map[string]*GoogleUserInfo),
		cleanupInterval:    cleanupInterval,
	}

	// Start background cleanup goroutine
	go s.cleanupExpiredTokens()

	return s
}

// SaveClient saves a client registration
func (s *Store) SaveClient(client *ClientInfo) error {
	if client.ClientID == "" {
		return fmt.Errorf("client ID cannot be empty")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.clients[client.ClientID] = client
	return nil
}

// GetClient retrieves a client by ID
func (s *Store) GetClient(clientID string) (*ClientInfo, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	client, ok := s.clients[clientID]
	if !ok {
		return nil, fmt.Errorf("client not found: %s", clientID)
	}

	return client, nil
}

// DeleteClient removes a client registration
func (s *Store) DeleteClient(clientID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.clients[clientID]; !ok {
		return fmt.Errorf("client not found: %s", clientID)
	}

	delete(s.clients, clientID)

	// Also clean up any tokens for this client
	for tokenValue, token := range s.tokens {
		if token.ClientID == clientID {
			delete(s.tokens, tokenValue)
		}
	}

	for refreshToken, token := range s.refreshTokens {
		if token.ClientID == clientID {
			delete(s.refreshTokens, refreshToken)
		}
	}

	return nil
}

// ListClients returns all registered clients
func (s *Store) ListClients() []*ClientInfo {
	s.mu.RLock()
	defer s.mu.RUnlock()

	clients := make([]*ClientInfo, 0, len(s.clients))
	for _, client := range s.clients {
		clients = append(clients, client)
	}

	return clients
}

// ValidateClientCredentials checks if the client ID and secret are valid
func (s *Store) ValidateClientCredentials(clientID, clientSecret string) (*ClientInfo, error) {
	client, err := s.GetClient(clientID)
	if err != nil {
		return nil, err
	}

	// Public clients don't have secrets
	if client.IsPublic {
		return client, nil
	}

	// Confidential clients must provide correct secret
	if client.ClientSecret != clientSecret {
		return nil, fmt.Errorf("invalid client credentials")
	}

	return client, nil
}

// SaveAuthorizationCode saves a pending authorization
func (s *Store) SaveAuthorizationCode(code *AuthorizationCode) error {
	if code.Code == "" {
		return fmt.Errorf("authorization code cannot be empty")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.authorizationCodes[code.Code] = code
	return nil
}

// GetAuthorizationCode retrieves an authorization code
func (s *Store) GetAuthorizationCode(code string) (*AuthorizationCode, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	authCode, ok := s.authorizationCodes[code]
	if !ok {
		return nil, fmt.Errorf("authorization code not found")
	}

	if authCode.IsExpired() {
		return nil, fmt.Errorf("authorization code expired")
	}

	if authCode.Used {
		return nil, fmt.Errorf("authorization code already used")
	}

	return authCode, nil
}

// MarkAuthorizationCodeUsed marks an authorization code as used
func (s *Store) MarkAuthorizationCodeUsed(code string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	authCode, ok := s.authorizationCodes[code]
	if !ok {
		return fmt.Errorf("authorization code not found")
	}

	authCode.Used = true
	return nil
}

// DeleteAuthorizationCode removes an authorization code
func (s *Store) DeleteAuthorizationCode(code string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.authorizationCodes, code)
	return nil
}

// SaveToken saves an access token
func (s *Store) SaveToken(token *Token) error {
	if token.AccessToken == "" {
		return fmt.Errorf("access token cannot be empty")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.tokens[token.AccessToken] = token

	if token.RefreshToken != "" {
		s.refreshTokens[token.RefreshToken] = token
	}

	return nil
}

// GetToken retrieves a token by access token value
func (s *Store) GetToken(accessToken string) (*Token, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	token, ok := s.tokens[accessToken]
	if !ok {
		return nil, fmt.Errorf("token not found")
	}

	if token.IsExpired() {
		return nil, fmt.Errorf("token expired")
	}

	return token, nil
}

// GetTokenByRefreshToken retrieves a token by refresh token value
func (s *Store) GetTokenByRefreshToken(refreshToken string) (*Token, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	token, ok := s.refreshTokens[refreshToken]
	if !ok {
		return nil, fmt.Errorf("refresh token not found")
	}

	return token, nil
}

// DeleteToken removes a token
func (s *Store) DeleteToken(accessToken string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	token, ok := s.tokens[accessToken]
	if !ok {
		return fmt.Errorf("token not found")
	}

	delete(s.tokens, accessToken)

	if token.RefreshToken != "" {
		delete(s.refreshTokens, token.RefreshToken)
	}

	return nil
}

// DeleteTokenByRefreshToken removes a token by refresh token
func (s *Store) DeleteTokenByRefreshToken(refreshToken string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	token, ok := s.refreshTokens[refreshToken]
	if !ok {
		return fmt.Errorf("refresh token not found")
	}

	delete(s.refreshTokens, refreshToken)
	delete(s.tokens, token.AccessToken)

	return nil
}

// cleanupExpiredTokens periodically removes expired tokens and authorization codes
// Uses optimized locking strategy to minimize write lock duration
func (s *Store) cleanupExpiredTokens() {
	ticker := time.NewTicker(s.cleanupInterval)
	defer ticker.Stop()

	for range ticker.C {
		// Collect expired items with read lock first
		s.mu.RLock()

		expiredTokens := []string{}
		expiredRefreshTokens := []string{}
		for accessToken, token := range s.tokens {
			if token.IsExpired() {
				expiredTokens = append(expiredTokens, accessToken)
				if token.RefreshToken != "" {
					expiredRefreshTokens = append(expiredRefreshTokens, token.RefreshToken)
				}
			}
		}

		expiredCodes := []string{}
		for code, authCode := range s.authorizationCodes {
			if authCode.IsExpired() || authCode.Used {
				expiredCodes = append(expiredCodes, code)
			}
		}

		expiredGoogleTokens := []string{}
		now := time.Now()
		for email, token := range s.googleTokens {
			if token.Expiry.Before(now) {
				expiredGoogleTokens = append(expiredGoogleTokens, email)
			}
		}

		s.mu.RUnlock()

		// Delete in batch with write lock only if there's something to delete
		if len(expiredTokens) > 0 || len(expiredCodes) > 0 || len(expiredGoogleTokens) > 0 {
			s.mu.Lock()

			// Re-check expiration under write lock to avoid race conditions
			// Tokens might have been refreshed between read and write locks
			for _, accessToken := range expiredTokens {
				if token, ok := s.tokens[accessToken]; ok && token.IsExpired() {
					delete(s.tokens, accessToken)
					// Also delete refresh token if it exists
					if token.RefreshToken != "" {
						delete(s.refreshTokens, token.RefreshToken)
					}
				}
			}
			
			// Re-check authorization codes
			for _, code := range expiredCodes {
				if authCode, ok := s.authorizationCodes[code]; ok && (authCode.IsExpired() || authCode.Used) {
					delete(s.authorizationCodes, code)
				}
			}
			
			// Re-check Google tokens
			currentTime := time.Now()
			for _, email := range expiredGoogleTokens {
				if token, ok := s.googleTokens[email]; ok && token.Expiry.Before(currentTime) {
					delete(s.googleTokens, email)
					delete(s.googleUserInfo, email)
				}
			}

			s.mu.Unlock()
		}
	}
}

// Stats returns statistics about the store
func (s *Store) Stats() map[string]int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return map[string]int{
		"clients":             len(s.clients),
		"tokens":              len(s.tokens),
		"refresh_tokens":      len(s.refreshTokens),
		"authorization_codes": len(s.authorizationCodes),
		"google_tokens":       len(s.googleTokens),
	}
}

// SaveGoogleToken saves a Google OAuth token for a user
func (s *Store) SaveGoogleToken(email string, token *oauth2.Token) error {
	if email == "" {
		return fmt.Errorf("email cannot be empty")
	}
	if token == nil {
		return fmt.Errorf("token cannot be nil")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.googleTokens[email] = token
	return nil
}

// GetGoogleToken retrieves a Google OAuth token for a user
func (s *Store) GetGoogleToken(email string) (*oauth2.Token, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	token, ok := s.googleTokens[email]
	if !ok {
		return nil, fmt.Errorf("Google token not found for user: %s", email)
	}

	// Check if token is expired
	if token.Expiry.Before(time.Now()) {
		return nil, fmt.Errorf("Google token expired for user: %s", email)
	}

	return token, nil
}

// DeleteGoogleToken removes a Google OAuth token for a user
func (s *Store) DeleteGoogleToken(email string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.googleTokens, email)
	delete(s.googleUserInfo, email)
	return nil
}

// SaveGoogleUserInfo saves Google user info
func (s *Store) SaveGoogleUserInfo(email string, userInfo *GoogleUserInfo) error {
	if email == "" {
		return fmt.Errorf("email cannot be empty")
	}
	if userInfo == nil {
		return fmt.Errorf("userInfo cannot be nil")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.googleUserInfo[email] = userInfo
	return nil
}

// GetGoogleUserInfo retrieves Google user info
func (s *Store) GetGoogleUserInfo(email string) (*GoogleUserInfo, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	userInfo, ok := s.googleUserInfo[email]
	if !ok {
		return nil, fmt.Errorf("Google user info not found for user: %s", email)
	}

	return userInfo, nil
}
