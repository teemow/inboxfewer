package oauth

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestHandler_ValidateToken(t *testing.T) {
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

	// Create a valid token
	token := &Token{
		AccessToken:  "valid-token",
		TokenType:    "Bearer",
		ExpiresIn:    3600,
		RefreshToken: "refresh-token",
		Scope:        "mcp",
		IssuedAt:     time.Now(),
		ClientID:     "test-client",
		Resource:     "https://mcp.example.com",
	}
	if err := handler.store.SaveToken(token); err != nil {
		t.Fatalf("SaveToken() error = %v", err)
	}

	// Create a test handler that checks context
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token, ok := GetTokenFromContext(r.Context())
		if !ok {
			t.Error("Token should be in context")
		}
		if token.AccessToken != "valid-token" {
			t.Errorf("AccessToken = %s, want valid-token", token.AccessToken)
		}

		client, ok := GetClientFromContext(r.Context())
		if !ok {
			t.Error("Client should be in context")
		}
		if client.ClientID != "test-client" {
			t.Errorf("ClientID = %s, want test-client", client.ClientID)
		}

		w.WriteHeader(http.StatusOK)
	})

	// Wrap with ValidateToken middleware
	wrappedHandler := handler.ValidateToken(testHandler)

	// Test with valid token
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Bearer valid-token")
	w := httptest.NewRecorder()

	wrappedHandler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("ValidateToken() status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}
}

func TestHandler_ValidateToken_MissingHeader(t *testing.T) {
	handler, _ := NewHandler(&Config{
		Issuer:   "https://example.com",
		Resource: "https://mcp.example.com",
	})

	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("Handler should not be called")
	})

	wrappedHandler := handler.ValidateToken(testHandler)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()

	wrappedHandler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("ValidateToken() status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestHandler_ValidateToken_InvalidFormat(t *testing.T) {
	handler, _ := NewHandler(&Config{
		Issuer:   "https://example.com",
		Resource: "https://mcp.example.com",
	})

	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("Handler should not be called")
	})

	wrappedHandler := handler.ValidateToken(testHandler)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "InvalidFormat")
	w := httptest.NewRecorder()

	wrappedHandler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("ValidateToken() status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestHandler_ValidateToken_InvalidToken(t *testing.T) {
	handler, _ := NewHandler(&Config{
		Issuer:   "https://example.com",
		Resource: "https://mcp.example.com",
	})

	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("Handler should not be called")
	})

	wrappedHandler := handler.ValidateToken(testHandler)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Bearer invalid-token")
	w := httptest.NewRecorder()

	wrappedHandler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("ValidateToken() status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestHandler_ValidateToken_WrongResource(t *testing.T) {
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
	}
	if err := handler.store.SaveClient(client); err != nil {
		t.Fatalf("SaveClient() error = %v", err)
	}

	// Create a token with wrong resource
	token := &Token{
		AccessToken: "valid-token",
		TokenType:   "Bearer",
		ExpiresIn:   3600,
		IssuedAt:    time.Now(),
		ClientID:    "test-client",
		Resource:    "https://different.example.com",
	}
	if err := handler.store.SaveToken(token); err != nil {
		t.Fatalf("SaveToken() error = %v", err)
	}

	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("Handler should not be called")
	})

	wrappedHandler := handler.ValidateToken(testHandler)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Bearer valid-token")
	w := httptest.NewRecorder()

	wrappedHandler.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("ValidateToken() status = %d, want %d", w.Code, http.StatusForbidden)
	}
}

func TestHandler_RequireScope(t *testing.T) {
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
	}
	if err := handler.store.SaveClient(client); err != nil {
		t.Fatalf("SaveClient() error = %v", err)
	}

	// Create a token with specific scopes
	token := &Token{
		AccessToken: "valid-token",
		TokenType:   "Bearer",
		ExpiresIn:   3600,
		Scope:       "mcp admin",
		IssuedAt:    time.Now(),
		ClientID:    "test-client",
		Resource:    "https://mcp.example.com",
	}
	if err := handler.store.SaveToken(token); err != nil {
		t.Fatalf("SaveToken() error = %v", err)
	}

	// Test with required scope present
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	wrappedHandler := handler.ValidateToken(handler.RequireScope("mcp")(testHandler))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Bearer valid-token")
	w := httptest.NewRecorder()

	wrappedHandler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("RequireScope() with valid scope status = %d, want %d", w.Code, http.StatusOK)
	}

	// Test with required scope missing
	wrappedHandler2 := handler.ValidateToken(handler.RequireScope("write")(testHandler))

	req2 := httptest.NewRequest(http.MethodGet, "/test", nil)
	req2.Header.Set("Authorization", "Bearer valid-token")
	w2 := httptest.NewRecorder()

	wrappedHandler2.ServeHTTP(w2, req2)

	if w2.Code != http.StatusForbidden {
		t.Errorf("RequireScope() with missing scope status = %d, want %d", w2.Code, http.StatusForbidden)
	}
}

