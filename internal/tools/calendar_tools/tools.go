package calendar_tools

import (
	"context"
	"fmt"

	mcpserver "github.com/mark3labs/mcp-go/server"

	"github.com/teemow/inboxfewer/internal/calendar"
	"github.com/teemow/inboxfewer/internal/server"
)

// getAccountFromArgs extracts the account name from request arguments, defaulting to "default"
func getAccountFromArgs(args map[string]interface{}) string {
	account := "default"
	if accountVal, ok := args["account"].(string); ok && accountVal != "" {
		account = accountVal
	}
	return account
}

// getCalendarClient retrieves or creates a calendar client for the specified account
func getCalendarClient(ctx context.Context, account string, sc *server.ServerContext) (*calendar.Client, error) {
	client := sc.CalendarClientForAccount(account)
	if client == nil {
		// Check if token exists before trying to create client
		if !calendar.HasTokenForAccount(account) {
			authURL := calendar.GetAuthURLForAccount(account)
			return nil, fmt.Errorf(`Google OAuth token not found for account "%s". To authorize access:

1. Visit this URL in your browser:
   %s

2. Sign in with your Google account
3. Grant access to Google services (Calendar, Gmail, Docs, Drive)
4. Copy the authorization code

5. Provide the authorization code to your AI agent
   The agent will use the google_save_auth_code tool with account="%s" to complete authentication.

Note: You only need to authorize once. The tokens will be automatically refreshed.`, account, authURL, account)
		}

		var err error
		client, err = calendar.NewClientForAccount(ctx, account)
		if err != nil {
			return nil, fmt.Errorf("failed to create Calendar client for account %s: %w", account, err)
		}
		sc.SetCalendarClientForAccount(account, client)
	}
	return client, nil
}

// RegisterCalendarTools registers all Calendar-related tools with the MCP server
func RegisterCalendarTools(s *mcpserver.MCPServer, sc *server.ServerContext) error {
	// Register event tools
	if err := RegisterEventTools(s, sc); err != nil {
		return fmt.Errorf("failed to register event tools: %w", err)
	}

	// Register calendar list tools
	if err := RegisterCalendarListTools(s, sc); err != nil {
		return fmt.Errorf("failed to register calendar list tools: %w", err)
	}

	// Register scheduling tools
	if err := RegisterSchedulingTools(s, sc); err != nil {
		return fmt.Errorf("failed to register scheduling tools: %w", err)
	}

	return nil
}
