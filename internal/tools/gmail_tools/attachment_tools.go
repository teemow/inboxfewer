package gmail_tools

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"
	"github.com/teemow/inboxfewer/internal/gmail"
	"github.com/teemow/inboxfewer/internal/server"
	"github.com/teemow/inboxfewer/internal/tools/batch"
)

// RegisterAttachmentTools registers attachment-related tools with the MCP server
func RegisterAttachmentTools(s *mcpserver.MCPServer, sc *server.ServerContext) error {
	// List attachments tool
	listAttachmentsTool := mcp.NewTool("gmail_list_attachments",
		mcp.WithDescription("List all attachments in a Gmail message"),
		mcp.WithString("account",
			mcp.Description("Account name (default: 'default'). Used to manage multiple Google accounts."),
		),
		mcp.WithString("messageId",
			mcp.Required(),
			mcp.Description("The ID of the Gmail message"),
		),
	)

	s.AddTool(listAttachmentsTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleListAttachments(ctx, request, sc)
	})

	// Get attachment tool
	getAttachmentTool := mcp.NewTool("gmail_get_attachment",
		mcp.WithDescription("Get the content of an attachment"),
		mcp.WithString("account",
			mcp.Description("Account name (default: 'default'). Used to manage multiple Google accounts."),
		),
		mcp.WithString("messageId",
			mcp.Required(),
			mcp.Description("The ID of the Gmail message"),
		),
		mcp.WithString("attachmentId",
			mcp.Required(),
			mcp.Description("The ID of the attachment"),
		),
		mcp.WithString("encoding",
			mcp.Description("Encoding format: 'base64' (default) or 'text'"),
		),
	)

	s.AddTool(getAttachmentTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleGetAttachment(ctx, request, sc)
	})

	// Get message bodies tool
	getMessageBodiesTool := mcp.NewTool("gmail_get_message_bodies",
		mcp.WithDescription("Extract text or HTML body from one or more Gmail messages"),
		mcp.WithString("account",
			mcp.Description("Account name (default: 'default'). Used to manage multiple Google accounts."),
		),
		mcp.WithString("messageIds",
			mcp.Required(),
			mcp.Description("Message ID (string) or array of message IDs"),
		),
		mcp.WithString("format",
			mcp.Description("Body format: 'text' (default) or 'html'"),
		),
	)

	s.AddTool(getMessageBodiesTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleGetMessageBodies(ctx, request, sc)
	})

	// Extract doc links tool
	extractDocLinksTool := mcp.NewTool("gmail_extract_doc_links",
		mcp.WithDescription("Extract Google Docs/Drive links from a Gmail message"),
		mcp.WithString("account",
			mcp.Description("Account name (default: 'default'). Used to manage multiple Google accounts."),
		),
		mcp.WithString("messageId",
			mcp.Required(),
			mcp.Description("The ID of the Gmail message"),
		),
		mcp.WithString("format",
			mcp.Description("Body format to search: 'text' (default) or 'html'"),
		),
	)

	s.AddTool(extractDocLinksTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleExtractDocLinks(ctx, request, sc)
	})

	return nil
}

