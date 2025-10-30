package gmail_tools

import (
	"context"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"
	gmail_v1 "google.golang.org/api/gmail/v1"

	"github.com/teemow/inboxfewer/internal/gmail"
	"github.com/teemow/inboxfewer/internal/server"
)

// RegisterGmailTools registers all Gmail-related tools with the MCP server
func RegisterGmailTools(s *mcpserver.MCPServer, sc *server.ServerContext) error {
	// Register attachment tools
	if err := RegisterAttachmentTools(s, sc); err != nil {
		return fmt.Errorf("failed to register attachment tools: %w", err)
	}

	// Register contact tools
	if err := RegisterContactTools(s, sc); err != nil {
		return fmt.Errorf("failed to register contact tools: %w", err)
	}

	// Register email tools
	if err := RegisterEmailTools(s, sc); err != nil {
		return fmt.Errorf("failed to register email tools: %w", err)
	}

	// List threads tool
	listThreadsTool := mcp.NewTool("gmail_list_threads",
		mcp.WithDescription("List Gmail threads matching a query"),
		mcp.WithString("query",
			mcp.Required(),
			mcp.Description("Gmail search query (e.g., 'in:inbox', 'from:user@example.com')"),
		),
		mcp.WithNumber("maxResults",
			mcp.Description("Maximum number of results to return (default: 10)"),
		),
	)

	s.AddTool(listThreadsTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleListThreads(ctx, request, sc)
	})

	// Archive thread tool
	archiveThreadTool := mcp.NewTool("gmail_archive_thread",
		mcp.WithDescription("Archive a Gmail thread by removing it from the inbox"),
		mcp.WithString("threadId",
			mcp.Required(),
			mcp.Description("The ID of the thread to archive"),
		),
	)

	s.AddTool(archiveThreadTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleArchiveThread(ctx, request, sc)
	})

	// Classify thread tool
	classifyThreadTool := mcp.NewTool("gmail_classify_thread",
		mcp.WithDescription("Classify a Gmail thread to determine if it's related to GitHub issues or PRs"),
		mcp.WithString("threadId",
			mcp.Required(),
			mcp.Description("The ID of the thread to classify"),
		),
	)

	s.AddTool(classifyThreadTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleClassifyThread(ctx, request, sc)
	})

	// Check stale tool
	checkStaleTool := mcp.NewTool("gmail_check_stale",
		mcp.WithDescription("Check if a Gmail thread is stale (GitHub issue/PR is closed)"),
		mcp.WithString("threadId",
			mcp.Required(),
			mcp.Description("The ID of the thread to check"),
		),
	)

	s.AddTool(checkStaleTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleCheckStale(ctx, request, sc)
	})

	// Archive stale threads tool
	archiveStaleTool := mcp.NewTool("gmail_archive_stale_threads",
		mcp.WithDescription("Archive all Gmail threads in inbox that are related to closed GitHub issues/PRs"),
		mcp.WithString("query",
			mcp.Description("Gmail search query (default: 'in:inbox')"),
		),
	)

	s.AddTool(archiveStaleTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleArchiveStaleThreads(ctx, request, sc)
	})

	return nil
}

