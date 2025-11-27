package common

import (
	"context"
	"testing"

	"github.com/teemow/inboxfewer/internal/mcp/oauth"
)

func TestGetAccountFromArgs(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name     string
		args     map[string]interface{}
		expected string
	}{
		{
			name:     "no account specified returns default",
			args:     map[string]interface{}{},
			expected: "default",
		},
		{
			name: "account specified returns account",
			args: map[string]interface{}{
				"account": "work",
			},
			expected: "work",
		},
		{
			name: "empty account returns default",
			args: map[string]interface{}{
				"account": "",
			},
			expected: "default",
		},
		{
			name: "account with other params",
			args: map[string]interface{}{
				"account": "personal",
				"other":   "value",
			},
			expected: "personal",
		},
		{
			name:     "nil args returns default",
			args:     nil,
			expected: "default",
		},
		{
			name: "non-string account type returns default",
			args: map[string]interface{}{
				"account": 123,
			},
			expected: "default",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetAccountFromArgs(ctx, tt.args)
			if result != tt.expected {
				t.Errorf("GetAccountFromArgs() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestGetAccountFromArgs_WithOAuthContext(t *testing.T) {
	userInfo := &oauth.UserInfo{
		ID:            "user-123",
		Email:         "oauth-user@example.com",
		EmailVerified: true,
		Name:          "OAuth User",
	}
	ctx := oauth.ContextWithUserInfo(context.Background(), userInfo)

	tests := []struct {
		name     string
		args     map[string]interface{}
		expected string
	}{
		{
			name:     "OAuth context takes precedence over default",
			args:     map[string]interface{}{},
			expected: "oauth-user@example.com",
		},
		{
			name: "OAuth context takes precedence over explicit account",
			args: map[string]interface{}{
				"account": "explicit-account",
			},
			expected: "oauth-user@example.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetAccountFromArgs(ctx, tt.args)
			if result != tt.expected {
				t.Errorf("GetAccountFromArgs() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestGetAccountFromArgs_WithEmptyOAuthEmail(t *testing.T) {
	userInfo := &oauth.UserInfo{
		ID:    "user-456",
		Email: "",
	}
	ctx := oauth.ContextWithUserInfo(context.Background(), userInfo)

	tests := []struct {
		name     string
		args     map[string]interface{}
		expected string
	}{
		{
			name:     "empty OAuth email falls back to default",
			args:     map[string]interface{}{},
			expected: "default",
		},
		{
			name: "empty OAuth email falls back to explicit account",
			args: map[string]interface{}{
				"account": "fallback-account",
			},
			expected: "fallback-account",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetAccountFromArgs(ctx, tt.args)
			if result != tt.expected {
				t.Errorf("GetAccountFromArgs() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestGetAccountFromArgs_WithNilOAuthUserInfo(t *testing.T) {
	// Context with nil user info should fall back to args
	ctx := oauth.ContextWithUserInfo(context.Background(), nil)

	args := map[string]interface{}{
		"account": "fallback-account",
	}

	result := GetAccountFromArgs(ctx, args)
	if result != "fallback-account" {
		t.Errorf("Expected 'fallback-account' when OAuth user is nil, got %s", result)
	}
}
