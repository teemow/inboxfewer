package docs_tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"

	"github.com/teemow/inboxfewer/internal/docs"
	"github.com/teemow/inboxfewer/internal/google"
	"github.com/teemow/inboxfewer/internal/server"
	"github.com/teemow/inboxfewer/internal/tools/batch"
	"github.com/teemow/inboxfewer/internal/tools/common"
)

// RegisterDocsTools registers all Google Docs-related tools with the MCP server
func RegisterDocsTools(s *mcpserver.MCPServer, sc *server.ServerContext) error {
	// Get documents tool
	getDocumentsTool := mcp.NewTool("docs_get_documents",
		mcp.WithDescription("Get Google Docs content for one or more documents"),
		mcp.WithString("account",
			mcp.Description("Account name (default: 'default'). Used to manage multiple Google accounts."),
		),
		mcp.WithString("documentIds",
			mcp.Required(),
			mcp.Description("Document ID (string) or array of document IDs"),
		),
		mcp.WithString("format",
			mcp.Description("Output format: 'markdown' (default), 'text', or 'json'"),
		),
	)

	s.AddTool(getDocumentsTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleGetDocuments(ctx, request, sc)
	})

	// Get document metadata tool
	getMetadataTool := mcp.NewTool("docs_get_documents_metadata",
		mcp.WithDescription("Get metadata about one or more Google Docs or Drive files"),
		mcp.WithString("account",
			mcp.Description("Account name (default: 'default'). Used to manage multiple Google accounts."),
		),
		mcp.WithString("documentIds",
			mcp.Required(),
			mcp.Description("Document ID (string) or array of document IDs"),
		),
	)

	s.AddTool(getMetadataTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleGetDocumentsMetadata(ctx, request, sc)
	})

	return nil
}

func handleGetDocuments(ctx context.Context, request mcp.CallToolRequest, sc *server.ServerContext) (*mcp.CallToolResult, error) {
	args := request.GetArguments()
	account := common.GetAccountFromArgs(ctx, args)

	documentIDs, err := batch.ParseStringOrArray(args["documentIds"], "documentIds")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	format := "markdown"
	if formatVal, ok := args["format"].(string); ok && formatVal != "" {
		format = formatVal
	}

	// Get or create docs client for the specified account
	docsClient := sc.DocsClientForAccount(account)
	if docsClient == nil {
		// Check if token exists before trying to create client
		if !docs.HasTokenForAccount(account) {
			errorMsg := google.GetAuthenticationErrorMessage(account)
			return mcp.NewToolResultError(errorMsg), nil
		}

		var err error
		docsClient, err = docs.NewClientForAccount(ctx, account)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to create Docs client for account %s: %v", account, err)), nil
		}
		sc.SetDocsClientForAccount(account, docsClient)
	}

	results := batch.ProcessBatch(documentIDs, func(documentID string) (string, error) {
		switch format {
		case "markdown":
			content, err := docsClient.GetDocumentAsMarkdown(documentID)
			if err != nil {
				return "", err
			}
			return fmt.Sprintf("Markdown content (%d bytes):\n%s", len(content), content), nil

		case "text":
			content, err := docsClient.GetDocumentAsPlainText(documentID)
			if err != nil {
				return "", err
			}
			return fmt.Sprintf("Plain text content (%d bytes):\n%s", len(content), content), nil

		case "json":
			doc, err := docsClient.GetDocument(documentID)
			if err != nil {
				return "", err
			}
			jsonBytes, err := json.MarshalIndent(doc, "", "  ")
			if err != nil {
				return "", fmt.Errorf("failed to serialize: %w", err)
			}
			return fmt.Sprintf("JSON content (%d bytes):\n%s", len(jsonBytes), string(jsonBytes)), nil

		default:
			return "", fmt.Errorf("invalid format '%s', must be 'markdown', 'text', or 'json'", format)
		}
	})

	return mcp.NewToolResultText(batch.FormatResults(results)), nil
}

func handleGetDocumentsMetadata(ctx context.Context, request mcp.CallToolRequest, sc *server.ServerContext) (*mcp.CallToolResult, error) {
	args := request.GetArguments()
	account := common.GetAccountFromArgs(ctx, args)

	documentIDs, err := batch.ParseStringOrArray(args["documentIds"], "documentIds")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	// Get or create docs client for the specified account
	docsClient := sc.DocsClientForAccount(account)
	if docsClient == nil {
		// Check if token exists before trying to create client
		if !docs.HasTokenForAccount(account) {
			errorMsg := google.GetAuthenticationErrorMessage(account)
			return mcp.NewToolResultError(errorMsg), nil
		}

		var err error
		docsClient, err = docs.NewClientForAccount(ctx, account)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to create Docs client for account %s: %v", account, err)), nil
		}
		sc.SetDocsClientForAccount(account, docsClient)
	}

	results := batch.ProcessBatch(documentIDs, func(documentID string) (string, error) {
		metadata, err := docsClient.GetFileMetadata(documentID)
		if err != nil {
			return "", err
		}

		jsonBytes, err := json.MarshalIndent(metadata, "", "  ")
		if err != nil {
			return "", fmt.Errorf("failed to serialize: %w", err)
		}

		return fmt.Sprintf("Document metadata:\n%s", string(jsonBytes)), nil
	})

	return mcp.NewToolResultText(batch.FormatResults(results)), nil
}
