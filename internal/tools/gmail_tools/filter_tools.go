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

// RegisterFilterTools registers filter-related tools with the MCP server
func RegisterFilterTools(s *mcpserver.MCPServer, sc *server.ServerContext) error {
	// Create filter tool
	createFilterTool := mcp.NewTool("gmail_create_filter",
		mcp.WithDescription("Create a new Gmail filter to automatically organize incoming emails. Filters can match on sender, recipient, subject, or custom queries, and perform actions like labeling, archiving, or marking as read."),
		mcp.WithString("account",
			mcp.Description("Account name (default: 'default'). Used to manage multiple Google accounts."),
		),
		// Criteria fields
		mcp.WithString("from",
			mcp.Description("Filter emails from this sender (e.g., 'newsletter@example.com')"),
		),
		mcp.WithString("to",
			mcp.Description("Filter emails sent to this recipient (e.g., 'myalias@example.com')"),
		),
		mcp.WithString("subject",
			mcp.Description("Filter emails with this subject (e.g., 'Weekly Report')"),
		),
		mcp.WithString("query",
			mcp.Description("Gmail search query for advanced filtering (e.g., 'has:attachment larger:10M')"),
		),
		mcp.WithBoolean("hasAttachment",
			mcp.Description("Filter emails that have attachments"),
		),
		// Action fields
		mcp.WithString("addLabelIds",
			mcp.Description("Comma-separated list of label IDs to add (e.g., 'Label_1,Label_2'). Use gmail_list_labels to get label IDs."),
		),
		mcp.WithString("removeLabelIds",
			mcp.Description("Comma-separated list of label IDs to remove (e.g., 'INBOX,UNREAD')"),
		),
		mcp.WithBoolean("archive",
			mcp.Description("Remove from inbox (archive)"),
		),
		mcp.WithBoolean("markAsRead",
			mcp.Description("Mark as read"),
		),
		mcp.WithBoolean("star",
			mcp.Description("Add star"),
		),
		mcp.WithBoolean("markAsSpam",
			mcp.Description("Mark as spam"),
		),
		mcp.WithBoolean("delete",
			mcp.Description("Send to trash"),
		),
		mcp.WithString("forward",
			mcp.Description("Forward to this email address"),
		),
	)

	s.AddTool(createFilterTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleCreateFilter(ctx, request, sc)
	})

	// List filters tool
	listFiltersTool := mcp.NewTool("gmail_list_filters",
		mcp.WithDescription("List all existing Gmail filters for the account"),
		mcp.WithString("account",
			mcp.Description("Account name (default: 'default'). Used to manage multiple Google accounts."),
		),
	)

	s.AddTool(listFiltersTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleListFilters(ctx, request, sc)
	})

	// Delete filter tool
	deleteFilterTool := mcp.NewTool("gmail_delete_filter",
		mcp.WithDescription("Delete a Gmail filter by its ID (obtain ID from gmail_list_filters)"),
		mcp.WithString("account",
			mcp.Description("Account name (default: 'default'). Used to manage multiple Google accounts."),
		),
		mcp.WithString("filterId",
			mcp.Required(),
			mcp.Description("The ID of the filter to delete (obtained from gmail_list_filters)"),
		),
	)

	s.AddTool(deleteFilterTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleDeleteFilter(ctx, request, sc)
	})

	// List labels tool
	listLabelsTool := mcp.NewTool("gmail_list_labels",
		mcp.WithDescription("List all Gmail labels for the account. Use this to get label IDs for creating filters."),
		mcp.WithString("account",
			mcp.Description("Account name (default: 'default'). Used to manage multiple Google accounts."),
		),
	)

	s.AddTool(listLabelsTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleListLabels(ctx, request, sc)
	})

	return nil
}

