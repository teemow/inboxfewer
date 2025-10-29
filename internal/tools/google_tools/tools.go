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
		mcp.WithDescription("Get the OAuth URL to authorize Google services access (Gmail, Docs, Drive)"),
	)

	s.AddTool(getAuthURLTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleGetAuthURL(ctx, request, sc)
	})

	// Save authorization code tool
	saveAuthCodeTool := mcp.NewTool("google_save_auth_code",
		mcp.WithDescription("Save the OAuth authorization code to complete Google services authentication (Gmail, Docs, Drive)"),
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
	authURL := google.GetAuthURL()

	result := fmt.Sprintf(`To authorize Google services access (Gmail, Docs, Drive):

1. Visit this URL in your browser:
   %s

2. Sign in with your Google account
3. Grant access to Google services
4. Copy the authorization code

5. Call the google_save_auth_code tool with the code to complete authentication`, authURL)

	return mcp.NewToolResultText(result), nil
}

func handleSaveAuthCode(ctx context.Context, request mcp.CallToolRequest, sc *server.ServerContext) (*mcp.CallToolResult, error) {
	args := request.GetArguments()

	authCode, ok := args["authCode"].(string)
	if !ok || authCode == "" {
		return mcp.NewToolResultError("authCode is required"), nil
	}

	err := google.SaveToken(ctx, authCode)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to save authorization code: %v", err)), nil
	}

	return mcp.NewToolResultText("✅ Authorization successful! Google services token saved. You can now use all Gmail and Google Docs tools."), nil
}
