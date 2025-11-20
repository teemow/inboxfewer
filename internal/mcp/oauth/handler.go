package oauth

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Config holds the OAuth handler configuration
type Config struct {
	// Issuer is the OAuth issuer identifier (typically the base URL)
	Issuer string

	// Resource is the MCP server resource identifier for RFC 8707
	Resource string

	// DefaultTokenTTL is the default access token TTL in seconds (default: 3600)
	DefaultTokenTTL int64

	// AuthorizationCodeTTL is the authorization code TTL in seconds (default: 600)
	AuthorizationCodeTTL int64

	// AllowedRedirectURIs are the allowed redirect URI patterns
	AllowedRedirectURIs []string

	// DefaultScopes are the default scopes if none requested
	DefaultScopes []string

	// SupportedScopes are all available scopes
	SupportedScopes []string
}

// Handler implements OAuth 2.1 endpoints for the MCP server
type Handler struct {
	config *Config
	store  *Store
}

// NewHandler creates a new OAuth handler
func NewHandler(config *Config) (*Handler, error) {
	if config.Issuer == "" {
		return nil, fmt.Errorf("issuer is required")
	}

	if config.Resource == "" {
		return nil, fmt.Errorf("resource is required")
	}

	// Set defaults
	if config.DefaultTokenTTL == 0 {
		config.DefaultTokenTTL = 3600 // 1 hour
	}

	if config.AuthorizationCodeTTL == 0 {
		config.AuthorizationCodeTTL = 600 // 10 minutes
	}

	if len(config.DefaultScopes) == 0 {
		config.DefaultScopes = []string{"mcp"}
	}

	if len(config.SupportedScopes) == 0 {
		config.SupportedScopes = []string{"mcp"}
	}

	return &Handler{
		config: config,
		store:  NewStore(),
	}, nil
}

// GetStore returns the underlying store (for testing)
func (h *Handler) GetStore() *Store {
	return h.store
}

