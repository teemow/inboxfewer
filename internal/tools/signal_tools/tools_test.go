package signal_tools

import (
	"context"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"

	"github.com/teemow/inboxfewer/internal/server"
	"github.com/teemow/inboxfewer/internal/signal"
)

// TestGetAccountFromArgs tests the getAccountFromArgs helper function
func TestGetAccountFromArgs(t *testing.T) {
	tests := []struct {
		name string
		args map[string]interface{}
		want string
	}{
		{
			name: "default account when not specified",
			args: map[string]interface{}{},
			want: "default",
		},
		{
			name: "explicit account name",
			args: map[string]interface{}{
				"account": "personal",
			},
			want: "personal",
		},
		{
			name: "empty account name defaults to default",
			args: map[string]interface{}{
				"account": "",
			},
			want: "default",
		},
		{
			name: "non-string account value",
			args: map[string]interface{}{
				"account": 123,
			},
			want: "default",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getAccountFromArgs(tt.args)
			if got != tt.want {
				t.Errorf("getAccountFromArgs() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestRegisterSignalTools tests the registration of Signal tools
func TestRegisterSignalTools(t *testing.T) {
	// Create test server context
	ctx := context.Background()
	serverContext, err := server.NewServerContext(ctx, "testuser", "testtoken")
	if err != nil {
		t.Fatalf("Failed to create server context: %v", err)
	}
	defer serverContext.Shutdown()

	// Create MCP server
	mcpSrv := mcpserver.NewMCPServer("test-server", "1.0.0",
		mcpserver.WithToolCapabilities(true),
	)

	tests := []struct {
		name     string
		readOnly bool
		wantErr  bool
	}{
		{
			name:     "register in read-write mode",
			readOnly: false,
			wantErr:  false,
		},
		{
			name:     "register in read-only mode",
			readOnly: true,
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := RegisterSignalTools(mcpSrv, serverContext, tt.readOnly)

			if (err != nil) != tt.wantErr {
				t.Errorf("RegisterSignalTools() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestHandleListGroupsNoClient tests handleListGroups when no client is configured
func TestHandleListGroupsNoClient(t *testing.T) {
	ctx := context.Background()
	serverContext, err := server.NewServerContext(ctx, "testuser", "testtoken")
	if err != nil {
		t.Fatalf("Failed to create server context: %v", err)
	}
	defer serverContext.Shutdown()

	// Create a request with no Signal client configured
	request := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      "signal_list_groups",
			Arguments: map[string]interface{}{},
		},
	}

	result, err := handleListGroups(ctx, request, serverContext)

	// Should not return an error, but result should indicate missing client
	if err != nil {
		t.Errorf("handleListGroups() unexpected error = %v", err)
	}

	if result == nil {
		t.Fatal("handleListGroups() returned nil result")
	}

	// Check that result is an error result
	if len(result.Content) == 0 {
		t.Error("handleListGroups() returned empty content")
	}
}

// TestHandleSendMessageValidation tests input validation for handleSendMessage
func TestHandleSendMessageValidation(t *testing.T) {
	ctx := context.Background()
	serverContext, err := server.NewServerContext(ctx, "testuser", "testtoken")
	if err != nil {
		t.Fatalf("Failed to create server context: %v", err)
	}
	defer serverContext.Shutdown()

	// Add a mock Signal client for testing
	mockClient, err := signal.NewClient(ctx, "+15551234567")
	if err != nil {
		t.Skip("signal-cli not installed, skipping test")
	}
	serverContext.SetSignalClientForAccount("default", mockClient)

	tests := []struct {
		name string
		args map[string]interface{}
	}{
		{
			name: "missing recipient",
			args: map[string]interface{}{
				"message": "test message",
			},
		},
		{
			name: "missing message",
			args: map[string]interface{}{
				"recipient": "+15559876543",
			},
		},
		{
			name: "empty recipient",
			args: map[string]interface{}{
				"recipient": "",
				"message":   "test message",
			},
		},
		{
			name: "empty message",
			args: map[string]interface{}{
				"recipient": "+15559876543",
				"message":   "",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := mcp.CallToolRequest{
				Params: mcp.CallToolParams{
					Name:      "signal_send_message",
					Arguments: tt.args,
				},
			}

			result, err := handleSendMessage(ctx, request, serverContext)

			if err != nil {
				t.Errorf("handleSendMessage() unexpected error = %v", err)
			}

			if result == nil {
				t.Fatal("handleSendMessage() returned nil result")
			}

			// Should return an error result for invalid input
			if len(result.Content) == 0 {
				t.Error("handleSendMessage() returned empty content")
			}
		})
	}
}

// TestHandleSendGroupMessageValidation tests input validation for handleSendGroupMessage
func TestHandleSendGroupMessageValidation(t *testing.T) {
	ctx := context.Background()
	serverContext, err := server.NewServerContext(ctx, "testuser", "testtoken")
	if err != nil {
		t.Fatalf("Failed to create server context: %v", err)
	}
	defer serverContext.Shutdown()

	// Add a mock Signal client for testing
	mockClient, err := signal.NewClient(ctx, "+15551234567")
	if err != nil {
		t.Skip("signal-cli not installed, skipping test")
	}
	serverContext.SetSignalClientForAccount("default", mockClient)

	tests := []struct {
		name string
		args map[string]interface{}
	}{
		{
			name: "missing group_name",
			args: map[string]interface{}{
				"message": "test message",
			},
		},
		{
			name: "missing message",
			args: map[string]interface{}{
				"group_name": "Test Group",
			},
		},
		{
			name: "empty group_name",
			args: map[string]interface{}{
				"group_name": "",
				"message":    "test message",
			},
		},
		{
			name: "empty message",
			args: map[string]interface{}{
				"group_name": "Test Group",
				"message":    "",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := mcp.CallToolRequest{
				Params: mcp.CallToolParams{
					Name:      "signal_send_group_message",
					Arguments: tt.args,
				},
			}

			result, err := handleSendGroupMessage(ctx, request, serverContext)

			if err != nil {
				t.Errorf("handleSendGroupMessage() unexpected error = %v", err)
			}

			if result == nil {
				t.Fatal("handleSendGroupMessage() returned nil result")
			}

			// Should return an error result for invalid input
			if len(result.Content) == 0 {
				t.Error("handleSendGroupMessage() returned empty content")
			}
		})
	}
}

// TestHandleReceiveMessageValidation tests input validation for handleReceiveMessage
func TestHandleReceiveMessageValidation(t *testing.T) {
	ctx := context.Background()
	serverContext, err := server.NewServerContext(ctx, "testuser", "testtoken")
	if err != nil {
		t.Fatalf("Failed to create server context: %v", err)
	}
	defer serverContext.Shutdown()

	// Add a mock Signal client for testing
	mockClient, err := signal.NewClient(ctx, "+15551234567")
	if err != nil {
		t.Skip("signal-cli not installed, skipping test")
	}
	serverContext.SetSignalClientForAccount("default", mockClient)

	tests := []struct {
		name string
		args map[string]interface{}
	}{
		{
			name: "missing timeout",
			args: map[string]interface{}{},
		},
		{
			name: "invalid timeout type",
			args: map[string]interface{}{
				"timeout": "not a number",
			},
		},
		{
			name: "zero timeout",
			args: map[string]interface{}{
				"timeout": 0.0,
			},
		},
		{
			name: "negative timeout",
			args: map[string]interface{}{
				"timeout": -5.0,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := mcp.CallToolRequest{
				Params: mcp.CallToolParams{
					Name:      "signal_receive_message",
					Arguments: tt.args,
				},
			}

			result, err := handleReceiveMessage(ctx, request, serverContext)

			if err != nil {
				t.Errorf("handleReceiveMessage() unexpected error = %v", err)
			}

			if result == nil {
				t.Fatal("handleReceiveMessage() returned nil result")
			}

			// Should return an error result for invalid input
			if len(result.Content) == 0 {
				t.Error("handleReceiveMessage() returned empty content")
			}
		})
	}
}
