// Package google_tools provides MCP tools for Google OAuth authentication.
//
// This package registers OAuth-related tools that allow AI assistants to:
//   - Get the OAuth authorization URL for Google services
//   - Save the OAuth authorization code to complete authentication
//
// These tools manage a unified OAuth token that provides access to all Google
// services used by inboxfewer: Gmail, Google Docs, and Google Drive.
//
// The OAuth flow:
//  1. Check if a token exists (automatic)
//  2. If not, call google_get_auth_url to get the authorization URL
//  3. User visits the URL and authorizes access
//  4. User provides the authorization code
//  5. Call google_save_auth_code with the code to save the token
//
// Once authenticated, all Gmail and Google Docs tools will work seamlessly
// with the saved token, which is automatically refreshed as needed.
package google_tools
