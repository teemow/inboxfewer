package cmd

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	mcpserver "github.com/mark3labs/mcp-go/server"

	"github.com/teemow/inboxfewer/internal/mcp/oauth"
	"github.com/teemow/inboxfewer/internal/server"
	"github.com/teemow/inboxfewer/internal/tools/calendar_tools"
	"github.com/teemow/inboxfewer/internal/tools/docs_tools"
	"github.com/teemow/inboxfewer/internal/tools/drive_tools"
	"github.com/teemow/inboxfewer/internal/tools/gmail_tools"
	"github.com/teemow/inboxfewer/internal/tools/meet_tools"
	"github.com/teemow/inboxfewer/internal/tools/tasks_tools"
)

func newServeCmd() *cobra.Command {
	var (
		debugMode          bool
		transport          string
		httpAddr           string
		yolo               bool
		googleClientID     string
		googleClientSecret string
	)

	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Start the MCP server",
		Long: `Start the Model Context Protocol (MCP) server to provide Gmail and GitHub
integration tools for AI assistants.

Supports multiple transport types:
  - stdio: Standard input/output (default)
  - streamable-http: Streamable HTTP transport

Safety Mode:
  By default, the server operates in read-only mode, providing only safe operations.
  Use --yolo to enable write operations (email sending, file deletion, etc.)

OAuth Token Refresh (HTTP only):
  For automatic token refresh, provide Google OAuth credentials via:
    --google-client-id and --google-client-secret flags
  OR via environment variables:
    GOOGLE_CLIENT_ID and GOOGLE_CLIENT_SECRET
  Without these, users will need to re-authenticate when tokens expire (~1 hour).`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runServe(transport, debugMode, httpAddr, yolo, googleClientID, googleClientSecret)
		},
	}

	cmd.Flags().BoolVar(&debugMode, "debug", false, "Enable debug logging")
	cmd.Flags().StringVar(&transport, "transport", "stdio", "Transport type: stdio or streamable-http")
	cmd.Flags().StringVar(&httpAddr, "http-addr", ":8080", "HTTP server address (for streamable-http transport)")
	cmd.Flags().BoolVar(&yolo, "yolo", false, "Enable write operations (email sending, file deletion, etc.). Default is read-only mode.")
	cmd.Flags().StringVar(&googleClientID, "google-client-id", "", "Google OAuth Client ID for automatic token refresh (HTTP transport only). Can also use GOOGLE_CLIENT_ID env var.")
	cmd.Flags().StringVar(&googleClientSecret, "google-client-secret", "", "Google OAuth Client Secret for automatic token refresh (HTTP transport only). Can also use GOOGLE_CLIENT_SECRET env var.")

	return cmd
}

