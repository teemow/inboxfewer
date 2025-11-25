package resources

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"
	"github.com/teemow/inboxfewer/internal/mcp/oauth_library"
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

// extractAccountFromContext extracts the user's email from OAuth context
// Falls back to "default" for STDIO transport or if no OAuth context is available
func extractAccountFromContext(ctx context.Context) string {
	// Try to get user info from OAuth context (HTTP/SSE transport)
	if userInfo, ok := oauth_library.GetUserFromContext(ctx); ok {
		return userInfo.Email
	}
	// Fallback to default for STDIO transport
	return "default"
}

// handleUserProfile returns information about the current user's profile
func handleUserProfile(ctx context.Context, request mcp.ReadResourceRequest, sc *server.ServerContext) ([]mcp.ResourceContents, error) {
	// Extract account (email) from OAuth context
	account := extractAccountFromContext(ctx)

	gmailClient := sc.GmailClientForAccount(account)
	if gmailClient == nil {
		return nil, fmt.Errorf("no Gmail client available for account: %s", account)
	}

	// Get full user profile from Gmail API
	profile, err := gmailClient.GetProfile(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get user profile: %w", err)
	}

	profileData := map[string]interface{}{
		"account":       account,
		"email":         profile.EmailAddress,
		"historyId":     profile.HistoryId,
		"messagesTotal": profile.MessagesTotal,
		"threadsTotal":  profile.ThreadsTotal,
		"description":   "User profile for Google Workspace",
	}

	jsonData, err := json.MarshalIndent(profileData, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal profile data: %w", err)
	}

	return []mcp.ResourceContents{
		&mcp.TextResourceContents{
			URI:      request.Params.URI,
			MIMEType: "application/json",
			Text:     string(jsonData),
		},
	}, nil
}

// handleGmailSettings returns Gmail settings for the current user
func handleGmailSettings(ctx context.Context, request mcp.ReadResourceRequest, sc *server.ServerContext) ([]mcp.ResourceContents, error) {
	// Extract account (email) from OAuth context
	account := extractAccountFromContext(ctx)

	gmailClient := sc.GmailClientForAccount(account)
	if gmailClient == nil {
		return nil, fmt.Errorf("no Gmail client available for account: %s", account)
	}

	// Get vacation/auto-reply settings from Gmail API
	settings, err := gmailClient.GetVacationSettings(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get Gmail settings: %w", err)
	}

	settingsData := map[string]interface{}{
		"account":               account,
		"enableAutoReply":       settings.EnableAutoReply,
		"responseSubject":       settings.ResponseSubject,
		"responseBodyPlainText": settings.ResponseBodyPlainText,
		"responseBodyHtml":      settings.ResponseBodyHtml,
		"restrictToContacts":    settings.RestrictToContacts,
		"restrictToDomain":      settings.RestrictToDomain,
		"startTime":             settings.StartTime,
		"endTime":               settings.EndTime,
		"description":           "Gmail vacation/auto-reply settings",
	}

	jsonData, err := json.MarshalIndent(settingsData, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal settings data: %w", err)
	}

	return []mcp.ResourceContents{
		&mcp.TextResourceContents{
			URI:      request.Params.URI,
			MIMEType: "application/json",
			Text:     string(jsonData),
		},
	}, nil
}
