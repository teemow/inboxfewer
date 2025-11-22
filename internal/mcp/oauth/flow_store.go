package oauth

import (
	"fmt"
	"log/slog"
	"sync"
	"time"
)

// FlowStore manages OAuth authorization flows
type FlowStore struct {
	states map[string]*AuthorizationState
	codes  map[string]*AuthorizationCode
	mu     sync.RWMutex
	logger *slog.Logger
}

// NewFlowStore creates a new flow store
func NewFlowStore(logger *slog.Logger) *FlowStore {
	if logger == nil {
		logger = slog.Default()
	}

	store := &FlowStore{
		states: make(map[string]*AuthorizationState),
		codes:  make(map[string]*AuthorizationCode),
		logger: logger,
	}

	// Start cleanup goroutine
	go store.cleanup()

	return store
}

// SaveAuthorizationState saves an authorization state
func (s *FlowStore) SaveAuthorizationState(state *AuthorizationState) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.states[state.GoogleState] = state
	s.logger.Debug("Saved authorization state",
		"google_state", state.GoogleState,
		"client_id", state.ClientID,
		"expires_at", time.Unix(state.ExpiresAt, 0),
	)

	return nil
}

// GetAuthorizationState retrieves an authorization state by Google state parameter
func (s *FlowStore) GetAuthorizationState(googleState string) (*AuthorizationState, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	state, exists := s.states[googleState]
	if !exists {
		return nil, fmt.Errorf("authorization state not found")
	}

	// Check if expired
	if time.Now().Unix() > state.ExpiresAt {
		return nil, fmt.Errorf("authorization state expired")
	}

	return state, nil
}

// DeleteAuthorizationState deletes an authorization state
func (s *FlowStore) DeleteAuthorizationState(googleState string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.states, googleState)
	s.logger.Debug("Deleted authorization state", "google_state", googleState)
}

// SaveAuthorizationCode saves an authorization code
func (s *FlowStore) SaveAuthorizationCode(code *AuthorizationCode) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.codes[code.Code] = code
	s.logger.Debug("Saved authorization code",
		"code_prefix", code.Code[:8]+"...",
		"client_id", code.ClientID,
		"user_email", code.UserEmail,
		"expires_at", time.Unix(code.ExpiresAt, 0),
	)

	return nil
}

// GetAuthorizationCode retrieves and marks an authorization code as used
func (s *FlowStore) GetAuthorizationCode(code string) (*AuthorizationCode, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	authCode, exists := s.codes[code]
	if !exists {
		return nil, fmt.Errorf("authorization code not found")
	}

	// Check if already used
	if authCode.Used {
		return nil, fmt.Errorf("authorization code already used")
	}

	// Check if expired
	if time.Now().Unix() > authCode.ExpiresAt {
		return nil, fmt.Errorf("authorization code expired")
	}

	// Mark as used
	authCode.Used = true

	s.logger.Debug("Authorization code used",
		"code_prefix", code[:8]+"...",
		"client_id", authCode.ClientID,
		"user_email", authCode.UserEmail,
	)

	return authCode, nil
}

// DeleteAuthorizationCode deletes an authorization code
func (s *FlowStore) DeleteAuthorizationCode(code string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.codes, code)
	s.logger.Debug("Deleted authorization code", "code_prefix", code[:8]+"...")
}

// cleanup periodically removes expired states and codes
func (s *FlowStore) cleanup() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		s.cleanupExpired()
	}
}

// cleanupExpired removes expired states and codes
func (s *FlowStore) cleanupExpired() {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().Unix()
	statesDeleted := 0
	codesDeleted := 0

	// Clean up expired states
	for googleState, state := range s.states {
		if now > state.ExpiresAt {
			delete(s.states, googleState)
			statesDeleted++
		}
	}

	// Clean up expired or used codes
	for code, authCode := range s.codes {
		if now > authCode.ExpiresAt || authCode.Used {
			delete(s.codes, code)
			codesDeleted++
		}
	}

	if statesDeleted > 0 || codesDeleted > 0 {
		s.logger.Debug("Cleaned up OAuth flow data",
			"states_deleted", statesDeleted,
			"codes_deleted", codesDeleted,
		)
	}
}

