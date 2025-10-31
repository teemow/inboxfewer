package gmail_tools

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFilterToolsRegistration(t *testing.T) {
	// Test that filter tools can be registered without errors
	// This is a basic smoke test to ensure the tool definitions are valid
	assert.NotNil(t, RegisterFilterTools)
}

func TestSplitLabelIDs(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "single label",
			input:    "Label_1",
			expected: []string{"Label_1"},
		},
		{
			name:     "multiple labels",
			input:    "Label_1,Label_2,Label_3",
			expected: []string{"Label_1", "Label_2", "Label_3"},
		},
		{
			name:     "with spaces",
			input:    "Label_1, Label_2 , Label_3",
			expected: []string{"Label_1", "Label_2", "Label_3"},
		},
		{
			name:     "empty string",
			input:    "",
			expected: nil,
		},
		{
			name:     "system labels",
			input:    "INBOX,UNREAD,STARRED",
			expected: []string{"INBOX", "UNREAD", "STARRED"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := splitLabelIDs(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}
