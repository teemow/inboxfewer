package gmail_tools

import (
	"strings"
	"testing"
)

func TestSplitEmailAddresses(t *testing.T) {
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
			input:    "user1@example.com,user2@example.com",
			expected: []string{"user1@example.com", "user2@example.com"},
		},
		{
			name:     "emails with spaces",
			input:    "user1@example.com, user2@example.com, user3@example.com",
			expected: []string{"user1@example.com", "user2@example.com", "user3@example.com"},
		},
		{
			name:     "trailing comma",
			input:    "user1@example.com,",
			expected: []string{"user1@example.com"},
		},
		{
			name:     "multiple commas",
			input:    "user1@example.com,,user2@example.com",
			expected: []string{"user1@example.com", "user2@example.com"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := splitEmailAddresses(tt.input)
			if len(result) != len(tt.expected) {
				t.Errorf("splitEmailAddresses() length = %d, want %d", len(result), len(tt.expected))
				return
			}
			for i := range result {
				if result[i] != tt.expected[i] {
					t.Errorf("splitEmailAddresses()[%d] = %s, want %s", i, result[i], tt.expected[i])
				}
			}
		})
	}
}

func TestHandleSearchContacts_MissingQuery(t *testing.T) {
	// Create a mock request with missing query
	args := map[string]interface{}{}

	// Simulate the request
	request := &mockCallToolRequest{args: args}

	result, err := handleSearchContacts(nil, request, nil)

	if err != nil {
		t.Errorf("handleSearchContacts() error = %v, want nil", err)
	}

	if result == nil {
		t.Fatal("handleSearchContacts() result is nil")
	}

	// Check that it returns an error result
	if !result.IsError {
		t.Error("handleSearchContacts() should return error for missing query")
	}

	if !strings.Contains(result.Content[0].Text, "query is required") {
		t.Errorf("handleSearchContacts() error message = %s, want 'query is required'", result.Content[0].Text)
	}
}

func TestHandleSearchContacts_EmptyQuery(t *testing.T) {
	// Create a mock request with empty query
	args := map[string]interface{}{
		"query": "",
	}

	request := &mockCallToolRequest{args: args}

	result, err := handleSearchContacts(nil, request, nil)

	if err != nil {
		t.Errorf("handleSearchContacts() error = %v, want nil", err)
	}

	if result == nil {
		t.Fatal("handleSearchContacts() result is nil")
	}

	// Check that it returns an error result
	if !result.IsError {
		t.Error("handleSearchContacts() should return error for empty query")
	}

	if !strings.Contains(result.Content[0].Text, "query is required") {
		t.Errorf("handleSearchContacts() error message = %s, want 'query is required'", result.Content[0].Text)
	}
}

// mockCallToolRequest is a mock implementation of mcp.CallToolRequest
type mockCallToolRequest struct {
	args map[string]interface{}
}

func (m *mockCallToolRequest) GetArguments() map[string]interface{} {
	return m.args
}