func handleCreateFilter(ctx context.Context, request mcp.CallToolRequest, sc *server.ServerContext) (*mcp.CallToolResult, error) {
	args := request.GetArguments()

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

	// Parse criteria
	criteria := gmail.FilterCriteria{}
	if from, ok := args["from"].(string); ok && from != "" {
		criteria.From = from
	}
	if to, ok := args["to"].(string); ok && to != "" {
		criteria.To = to
	}
	if subject, ok := args["subject"].(string); ok && subject != "" {
		criteria.Subject = subject
	}
	if query, ok := args["query"].(string); ok && query != "" {
		criteria.Query = query
	}
	if hasAttachment, ok := args["hasAttachment"].(bool); ok {
		criteria.HasAttachment = hasAttachment
	}

	// Validate that at least one criteria is specified
	if criteria.From == "" && criteria.To == "" && criteria.Subject == "" && criteria.Query == "" && !criteria.HasAttachment {
		return mcp.NewToolResultError("At least one filter criteria must be specified (from, to, subject, query, or hasAttachment)"), nil
	}

	// Parse action
	action := gmail.FilterAction{}
	if addLabelIdsStr, ok := args["addLabelIds"].(string); ok && addLabelIdsStr != "" {
		action.AddLabelIDs = splitLabelIDs(addLabelIdsStr)
	}
	if removeLabelIdsStr, ok := args["removeLabelIds"].(string); ok && removeLabelIdsStr != "" {
		action.RemoveLabelIDs = splitLabelIDs(removeLabelIdsStr)
	}
	if archive, ok := args["archive"].(bool); ok {
		action.Archive = archive
	}
	if markAsRead, ok := args["markAsRead"].(bool); ok {
		action.MarkAsRead = markAsRead
	}
	if star, ok := args["star"].(bool); ok {
		action.Star = star
	}
	if markAsSpam, ok := args["markAsSpam"].(bool); ok {
		action.MarkAsSpam = markAsSpam
	}
	if deleteFlag, ok := args["delete"].(bool); ok {
		action.Delete = deleteFlag
	}
	if forward, ok := args["forward"].(string); ok && forward != "" {
		action.Forward = forward
	}

	// Validate that at least one action is specified
	if len(action.AddLabelIDs) == 0 && len(action.RemoveLabelIDs) == 0 &&
		!action.Archive && !action.MarkAsRead && !action.Star &&
		!action.MarkAsSpam && !action.Delete && action.Forward == "" {
		return mcp.NewToolResultError("At least one filter action must be specified"), nil
	}

	// Create the filter
	filterInfo, err := client.CreateFilter(criteria, action)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to create filter: %v", err)), nil
	}

	// Format result
	result := formatFilterInfo(filterInfo, "Filter created successfully!")
	return mcp.NewToolResultText(result), nil
}

func handleListFilters(ctx context.Context, request mcp.CallToolRequest, sc *server.ServerContext) (*mcp.CallToolResult, error) {
	args := request.GetArguments()

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

	// List filters
	filters, err := client.ListFilters()
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to list filters: %v", err)), nil
	}

	if len(filters) == 0 {
		return mcp.NewToolResultText("No filters found for this account."), nil
	}

	// Format result
	var result strings.Builder
	result.WriteString(fmt.Sprintf("Found %d filter(s):\n\n", len(filters)))

	for i, filter := range filters {
		result.WriteString(fmt.Sprintf("Filter %d:\n", i+1))
		result.WriteString(formatFilterInfo(filter, ""))
		result.WriteString("\n")
	}

	return mcp.NewToolResultText(result.String()), nil
}

func handleDeleteFilter(ctx context.Context, request mcp.CallToolRequest, sc *server.ServerContext) (*mcp.CallToolResult, error) {
	args := request.GetArguments()

	// Parse required fields
	filterID, ok := args["filterId"].(string)
	if !ok || filterID == "" {
		return mcp.NewToolResultError("'filterId' field is required"), nil
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

	// Delete the filter
	if err := client.DeleteFilter(filterID); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to delete filter: %v", err)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Successfully deleted filter %s", filterID)), nil
}

