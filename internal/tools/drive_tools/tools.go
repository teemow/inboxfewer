package drive_tools

import (
	"context"
	"fmt"

	mcpserver "github.com/mark3labs/mcp-go/server"

	"github.com/teemow/inboxfewer/internal/drive"
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

// getDriveClient retrieves or creates a drive client for the specified account
func getDriveClient(ctx context.Context, account string, sc *server.ServerContext) (*drive.Client, error) {
	client := sc.DriveClientForAccount(account)
	if client == nil {
		// Check if token exists before trying to create client
		if !drive.HasTokenForAccount(account) {
			errorMsg := google.GetAuthenticationErrorMessage(account)
			return nil, fmt.Errorf("%s", errorMsg)
		}

		var err error
		client, err = drive.NewClientForAccount(ctx, account)
		if err != nil {
			return nil, fmt.Errorf("failed to create Drive client for account %s: %w", account, err)
		}
		sc.SetDriveClientForAccount(account, client)
	}
	return client, nil
}

// RegisterDriveTools registers all Google Drive-related tools with the MCP server
func RegisterDriveTools(s *mcpserver.MCPServer, sc *server.ServerContext, readOnly bool) error {
	// Register file operation tools
	if err := registerFileTools(s, sc, readOnly); err != nil {
		return fmt.Errorf("failed to register file tools: %w", err)
	}

	// Register folder operation tools
	if err := registerFolderTools(s, sc, readOnly); err != nil {
		return fmt.Errorf("failed to register folder tools: %w", err)
	}

	// Register permission/sharing tools
	if err := registerShareTools(s, sc, readOnly); err != nil {
		return fmt.Errorf("failed to register share tools: %w", err)
	}

	return nil
}
