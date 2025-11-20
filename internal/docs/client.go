package docs

import (
	"context"
	"fmt"
	"net/http"

	"golang.org/x/oauth2"
	docs "google.golang.org/api/docs/v1"
	drive "google.golang.org/api/drive/v3"
	"google.golang.org/api/option"

	"github.com/teemow/inboxfewer/internal/google"
)

// Client wraps the Google Docs and Drive API services
type Client struct {
	docsService   *docs.Service
	driveService  *drive.Service
	account       string // The account this client is associated with
	tokenProvider google.TokenProvider
}

// Account returns the account name this client is associated with
func (c *Client) Account() string {
	return c.account
}

// HasTokenForAccountWithProvider checks if a valid OAuth token exists for the specified account
func HasTokenForAccountWithProvider(account string, provider google.TokenProvider) bool {
	if provider == nil {
		return false
	}
	return provider.HasTokenForAccount(account)
}

// HasTokenForAccount checks if a valid OAuth token exists for the specified account
func HasTokenForAccount(account string) bool {
	provider := google.NewFileTokenProvider()
	return HasTokenForAccountWithProvider(account, provider)
}

// HasToken checks if a valid OAuth token exists for the default account
func HasToken() bool {
	return HasTokenForAccount("default")
}

// NewClientForAccountWithProvider creates a new Google Docs client with OAuth2 authentication for a specific account
// The OAuth token is retrieved from the provided token provider
func NewClientForAccountWithProvider(ctx context.Context, account string, tokenProvider google.TokenProvider) (*Client, error) {
	if tokenProvider == nil {
		return nil, fmt.Errorf("token provider cannot be nil")
	}

	// Get token from the provided provider
	token, err := tokenProvider.GetTokenForAccount(ctx, account)
	if err != nil {
		return nil, fmt.Errorf("failed to get Google OAuth token for account %s: %w", account, err)
	}

	// Create OAuth2 config and token source
	conf := google.GetOAuthConfig()
	tokenSource := conf.TokenSource(ctx, token)

	// Create HTTP client with the token
	client := oauth2.NewClient(ctx, tokenSource)

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
		docsService:   docsService,
		driveService:  driveService,
		account:       account,
		tokenProvider: tokenProvider,
	}, nil
}

// NewClientForAccount creates a new Google Docs client with OAuth2 authentication for a specific account
// Uses the default file-based token provider for backward compatibility
func NewClientForAccount(ctx context.Context, account string) (*Client, error) {
	provider := google.NewFileTokenProvider()
	return NewClientForAccountWithProvider(ctx, account, provider)
}

// NewClientWithProvider creates a new Google Docs client with OAuth2 authentication for the default account
// using the provided token provider
func NewClientWithProvider(ctx context.Context, provider google.TokenProvider) (*Client, error) {
	return NewClientForAccountWithProvider(ctx, "default", provider)
}

// NewClient creates a new Google Docs client with OAuth2 authentication for the default account
// Returns an error if no valid token exists - use HasToken() to check first
func NewClient(ctx context.Context) (*Client, error) {
	return NewClientForAccount(ctx, "default")
}

// GetDocument retrieves a Google Doc's content by document ID
// This method automatically fetches all tabs to support documents with multiple tabs (introduced Oct 2024)
func (c *Client) GetDocument(documentID string) (*docs.Document, error) {
	if documentID == "" {
		return nil, fmt.Errorf("documentID is required")
	}

	// Use includeTabsContent=true to fetch all tabs in documents that have them
	// This returns document.tabs populated for multi-tab docs, or document.body for legacy docs
	doc, err := c.docsService.Documents.Get(documentID).IncludeTabsContent(true).Do()
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
