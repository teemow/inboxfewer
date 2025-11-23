package oauth

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"golang.org/x/oauth2"
)

// ServeAuthorizationServerMetadata serves the OAuth 2.0 Authorization Server Metadata (RFC 8414)
// This endpoint tells MCP clients about the inboxfewer OAuth server endpoints
func (h *Handler) ServeAuthorizationServerMetadata(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	metadata := AuthorizationServerMetadata{
		Issuer:                h.config.Resource,
		AuthorizationEndpoint: h.config.Resource + "/oauth/authorize",
		TokenEndpoint:         h.config.Resource + "/oauth/token",
		RegistrationEndpoint:  h.config.Resource + "/oauth/register",
		ScopesSupported:       h.config.SupportedScopes,
		ResponseTypesSupported: []string{
			"code", // Authorization code flow
		},
		GrantTypesSupported: []string{
			"authorization_code",
			"refresh_token",
		},
		TokenEndpointAuthMethodsSupported: []string{
			"client_secret_basic",
			"client_secret_post",
			"none", // For public clients
		},
		CodeChallengeMethodsSupported: []string{
			"S256", // SHA-256 PKCE (required by OAuth 2.1)
			"plain",
		},
	}

	h.setSecurityHeaders(w)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(metadata); err != nil {
		h.logger.Error("Failed to encode authorization server metadata", "error", err)
	}
}

// ServeDynamicClientRegistration handles Dynamic Client Registration (RFC 7591)
func (h *Handler) ServeDynamicClientRegistration(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse registration request
	var req ClientRegistrationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, "invalid_request", "Failed to parse registration request", http.StatusBadRequest)
		return
	}

	// Validate redirect URIs (at least one required for authorization_code flow)
	if len(req.RedirectURIs) == 0 {
		h.writeError(w, "invalid_redirect_uri", "At least one redirect_uri is required", http.StatusBadRequest)
		return
	}

	// Validate redirect URIs with comprehensive security checks
	for _, uri := range req.RedirectURIs {
		if err := validateRedirectURI(uri, h.config.Resource); err != nil {
			h.writeError(w, "invalid_redirect_uri", err.Error(), http.StatusBadRequest)
			return
		}
	}

	// Register the client
	resp, err := h.clientStore.RegisterClient(&req)
	if err != nil {
		h.logger.Error("Failed to register client", "error", err)
		h.writeError(w, "server_error", "Failed to register client", http.StatusInternalServerError)
		return
	}

	h.logger.Info("Client registered successfully",
		"client_id", resp.ClientID,
		"client_name", resp.ClientName,
	)

	h.setSecurityHeaders(w)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(resp)
}

