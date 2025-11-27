// Package resources provides MCP resources for exposing user and session data.
// Resources are read-only data sources that MCP clients can fetch, such as
// user profiles, settings, and other contextual information.
//
// Session-specific resources (new in mcp-go 0.42.0/0.43.0):
// Resources can be scoped to specific sessions, allowing multi-user/multi-account
// scenarios where each user sees their own data.
package resources