// ServeWellKnown serves the OAuth 2.0 Authorization Server Metadata (RFC 8414)
func (h *Handler) ServeWellKnown(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	metadata := AuthorizationServerMetadata{
		Issuer:                 h.config.Issuer,
		AuthorizationEndpoint:  h.config.Issuer + "/oauth/authorize",
		TokenEndpoint:          h.config.Issuer + "/oauth/token",
		RegistrationEndpoint:   h.config.Issuer + "/oauth/register",
		ScopesSupported:        h.config.SupportedScopes,
		ResponseTypesSupported: []string{"code"},
		GrantTypesSupported: []string{
			"authorization_code",
			"refresh_token",
		},
		TokenEndpointAuthMethodsSupported: []string{
			"client_secret_post",
			"client_secret_basic",
			"none", // For public clients
		},
		CodeChallengeMethodsSupported: []string{"S256", "plain"},
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(metadata); err != nil {
		http.Error(w, "Failed to encode metadata", http.StatusInternalServerError)
		return
	}
}

// ServeDynamicRegistration handles Dynamic Client Registration (RFC 7591)
func (h *Handler) ServeDynamicRegistration(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req DynamicClientRegistrationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, "invalid_request", "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate redirect URIs
	if len(req.RedirectURIs) == 0 {
		h.writeError(w, "invalid_redirect_uri", "At least one redirect URI is required", http.StatusBadRequest)
		return
	}

	for _, uri := range req.RedirectURIs {
		if err := h.validateRedirectURI(uri); err != nil {
			h.writeError(w, "invalid_redirect_uri", err.Error(), http.StatusBadRequest)
			return
		}
	}

	// Generate client ID
	clientID, err := GenerateClientID()
	if err != nil {
		h.writeError(w, "server_error", "Failed to generate client ID", http.StatusInternalServerError)
		return
	}

	// Determine if public client or confidential client
	isPublic := req.TokenEndpointAuthMethod == "none"
	var clientSecret string

	if !isPublic {
		clientSecret, err = GenerateClientSecret()
		if err != nil {
			h.writeError(w, "server_error", "Failed to generate client secret", http.StatusInternalServerError)
			return
		}
	}

	// Set defaults
	if len(req.GrantTypes) == 0 {
		req.GrantTypes = []string{"authorization_code", "refresh_token"}
	}
	if len(req.ResponseTypes) == 0 {
		req.ResponseTypes = []string{"code"}
	}
	if req.TokenEndpointAuthMethod == "" {
		if isPublic {
			req.TokenEndpointAuthMethod = "none"
		} else {
			req.TokenEndpointAuthMethod = "client_secret_post"
		}
	}

	// Create client info
	client := &ClientInfo{
		ClientID:                clientID,
		ClientSecret:            clientSecret,
		RedirectURIs:            req.RedirectURIs,
		ClientName:              req.ClientName,
		ClientURI:               req.ClientURI,
		GrantTypes:              req.GrantTypes,
		ResponseTypes:           req.ResponseTypes,
		TokenEndpointAuthMethod: req.TokenEndpointAuthMethod,
		CreatedAt:               time.Now(),
		IsPublic:                isPublic,
	}

	// Save client
	if err := h.store.SaveClient(client); err != nil {
		h.writeError(w, "server_error", "Failed to save client", http.StatusInternalServerError)
		return
	}

	// Return client info
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(client); err != nil {
		return
	}
}

// ServeAuthorize handles the authorization endpoint
func (h *Handler) ServeAuthorize(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse request parameters
	if err := r.ParseForm(); err != nil {
		h.redirectError(w, r, "", "invalid_request", "Failed to parse form")
		return
	}

	authReq := &AuthorizationRequest{
		ResponseType:        r.Form.Get("response_type"),
		ClientID:            r.Form.Get("client_id"),
		RedirectURI:         r.Form.Get("redirect_uri"),
		Scope:               r.Form.Get("scope"),
		State:               r.Form.Get("state"),
		CodeChallenge:       r.Form.Get("code_challenge"),
		CodeChallengeMethod: r.Form.Get("code_challenge_method"),
		Resource:            r.Form.Get("resource"),
	}

	// Validate request
	if authReq.ResponseType != "code" {
		h.redirectError(w, r, authReq.RedirectURI, "unsupported_response_type", "Only 'code' response type is supported")
		return
	}

	if authReq.ClientID == "" {
		h.redirectError(w, r, authReq.RedirectURI, "invalid_request", "client_id is required")
		return
	}

	// Get client
	client, err := h.store.GetClient(authReq.ClientID)
	if err != nil {
		h.redirectError(w, r, authReq.RedirectURI, "invalid_client", "Unknown client")
		return
	}

	// Validate redirect URI
	if authReq.RedirectURI == "" {
		if len(client.RedirectURIs) == 1 {
			authReq.RedirectURI = client.RedirectURIs[0]
		} else {
			h.redirectError(w, r, "", "invalid_request", "redirect_uri is required when client has multiple redirect URIs")
			return
		}
	}

	if !h.isAllowedRedirectURI(client, authReq.RedirectURI) {
		h.redirectError(w, r, "", "invalid_request", "Invalid redirect_uri")
		return
	}

	// Validate PKCE (required for public clients, optional for confidential)
	if authReq.CodeChallenge == "" && client.IsPublic {
		h.redirectError(w, r, authReq.RedirectURI, "invalid_request", "code_challenge is required for public clients")
		return
	}

	if authReq.CodeChallenge != "" && authReq.CodeChallengeMethod == "" {
		authReq.CodeChallengeMethod = "plain"
	}

	// Validate resource parameter (RFC 8707)
	if authReq.Resource != "" && authReq.Resource != h.config.Resource {
		h.redirectError(w, r, authReq.RedirectURI, "invalid_target", "Invalid resource parameter")
		return
	}

	// Set default scope if not specified
	if authReq.Scope == "" {
		authReq.Scope = strings.Join(h.config.DefaultScopes, " ")
	}

	// Generate authorization code
	code, err := GenerateAuthorizationCode()
	if err != nil {
		h.redirectError(w, r, authReq.RedirectURI, "server_error", "Failed to generate authorization code")
		return
	}

	// Store authorization code
	authCode := &AuthorizationCode{
		Code:                code,
		ClientID:            authReq.ClientID,
		RedirectURI:         authReq.RedirectURI,
		Scope:               authReq.Scope,
		CodeChallenge:       authReq.CodeChallenge,
		CodeChallengeMethod: authReq.CodeChallengeMethod,
		Resource:            authReq.Resource,
		ExpiresAt:           time.Now().Add(time.Duration(h.config.AuthorizationCodeTTL) * time.Second),
		Used:                false,
	}

	if err := h.store.SaveAuthorizationCode(authCode); err != nil {
		h.redirectError(w, r, authReq.RedirectURI, "server_error", "Failed to save authorization code")
		return
	}

	// Redirect back to client with code
	redirectURL, err := url.Parse(authReq.RedirectURI)
	if err != nil {
		h.redirectError(w, r, "", "invalid_request", "Invalid redirect_uri")
		return
	}

	q := redirectURL.Query()
	q.Set("code", code)
	if authReq.State != "" {
		q.Set("state", authReq.State)
	}
	redirectURL.RawQuery = q.Encode()

	http.Redirect(w, r, redirectURL.String(), http.StatusFound)
}

// ServeToken handles the token endpoint
func (h *Handler) ServeToken(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if err := r.ParseForm(); err != nil {
		h.writeError(w, "invalid_request", "Failed to parse form", http.StatusBadRequest)
		return
	}

	grantType := r.Form.Get("grant_type")

	switch grantType {
	case "authorization_code":
		h.handleAuthorizationCodeGrant(w, r)
	case "refresh_token":
		h.handleRefreshTokenGrant(w, r)
	default:
		h.writeError(w, "unsupported_grant_type", "Grant type not supported", http.StatusBadRequest)
	}
}

// handleAuthorizationCodeGrant handles the authorization_code grant type
func (h *Handler) handleAuthorizationCodeGrant(w http.ResponseWriter, r *http.Request) {
	tokenReq := &TokenRequest{
		GrantType:    r.Form.Get("grant_type"),
		Code:         r.Form.Get("code"),
		RedirectURI:  r.Form.Get("redirect_uri"),
		ClientID:     r.Form.Get("client_id"),
		ClientSecret: r.Form.Get("client_secret"),
		CodeVerifier: r.Form.Get("code_verifier"),
		Resource:     r.Form.Get("resource"),
	}

	// Try to get client credentials from Authorization header
	if username, password, ok := r.BasicAuth(); ok {
		tokenReq.ClientID = username
		tokenReq.ClientSecret = password
	}

	// Validate required parameters
	if tokenReq.Code == "" {
		h.writeError(w, "invalid_request", "code is required", http.StatusBadRequest)
		return
	}

	if tokenReq.ClientID == "" {
		h.writeError(w, "invalid_request", "client_id is required", http.StatusBadRequest)
		return
	}

	// Get authorization code
	authCode, err := h.store.GetAuthorizationCode(tokenReq.Code)
	if err != nil {
		h.writeError(w, "invalid_grant", "Invalid authorization code", http.StatusBadRequest)
		return
	}

	// Validate client
	if authCode.ClientID != tokenReq.ClientID {
		h.writeError(w, "invalid_grant", "Client ID mismatch", http.StatusBadRequest)
		return
	}

	// Get client info
	client, err := h.store.GetClient(tokenReq.ClientID)
	if err != nil {
		h.writeError(w, "invalid_client", "Unknown client", http.StatusUnauthorized)
		return
	}

	// Validate client credentials (for confidential clients)
	if !client.IsPublic {
		if _, err := h.store.ValidateClientCredentials(tokenReq.ClientID, tokenReq.ClientSecret); err != nil {
			h.writeError(w, "invalid_client", "Invalid client credentials", http.StatusUnauthorized)
			return
		}
	}

	// Validate redirect URI
	if tokenReq.RedirectURI != authCode.RedirectURI {
		h.writeError(w, "invalid_grant", "Redirect URI mismatch", http.StatusBadRequest)
		return
	}

	// Validate PKCE
	if authCode.CodeChallenge != "" {
		if tokenReq.CodeVerifier == "" {
			h.writeError(w, "invalid_grant", "code_verifier is required", http.StatusBadRequest)
			return
		}

		if !ValidateCodeChallenge(tokenReq.CodeVerifier, authCode.CodeChallenge, authCode.CodeChallengeMethod) {
			h.writeError(w, "invalid_grant", "Invalid code_verifier", http.StatusBadRequest)
			return
		}
	}

	// Mark code as used
	if err := h.store.MarkAuthorizationCodeUsed(tokenReq.Code); err != nil {
		h.writeError(w, "server_error", "Failed to mark code as used", http.StatusInternalServerError)
		return
	}

	// Generate tokens
	accessToken, err := GenerateAccessToken()
	if err != nil {
		h.writeError(w, "server_error", "Failed to generate access token", http.StatusInternalServerError)
		return
	}

	refreshToken, err := GenerateRefreshToken()
	if err != nil {
		h.writeError(w, "server_error", "Failed to generate refresh token", http.StatusInternalServerError)
		return
	}

	// Create token
	token := &Token{
		AccessToken:  accessToken,
		TokenType:    "Bearer",
		ExpiresIn:    h.config.DefaultTokenTTL,
		RefreshToken: refreshToken,
		Scope:        authCode.Scope,
		IssuedAt:     time.Now(),
		ClientID:     tokenReq.ClientID,
		Resource:     h.config.Resource,
	}

	// Save token
	if err := h.store.SaveToken(token); err != nil {
		h.writeError(w, "server_error", "Failed to save token", http.StatusInternalServerError)
		return
	}

	// Return token response
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Pragma", "no-cache")
	if err := json.NewEncoder(w).Encode(token); err != nil {
		return
	}
}

// handleRefreshTokenGrant handles the refresh_token grant type
func (h *Handler) handleRefreshTokenGrant(w http.ResponseWriter, r *http.Request) {
	tokenReq := &TokenRequest{
		GrantType:    r.Form.Get("grant_type"),
		RefreshToken: r.Form.Get("refresh_token"),
		ClientID:     r.Form.Get("client_id"),
		ClientSecret: r.Form.Get("client_secret"),
	}

	// Try to get client credentials from Authorization header
	if username, password, ok := r.BasicAuth(); ok {
		tokenReq.ClientID = username
		tokenReq.ClientSecret = password
	}

	// Validate required parameters
	if tokenReq.RefreshToken == "" {
		h.writeError(w, "invalid_request", "refresh_token is required", http.StatusBadRequest)
		return
	}

	if tokenReq.ClientID == "" {
		h.writeError(w, "invalid_request", "client_id is required", http.StatusBadRequest)
		return
	}

	// Get existing token
	oldToken, err := h.store.GetTokenByRefreshToken(tokenReq.RefreshToken)
	if err != nil {
		h.writeError(w, "invalid_grant", "Invalid refresh token", http.StatusBadRequest)
		return
	}

	// Validate client
	if oldToken.ClientID != tokenReq.ClientID {
		h.writeError(w, "invalid_grant", "Client ID mismatch", http.StatusBadRequest)
		return
	}

	// Get client info
	client, err := h.store.GetClient(tokenReq.ClientID)
	if err != nil {
		h.writeError(w, "invalid_client", "Unknown client", http.StatusUnauthorized)
		return
	}

	// Validate client credentials (for confidential clients)
	if !client.IsPublic {
		if _, err := h.store.ValidateClientCredentials(tokenReq.ClientID, tokenReq.ClientSecret); err != nil {
			h.writeError(w, "invalid_client", "Invalid client credentials", http.StatusUnauthorized)
			return
		}
	}

	// Delete old token
	if err := h.store.DeleteTokenByRefreshToken(tokenReq.RefreshToken); err != nil {
		h.writeError(w, "server_error", "Failed to delete old token", http.StatusInternalServerError)
		return
	}

	// Generate new tokens
	accessToken, err := GenerateAccessToken()
	if err != nil {
		h.writeError(w, "server_error", "Failed to generate access token", http.StatusInternalServerError)
		return
	}

	refreshToken, err := GenerateRefreshToken()
	if err != nil {
		h.writeError(w, "server_error", "Failed to generate refresh token", http.StatusInternalServerError)
		return
	}

	// Create new token
	token := &Token{
		AccessToken:  accessToken,
		TokenType:    "Bearer",
		ExpiresIn:    h.config.DefaultTokenTTL,
		RefreshToken: refreshToken,
		Scope:        oldToken.Scope,
		IssuedAt:     time.Now(),
		ClientID:     tokenReq.ClientID,
		Resource:     oldToken.Resource,
	}

	// Save token
	if err := h.store.SaveToken(token); err != nil {
		h.writeError(w, "server_error", "Failed to save token", http.StatusInternalServerError)
		return
	}

	// Return token response
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Pragma", "no-cache")
	if err := json.NewEncoder(w).Encode(token); err != nil {
		return
	}
}

// validateRedirectURI validates a redirect URI
func (h *Handler) validateRedirectURI(uri string) error {
	parsed, err := url.Parse(uri)
	if err != nil {
		return fmt.Errorf("invalid URI format")
	}

	// Must use HTTPS or localhost
	if parsed.Scheme != "https" && parsed.Hostname() != "localhost" && parsed.Hostname() != "127.0.0.1" {
		return fmt.Errorf("redirect URI must use HTTPS or localhost")
	}

	return nil
}

// isAllowedRedirectURI checks if a redirect URI is allowed for a client
func (h *Handler) isAllowedRedirectURI(client *ClientInfo, uri string) bool {
	for _, allowed := range client.RedirectURIs {
		if uri == allowed {
			return true
		}
	}
	return false
}

// redirectError redirects to the redirect URI with an error
func (h *Handler) redirectError(w http.ResponseWriter, r *http.Request, redirectURI, errorCode, errorDescription string) {
	if redirectURI == "" {
		h.writeError(w, errorCode, errorDescription, http.StatusBadRequest)
		return
	}

	parsed, err := url.Parse(redirectURI)
	if err != nil {
		h.writeError(w, errorCode, errorDescription, http.StatusBadRequest)
		return
	}

	q := parsed.Query()
	q.Set("error", errorCode)
	if errorDescription != "" {
		q.Set("error_description", errorDescription)
	}

	state := r.Form.Get("state")
	if state != "" {
		q.Set("state", state)
	}

	parsed.RawQuery = q.Encode()
	http.Redirect(w, r, parsed.String(), http.StatusFound)
}

// writeError writes an OAuth error response
func (h *Handler) writeError(w http.ResponseWriter, errorCode, errorDescription string, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	errResp := ErrorResponse{
		Error:            errorCode,
		ErrorDescription: errorDescription,
	}

	if err := json.NewEncoder(w).Encode(errResp); err != nil {
		return
	}
}
