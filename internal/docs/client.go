package docs

import (
	"context"
	"fmt"

	docs "google.golang.org/api/docs/v1"
	drive "google.golang.org/api/drive/v3"
	"google.golang.org/api/option"

	"github.com/teemow/inboxfewer/internal/google"
)

// Client wraps the Google Docs and Drive API services
type Client struct {
	docsService  *docs.Service
	driveService *drive.Service
	account      string // The account this client is associated with
}

// Account returns the account name this client is associated with
func (c *Client) Account() string {
	return c.account
}

// HasTokenForAccount checks if a valid OAuth token exists for the specified account
func HasTokenForAccount(account string) bool {
	return google.HasTokenForAccount(account)
}

// HasToken checks if a valid OAuth token exists for the default account
func HasToken() bool {
	return google.HasToken()
}

// GetAuthURLForAccount returns the OAuth URL for user authorization for a specific account
func GetAuthURLForAccount(account string) string {
	return google.GetAuthURLForAccount(account)
}

// GetAuthURL returns the OAuth URL for user authorization for the default account
func GetAuthURL() string {
	return google.GetAuthURL()
}

// SaveTokenForAccount exchanges an authorization code for tokens and saves them for a specific account
func SaveTokenForAccount(ctx context.Context, account string, authCode string) error {
	return google.SaveTokenForAccount(ctx, account, authCode)
}

// SaveToken exchanges an authorization code for tokens and saves them for the default account
func SaveToken(ctx context.Context, authCode string) error {
	return google.SaveToken(ctx, authCode)
}

// NewClientForAccount creates a new Google Docs client with OAuth2 authentication for a specific account
// Returns an error if no valid token exists - use HasTokenForAccount() to check first
func NewClientForAccount(ctx context.Context, account string) (*Client, error) {
	client, err := google.GetHTTPClientForAccount(ctx, account)
	if err != nil {
		return nil, fmt.Errorf("no valid Google OAuth token found for account %s. Please authorize access first: %w", account, err)
	}

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
		account:      account,
	}, nil
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
