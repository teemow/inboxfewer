package server

import (
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
