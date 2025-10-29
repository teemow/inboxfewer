package docs_tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"

	"github.com/teemow/inboxfewer/internal/docs"
	"github.com/teemow/inboxfewer/internal/server"
)

// RegisterDocsTools registers all Google Docs-related tools with the MCP server
func RegisterDocsTools(s *mcpserver.MCPServer, sc *server.ServerContext) error {
	// Get OAuth URL tool
	getAuthURLTool := mcp.NewTool("docs_get_auth_url",
		mcp.WithDescription("Get the OAuth URL to authorize Google Docs access"),
	)

	s.AddTool(getAuthURLTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleGetAuthURL(ctx, request, sc)
	})

	// Save authorization code tool
	saveAuthCodeTool := mcp.NewTool("docs_save_auth_code",
		mcp.WithDescription("Save the OAuth authorization code to complete Google Docs authentication"),
		mcp.WithString("authCode",
			mcp.Required(),
			mcp.Description("The authorization code from Google OAuth"),
		),
	)

	s.AddTool(saveAuthCodeTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleSaveAuthCode(ctx, request, sc)
	})

	// Get document tool
	getDocumentTool := mcp.NewTool("docs_get_document",
		mcp.WithDescription("Get Google Docs content by document ID"),
		mcp.WithString("documentId",
			mcp.Required(),
			mcp.Description("The ID of the Google Doc"),
		),
		mcp.WithString("format",
			mcp.Description("Output format: 'markdown' (default), 'text', or 'json'"),
		),
	)

	s.AddTool(getDocumentTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleGetDocument(ctx, request, sc)
	})

	// Get document metadata tool
	getMetadataTool := mcp.NewTool("docs_get_document_metadata",
		mcp.WithDescription("Get metadata about a Google Doc or Drive file"),
		mcp.WithString("documentId",
			mcp.Required(),
			mcp.Description("The ID of the Google Doc or Drive file"),
		),
	)

	s.AddTool(getMetadataTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleGetMetadata(ctx, request, sc)
	})

	return nil
}

func handleGetDocument(ctx context.Context, request mcp.CallToolRequest, sc *server.ServerContext) (*mcp.CallToolResult, error) {
	args := request.GetArguments()

	documentID, ok := args["documentId"].(string)
	if !ok || documentID == "" {
		return mcp.NewToolResultError("documentId is required"), nil
	}

	format := "markdown"
	if formatVal, ok := args["format"].(string); ok && formatVal != "" {
		format = formatVal
	}

	// Get or create docs client
	docsClient := sc.DocsClient()
	if docsClient == nil {
		// Check if token exists before trying to create client
		if !docs.HasToken() {
			authURL := docs.GetAuthURL()
			errorMsg := fmt.Sprintf(`Google Docs OAuth token not found. To authorize access:

1. Visit this URL in your browser:
   %s

2. Sign in with your Google account
3. Grant access to read documents
4. Copy the authorization code

5. Provide the authorization code to your AI agent
   The agent will use the docs_save_auth_code tool to complete authentication.

Note: You only need to authorize once. The tokens will be automatically refreshed.`, authURL)
			return mcp.NewToolResultError(errorMsg), nil
		}

		var err error
		docsClient, err = docs.NewClient(ctx)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to create Docs client: %v", err)), nil
		}
		sc.SetDocsClient(docsClient)
	}

	switch format {
	case "markdown":
		content, err := docsClient.GetDocumentAsMarkdown(documentID)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to get document: %v", err)), nil
		}
		result := fmt.Sprintf("Document content (Markdown, %d bytes):\n%s", len(content), content)
		return mcp.NewToolResultText(result), nil

	case "text":
		content, err := docsClient.GetDocumentAsPlainText(documentID)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to get document: %v", err)), nil
		}
		result := fmt.Sprintf("Document content (plain text, %d bytes):\n%s", len(content), content)
		return mcp.NewToolResultText(result), nil

	case "json":
		doc, err := docsClient.GetDocument(documentID)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to get document: %v", err)), nil
		}
		jsonBytes, err := json.MarshalIndent(doc, "", "  ")
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to serialize document: %v", err)), nil
		}
		result := fmt.Sprintf("Document content (JSON, %d bytes):\n%s", len(jsonBytes), string(jsonBytes))
		return mcp.NewToolResultText(result), nil

	default:
		return mcp.NewToolResultError(fmt.Sprintf("Invalid format '%s', must be 'markdown', 'text', or 'json'", format)), nil
	}
}

func handleGetMetadata(ctx context.Context, request mcp.CallToolRequest, sc *server.ServerContext) (*mcp.CallToolResult, error) {
	args := request.GetArguments()

	documentID, ok := args["documentId"].(string)
	if !ok || documentID == "" {
		return mcp.NewToolResultError("documentId is required"), nil
	}

	// Get or create docs client
	docsClient := sc.DocsClient()
	if docsClient == nil {
		// Check if token exists before trying to create client
		if !docs.HasToken() {
			authURL := docs.GetAuthURL()
			errorMsg := fmt.Sprintf(`Google Docs OAuth token not found. To authorize access:

1. Visit this URL in your browser:
   %s

2. Sign in with your Google account
3. Grant access to read documents
4. Copy the authorization code

5. Provide the authorization code to your AI agent
   The agent will use the docs_save_auth_code tool to complete authentication.

Note: You only need to authorize once. The tokens will be automatically refreshed.`, authURL)
			return mcp.NewToolResultError(errorMsg), nil
		}

		var err error
		docsClient, err = docs.NewClient(ctx)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to create Docs client: %v", err)), nil
		}
		sc.SetDocsClient(docsClient)
	}

	metadata, err := docsClient.GetFileMetadata(documentID)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to get metadata: %v", err)), nil
	}

	jsonBytes, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to serialize metadata: %v", err)), nil
	}

	result := fmt.Sprintf("Document metadata:\n%s", string(jsonBytes))
	return mcp.NewToolResultText(result), nil
}

func handleGetAuthURL(ctx context.Context, request mcp.CallToolRequest, sc *server.ServerContext) (*mcp.CallToolResult, error) {
	authURL := docs.GetAuthURL()

	result := fmt.Sprintf(`To authorize Google Docs access:

1. Visit this URL in your browser:
   %s

2. Sign in with your Google account
3. Grant access to read documents
4. Copy the authorization code

5. Call the docs_save_auth_code tool with the code to complete authentication`, authURL)

	return mcp.NewToolResultText(result), nil
}

func handleSaveAuthCode(ctx context.Context, request mcp.CallToolRequest, sc *server.ServerContext) (*mcp.CallToolResult, error) {
	args := request.GetArguments()

	authCode, ok := args["authCode"].(string)
	if !ok || authCode == "" {
		return mcp.NewToolResultError("authCode is required"), nil
	}

	err := docs.SaveToken(ctx, authCode)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to save authorization code: %v", err)), nil
	}

	return mcp.NewToolResultText("✅ Authorization successful! Google Docs token saved. You can now use the docs_get_document and docs_get_document_metadata tools."), nil
}
