package drive

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"golang.org/x/oauth2"
	drive "google.golang.org/api/drive/v3"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/option"

	"github.com/teemow/inboxfewer/internal/google"
)

const (
	// FolderMimeType is the MIME type for Google Drive folders
	FolderMimeType = "application/vnd.google-apps.folder"
)

// Client wraps the Google Drive API service
type Client struct {
	service       *drive.Service
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

// NewClientForAccountWithProvider creates a new Google Drive client with OAuth2 authentication for a specific account
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

	// Create Drive service
	driveService, err := drive.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return nil, fmt.Errorf("failed to create Drive service: %w", err)
	}

	return &Client{
		service:       driveService,
		account:       account,
		tokenProvider: tokenProvider,
	}, nil
}

// NewClientForAccount creates a new Google Drive client with OAuth2 authentication for a specific account
// Uses the default file-based token provider for backward compatibility
func NewClientForAccount(ctx context.Context, account string) (*Client, error) {
	provider := google.NewFileTokenProvider()
	return NewClientForAccountWithProvider(ctx, account, provider)
}

// NewClientWithProvider creates a new Google Drive client with OAuth2 authentication for the default account
// using the provided token provider
func NewClientWithProvider(ctx context.Context, provider google.TokenProvider) (*Client, error) {
	return NewClientForAccountWithProvider(ctx, "default", provider)
}

// NewClient creates a new Google Drive client with OAuth2 authentication for the default account
// Returns an error if no valid token exists - use HasToken() to check first
func NewClient(ctx context.Context) (*Client, error) {
	return NewClientForAccount(ctx, "default")
}

// UploadFile uploads a file to Google Drive
func (c *Client) UploadFile(ctx context.Context, name string, content io.Reader, options *UploadOptions) (*FileInfo, error) {
	if name == "" {
		return nil, fmt.Errorf("file name is required")
	}
	if content == nil {
		return nil, fmt.Errorf("file content is required")
	}

	file := &drive.File{
		Name: name,
	}

	if options != nil {
		if len(options.ParentFolders) > 0 {
			file.Parents = options.ParentFolders
		}
		if options.Description != "" {
			file.Description = options.Description
		}
		if options.MimeType != "" {
			file.MimeType = options.MimeType
		}
		if options.ModifiedTime != nil {
			file.ModifiedTime = options.ModifiedTime.Format(time.RFC3339)
		}
	}

	driveFile, err := c.service.Files.Create(file).
		Context(ctx).
		Media(content, googleapi.ContentType(file.MimeType)).
		Fields("id, name, mimeType, size, createdTime, modifiedTime, webViewLink, webContentLink, parents, owners, shared, trashed").
		Do()
	if err != nil {
		return nil, fmt.Errorf("failed to upload file: %w", err)
	}

	return convertToFileInfo(driveFile), nil
}

// buildListFilesQuery constructs the query string for listing files,
// combining user-provided queries with the trashed filter as needed.
// This function is extracted to make the query building logic testable.
func buildListFilesQuery(userQuery string, includeTrashed bool) string {
	if userQuery != "" {
		if includeTrashed {
			// User wants trashed files included, use their query as-is
			return userQuery
		}
		// Combine user query with trashed=false filter
		return fmt.Sprintf("(%s) and trashed=false", userQuery)
	}
	// No user query, just add trashed filter if needed
	if !includeTrashed {
		return "trashed=false"
	}
	return ""
}

// ListFiles lists files in Google Drive with optional filtering
func (c *Client) ListFiles(ctx context.Context, options *ListOptions) ([]*FileInfo, string, error) {
	call := c.service.Files.List().
		Context(ctx).
		Fields("nextPageToken, files(id, name, mimeType, size, createdTime, modifiedTime, webViewLink, webContentLink, parents, owners, shared, trashed, trashedTime)")

	if options != nil {
		// Build the query string using the helper function
		// Note: calling .Q() multiple times overwrites previous queries, so we must combine them
		finalQuery := buildListFilesQuery(options.Query, options.IncludeTrashed)
		if finalQuery != "" {
			call = call.Q(finalQuery)
		}

		if options.MaxResults > 0 {
			call = call.PageSize(int64(options.MaxResults))
		}
		if options.OrderBy != "" {
			call = call.OrderBy(options.OrderBy)
		}
		if options.PageToken != "" {
			call = call.PageToken(options.PageToken)
		}
		if options.Spaces != "" {
			call = call.Spaces(options.Spaces)
		}
	} else {
		// Default: exclude trashed files
		call = call.Q("trashed=false")
	}

	fileList, err := call.Do()
	if err != nil {
		return nil, "", fmt.Errorf("failed to list files: %w", err)
	}

	files := make([]*FileInfo, len(fileList.Files))
	for i, f := range fileList.Files {
		files[i] = convertToFileInfo(f)
	}

	return files, fileList.NextPageToken, nil
}

