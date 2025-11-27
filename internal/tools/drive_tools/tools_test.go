package drive_tools

import (
	"context"
	"testing"

	"github.com/teemow/inboxfewer/internal/mcp/oauth"
	"github.com/teemow/inboxfewer/internal/tools/common"
)

// TestCommonGetAccountFromArgs verifies that the drive_tools package
// correctly uses the shared common.GetAccountFromArgs function.
// Comprehensive tests for GetAccountFromArgs are in internal/tools/common/account_test.go
func TestCommonGetAccountFromArgs(t *testing.T) {
	ctx := context.Background()

	// Test basic functionality
	args := map[string]interface{}{
		"account": "test-account",
	}
	result := common.GetAccountFromArgs(ctx, args)
	if result != "test-account" {
		t.Errorf("GetAccountFromArgs() = %v, expected test-account", result)
	}

	// Test OAuth integration
	userInfo := &oauth.UserInfo{
		Email: "oauth-user@example.com",
	}
	ctxWithUser := oauth.ContextWithUserInfo(context.Background(), userInfo)
	result = common.GetAccountFromArgs(ctxWithUser, args)
	if result != "oauth-user@example.com" {
		t.Errorf("GetAccountFromArgs() with OAuth = %v, expected oauth-user@example.com", result)
	}
}

func TestParseCommaList(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "single value",
			input:    "test@example.com",
			expected: []string{"test@example.com"},
		},
		{
			name:     "multiple values",
			input:    "test@example.com,another@example.com",
			expected: []string{"test@example.com", "another@example.com"},
		},
		{
			name:     "values with spaces",
			input:    "test@example.com, another@example.com , third@example.com",
			expected: []string{"test@example.com", "another@example.com", "third@example.com"},
		},
		{
			name:     "empty string",
			input:    "",
			expected: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseCommaList(tt.input)
			if len(result) != len(tt.expected) {
				t.Errorf("Expected %d items, got %d", len(tt.expected), len(result))
				return
			}
			for i, v := range result {
				if v != tt.expected[i] {
					t.Errorf("Item %d: expected %s, got %s", i, tt.expected[i], v)
				}
			}
		})
	}
}
