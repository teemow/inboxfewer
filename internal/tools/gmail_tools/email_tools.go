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

// RegisterEmailTools registers email-related tools with the MCP server
func RegisterEmailTools(s *mcpserver.MCPServer, sc *server.ServerContext) error {
	// Send email tool
	sendEmailTool := mcp.NewTool("gmail_send_email",
		mcp.WithDescription("Send an email through Gmail"),
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

	s.AddTool(sendEmailTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleSendEmail(ctx, request, sc)
	})

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

	// Get or create Gmail client
	client := sc.GmailClient()
	if client == nil {
		if !gmail.HasToken() {
			authURL := gmail.GetAuthURL()
			errorMsg := fmt.Sprintf(`Google OAuth token not found. To authorize access:

1. Visit this URL in your browser:
   %s

2. Sign in with your Google account
3. Grant access to Google services (Gmail, Docs, Drive, Contacts)
4. Copy the authorization code

5. Provide the authorization code to your AI agent
   The agent will use the google_save_auth_code tool to complete authentication.

Note: You only need to authorize once. The tokens will be automatically refreshed.`, authURL)
			return mcp.NewToolResultError(errorMsg), nil
		}

		var err error
		client, err = gmail.NewClient(ctx)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to create Gmail client: %v", err)), nil
		}
		sc.SetGmailClient(client)
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
