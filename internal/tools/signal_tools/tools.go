package signal_tools

import (
	"context"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"

	"github.com/teemow/inboxfewer/internal/server"
	"github.com/teemow/inboxfewer/internal/signal"
)

// getAccountFromArgs extracts the account name from request arguments, defaulting to "default"
func getAccountFromArgs(args map[string]interface{}) string {
	account := "default"
	if accountVal, ok := args["account"].(string); ok && accountVal != "" {
		account = accountVal
	}
	return account
}

// getSignalClient gets or creates a Signal client for the specified account
func getSignalClient(ctx context.Context, sc *server.ServerContext, account string) (*signal.Client, error) {
	client := sc.SignalClientForAccount(account)
	if client == nil {
		return nil, fmt.Errorf("no Signal client configured for account %s. Please configure signal-cli for this account", account)
	}
	return client, nil
}

// RegisterSignalTools registers all Signal-related tools with the MCP server
func RegisterSignalTools(s *mcpserver.MCPServer, sc *server.ServerContext, readOnly bool) error {
	// Register send tools (require write permissions)
	if err := RegisterSendTools(s, sc, readOnly); err != nil {
		return fmt.Errorf("failed to register send tools: %w", err)
	}

	// Register receive tools (read-only by nature)
	if err := RegisterReceiveTools(s, sc); err != nil {
		return fmt.Errorf("failed to register receive tools: %w", err)
	}

	// List groups tool (read-only)
	listGroupsTool := mcp.NewTool("signal_list_groups",
		mcp.WithDescription("List all Signal groups the user is a member of"),
		mcp.WithString("account",
			mcp.Description("Account name (default: 'default'). Used to manage multiple Signal accounts."),
		),
	)

	s.AddTool(listGroupsTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleListGroups(ctx, request, sc)
	})

	return nil
}

// handleListGroups handles the signal_list_groups tool
func handleListGroups(ctx context.Context, request mcp.CallToolRequest, sc *server.ServerContext) (*mcp.CallToolResult, error) {
	// Type assert arguments to map
	args, ok := request.Params.Arguments.(map[string]interface{})
	if !ok {
		args = make(map[string]interface{})
	}

	// Get account from arguments
	account := getAccountFromArgs(args)

	// Get Signal client
	client, err := getSignalClient(ctx, sc, account)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	// List groups
	groups, err := client.ListGroups()
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to list groups: %v", err)), nil
	}

	// Format response
	if len(groups) == 0 {
		return mcp.NewToolResultText("No Signal groups found"), nil
	}

	result := fmt.Sprintf("Found %d Signal group(s):\n", len(groups))
	for i, group := range groups {
		result += fmt.Sprintf("\n%d. %s\n", i+1, group.Name)
		if group.ID != "" {
			result += fmt.Sprintf("   ID: %s\n", group.ID)
		}
		if len(group.Members) > 0 {
			result += fmt.Sprintf("   Members: %d\n", len(group.Members))
		}
	}

	return mcp.NewToolResultText(result), nil
}
