package gmail

import (
	"bufio"
	"context"
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

// NewClient creates a new Gmail client with OAuth2 authentication
func NewClient(ctx context.Context) (*Client, error) {
	const OOB = "urn:ietf:wg:oauth:2.0:oob"
	conf := &oauth2.Config{
		ClientID:     "881077086782-039l7vctubc7vrvjmubv6a7v0eg96sqg.apps.googleusercontent.com",
		ClientSecret: "y9Rj5-KheyZSFyjCH1dCBXWs",
		Endpoint:     google.Endpoint,
		RedirectURL:  OOB,
		Scopes:       []string{gmail.MailGoogleComScope},
	}

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
		authCode := conf.AuthCodeURL("state")
		log.Printf("Go to %v", authCode)
		io.WriteString(os.Stdout, "Enter code> ")

		bs := bufio.NewScanner(os.Stdin)
		if !bs.Scan() {
			return nil, io.EOF
		}
		code := strings.TrimSpace(bs.Text())
		t, err := conf.Exchange(ctx, code)
		if err != nil {
			return nil, err
		}
		os.MkdirAll(cacheDir, 0700)
		ioutil.WriteFile(gmailTokenFile, []byte(t.AccessToken+" "+t.RefreshToken), 0600)
		ts = conf.TokenSource(ctx, t)
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
