package oauth

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"golang.org/x/oauth2"
)

// contextKey is the type for context keys
type contextKey string

const (
	// userContextKey is the key for storing the user info in the request context
	userContextKey contextKey = "oauth_user"

	// tokenContextKey is the key for storing the Google token in the request context
	tokenContextKey contextKey = "google_token"
)

// ValidateGoogleToken is middleware that validates Google OAuth tokens
// It validates the token with Google's userinfo endpoint and stores user info in context
func (h *Handler) ValidateGoogleToken(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Extract token from Authorization header
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			// Return 401 with WWW-Authenticate header pointing to resource metadata
			w.Header().Set("WWW-Authenticate", fmt.Sprintf(
				`Bearer realm="%s", resource_metadata="/.well-known/oauth-protected-resource"`,
				h.config.Resource,
			))
			h.writeUnauthorizedError(w, "missing_token", "Missing Authorization header")
			return
		}

		// Check for Bearer token
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			w.Header().Set("WWW-Authenticate", fmt.Sprintf(
				`Bearer realm="%s", resource_metadata="/.well-known/oauth-protected-resource", error="invalid_token", error_description="Invalid Authorization header format"`,
				h.config.Resource,
			))
			h.writeUnauthorizedError(w, "invalid_token", "Invalid Authorization header format")
			return
		}

	accessToken := parts[1]

	// Create OAuth2 token
	token := &oauth2.Token{
		AccessToken: accessToken,
		TokenType:   "Bearer",
	}

	// Validate token by calling Google's userinfo endpoint
	userInfo, err := h.getUserInfoFromGoogle(r.Context(), token)
	if err != nil {
		// Provide more actionable error messages based on error type
		errorDesc := getActionableErrorMessage(err)
		
		w.Header().Set("WWW-Authenticate", fmt.Sprintf(
			`Bearer realm="%s", resource_metadata="/.well-known/oauth-protected-resource", error="invalid_token", error_description="%s"`,
			h.config.Resource,
			errorDesc,
		))
		h.writeUnauthorizedError(w, "invalid_token", errorDesc)
		return
	}
	// Store user info and token in context
	ctx := context.WithValue(r.Context(), userContextKey, userInfo)
	ctx = context.WithValue(ctx, tokenContextKey, token)

	// Save the token for this user so we can use it to access Google APIs
	// Use email as the account identifier
	if err := h.store.SaveGoogleToken(userInfo.Email, token); err != nil {
		// Log but don't fail - we can still process the request
		fmt.Printf("Warning: Failed to save Google token for user %s: %v\n", userInfo.Email, err)
	}

	// Call next handler
	next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// ValidateGoogleTokenFunc is a function-based middleware that validates Google OAuth tokens
func (h *Handler) ValidateGoogleTokenFunc(next http.HandlerFunc) http.HandlerFunc {
	return h.ValidateGoogleToken(next).ServeHTTP
}

// OptionalGoogleToken is middleware that optionally validates Google OAuth tokens
// If a token is present, it validates it; if not, it continues without authentication
func (h *Handler) OptionalGoogleToken(next http.Handler) http.Handler {
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
			h.writeUnauthorizedError(w, "invalid_token", "Invalid Authorization header format")
			return
		}

		accessToken := parts[1]

		// Create OAuth2 token
		token := &oauth2.Token{
			AccessToken: accessToken,
			TokenType:   "Bearer",
		}

		// Validate token by calling Google's userinfo endpoint
		userInfo, err := h.getUserInfoFromGoogle(r.Context(), token)
		if err != nil {
			h.writeUnauthorizedError(w, "invalid_token", fmt.Sprintf("Token validation failed: %v", err))
			return
		}

		// Store user info and token in context
		ctx := context.WithValue(r.Context(), userContextKey, userInfo)
		ctx = context.WithValue(ctx, tokenContextKey, token)

		// Save the token for this user
		if err := h.store.SaveGoogleToken(userInfo.Email, token); err != nil {
			fmt.Printf("Warning: Failed to save Google token for user %s: %v\n", userInfo.Email, err)
		}

		// Call next handler
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// getUserInfoFromGoogle validates a token by calling Google's userinfo endpoint
func (h *Handler) getUserInfoFromGoogle(ctx context.Context, token *oauth2.Token) (*GoogleUserInfo, error) {
	// Create HTTP client with the token
	client := oauth2.NewClient(ctx, oauth2.StaticTokenSource(token))

	// Call Google's userinfo endpoint
	resp, err := client.Get("https://www.googleapis.com/oauth2/v2/userinfo")
	if err != nil {
		return nil, fmt.Errorf("failed to get user info: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("userinfo request failed with status %d", resp.StatusCode)
	}

	// Parse user info
	var userInfo GoogleUserInfo
	if err := json.NewDecoder(resp.Body).Decode(&userInfo); err != nil {
		return nil, fmt.Errorf("failed to decode user info: %w", err)
	}

	return &userInfo, nil
}

// GetUserFromContext retrieves the Google user info from the request context
func GetUserFromContext(ctx context.Context) (*GoogleUserInfo, bool) {
	userInfo, ok := ctx.Value(userContextKey).(*GoogleUserInfo)
	return userInfo, ok
}

// GetGoogleTokenFromContext retrieves the Google token from the request context
func GetGoogleTokenFromContext(ctx context.Context) (*oauth2.Token, bool) {
	token, ok := ctx.Value(tokenContextKey).(*oauth2.Token)
	return token, ok
}

// writeUnauthorizedError writes an OAuth error response with 401 status
func (h *Handler) writeUnauthorizedError(w http.ResponseWriter, errorCode, description string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)
	json.NewEncoder(w).Encode(ErrorResponse{
		Error:            errorCode,
		ErrorDescription: description,
	})
}

// getActionableErrorMessage converts technical errors into user-friendly, actionable messages
func getActionableErrorMessage(err error) string {
	errStr := err.Error()
	
	// Check for common error patterns and provide actionable guidance
	if strings.Contains(errStr, "401") || strings.Contains(errStr, "Unauthorized") {
		return "Google token is invalid or expired. Please re-authenticate through your MCP client to continue."
	}
	
	if strings.Contains(errStr, "403") || strings.Contains(errStr, "Forbidden") {
		return "Access denied by Google. Please ensure your token has the required scopes and re-authenticate through your MCP client."
	}
	
	if strings.Contains(errStr, "network") || strings.Contains(errStr, "connection") || 
	   strings.Contains(errStr, "timeout") || strings.Contains(errStr, "dial") {
		return "Unable to verify token with Google due to network issues. Please try again in a moment."
	}
	
	if strings.Contains(errStr, "429") || strings.Contains(errStr, "rate limit") {
		return "Google API rate limit exceeded. Please wait a moment and try again."
	}
	
	if strings.Contains(errStr, "500") || strings.Contains(errStr, "502") || 
	   strings.Contains(errStr, "503") || strings.Contains(errStr, "504") {
		return "Google authentication service is temporarily unavailable. Please try again in a few minutes."
	}
	
	// Default message with error details
	return fmt.Sprintf("Token validation failed: %v. Please re-authenticate through your MCP client.", err)
}

// CacheGoogleToken caches a Google token for future use
// This can be called by endpoints that receive tokens through other means
func (h *Handler) CacheGoogleToken(email string, token *oauth2.Token) error {
	return h.store.SaveGoogleToken(email, token)
}

// GetCachedGoogleToken retrieves a cached Google token for a user
func (h *Handler) GetCachedGoogleToken(email string) (*oauth2.Token, error) {
	return h.store.GetGoogleToken(email)
}
