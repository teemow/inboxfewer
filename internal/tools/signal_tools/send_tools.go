package signal_tools

import (
	"context"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"

	"github.com/teemow/inboxfewer/internal/server"
)

// RegisterSendTools registers Signal message sending tools
func RegisterSendTools(s *mcpserver.MCPServer, sc *server.ServerContext, readOnly bool) error {
	// Send message to user tool
	sendMessageTool := mcp.NewTool("signal_send_message",
		mcp.WithDescription("Send a direct message to a Signal user"),
		mcp.WithString("account",
			mcp.Description("Account name (default: 'default'). Used to manage multiple Signal accounts."),
		),
		mcp.WithString("recipient",
			mcp.Required(),
			mcp.Description("Phone number of the recipient (e.g., '+15559876543')"),
		),
		mcp.WithString("message",
			mcp.Required(),
			mcp.Description("Text message to send"),
		),
	)

	// Only register if not in read-only mode
	if readOnly {
		// In read-only mode, return an informational response
		s.AddTool(sendMessageTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return mcp.NewToolResultError("Cannot send messages in read-only mode. Use --yolo flag to enable write operations."), nil
		})
	} else {
		s.AddTool(sendMessageTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return handleSendMessage(ctx, request, sc)
		})
	}

	// Send group message tool
	sendGroupMessageTool := mcp.NewTool("signal_send_group_message",
		mcp.WithDescription("Send a message to a Signal group"),
		mcp.WithString("account",
			mcp.Description("Account name (default: 'default'). Used to manage multiple Signal accounts."),
		),
		mcp.WithString("group_name",
			mcp.Required(),
			mcp.Description("Name of the Signal group"),
		),
		mcp.WithString("message",
			mcp.Required(),
			mcp.Description("Text message to send"),
		),
	)

	// Only register if not in read-only mode
	if readOnly {
		s.AddTool(sendGroupMessageTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return mcp.NewToolResultError("Cannot send messages in read-only mode. Use --yolo flag to enable write operations."), nil
		})
	} else {
		s.AddTool(sendGroupMessageTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return handleSendGroupMessage(ctx, request, sc)
		})
	}

	return nil
}

// handleSendMessage handles the signal_send_message tool
func handleSendMessage(ctx context.Context, request mcp.CallToolRequest, sc *server.ServerContext) (*mcp.CallToolResult, error) {
	// Type assert arguments to map
	args, ok := request.Params.Arguments.(map[string]interface{})
	if !ok {
		return mcp.NewToolResultError("Invalid arguments"), nil
	}

	// Get account from arguments
	account := getAccountFromArgs(args)

	// Get recipient
	recipient, ok := args["recipient"].(string)
	if !ok || recipient == "" {
		return mcp.NewToolResultError("Missing or invalid 'recipient' parameter"), nil
	}

	// Get message
	message, ok := args["message"].(string)
	if !ok || message == "" {
		return mcp.NewToolResultError("Missing or invalid 'message' parameter"), nil
	}

	// Get Signal client
	client, err := getSignalClient(ctx, sc, account)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	// Send message
	err = client.SendMessage(recipient, message)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to send message: %v", err)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Successfully sent message to %s", recipient)), nil
}

// handleSendGroupMessage handles the signal_send_group_message tool
func handleSendGroupMessage(ctx context.Context, request mcp.CallToolRequest, sc *server.ServerContext) (*mcp.CallToolResult, error) {
	// Type assert arguments to map
	args, ok := request.Params.Arguments.(map[string]interface{})
	if !ok {
		return mcp.NewToolResultError("Invalid arguments"), nil
	}

	// Get account from arguments
	account := getAccountFromArgs(args)

	// Get group name
	groupName, ok := args["group_name"].(string)
	if !ok || groupName == "" {
		return mcp.NewToolResultError("Missing or invalid 'group_name' parameter"), nil
	}

	// Get message
	message, ok := args["message"].(string)
	if !ok || message == "" {
		return mcp.NewToolResultError("Missing or invalid 'message' parameter"), nil
	}

	// Get Signal client
	client, err := getSignalClient(ctx, sc, account)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	// Send group message
	err = client.SendGroupMessage(groupName, message)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to send group message: %v", err)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Successfully sent message to group '%s'", groupName)), nil
}
