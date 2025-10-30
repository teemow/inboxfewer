package google

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	gmail "google.golang.org/api/gmail/v1"
)

// HasToken checks if a valid OAuth token exists
func HasToken() bool {
	cacheDir := filepath.Join(userCacheDir(), "inboxfewer")
	tokenFile := filepath.Join(cacheDir, "google.token")
	_, err := ioutil.ReadFile(tokenFile)
	return err == nil
}

// GetAuthURL returns the OAuth URL for user authorization
func GetAuthURL() string {
	conf := getOAuthConfig()
	return conf.AuthCodeURL("state")
}

// SaveToken exchanges an authorization code for tokens and saves them
func SaveToken(ctx context.Context, authCode string) error {
	conf := getOAuthConfig()

	t, err := conf.Exchange(ctx, authCode)
	if err != nil {
		return fmt.Errorf("failed to exchange auth code: %w", err)
	}

	cacheDir := filepath.Join(userCacheDir(), "inboxfewer")
	tokenFile := filepath.Join(cacheDir, "google.token")

	if err := os.MkdirAll(cacheDir, 0700); err != nil {
		return fmt.Errorf("failed to create cache directory: %w", err)
	}

	tokenData := t.AccessToken + " " + t.RefreshToken
	if err := ioutil.WriteFile(tokenFile, []byte(tokenData), 0600); err != nil {
		return fmt.Errorf("failed to write token file: %w", err)
	}

	return nil
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
		},
	}
}

// GetTokenSource returns an OAuth2 token source for the stored token
// Returns nil if no valid token exists
func GetTokenSource(ctx context.Context) (oauth2.TokenSource, error) {
	conf := getOAuthConfig()

	cacheDir := filepath.Join(userCacheDir(), "inboxfewer")
	tokenFile := filepath.Join(cacheDir, "google.token")

	slurp, err := ioutil.ReadFile(tokenFile)
	if err != nil {
		return nil, fmt.Errorf("no valid Google OAuth token found")
	}

	f := strings.Fields(strings.TrimSpace(string(slurp)))
	if len(f) != 2 {
		return nil, fmt.Errorf("invalid token format")
	}

	ts := conf.TokenSource(ctx, &oauth2.Token{
		AccessToken:  f[0],
		TokenType:    "Bearer",
		RefreshToken: f[1],
		Expiry:       time.Unix(1, 0),
	})

	// Validate the token
	if _, err := ts.Token(); err != nil {
		log.Printf("Cached token invalid: %v", err)
		return nil, fmt.Errorf("cached token is invalid: %w", err)
	}

	return ts, nil
}

// GetHTTPClient returns an HTTP client configured with OAuth2 authentication
// The client is configured to use HTTP/1.1 to avoid HTTP/2 protocol errors
func GetHTTPClient(ctx context.Context) (*http.Client, error) {
	ts, err := GetTokenSource(ctx)
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