func TestHandler_OptionalToken(t *testing.T) {
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
	}
	if err := handler.store.SaveClient(client); err != nil {
		t.Fatalf("SaveClient() error = %v", err)
	}

	// Create a valid token
	token := &Token{
		AccessToken: "valid-token",
		TokenType:   "Bearer",
		ExpiresIn:   3600,
		IssuedAt:    time.Now(),
		ClientID:    "test-client",
		Resource:    "https://mcp.example.com",
	}
	if err := handler.store.SaveToken(token); err != nil {
		t.Fatalf("SaveToken() error = %v", err)
	}

	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	wrappedHandler := handler.OptionalToken(testHandler)

	// Test without token (should succeed)
	req1 := httptest.NewRequest(http.MethodGet, "/test", nil)
	w1 := httptest.NewRecorder()

	wrappedHandler.ServeHTTP(w1, req1)

	if w1.Code != http.StatusOK {
		t.Errorf("OptionalToken() without token status = %d, want %d", w1.Code, http.StatusOK)
	}

	// Test with valid token (should succeed)
	req2 := httptest.NewRequest(http.MethodGet, "/test", nil)
	req2.Header.Set("Authorization", "Bearer valid-token")
	w2 := httptest.NewRecorder()

	wrappedHandler.ServeHTTP(w2, req2)

	if w2.Code != http.StatusOK {
		t.Errorf("OptionalToken() with valid token status = %d, want %d", w2.Code, http.StatusOK)
	}

	// Test with invalid token (should fail)
	req3 := httptest.NewRequest(http.MethodGet, "/test", nil)
	req3.Header.Set("Authorization", "Bearer invalid-token")
	w3 := httptest.NewRecorder()

	wrappedHandler.ServeHTTP(w3, req3)

	if w3.Code != http.StatusUnauthorized {
		t.Errorf("OptionalToken() with invalid token status = %d, want %d", w3.Code, http.StatusUnauthorized)
	}
}

func TestGetTokenFromContext(t *testing.T) {
	handler, _ := NewHandler(&Config{
		Issuer:   "https://example.com",
		Resource: "https://mcp.example.com",
	})

	// Register a client
	client := &ClientInfo{
		ClientID: "test-client",
	}
	handler.store.SaveClient(client)

	// Create a valid token
	token := &Token{
		AccessToken: "valid-token",
		TokenType:   "Bearer",
		ExpiresIn:   3600,
		IssuedAt:    time.Now(),
		ClientID:    "test-client",
		Resource:    "https://mcp.example.com",
	}
	handler.store.SaveToken(token)

	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token, ok := GetTokenFromContext(r.Context())
		if !ok {
			t.Error("GetTokenFromContext() should return true")
		}
		if token == nil {
			t.Error("GetTokenFromContext() should return non-nil token")
		}
		if token.AccessToken != "valid-token" {
			t.Errorf("token.AccessToken = %s, want valid-token", token.AccessToken)
		}
	})

	wrappedHandler := handler.ValidateToken(testHandler)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Bearer valid-token")
	w := httptest.NewRecorder()

	wrappedHandler.ServeHTTP(w, req)
}

func TestGetClientFromContext(t *testing.T) {
	handler, _ := NewHandler(&Config{
		Issuer:   "https://example.com",
		Resource: "https://mcp.example.com",
	})

	// Register a client
	client := &ClientInfo{
		ClientID: "test-client",
	}
	handler.store.SaveClient(client)

	// Create a valid token
	token := &Token{
		AccessToken: "valid-token",
		TokenType:   "Bearer",
		ExpiresIn:   3600,
		IssuedAt:    time.Now(),
		ClientID:    "test-client",
		Resource:    "https://mcp.example.com",
	}
	handler.store.SaveToken(token)

	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		client, ok := GetClientFromContext(r.Context())
		if !ok {
			t.Error("GetClientFromContext() should return true")
		}
		if client == nil {
			t.Error("GetClientFromContext() should return non-nil client")
		}
		if client.ClientID != "test-client" {
			t.Errorf("client.ClientID = %s, want test-client", client.ClientID)
		}
	})

	wrappedHandler := handler.ValidateToken(testHandler)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Bearer valid-token")
	w := httptest.NewRecorder()

	wrappedHandler.ServeHTTP(w, req)
}

func TestHandler_ValidateTokenFunc(t *testing.T) {
	handler, _ := NewHandler(&Config{
		Issuer:   "https://example.com",
		Resource: "https://mcp.example.com",
	})

	// Register a client
	client := &ClientInfo{
		ClientID: "test-client",
	}
	handler.store.SaveClient(client)

	// Create a valid token
	token := &Token{
		AccessToken: "valid-token",
		TokenType:   "Bearer",
		ExpiresIn:   3600,
		IssuedAt:    time.Now(),
		ClientID:    "test-client",
		Resource:    "https://mcp.example.com",
	}
	handler.store.SaveToken(token)

	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	wrappedHandler := handler.ValidateTokenFunc(testHandler)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Bearer valid-token")
	w := httptest.NewRecorder()

	wrappedHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("ValidateTokenFunc() status = %d, want %d", w.Code, http.StatusOK)
	}
}
