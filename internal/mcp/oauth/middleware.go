package oauth

import (
	"context"
	"net/http"
	"strings"
)

// contextKey is the type for context keys
type contextKey string

const (
	// tokenContextKey is the key for storing the token in the request context
	tokenContextKey contextKey = "oauth_token"

	// clientContextKey is the key for storing the client in the request context
	clientContextKey contextKey = "oauth_client"
)

// ValidateToken is middleware that validates the OAuth access token
func (h *Handler) ValidateToken(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Extract token from Authorization header
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			h.writeError(w, "invalid_token", "Missing Authorization header", http.StatusUnauthorized)
			return
		}

		// Check for Bearer token
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			h.writeError(w, "invalid_token", "Invalid Authorization header format", http.StatusUnauthorized)
			return
		}

		accessToken := parts[1]

		// Get token from store
		token, err := h.store.GetToken(accessToken)
		if err != nil {
			h.writeError(w, "invalid_token", "Invalid or expired token", http.StatusUnauthorized)
			return
		}

		// Validate resource (RFC 8707)
		if token.Resource != h.config.Resource {
			h.writeError(w, "invalid_token", "Token not valid for this resource", http.StatusForbidden)
			return
		}

		// Get client info
		client, err := h.store.GetClient(token.ClientID)
		if err != nil {
			h.writeError(w, "invalid_token", "Client not found", http.StatusUnauthorized)
			return
		}

		// Add token and client to request context
		ctx := context.WithValue(r.Context(), tokenContextKey, token)
		ctx = context.WithValue(ctx, clientContextKey, client)

		// Call next handler
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// ValidateTokenFunc is a function-based middleware that validates the OAuth access token
func (h *Handler) ValidateTokenFunc(next http.HandlerFunc) http.HandlerFunc {
	return h.ValidateToken(next).ServeHTTP
}

// GetTokenFromContext retrieves the token from the request context
func GetTokenFromContext(ctx context.Context) (*Token, bool) {
	token, ok := ctx.Value(tokenContextKey).(*Token)
	return token, ok
}

// GetClientFromContext retrieves the client from the request context
func GetClientFromContext(ctx context.Context) (*ClientInfo, bool) {
	client, ok := ctx.Value(clientContextKey).(*ClientInfo)
	return client, ok
}

// RequireScope is middleware that requires specific scopes
func (h *Handler) RequireScope(scopes ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token, ok := GetTokenFromContext(r.Context())
			if !ok {
				h.writeError(w, "invalid_token", "No token in context", http.StatusUnauthorized)
				return
			}

			// Check if token has required scopes
			tokenScopes := strings.Split(token.Scope, " ")
			scopeMap := make(map[string]bool)
			for _, scope := range tokenScopes {
				scopeMap[scope] = true
			}

			for _, requiredScope := range scopes {
				if !scopeMap[requiredScope] {
					h.writeError(w, "insufficient_scope", "Token missing required scope: "+requiredScope, http.StatusForbidden)
					return
				}
			}

			next.ServeHTTP(w, r)
		})
	}
}

// OptionalToken is middleware that optionally validates the OAuth access token
// If a token is present, it validates it; if not, it continues without authentication
func (h *Handler) OptionalToken(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Extract token from Authorization header
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			// No token provided, continue without authentication
			next.ServeHTTP(w, r)
			return
		}

		// Token provided, validate it
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			h.writeError(w, "invalid_token", "Invalid Authorization header format", http.StatusUnauthorized)
			return
		}

		accessToken := parts[1]

		// Get token from store
		token, err := h.store.GetToken(accessToken)
		if err != nil {
			h.writeError(w, "invalid_token", "Invalid or expired token", http.StatusUnauthorized)
			return
		}

		// Validate resource
		if token.Resource != h.config.Resource {
			h.writeError(w, "invalid_token", "Token not valid for this resource", http.StatusForbidden)
			return
		}

		// Get client info
		client, err := h.store.GetClient(token.ClientID)
		if err != nil {
			h.writeError(w, "invalid_token", "Client not found", http.StatusUnauthorized)
			return
		}

		// Add token and client to request context
		ctx := context.WithValue(r.Context(), tokenContextKey, token)
		ctx = context.WithValue(ctx, clientContextKey, client)

		// Call next handler
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
