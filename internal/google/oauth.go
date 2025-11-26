package google

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	calendar "google.golang.org/api/calendar/v3"
	gmail "google.golang.org/api/gmail/v1"
	meet "google.golang.org/api/meet/v2"
	tasks "google.golang.org/api/tasks/v1"
)

// hashEmail creates a hash of an email for logging purposes
// This prevents PII leakage in logs while maintaining traceability
func hashEmail(email string) string {
	hash := sha256.Sum256([]byte(email))
	return hex.EncodeToString(hash[:8]) // First 8 bytes (16 hex chars) for brevity
}

const defaultAccount = "default"

// Account name validation regex: alphanumeric, hyphens, and underscores only
var accountNameRegex = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)

// MigrateDefaultToken migrates the old google.token to google-default.token if needed
// This ensures backward compatibility for existing users
func MigrateDefaultToken() error {
	cacheDir := filepath.Join(userCacheDir(), "inboxfewer")
	oldTokenFile := filepath.Join(cacheDir, "google.token")
	newTokenFile := getTokenFilePath(defaultAccount)

	// Check if old token exists and new one doesn't
	if _, err := os.Stat(oldTokenFile); err == nil {
		if _, err := os.Stat(newTokenFile); os.IsNotExist(err) {
			// Rename old token to new format
			if err := os.Rename(oldTokenFile, newTokenFile); err != nil {
				return fmt.Errorf("failed to migrate token file: %w", err)
			}
			// Fix permissions on migrated file
			if err := os.Chmod(newTokenFile, 0600); err != nil {
				log.Printf("Warning: Failed to set permissions on migrated token file: %v", err)
			}
			log.Printf("Migrated google.token to google-default.token")
		}
	}

	return nil
}

// getTokenFilePath returns the token file path for the given account
func getTokenFilePath(account string) string {
	cacheDir := filepath.Join(userCacheDir(), "inboxfewer")
	return filepath.Join(cacheDir, fmt.Sprintf("google-%s.token", account))
}

// validateAccountName validates that the account name is acceptable
func validateAccountName(account string) error {
	if account == "" {
		return fmt.Errorf("account name cannot be empty")
	}
	if len(account) > 255 {
		return fmt.Errorf("account name too long (max 255 characters)")
	}
	// Additional path traversal protection
	if strings.Contains(account, "..") || strings.Contains(account, "/") || strings.Contains(account, "\\") {
		return fmt.Errorf("account name contains invalid path characters")
	}
	if !accountNameRegex.MatchString(account) {
		return fmt.Errorf("account name must contain only alphanumeric characters, hyphens, and underscores")
	}
	return nil
}

// HasTokenForAccount checks if a valid OAuth token exists for the specified account
func HasTokenForAccount(account string) bool {
	if err := validateAccountName(account); err != nil {
		return false
	}
	tokenFile := getTokenFilePath(account)
	_, err := os.ReadFile(tokenFile)
	return err == nil
}

// HasToken checks if a valid OAuth token exists for the default account
func HasToken() bool {
	return HasTokenForAccount(defaultAccount)
}

// GetAuthenticationErrorMessage returns a user-friendly error message when authentication is required
func GetAuthenticationErrorMessage(account string) string {
	return fmt.Sprintf(`Google OAuth authentication required for account "%s".

For HTTP/SSE transports:
  Your MCP client (e.g., Cursor, Claude Desktop) will automatically handle
  the OAuth flow with Google. Make sure you're connected to an MCP server
  that supports OAuth authentication.

For STDIO transport:
  Authentication tokens should be managed through environment variables or
  the Google Cloud SDK.

Account: %s`, account, account)
}

