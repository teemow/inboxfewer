package server

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"sync"

	"github.com/mark3labs/mcp-go/mcp"
)

// SessionIDManager implements session management for multi-account support
// It ensures that each Google account gets its own session, allowing
// multiple users or accounts to use the same MCP server instance
type SessionIDManager struct {
	sessions map[string]string // Maps session ID to account email
	mu       sync.RWMutex
}

// NewSessionIDManager creates a new session ID manager
func NewSessionIDManager() *SessionIDManager {
	return &SessionIDManager{
		sessions: make(map[string]string),
	}
}

// ResolveSessionID resolves the session ID from an HTTP request
// This implementation uses the Authorization header (Bearer token) to determine
// which session (account) the request belongs to
func (m *SessionIDManager) ResolveSessionID(r *http.Request) (string, error) {
	// Extract the Bearer token from the Authorization header
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return "", fmt.Errorf("no authorization header provided")
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
	m.mu.RLock()
	defer m.mu.RUnlock()

	if account, ok := m.sessions[sessionID]; ok {
		return account
	}
	return "default"
}

// SetAccountForSession associates an account email with a session ID
func (m *SessionIDManager) SetAccountForSession(sessionID, account string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sessions[sessionID] = account
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
