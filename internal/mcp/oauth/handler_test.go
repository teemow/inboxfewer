package oauth

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestNewHandler(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr bool
	}{
		{
			name: "valid config",
			config: &Config{
				Issuer:   "https://example.com",
				Resource: "https://mcp.example.com",
			},
			wantErr: false,
		},
		{
			name: "missing issuer",
			config: &Config{
				Resource: "https://mcp.example.com",
			},
			wantErr: true,
		},
		{
			name: "missing resource",
			config: &Config{
				Issuer: "https://example.com",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler, err := NewHandler(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewHandler() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && handler == nil {
				t.Error("NewHandler() returned nil handler")
			}
		})
	}
}

func TestHandler_ServeWellKnown(t *testing.T) {
	handler, err := NewHandler(&Config{
		Issuer:          "https://example.com",
		Resource:        "https://mcp.example.com",
		SupportedScopes: []string{"mcp", "admin"},
	})
	if err != nil {
		t.Fatalf("NewHandler() error = %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/.well-known/oauth-authorization-server", nil)
	w := httptest.NewRecorder()

	handler.ServeWellKnown(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("ServeWellKnown() status = %d, want %d", w.Code, http.StatusOK)
	}

	var metadata AuthorizationServerMetadata
	if err := json.NewDecoder(w.Body).Decode(&metadata); err != nil {
		t.Fatalf("Failed to decode metadata: %v", err)
	}

	if metadata.Issuer != "https://example.com" {
		t.Errorf("metadata.Issuer = %s, want https://example.com", metadata.Issuer)
	}

	if len(metadata.ScopesSupported) != 2 {
		t.Errorf("metadata.ScopesSupported length = %d, want 2", len(metadata.ScopesSupported))
	}
}

func TestHandler_ServeWellKnown_MethodNotAllowed(t *testing.T) {
	handler, _ := NewHandler(&Config{
		Issuer:   "https://example.com",
		Resource: "https://mcp.example.com",
	})

	req := httptest.NewRequest(http.MethodPost, "/.well-known/oauth-authorization-server", nil)
	w := httptest.NewRecorder()

	handler.ServeWellKnown(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("ServeWellKnown() status = %d, want %d", w.Code, http.StatusMethodNotAllowed)
	}
}

func TestHandler_ServeDynamicRegistration(t *testing.T) {
	handler, err := NewHandler(&Config{
		Issuer:   "https://example.com",
		Resource: "https://mcp.example.com",
	})
	if err != nil {
		t.Fatalf("NewHandler() error = %v", err)
	}

	regReq := DynamicClientRegistrationRequest{
		RedirectURIs: []string{"https://client.example.com/callback"},
		ClientName:   "Test Client",
	}

	body, _ := json.Marshal(regReq)
	req := httptest.NewRequest(http.MethodPost, "/oauth/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.ServeDynamicRegistration(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("ServeDynamicRegistration() status = %d, want %d, body: %s", w.Code, http.StatusCreated, w.Body.String())
	}

	var client ClientInfo
	if err := json.NewDecoder(w.Body).Decode(&client); err != nil {
		t.Fatalf("Failed to decode client: %v", err)
	}

	if client.ClientID == "" {
		t.Error("ClientID should not be empty")
	}

	if client.ClientName != "Test Client" {
		t.Errorf("ClientName = %s, want Test Client", client.ClientName)
	}
}

func TestHandler_ServeDynamicRegistration_InvalidRedirectURI(t *testing.T) {
	handler, _ := NewHandler(&Config{
		Issuer:   "https://example.com",
		Resource: "https://mcp.example.com",
	})

	regReq := DynamicClientRegistrationRequest{
		RedirectURIs: []string{"http://insecure.example.com/callback"},
		ClientName:   "Test Client",
	}

	body, _ := json.Marshal(regReq)
	req := httptest.NewRequest(http.MethodPost, "/oauth/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.ServeDynamicRegistration(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("ServeDynamicRegistration() status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestHandler_ServeAuthorize(t *testing.T) {
	handler, err := NewHandler(&Config{
		Issuer:   "https://example.com",
		Resource: "https://mcp.example.com",
	})
	if err != nil {
		t.Fatalf("NewHandler() error = %v", err)
	}

	// Register a client first
	client := &ClientInfo{
		ClientID:     "test-client",
		RedirectURIs: []string{"https://client.example.com/callback"},
		IsPublic:     true,
	}
	if err := handler.store.SaveClient(client); err != nil {
		t.Fatalf("SaveClient() error = %v", err)
	}

	// Generate PKCE challenge
	verifier, _ := GenerateCodeVerifier()
	challenge := GenerateCodeChallenge(verifier)

	// Build authorization request
	params := url.Values{}
	params.Set("response_type", "code")
	params.Set("client_id", "test-client")
	params.Set("redirect_uri", "https://client.example.com/callback")
	params.Set("state", "test-state")
	params.Set("code_challenge", challenge)
	params.Set("code_challenge_method", "S256")

	req := httptest.NewRequest(http.MethodGet, "/oauth/authorize?"+params.Encode(), nil)
	w := httptest.NewRecorder()

	handler.ServeAuthorize(w, req)

	if w.Code != http.StatusFound {
		t.Errorf("ServeAuthorize() status = %d, want %d, body: %s", w.Code, http.StatusFound, w.Body.String())
	}

	location := w.Header().Get("Location")
	if location == "" {
		t.Fatal("Location header should not be empty")
	}

	parsedURL, _ := url.Parse(location)
	code := parsedURL.Query().Get("code")
	if code == "" {
		t.Error("Authorization code should not be empty")
	}

	state := parsedURL.Query().Get("state")
	if state != "test-state" {
		t.Errorf("state = %s, want test-state", state)
	}
}

func TestHandler_ServeAuthorize_MissingClientID(t *testing.T) {
	handler, _ := NewHandler(&Config{
		Issuer:   "https://example.com",
		Resource: "https://mcp.example.com",
	})

	req := httptest.NewRequest(http.MethodGet, "/oauth/authorize?response_type=code", nil)
	w := httptest.NewRecorder()

	handler.ServeAuthorize(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("ServeAuthorize() status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestHandler_ServeToken_AuthorizationCode(t *testing.T) {
	handler, err := NewHandler(&Config{
		Issuer:   "https://example.com",
		Resource: "https://mcp.example.com",
	})
	if err != nil {
		t.Fatalf("NewHandler() error = %v", err)
	}

	// Register a client
	client := &ClientInfo{
		ClientID:     "test-client",
		RedirectURIs: []string{"https://client.example.com/callback"},
		IsPublic:     true,
	}
	if err := handler.store.SaveClient(client); err != nil {
		t.Fatalf("SaveClient() error = %v", err)
	}

	// Generate PKCE
	verifier, _ := GenerateCodeVerifier()
	challenge := GenerateCodeChallenge(verifier)

	// Create authorization code
	code, _ := GenerateAuthorizationCode()
	authCode := &AuthorizationCode{
		Code:                code,
		ClientID:            "test-client",
		RedirectURI:         "https://client.example.com/callback",
		Scope:               "mcp",
		CodeChallenge:       challenge,
		CodeChallengeMethod: "S256",
		ExpiresAt:           time.Now().Add(10 * time.Minute),
		Used:                false,
	}
	if err := handler.store.SaveAuthorizationCode(authCode); err != nil {
		t.Fatalf("SaveAuthorizationCode() error = %v", err)
	}

	// Build token request
	params := url.Values{}
	params.Set("grant_type", "authorization_code")
	params.Set("code", code)
	params.Set("redirect_uri", "https://client.example.com/callback")
	params.Set("client_id", "test-client")
	params.Set("code_verifier", verifier)

	req := httptest.NewRequest(http.MethodPost, "/oauth/token", strings.NewReader(params.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	handler.ServeToken(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("ServeToken() status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var token Token
	if err := json.NewDecoder(w.Body).Decode(&token); err != nil {
		t.Fatalf("Failed to decode token: %v", err)
	}

	if token.AccessToken == "" {
		t.Error("AccessToken should not be empty")
	}

	if token.RefreshToken == "" {
		t.Error("RefreshToken should not be empty")
	}

	if token.TokenType != "Bearer" {
		t.Errorf("TokenType = %s, want Bearer", token.TokenType)
	}
}

func TestHandler_ServeToken_InvalidCode(t *testing.T) {
	handler, _ := NewHandler(&Config{
		Issuer:   "https://example.com",
		Resource: "https://mcp.example.com",
	})

	params := url.Values{}
	params.Set("grant_type", "authorization_code")
	params.Set("code", "invalid-code")
	params.Set("redirect_uri", "https://client.example.com/callback")
	params.Set("client_id", "test-client")

	req := httptest.NewRequest(http.MethodPost, "/oauth/token", strings.NewReader(params.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	handler.ServeToken(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("ServeToken() status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestHandler_ServeToken_RefreshToken(t *testing.T) {
	handler, err := NewHandler(&Config{
		Issuer:   "https://example.com",
		Resource: "https://mcp.example.com",
	})
	if err != nil {
		t.Fatalf("NewHandler() error = %v", err)
	}

	// Register a client
	client := &ClientInfo{
		ClientID: "test-client",
		IsPublic: true,
	}
	if err := handler.store.SaveClient(client); err != nil {
		t.Fatalf("SaveClient() error = %v", err)
	}

	// Create a token with refresh token
	accessToken, _ := GenerateAccessToken()
	refreshToken, _ := GenerateRefreshToken()
	token := &Token{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		TokenType:    "Bearer",
		ExpiresIn:    3600,
		Scope:        "mcp",
		IssuedAt:     time.Now(),
		ClientID:     "test-client",
		Resource:     "https://mcp.example.com",
	}
	if err := handler.store.SaveToken(token); err != nil {
		t.Fatalf("SaveToken() error = %v", err)
	}

	// Build refresh token request
	params := url.Values{}
	params.Set("grant_type", "refresh_token")
	params.Set("refresh_token", refreshToken)
	params.Set("client_id", "test-client")

	req := httptest.NewRequest(http.MethodPost, "/oauth/token", strings.NewReader(params.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	handler.ServeToken(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("ServeToken() status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var newToken Token
	if err := json.NewDecoder(w.Body).Decode(&newToken); err != nil {
		t.Fatalf("Failed to decode token: %v", err)
	}

	if newToken.AccessToken == "" {
		t.Error("AccessToken should not be empty")
	}

	if newToken.AccessToken == accessToken {
		t.Error("New access token should be different from old token")
	}
}

func TestHandler_ServeToken_UnsupportedGrantType(t *testing.T) {
	handler, _ := NewHandler(&Config{
		Issuer:   "https://example.com",
		Resource: "https://mcp.example.com",
	})

	params := url.Values{}
	params.Set("grant_type", "password")

	req := httptest.NewRequest(http.MethodPost, "/oauth/token", strings.NewReader(params.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	handler.ServeToken(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("ServeToken() status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestHandler_validateRedirectURI(t *testing.T) {
	handler, _ := NewHandler(&Config{
		Issuer:   "https://example.com",
		Resource: "https://mcp.example.com",
	})

	tests := []struct {
		name    string
		uri     string
		wantErr bool
	}{
		{
			name:    "valid https",
			uri:     "https://example.com/callback",
			wantErr: false,
		},
		{
			name:    "valid localhost http",
			uri:     "http://localhost:8080/callback",
			wantErr: false,
		},
		{
			name:    "valid 127.0.0.1 http",
			uri:     "http://127.0.0.1:8080/callback",
			wantErr: false,
		},
		{
			name:    "invalid http",
			uri:     "http://example.com/callback",
			wantErr: true,
		},
		{
			name:    "invalid format",
			uri:     "not a uri",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := handler.validateRedirectURI(tt.uri)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateRedirectURI() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestHandler_isAllowedRedirectURI(t *testing.T) {
	handler, _ := NewHandler(&Config{
		Issuer:   "https://example.com",
		Resource: "https://mcp.example.com",
	})

	client := &ClientInfo{
		RedirectURIs: []string{
			"https://example.com/callback",
			"https://example.com/callback2",
		},
	}

	tests := []struct {
		name string
		uri  string
		want bool
	}{
		{
			name: "allowed uri 1",
			uri:  "https://example.com/callback",
			want: true,
		},
		{
			name: "allowed uri 2",
			uri:  "https://example.com/callback2",
			want: true,
		},
		{
			name: "not allowed uri",
			uri:  "https://example.com/other",
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := handler.isAllowedRedirectURI(client, tt.uri)
			if got != tt.want {
				t.Errorf("isAllowedRedirectURI() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestHandler_ServeAuthorize_PublicClientNoPKCE(t *testing.T) {
	handler, _ := NewHandler(&Config{
		Issuer:   "https://example.com",
		Resource: "https://mcp.example.com",
	})

	// Register a public client
	client := &ClientInfo{
		ClientID:     "test-client",
		RedirectURIs: []string{"https://client.example.com/callback"},
		IsPublic:     true,
	}
	handler.store.SaveClient(client)

	params := url.Values{}
	params.Set("response_type", "code")
	params.Set("client_id", "test-client")
	params.Set("redirect_uri", "https://client.example.com/callback")

	req := httptest.NewRequest(http.MethodGet, "/oauth/authorize?"+params.Encode(), nil)
	w := httptest.NewRecorder()

	handler.ServeAuthorize(w, req)

	// Should redirect with error because PKCE is required for public clients
	if w.Code != http.StatusFound {
		t.Errorf("ServeAuthorize() status = %d, want %d", w.Code, http.StatusFound)
	}

	location := w.Header().Get("Location")
	parsedURL, _ := url.Parse(location)
	if parsedURL.Query().Get("error") != "invalid_request" {
		t.Errorf("Expected error=invalid_request in redirect, got %s", parsedURL.Query().Get("error"))
	}
}

func TestHandler_ServeAuthorize_MultipleRedirectURIsNoDefault(t *testing.T) {
	handler, _ := NewHandler(&Config{
		Issuer:   "https://example.com",
		Resource: "https://mcp.example.com",
	})

	client := &ClientInfo{
		ClientID: "test-client",
		RedirectURIs: []string{
			"https://client.example.com/callback1",
			"https://client.example.com/callback2",
		},
		IsPublic: false,
	}
	handler.store.SaveClient(client)

	params := url.Values{}
	params.Set("response_type", "code")
	params.Set("client_id", "test-client")

	req := httptest.NewRequest(http.MethodGet, "/oauth/authorize?"+params.Encode(), nil)
	w := httptest.NewRecorder()

	handler.ServeAuthorize(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("ServeAuthorize() status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestHandler_ServeToken_WrongClientID(t *testing.T) {
	handler, _ := NewHandler(&Config{
		Issuer:   "https://example.com",
		Resource: "https://mcp.example.com",
	})

	// Register client
	client := &ClientInfo{
		ClientID: "test-client",
	}
	handler.store.SaveClient(client)

	// Create authorization code for different client
	code, _ := GenerateAuthorizationCode()
	authCode := &AuthorizationCode{
		Code:      code,
		ClientID:  "test-client",
		ExpiresAt: time.Now().Add(10 * time.Minute),
	}
	handler.store.SaveAuthorizationCode(authCode)

	params := url.Values{}
	params.Set("grant_type", "authorization_code")
	params.Set("code", code)
	params.Set("client_id", "wrong-client")
	params.Set("redirect_uri", "https://example.com/callback")

	req := httptest.NewRequest(http.MethodPost, "/oauth/token", strings.NewReader(params.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	handler.ServeToken(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("ServeToken() status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestHandler_ServeToken_ConfidentialClientNoSecret(t *testing.T) {
	handler, _ := NewHandler(&Config{
		Issuer:   "https://example.com",
		Resource: "https://mcp.example.com",
	})

	// Register confidential client
	client := &ClientInfo{
		ClientID:     "test-client",
		ClientSecret: "secret",
		IsPublic:     false,
	}
	handler.store.SaveClient(client)

	verifier, _ := GenerateCodeVerifier()
	challenge := GenerateCodeChallenge(verifier)

	code, _ := GenerateAuthorizationCode()
	authCode := &AuthorizationCode{
		Code:                code,
		ClientID:            "test-client",
		RedirectURI:         "https://example.com/callback",
		CodeChallenge:       challenge,
		CodeChallengeMethod: "S256",
		ExpiresAt:           time.Now().Add(10 * time.Minute),
	}
	handler.store.SaveAuthorizationCode(authCode)

	params := url.Values{}
	params.Set("grant_type", "authorization_code")
	params.Set("code", code)
	params.Set("client_id", "test-client")
	params.Set("redirect_uri", "https://example.com/callback")
	params.Set("code_verifier", verifier)
	// Missing client_secret

	req := httptest.NewRequest(http.MethodPost, "/oauth/token", strings.NewReader(params.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	handler.ServeToken(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("ServeToken() status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestHandler_ServeToken_RedirectURIMismatch(t *testing.T) {
	handler, _ := NewHandler(&Config{
		Issuer:   "https://example.com",
		Resource: "https://mcp.example.com",
	})

	client := &ClientInfo{
		ClientID: "test-client",
		IsPublic: true,
	}
	handler.store.SaveClient(client)

	code, _ := GenerateAuthorizationCode()
	authCode := &AuthorizationCode{
		Code:        code,
		ClientID:    "test-client",
		RedirectURI: "https://example.com/callback",
		ExpiresAt:   time.Now().Add(10 * time.Minute),
	}
	handler.store.SaveAuthorizationCode(authCode)

	params := url.Values{}
	params.Set("grant_type", "authorization_code")
	params.Set("code", code)
	params.Set("client_id", "test-client")
	params.Set("redirect_uri", "https://example.com/different")

	req := httptest.NewRequest(http.MethodPost, "/oauth/token", strings.NewReader(params.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	handler.ServeToken(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("ServeToken() status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestHandler_ServeToken_InvalidPKCE(t *testing.T) {
	handler, _ := NewHandler(&Config{
		Issuer:   "https://example.com",
		Resource: "https://mcp.example.com",
	})

	client := &ClientInfo{
		ClientID: "test-client",
		IsPublic: true,
	}
	handler.store.SaveClient(client)

	verifier, _ := GenerateCodeVerifier()
	challenge := GenerateCodeChallenge(verifier)

	code, _ := GenerateAuthorizationCode()
	authCode := &AuthorizationCode{
		Code:                code,
		ClientID:            "test-client",
		RedirectURI:         "https://example.com/callback",
		CodeChallenge:       challenge,
		CodeChallengeMethod: "S256",
		ExpiresAt:           time.Now().Add(10 * time.Minute),
	}
	handler.store.SaveAuthorizationCode(authCode)

	params := url.Values{}
	params.Set("grant_type", "authorization_code")
	params.Set("code", code)
	params.Set("client_id", "test-client")
	params.Set("redirect_uri", "https://example.com/callback")
	params.Set("code_verifier", "wrong-verifier")

	req := httptest.NewRequest(http.MethodPost, "/oauth/token", strings.NewReader(params.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	handler.ServeToken(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("ServeToken() status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestHandler_ServeToken_MissingCode(t *testing.T) {
	handler, _ := NewHandler(&Config{
		Issuer:   "https://example.com",
		Resource: "https://mcp.example.com",
	})

	params := url.Values{}
	params.Set("grant_type", "authorization_code")
	params.Set("client_id", "test-client")

	req := httptest.NewRequest(http.MethodPost, "/oauth/token", strings.NewReader(params.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	handler.ServeToken(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("ServeToken() status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestHandler_ServeDynamicRegistration_NoRedirectURIs(t *testing.T) {
	handler, _ := NewHandler(&Config{
		Issuer:   "https://example.com",
		Resource: "https://mcp.example.com",
	})

	regReq := DynamicClientRegistrationRequest{
		ClientName: "Test Client",
	}

	body, _ := json.Marshal(regReq)
	req := httptest.NewRequest(http.MethodPost, "/oauth/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.ServeDynamicRegistration(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("ServeDynamicRegistration() status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestHandler_ServeDynamicRegistration_PublicClient(t *testing.T) {
	handler, _ := NewHandler(&Config{
		Issuer:   "https://example.com",
		Resource: "https://mcp.example.com",
	})

	regReq := DynamicClientRegistrationRequest{
		RedirectURIs:            []string{"https://client.example.com/callback"},
		ClientName:              "Public Client",
		TokenEndpointAuthMethod: "none",
	}

	body, _ := json.Marshal(regReq)
	req := httptest.NewRequest(http.MethodPost, "/oauth/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.ServeDynamicRegistration(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("ServeDynamicRegistration() status = %d, want %d, body: %s", w.Code, http.StatusCreated, w.Body.String())
	}

	var client ClientInfo
	json.NewDecoder(w.Body).Decode(&client)

	if client.ClientSecret != "" {
		t.Error("Public client should not have a secret")
	}

	// Retrieve from store to check IsPublic flag
	storedClient, err := handler.store.GetClient(client.ClientID)
	if err != nil {
		t.Fatalf("Failed to get stored client: %v", err)
	}

	if !storedClient.IsPublic {
		t.Error("Client should be marked as public")
	}
}

func TestHandler_ServeAuthorize_UnknownClient(t *testing.T) {
	handler, _ := NewHandler(&Config{
		Issuer:   "https://example.com",
		Resource: "https://mcp.example.com",
	})

	params := url.Values{}
	params.Set("response_type", "code")
	params.Set("client_id", "unknown-client")
	params.Set("redirect_uri", "https://example.com/callback")

	req := httptest.NewRequest(http.MethodGet, "/oauth/authorize?"+params.Encode(), nil)
	w := httptest.NewRecorder()

	handler.ServeAuthorize(w, req)

	// Unknown client should redirect with error since we have redirect_uri
	if w.Code != http.StatusFound {
		t.Errorf("ServeAuthorize() status = %d, want %d", w.Code, http.StatusFound)
	}

	location := w.Header().Get("Location")
	parsedURL, _ := url.Parse(location)
	if parsedURL.Query().Get("error") != "invalid_client" {
		t.Errorf("Expected error=invalid_client, got %s", parsedURL.Query().Get("error"))
	}
}

func TestHandler_ServeAuthorize_InvalidResource(t *testing.T) {
	handler, _ := NewHandler(&Config{
		Issuer:   "https://example.com",
		Resource: "https://mcp.example.com",
	})

	client := &ClientInfo{
		ClientID:     "test-client",
		RedirectURIs: []string{"https://client.example.com/callback"},
	}
	handler.store.SaveClient(client)

	params := url.Values{}
	params.Set("response_type", "code")
	params.Set("client_id", "test-client")
	params.Set("redirect_uri", "https://client.example.com/callback")
	params.Set("resource", "https://wrong-resource.example.com")

	req := httptest.NewRequest(http.MethodGet, "/oauth/authorize?"+params.Encode(), nil)
	w := httptest.NewRecorder()

	handler.ServeAuthorize(w, req)

	if w.Code != http.StatusFound {
		t.Errorf("ServeAuthorize() status = %d, want %d", w.Code, http.StatusFound)
	}

	location := w.Header().Get("Location")
	parsedURL, _ := url.Parse(location)
	if parsedURL.Query().Get("error") != "invalid_target" {
		t.Errorf("Expected error=invalid_target, got %s", parsedURL.Query().Get("error"))
	}
}

func TestHandler_GetStore(t *testing.T) {
	handler, _ := NewHandler(&Config{
		Issuer:   "https://example.com",
		Resource: "https://mcp.example.com",
	})

	store := handler.GetStore()
	if store == nil {
		t.Error("GetStore() should return non-nil store")
	}
}

func TestHandler_ServeToken_BasicAuth(t *testing.T) {
	handler, _ := NewHandler(&Config{
		Issuer:   "https://example.com",
		Resource: "https://mcp.example.com",
	})

	// Register confidential client
	client := &ClientInfo{
		ClientID:     "test-client",
		ClientSecret: "test-secret",
		IsPublic:     false,
	}
	handler.store.SaveClient(client)

	// Create authorization code
	code, _ := GenerateAuthorizationCode()
	authCode := &AuthorizationCode{
		Code:        code,
		ClientID:    "test-client",
		RedirectURI: "https://example.com/callback",
		Scope:       "mcp",
		ExpiresAt:   time.Now().Add(10 * time.Minute),
	}
	handler.store.SaveAuthorizationCode(authCode)

	// Build token request using Basic Auth
	params := url.Values{}
	params.Set("grant_type", "authorization_code")
	params.Set("code", code)
	params.Set("redirect_uri", "https://example.com/callback")

	req := httptest.NewRequest(http.MethodPost, "/oauth/token", strings.NewReader(params.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetBasicAuth("test-client", "test-secret")
	w := httptest.NewRecorder()

	handler.ServeToken(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("ServeToken() with Basic Auth status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}
}

func TestHandler_ServeToken_MissingCodeVerifier(t *testing.T) {
	handler, _ := NewHandler(&Config{
		Issuer:   "https://example.com",
		Resource: "https://mcp.example.com",
	})

	client := &ClientInfo{
		ClientID: "test-client",
		IsPublic: true,
	}
	handler.store.SaveClient(client)

	verifier, _ := GenerateCodeVerifier()
	challenge := GenerateCodeChallenge(verifier)

	code, _ := GenerateAuthorizationCode()
	authCode := &AuthorizationCode{
		Code:                code,
		ClientID:            "test-client",
		RedirectURI:         "https://example.com/callback",
		CodeChallenge:       challenge,
		CodeChallengeMethod: "S256",
		ExpiresAt:           time.Now().Add(10 * time.Minute),
	}
	handler.store.SaveAuthorizationCode(authCode)

	params := url.Values{}
	params.Set("grant_type", "authorization_code")
	params.Set("code", code)
	params.Set("client_id", "test-client")
	params.Set("redirect_uri", "https://example.com/callback")
	// Missing code_verifier

	req := httptest.NewRequest(http.MethodPost, "/oauth/token", strings.NewReader(params.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	handler.ServeToken(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("ServeToken() status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestHandler_ServeToken_UnknownClient(t *testing.T) {
	handler, _ := NewHandler(&Config{
		Issuer:   "https://example.com",
		Resource: "https://mcp.example.com",
	})

	code, _ := GenerateAuthorizationCode()
	authCode := &AuthorizationCode{
		Code:      code,
		ClientID:  "test-client",
		ExpiresAt: time.Now().Add(10 * time.Minute),
	}
	handler.store.SaveAuthorizationCode(authCode)

	params := url.Values{}
	params.Set("grant_type", "authorization_code")
	params.Set("code", code)
	params.Set("client_id", "test-client")
	params.Set("redirect_uri", "https://example.com/callback")

	req := httptest.NewRequest(http.MethodPost, "/oauth/token", strings.NewReader(params.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	handler.ServeToken(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("ServeToken() status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestHandler_ServeToken_RefreshToken_UnknownClient(t *testing.T) {
	handler, _ := NewHandler(&Config{
		Issuer:   "https://example.com",
		Resource: "https://mcp.example.com",
	})

	// Create token with non-existent client
	token := &Token{
		AccessToken:  "access-token",
		RefreshToken: "refresh-token",
		ExpiresIn:    3600,
		IssuedAt:     time.Now(),
		ClientID:     "nonexistent-client",
		Resource:     "https://mcp.example.com",
	}
	handler.store.SaveToken(token)

	params := url.Values{}
	params.Set("grant_type", "refresh_token")
	params.Set("refresh_token", "refresh-token")
	params.Set("client_id", "nonexistent-client")

	req := httptest.NewRequest(http.MethodPost, "/oauth/token", strings.NewReader(params.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	handler.ServeToken(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("ServeToken() refresh token with unknown client status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestHandler_ServeToken_RefreshToken_MissingRefreshToken(t *testing.T) {
	handler, _ := NewHandler(&Config{
		Issuer:   "https://example.com",
		Resource: "https://mcp.example.com",
	})

	params := url.Values{}
	params.Set("grant_type", "refresh_token")
	params.Set("client_id", "test-client")

	req := httptest.NewRequest(http.MethodPost, "/oauth/token", strings.NewReader(params.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	handler.ServeToken(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("ServeToken() status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestHandler_ServeToken_RefreshToken_ClientMismatch(t *testing.T) {
	handler, _ := NewHandler(&Config{
		Issuer:   "https://example.com",
		Resource: "https://mcp.example.com",
	})

	// Register clients
	client1 := &ClientInfo{ClientID: "client1", IsPublic: true}
	client2 := &ClientInfo{ClientID: "client2", IsPublic: true}
	handler.store.SaveClient(client1)
	handler.store.SaveClient(client2)

	// Create token for client1
	token := &Token{
		AccessToken:  "access-token",
		RefreshToken: "refresh-token",
		ExpiresIn:    3600,
		IssuedAt:     time.Now(),
		ClientID:     "client1",
		Resource:     "https://mcp.example.com",
	}
	handler.store.SaveToken(token)

	// Try to use refresh token with client2
	params := url.Values{}
	params.Set("grant_type", "refresh_token")
	params.Set("refresh_token", "refresh-token")
	params.Set("client_id", "client2")

	req := httptest.NewRequest(http.MethodPost, "/oauth/token", strings.NewReader(params.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	handler.ServeToken(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("ServeToken() status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestHandler_ServeAuthorize_DefaultScope(t *testing.T) {
	handler, _ := NewHandler(&Config{
		Issuer:        "https://example.com",
		Resource:      "https://mcp.example.com",
		DefaultScopes: []string{"read", "write"},
	})

	client := &ClientInfo{
		ClientID:     "test-client",
		RedirectURIs: []string{"https://client.example.com/callback"},
	}
	handler.store.SaveClient(client)

	params := url.Values{}
	params.Set("response_type", "code")
	params.Set("client_id", "test-client")
	params.Set("redirect_uri", "https://client.example.com/callback")
	// No scope parameter

	req := httptest.NewRequest(http.MethodGet, "/oauth/authorize?"+params.Encode(), nil)
	w := httptest.NewRecorder()

	handler.ServeAuthorize(w, req)

	if w.Code != http.StatusFound {
		t.Errorf("ServeAuthorize() status = %d, want %d", w.Code, http.StatusFound)
	}

	location := w.Header().Get("Location")
	parsedURL, _ := url.Parse(location)
	code := parsedURL.Query().Get("code")

	// Verify the authorization code has the default scope
	authCode, _ := handler.store.GetAuthorizationCode(code)
	if authCode.Scope != "read write" {
		t.Errorf("authCode.Scope = %s, want 'read write'", authCode.Scope)
	}
}

func TestHandler_ServeAuthorize_UnsupportedResponseType(t *testing.T) {
	handler, _ := NewHandler(&Config{
		Issuer:   "https://example.com",
		Resource: "https://mcp.example.com",
	})

	params := url.Values{}
	params.Set("response_type", "token") // Not supported
	params.Set("client_id", "test-client")
	params.Set("redirect_uri", "https://client.example.com/callback")

	req := httptest.NewRequest(http.MethodGet, "/oauth/authorize?"+params.Encode(), nil)
	w := httptest.NewRecorder()

	handler.ServeAuthorize(w, req)

	if w.Code != http.StatusFound {
		t.Errorf("ServeAuthorize() status = %d, want %d", w.Code, http.StatusFound)
	}

	location := w.Header().Get("Location")
	parsedURL, _ := url.Parse(location)
	if parsedURL.Query().Get("error") != "unsupported_response_type" {
		t.Errorf("Expected error=unsupported_response_type, got %s", parsedURL.Query().Get("error"))
	}
}

func TestHandler_ServeDynamicRegistration_InvalidJSON(t *testing.T) {
	handler, _ := NewHandler(&Config{
		Issuer:   "https://example.com",
		Resource: "https://mcp.example.com",
	})

	req := httptest.NewRequest(http.MethodPost, "/oauth/register", bytes.NewReader([]byte("invalid json")))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.ServeDynamicRegistration(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("ServeDynamicRegistration() status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestHandler_ServeAuthorize_SingleRedirectURI(t *testing.T) {
	handler, _ := NewHandler(&Config{
		Issuer:   "https://example.com",
		Resource: "https://mcp.example.com",
	})

	// Client with single redirect URI
	client := &ClientInfo{
		ClientID:     "test-client",
		RedirectURIs: []string{"https://client.example.com/callback"},
	}
	handler.store.SaveClient(client)

	params := url.Values{}
	params.Set("response_type", "code")
	params.Set("client_id", "test-client")
	// No redirect_uri parameter - should use the single registered one

	req := httptest.NewRequest(http.MethodGet, "/oauth/authorize?"+params.Encode(), nil)
	w := httptest.NewRecorder()

	handler.ServeAuthorize(w, req)

	if w.Code != http.StatusFound {
		t.Errorf("ServeAuthorize() status = %d, want %d", w.Code, http.StatusFound)
	}
}

func TestHandler_ServeAuthorize_InvalidRedirectURI(t *testing.T) {
	handler, _ := NewHandler(&Config{
		Issuer:   "https://example.com",
		Resource: "https://mcp.example.com",
	})

	client := &ClientInfo{
		ClientID:     "test-client",
		RedirectURIs: []string{"https://client.example.com/callback"},
	}
	handler.store.SaveClient(client)

	params := url.Values{}
	params.Set("response_type", "code")
	params.Set("client_id", "test-client")
	params.Set("redirect_uri", "https://different.example.com/callback")

	req := httptest.NewRequest(http.MethodGet, "/oauth/authorize?"+params.Encode(), nil)
	w := httptest.NewRecorder()

	handler.ServeAuthorize(w, req)

	// Should return error without redirect since redirect_uri is invalid
	if w.Code != http.StatusBadRequest {
		t.Errorf("ServeAuthorize() status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestNewHandler_WithDefaults(t *testing.T) {
	config := &Config{
		Issuer:   "https://example.com",
		Resource: "https://mcp.example.com",
		// Leave other fields empty to test defaults
	}

	handler, err := NewHandler(config)
	if err != nil {
		t.Fatalf("NewHandler() error = %v", err)
	}

	if config.DefaultTokenTTL != 3600 {
		t.Errorf("Default TokenTTL = %d, want 3600", config.DefaultTokenTTL)
	}

	if config.AuthorizationCodeTTL != 600 {
		t.Errorf("Default AuthorizationCodeTTL = %d, want 600", config.AuthorizationCodeTTL)
	}

	if len(config.DefaultScopes) != 1 || config.DefaultScopes[0] != "mcp" {
		t.Errorf("Default DefaultScopes = %v, want [mcp]", config.DefaultScopes)
	}

	if handler.GetStore() == nil {
		t.Error("Handler store should not be nil")
	}
}