// SaveTokenForAccount saves a Google OAuth token for a specific account
// This is called by the OAuth middleware after validating the user's token
func SaveTokenForAccount(ctx context.Context, account string, token *oauth2.Token) error {
	if err := validateAccountName(account); err != nil {
		return fmt.Errorf("invalid account name: %w", err)
	}

	cacheDir := filepath.Join(userCacheDir(), "inboxfewer")
	tokenFile := getTokenFilePath(account)

	if err := os.MkdirAll(cacheDir, 0700); err != nil {
		return fmt.Errorf("failed to create cache directory: %w", err)
	}

	// Verify directory permissions for security
	dirInfo, err := os.Stat(cacheDir)
	if err != nil {
		return fmt.Errorf("failed to stat cache directory: %w", err)
	}
	if dirInfo.Mode().Perm() != 0700 {
		// Attempt to fix permissions
		if err := os.Chmod(cacheDir, 0700); err != nil {
			return fmt.Errorf("insecure cache directory permissions %o and failed to fix: %w", dirInfo.Mode().Perm(), err)
		}
		log.Printf("Fixed insecure cache directory permissions from %o to 0700", dirInfo.Mode().Perm())
	}

	tokenData := token.AccessToken + " " + token.RefreshToken
	if err := os.WriteFile(tokenFile, []byte(tokenData), 0600); err != nil {
		return fmt.Errorf("failed to write token file for account %s: %w", account, err)
	}

	// Verify file permissions after write
	fileInfo, err := os.Stat(tokenFile)
	if err != nil {
		return fmt.Errorf("failed to stat token file: %w", err)
	}
	if fileInfo.Mode().Perm() != 0600 {
		// Attempt to fix permissions
		if err := os.Chmod(tokenFile, 0600); err != nil {
			return fmt.Errorf("insecure token file permissions %o and failed to fix: %w", fileInfo.Mode().Perm(), err)
		}
		log.Printf("Fixed insecure token file permissions from %o to 0600", fileInfo.Mode().Perm())
	}

	// Log with hashed email to prevent PII leakage
	log.Printf("Saved OAuth token for account hash: %s", hashEmail(account))
	return nil
}

// getOAuthConfig returns the OAuth2 configuration for all Google services (internal use)
// Credentials are read from environment variables for security
func getOAuthConfig() *oauth2.Config {
	const OOB = "urn:ietf:wg:oauth:2.0:oob"

	// Get credentials from environment variables
	// These are required for STDIO transport token refresh
	clientID := os.Getenv("GOOGLE_STDIO_CLIENT_ID")
	clientSecret := os.Getenv("GOOGLE_STDIO_CLIENT_SECRET")

	// Fall back to GOOGLE_CLIENT_ID/SECRET if STDIO-specific vars not set
	if clientID == "" {
		clientID = os.Getenv("GOOGLE_CLIENT_ID")
	}
	if clientSecret == "" {
		clientSecret = os.Getenv("GOOGLE_CLIENT_SECRET")
	}

	return &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		Endpoint:     google.Endpoint,
		RedirectURL:  OOB,
		Scopes: []string{
			gmail.MailGoogleComScope,                                  // Gmail access (includes send)
			gmail.GmailSettingsBasicScope,                             // Gmail settings (filters, labels, etc.)
			"https://www.googleapis.com/auth/documents.readonly",      // Google Docs access
			"https://www.googleapis.com/auth/drive",                   // Google Drive access (read/write)
			"https://www.googleapis.com/auth/contacts.readonly",       // Google Contacts access
			"https://www.googleapis.com/auth/contacts.other.readonly", // Other contacts (interaction history)
			"https://www.googleapis.com/auth/directory.readonly",      // Directory contacts (Workspace)
			calendar.CalendarScope,                                    // Google Calendar access (read/write)
			meet.MeetingsSpaceReadonlyScope,                           // Google Meet access (read-only artifacts)
			"https://www.googleapis.com/auth/meetings.space.settings", // Google Meet settings (configure spaces)
			tasks.TasksScope,                                          // Google Tasks access (read/write)
		},
	}
}

// GetOAuthConfig returns the OAuth2 configuration for all Google services (exported for use by clients)
func GetOAuthConfig() *oauth2.Config {
	return getOAuthConfig()
}

