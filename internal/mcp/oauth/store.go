package oauth

import (
	"fmt"
	"log/slog"
	"sync"
	"time"

	"golang.org/x/oauth2"
)

// Store manages Google OAuth tokens in memory
// This is a simplified store that only handles Google tokens for authenticated users
type Store struct {
	mu                   sync.RWMutex
	googleTokens         map[string]*oauth2.Token   // user email or access token -> Google token
	googleUserInfo       map[string]*GoogleUserInfo // user email -> Google user info
	refreshTokens        map[string]string          // refresh token -> user email
	refreshTokenExpiries map[string]int64           // refresh token -> expiry timestamp
	tokenToEmailMap      map[string]string          // inboxfewer access token -> user email (for cleanup)
	cleanupInterval      time.Duration              // How often to cleanup expired tokens
	logger               *slog.Logger
}

// NewStore creates a new in-memory Google token store with default cleanup interval
func NewStore() *Store {
	return NewStoreWithInterval(1 * time.Minute)
}

// NewStoreWithInterval creates a new in-memory Google token store with custom cleanup interval
func NewStoreWithInterval(cleanupInterval time.Duration) *Store {
	s := &Store{
		googleTokens:         make(map[string]*oauth2.Token),
		googleUserInfo:       make(map[string]*GoogleUserInfo),
		refreshTokens:        make(map[string]string),
		refreshTokenExpiries: make(map[string]int64),
		tokenToEmailMap:      make(map[string]string),
		cleanupInterval:      cleanupInterval,
		logger:               slog.Default(),
	}

	// Start background cleanup goroutine
	go s.cleanupExpiredTokens()

	return s
}

// SetLogger sets a custom logger for the store
func (s *Store) SetLogger(logger *slog.Logger) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.logger = logger
}

