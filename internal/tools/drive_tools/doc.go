// Package drive_tools provides MCP (Model Context Protocol) tools for Google Drive operations.
//
// This package exposes Drive functionality to MCP clients (like AI assistants) through
// a set of tools that handle file management, folder operations, and permission sharing.
//
// Available tools:
//   - drive_upload_file: Upload files to Google Drive with metadata
//   - drive_list_files: List and search files with filtering
//   - drive_get_file: Get metadata for a specific file
//   - drive_download_file: Download file content
//   - drive_delete_file: Delete files from Drive
//   - drive_create_folder: Create new folders
//   - drive_move_file: Move or rename files
//   - drive_share_file: Share files with specific permissions
//   - drive_list_permissions: List all permissions for a file
//   - drive_remove_permission: Remove a permission from a file
//
// All tools support multi-account functionality through an optional 'account' parameter,
// allowing management of multiple Google accounts simultaneously.
//
// Example tool usage:
//
//	drive_upload_file({
//	  account: "work",
//	  name: "report.pdf",
//	  content: "<base64-encoded-content>",
//	  mimeType: "application/pdf",
//	  parentFolders: ["folder_id"]
//	})
//
//	drive_list_files({
//	  account: "personal",
//	  query: "mimeType='application/pdf' and name contains 'invoice'",
//	  maxResults: 10
//	})
package drive_tools