func runServe(transport string, debugMode bool, httpAddr string, yolo bool, googleClientID, googleClientSecret string) error {
	// Setup graceful shutdown
	shutdownCtx, cancel := signal.NotifyContext(context.Background(),
		os.Interrupt, syscall.SIGTERM)
	defer cancel()

	// Read GitHub config (optional for serve mode - will use empty strings if not available)
	// Users can authenticate via OAuth for MCP server usage
	if err := readGithubConfig(); err != nil {
		// Log warning but continue - GitHub config is optional for MCP server
		if transport != "stdio" {
			log.Printf("Warning: GitHub config not found (this is OK for MCP server): %v", err)
		}
		// Set empty values - server will work without GitHub integration
		githubUser = ""
		githubToken = ""
	}

	// Get Google OAuth credentials from environment if not provided via flags
	if googleClientID == "" {
		googleClientID = os.Getenv("GOOGLE_CLIENT_ID")
	}
	if googleClientSecret == "" {
		googleClientSecret = os.Getenv("GOOGLE_CLIENT_SECRET")
	}

	// Create server context (will be recreated for HTTP with OAuth token provider)
	serverContext, err := server.NewServerContext(shutdownCtx, githubUser, githubToken)
	if err != nil {
		return fmt.Errorf("failed to create server context: %w", err)
	}
	defer func() {
		if err := serverContext.Shutdown(); err != nil {
			if transport != "stdio" {
				log.Printf("Error during server context shutdown: %v", err)
			}
		}
	}()

	// Create MCP server
	mcpSrv := mcpserver.NewMCPServer("inboxfewer", version,
		mcpserver.WithToolCapabilities(true),
	)

	// readOnly is the inverse of yolo
	readOnly := !yolo

	// Log the mode for visibility (only for non-stdio transports)
	if transport != "stdio" {
		if readOnly {
			log.Println("Starting server in READ-ONLY mode (use --yolo to enable write operations)")
		} else {
			log.Println("Starting server with WRITE operations enabled (--yolo flag is set)")
		}
	}

	// Register Gmail tools
	if err := gmail_tools.RegisterGmailTools(mcpSrv, serverContext, readOnly); err != nil {
		return fmt.Errorf("failed to register Gmail tools: %w", err)
	}

	// Register Docs tools (read-only by nature)
	if err := docs_tools.RegisterDocsTools(mcpSrv, serverContext); err != nil {
		return fmt.Errorf("failed to register Docs tools: %w", err)
	}

	// Register Drive tools
	if err := drive_tools.RegisterDriveTools(mcpSrv, serverContext, readOnly); err != nil {
		return fmt.Errorf("failed to register Drive tools: %w", err)
	}

	// Register Calendar tools
	if err := calendar_tools.RegisterCalendarTools(mcpSrv, serverContext, readOnly); err != nil {
		return fmt.Errorf("failed to register Calendar tools: %w", err)
	}

	// Register Meet tools
	if err := meet_tools.RegisterMeetTools(mcpSrv, serverContext, readOnly); err != nil {
		return fmt.Errorf("failed to register Meet tools: %w", err)
	}

	// Register Tasks tools
	if err := tasks_tools.RegisterTasksTools(mcpSrv, serverContext, readOnly); err != nil {
		return fmt.Errorf("failed to register Tasks tools: %w", err)
	}

	// Start the appropriate server based on transport type
	switch transport {
	case "stdio":
		return runStdioServer(mcpSrv)
	case "streamable-http":
		fmt.Printf("Starting inboxfewer MCP server with %s transport...\n", transport)
		return runStreamableHTTPServer(mcpSrv, serverContext, httpAddr, shutdownCtx, debugMode, googleClientID, googleClientSecret, readOnly)
	default:
		return fmt.Errorf("unsupported transport type: %s (supported: stdio, streamable-http)", transport)
	}
}

func runStdioServer(mcpSrv *mcpserver.MCPServer) error {
	serverDone := make(chan error, 1)
	go func() {
		defer close(serverDone)
		if err := mcpserver.ServeStdio(mcpSrv); err != nil {
			serverDone <- err
		}
	}()

	err := <-serverDone
	if err != nil {
		return fmt.Errorf("server stopped with error: %w", err)
	}
	return nil
}

