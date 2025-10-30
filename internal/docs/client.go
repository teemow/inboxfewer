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

// NewClient creates a new Google Docs client with OAuth2 authentication
// Returns an error if no valid token exists - use HasToken() to check first
func NewClient(ctx context.Context) (*Client, error) {
	client, err := google.GetHTTPClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("no valid Google OAuth token found. Please authorize access first: %w", err)
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
	}, nil
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
