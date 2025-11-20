package oauth

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

// Config holds the OAuth handler configuration
type Config struct {
	// Resource is the MCP server resource identifier for RFC 8707
	// This should be the base URL of the MCP server
	Resource string

	// SupportedScopes are all available scopes
	SupportedScopes []string

	// RateLimitRate is the number of requests per second allowed per IP (0 = no limit)
	RateLimitRate int

	// RateLimitBurst is the maximum burst size allowed per IP
	RateLimitBurst int

	// CleanupInterval is how often to cleanup expired tokens (default: 1 minute)
	CleanupInterval time.Duration

	// TrustProxy indicates whether to trust X-Forwarded-For and X-Real-IP headers
	// Only set to true if the server is behind a trusted proxy
	TrustProxy bool
}

// Handler implements OAuth 2.1 endpoints for the MCP server
// It acts as an OAuth 2.1 Resource Server with Google as the Authorization Server
type Handler struct {
	config      *Config      // Lowercase for internal use, exposed via getter if needed
	store       *Store
	rateLimiter *RateLimiter // Optional rate limiter for protecting endpoints
	oauthConfig *oauth2.Config // Google OAuth config for token refresh
}

// NewHandler creates a new OAuth handler
func NewHandler(config *Config) (*Handler, error) {
	if config.Resource == "" {
		return nil, fmt.Errorf("resource is required")
	}

	// Set default scopes if none provided
	if len(config.SupportedScopes) == 0 {
		config.SupportedScopes = []string{
			"https://www.googleapis.com/auth/gmail.readonly",
			"https://www.googleapis.com/auth/gmail.modify",
			"https://www.googleapis.com/auth/gmail.send",
			"https://www.googleapis.com/auth/gmail.settings.basic",
			"https://www.googleapis.com/auth/documents.readonly",
			"https://www.googleapis.com/auth/drive",
			"https://www.googleapis.com/auth/calendar",
			"https://www.googleapis.com/auth/meetings.space.readonly",
			"https://www.googleapis.com/auth/tasks",
		}
	}

	// Set default cleanup interval if not specified
	if config.CleanupInterval == 0 {
		config.CleanupInterval = 1 * time.Minute
	}

	// Create rate limiter if configured
	var rateLimiter *RateLimiter
	if config.RateLimitRate > 0 {
		burst := config.RateLimitBurst
		if burst == 0 {
			burst = config.RateLimitRate * 2 // Default burst is 2x rate
		}
		rateLimiter = NewRateLimiter(config.RateLimitRate, burst, config.TrustProxy)
	}

	// Create Google OAuth config for token refresh
	// Note: ClientID and ClientSecret would need to be provided for actual refresh
	// For now, we create a minimal config that can be used with existing tokens
	oauthConfig := &oauth2.Config{
		Endpoint: google.Endpoint,
		Scopes:   config.SupportedScopes,
	}

	return &Handler{
		config:      config,
		store:       NewStoreWithInterval(config.CleanupInterval),
		rateLimiter: rateLimiter,
		oauthConfig: oauthConfig,
	}, nil
}

// GetStore returns the underlying store (for testing and token management)
func (h *Handler) GetStore() *Store {
	return h.store
}

// GetConfig returns the OAuth configuration
func (h *Handler) GetConfig() *Config {
	return h.config
}

// ServeProtectedResourceMetadata serves the OAuth 2.0 Protected Resource Metadata (RFC 9728)
// This endpoint tells MCP clients where to find the authorization server (Google)
//
// The MCP client will:
// 1. Make an unauthenticated request to the MCP server
// 2. Receive a 401 with WWW-Authenticate header pointing to this endpoint
// 3. Fetch this metadata to discover the authorization server
// 4. Use Google's OAuth 2.0 flow to obtain an access token
// 5. Include the token in subsequent requests to the MCP server
func (h *Handler) ServeProtectedResourceMetadata(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Point to Google as the authorization server
	// MCP clients will use Google's well-known endpoint at:
	// https://accounts.google.com/.well-known/openid-configuration
	metadata := ProtectedResourceMetadata{
		Resource: h.config.Resource,
		AuthorizationServers: []string{
			"https://accounts.google.com",
		},
		BearerMethodsSupported: []string{
			"header", // Authorization: Bearer <token>
		},
		ScopesSupported: h.config.SupportedScopes,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(metadata); err != nil {
		http.Error(w, "Failed to encode metadata", http.StatusInternalServerError)
	}
}

// writeError is a helper to write OAuth error responses
func (h *Handler) writeError(w http.ResponseWriter, errorCode, description string, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(ErrorResponse{
		Error:            errorCode,
		ErrorDescription: description,
	})
}

// RevokeToken revokes a Google OAuth token for a specific user
// This removes the token from the store, forcing re-authentication
func (h *Handler) RevokeToken(email string) error {
	return h.store.DeleteGoogleToken(email)
}

// ServeRevoke handles token revocation requests
// POST /oauth/revoke with {"email": "user@example.com"}
func (h *Handler) ServeRevoke(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse request body
	var req struct {
		Email string `json:"email"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, "invalid_request", "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Email == "" {
		h.writeError(w, "invalid_request", "Email is required", http.StatusBadRequest)
		return
	}

	// Revoke the token
	if err := h.RevokeToken(req.Email); err != nil {
		h.writeError(w, "server_error", fmt.Sprintf("Failed to revoke token: %v", err), http.StatusInternalServerError)
		return
	}

	// Return success
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "success",
		"message": fmt.Sprintf("Token revoked for %s", req.Email),
	})
}
