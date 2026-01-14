package server

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestValidateHTTPSRequirement(t *testing.T) {
	tests := []struct {
		name    string
		baseURL string
		wantErr bool
	}{
		{
			name:    "valid HTTPS URL",
			baseURL: "https://mcp.example.com",
			wantErr: false,
		},
		{
			name:    "valid HTTP localhost",
			baseURL: "http://localhost:8080",
			wantErr: false,
		},
		{
			name:    "valid HTTP 127.0.0.1",
			baseURL: "http://127.0.0.1:8080",
			wantErr: false,
		},
		{
			name:    "valid HTTP ::1 (IPv6 loopback)",
			baseURL: "http://[::1]:8080",
			wantErr: false,
		},
		{
			name:    "invalid HTTP non-localhost",
			baseURL: "http://mcp.example.com",
			wantErr: true,
		},
		{
			name:    "invalid HTTP with localhost substring",
			baseURL: "http://localhost.example.com",
			wantErr: true,
		},
		{
			name:    "invalid HTTP with 127.0.0.1 in domain",
			baseURL: "http://127.0.0.1.example.com",
			wantErr: true,
		},
		{
			name:    "empty URL",
			baseURL: "",
			wantErr: true,
		},
		{
			name:    "invalid URL format",
			baseURL: "not a url",
			wantErr: true,
		},
		{
			name:    "invalid scheme",
			baseURL: "ftp://example.com",
			wantErr: true,
		},
		{
			name:    "HTTPS with path",
			baseURL: "https://mcp.example.com/api",
			wantErr: false,
		},
		{
			name:    "HTTPS with port",
			baseURL: "https://mcp.example.com:8443",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateHTTPSRequirement(tt.baseURL)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateHTTPSRequirement() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestResponseWriter(t *testing.T) {
	t.Run("captures status code", func(t *testing.T) {
		recorder := httptest.NewRecorder()
		rw := newResponseWriter(recorder)

		rw.WriteHeader(http.StatusNotFound)

		if rw.statusCode != http.StatusNotFound {
			t.Errorf("statusCode = %d, want %d", rw.statusCode, http.StatusNotFound)
		}
	})

	t.Run("defaults to 200", func(t *testing.T) {
		recorder := httptest.NewRecorder()
		rw := newResponseWriter(recorder)

		// Don't call WriteHeader, check default
		if rw.statusCode != http.StatusOK {
			t.Errorf("statusCode = %d, want %d", rw.statusCode, http.StatusOK)
		}
	})

	t.Run("passes write header to underlying writer", func(t *testing.T) {
		recorder := httptest.NewRecorder()
		rw := newResponseWriter(recorder)

		rw.WriteHeader(http.StatusCreated)

		if recorder.Code != http.StatusCreated {
			t.Errorf("recorder.Code = %d, want %d", recorder.Code, http.StatusCreated)
		}
	})
}

func TestInstrumentationMiddleware(t *testing.T) {
	t.Run("calls next handler when no metrics", func(t *testing.T) {
		server := &OAuthHTTPServer{} // No metrics set
		called := false
		next := http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
			called = true
		})

		handler := server.instrumentationMiddleware(next)
		req := httptest.NewRequest("GET", "/test", nil)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if !called {
			t.Error("expected next handler to be called")
		}
	})
}

func TestOAuthInstrumentationWrapper(t *testing.T) {
	t.Run("calls next handler when no metrics", func(t *testing.T) {
		server := &OAuthHTTPServer{} // No metrics set
		called := false
		next := http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
			called = true
		})

		handler := server.oauthInstrumentationWrapper(next)
		req := httptest.NewRequest("GET", "/mcp", nil)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if !called {
			t.Error("expected next handler to be called")
		}
	})
}
