package calendar_tools

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
			name:     "no account provided",
			args:     map[string]interface{}{},
			expected: "default",
		},
		{
			name: "account provided",
			args: map[string]interface{}{
				"account": "test-account",
			},
			expected: "test-account",
		},
		{
			name: "empty account string",
			args: map[string]interface{}{
				"account": "",
			},
			expected: "default",
		},
		{
			name: "account with other args",
			args: map[string]interface{}{
				"account":    "work-account",
				"calendarId": "primary",
			},
			expected: "work-account",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getAccountFromArgs(tt.args)
			if result != tt.expected {
				t.Errorf("getAccountFromArgs() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestGetAccountFromArgs_NonStringType(t *testing.T) {
	// Test with non-string account value
	args := map[string]interface{}{
		"account": 123, // wrong type
	}

	result := getAccountFromArgs(args)
	if result != "default" {
		t.Errorf("Expected 'default' for non-string account, got %s", result)
	}
}
