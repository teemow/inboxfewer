package common

import (
	"context"

	"github.com/teemow/inboxfewer/internal/mcp/oauth"
)

// GetAccountFromArgs extracts the account name from request arguments and context.
// For OAuth-authenticated requests, uses the authenticated user's email.
// Otherwise defaults to "default" or the explicitly provided account name.
//
// Priority order:
//  1. OAuth user email from context (set by OAuth middleware)
//  2. Explicit "account" argument in request
//  3. "default"
func GetAccountFromArgs(ctx context.Context, args map[string]interface{}) string {
	// First, check if there's an authenticated user in the OAuth context
	// This is set by the OAuth middleware after validating the Bearer token
	if userInfo, ok := oauth.GetUserFromContext(ctx); ok && userInfo != nil && userInfo.Email != "" {
		return userInfo.Email
	}

	// Fall back to explicit account argument or "default"
	if accountVal, ok := args["account"].(string); ok && accountVal != "" {
		return accountVal
	}
	return "default"
}