func runStreamableHTTPServer(mcpSrv *mcpserver.MCPServer, oldServerContext *server.ServerContext, addr string, ctx context.Context, debugMode bool, googleClientID, googleClientSecret string, readOnly bool) error {
	// Create OAuth-enabled HTTP server
	// Base URL should be the full URL where the server is accessible
	// For development, use http://localhost:8080
	// For production, use the actual HTTPS URL
	baseURL := fmt.Sprintf("http://%s", addr)
	if addr[0] == ':' {
		baseURL = fmt.Sprintf("http://localhost%s", addr)
	}

	// Create OAuth handler first so we can inject its token provider
	oauthConfig := server.OAuthConfig{
		BaseURL:            baseURL,
		GoogleClientID:     googleClientID,
		GoogleClientSecret: googleClientSecret,
	}

	oauthHandler, err := server.CreateOAuthHandler(oauthConfig)
	if err != nil {
		return fmt.Errorf("failed to create OAuth handler: %w", err)
	}

	// Create token provider from OAuth store
	tokenProvider := oauth.NewTokenProvider(oauthHandler.GetStore())

	// Recreate server context with OAuth token provider
	// This ensures Google API clients use tokens from OAuth authentication
	githubUser := oldServerContext.GithubUser()
	githubToken := oldServerContext.GithubToken()

	// Shutdown old context and create new one with OAuth token provider
	if err := oldServerContext.Shutdown(); err != nil {
		log.Printf("Warning: failed to shutdown old server context: %v", err)
	}

	serverContext, err := server.NewServerContextWithProvider(ctx, githubUser, githubToken, tokenProvider)
	if err != nil {
		return fmt.Errorf("failed to create server context with OAuth token provider: %w", err)
	}
	defer func() {
		if err := serverContext.Shutdown(); err != nil {
			log.Printf("Error during server context shutdown: %v", err)
		}
	}()

	// Re-register all tools with the new context
	if err := gmail_tools.RegisterGmailTools(mcpSrv, serverContext, readOnly); err != nil {
		return fmt.Errorf("failed to register Gmail tools: %w", err)
	}
	if err := drive_tools.RegisterDriveTools(mcpSrv, serverContext, readOnly); err != nil {
		return fmt.Errorf("failed to register Drive tools: %w", err)
	}
	if err := docs_tools.RegisterDocsTools(mcpSrv, serverContext); err != nil {
		return fmt.Errorf("failed to register Docs tools: %w", err)
	}
	if err := calendar_tools.RegisterCalendarTools(mcpSrv, serverContext, readOnly); err != nil {
		return fmt.Errorf("failed to register Calendar tools: %w", err)
	}
	if err := meet_tools.RegisterMeetTools(mcpSrv, serverContext, readOnly); err != nil {
		return fmt.Errorf("failed to register Meet tools: %w", err)
	}
	if err := tasks_tools.RegisterTasksTools(mcpSrv, serverContext, readOnly); err != nil {
		return fmt.Errorf("failed to register Tasks tools: %w", err)
	}

	// Create OAuth server with existing handler
	oauthServer, err := server.NewOAuthHTTPServerWithHandler(mcpSrv, "streamable-http", oauthHandler)
	if err != nil {
		return fmt.Errorf("failed to create OAuth HTTP server: %w", err)
	}

	fmt.Printf("Streamable HTTP server with Google OAuth authentication starting on %s\n", addr)
	fmt.Printf("  HTTP endpoint: /mcp\n")
	fmt.Printf("  OAuth metadata: /.well-known/oauth-protected-resource\n")
	fmt.Printf("  Authorization Server: https://accounts.google.com\n")

	if oauthHandler.CanRefreshTokens() {
		fmt.Println("\n✓ Automatic token refresh: ENABLED")
		fmt.Println("  Tokens will be refreshed automatically before expiration")
	} else {
		fmt.Println("\n⚠ Automatic token refresh: DISABLED")
		fmt.Println("  Users will need to re-authenticate when tokens expire (~1 hour)")
		fmt.Println("  To enable, provide --google-client-id and --google-client-secret")
	}

	fmt.Println("\nClients must authenticate with Google OAuth to access this server.")
	fmt.Println("The MCP client (e.g., Cursor, Claude Desktop) will handle the OAuth flow automatically.")

	serverDone := make(chan error, 1)
	go func() {
		defer close(serverDone)
		if err := oauthServer.Start(addr); err != nil {
			serverDone <- err
		}
	}()

	select {
	case <-ctx.Done():
		fmt.Println("Shutdown signal received, stopping HTTP server...")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		if err := oauthServer.Shutdown(shutdownCtx); err != nil {
			return fmt.Errorf("error shutting down HTTP server: %w", err)
		}
	case err := <-serverDone:
		if err != nil {
			return fmt.Errorf("HTTP server stopped with error: %w", err)
		}
		fmt.Println("HTTP server stopped normally")
	}

	fmt.Println("HTTP server gracefully stopped")
	return nil
}