// SaveGoogleToken saves a Google OAuth token for a user
// The key can be either a user email (canonical storage) or an access token (for quick lookup)
func (s *Store) SaveGoogleToken(key string, token *oauth2.Token) error {
	if key == "" {
		return fmt.Errorf("key cannot be empty")
	}
	if token == nil {
		return fmt.Errorf("token cannot be nil")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.googleTokens[key] = token
	s.logger.Debug("Saved Google token", "key", key, "expiry", token.Expiry)
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

	// Also delete any refresh tokens for this user
	for refreshToken, userEmail := range s.refreshTokens {
		if userEmail == email {
			delete(s.refreshTokens, refreshToken)
		}
	}

	s.logger.Info("Deleted Google token and refresh tokens", "email", email)
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

// cleanupExpiredTokens periodically removes expired tokens
// Uses optimized locking strategy to minimize write lock duration
// Re-validates expiration under write lock to prevent race conditions
func (s *Store) cleanupExpiredTokens() {
	ticker := time.NewTicker(s.cleanupInterval)
	defer ticker.Stop()

	for range ticker.C {
		// Collect expired items with read lock first
		s.mu.RLock()

		expiredGoogleTokens := []string{}
		expiredRefreshTokens := []string{}
		now := time.Now()
		nowUnix := now.Unix()

		// Find expired Google tokens
		for key, token := range s.googleTokens {
			if !token.Expiry.IsZero() && token.Expiry.Before(now) {
				expiredGoogleTokens = append(expiredGoogleTokens, key)
			}
		}

		// Find expired refresh tokens
		for refreshToken, expiresAt := range s.refreshTokenExpiries {
			if nowUnix > expiresAt {
				expiredRefreshTokens = append(expiredRefreshTokens, refreshToken)
			}
		}

		s.mu.RUnlock()

		// Delete in batch with write lock only if there's something to delete
		if len(expiredGoogleTokens) > 0 || len(expiredRefreshTokens) > 0 {
			s.mu.Lock()

			// Re-check expiration under write lock to prevent race conditions
			// Tokens might have been refreshed between read and write locks
			currentTime := time.Now()
			currentTimeUnix := currentTime.Unix()

			for _, key := range expiredGoogleTokens {
				if token, ok := s.googleTokens[key]; ok {
					if !token.Expiry.IsZero() && token.Expiry.Before(currentTime) {
						delete(s.googleTokens, key)
						// Only delete user info if this is an email key (not an access token)
						if email, hasEmail := s.tokenToEmailMap[key]; hasEmail {
							delete(s.tokenToEmailMap, key)
							// Check if this was the last token for this email
							if _, stillHasToken := s.googleTokens[email]; !stillHasToken {
								delete(s.googleUserInfo, email)
							}
						} else {
							// This is an email key
							delete(s.googleUserInfo, key)
						}
						s.logger.Debug("Cleaned up expired Google token", "key", key)
					}
				}
			}

			for _, refreshToken := range expiredRefreshTokens {
				if expiresAt, ok := s.refreshTokenExpiries[refreshToken]; ok {
					if currentTimeUnix > expiresAt {
						email := s.refreshTokens[refreshToken]
						delete(s.refreshTokens, refreshToken)
						delete(s.refreshTokenExpiries, refreshToken)
						s.logger.Debug("Cleaned up expired refresh token", "email", email)
					}
				}
			}

			s.mu.Unlock()
		}
	}
}

// SaveRefreshToken saves a refresh token mapping to user email with expiry
func (s *Store) SaveRefreshToken(refreshToken, email string, expiresAt int64) error {
	if refreshToken == "" {
		return fmt.Errorf("refresh token cannot be empty")
	}
	if email == "" {
		return fmt.Errorf("email cannot be empty")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.refreshTokens[refreshToken] = email
	s.refreshTokenExpiries[refreshToken] = expiresAt
	s.logger.Debug("Saved refresh token",
		"email", email,
		"expires_at", time.Unix(expiresAt, 0))
	return nil
}

// GetRefreshToken retrieves the user email associated with a refresh token
// Returns an error if the token is expired
func (s *Store) GetRefreshToken(refreshToken string) (string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	email, ok := s.refreshTokens[refreshToken]
	if !ok {
		return "", fmt.Errorf("refresh token not found")
	}

	// Check if refresh token is expired
	if expiresAt, hasExpiry := s.refreshTokenExpiries[refreshToken]; hasExpiry {
		if time.Now().Unix() > expiresAt {
			return "", fmt.Errorf("refresh token expired")
		}
	}

	return email, nil
}

// DeleteRefreshToken removes a refresh token and its expiry tracking
func (s *Store) DeleteRefreshToken(refreshToken string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.refreshTokens, refreshToken)
	delete(s.refreshTokenExpiries, refreshToken)
	s.logger.Debug("Deleted refresh token")
	return nil
}

// SaveTokenWithEmailMapping saves a Google token by both email and access token
// This is a convenience method to ensure tokens are stored consistently
func (s *Store) SaveTokenWithEmailMapping(email, accessToken string, token *oauth2.Token) error {
	if email == "" {
		return fmt.Errorf("email cannot be empty")
	}
	if accessToken == "" {
		return fmt.Errorf("access token cannot be empty")
	}
	if token == nil {
		return fmt.Errorf("token cannot be nil")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// Save by email (canonical)
	s.googleTokens[email] = token
	// Save by access token (for quick lookup)
	s.googleTokens[accessToken] = token
	// Track the mapping for cleanup
	s.tokenToEmailMap[accessToken] = email

	s.logger.Debug("Saved Google token with email mapping",
		"email", email,
		"token_prefix", accessToken[:min(10, len(accessToken))])
	return nil
}

// Stats returns statistics about the store
func (s *Store) Stats() map[string]int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return map[string]int{
		"google_tokens":         len(s.googleTokens),
		"user_info":             len(s.googleUserInfo),
		"refresh_tokens":        len(s.refreshTokens),
		"refresh_token_expiries": len(s.refreshTokenExpiries),
		"token_email_mappings":  len(s.tokenToEmailMap),
	}
}