func handleListThreads(ctx context.Context, request mcp.CallToolRequest, sc *server.ServerContext) (*mcp.CallToolResult, error) {
	args := request.GetArguments()

	query, ok := args["query"].(string)
	if !ok || query == "" {
		return mcp.NewToolResultError("query is required"), nil
	}

	maxResults := int64(10)
	if maxResultsVal, ok := args["maxResults"]; ok {
		if maxResultsFloat, ok := maxResultsVal.(float64); ok {
			maxResults = int64(maxResultsFloat)
		}
	}

	// Get or create Gmail client
	client := sc.GmailClient()
	if client == nil {
		// Check if token exists before trying to create client
		if !gmail.HasToken() {
			authURL := gmail.GetAuthURL()
			errorMsg := fmt.Sprintf(`Google OAuth token not found. To authorize access:

1. Visit this URL in your browser:
   %s

2. Sign in with your Google account
3. Grant access to Google services (Gmail, Docs, Drive)
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

	threads, err := client.ListThreads(query, maxResults)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to list threads: %v", err)), nil
	}

	result := fmt.Sprintf("Found %d threads:\n", len(threads))
	for i, thread := range threads {
		result += fmt.Sprintf("%d. Thread ID: %s (Snippet: %s)\n", i+1, thread.Id, thread.Snippet)
	}

	return mcp.NewToolResultText(result), nil
}

func handleArchiveThread(ctx context.Context, request mcp.CallToolRequest, sc *server.ServerContext) (*mcp.CallToolResult, error) {
	args := request.GetArguments()

	threadID, ok := args["threadId"].(string)
	if !ok || threadID == "" {
		return mcp.NewToolResultError("threadId is required"), nil
	}

	// Get or create Gmail client
	client := sc.GmailClient()
	if client == nil {
		if !gmail.HasToken() {
			authURL := gmail.GetAuthURL()
			errorMsg := fmt.Sprintf(`Google OAuth token not found. To authorize access:

1. Visit this URL in your browser:
   %s

2. Sign in with your Google account
3. Grant access to Google services (Gmail, Docs, Drive)
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

	if err := client.ArchiveThread(threadID); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to archive thread: %v", err)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Successfully archived thread %s", threadID)), nil
}

func handleClassifyThread(ctx context.Context, request mcp.CallToolRequest, sc *server.ServerContext) (*mcp.CallToolResult, error) {
	args := request.GetArguments()

	threadID, ok := args["threadId"].(string)
	if !ok || threadID == "" {
		return mcp.NewToolResultError("threadId is required"), nil
	}

	// Get or create Gmail client
	client := sc.GmailClient()
	if client == nil {
		if !gmail.HasToken() {
			authURL := gmail.GetAuthURL()
			errorMsg := fmt.Sprintf(`Google OAuth token not found. To authorize access:

1. Visit this URL in your browser:
   %s

2. Sign in with your Google account
3. Grant access to Google services (Gmail, Docs, Drive)
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

	thread := &gmail_v1.Thread{Id: threadID}
	if err := client.PopulateThread(thread); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to get thread: %v", err)), nil
	}

	classification := gmail.ClassifyThread(thread, sc.GithubUser(), sc.GithubToken())
	if classification == nil {
		return mcp.NewToolResultText("Thread is not related to GitHub issues or PRs"), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Thread classification: %T - %+v", classification, classification)), nil
}

func handleCheckStale(ctx context.Context, request mcp.CallToolRequest, sc *server.ServerContext) (*mcp.CallToolResult, error) {
	args := request.GetArguments()

	threadID, ok := args["threadId"].(string)
	if !ok || threadID == "" {
		return mcp.NewToolResultError("threadId is required"), nil
	}

	// Get or create Gmail client
	client := sc.GmailClient()
	if client == nil {
		if !gmail.HasToken() {
			authURL := gmail.GetAuthURL()
			errorMsg := fmt.Sprintf(`Google OAuth token not found. To authorize access:

1. Visit this URL in your browser:
   %s

2. Sign in with your Google account
3. Grant access to Google services (Gmail, Docs, Drive)
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

	thread := &gmail_v1.Thread{Id: threadID}
	if err := client.PopulateThread(thread); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to get thread: %v", err)), nil
	}

	classification := gmail.ClassifyThread(thread, sc.GithubUser(), sc.GithubToken())
	if classification == nil {
		return mcp.NewToolResultText("Thread is not related to GitHub issues or PRs"), nil
	}

	stale, err := classification.IsStale()
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to check if thread is stale: %v", err)), nil
	}

	if stale {
		return mcp.NewToolResultText(fmt.Sprintf("Thread is STALE (GitHub issue/PR is closed): %+v", classification)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Thread is ACTIVE (GitHub issue/PR is open): %+v", classification)), nil
}

func handleArchiveStaleThreads(ctx context.Context, request mcp.CallToolRequest, sc *server.ServerContext) (*mcp.CallToolResult, error) {
	args := request.GetArguments()

	query := "in:inbox"
	if queryVal, ok := args["query"].(string); ok && queryVal != "" {
		query = queryVal
	}

	// Get or create Gmail client
	client := sc.GmailClient()
	if client == nil {
		if !gmail.HasToken() {
			authURL := gmail.GetAuthURL()
			errorMsg := fmt.Sprintf(`Google OAuth token not found. To authorize access:

1. Visit this URL in your browser:
   %s

2. Sign in with your Google account
3. Grant access to Google services (Gmail, Docs, Drive)
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

	archived := 0
	checked := 0

	err := client.ForeachThread(query, func(t *gmail_v1.Thread) error {
		if err := client.PopulateThread(t); err != nil {
			return err
		}

		classification := gmail.ClassifyThread(t, sc.GithubUser(), sc.GithubToken())
		checked++

		if classification == nil {
			return nil
		}

		stale, err := classification.IsStale()
		if err != nil {
			return err
		}

		if stale {
			if err := client.ArchiveThread(t.Id); err != nil {
				return err
			}
			archived++
		}

		return nil
	})

	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to archive stale threads: %v", err)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Checked %d threads, archived %d stale threads", checked, archived)), nil
}
