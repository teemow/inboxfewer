package oauth

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHandler_ValidateGoogleToken_MissingHeader(t *testing.T) {
	handler, _ := NewHandler(&Config{
		Resource: "https://mcp.example.com",
	})

	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("Handler should not be called")
	})

	wrappedHandler := handler.ValidateGoogleToken(testHandler)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()

	wrappedHandler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("ValidateGoogleToken() status = %d, want %d", w.Code, http.StatusUnauthorized)
	}

	// Check WWW-Authenticate header
	wwwAuth := w.Header().Get("WWW-Authenticate")
	if wwwAuth == "" {
		t.Error("ValidateGoogleToken() should set WWW-Authenticate header")
	}
}

func TestHandler_ValidateGoogleToken_InvalidFormat(t *testing.T) {
	handler, _ := NewHandler(&Config{
		Resource: "https://mcp.example.com",
	})

	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("Handler should not be called")
	})

	wrappedHandler := handler.ValidateGoogleToken(testHandler)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "InvalidFormat")
	w := httptest.NewRecorder()

	wrappedHandler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("ValidateGoogleToken() status = %d, want %d", w.Code, http.StatusUnauthorized)
	}

	// Check WWW-Authenticate header
	wwwAuth := w.Header().Get("WWW-Authenticate")
	if wwwAuth == "" {
		t.Error("ValidateGoogleToken() should set WWW-Authenticate header")
	}
}

func TestHandler_ValidateGoogleToken_InvalidToken(t *testing.T) {
	handler, _ := NewHandler(&Config{
		Resource: "https://mcp.example.com",
	})

	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("Handler should not be called")
	})

	wrappedHandler := handler.ValidateGoogleToken(testHandler)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Bearer invalid-token")
	w := httptest.NewRecorder()

	wrappedHandler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("ValidateGoogleToken() status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestHandler_OptionalGoogleToken_NoToken(t *testing.T) {
	handler, err := NewHandler(&Config{
		Resource: "https://mcp.example.com",
	})
	if err != nil {
		t.Fatalf("NewHandler() error = %v", err)
	}

	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	wrappedHandler := handler.OptionalGoogleToken(testHandler)

	// Test without token (should succeed)
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()

	wrappedHandler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("OptionalGoogleToken() without token status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestHandler_ValidateGoogleTokenFunc(t *testing.T) {
	handler, _ := NewHandler(&Config{
		Resource: "https://mcp.example.com",
	})

	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("Handler should not be called without valid token")
	})

	wrappedHandler := handler.ValidateGoogleTokenFunc(testHandler)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Bearer invalid-token")
	w := httptest.NewRecorder()

	wrappedHandler(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("ValidateGoogleTokenFunc() status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestGetUserFromContext(t *testing.T) {
	// Test with empty context (should not panic)
	ctx := context.Background()
	user, ok := GetUserFromContext(ctx)
	if ok {
		t.Error("GetUserFromContext() should return false for empty context")
	}
	if user != nil {
		t.Error("GetUserFromContext() should return nil user for empty context")
	}
}

func TestGetGoogleTokenFromContext(t *testing.T) {
	// Test with empty context (should not panic)
	ctx := context.Background()
	token, ok := GetGoogleTokenFromContext(ctx)
	if ok {
		t.Error("GetGoogleTokenFromContext() should return false for empty context")
	}
	if token != nil {
		t.Error("GetGoogleTokenFromContext() should return nil token for empty context")
	}
}
