package cmd

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"

	mcpserver "github.com/mark3labs/mcp-go/server"

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
		debugMode bool
		transport string
		httpAddr  string
		yolo      bool
	)

	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Start the MCP server",
		Long: `Start the Model Context Protocol (MCP) server to provide Gmail and GitHub
integration tools for AI assistants.

Supports multiple transport types:
  - stdio: Standard input/output (default)
  - sse: Server-Sent Events over HTTP
  - streamable-http: Streamable HTTP transport

Safety Mode:
  By default, the server operates in read-only mode, providing only safe operations.
  Use --yolo to enable write operations (email sending, file deletion, etc.)`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runServe(transport, debugMode, httpAddr, yolo)
		},
	}

	cmd.Flags().BoolVar(&debugMode, "debug", false, "Enable debug logging")
	cmd.Flags().StringVar(&transport, "transport", "stdio", "Transport type: stdio, sse, or streamable-http")
	cmd.Flags().StringVar(&httpAddr, "http-addr", ":8080", "HTTP server address (for sse and streamable-http transports)")
	cmd.Flags().BoolVar(&yolo, "yolo", false, "Enable write operations (email sending, file deletion, etc.). Default is read-only mode.")

	return cmd
}

func runServe(transport string, debugMode bool, httpAddr string, yolo bool) error {
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

	// Create server context
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
	case "sse":
		fmt.Printf("Starting inboxfewer MCP server with %s transport...\n", transport)
		return runSSEServer(mcpSrv, httpAddr, shutdownCtx, debugMode)
	case "streamable-http":
		fmt.Printf("Starting inboxfewer MCP server with %s transport...\n", transport)
		return runStreamableHTTPServer(mcpSrv, httpAddr, shutdownCtx, debugMode)
	default:
		return fmt.Errorf("unsupported transport type: %s (supported: stdio, sse, streamable-http)", transport)
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

func runSSEServer(mcpSrv *mcpserver.MCPServer, addr string, ctx context.Context, debugMode bool) error {
	// Create OAuth-enabled SSE server
	// Base URL should be the full URL where the server is accessible
	// For development, use http://localhost:8080
	// For production, use the actual HTTPS URL
	baseURL := fmt.Sprintf("http://%s", addr)
	if addr[0] == ':' {
		baseURL = fmt.Sprintf("http://localhost%s", addr)
	}

	oauthServer, err := server.NewOAuthHTTPServer(mcpSrv, "sse", baseURL)
	if err != nil {
		return fmt.Errorf("failed to create OAuth SSE server: %w", err)
	}

	fmt.Printf("SSE server with Google OAuth authentication starting on %s\n", addr)
	fmt.Printf("  SSE endpoint: /sse\n")
	fmt.Printf("  Message endpoint: /message\n")
	fmt.Printf("  OAuth metadata: /.well-known/oauth-protected-resource\n")
	fmt.Printf("  Authorization Server: https://accounts.google.com\n")
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
		fmt.Println("Shutdown signal received, stopping SSE server...")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 30)
		defer cancel()
		if err := oauthServer.Shutdown(shutdownCtx); err != nil {
			return fmt.Errorf("error shutting down SSE server: %w", err)
		}
	case err := <-serverDone:
		if err != nil {
			return fmt.Errorf("SSE server stopped with error: %w", err)
		}
		fmt.Println("SSE server stopped normally")
	}

	fmt.Println("SSE server gracefully stopped")
	return nil
}

func runStreamableHTTPServer(mcpSrv *mcpserver.MCPServer, addr string, ctx context.Context, debugMode bool) error {
	// Create OAuth-enabled HTTP server
	// Base URL should be the full URL where the server is accessible
	// For development, use http://localhost:8080
	// For production, use the actual HTTPS URL
	baseURL := fmt.Sprintf("http://%s", addr)
	if addr[0] == ':' {
		baseURL = fmt.Sprintf("http://localhost%s", addr)
	}

	oauthServer, err := server.NewOAuthHTTPServer(mcpSrv, "streamable-http", baseURL)
	if err != nil {
		return fmt.Errorf("failed to create OAuth HTTP server: %w", err)
	}

	fmt.Printf("Streamable HTTP server with Google OAuth authentication starting on %s\n", addr)
	fmt.Printf("  HTTP endpoint: /mcp\n")
	fmt.Printf("  OAuth metadata: /.well-known/oauth-protected-resource\n")
	fmt.Printf("  Authorization Server: https://accounts.google.com\n")
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
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 30)
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
