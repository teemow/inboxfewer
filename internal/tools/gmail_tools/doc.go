// Package gmail_tools provides MCP (Model Context Protocol) tools for interacting with Gmail.
//
// This package exposes Gmail functionality through MCP tools that can be called by
// AI agents or other MCP clients. It provides capabilities for:
//
// Thread Management:
//   - gmail_list_threads: List Gmail threads matching a search query
//   - gmail_archive_threads: Archive threads by removing them from inbox
//   - gmail_unarchive_threads: Move archived threads back to inbox
//   - gmail_mark_threads_as_spam: Mark threads as spam and remove from inbox
//   - gmail_unmark_threads_as_spam: Remove spam label and move threads back to inbox
//   - gmail_classify_thread: Classify threads based on GitHub issue/PR references
//   - gmail_check_stale: Check if a thread is stale (linked issue/PR is closed)
//   - gmail_archive_stale_threads: Bulk archive stale threads
//
// Attachment Management:
//   - gmail_list_attachments: List all attachments in a message
//   - gmail_get_attachment: Retrieve attachment content (base64 or text)
//   - gmail_get_message_body: Extract text or HTML body from a message
//
// All tools require an authenticated Gmail client which is provided through the
// server context. The client handles OAuth2 authentication and token management.
//
// Example usage of attachment tools:
//
//	// List attachments in a message
//	gmail_list_attachments(messageId: "msg123")
//
//	// Get attachment content as base64
//	gmail_get_attachment(messageId: "msg123", attachmentId: "att456", encoding: "base64")
//
//	// Get attachment content as text (for .ics, .txt, etc.)
//	gmail_get_attachment(messageId: "msg123", attachmentId: "att456", encoding: "text")
//
//	// Extract message body
//	gmail_get_message_body(messageId: "msg123", format: "text")
//
// Security Considerations:
//   - Attachment size is limited to 25MB (MaxAttachmentSize)
//   - Filenames are sanitized to prevent path traversal attacks
//   - OAuth2 tokens are securely stored and refreshed automatically
//
// The package follows the project's architecture principles:
//   - Dependency injection for testability
//   - Error wrapping with context
//   - Proper GoDoc documentation for all exported functions
package gmail_tools
