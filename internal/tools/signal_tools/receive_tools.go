package signal_tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"

	"github.com/teemow/inboxfewer/internal/server"
)

// RegisterReceiveTools registers Signal message receiving tools
func RegisterReceiveTools(s *mcpserver.MCPServer, sc *server.ServerContext) error {
	// Receive message tool
	receiveMessageTool := mcp.NewTool("signal_receive_message",
		mcp.WithDescription("Wait for and receive a Signal message with timeout support"),
		mcp.WithString("account",
			mcp.Description("Account name (default: 'default'). Used to manage multiple Signal accounts."),
		),
		mcp.WithNumber("timeout",
			mcp.Required(),
			mcp.Description("Timeout in seconds to wait for a message (e.g., 30)"),
		),
	)

	s.AddTool(receiveMessageTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleReceiveMessage(ctx, request, sc)
	})

	return nil
}

// handleReceiveMessage handles the signal_receive_message tool
func handleReceiveMessage(ctx context.Context, request mcp.CallToolRequest, sc *server.ServerContext) (*mcp.CallToolResult, error) {
	// Type assert arguments to map
	args, ok := request.Params.Arguments.(map[string]interface{})
	if !ok {
		return mcp.NewToolResultError("Invalid arguments"), nil
	}

	// Get account from arguments
	account := getAccountFromArgs(args)

	// Get timeout
	timeout, ok := args["timeout"].(float64)
	if !ok || timeout <= 0 {
		return mcp.NewToolResultError("Missing or invalid 'timeout' parameter (must be a positive number)"), nil
	}

	// Get Signal client
	client, err := getSignalClient(ctx, sc, account)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	// Receive message
	response, err := client.ReceiveMessage(int(timeout))
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to receive message: %v", err)), nil
	}

	// Check if an error occurred during receive
	if response.Error != "" {
		return mcp.NewToolResultError(response.Error), nil
	}

	// If no message was received
	if response.Message == "" {
		return mcp.NewToolResultText("No message received within timeout"), nil
	}

	// Format the response as JSON for structured output
	jsonResponse, err := json.MarshalIndent(response, "", "  ")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to format response: %v", err)), nil
	}

	// Create a formatted text response
	result := fmt.Sprintf("Received message from %s", response.SenderID)
	if response.GroupName != "" {
		result += fmt.Sprintf(" in group '%s'", response.GroupName)
	}
	result += fmt.Sprintf(":\n\n%s\n\nFull response:\n%s", response.Message, string(jsonResponse))

	return mcp.NewToolResultText(result), nil
}
