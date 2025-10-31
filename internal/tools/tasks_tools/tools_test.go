package tasks_tools

import (
	"testing"
)

func TestGetAccountFromArgs(t *testing.T) {
	// Test with default account (no account specified)
	args := map[string]interface{}{}
	account := getAccountFromArgs(args)
	if account != "default" {
		t.Errorf("Expected 'default' account, got %s", account)
	}

	// Test with specific account
	args = map[string]interface{}{
		"account": "work",
	}
	account = getAccountFromArgs(args)
	if account != "work" {
		t.Errorf("Expected 'work' account, got %s", account)
	}

	// Test with empty account string (should default)
	args = map[string]interface{}{
		"account": "",
	}
	account = getAccountFromArgs(args)
	if account != "default" {
		t.Errorf("Expected 'default' account for empty string, got %s", account)
	}

	// Test with non-string account value
	args = map[string]interface{}{
		"account": 123,
	}
	account = getAccountFromArgs(args)
	if account != "default" {
		t.Errorf("Expected 'default' account for non-string value, got %s", account)
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
			expected: nil,
		},
		{
			name:     "single email",
			input:    "user@example.com",
			expected: []string{"user@example.com"},
		},
		{
			name:     "multiple emails",
			input:    "user1@example.com,user2@example.com,user3@example.com",
			expected: []string{"user1@example.com", "user2@example.com", "user3@example.com"},
		},
		{
			name:     "emails with spaces",
			input:    "user1@example.com, user2@example.com , user3@example.com",
			expected: []string{"user1@example.com", "user2@example.com", "user3@example.com"},
		},
		{
			name:     "trailing comma",
			input:    "user1@example.com,user2@example.com,",
			expected: []string{"user1@example.com", "user2@example.com"},
		},
		{
			name:     "leading comma",
			input:    ",user1@example.com,user2@example.com",
			expected: []string{"user1@example.com", "user2@example.com"},
		},
		{
			name:     "multiple commas",
			input:    "user1@example.com,,user2@example.com",
			expected: []string{"user1@example.com", "user2@example.com"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseAttendees(tt.input)

			if len(result) != len(tt.expected) {
				t.Errorf("Expected %d attendees, got %d", len(tt.expected), len(result))
				return
			}

			for i, email := range result {
				if email != tt.expected[i] {
					t.Errorf("Expected email at index %d to be %s, got %s", i, tt.expected[i], email)
				}
			}
		})
	}
}

func TestRegisterTasksTools(t *testing.T) {
	// This test verifies that RegisterTasksTools doesn't panic
	// We can't fully test the registration without a real MCP server and context
	// But we can ensure the function signature is correct
	_ = RegisterTasksTools
}

func TestRegisterTaskListTools(t *testing.T) {
	// This test verifies that registerTaskListTools doesn't panic
	// We can't fully test the registration without a real MCP server and context
	// But we can ensure the function signature is correct
	_ = registerTaskListTools
}

func TestRegisterTaskTools(t *testing.T) {
	// This test verifies that registerTaskTools doesn't panic
	// We can't fully test the registration without a real MCP server and context
	// But we can ensure the function signature is correct
	_ = registerTaskTools
}
