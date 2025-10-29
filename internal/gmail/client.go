package gmail

import (
	"bufio"
	"context"
	"fmt"
	"io"
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

// Client wraps the Gmail Users service
type Client struct {
	svc *gmail.UsersService
}

// HasToken checks if a valid OAuth token exists
func HasToken() bool {
	cacheDir := filepath.Join(userCacheDir(), "inboxfewer")
	gmailTokenFile := filepath.Join(cacheDir, "gmail.token")
	_, err := ioutil.ReadFile(gmailTokenFile)
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
	gmailTokenFile := filepath.Join(cacheDir, "gmail.token")

	if err := os.MkdirAll(cacheDir, 0700); err != nil {
		return fmt.Errorf("failed to create cache directory: %w", err)
	}

	tokenData := t.AccessToken + " " + t.RefreshToken
	if err := ioutil.WriteFile(gmailTokenFile, []byte(tokenData), 0600); err != nil {
		return fmt.Errorf("failed to write token file: %w", err)
	}

	return nil
}

func getOAuthConfig() *oauth2.Config {
	const OOB = "urn:ietf:wg:oauth:2.0:oob"
	return &oauth2.Config{
		ClientID:     "881077086782-039l7vctubc7vrvjmubv6a7v0eg96sqg.apps.googleusercontent.com",
		ClientSecret: "y9Rj5-KheyZSFyjCH1dCBXWs",
		Endpoint:     google.Endpoint,
		RedirectURL:  OOB,
		Scopes:       []string{gmail.MailGoogleComScope},
	}
}

// NewClient creates a new Gmail client with OAuth2 authentication
// For CLI usage, it will prompt for auth code via stdin if no token exists
// For MCP usage, it will return an error if no token exists
func NewClient(ctx context.Context) (*Client, error) {
	conf := getOAuthConfig()

	cacheDir := filepath.Join(userCacheDir(), "inboxfewer")
	gmailTokenFile := filepath.Join(cacheDir, "gmail.token")

	slurp, err := ioutil.ReadFile(gmailTokenFile)
	var ts oauth2.TokenSource
	if err == nil {
		f := strings.Fields(strings.TrimSpace(string(slurp)))
		if len(f) == 2 {
			ts = conf.TokenSource(ctx, &oauth2.Token{
				AccessToken:  f[0],
				TokenType:    "Bearer",
				RefreshToken: f[1],
				Expiry:       time.Unix(1, 0),
			})
			if _, err := ts.Token(); err != nil {
				log.Printf("Cached token invalid: %v", err)
				ts = nil
			}
		}
	}

	if ts == nil {
		// Check if we're in a terminal (CLI mode)
		if isTerminal() {
			authCode := conf.AuthCodeURL("state")
			log.Printf("Go to %v", authCode)
			io.WriteString(os.Stdout, "Enter code> ")

			bs := bufio.NewScanner(os.Stdin)
			if !bs.Scan() {
				return nil, io.EOF
			}
			code := strings.TrimSpace(bs.Text())
			if err := SaveToken(ctx, code); err != nil {
				return nil, err
			}
			// Re-read the token we just saved
			slurp, _ = ioutil.ReadFile(gmailTokenFile)
			f := strings.Fields(strings.TrimSpace(string(slurp)))
			ts = conf.TokenSource(ctx, &oauth2.Token{
				AccessToken:  f[0],
				TokenType:    "Bearer",
				RefreshToken: f[1],
				Expiry:       time.Unix(1, 0),
			})
		} else {
			// MCP mode - return error with instructions
			return nil, fmt.Errorf("no valid Gmail OAuth token found. Use gmail_get_auth_url and gmail_save_auth_code tools to authenticate")
		}
	}

	// Create client with HTTP/1.1 to avoid HTTP/2 protocol errors
	client := oauth2.NewClient(ctx, ts)

	// Force HTTP/1.1 by disabling HTTP/2
	transport := client.Transport.(*oauth2.Transport)
	baseTransport := &http.Transport{
		ForceAttemptHTTP2: false,
	}
	transport.Base = baseTransport

	svc, err := gmail.New(client)
	if err != nil {
		return nil, err
	}

	return &Client{
		svc: svc.Users,
	}, nil
}

// ArchiveThread archives a thread by removing the INBOX label
func (c *Client) ArchiveThread(tid string) error {
	_, err := c.svc.Threads.Modify("me", tid, &gmail.ModifyThreadRequest{
		RemoveLabelIds: []string{"INBOX"},
	}).Do()
	return err
}

// ForeachThread iterates over all threads matching the query
func (c *Client) ForeachThread(q string, fn func(*gmail.Thread) error) error {
	pageToken := ""
	for {
		req := c.svc.Threads.List("me").Q(q)
		if pageToken != "" {
			req.PageToken(pageToken)
		}
		res, err := req.Do()
		if err != nil {
			return err
		}
		for _, t := range res.Threads {
			if err := fn(t); err != nil {
				return err
			}
		}
		if res.NextPageToken == "" {
			return nil
		}
		pageToken = res.NextPageToken
	}
}

// PopulateThread populates t with its full data. t.Id must be set initially.
func (c *Client) PopulateThread(t *gmail.Thread) error {
	req := c.svc.Threads.Get("me", t.Id).Format("full")
	tfull, err := req.Do()
	if err != nil {
		return err
	}
	*t = *tfull
	return nil
}

// ListThreads lists threads matching the query with pagination
func (c *Client) ListThreads(q string, maxResults int64) ([]*gmail.Thread, error) {
	req := c.svc.Threads.List("me").Q(q).MaxResults(maxResults)
	res, err := req.Do()
	if err != nil {
		return nil, err
	}
	return res.Threads, nil
}

func userCacheDir() string {
	switch runtime.GOOS {
	case "darwin":
		return filepath.Join(HomeDir(), "Library", "Caches")
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
	return filepath.Join(HomeDir(), ".cache")
}

func HomeDir() string {
	if runtime.GOOS == "windows" {
		return os.Getenv("HOMEDRIVE") + os.Getenv("HOMEPATH")
	}
	return os.Getenv("HOME")
}

// isTerminal checks if stdin is connected to a terminal (CLI mode)
func isTerminal() bool {
	fileInfo, _ := os.Stdin.Stat()
	return (fileInfo.Mode() & os.ModeCharDevice) != 0
}