// ServeAuthorization handles the OAuth authorization endpoint
// This proxies the authorization request to Google
func (h *Handler) ServeAuthorization(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Check if Google OAuth is configured
	if h.googleConfig == nil {
		h.logger.Error("Google OAuth not configured")
		h.writeError(w, "server_error", "OAuth proxy not configured", http.StatusInternalServerError)
		return
	}

	// Parse query parameters
	query := r.URL.Query()
	clientID := query.Get("client_id")
	redirectURI := query.Get("redirect_uri")
	state := query.Get("state")
	scope := query.Get("scope")
	codeChallenge := query.Get("code_challenge")
	codeChallengeMethod := query.Get("code_challenge_method")
	nonce := query.Get("nonce")

	// Validate required parameters
	if clientID == "" {
		h.writeError(w, "invalid_request", "client_id is required", http.StatusBadRequest)
		return
	}

	if redirectURI == "" {
		h.writeError(w, "invalid_request", "redirect_uri is required", http.StatusBadRequest)
		return
	}

	// OAuth 2.1: state is RECOMMENDED (not required) for CSRF protection
	// We allow requests without state for compatibility with some clients (e.g., Cursor)
	// However, this weakens CSRF protection
	if state == "" {
		h.logger.Warn("Authorization request without state parameter (CSRF protection disabled)",
			"client_id", clientID,
			"redirect_uri", redirectURI)
	}

	// Validate requested scopes
	if scope != "" {
		if err := h.validateScopes(scope); err != nil {
			h.writeError(w, "invalid_scope", err.Error(), http.StatusBadRequest)
			return
		}
	}

	// Validate client exists
	client, err := h.clientStore.GetClient(clientID)
	if err != nil {
		h.logger.Warn("Invalid client_id", "client_id", clientID, "error", err)
		h.writeError(w, "invalid_client", "Invalid client_id", http.StatusUnauthorized)
		return
	}

	// Validate redirect_uri
	if err := h.clientStore.ValidateRedirectURI(clientID, redirectURI); err != nil {
		h.logger.Warn("Invalid redirect_uri",
			"client_id", clientID,
			"redirect_uri", redirectURI,
			"error", err,
		)
		h.writeError(w, "invalid_request", "redirect_uri not registered for this client", http.StatusBadRequest)
		return
	}

	// OAuth 2.1 requires PKCE for public clients
	if codeChallenge == "" && client.TokenEndpointAuthMethod == "none" {
		h.writeError(w, "invalid_request", "PKCE is required for public clients", http.StatusBadRequest)
		return
	}

	// Validate code challenge method if PKCE is used
	if codeChallenge != "" {
		if codeChallengeMethod == "" {
			codeChallengeMethod = "plain" // Default to plain if not specified
		}
		if codeChallengeMethod != "S256" && codeChallengeMethod != "plain" {
			h.writeError(w, "invalid_request", "Invalid code_challenge_method", http.StatusBadRequest)
			return
		}
	}

	// Generate Google state parameter
	googleState, err := generateSecureToken(32)
	if err != nil {
		h.logger.Error("Failed to generate state", "error", err)
		h.writeError(w, "server_error", "Failed to generate state", http.StatusInternalServerError)
		return
	}

	// Save authorization state
	now := time.Now().Unix()
	authState := &AuthorizationState{
		State:               state,
		ClientID:            clientID,
		RedirectURI:         redirectURI,
		Scope:               scope,
		CodeChallenge:       codeChallenge,
		CodeChallengeMethod: codeChallengeMethod,
		GoogleState:         googleState,
		CreatedAt:           now,
		ExpiresAt:           now + 600, // 10 minutes
		Nonce:               nonce,
	}

	if err := h.flowStore.SaveAuthorizationState(authState); err != nil {
		h.logger.Error("Failed to save authorization state", "error", err)
		h.writeError(w, "server_error", "Failed to save state", http.StatusInternalServerError)
		return
	}

	// Build Google authorization URL
	googleAuthURL := h.googleConfig.AuthCodeURL(googleState,
		oauth2.AccessTypeOffline, // Request refresh token
		oauth2.ApprovalForce,     // Always show consent screen
	)

	h.logger.Info("Redirecting to Google for authorization",
		"client_id", clientID,
		"redirect_uri", redirectURI,
		"google_state", googleState,
	)

	// Redirect to Google
	http.Redirect(w, r, googleAuthURL, http.StatusFound)
}

