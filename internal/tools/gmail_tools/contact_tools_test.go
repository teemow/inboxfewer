package gmail_tools

import (
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