func handleListAttachments(ctx context.Context, request mcp.CallToolRequest, sc *server.ServerContext) (*mcp.CallToolResult, error) {
	args := request.GetArguments()

	messageID, ok := args["messageId"].(string)
	if !ok || messageID == "" {
		return mcp.NewToolResultError("messageId is required"), nil
	}

	// Get or create Gmail client
	account := getAccountFromArgs(args)
	client := sc.GmailClientForAccount(account)
	if client == nil {
		if !gmail.HasTokenForAccount(account) {
			authURL := gmail.GetAuthURLForAccount(account)
			errorMsg := fmt.Sprintf(`Gmail OAuth token not found. To authorize access:

1. Visit this URL in your browser:
   %s

2. Sign in with your Google account
3. Grant access to Gmail
4. Copy the authorization code

5. Provide the authorization code to your AI agent
   The agent will use the google_save_auth_code tool to complete authentication.

Note: You only need to authorize once. The tokens will be automatically refreshed.`, authURL)
			return mcp.NewToolResultError(errorMsg), nil
		}

		var err error
		client, err = gmail.NewClientForAccount(ctx, account)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to create Gmail client: %v", err)), nil
		}
		sc.SetGmailClientForAccount(account, client)
	}

	attachments, err := client.ListAttachments(messageID)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to list attachments: %v", err)), nil
	}

	if len(attachments) == 0 {
		return mcp.NewToolResultText("No attachments found in message"), nil
	}

	// Convert attachments to JSON for structured output
	type attachmentOutput struct {
		AttachmentID string `json:"attachmentId"`
		Filename     string `json:"filename"`
		MimeType     string `json:"mimeType"`
		Size         int64  `json:"size"`
		SizeHuman    string `json:"sizeHuman"`
	}

	outputs := make([]attachmentOutput, len(attachments))
	for i, att := range attachments {
		outputs[i] = attachmentOutput{
			AttachmentID: att.AttachmentID,
			Filename:     att.Filename,
			MimeType:     att.MimeType,
			Size:         att.Size,
			SizeHuman:    formatSize(att.Size),
		}
	}

	jsonBytes, err := json.MarshalIndent(outputs, "", "  ")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to format output: %v", err)), nil
	}

	result := fmt.Sprintf("Found %d attachment(s):\n%s", len(attachments), string(jsonBytes))
	return mcp.NewToolResultText(result), nil
}

func handleGetAttachment(ctx context.Context, request mcp.CallToolRequest, sc *server.ServerContext) (*mcp.CallToolResult, error) {
	args := request.GetArguments()

	messageID, ok := args["messageId"].(string)
	if !ok || messageID == "" {
		return mcp.NewToolResultError("messageId is required"), nil
	}

	attachmentID, ok := args["attachmentId"].(string)
	if !ok || attachmentID == "" {
		return mcp.NewToolResultError("attachmentId is required"), nil
	}

	encoding := "base64"
	if encodingVal, ok := args["encoding"].(string); ok && encodingVal != "" {
		encoding = encodingVal
	}

	// Get or create Gmail client
	account := getAccountFromArgs(args)
	client := sc.GmailClientForAccount(account)
	if client == nil {
		if !gmail.HasTokenForAccount(account) {
			authURL := gmail.GetAuthURLForAccount(account)
			errorMsg := fmt.Sprintf(`Gmail OAuth token not found. To authorize access:

1. Visit this URL in your browser:
   %s

2. Sign in with your Google account
3. Grant access to Gmail
4. Copy the authorization code

5. Provide the authorization code to your AI agent
   The agent will use the google_save_auth_code tool to complete authentication.

Note: You only need to authorize once. The tokens will be automatically refreshed.`, authURL)
			return mcp.NewToolResultError(errorMsg), nil
		}

		var err error
		client, err = gmail.NewClientForAccount(ctx, account)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to create Gmail client: %v", err)), nil
		}
		sc.SetGmailClientForAccount(account, client)
	}

	switch encoding {
	case "base64":
		data, err := client.GetAttachment(messageID, attachmentID)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to get attachment: %v", err)), nil
		}

		encoded := base64.StdEncoding.EncodeToString(data)
		result := fmt.Sprintf("Attachment content (base64, %d bytes):\n%s", len(data), encoded)
		return mcp.NewToolResultText(result), nil

	case "text":
		text, err := client.GetAttachmentAsString(messageID, attachmentID)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to get attachment: %v", err)), nil
		}

		result := fmt.Sprintf("Attachment content (text, %d bytes):\n%s", len(text), text)
		return mcp.NewToolResultText(result), nil

	default:
		return mcp.NewToolResultError(fmt.Sprintf("Invalid encoding '%s', must be 'base64' or 'text'", encoding)), nil
	}
}

