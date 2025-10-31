// Package server provides the MCP (Model Context Protocol) server context.
//
// This package manages the lifecycle and state of the MCP server, including:
//   - Context management with cancellation support
//   - Multi-account client management for all Google services
//   - Lazy initialization of clients on first use
//   - Thread-safe access to shared resources
//   - GitHub credentials management
//
// The ServerContext maintains separate client instances for each account across
// all supported Google services: Gmail, Google Docs, Google Drive, Google Calendar,
// Google Meet, and Google Tasks. Clients are created lazily when first requested
// and cached for subsequent use.
//
// Multi-Account Support:
// Each Google service client can be associated with a specific account (e.g., "work",
// "personal"). The "default" account is used when no specific account is specified.
//
// Example usage:
//
//	ctx := context.Background()
//	serverCtx, err := server.NewServerContext(ctx, githubUser, githubToken)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer serverCtx.Shutdown()
//
//	// Get Gmail client for default account
//	gmailClient := serverCtx.GmailClient()
//
//	// Get Calendar client for a specific account
//	calendarClient := serverCtx.CalendarClientForAccount("work")
package server
