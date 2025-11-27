package meet_tools

import (
	"context"
	"testing"

	"github.com/teemow/inboxfewer/internal/mcp/oauth"
	"github.com/teemow/inboxfewer/internal/tools/common"
)

// TestCommonGetAccountFromArgs verifies that the meet_tools package
// correctly uses the shared common.GetAccountFromArgs function.
// Comprehensive tests for GetAccountFromArgs are in internal/tools/common/account_test.go
func TestCommonGetAccountFromArgs(t *testing.T) {
	ctx := context.Background()

	// Test basic functionality
	args := map[string]interface{}{
		"account": "test-account",
	}
	result := common.GetAccountFromArgs(ctx, args)
	if result != "test-account" {
		t.Errorf("GetAccountFromArgs() = %v, expected test-account", result)
	}

	// Test OAuth integration
	userInfo := &oauth.UserInfo{
		Email: "oauth-user@example.com",
	}
	ctxWithUser := oauth.ContextWithUserInfo(context.Background(), userInfo)
	result = common.GetAccountFromArgs(ctxWithUser, args)
	if result != "oauth-user@example.com" {
		t.Errorf("GetAccountFromArgs() with OAuth = %v, expected oauth-user@example.com", result)
	}
}
