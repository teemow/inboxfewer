package gmail_tools

import (
	"context"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"

	"github.com/teemow/inboxfewer/internal/gmail"
	"github.com/teemow/inboxfewer/internal/google"
	"github.com/teemow/inboxfewer/internal/server"
	"github.com/teemow/inboxfewer/internal/tools/common"
)

// RegisterContactTools registers contact-related tools with the MCP server
func RegisterContactTools(s *mcpserver.MCPServer, sc *server.ServerContext) error {
	// Search contacts tool
	searchContactsTool := mcp.NewTool("gmail_search_contacts",
		mcp.WithDescription("Search for contacts in Google Contacts"),
		mcp.WithString("account",
			mcp.Description("Account name (default: 'default'). Used to manage multiple Google accounts."),
		),
		mcp.WithString("query",
			mcp.Required(),
			mcp.Description("Search query to find contacts (e.g., name, email, phone number)"),
		),
		mcp.WithNumber("maxResults",
			mcp.Description("Maximum number of results to return (default: 10)"),
		),
	)

	s.AddTool(searchContactsTool, common.InstrumentedToolHandlerWithService(
		"gmail_search_contacts", "gmail", "list", sc,
		func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return handleSearchContacts(ctx, request, sc)
		}))

	return nil
}

func handleSearchContacts(ctx context.Context, request mcp.CallToolRequest, sc *server.ServerContext) (*mcp.CallToolResult, error) {
	args := request.GetArguments()

	query, ok := args["query"].(string)
	if !ok || query == "" {
		return mcp.NewToolResultError("query is required"), nil
	}

	maxResults := 10
	if maxResultsVal, ok := args["maxResults"]; ok {
		if maxResultsFloat, ok := maxResultsVal.(float64); ok {
			maxResults = int(maxResultsFloat)
		}
	}

	// Get or create Gmail client for the specified account
	account := common.GetAccountFromArgs(ctx, args)
	client := sc.GmailClientForAccount(account)
	if client == nil {
		if !gmail.HasTokenForAccount(account) {
			errorMsg := google.GetAuthenticationErrorMessage(account)
			return mcp.NewToolResultError(errorMsg), nil
		}

		var err error
		client, err = gmail.NewClientForAccount(ctx, account)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to create Gmail client for account %s: %v", account, err)), nil
		}
		sc.SetGmailClientForAccount(account, client)
	}

	contacts, err := client.SearchContacts(query, maxResults)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to search contacts: %v", err)), nil
	}

	if len(contacts) == 0 {
		return mcp.NewToolResultText(fmt.Sprintf("No contacts found for query: %s", query)), nil
	}

	result := fmt.Sprintf("Found %d contact(s):\n\n", len(contacts))
	for i, contact := range contacts {
		result += fmt.Sprintf("%d. %s\n", i+1, contact.DisplayName)
		if contact.EmailAddress != "" {
			result += fmt.Sprintf("   Email: %s\n", contact.EmailAddress)
		}
		if contact.PhoneNumber != "" {
			result += fmt.Sprintf("   Phone: %s\n", contact.PhoneNumber)
		}
		result += "\n"
	}

	return mcp.NewToolResultText(result), nil
}
