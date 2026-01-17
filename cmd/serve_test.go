package cmd

import (
	"testing"
)

func TestParseCommaSeparatedList(t *testing.T) {
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
			name:     "single value",
			input:    "muster-client",
			expected: []string{"muster-client"},
		},
		{
			name:     "multiple values",
			input:    "muster-client,other-client",
			expected: []string{"muster-client", "other-client"},
		},
		{
			name:     "values with spaces around comma",
			input:    "muster-client, other-client",
			expected: []string{"muster-client", "other-client"},
		},
		{
			name:     "values with leading/trailing spaces",
			input:    "  muster-client  ,  other-client  ",
			expected: []string{"muster-client", "other-client"},
		},
		{
			name:     "trailing comma",
			input:    "muster-client,other-client,",
			expected: []string{"muster-client", "other-client"},
		},
		{
			name:     "leading comma",
			input:    ",muster-client,other-client",
			expected: []string{"muster-client", "other-client"},
		},
		{
			name:     "multiple consecutive commas",
			input:    "muster-client,,other-client",
			expected: []string{"muster-client", "other-client"},
		},
		{
			name:     "only commas and spaces",
			input:    ",  , , ",
			expected: []string{},
		},
		{
			name:     "single value with surrounding whitespace",
			input:    "  muster-client  ",
			expected: []string{"muster-client"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseCommaSeparatedList(tt.input)

			// Handle nil vs empty slice comparison
			if tt.expected == nil {
				if result != nil {
					t.Errorf("parseCommaSeparatedList(%q) = %v, want nil", tt.input, result)
				}
				return
			}

			if len(result) != len(tt.expected) {
				t.Errorf("parseCommaSeparatedList(%q) = %v (len %d), want %v (len %d)",
					tt.input, result, len(result), tt.expected, len(tt.expected))
				return
			}

			for i, v := range result {
				if v != tt.expected[i] {
					t.Errorf("parseCommaSeparatedList(%q)[%d] = %q, want %q",
						tt.input, i, v, tt.expected[i])
				}
			}
		})
	}
}
