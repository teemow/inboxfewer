package meet_tools

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
			name:     "no account specified",
			args:     map[string]interface{}{},
			expected: "default",
		},
		{
			name: "account specified",
			args: map[string]interface{}{
				"account": "work",
			},
			expected: "work",
		},
		{
			name: "empty account",
			args: map[string]interface{}{
				"account": "",
			},
			expected: "default",
		},
		{
			name: "account with other params",
			args: map[string]interface{}{
				"account":           "personal",
				"conference_record": "spaces/test/conferenceRecords/conf123",
			},
			expected: "personal",
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

func TestGetAccountFromArgsWithInvalidType(t *testing.T) {
	args := map[string]interface{}{
		"account": 123, // Invalid type
	}
	result := getAccountFromArgs(args)
	if result != "default" {
		t.Errorf("Expected 'default' for invalid type, got %s", result)
	}
}
