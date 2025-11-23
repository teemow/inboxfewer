package google

// DefaultOAuthScopes are the Google OAuth scopes required for full MCP functionality
// These scopes are used consistently across the application for OAuth configurations
var DefaultOAuthScopes = []string{
	"openid", // Required for user info endpoint
	"https://www.googleapis.com/auth/userinfo.email",
	"https://www.googleapis.com/auth/gmail.readonly",
	"https://www.googleapis.com/auth/gmail.modify",
	"https://www.googleapis.com/auth/gmail.send",
	"https://www.googleapis.com/auth/gmail.settings.basic",
	"https://www.googleapis.com/auth/documents.readonly",
	"https://www.googleapis.com/auth/drive",
	"https://www.googleapis.com/auth/calendar",
	"https://www.googleapis.com/auth/meetings.space.readonly",
	"https://www.googleapis.com/auth/tasks",
}
