package google

// DefaultOAuthScopes are the Google OAuth scopes required for full MCP functionality.
// These scopes are used consistently across the application for OAuth configurations.
//
// The scopes provide access to:
//   - Gmail: read, modify, send, settings
//   - Google Docs: read-only
//   - Google Drive: full access
//   - Google Calendar: full access
//   - Google Meet: read spaces and settings
//   - Google Tasks: full access
//   - Contacts: read-only (including other contacts and directory)
var DefaultOAuthScopes = []string{
	// OpenID Connect scopes (required for user info)
	"openid",
	"https://www.googleapis.com/auth/userinfo.email",

	// Gmail scopes
	"https://mail.google.com/", // Full Gmail access (includes send)
	"https://www.googleapis.com/auth/gmail.readonly",
	"https://www.googleapis.com/auth/gmail.modify",
	"https://www.googleapis.com/auth/gmail.send",
	"https://www.googleapis.com/auth/gmail.settings.basic",

	// Google Docs scope
	"https://www.googleapis.com/auth/documents.readonly",

	// Google Drive scope
	"https://www.googleapis.com/auth/drive",

	// Google Calendar scope
	"https://www.googleapis.com/auth/calendar",

	// Google Meet scopes
	"https://www.googleapis.com/auth/meetings.space.readonly",
	"https://www.googleapis.com/auth/meetings.space.settings",

	// Google Tasks scope
	"https://www.googleapis.com/auth/tasks",

	// Contacts scopes
	"https://www.googleapis.com/auth/contacts.readonly",
	"https://www.googleapis.com/auth/contacts.other.readonly",
	"https://www.googleapis.com/auth/directory.readonly",
}
