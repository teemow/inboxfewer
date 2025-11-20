package oauth

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// Config holds the OAuth handler configuration
type Config struct {
	// Resource is the MCP server resource identifier for RFC 8707
	// This should be the base URL of the MCP server
	Resource string

	// SupportedScopes are all available scopes
	SupportedScopes []string
}

// Handler implements OAuth 2.1 endpoints for the MCP server
// It acts as an OAuth 2.1 Resource Server with Google as the Authorization Server
type Handler struct {
	config *Config
	store  *Store
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

	return &Handler{
		config: config,
		store:  NewStore(),
	}, nil
}

// GetStore returns the underlying store (for testing and token management)
func (h *Handler) GetStore() *Store {
	return h.store
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
