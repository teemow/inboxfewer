package gmail_tools

import (
	"testing"
)

func TestGetAccountFromArgs(t *testing.T) {
	tests := []struct {
		name string
		args map[string]interface{}
		want string
	}{
		{
			name: "with account specified",
			args: map[string]interface{}{
				"account": "work",
			},
			want: "work",
		},
		{
			name: "without account specified",
			args: map[string]interface{}{},
			want: "default",
		},
		{
			name: "with empty account string",
			args: map[string]interface{}{
				"account": "",
			},
			want: "default",
		},
		{
			name: "with non-string account",
			args: map[string]interface{}{
				"account": 123,
			},
			want: "default",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getAccountFromArgs(tt.args)
			if got != tt.want {
				t.Errorf("getAccountFromArgs() = %v, want %v", got, tt.want)
			}
		})
	}
}

// Test email tools package
func TestEmailToolsPackage(t *testing.T) {
	// Test that the package compiles and basic functionality works
	result := splitEmailAddresses("test@example.com")
	if len(result) != 1 || result[0] != "test@example.com" {
		t.Error("splitEmailAddresses not working correctly")
	}
}
