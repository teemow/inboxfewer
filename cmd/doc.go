// Package cmd implements the command-line interface for inboxfewer.
//
// This package provides the following commands:
//   - cleanup: Archive Gmail threads related to closed GitHub issues and pull requests
//   - serve: Start the MCP server to provide tools for AI assistants
//   - version: Display version information
//   - generate-docs: Generate markdown documentation for all MCP tools
//
// The cleanup command is the default command when no subcommand is specified.
package cmd
