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
	mu              sync.RWMutex
	googleTokens    map[string]*oauth2.Token   // user email or access token -> Google token
	googleUserInfo  map[string]*GoogleUserInfo // user email -> Google user info
	refreshTokens   map[string]string          // refresh token -> user email
	cleanupInterval time.Duration              // How often to cleanup expired tokens
	logger          *slog.Logger
}

// NewStore creates a new in-memory Google token store with default cleanup interval
func NewStore() *Store {
	return NewStoreWithInterval(1 * time.Minute)
}

// NewStoreWithInterval creates a new in-memory Google token store with custom cleanup interval
func NewStoreWithInterval(cleanupInterval time.Duration) *Store {
	s := &Store{
		googleTokens:    make(map[string]*oauth2.Token),
		googleUserInfo:  make(map[string]*GoogleUserInfo),
		refreshTokens:   make(map[string]string),
		cleanupInterval: cleanupInterval,
		logger:          slog.Default(),
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
	s.logger.Debug("Saved Google token", "email", email, "expiry", token.Expiry)
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
		now := time.Now()
		for email, token := range s.googleTokens {
			if !token.Expiry.IsZero() && token.Expiry.Before(now) {
				expiredGoogleTokens = append(expiredGoogleTokens, email)
			}
		}

		s.mu.RUnlock()

		// Delete in batch with write lock only if there's something to delete
		if len(expiredGoogleTokens) > 0 {
			s.mu.Lock()

			// Re-check expiration under write lock to prevent race conditions
			// Tokens might have been refreshed between read and write locks
			currentTime := time.Now()
			for _, email := range expiredGoogleTokens {
				if token, ok := s.googleTokens[email]; ok {
					if !token.Expiry.IsZero() && token.Expiry.Before(currentTime) {
						delete(s.googleTokens, email)
						delete(s.googleUserInfo, email)
						s.logger.Debug("Cleaned up expired token", "email", email)
					}
				}
			}

			s.mu.Unlock()
		}
	}
}

// SaveRefreshToken saves a refresh token mapping to user email
func (s *Store) SaveRefreshToken(refreshToken, email string) error {
	if refreshToken == "" {
		return fmt.Errorf("refresh token cannot be empty")
	}
	if email == "" {
		return fmt.Errorf("email cannot be empty")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.refreshTokens[refreshToken] = email
	s.logger.Debug("Saved refresh token", "email", email)
	return nil
}

// GetRefreshToken retrieves the user email associated with a refresh token
func (s *Store) GetRefreshToken(refreshToken string) (string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	email, ok := s.refreshTokens[refreshToken]
	if !ok {
		return "", fmt.Errorf("refresh token not found")
	}

	return email, nil
}

// DeleteRefreshToken removes a refresh token
func (s *Store) DeleteRefreshToken(refreshToken string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.refreshTokens, refreshToken)
	s.logger.Debug("Deleted refresh token")
	return nil
}

// Stats returns statistics about the store
func (s *Store) Stats() map[string]int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return map[string]int{
		"google_tokens":  len(s.googleTokens),
		"user_info":      len(s.googleUserInfo),
		"refresh_tokens": len(s.refreshTokens),
	}
}
