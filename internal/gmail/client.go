package gmail

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log"
	"os"

	gmail "google.golang.org/api/gmail/v1"

	"github.com/teemow/inboxfewer/internal/google"
)

// Client wraps the Gmail Users service
type Client struct {
	svc *gmail.UsersService
}

// HasToken checks if a valid OAuth token exists
func HasToken() bool {
	return google.HasToken()
}

// GetAuthURL returns the OAuth URL for user authorization
func GetAuthURL() string {
	return google.GetAuthURL()
}

// SaveToken exchanges an authorization code for tokens and saves them
func SaveToken(ctx context.Context, authCode string) error {
	return google.SaveToken(ctx, authCode)
}

// NewClient creates a new Gmail client with OAuth2 authentication
// For CLI usage, it will prompt for auth code via stdin if no token exists
// For MCP usage, it will return an error if no token exists
func NewClient(ctx context.Context) (*Client, error) {
	// Try to get existing token
	client, err := google.GetHTTPClient(ctx)
	if err != nil {
		// Check if we're in a terminal (CLI mode)
		if isTerminal() {
			authURL := google.GetAuthURL()
			log.Printf("Go to %v", authURL)
			io.WriteString(os.Stdout, "Enter code> ")

			bs := bufio.NewScanner(os.Stdin)
			if !bs.Scan() {
				return nil, io.EOF
			}
			code := bs.Text()
			if err := google.SaveToken(ctx, code); err != nil {
				return nil, err
			}
			// Try again with the new token
			client, err = google.GetHTTPClient(ctx)
			if err != nil {
				return nil, err
			}
		} else {
			// MCP mode - return error with instructions
			return nil, fmt.Errorf("no valid Google OAuth token found. Use gmail_get_auth_url and gmail_save_auth_code tools to authenticate")
		}
	}

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

// isTerminal checks if stdin is connected to a terminal (CLI mode)
func isTerminal() bool {
	fileInfo, _ := os.Stdin.Stat()
	return (fileInfo.Mode() & os.ModeCharDevice) != 0
}
