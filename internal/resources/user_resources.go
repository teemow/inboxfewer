package resources

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"
	"github.com/teemow/inboxfewer/internal/server"
)

// RegisterUserResources registers session-specific user resources
// These resources provide information about the current authenticated user
func RegisterUserResources(s *mcpserver.MCPServer, sc *server.ServerContext) error {
	// Register user profile resource
	profileResource := mcp.NewResource(
		"user://profile",
		"Current User Profile",
		mcp.WithResourceDescription("Information about the currently authenticated Google account"),
		mcp.WithMIMEType("application/json"),
	)

	s.AddResource(profileResource, func(ctx context.Context, request mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
		return handleUserProfile(ctx, request, sc)
	})

	// Register user settings resource
	settingsResource := mcp.NewResource(
		"user://gmail/settings",
		"Gmail Settings",
		mcp.WithResourceDescription("Gmail settings for the current user"),
		mcp.WithMIMEType("application/json"),
	)

	s.AddResource(settingsResource, func(ctx context.Context, request mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
		return handleGmailSettings(ctx, request, sc)
	})

	return nil
}

// handleUserProfile returns information about the current user's profile
func handleUserProfile(ctx context.Context, request mcp.ReadResourceRequest, sc *server.ServerContext) ([]mcp.ResourceContents, error) {
	// Get the account from the session context or use default
	account := "default"
	// TODO: Extract account from session when SessionIdManager is integrated

	gmailClient := sc.GmailClientForAccount(account)
	if gmailClient == nil {
		return nil, fmt.Errorf("no Gmail client available for account: %s", account)
	}

	// For now, return account information
	// TODO: Add full profile information by exposing Gmail Users.GetProfile through Client
	profileData := map[string]interface{}{
		"account":     account,
		"description": "User profile for Google Workspace",
		"note":        "Full profile details can be added by exposing Gmail API methods",
	}

	jsonData, err := json.MarshalIndent(profileData, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal profile data: %w", err)
	}

	return []mcp.ResourceContents{
		&mcp.TextResourceContents{
			URI:       request.Params.URI,
			MIMEType:  "application/json",
			Text:      string(jsonData),
		},
	}, nil
}

// handleGmailSettings returns Gmail settings for the current user
func handleGmailSettings(ctx context.Context, request mcp.ReadResourceRequest, sc *server.ServerContext) ([]mcp.ResourceContents, error) {
	// Get the account from the session context or use default
	account := "default"
	// TODO: Extract account from session when SessionIdManager is integrated

	gmailClient := sc.GmailClientForAccount(account)
	if gmailClient == nil {
		return nil, fmt.Errorf("no Gmail client available for account: %s", account)
	}

	// For now, return placeholder settings info
	// TODO: Add full settings by exposing Gmail Users.Settings API through Client
	settingsData := map[string]interface{}{
		"account":     account,
		"description": "Gmail settings for this account",
		"note":        "Full settings details can be added by exposing Gmail Settings API methods",
	}

	jsonData, err := json.MarshalIndent(settingsData, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal settings data: %w", err)
	}

	return []mcp.ResourceContents{
		&mcp.TextResourceContents{
			URI:       request.Params.URI,
			MIMEType:  "application/json",
			Text:      string(jsonData),
		},
	}, nil
}
