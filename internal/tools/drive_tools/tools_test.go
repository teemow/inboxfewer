package drive_tools

import (
	"testing"
)

func TestGetAccountFromArgs(t *testing.T) {
	tests := []struct {
		name     string
		args     map[string]interface{}
		expected string
	}{
		{
			name:     "default account when no account specified",
			args:     map[string]interface{}{},
			expected: "default",
		},
		{
			name: "default account when account is empty",
			args: map[string]interface{}{
				"account": "",
			},
			expected: "default",
		},
		{
			name: "specific account when provided",
			args: map[string]interface{}{
				"account": "work",
			},
			expected: "work",
		},
		{
			name: "default when account is not a string",
			args: map[string]interface{}{
				"account": 123,
			},
			expected: "default",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getAccountFromArgs(tt.args)
			if result != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestParseCommaList(t *testing.T) {
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
			name:     "single item",
			input:    "item1",
			expected: []string{"item1"},
		},
		{
			name:     "multiple items",
			input:    "item1,item2,item3",
			expected: []string{"item1", "item2", "item3"},
		},
		{
			name:     "items with spaces",
			input:    "item1, item2 , item3",
			expected: []string{"item1", "item2", "item3"},
		},
		{
			name:     "items with empty values",
			input:    "item1,,item3",
			expected: []string{"item1", "item3"},
		},
		{
			name:     "only commas",
			input:    ",,,",
			expected: nil,
		},
		{
			name:     "trailing comma",
			input:    "item1,item2,",
			expected: []string{"item1", "item2"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseCommaList(tt.input)
			if len(result) != len(tt.expected) {
				t.Errorf("Expected length %d, got %d", len(tt.expected), len(result))
				return
			}
			for i, v := range result {
				if v != tt.expected[i] {
					t.Errorf("At index %d: expected %s, got %s", i, tt.expected[i], v)
				}
			}
		})
	}
}
