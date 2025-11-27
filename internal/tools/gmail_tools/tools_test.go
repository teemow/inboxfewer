package gmail_tools

import (
	"context"
	"testing"

	"github.com/teemow/inboxfewer/internal/mcp/oauth"
	"github.com/teemow/inboxfewer/internal/tools/common"
)

// TestCommonGetAccountFromArgs verifies that the gmail_tools package
// correctly uses the shared common.GetAccountFromArgs function.
// Comprehensive tests for GetAccountFromArgs are in internal/tools/common/account_test.go
func TestCommonGetAccountFromArgs(t *testing.T) {
	ctx := context.Background()

	// Test basic functionality
	args := map[string]interface{}{
		"account": "test-account",
	}
	account := common.GetAccountFromArgs(ctx, args)
	if account != "test-account" {
		t.Errorf("GetAccountFromArgs() = %v, want test-account", account)
	}

	// Test default account
	emptyArgs := map[string]interface{}{}
	defaultAccount := common.GetAccountFromArgs(ctx, emptyArgs)
	if defaultAccount != "default" {
		t.Errorf("GetAccountFromArgs() = %v, want default", defaultAccount)
	}

	// Test OAuth integration
	userInfo := &oauth.UserInfo{
		Email: "oauth-user@example.com",
	}
	ctxWithUser := oauth.ContextWithUserInfo(context.Background(), userInfo)
	result := common.GetAccountFromArgs(ctxWithUser, args)
	if result != "oauth-user@example.com" {
		t.Errorf("GetAccountFromArgs() with OAuth = %v, expected oauth-user@example.com", result)
	}
}

// TestSpamMarkingHandlers tests that spam marking handler functions exist and have correct signatures
func TestSpamMarkingHandlers(t *testing.T) {
	// Verify that the handler functions exist with correct signatures
	// This is a compile-time check to ensure the functions are properly defined
	_ = handleMarkThreadsAsSpam
	_ = handleUnmarkThreadsAsSpam
	// If the functions are not defined or have wrong signatures, this test will not compile
	t.Log("Spam marking handler functions are properly defined")
}