// ServeGoogleCallback handles the callback from Google OAuth
func (h *Handler) ServeGoogleCallback(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	query := r.URL.Query()
	googleState := query.Get("state")
	code := query.Get("code")
	errorParam := query.Get("error")

	// Check for error from Google
	if errorParam != "" {
		errorDesc := query.Get("error_description")
		h.logger.Warn("Google OAuth error",
			"error", errorParam,
			"description", errorDesc,
		)
		http.Error(w, fmt.Sprintf("Google OAuth error: %s - %s", errorParam, errorDesc), http.StatusBadRequest)
		return
	}

	// Validate state and retrieve authorization state
	authState, err := h.flowStore.GetAuthorizationState(googleState)
	if err != nil {
		h.logger.Error("Invalid or expired state", "google_state", googleState, "error", err)
		http.Error(w, "Invalid or expired state", http.StatusBadRequest)
		return
	}

	// Exchange code for Google token
	ctx := context.Background()
	googleToken, err := h.googleConfig.Exchange(ctx, code)
	if err != nil {
		h.logger.Error("Failed to exchange code for Google token", "error", err)
		http.Error(w, "Failed to exchange authorization code", http.StatusInternalServerError)
		return
	}

	// Get user info from Google
	userInfo, err := h.fetchGoogleUserInfo(ctx, googleToken.AccessToken)
	if err != nil {
		h.logger.Error("Failed to fetch Google user info", "error", err)
		http.Error(w, "Failed to fetch user information", http.StatusInternalServerError)
		return
	}

	h.logger.Info("Google OAuth successful",
		"user_email", userInfo.Email,
		"client_id", authState.ClientID,
	)

	// Generate authorization code for the MCP client
	authCode, err := generateSecureToken(32)
	if err != nil {
		h.logger.Error("Failed to generate authorization code", "error", err)
		http.Error(w, "Failed to generate authorization code", http.StatusInternalServerError)
		return
	}

	// Save authorization code
	now := time.Now().Unix()
	authCodeData := &AuthorizationCode{
		Code:                authCode,
		ClientID:            authState.ClientID,
		RedirectURI:         authState.RedirectURI,
		Scope:               authState.Scope,
		CodeChallenge:       authState.CodeChallenge,
		CodeChallengeMethod: authState.CodeChallengeMethod,
		GoogleAccessToken:   googleToken.AccessToken,
		GoogleRefreshToken:  googleToken.RefreshToken,
		GoogleTokenExpiry:   googleToken.Expiry.Unix(),
		UserEmail:           userInfo.Email,
		CreatedAt:           now,
		ExpiresAt:           now + 600, // 10 minutes
		Used:                false,
	}

	if err := h.flowStore.SaveAuthorizationCode(authCodeData); err != nil {
		h.logger.Error("Failed to save authorization code", "error", err)
		http.Error(w, "Failed to save authorization code", http.StatusInternalServerError)
		return
	}

	// Clean up authorization state
	h.flowStore.DeleteAuthorizationState(googleState)

	// Build redirect URL with authorization code
	redirectURL, err := url.Parse(authState.RedirectURI)
	if err != nil {
		h.logger.Error("Invalid redirect URI", "redirect_uri", authState.RedirectURI, "error", err)
		http.Error(w, "Invalid redirect URI", http.StatusInternalServerError)
		return
	}

	redirectQuery := redirectURL.Query()
	redirectQuery.Set("code", authCode)
	// Only include state if the client provided one (OAuth 2.1 CSRF protection)
	if authState.State != "" {
		redirectQuery.Set("state", authState.State)
	}
	redirectURL.RawQuery = redirectQuery.Encode()

	h.logger.Info("Redirecting back to MCP client",
		"client_id", authState.ClientID,
		"redirect_uri", authState.RedirectURI,
	)

	// Redirect back to MCP client
	http.Redirect(w, r, redirectURL.String(), http.StatusFound)
}

// ServeToken handles the OAuth token endpoint
func (h *Handler) ServeToken(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse form data
	if err := r.ParseForm(); err != nil {
		h.writeError(w, "invalid_request", "Failed to parse request", http.StatusBadRequest)
		return
	}

	grantType := r.FormValue("grant_type")

	switch grantType {
	case "authorization_code":
		h.handleAuthorizationCodeGrant(w, r)
	case "refresh_token":
		h.handleRefreshTokenGrant(w, r)
	default:
		h.writeError(w, "unsupported_grant_type", fmt.Sprintf("Grant type %s not supported", grantType), http.StatusBadRequest)
	}
}

