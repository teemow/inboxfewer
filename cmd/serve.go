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
	"github.com/teemow/inboxfewer/internal/tools/gmail_tools"
	"github.com/teemow/inboxfewer/internal/tools/google_tools"
	"github.com/teemow/inboxfewer/internal/tools/meet_tools"
)

func newServeCmd() *cobra.Command {
	var (
		debugMode bool
		transport string
		httpAddr  string
	)

	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Start the MCP server",
		Long: `Start the Model Context Protocol (MCP) server to provide Gmail and GitHub
integration tools for AI assistants.

Supports multiple transport types:
  - stdio: Standard input/output (default)
  - sse: Server-Sent Events over HTTP
  - streamable-http: Streamable HTTP transport`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runServe(transport, debugMode, httpAddr)
		},
	}

	cmd.Flags().BoolVar(&debugMode, "debug", false, "Enable debug logging")
	cmd.Flags().StringVar(&transport, "transport", "stdio", "Transport type: stdio, sse, or streamable-http")
	cmd.Flags().StringVar(&httpAddr, "http-addr", ":8080", "HTTP server address (for sse and streamable-http transports)")

	return cmd
}

func runServe(transport string, debugMode bool, httpAddr string) error {
	// Setup graceful shutdown
	shutdownCtx, cancel := signal.NotifyContext(context.Background(),
		os.Interrupt, syscall.SIGTERM)
	defer cancel()

	// Read GitHub config
	if err := readGithubConfig(); err != nil {
		return fmt.Errorf("failed to read GitHub config: %w", err)
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

	// Register Google OAuth tools
	if err := google_tools.RegisterGoogleTools(mcpSrv, serverContext); err != nil {
		return fmt.Errorf("failed to register Google OAuth tools: %w", err)
	}

	// Register Gmail tools
	if err := gmail_tools.RegisterGmailTools(mcpSrv, serverContext); err != nil {
		return fmt.Errorf("failed to register Gmail tools: %w", err)
	}

	// Register Docs tools
	if err := docs_tools.RegisterDocsTools(mcpSrv, serverContext); err != nil {
		return fmt.Errorf("failed to register Docs tools: %w", err)
	}

	// Register Calendar tools
	if err := calendar_tools.RegisterCalendarTools(mcpSrv, serverContext); err != nil {
		return fmt.Errorf("failed to register Calendar tools: %w", err)
	}

	// Register Meet tools
	if err := meet_tools.RegisterMeetTools(mcpSrv, serverContext); err != nil {
		return fmt.Errorf("failed to register Meet tools: %w", err)
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
	sseServer := mcpserver.NewSSEServer(mcpSrv,
		mcpserver.WithSSEEndpoint("/sse"),
		mcpserver.WithMessageEndpoint("/message"),
	)

	fmt.Printf("SSE server starting on %s\n", addr)
	fmt.Printf("  SSE endpoint: /sse\n")
	fmt.Printf("  Message endpoint: /message\n")

	serverDone := make(chan error, 1)
	go func() {
		defer close(serverDone)
		if err := sseServer.Start(addr); err != nil {
			serverDone <- err
		}
	}()

	select {
	case <-ctx.Done():
		fmt.Println("Shutdown signal received, stopping SSE server...")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 30)
		defer cancel()
		if err := sseServer.Shutdown(shutdownCtx); err != nil {
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
	httpServer := mcpserver.NewStreamableHTTPServer(mcpSrv,
		mcpserver.WithEndpointPath("/mcp"),
	)

	fmt.Printf("Streamable HTTP server starting on %s\n", addr)
	fmt.Printf("  HTTP endpoint: /mcp\n")

	serverDone := make(chan error, 1)
	go func() {
		defer close(serverDone)
		if err := httpServer.Start(addr); err != nil {
			serverDone <- err
		}
	}()

	select {
	case <-ctx.Done():
		fmt.Println("Shutdown signal received, stopping HTTP server...")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 30)
		defer cancel()
		if err := httpServer.Shutdown(shutdownCtx); err != nil {
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
