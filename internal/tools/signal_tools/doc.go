// Package signal_tools provides MCP tools for Signal messaging operations.
//
// This package exposes Signal messaging capabilities through the Model Context Protocol (MCP),
// allowing AI agents to send and receive Signal messages. It wraps the signal client package
// and provides the following tools:
//
//   - signal_send_message: Send a direct message to a Signal user
//   - signal_send_group_message: Send a message to a Signal group
//   - signal_receive_message: Receive messages with timeout support
//   - signal_list_groups: List all Signal groups the user is a member of
//
// Prerequisites:
// Users must have signal-cli installed and configured before using these tools.
// See the signal package documentation for setup instructions.
//
// Account Support:
// Similar to other tools in inboxfewer, Signal tools support an optional 'account'
// parameter to manage multiple Signal accounts (phone numbers).
//
// Example MCP tool call:
//
//	{
//	  "tool": "signal_send_message",
//	  "arguments": {
//	    "recipient": "+15559876543",
//	    "message": "Hello from Signal MCP!",
//	    "account": "default"
//	  }
//	}
package signal_tools
