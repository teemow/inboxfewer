package gmail_tools

import (
	"context"
	"fmt"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"

	"github.com/teemow/inboxfewer/internal/gmail"
	"github.com/teemow/inboxfewer/internal/google"
	"github.com/teemow/inboxfewer/internal/server"
	"github.com/teemow/inboxfewer/internal/tools/common"
)

// RegisterEmailTools registers email-related tools with the MCP server
func RegisterEmailTools(s *mcpserver.MCPServer, sc *server.ServerContext, readOnly bool) error {
	// Only register write tools if not in read-only mode
	if readOnly {
		return nil
	}

	// Send email tool
	sendEmailTool := mcp.NewTool("gmail_send_email",
		mcp.WithDescription("Send an email through Gmail"),
		mcp.WithString("account",
			mcp.Description("Account name (default: 'default'). Used to manage multiple Google accounts."),
		),
		mcp.WithString("to",
			mcp.Required(),
			mcp.Description("Recipient email address(es), comma-separated for multiple recipients"),
		),
		mcp.WithString("subject",
			mcp.Required(),
			mcp.Description("Email subject"),
		),
		mcp.WithString("body",
			mcp.Required(),
			mcp.Description("Email body content"),
		),
		mcp.WithString("cc",
			mcp.Description("CC email address(es), comma-separated for multiple recipients"),
		),
		mcp.WithString("bcc",
			mcp.Description("BCC email address(es), comma-separated for multiple recipients"),
		),
		mcp.WithBoolean("isHTML",
			mcp.Description("Whether the body is HTML (default: false for plain text)"),
		),
	)

	s.AddTool(sendEmailTool, common.InstrumentedToolHandlerWithService(
		"gmail_send_email", "gmail", "send", sc,
		func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return handleSendEmail(ctx, request, sc)
		}))

	// Reply to email tool
	replyToEmailTool := mcp.NewTool("gmail_reply_to_email",
		mcp.WithDescription("Reply to an existing email message in a thread"),
		mcp.WithString("account",
			mcp.Description("Account name (default: 'default'). Used to manage multiple Google accounts."),
		),
		mcp.WithString("messageId",
			mcp.Required(),
			mcp.Description("The ID of the message to reply to"),
		),
		mcp.WithString("threadId",
			mcp.Required(),
			mcp.Description("The ID of the email thread"),
		),
		mcp.WithString("body",
			mcp.Required(),
			mcp.Description("Reply body content"),
		),
		mcp.WithString("cc",
			mcp.Description("CC email address(es), comma-separated for multiple recipients"),
		),
		mcp.WithString("bcc",
			mcp.Description("BCC email address(es), comma-separated for multiple recipients"),
		),
		mcp.WithBoolean("isHTML",
			mcp.Description("Whether the body is HTML (default: false for plain text)"),
		),
	)

	s.AddTool(replyToEmailTool, common.InstrumentedToolHandlerWithService(
		"gmail_reply_to_email", "gmail", "send", sc,
		func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return handleReplyToEmail(ctx, request, sc)
		}))

	// Forward email tool
	forwardEmailTool := mcp.NewTool("gmail_forward_email",
		mcp.WithDescription("Forward an existing email message to new recipients"),
		mcp.WithString("account",
			mcp.Description("Account name (default: 'default'). Used to manage multiple Google accounts."),
		),
		mcp.WithString("messageId",
			mcp.Required(),
			mcp.Description("The ID of the message to forward"),
		),
		mcp.WithString("to",
			mcp.Required(),
			mcp.Description("Recipient email address(es), comma-separated for multiple recipients"),
		),
		mcp.WithString("additionalBody",
			mcp.Description("Additional message to add before the forwarded content"),
		),
		mcp.WithString("cc",
			mcp.Description("CC email address(es), comma-separated for multiple recipients"),
		),
		mcp.WithString("bcc",
			mcp.Description("BCC email address(es), comma-separated for multiple recipients"),
		),
		mcp.WithBoolean("isHTML",
			mcp.Description("Whether the body is HTML (default: false for plain text)"),
		),
	)

	s.AddTool(forwardEmailTool, common.InstrumentedToolHandlerWithService(
		"gmail_forward_email", "gmail", "send", sc,
		func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return handleForwardEmail(ctx, request, sc)
		}))

	return nil
}

