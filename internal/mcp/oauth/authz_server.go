package oauth

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
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

	// Validate redirect URIs format
	for _, uri := range req.RedirectURIs {
		parsedURI, err := url.Parse(uri)
		if err != nil {
			h.writeError(w, "invalid_redirect_uri", fmt.Sprintf("Invalid redirect_uri: %s", uri), http.StatusBadRequest)
			return
		}
		// Must have a scheme and host (or be a custom URI scheme)
		if parsedURI.Scheme == "" {
			h.writeError(w, "invalid_redirect_uri", fmt.Sprintf("redirect_uri must have a scheme: %s", uri), http.StatusBadRequest)
			return
		}
		// For http/https schemes, must have a valid host
		if (parsedURI.Scheme == "http" || parsedURI.Scheme == "https") && parsedURI.Host == "" {
			h.writeError(w, "invalid_redirect_uri", fmt.Sprintf("redirect_uri must have a host: %s", uri), http.StatusBadRequest)
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

	if state == "" {
		h.writeError(w, "invalid_request", "state is required", http.StatusBadRequest)
		return
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
		oauth2.AccessTypeOffline,  // Request refresh token
		oauth2.ApprovalForce,      // Always show consent screen
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
	redirectQuery.Set("state", authState.State)
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

	if clientID == "" {
		h.writeError(w, "invalid_request", "client_id is required", http.StatusBadRequest)
		return
	}

	// Retrieve and validate authorization code
	authCode, err := h.flowStore.GetAuthorizationCode(code)
	if err != nil {
		h.logger.Warn("Invalid authorization code", "error", err)
		h.writeError(w, "invalid_grant", "Invalid or expired authorization code", http.StatusBadRequest)
		return
	}

	// Validate client_id matches
	if authCode.ClientID != clientID {
		h.logger.Warn("Client ID mismatch",
			"expected", authCode.ClientID,
			"got", clientID,
		)
		h.writeError(w, "invalid_grant", "Client ID mismatch", http.StatusBadRequest)
		return
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

		// Verify code_verifier
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

	// Calculate token expiry (use Google token expiry)
	expiresIn := authCode.GoogleTokenExpiry - time.Now().Unix()
	if expiresIn < 0 {
		expiresIn = 3600 // Default to 1 hour if Google token already expired
	}

	// Store the inboxfewer token mapped to Google token
	googleToken := &oauth2.Token{
		AccessToken:  authCode.GoogleAccessToken,
		RefreshToken: authCode.GoogleRefreshToken,
		Expiry:       time.Unix(authCode.GoogleTokenExpiry, 0),
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

	// Clean up authorization code
	h.flowStore.DeleteAuthorizationCode(code)

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
			tokenResp.RefreshToken = refreshToken
			// TODO: Store refresh token mapping
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Pragma", "no-cache")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(tokenResp)
}

// handleRefreshTokenGrant handles the refresh_token grant type
func (h *Handler) handleRefreshTokenGrant(w http.ResponseWriter, r *http.Request) {
	// TODO: Implement refresh token grant
	h.writeError(w, "unsupported_grant_type", "Refresh token grant not yet implemented", http.StatusNotImplemented)
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

