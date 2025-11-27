package gmail_tools

import (
	"context"
	"testing"

	"github.com/teemow/inboxfewer/internal/mcp/oauth"
	"github.com/teemow/inboxfewer/internal/tools/common"
)

// TestEmailToolsGetAccountFromArgs verifies that email_tools correctly uses the shared
// common.GetAccountFromArgs function.
// Comprehensive tests for GetAccountFromArgs are in internal/tools/common/account_test.go
func TestEmailToolsGetAccountFromArgs(t *testing.T) {
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := common.GetAccountFromArgs(context.Background(), tt.args)
			if got != tt.want {
				t.Errorf("GetAccountFromArgs() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEmailToolsGetAccountFromArgs_WithOAuthContext(t *testing.T) {
	// Create a context with OAuth user info
	userInfo := &oauth.UserInfo{
		Email: "oauth-user@example.com",
	}
	ctx := oauth.ContextWithUserInfo(context.Background(), userInfo)

	// OAuth context should take precedence
	args := map[string]interface{}{
		"account": "explicit-account",
	}
	got := common.GetAccountFromArgs(ctx, args)
	if got != "oauth-user@example.com" {
		t.Errorf("GetAccountFromArgs() = %v, want oauth-user@example.com", got)
	}
}

// TestEmailToolsPackage tests email tools package helper functions
func TestEmailToolsPackage(t *testing.T) {
	// Test that the package compiles and basic functionality works
	result := splitEmailAddresses("test@example.com")
	if len(result) != 1 || result[0] != "test@example.com" {
		t.Error("splitEmailAddresses not working correctly")
	}
}