func handleGetMessageBodies(ctx context.Context, request mcp.CallToolRequest, sc *server.ServerContext) (*mcp.CallToolResult, error) {
	args := request.GetArguments()

	messageIDs, err := batch.ParseStringOrArray(args["messageIds"], "messageIds")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	format := "text"
	if formatVal, ok := args["format"].(string); ok && formatVal != "" {
		format = formatVal
	}

	// Get or create Gmail client
	account := getAccountFromArgs(args)
	client := sc.GmailClientForAccount(account)
	if client == nil {
		if !gmail.HasTokenForAccount(account) {
			authURL := gmail.GetAuthURLForAccount(account)
			errorMsg := fmt.Sprintf(`Gmail OAuth token not found. To authorize access:

1. Visit this URL in your browser:
   %s

2. Sign in with your Google account
3. Grant access to Gmail
4. Copy the authorization code

5. Provide the authorization code to your AI agent
   The agent will use the google_save_auth_code tool to complete authentication.

Note: You only need to authorize once. The tokens will be automatically refreshed.`, authURL)
			return mcp.NewToolResultError(errorMsg), nil
		}

		var err error
		client, err = gmail.NewClientForAccount(ctx, account)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to create Gmail client: %v", err)), nil
		}
		sc.SetGmailClientForAccount(account, client)
	}

	results := batch.ProcessBatch(messageIDs, func(messageID string) (string, error) {
		body, err := client.GetMessageBody(messageID, format)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("Message body (%s, %d bytes):\n%s", format, len(body), body), nil
	})

	return mcp.NewToolResultText(batch.FormatResults(results)), nil
}

func handleExtractDocLinks(ctx context.Context, request mcp.CallToolRequest, sc *server.ServerContext) (*mcp.CallToolResult, error) {
	args := request.GetArguments()

	messageID, ok := args["messageId"].(string)
	if !ok || messageID == "" {
		return mcp.NewToolResultError("messageId is required"), nil
	}

	format := "text"
	if formatVal, ok := args["format"].(string); ok && formatVal != "" {
		format = formatVal
	}

	// Get or create Gmail client
	account := getAccountFromArgs(args)
	client := sc.GmailClientForAccount(account)
	if client == nil {
		if !gmail.HasTokenForAccount(account) {
			authURL := gmail.GetAuthURLForAccount(account)
			errorMsg := fmt.Sprintf(`Gmail OAuth token not found. To authorize access:

1. Visit this URL in your browser:
   %s

2. Sign in with your Google account
3. Grant access to Gmail
4. Copy the authorization code

5. Provide the authorization code to your AI agent
   The agent will use the google_save_auth_code tool to complete authentication.

Note: You only need to authorize once. The tokens will be automatically refreshed.`, authURL)
			return mcp.NewToolResultError(errorMsg), nil
		}

		var err error
		client, err = gmail.NewClientForAccount(ctx, account)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to create Gmail client: %v", err)), nil
		}
		sc.SetGmailClientForAccount(account, client)
	}

	body, err := client.GetMessageBody(messageID, format)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to get message body: %v", err)), nil
	}

	// Extract doc links from the body
	docLinks := gmail.ExtractDocLinks(body)

	if len(docLinks) == 0 {
		return mcp.NewToolResultText("No Google Docs/Drive links found in message"), nil
	}

	// Convert to JSON for structured output
	jsonBytes, err := json.MarshalIndent(docLinks, "", "  ")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to format output: %v", err)), nil
	}

	result := fmt.Sprintf("Found %d Google Docs/Drive link(s):\n%s", len(docLinks), string(jsonBytes))
	return mcp.NewToolResultText(result), nil
}

// formatSize formats a byte size into human-readable format
func formatSize(bytes int64) string {
	const (
		KB = 1024
		MB = 1024 * KB
		GB = 1024 * MB
	)

	switch {
	case bytes >= GB:
		return fmt.Sprintf("%.2f GB", float64(bytes)/float64(GB))
	case bytes >= MB:
		return fmt.Sprintf("%.2f MB", float64(bytes)/float64(MB))
	case bytes >= KB:
		return fmt.Sprintf("%.2f KB", float64(bytes)/float64(KB))
	default:
		return fmt.Sprintf("%d bytes", bytes)
	}
}