func handleSendEmail(ctx context.Context, request mcp.CallToolRequest, sc *server.ServerContext) (*mcp.CallToolResult, error) {
	args := request.GetArguments()

	// Parse required fields
	toStr, ok := args["to"].(string)
	if !ok || toStr == "" {
		return mcp.NewToolResultError("'to' field is required"), nil
	}

	subject, ok := args["subject"].(string)
	if !ok || subject == "" {
		return mcp.NewToolResultError("'subject' field is required"), nil
	}

	body, ok := args["body"].(string)
	if !ok || body == "" {
		return mcp.NewToolResultError("'body' field is required"), nil
	}

	// Parse optional fields
	ccStr := ""
	if ccVal, ok := args["cc"].(string); ok {
		ccStr = ccVal
	}

	bccStr := ""
	if bccVal, ok := args["bcc"].(string); ok {
		bccStr = bccVal
	}

	isHTML := false
	if isHTMLVal, ok := args["isHTML"].(bool); ok {
		isHTML = isHTMLVal
	}

	// Split email addresses
	to := splitEmailAddresses(toStr)
	cc := splitEmailAddresses(ccStr)
	bcc := splitEmailAddresses(bccStr)

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

	// Create email message
	msg := &gmail.EmailMessage{
		To:      to,
		Cc:      cc,
		Bcc:     bcc,
		Subject: subject,
		Body:    body,
		IsHTML:  isHTML,
	}

	messageID, err := client.SendEmail(msg)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to send email: %v", err)), nil
	}

	result := fmt.Sprintf("Email sent successfully!\nMessage ID: %s\nTo: %s\nSubject: %s",
		messageID, strings.Join(to, ", "), subject)

	if len(cc) > 0 {
		result += fmt.Sprintf("\nCC: %s", strings.Join(cc, ", "))
	}
	if len(bcc) > 0 {
		result += fmt.Sprintf("\nBCC: %s", strings.Join(bcc, ", "))
	}

	return mcp.NewToolResultText(result), nil
}

// splitEmailAddresses splits a comma-separated string of email addresses
func splitEmailAddresses(addresses string) []string {
	if addresses == "" {
		return nil
	}

	parts := strings.Split(addresses, ",")
	result := make([]string, 0, len(parts))
	for _, addr := range parts {
		trimmed := strings.TrimSpace(addr)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

func handleReplyToEmail(ctx context.Context, request mcp.CallToolRequest, sc *server.ServerContext) (*mcp.CallToolResult, error) {
	args := request.GetArguments()

	// Parse required fields
	messageID, ok := args["messageId"].(string)
	if !ok || messageID == "" {
		return mcp.NewToolResultError("'messageId' field is required"), nil
	}

	threadID, ok := args["threadId"].(string)
	if !ok || threadID == "" {
		return mcp.NewToolResultError("'threadId' field is required"), nil
	}

	body, ok := args["body"].(string)
	if !ok || body == "" {
		return mcp.NewToolResultError("'body' field is required"), nil
	}

	// Parse optional fields
	ccStr := ""
	if ccVal, ok := args["cc"].(string); ok {
		ccStr = ccVal
	}

	bccStr := ""
	if bccVal, ok := args["bcc"].(string); ok {
		bccStr = bccVal
	}

	isHTML := false
	if isHTMLVal, ok := args["isHTML"].(bool); ok {
		isHTML = isHTMLVal
	}

	// Split email addresses
	cc := splitEmailAddresses(ccStr)
	bcc := splitEmailAddresses(bccStr)

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

	// Send the reply
	replyID, err := client.ReplyToEmail(messageID, threadID, body, cc, bcc, isHTML)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to send reply: %v", err)), nil
	}

	result := fmt.Sprintf("Reply sent successfully!\nMessage ID: %s\nThread ID: %s",
		replyID, threadID)

	if len(cc) > 0 {
		result += fmt.Sprintf("\nCC: %s", strings.Join(cc, ", "))
	}
	if len(bcc) > 0 {
		result += fmt.Sprintf("\nBCC: %s", strings.Join(bcc, ", "))
	}

	return mcp.NewToolResultText(result), nil
}

func handleForwardEmail(ctx context.Context, request mcp.CallToolRequest, sc *server.ServerContext) (*mcp.CallToolResult, error) {
	args := request.GetArguments()

	// Parse required fields
	messageID, ok := args["messageId"].(string)
	if !ok || messageID == "" {
		return mcp.NewToolResultError("'messageId' field is required"), nil
	}

	toStr, ok := args["to"].(string)
	if !ok || toStr == "" {
		return mcp.NewToolResultError("'to' field is required"), nil
	}

	// Parse optional fields
	additionalBody := ""
	if bodyVal, ok := args["additionalBody"].(string); ok {
		additionalBody = bodyVal
	}

	ccStr := ""
	if ccVal, ok := args["cc"].(string); ok {
		ccStr = ccVal
	}

	bccStr := ""
	if bccVal, ok := args["bcc"].(string); ok {
		bccStr = bccVal
	}

	isHTML := false
	if isHTMLVal, ok := args["isHTML"].(bool); ok {
		isHTML = isHTMLVal
	}

	// Split email addresses
	to := splitEmailAddresses(toStr)
	cc := splitEmailAddresses(ccStr)
	bcc := splitEmailAddresses(bccStr)

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

	// Forward the email
	forwardID, err := client.ForwardEmail(messageID, to, cc, bcc, additionalBody, isHTML)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to forward email: %v", err)), nil
	}

	result := fmt.Sprintf("Email forwarded successfully!\nMessage ID: %s\nTo: %s",
		forwardID, strings.Join(to, ", "))

	if len(cc) > 0 {
		result += fmt.Sprintf("\nCC: %s", strings.Join(cc, ", "))
	}
	if len(bcc) > 0 {
		result += fmt.Sprintf("\nBCC: %s", strings.Join(bcc, ", "))
	}

	return mcp.NewToolResultText(result), nil
}
