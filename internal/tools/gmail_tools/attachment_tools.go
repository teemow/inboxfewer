package gmail_tools

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"
	"github.com/teemow/inboxfewer/internal/drive"
	"github.com/teemow/inboxfewer/internal/gmail"
	"github.com/teemow/inboxfewer/internal/google"
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
		mcp.WithDescription("Extract text or HTML body from one or more Gmail messages or threads. Accepts both Message IDs and Thread IDs."),
		mcp.WithString("account",
			mcp.Description("Account name (default: 'default'). Used to manage multiple Google accounts."),
		),
		mcp.WithString("messageIds",
			mcp.Required(),
			mcp.Description("Message ID or Thread ID (string) or array of IDs. Thread IDs will automatically fetch all messages in the thread."),
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

	// Transfer attachments to Drive tool
	transferAttachmentsTool := mcp.NewTool("gmail_transfer_attachments_to_drive",
		mcp.WithDescription("Transfer Gmail attachments directly to Google Drive in a single operation"),
		mcp.WithString("account",
			mcp.Description("Account name (default: 'default'). Used to manage multiple Google accounts."),
		),
		mcp.WithString("messageId",
			mcp.Required(),
			mcp.Description("The ID of the Gmail message containing the attachments"),
		),
		mcp.WithString("attachmentIds",
			mcp.Required(),
			mcp.Description("Attachment identifier(s) to transfer. Can be: attachment ID from gmail_list_attachments, exact filename, or numeric index (0, 1, 2, etc.). Supports single string or array of strings."),
		),
		mcp.WithString("parentFolders",
			mcp.Description("Comma-separated list of parent folder IDs in Google Drive where files should be placed"),
		),
		mcp.WithString("description",
			mcp.Description("Optional description to add to the files in Google Drive"),
		),
	)

	s.AddTool(transferAttachmentsTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleTransferAttachmentsToDrive(ctx, request, sc)
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
			authURL := google.GetAuthenticationErrorMessage(account)
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
			authURL := google.GetAuthenticationErrorMessage(account)
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
			authURL := google.GetAuthenticationErrorMessage(account)
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
			authURL := google.GetAuthenticationErrorMessage(account)
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

// handleTransferAttachmentsToDrive transfers Gmail attachments directly to Google Drive
// This handler fetches attachment(s) from Gmail and uploads them to Drive in a single operation,
// preserving the original filename and MIME type. Supports batch processing for multiple attachments.
//
// Attachment matching uses a fallback strategy:
// 1. Try matching by attachment ID (from gmail_list_attachments)
// 2. If not found, try matching by exact filename
// 3. If still not found, try parsing as numeric index (0, 1, 2, etc.)
//
// This multi-strategy approach ensures robustness against Gmail API attachment ID inconsistencies.
func handleTransferAttachmentsToDrive(ctx context.Context, request mcp.CallToolRequest, sc *server.ServerContext) (*mcp.CallToolResult, error) {
	args := request.GetArguments()

	messageID, ok := args["messageId"].(string)
	if !ok || messageID == "" {
		return mcp.NewToolResultError("messageId is required"), nil
	}

	// Parse attachmentIds - can be string or array
	attachmentIDs, err := batch.ParseStringOrArray(args["attachmentIds"], "attachmentIds")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	// Get account name
	account := getAccountFromArgs(args)

	// Get or create Gmail client
	gmailClient := sc.GmailClientForAccount(account)
	if gmailClient == nil {
		if !gmail.HasTokenForAccount(account) {
			errorMsg := google.GetAuthenticationErrorMessage(account)
			return mcp.NewToolResultError(errorMsg), nil
		}

		var err error
		gmailClient, err = gmail.NewClientForAccount(ctx, account)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to create Gmail client: %v", err)), nil
		}
		sc.SetGmailClientForAccount(account, gmailClient)
	}

	// Get or create Drive client
	driveClient := sc.DriveClientForAccount(account)
	if driveClient == nil {
		if !drive.HasTokenForAccount(account) {
			errorMsg := google.GetAuthenticationErrorMessage(account)
			return mcp.NewToolResultError(errorMsg), nil
		}

		var err error
		driveClient, err = drive.NewClientForAccount(ctx, account)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to create Drive client: %v", err)), nil
		}
		sc.SetDriveClientForAccount(account, driveClient)
	}

	// Parse optional parent folders
	var parentFolders []string
	if parentFoldersStr, ok := args["parentFolders"].(string); ok && parentFoldersStr != "" {
		for _, folder := range strings.Split(parentFoldersStr, ",") {
			folder = strings.TrimSpace(folder)
			if folder != "" {
				parentFolders = append(parentFolders, folder)
			}
		}
	}

	// Get optional description
	description := ""
	if desc, ok := args["description"].(string); ok {
		description = desc
	}

	// First, get all attachments for the message to map IDs to metadata
	allAttachments, err := gmailClient.ListAttachments(messageID)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to list attachments: %v", err)), nil
	}

	// Create a map of attachment ID to attachment info
	// Trim attachment IDs to handle any whitespace issues
	attachmentMap := make(map[string]*gmail.AttachmentInfo)
	for _, att := range allAttachments {
		attachmentMap[strings.TrimSpace(att.AttachmentID)] = att
	}

	// Process each attachment
	results := batch.ProcessBatch(attachmentIDs, func(attachmentID string) (string, error) {
		// Trim whitespace from attachment ID to handle any formatting issues
		attachmentID = strings.TrimSpace(attachmentID)

		// Strategy 1: Try to match by AttachmentID
		attInfo, ok := attachmentMap[attachmentID]

		// Strategy 2: If not found by ID, try matching by filename
		if !ok {
			for _, att := range allAttachments {
				if att.Filename == attachmentID {
					attInfo = att
					ok = true
					break
				}
			}
		}

		// Strategy 3: If still not found, check if it's an index number (0, 1, 2, etc.)
		if !ok {
			// Try parsing as index
			if idx, err := fmt.Sscanf(attachmentID, "%d", new(int)); err == nil && idx == 1 {
				var index int
				fmt.Sscanf(attachmentID, "%d", &index)
				if index >= 0 && index < len(allAttachments) {
					attInfo = allAttachments[index]
					ok = true
				}
			}
		}

		if !ok {
			// Build a detailed error message for debugging
			var availableInfo []string
			for i, att := range allAttachments {
				availableInfo = append(availableInfo, fmt.Sprintf("[%d] ID=%s, Name=%s", i, att.AttachmentID, att.Filename))
			}
			return "", fmt.Errorf("attachment '%s' not found in message %s. Available attachments:\n%s",
				attachmentID, messageID, strings.Join(availableInfo, "\n"))
		}

		// Fetch attachment data from Gmail using the actual attachment ID
		data, err := gmailClient.GetAttachment(messageID, attInfo.AttachmentID)
		if err != nil {
			return "", fmt.Errorf("failed to fetch attachment: %w", err)
		}

		// Prepare upload options
		uploadOpts := &drive.UploadOptions{
			MimeType:      attInfo.MimeType,
			ParentFolders: parentFolders,
		}
		if description != "" {
			uploadOpts.Description = description
		}

		// Upload to Drive
		fileInfo, err := driveClient.UploadFile(ctx, attInfo.Filename, bytes.NewReader(data), uploadOpts)
		if err != nil {
			return "", fmt.Errorf("failed to upload to Drive: %w", err)
		}

		// Return formatted result
		result := map[string]interface{}{
			"filename":     fileInfo.Name,
			"driveFileId":  fileInfo.ID,
			"size":         fileInfo.Size,
			"sizeHuman":    formatSize(fileInfo.Size),
			"mimeType":     fileInfo.MimeType,
			"webViewLink":  fileInfo.WebViewLink,
			"attachmentId": attInfo.AttachmentID,
			"identifier":   attachmentID, // What the user provided (ID, filename, or index)
		}

		jsonBytes, _ := json.Marshal(result)
		return string(jsonBytes), nil
	})

	return mcp.NewToolResultText(batch.FormatResults(results)), nil
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
