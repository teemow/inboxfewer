package google_tools

import (
	"context"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"

	"github.com/teemow/inboxfewer/internal/google"
	"github.com/teemow/inboxfewer/internal/server"
)

// RegisterGoogleTools registers all Google OAuth-related tools with the MCP server
func RegisterGoogleTools(s *mcpserver.MCPServer, sc *server.ServerContext) error {
	// Get OAuth URL tool
	getAuthURLTool := mcp.NewTool("google_get_auth_url",
		mcp.WithDescription("Get the OAuth URL to authorize Google services access (Gmail, Docs, Drive) for a specific account"),
		mcp.WithString("account",
			mcp.Description("Account name (default: 'default'). Used to manage multiple Google accounts."),
		),
	)

	s.AddTool(getAuthURLTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleGetAuthURL(ctx, request, sc)
	})

	// Save authorization code tool
	saveAuthCodeTool := mcp.NewTool("google_save_auth_code",
		mcp.WithDescription("Save the OAuth authorization code to complete Google services authentication (Gmail, Docs, Drive) for a specific account"),
		mcp.WithString("account",
			mcp.Description("Account name (default: 'default'). Used to manage multiple Google accounts."),
		),
		mcp.WithString("authCode",
			mcp.Required(),
			mcp.Description("The authorization code from Google OAuth"),
		),
	)

	s.AddTool(saveAuthCodeTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleSaveAuthCode(ctx, request, sc)
	})

	return nil
}

func handleGetAuthURL(ctx context.Context, request mcp.CallToolRequest, sc *server.ServerContext) (*mcp.CallToolResult, error) {
	args := request.GetArguments()

	// Get account name, default to "default"
	account := "default"
	if accountVal, ok := args["account"].(string); ok && accountVal != "" {
		account = accountVal
	}

	authURL := google.GetAuthURLForAccount(account)

	result := fmt.Sprintf(`To authorize Google services access (Gmail, Docs, Drive) for account "%s":

1. Visit this URL in your browser:
   %s

2. Sign in with your Google account
3. Grant access to Google services
4. Copy the authorization code

5. Call the google_save_auth_code tool with the code and account name to complete authentication`, account, authURL)

	return mcp.NewToolResultText(result), nil
}

func handleSaveAuthCode(ctx context.Context, request mcp.CallToolRequest, sc *server.ServerContext) (*mcp.CallToolResult, error) {
	args := request.GetArguments()

	// Get account name, default to "default"
	account := "default"
	if accountVal, ok := args["account"].(string); ok && accountVal != "" {
		account = accountVal
	}

	authCode, ok := args["authCode"].(string)
	if !ok || authCode == "" {
		return mcp.NewToolResultError("authCode is required"), nil
	}

	err := google.SaveTokenForAccount(ctx, account, authCode)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to save authorization code for account %s: %v", account, err)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("✅ Authorization successful for account '%s'! Google services token saved. You can now use all Gmail and Google Docs tools with this account.", account)), nil
}