// handleAuthorizationCodeGrant handles the authorization_code grant type
func (h *Handler) handleAuthorizationCodeGrant(w http.ResponseWriter, r *http.Request) {
	code := r.FormValue("code")
	redirectURI := r.FormValue("redirect_uri")
	clientID := r.FormValue("client_id")
	codeVerifier := r.FormValue("code_verifier")

	// Validate required parameters
	if code == "" {
		h.writeError(w, "invalid_request", "code is required", http.StatusBadRequest)
		return
	}

	// Retrieve and validate authorization code
	authCode, err := h.flowStore.GetAuthorizationCode(code)
	if err != nil {
		h.logger.Warn("Invalid authorization code", "error", err)
		h.writeError(w, "invalid_grant", "Invalid or expired authorization code", http.StatusBadRequest)
		return
	}

	// OAuth 2.1: For public clients using PKCE, client_id is optional
	// If not provided, use the client_id from the authorization code
	if clientID == "" {
		clientID = authCode.ClientID
		h.logger.Debug("Using client_id from authorization code",
			"client_id", clientID)
	} else {
		// If client_id is provided, validate it matches
		if authCode.ClientID != clientID {
			h.logger.Warn("Client ID mismatch",
				"expected", authCode.ClientID,
				"got", clientID,
			)
			h.writeError(w, "invalid_grant", "Client ID mismatch", http.StatusBadRequest)
			return
		}
	}

	// Validate redirect_uri matches
	if authCode.RedirectURI != redirectURI {
		h.logger.Warn("Redirect URI mismatch",
			"expected", authCode.RedirectURI,
			"got", redirectURI,
		)
		h.writeError(w, "invalid_grant", "Redirect URI mismatch", http.StatusBadRequest)
		return
	}

	// Validate PKCE if code_challenge was used
	if authCode.CodeChallenge != "" {
		if codeVerifier == "" {
			h.writeError(w, "invalid_request", "code_verifier is required", http.StatusBadRequest)
			return
		}

		// Validate code_verifier entropy (RFC 7636: min 43 chars, max 128 chars)
		if len(codeVerifier) < 43 {
			h.logger.Warn("code_verifier too short (insufficient entropy)",
				"client_id", clientID,
				"length", len(codeVerifier))
			h.writeError(w, "invalid_request",
				"code_verifier must be at least 43 characters (RFC 7636)",
				http.StatusBadRequest)
			return
		}
		if len(codeVerifier) > 128 {
			h.logger.Warn("code_verifier too long",
				"client_id", clientID,
				"length", len(codeVerifier))
			h.writeError(w, "invalid_request",
				"code_verifier must be at most 128 characters (RFC 7636)",
				http.StatusBadRequest)
			return
		}

		// Verify code_verifier against code_challenge
		var computedChallenge string
		if authCode.CodeChallengeMethod == "S256" {
			hash := sha256.Sum256([]byte(codeVerifier))
			computedChallenge = base64.URLEncoding.WithPadding(base64.NoPadding).EncodeToString(hash[:])
		} else {
			computedChallenge = codeVerifier
		}

		if computedChallenge != authCode.CodeChallenge {
			h.logger.Warn("PKCE verification failed",
				"client_id", clientID,
			)
			h.writeError(w, "invalid_grant", "Invalid code_verifier", http.StatusBadRequest)
			return
		}
	}

	// Authenticate client (if not public client)
	client, err := h.clientStore.GetClient(clientID)
	if err != nil {
		h.logger.Error("Failed to get client", "client_id", clientID, "error", err)
		h.writeError(w, "invalid_client", "Invalid client", http.StatusUnauthorized)
		return
	}

	if client.TokenEndpointAuthMethod != "none" {
		// Confidential client - validate client secret
		clientSecret := r.FormValue("client_secret")
		if clientSecret == "" {
			// Try Basic Auth
			username, password, ok := r.BasicAuth()
			if !ok || username != clientID {
				h.writeError(w, "invalid_client", "Client authentication required", http.StatusUnauthorized)
				return
			}
			clientSecret = password
		}

		if err := h.clientStore.ValidateClientSecret(clientID, clientSecret); err != nil {
			h.logger.Warn("Client authentication failed", "client_id", clientID)
			h.writeError(w, "invalid_client", "Client authentication failed", http.StatusUnauthorized)
			return
		}
	}

	// Generate inboxfewer access token
	accessToken, err := generateSecureToken(48)
	if err != nil {
		h.logger.Error("Failed to generate access token", "error", err)
		h.writeError(w, "server_error", "Failed to generate access token", http.StatusInternalServerError)
		return
	}

	// Build Google token from authorization code
	googleToken := &oauth2.Token{
		AccessToken:  authCode.GoogleAccessToken,
		RefreshToken: authCode.GoogleRefreshToken,
		Expiry:       time.Unix(authCode.GoogleTokenExpiry, 0),
	}

	// Calculate token expiry
	expiresIn := authCode.GoogleTokenExpiry - time.Now().Unix()
	
	// If token is expired or expiring very soon (< 60 seconds), try to refresh
	if expiresIn < 60 {
		if h.CanRefreshTokens() && authCode.GoogleRefreshToken != "" {
			h.logger.Info("Google token expired or expiring soon, attempting refresh",
				"email", authCode.UserEmail,
				"expires_in", expiresIn)

			// Attempt immediate refresh
			newToken, refreshErr := refreshGoogleToken(r.Context(), googleToken, h.googleConfig, h.httpClient)
			if refreshErr == nil {
				// Successfully refreshed - use the new token
				h.logger.Info("Google token refreshed during code exchange",
					"email", authCode.UserEmail)
				googleToken = newToken
				expiresIn = newToken.Expiry.Unix() - time.Now().Unix()
			} else {
				// Refresh failed - authorization code is too old
				h.logger.Warn("Failed to refresh expired token during code exchange",
					"email", authCode.UserEmail,
					"error", refreshErr)
				h.writeError(w, "invalid_grant",
					"Authorization code expired and token refresh failed. Please re-authenticate.",
					http.StatusBadRequest)
				return
			}
		} else {
			// Can't refresh - authorization code is too old
			h.logger.Warn("Authorization code expired and refresh not available",
				"email", authCode.UserEmail,
				"expires_in", expiresIn)
			h.writeError(w, "invalid_grant",
				"Authorization code expired. Please re-authenticate.",
				http.StatusBadRequest)
			return
		}
	}

	// Store the Google token for this user so we can use it to access Google APIs
	if err := h.store.SaveGoogleToken(authCode.UserEmail, googleToken); err != nil {
		h.logger.Error("Failed to store Google token", "error", err)
		h.writeError(w, "server_error", "Failed to store token", http.StatusInternalServerError)
		return
	}

	// Map the inboxfewer access token to the user's Google token
	// We use the access token as the key so when requests come in with the Bearer token,
	// we can look up the associated Google token
	if err := h.store.SaveGoogleToken(accessToken, googleToken); err != nil {
		h.logger.Error("Failed to map access token", "error", err)
		h.writeError(w, "server_error", "Failed to store token", http.StatusInternalServerError)
		return
	}

	// Authorization code already deleted by GetAuthorizationCode
	// No cleanup needed here

	h.logger.Info("Issued access token",
		"client_id", clientID,
		"user_email", authCode.UserEmail,
		"scope", authCode.Scope,
	)

	// Return token response
	tokenResp := TokenResponse{
		AccessToken: accessToken,
		TokenType:   "Bearer",
		ExpiresIn:   expiresIn,
		Scope:       authCode.Scope,
	}

	// Include refresh token if Google provided one
	if authCode.GoogleRefreshToken != "" {
		// Generate inboxfewer refresh token
		refreshToken, err := generateSecureToken(48)
		if err == nil {
			// Store refresh token mapping to user email
			if saveErr := h.store.SaveRefreshToken(refreshToken, authCode.UserEmail); saveErr != nil {
				h.logger.Warn("Failed to store refresh token",
					"email", authCode.UserEmail,
					"error", saveErr)
				// Continue without refresh token in response
			} else {
				tokenResp.RefreshToken = refreshToken
				h.logger.Debug("Issued refresh token", "email", authCode.UserEmail)
			}
		} else {
			h.logger.Warn("Failed to generate refresh token", "error", err)
		}
	}

	h.setSecurityHeaders(w)
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Pragma", "no-cache")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(tokenResp)
}