// GetFile retrieves metadata for a specific file
func (c *Client) GetFile(ctx context.Context, fileID string) (*FileInfo, error) {
	if fileID == "" {
		return nil, fmt.Errorf("fileID is required")
	}

	file, err := c.service.Files.Get(fileID).
		Context(ctx).
		Fields("id, name, mimeType, size, createdTime, modifiedTime, webViewLink, webContentLink, parents, owners, shared, trashed, trashedTime, permissions").
		Do()
	if err != nil {
		return nil, fmt.Errorf("failed to get file %s: %w", fileID, err)
	}

	return convertToFileInfo(file), nil
}

// DownloadFile downloads the content of a file
func (c *Client) DownloadFile(ctx context.Context, fileID string) (io.ReadCloser, error) {
	if fileID == "" {
		return nil, fmt.Errorf("fileID is required")
	}

	resp, err := c.service.Files.Get(fileID).
		Context(ctx).
		Download()
	if err != nil {
		return nil, fmt.Errorf("failed to download file %s: %w", fileID, err)
	}

	return resp.Body, nil
}

// DeleteFile deletes a file from Google Drive
func (c *Client) DeleteFile(ctx context.Context, fileID string) error {
	if fileID == "" {
		return fmt.Errorf("fileID is required")
	}

	err := c.service.Files.Delete(fileID).Context(ctx).Do()
	if err != nil {
		return fmt.Errorf("failed to delete file %s: %w", fileID, err)
	}

	return nil
}

// CreateFolder creates a new folder in Google Drive
func (c *Client) CreateFolder(ctx context.Context, name string, parentFolders []string) (*FileInfo, error) {
	if name == "" {
		return nil, fmt.Errorf("folder name is required")
	}

	file := &drive.File{
		Name:     name,
		MimeType: FolderMimeType,
	}

	if len(parentFolders) > 0 {
		file.Parents = parentFolders
	}

	driveFile, err := c.service.Files.Create(file).
		Context(ctx).
		Fields("id, name, mimeType, createdTime, modifiedTime, webViewLink, parents, owners, shared, trashed").
		Do()
	if err != nil {
		return nil, fmt.Errorf("failed to create folder: %w", err)
	}

	return convertToFileInfo(driveFile), nil
}

// MoveFile moves or renames a file
func (c *Client) MoveFile(ctx context.Context, fileID string, options *MoveOptions) (*FileInfo, error) {
	if fileID == "" {
		return nil, fmt.Errorf("fileID is required")
	}
	if options == nil {
		return nil, fmt.Errorf("move options are required")
	}

	update := &drive.File{}
	if options.NewName != "" {
		update.Name = options.NewName
	}

	call := c.service.Files.Update(fileID, update).
		Context(ctx).
		Fields("id, name, mimeType, size, createdTime, modifiedTime, webViewLink, webContentLink, parents, owners, shared, trashed")

	if len(options.AddParents) > 0 {
		addParentsStr := ""
		for i, p := range options.AddParents {
			if i > 0 {
				addParentsStr += ","
			}
			addParentsStr += p
		}
		call = call.AddParents(addParentsStr)
	}

	if len(options.RemoveParents) > 0 {
		removeParentsStr := ""
		for i, p := range options.RemoveParents {
			if i > 0 {
				removeParentsStr += ","
			}
			removeParentsStr += p
		}
		call = call.RemoveParents(removeParentsStr)
	}

	driveFile, err := call.Do()
	if err != nil {
		return nil, fmt.Errorf("failed to move file: %w", err)
	}

	return convertToFileInfo(driveFile), nil
}

