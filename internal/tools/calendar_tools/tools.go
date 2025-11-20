package calendar_tools

import (
	"context"
	"fmt"

	mcpserver "github.com/mark3labs/mcp-go/server"

	"github.com/teemow/inboxfewer/internal/calendar"
	"github.com/teemow/inboxfewer/internal/google"
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
			errorMsg := google.GetAuthenticationErrorMessage(account)
			return nil, fmt.Errorf("%s", errorMsg)
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
func RegisterCalendarTools(s *mcpserver.MCPServer, sc *server.ServerContext, readOnly bool) error {
	// Register event tools (some write operations require !readOnly)
	if err := RegisterEventTools(s, sc, readOnly); err != nil {
		return fmt.Errorf("failed to register event tools: %w", err)
	}

	// Register calendar list tools (read-only)
	if err := RegisterCalendarListTools(s, sc); err != nil {
		return fmt.Errorf("failed to register calendar list tools: %w", err)
	}

	// Register scheduling tools (read-only)
	if err := RegisterSchedulingTools(s, sc); err != nil {
		return fmt.Errorf("failed to register scheduling tools: %w", err)
	}

	return nil
}