func handleListLabels(ctx context.Context, request mcp.CallToolRequest, sc *server.ServerContext) (*mcp.CallToolResult, error) {
	args := request.GetArguments()

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

	// List labels
	labels, err := client.ListLabels()
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to list labels: %v", err)), nil
	}

	if len(labels) == 0 {
		return mcp.NewToolResultText("No labels found for this account."), nil
	}

	// Format result
	var result strings.Builder
	result.WriteString(fmt.Sprintf("Found %d label(s):\n\n", len(labels)))

	// Separate system labels and user labels
	var systemLabels []string
	var userLabels []string

	for _, label := range labels {
		labelInfo := fmt.Sprintf("  ID: %s, Name: %s", label.Id, label.Name)
		if label.Type == "system" {
			systemLabels = append(systemLabels, labelInfo)
		} else {
			userLabels = append(userLabels, labelInfo)
		}
	}

	if len(systemLabels) > 0 {
		result.WriteString("System Labels:\n")
		for _, label := range systemLabels {
			result.WriteString(label + "\n")
		}
		result.WriteString("\n")
	}

	if len(userLabels) > 0 {
		result.WriteString("User Labels:\n")
		for _, label := range userLabels {
			result.WriteString(label + "\n")
		}
	}

	return mcp.NewToolResultText(result.String()), nil
}

// splitLabelIDs splits a comma-separated string of label IDs
func splitLabelIDs(labelIDs string) []string {
	if labelIDs == "" {
		return nil
	}

	parts := strings.Split(labelIDs, ",")
	result := make([]string, 0, len(parts))
	for _, id := range parts {
		trimmed := strings.TrimSpace(id)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

// formatFilterInfo formats a FilterInfo for display
func formatFilterInfo(filter *gmail.FilterInfo, header string) string {
	var result strings.Builder

	if header != "" {
		result.WriteString(header + "\n\n")
	}

	result.WriteString(fmt.Sprintf("Filter ID: %s\n\n", filter.ID))

	// Format criteria
	result.WriteString("Criteria:\n")
	if filter.Criteria.From != "" {
		result.WriteString(fmt.Sprintf("  From: %s\n", filter.Criteria.From))
	}
	if filter.Criteria.To != "" {
		result.WriteString(fmt.Sprintf("  To: %s\n", filter.Criteria.To))
	}
	if filter.Criteria.Subject != "" {
		result.WriteString(fmt.Sprintf("  Subject: %s\n", filter.Criteria.Subject))
	}
	if filter.Criteria.Query != "" {
		result.WriteString(fmt.Sprintf("  Query: %s\n", filter.Criteria.Query))
	}
	if filter.Criteria.HasAttachment {
		result.WriteString("  Has Attachment: true\n")
	}

	// Format actions
	result.WriteString("\nActions:\n")
	if len(filter.Action.AddLabelIDs) > 0 {
		result.WriteString(fmt.Sprintf("  Add Labels: %s\n", strings.Join(filter.Action.AddLabelIDs, ", ")))
	}
	if len(filter.Action.RemoveLabelIDs) > 0 {
		result.WriteString(fmt.Sprintf("  Remove Labels: %s\n", strings.Join(filter.Action.RemoveLabelIDs, ", ")))
	}
	if filter.Action.Archive {
		result.WriteString("  Archive: true\n")
	}
	if filter.Action.MarkAsRead {
		result.WriteString("  Mark as Read: true\n")
	}
	if filter.Action.Star {
		result.WriteString("  Star: true\n")
	}
	if filter.Action.MarkAsSpam {
		result.WriteString("  Mark as Spam: true\n")
	}
	if filter.Action.Delete {
		result.WriteString("  Delete: true\n")
	}
	if filter.Action.Forward != "" {
		result.WriteString(fmt.Sprintf("  Forward to: %s\n", filter.Action.Forward))
	}

	return result.String()
}