// ShareFile creates a permission on a file to share it
func (c *Client) ShareFile(ctx context.Context, fileID string, options *ShareOptions) (*Permission, error) {
	if fileID == "" {
		return nil, fmt.Errorf("fileID is required")
	}
	if options == nil {
		return nil, fmt.Errorf("share options are required")
	}
	if options.Type == "" {
		return nil, fmt.Errorf("permission type is required")
	}
	if options.Role == "" {
		return nil, fmt.Errorf("permission role is required")
	}

	permission := &drive.Permission{
		Type: options.Type,
		Role: options.Role,
	}

	if options.EmailAddress != "" {
		permission.EmailAddress = options.EmailAddress
	}
	if options.Domain != "" {
		permission.Domain = options.Domain
	}

	call := c.service.Permissions.Create(fileID, permission).
		Context(ctx).
		Fields("id, type, role, emailAddress, domain, displayName")

	if options.SendNotificationEmail {
		call = call.SendNotificationEmail(true)
		if options.EmailMessage != "" {
			call = call.EmailMessage(options.EmailMessage)
		}
	}

	drivePermission, err := call.Do()
	if err != nil {
		return nil, fmt.Errorf("failed to share file: %w", err)
	}

	return convertToPermission(drivePermission), nil
}

// RemovePermission removes a permission from a file
func (c *Client) RemovePermission(ctx context.Context, fileID, permissionID string) error {
	if fileID == "" {
		return fmt.Errorf("fileID is required")
	}
	if permissionID == "" {
		return fmt.Errorf("permissionID is required")
	}

	err := c.service.Permissions.Delete(fileID, permissionID).Context(ctx).Do()
	if err != nil {
		return fmt.Errorf("failed to remove permission: %w", err)
	}

	return nil
}

// ListPermissions lists all permissions for a file
func (c *Client) ListPermissions(ctx context.Context, fileID string) ([]*Permission, error) {
	if fileID == "" {
		return nil, fmt.Errorf("fileID is required")
	}

	permList, err := c.service.Permissions.List(fileID).
		Context(ctx).
		Fields("permissions(id, type, role, emailAddress, domain, displayName)").
		Do()
	if err != nil {
		return nil, fmt.Errorf("failed to list permissions: %w", err)
	}

	permissions := make([]*Permission, len(permList.Permissions))
	for i, p := range permList.Permissions {
		permissions[i] = convertToPermission(p)
	}

	return permissions, nil
}

// convertToFileInfo converts a Drive API File to our FileInfo type
func convertToFileInfo(f *drive.File) *FileInfo {
	fileInfo := &FileInfo{
		ID:             f.Id,
		Name:           f.Name,
		MimeType:       f.MimeType,
		Size:           f.Size,
		WebViewLink:    f.WebViewLink,
		WebContentLink: f.WebContentLink,
		Parents:        f.Parents,
		Shared:         f.Shared,
		Trashed:        f.Trashed,
	}

	// Parse timestamps
	if f.CreatedTime != "" {
		if t, err := time.Parse(time.RFC3339, f.CreatedTime); err == nil {
			fileInfo.CreatedTime = t
		}
	}
	if f.ModifiedTime != "" {
		if t, err := time.Parse(time.RFC3339, f.ModifiedTime); err == nil {
			fileInfo.ModifiedTime = t
		}
	}
	if f.TrashedTime != "" {
		if t, err := time.Parse(time.RFC3339, f.TrashedTime); err == nil {
			fileInfo.TrashedTime = &t
		}
	}

	// Convert owners
	for _, owner := range f.Owners {
		fileInfo.Owners = append(fileInfo.Owners, User{
			DisplayName:  owner.DisplayName,
			EmailAddress: owner.EmailAddress,
			PhotoLink:    owner.PhotoLink,
		})
	}

	// Convert permissions if present
	for _, perm := range f.Permissions {
		fileInfo.Permissions = append(fileInfo.Permissions, *convertToPermission(perm))
	}

	return fileInfo
}

// convertToPermission converts a Drive API Permission to our Permission type
func convertToPermission(p *drive.Permission) *Permission {
	return &Permission{
		ID:           p.Id,
		Type:         p.Type,
		Role:         p.Role,
		EmailAddress: p.EmailAddress,
		Domain:       p.Domain,
		DisplayName:  p.DisplayName,
	}
}
