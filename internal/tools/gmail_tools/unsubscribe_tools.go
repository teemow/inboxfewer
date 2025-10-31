package gmail_tools

import (
	"context"
	"fmt"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"

	"github.com/teemow/inboxfewer/internal/gmail"
	"github.com/teemow/inboxfewer/internal/server"
)

// RegisterUnsubscribeTools registers unsubscribe-related tools with the MCP server
func RegisterUnsubscribeTools(s *mcpserver.MCPServer, sc *server.ServerContext, readOnly bool) error {
	// Get unsubscribe info tool (read-only, always available)
	getUnsubscribeInfoTool := mcp.NewTool("gmail_get_unsubscribe_info",
		mcp.WithDescription("Extract unsubscribe information from a Gmail message. Returns available unsubscribe methods (mailto or HTTP)."),
		mcp.WithString("account",
			mcp.Description("Account name (default: 'default'). Used to manage multiple Google accounts."),
		),
		mcp.WithString("messageId",
			mcp.Required(),
			mcp.Description("The ID of the Gmail message to check for unsubscribe information"),
		),
	)

	s.AddTool(getUnsubscribeInfoTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleGetUnsubscribeInfo(ctx, request, sc)
	})

	// Unsubscribe via HTTP tool (safe operation, always available)
	unsubscribeViaHTTPTool := mcp.NewTool("gmail_unsubscribe_via_http",
		mcp.WithDescription("Unsubscribe from a newsletter using an HTTP unsubscribe link. Use gmail_get_unsubscribe_info first to get available unsubscribe methods."),
		mcp.WithString("account",
			mcp.Description("Account name (default: 'default'). Used to manage multiple Google accounts."),
		),
		mcp.WithString("url",
			mcp.Required(),
			mcp.Description("The HTTP/HTTPS unsubscribe URL (obtained from gmail_get_unsubscribe_info)"),
		),
	)

	s.AddTool(unsubscribeViaHTTPTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleUnsubscribeViaHTTP(ctx, request, sc)
	})

	return nil
}

func handleGetUnsubscribeInfo(ctx context.Context, request mcp.CallToolRequest, sc *server.ServerContext) (*mcp.CallToolResult, error) {
	args := request.GetArguments()

	// Parse required fields
	messageID, ok := args["messageId"].(string)
	if !ok || messageID == "" {
		return mcp.NewToolResultError("'messageId' field is required"), nil
	}

	// Get or create Gmail client for the specified account
	account := getAccountFromArgs(args)
	client := sc.GmailClientForAccount(account)
	if client == nil {
		if !gmail.HasTokenForAccount(account) {
			authURL := gmail.GetAuthURLForAccount(account)
			errorMsg := fmt.Sprintf(`Google OAuth token not found for account "%s". To authorize access:

1. Visit this URL in your browser:
   %s

2. Sign in with your Google account
3. Grant access to Google services (Gmail, Docs, Drive, Contacts)
4. Copy the authorization code

5. Provide the authorization code to your AI agent
   The agent will use the google_save_auth_code tool with account="%s" to complete authentication.

Note: You only need to authorize once. The tokens will be automatically refreshed.`, account, authURL, account)
			return mcp.NewToolResultError(errorMsg), nil
		}

		var err error
		client, err = gmail.NewClientForAccount(ctx, account)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to create Gmail client for account %s: %v", account, err)), nil
		}
		sc.SetGmailClientForAccount(account, client)
	}

	// Get unsubscribe info
	info, err := client.GetUnsubscribeInfo(messageID)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to get unsubscribe info: %v", err)), nil
	}

	// Format result
	if !info.HasUnsubscribe {
		return mcp.NewToolResultText("This message does not contain unsubscribe information (no List-Unsubscribe header found)."), nil
	}

	var result strings.Builder
	result.WriteString(fmt.Sprintf("Unsubscribe information for message %s:\n\n", messageID))
	result.WriteString(fmt.Sprintf("Found %d unsubscribe method(s):\n\n", len(info.Methods)))

	for i, method := range info.Methods {
		result.WriteString(fmt.Sprintf("%d. Type: %s\n", i+1, method.Type))
		result.WriteString(fmt.Sprintf("   URL: %s\n", method.URL))

		if method.Type == "http" {
			result.WriteString("   Action: Use gmail_unsubscribe_via_http with this URL to unsubscribe automatically\n")
		} else if method.Type == "mailto" {
			result.WriteString("   Action: Send an email to this address to unsubscribe (use gmail_send_email)\n")
		}
		result.WriteString("\n")
	}

	return mcp.NewToolResultText(result.String()), nil
}

func handleUnsubscribeViaHTTP(ctx context.Context, request mcp.CallToolRequest, sc *server.ServerContext) (*mcp.CallToolResult, error) {
	args := request.GetArguments()

	// Parse required fields
	url, ok := args["url"].(string)
	if !ok || url == "" {
		return mcp.NewToolResultError("'url' field is required"), nil
	}

	// Validate URL format
	if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
		return mcp.NewToolResultError("URL must start with http:// or https://"), nil
	}

	// Get or create Gmail client for the specified account
	account := getAccountFromArgs(args)
	client := sc.GmailClientForAccount(account)
	if client == nil {
		if !gmail.HasTokenForAccount(account) {
			authURL := gmail.GetAuthURLForAccount(account)
			errorMsg := fmt.Sprintf(`Google OAuth token not found for account "%s". To authorize access:

1. Visit this URL in your browser:
   %s

2. Sign in with your Google account
3. Grant access to Google services (Gmail, Docs, Drive, Contacts)
4. Copy the authorization code

5. Provide the authorization code to your AI agent
   The agent will use the google_save_auth_code tool with account="%s" to complete authentication.

Note: You only need to authorize once. The tokens will be automatically refreshed.`, account, authURL, account)
			return mcp.NewToolResultError(errorMsg), nil
		}

		var err error
		client, err = gmail.NewClientForAccount(ctx, account)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to create Gmail client for account %s: %v", account, err)), nil
		}
		sc.SetGmailClientForAccount(account, client)
	}

	// Perform unsubscribe
	if err := client.UnsubscribeViaHTTP(url); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to unsubscribe: %v", err)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Successfully unsubscribed via HTTP!\nURL: %s\n\nNote: You should receive a confirmation from the sender. You may want to archive or delete emails from this sender.", url)), nil
}