// GetTokenSourceForAccount returns an OAuth2 token source for the stored token for a specific account
// Returns nil if no valid token exists
func GetTokenSourceForAccount(ctx context.Context, account string) (oauth2.TokenSource, error) {
	if err := validateAccountName(account); err != nil {
		return nil, fmt.Errorf("invalid account name: %w", err)
	}

	conf := getOAuthConfig()
	if conf.ClientID == "" || conf.ClientSecret == "" {
		return nil, fmt.Errorf("Google OAuth credentials not configured. Set GOOGLE_STDIO_CLIENT_ID and GOOGLE_STDIO_CLIENT_SECRET (or GOOGLE_CLIENT_ID and GOOGLE_CLIENT_SECRET) environment variables")
	}

	tokenFile := getTokenFilePath(account)

	// Verify file permissions before reading
	fileInfo, err := os.Stat(tokenFile)
	if err != nil {
		return nil, fmt.Errorf("no valid Google OAuth token found for account %s", account)
	}
	if fileInfo.Mode().Perm() != 0600 {
		log.Printf("Warning: Token file has insecure permissions %o (expected 0600) for account hash %s", fileInfo.Mode().Perm(), hashEmail(account))
		// Attempt to fix permissions
		if err := os.Chmod(tokenFile, 0600); err != nil {
			return nil, fmt.Errorf("token file has insecure permissions %o and cannot be fixed: %w", fileInfo.Mode().Perm(), err)
		}
		log.Printf("Fixed token file permissions to 0600 for account hash %s", hashEmail(account))
	}

	slurp, err := os.ReadFile(tokenFile)
	if err != nil {
		return nil, fmt.Errorf("no valid Google OAuth token found for account %s", account)
	}

	f := strings.Fields(strings.TrimSpace(string(slurp)))
	if len(f) != 2 {
		return nil, fmt.Errorf("invalid token format for account %s", account)
	}

	ts := conf.TokenSource(ctx, &oauth2.Token{
		AccessToken:  f[0],
		TokenType:    "Bearer",
		RefreshToken: f[1],
		Expiry:       time.Unix(1, 0),
	})

	// Validate the token
	if _, err := ts.Token(); err != nil {
		log.Printf("Cached token invalid for account hash %s: %v", hashEmail(account), err)
		return nil, fmt.Errorf("cached token is invalid for account %s: %w", account, err)
	}

	return ts, nil
}

// GetTokenSource returns an OAuth2 token source for the stored token for the default account
// Returns nil if no valid token exists
func GetTokenSource(ctx context.Context) (oauth2.TokenSource, error) {
	return GetTokenSourceForAccount(ctx, defaultAccount)
}

// GetHTTPClientForAccount returns an HTTP client configured with OAuth2 authentication for a specific account
// The client is configured to use HTTP/1.1 to avoid HTTP/2 protocol errors
func GetHTTPClientForAccount(ctx context.Context, account string) (*http.Client, error) {
	ts, err := GetTokenSourceForAccount(ctx, account)
	if err != nil {
		return nil, err
	}

	client := oauth2.NewClient(ctx, ts)

	// Force HTTP/1.1 by disabling HTTP/2
	transport := client.Transport.(*oauth2.Transport)
	baseTransport := &http.Transport{
		ForceAttemptHTTP2: false,
	}
	transport.Base = baseTransport

	return client, nil
}

// GetHTTPClient returns an HTTP client configured with OAuth2 authentication for the default account
// The client is configured to use HTTP/1.1 to avoid HTTP/2 protocol errors
func GetHTTPClient(ctx context.Context) (*http.Client, error) {
	return GetHTTPClientForAccount(ctx, defaultAccount)
}

func userCacheDir() string {
	switch runtime.GOOS {
	case "darwin":
		return filepath.Join(homeDir(), "Library", "Caches")
	case "windows":
		for _, ev := range []string{"TEMP", "TMP"} {
			if v := os.Getenv(ev); v != "" {
				return v
			}
		}
		panic("No Windows TEMP or TMP environment variables found")
	}
	if xdg := os.Getenv("XDG_CACHE_HOME"); xdg != "" {
		return xdg
	}
	return filepath.Join(homeDir(), ".cache")
}

func homeDir() string {
	if runtime.GOOS == "windows" {
		return os.Getenv("HOMEDRIVE") + os.Getenv("HOMEPATH")
	}
	return os.Getenv("HOME")
}
