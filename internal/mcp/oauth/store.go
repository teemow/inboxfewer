package oauth

import (
	"fmt"
	"sync"
	"time"
)

// Store manages OAuth clients, tokens, and authorization codes in memory
type Store struct {
	mu                 sync.RWMutex
	clients            map[string]*ClientInfo
	tokens             map[string]*Token // access token -> Token
	refreshTokens      map[string]*Token // refresh token -> Token
	authorizationCodes map[string]*AuthorizationCode
}

// NewStore creates a new in-memory OAuth store
func NewStore() *Store {
	s := &Store{
		clients:            make(map[string]*ClientInfo),
		tokens:             make(map[string]*Token),
		refreshTokens:      make(map[string]*Token),
		authorizationCodes: make(map[string]*AuthorizationCode),
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
func (s *Store) cleanupExpiredTokens() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		s.mu.Lock()

		// Clean up expired tokens
		for accessToken, token := range s.tokens {
			if token.IsExpired() {
				delete(s.tokens, accessToken)
				if token.RefreshToken != "" {
					delete(s.refreshTokens, token.RefreshToken)
				}
			}
		}

		// Clean up expired authorization codes
		for code, authCode := range s.authorizationCodes {
			if authCode.IsExpired() || authCode.Used {
				delete(s.authorizationCodes, code)
			}
		}

		s.mu.Unlock()
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
	}
}
