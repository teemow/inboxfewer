package tasks_tools

import (
	"context"
	"testing"

	"github.com/teemow/inboxfewer/internal/mcp/oauth"
	"github.com/teemow/inboxfewer/internal/tools/common"
)

// TestCommonGetAccountFromArgs verifies that the tasks_tools package
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

func TestParseAttendees(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "empty string",
			input:    "",
			expected: []string{},
		},
		{
			name:     "single email",
			input:    "test@example.com",
			expected: []string{"test@example.com"},
		},
		{
			name:     "multiple emails",
			input:    "test@example.com,another@example.com",
			expected: []string{"test@example.com", "another@example.com"},
		},
		{
			name:     "emails with spaces",
			input:    "test@example.com, another@example.com , third@example.com",
			expected: []string{"test@example.com", "another@example.com", "third@example.com"},
		},
		{
			name:     "trailing comma",
			input:    "test@example.com,",
			expected: []string{"test@example.com"},
		},
		{
			name:     "leading comma",
			input:    ",test@example.com",
			expected: []string{"test@example.com"},
		},
		{
			name:     "multiple commas",
			input:    "test@example.com,,another@example.com",
			expected: []string{"test@example.com", "another@example.com"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseAttendees(tt.input)
			if len(result) != len(tt.expected) {
				t.Errorf("parseAttendees(%q) returned %d items, expected %d", tt.input, len(result), len(tt.expected))
				return
			}
			for i, v := range result {
				if v != tt.expected[i] {
					t.Errorf("parseAttendees(%q)[%d] = %q, expected %q", tt.input, i, v, tt.expected[i])
				}
			}
		})
	}
}

func TestRegisterTasksTools(t *testing.T) {
	// This test verifies that the tools package can be imported and used
	// Actual tool registration requires a full MCP server setup
}

func TestRegisterTaskListTools(t *testing.T) {
	// This test verifies that task list tools can be imported
}

func TestRegisterTaskTools(t *testing.T) {
	// This test verifies that task tools can be imported
}
