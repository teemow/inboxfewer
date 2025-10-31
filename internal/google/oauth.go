package google

import (
	"context"
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

// GetAuthURLForAccount returns the OAuth URL for user authorization for a specific account
func GetAuthURLForAccount(account string) string {
	conf := getOAuthConfig()
	// Include account name in state for better user experience
	return conf.AuthCodeURL(fmt.Sprintf("state-%s", account))
}

// GetAuthURL returns the OAuth URL for user authorization for the default account
func GetAuthURL() string {
	return GetAuthURLForAccount(defaultAccount)
}

// SaveTokenForAccount exchanges an authorization code for tokens and saves them for a specific account
func SaveTokenForAccount(ctx context.Context, account string, authCode string) error {
	if err := validateAccountName(account); err != nil {
		return fmt.Errorf("invalid account name: %w", err)
	}

	conf := getOAuthConfig()

	t, err := conf.Exchange(ctx, authCode)
	if err != nil {
		return fmt.Errorf("failed to exchange auth code for account %s: %w", account, err)
	}

	cacheDir := filepath.Join(userCacheDir(), "inboxfewer")
	tokenFile := getTokenFilePath(account)

	if err := os.MkdirAll(cacheDir, 0700); err != nil {
		return fmt.Errorf("failed to create cache directory: %w", err)
	}

	tokenData := t.AccessToken + " " + t.RefreshToken
	if err := os.WriteFile(tokenFile, []byte(tokenData), 0600); err != nil {
		return fmt.Errorf("failed to write token file for account %s: %w", account, err)
	}

	log.Printf("Saved OAuth token for account: %s", account)
	return nil
}

// SaveToken exchanges an authorization code for tokens and saves them for the default account
func SaveToken(ctx context.Context, authCode string) error {
	return SaveTokenForAccount(ctx, defaultAccount, authCode)
}

// getOAuthConfig returns the OAuth2 configuration for all Google services
func getOAuthConfig() *oauth2.Config {
	const OOB = "urn:ietf:wg:oauth:2.0:oob"
	return &oauth2.Config{
		ClientID:     "615260903473-ctldo9bte5phiu092s8ovfbe7c8aao1o.apps.googleusercontent.com",
		ClientSecret: "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",
		Endpoint:     google.Endpoint,
		RedirectURL:  OOB,
		Scopes: []string{
			gmail.MailGoogleComScope,                                  // Gmail access (includes send)
			"https://www.googleapis.com/auth/documents.readonly",      // Google Docs access
			"https://www.googleapis.com/auth/drive.readonly",          // Google Drive access
			"https://www.googleapis.com/auth/contacts.readonly",       // Google Contacts access
			"https://www.googleapis.com/auth/contacts.other.readonly", // Other contacts (interaction history)
			"https://www.googleapis.com/auth/directory.readonly",      // Directory contacts (Workspace)
			calendar.CalendarScope,                                    // Google Calendar access (read/write)
			meet.MeetingsSpaceReadonlyScope,                           // Google Meet access (read-only artifacts)
			"https://www.googleapis.com/auth/meetings.space.settings", // Google Meet settings (configure spaces)
			tasks.TasksScope, // Google Tasks access (read/write)
		},
	}
}

// GetTokenSourceForAccount returns an OAuth2 token source for the stored token for a specific account
// Returns nil if no valid token exists
func GetTokenSourceForAccount(ctx context.Context, account string) (oauth2.TokenSource, error) {
	if err := validateAccountName(account); err != nil {
		return nil, fmt.Errorf("invalid account name: %w", err)
	}

	conf := getOAuthConfig()
	tokenFile := getTokenFilePath(account)

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
		log.Printf("Cached token invalid for account %s: %v", account, err)
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
