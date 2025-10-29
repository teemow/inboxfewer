package docs

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
	docs "google.golang.org/api/docs/v1"
	drive "google.golang.org/api/drive/v3"
	"google.golang.org/api/option"
)

// Client wraps the Google Docs and Drive API services
type Client struct {
	docsService  *docs.Service
	driveService *drive.Service
}

// HasToken checks if a valid OAuth token exists
func HasToken() bool {
	cacheDir := filepath.Join(userCacheDir(), "inboxfewer")
	docsTokenFile := filepath.Join(cacheDir, "docs.token")
	_, err := ioutil.ReadFile(docsTokenFile)
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
	docsTokenFile := filepath.Join(cacheDir, "docs.token")

	if err := os.MkdirAll(cacheDir, 0700); err != nil {
		return fmt.Errorf("failed to create cache directory: %w", err)
	}

	tokenData := t.AccessToken + " " + t.RefreshToken
	if err := ioutil.WriteFile(docsTokenFile, []byte(tokenData), 0600); err != nil {
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
		Scopes: []string{
			"https://www.googleapis.com/auth/documents.readonly",
			"https://www.googleapis.com/auth/drive.readonly",
		},
	}
}

// NewClient creates a new Google Docs client with OAuth2 authentication
// Returns an error if no valid token exists - use HasToken() to check first
func NewClient(ctx context.Context) (*Client, error) {
	conf := getOAuthConfig()

	cacheDir := filepath.Join(userCacheDir(), "inboxfewer")
	docsTokenFile := filepath.Join(cacheDir, "docs.token")

	slurp, err := ioutil.ReadFile(docsTokenFile)
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
				log.Printf("Cached Docs token invalid: %v", err)
				ts = nil
			}
		}
	}

	if ts == nil {
		return nil, fmt.Errorf("no valid Google Docs OAuth token found. Please authorize access first")
	}

	// Create client with HTTP/1.1 to avoid HTTP/2 protocol errors
	client := oauth2.NewClient(ctx, ts)

	// Force HTTP/1.1 by disabling HTTP/2
	transport := client.Transport.(*oauth2.Transport)
	baseTransport := &http.Transport{
		ForceAttemptHTTP2: false,
	}
	transport.Base = baseTransport

	// Create Docs service
	docsService, err := docs.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return nil, fmt.Errorf("failed to create Docs service: %w", err)
	}

	// Create Drive service
	driveService, err := drive.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return nil, fmt.Errorf("failed to create Drive service: %w", err)
	}

	return &Client{
		docsService:  docsService,
		driveService: driveService,
	}, nil
}

// GetDocument retrieves a Google Doc's content by document ID
func (c *Client) GetDocument(documentID string) (*docs.Document, error) {
	if documentID == "" {
		return nil, fmt.Errorf("documentID is required")
	}

	doc, err := c.docsService.Documents.Get(documentID).Do()
	if err != nil {
		return nil, fmt.Errorf("failed to get document %s: %w", documentID, err)
	}

	return doc, nil
}

// GetDocumentAsMarkdown converts a Google Doc to Markdown format
func (c *Client) GetDocumentAsMarkdown(documentID string) (string, error) {
	doc, err := c.GetDocument(documentID)
	if err != nil {
		return "", err
	}

	return DocumentToMarkdown(doc)
}

// GetDocumentAsPlainText extracts plain text from a Google Doc
func (c *Client) GetDocumentAsPlainText(documentID string) (string, error) {
	doc, err := c.GetDocument(documentID)
	if err != nil {
		return "", err
	}

	return DocumentToPlainText(doc)
}

// GetFileMetadata retrieves metadata for any Google Drive file
func (c *Client) GetFileMetadata(fileID string) (*DocumentMetadata, error) {
	if fileID == "" {
		return nil, fmt.Errorf("fileID is required")
	}

	file, err := c.driveService.Files.Get(fileID).
		Fields("id, name, mimeType, createdTime, modifiedTime, size, owners").
		Do()
	if err != nil {
		return nil, fmt.Errorf("failed to get file metadata %s: %w", fileID, err)
	}

	metadata := &DocumentMetadata{
		ID:           file.Id,
		Name:         file.Name,
		MimeType:     file.MimeType,
		CreatedTime:  file.CreatedTime,
		ModifiedTime: file.ModifiedTime,
		Size:         file.Size,
	}

	// Convert owners
	for _, owner := range file.Owners {
		metadata.Owners = append(metadata.Owners, User{
			DisplayName:  owner.DisplayName,
			EmailAddress: owner.EmailAddress,
		})
	}

	return metadata, nil
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
