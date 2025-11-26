package server

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
)

// sessionInfo tracks session metadata for cleanup
type sessionInfo struct {
	account    string
	lastAccess time.Time
}

// SessionIDManager implements session management for multi-account support
// It ensures that each Google account gets its own session, allowing
// multiple users or accounts to use the same MCP server instance
type SessionIDManager struct {
	sessions       map[string]*sessionInfo // Maps session ID to session info
	mu             sync.RWMutex
	cleanupTicker  *time.Ticker
	cleanupDone    chan bool
	sessionTimeout time.Duration
	logger         *slog.Logger
}

// NewSessionIDManager creates a new session ID manager with default logger
func NewSessionIDManager() *SessionIDManager {
	return NewSessionIDManagerWithLogger(24*time.Hour, slog.Default())
}

// NewSessionIDManagerWithTimeout creates a new session ID manager with custom timeout
func NewSessionIDManagerWithTimeout(timeout time.Duration) *SessionIDManager {
	return NewSessionIDManagerWithLogger(timeout, slog.Default())
}

// NewSessionIDManagerWithLogger creates a new session ID manager with custom timeout and logger
func NewSessionIDManagerWithLogger(timeout time.Duration, logger *slog.Logger) *SessionIDManager {
	if logger == nil {
		logger = slog.Default()
	}

	m := &SessionIDManager{
		sessions:       make(map[string]*sessionInfo),
		cleanupTicker:  time.NewTicker(10 * time.Minute),
		cleanupDone:    make(chan bool),
		sessionTimeout: timeout,
		logger:         logger,
	}

	// Start cleanup goroutine
	go m.cleanupExpiredSessions()

	return m
}

// ErrNoAuthorizationHeader is returned when no Authorization header is provided
var ErrNoAuthorizationHeader = errors.New("no authorization header provided")

// ResolveSessionID resolves the session ID from an HTTP request
// This implementation uses the Authorization header (Bearer token) to determine
// which session (account) the request belongs to
func (m *SessionIDManager) ResolveSessionID(r *http.Request) (string, error) {
	// Extract the Bearer token from the Authorization header
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return "", ErrNoAuthorizationHeader
	}

	// The token uniquely identifies the user/account
	// Generate a stable session ID from the token
	sessionID := m.generateSessionID(authHeader)

	return sessionID, nil
}

// ResolveSessionIDFromRequest resolves the session ID from an MCP request
// This is called for MCP protocol-level requests
func (m *SessionIDManager) ResolveSessionIDFromRequest(request *mcp.JSONRPCRequest) (string, error) {
	// For now, we return a default session ID for stdio transport
	// In HTTP transport, ResolveSessionID is used instead
	return "default", nil
}

// GetAccountForSession returns the account email associated with a session ID
func (m *SessionIDManager) GetAccountForSession(sessionID string) string {
	m.mu.Lock()
	defer m.mu.Unlock()

	if info, ok := m.sessions[sessionID]; ok {
		// Update last access time
		info.lastAccess = time.Now()
		return info.account
	}
	return "default"
}

// SetAccountForSession associates an account email with a session ID
func (m *SessionIDManager) SetAccountForSession(sessionID, account string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sessions[sessionID] = &sessionInfo{
		account:    account,
		lastAccess: time.Now(),
	}
}

// generateSessionID creates a stable session ID from the auth token
func (m *SessionIDManager) generateSessionID(token string) string {
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
}

// RemoveSession removes a session from the manager
func (m *SessionIDManager) RemoveSession(sessionID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.sessions, sessionID)
}

// ListSessions returns all active session IDs
func (m *SessionIDManager) ListSessions() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	sessions := make([]string, 0, len(m.sessions))
	for sessionID := range m.sessions {
		sessions = append(sessions, sessionID)
	}
	return sessions
}

// cleanupExpiredSessions periodically removes expired sessions
func (m *SessionIDManager) cleanupExpiredSessions() {
	for {
		select {
		case <-m.cleanupTicker.C:
			m.mu.Lock()
			now := time.Now()
			expiredCount := 0
			for sessionID, info := range m.sessions {
				if now.Sub(info.lastAccess) > m.sessionTimeout {
					delete(m.sessions, sessionID)
					expiredCount++
				}
			}
			m.mu.Unlock()
			if expiredCount > 0 {
				m.logger.Info("Cleaned up expired sessions", "count", expiredCount)
			}
		case <-m.cleanupDone:
			return
		}
	}
}

// Stop stops the session cleanup goroutine
func (m *SessionIDManager) Stop() {
	if m.cleanupTicker != nil {
		m.cleanupTicker.Stop()
	}
	if m.cleanupDone != nil {
		close(m.cleanupDone)
	}
}
