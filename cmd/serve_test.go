package cmd

import (
	"os"
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
			name:     "only commas and spaces returns nil",
			input:    ",  , , ",
			expected: nil,
		},
		{
			name:     "whitespace only returns nil",
			input:    "   ",
			expected: nil,
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

// TestOAuthTrustedAudiencesEnvVar tests that OAUTH_TRUSTED_AUDIENCES is correctly
// parsed and applied when the --oauth-trusted-audiences flag is not set.
func TestOAuthTrustedAudiencesEnvVar(t *testing.T) {
	tests := []struct {
		name          string
		envValue      string
		flagValue     []string
		expectedValue []string
	}{
		{
			name:          "env var sets trusted audiences",
			envValue:      "muster-client,another-aggregator",
			flagValue:     nil,
			expectedValue: []string{"muster-client", "another-aggregator"},
		},
		{
			name:          "env var with whitespace is trimmed",
			envValue:      " muster-client , another-aggregator ",
			flagValue:     nil,
			expectedValue: []string{"muster-client", "another-aggregator"},
		},
		{
			name:          "flag overrides env var",
			envValue:      "env-client",
			flagValue:     []string{"flag-client"},
			expectedValue: []string{"flag-client"},
		},
		{
			name:          "empty env var returns nil",
			envValue:      "",
			flagValue:     nil,
			expectedValue: nil,
		},
		{
			name:          "only commas in env var returns nil",
			envValue:      ",,,",
			flagValue:     nil,
			expectedValue: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set environment variable
			if tt.envValue != "" {
				t.Setenv("OAUTH_TRUSTED_AUDIENCES", tt.envValue)
			}

			// Simulate the logic from runServe
			var result []string
			if len(tt.flagValue) > 0 {
				result = tt.flagValue
			} else if envVal := os.Getenv("OAUTH_TRUSTED_AUDIENCES"); envVal != "" {
				result = parseCommaSeparatedList(envVal)
			}

			// Compare results
			if tt.expectedValue == nil {
				if result != nil {
					t.Errorf("expected nil, got %v", result)
				}
				return
			}

			if len(result) != len(tt.expectedValue) {
				t.Errorf("expected %v (len %d), got %v (len %d)",
					tt.expectedValue, len(tt.expectedValue), result, len(result))
				return
			}

			for i, v := range result {
				if v != tt.expectedValue[i] {
					t.Errorf("expected[%d] = %q, got %q", i, tt.expectedValue[i], v)
				}
			}
		})
	}
}