// handleRefreshTokenGrant handles the refresh_token grant type
func (h *Handler) handleRefreshTokenGrant(w http.ResponseWriter, r *http.Request) {
	refreshToken := r.FormValue("refresh_token")
	clientID := r.FormValue("client_id")

	// Validate required parameters
	if refreshToken == "" {
		h.writeError(w, "invalid_request", "refresh_token is required", http.StatusBadRequest)
		return
	}

	// Retrieve user email from refresh token
	userEmail, err := h.store.GetRefreshToken(refreshToken)
	if err != nil {
		h.logger.Warn("Invalid refresh token", "error", err)
		h.writeError(w, "invalid_grant", "Invalid or expired refresh token", http.StatusBadRequest)
		return
	}

	// Get the stored Google token for this user
	googleToken, err := h.store.GetGoogleToken(userEmail)
	if err != nil {
		h.logger.Warn("No Google token found for refresh",
			"email", userEmail,
			"error", err)
		h.writeError(w, "invalid_grant", "User token not found. Please re-authenticate.", http.StatusBadRequest)
		return
	}

	// Validate client if client_id provided
	if clientID != "" {
		// Authenticate client
		_, err := h.clientStore.GetClient(clientID)
		if err != nil {
			h.logger.Warn("Invalid client_id in refresh", "client_id", clientID, "error", err)
			h.writeError(w, "invalid_client", "Invalid client", http.StatusUnauthorized)
			return
		}

		// For confidential clients, validate client secret
		// (similar to authorization code grant)
		// For now, we accept public clients without secret validation
	}

	// Attempt to refresh the Google token if needed
	if h.CanRefreshTokens() && googleToken.RefreshToken != "" {
		newToken, refreshErr := refreshGoogleToken(r.Context(), googleToken, h.googleConfig, h.httpClient)
		if refreshErr == nil {
			// Successfully refreshed - use the new token
			h.logger.Info("Google token refreshed via refresh_token grant", "email", userEmail)
			googleToken = newToken
			// Save the refreshed Google token
			if saveErr := h.store.SaveGoogleToken(userEmail, newToken); saveErr != nil {
				h.logger.Warn("Failed to save refreshed Google token",
					"email", userEmail,
					"error", saveErr)
			}
		} else {
			// Refresh failed - token might be revoked
			h.logger.Warn("Failed to refresh Google token",
				"email", userEmail,
				"error", refreshErr)
			h.writeError(w, "invalid_grant", "Token refresh failed. Please re-authenticate.", http.StatusBadRequest)
			return
		}
	}

	// Generate new inboxfewer access token
	accessToken, err := generateSecureToken(48)
	if err != nil {
		h.logger.Error("Failed to generate access token", "error", err)
		h.writeError(w, "server_error", "Failed to generate access token", http.StatusInternalServerError)
		return
	}

	// Calculate token expiry
	expiresIn := int64(3600) // Default 1 hour
	if !googleToken.Expiry.IsZero() {
		expiresIn = googleToken.Expiry.Unix() - time.Now().Unix()
		if expiresIn < 0 {
			expiresIn = 3600
		}
	}

	// Map the new access token to the Google token
	if err := h.store.SaveGoogleToken(accessToken, googleToken); err != nil {
		h.logger.Error("Failed to map access token", "error", err)
		h.writeError(w, "server_error", "Failed to store token", http.StatusInternalServerError)
		return
	}

	h.logger.Info("Issued new access token via refresh_token grant",
		"email", userEmail)

	// Return token response
	tokenResp := TokenResponse{
		AccessToken: accessToken,
		TokenType:   "Bearer",
		ExpiresIn:   expiresIn,
	}

	// Include the same refresh token (refresh tokens are long-lived)
	tokenResp.RefreshToken = refreshToken

	h.setSecurityHeaders(w)
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Pragma", "no-cache")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(tokenResp)
}

// validateScopes validates that all requested scopes are supported
func (h *Handler) validateScopes(scope string) error {
	if scope == "" {
		return nil // Empty scope is valid
	}

	// Split space-separated scopes
	requestedScopes := strings.Split(scope, " ")

	for _, requested := range requestedScopes {
		requested = strings.TrimSpace(requested)
		if requested == "" {
			continue
		}

		// Check if scope is in the supported list
		found := false
		for _, supported := range h.config.SupportedScopes {
			if requested == supported {
				found = true
				break
			}
		}

		if !found {
			return fmt.Errorf("unsupported scope: %s", requested)
		}
	}

	return nil
}

// validateRedirectURI validates a redirect URI according to OAuth 2.0 Security Best Current Practice
func validateRedirectURI(uri string, serverResource string) error {
	parsed, err := url.Parse(uri)
	if err != nil {
		return fmt.Errorf("invalid redirect_uri format: %s", uri)
	}

	// Reject fragments (OAuth 2.0 Security BCP Section 4.1.3)
	if parsed.Fragment != "" {
		return fmt.Errorf("redirect_uri must not contain fragments: %s", uri)
	}

	// Must have a scheme
	if parsed.Scheme == "" {
		return fmt.Errorf("redirect_uri must have a scheme: %s", uri)
	}

	// Allow custom schemes (for native apps like com.example.app:// or myapp://callback)
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		// Custom schemes are allowed for native mobile/desktop apps
		// We don't enforce structure requirements since custom schemes
		// don't follow http/https rules
		return nil
	}

	// For http/https schemes, require host
	if parsed.Host == "" {
		return fmt.Errorf("http/https redirect_uri must have a host: %s", uri)
	}

	// Determine if we're in production (not localhost)
	serverURL, err := url.Parse(serverResource)
	if err != nil {
		// If we can't parse server resource, be conservative
		return fmt.Errorf("cannot validate redirect_uri: invalid server resource")
	}

	isProduction := !isLoopback(serverURL.Hostname())

	// OAuth 2.0 Security BCP: Loopback redirect URIs are always allowed for local development
	// even in production environments (they can't be intercepted since they're local)
	isLoopbackRedirect := isLoopback(parsed.Hostname())

	if isProduction && !isLoopbackRedirect {
		// In production, require HTTPS for non-localhost redirect URIs
		if parsed.Scheme != "https" {
			return fmt.Errorf("redirect_uri must use HTTPS in production (non-localhost redirects): %s", uri)
		}
	}

	return nil
}

// isLoopback checks if a hostname is a loopback address
func isLoopback(hostname string) bool {
	// Normalize hostname (remove brackets for IPv6)
	hostname = strings.Trim(hostname, "[]")

	return hostname == "localhost" ||
		hostname == "127.0.0.1" ||
		hostname == "::1" ||
		strings.HasPrefix(hostname, "127.") ||
		strings.HasPrefix(hostname, "localhost:")
}

// fetchGoogleUserInfo fetches user info from Google
func (h *Handler) fetchGoogleUserInfo(ctx context.Context, accessToken string) (*GoogleUserInfo, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://www.googleapis.com/oauth2/v2/userinfo", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := h.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch user info: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Google userinfo returned status %d", resp.StatusCode)
	}

	var userInfo GoogleUserInfo
	if err := json.NewDecoder(resp.Body).Decode(&userInfo); err != nil {
		return nil, fmt.Errorf("failed to decode user info: %w", err)
	}

	return &userInfo, nil
}
