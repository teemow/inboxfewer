package drive

import "time"

// FileInfo represents metadata about a file or folder in Google Drive
type FileInfo struct {
	// ID is the unique identifier for the file
	ID string `json:"id"`

	// Name is the name of the file
	Name string `json:"name"`

	// MimeType is the MIME type of the file
	MimeType string `json:"mimeType"`

	// Size is the size of the file in bytes (not populated for folders)
	Size int64 `json:"size,omitempty"`

	// CreatedTime is when the file was created
	CreatedTime time.Time `json:"createdTime"`

	// ModifiedTime is when the file was last modified
	ModifiedTime time.Time `json:"modifiedTime"`

	// WebViewLink is a link for opening the file in a relevant Google editor or viewer
	WebViewLink string `json:"webViewLink,omitempty"`

	// WebContentLink is a link for downloading the file content (not available for folders)
	WebContentLink string `json:"webContentLink,omitempty"`

	// Parents are the IDs of the parent folders
	Parents []string `json:"parents,omitempty"`

	// Owners are the owners of the file
	Owners []User `json:"owners,omitempty"`

	// Shared indicates whether the file is shared
	Shared bool `json:"shared"`

	// Permissions are the access permissions for the file
	Permissions []Permission `json:"permissions,omitempty"`

	// TrashedTime is when the file was trashed (if trashed)
	TrashedTime *time.Time `json:"trashedTime,omitempty"`

	// Trashed indicates whether the file is in the trash
	Trashed bool `json:"trashed"`
}

// User represents a Google Drive user (owner, permission holder, etc.)
type User struct {
	// DisplayName is the display name of the user
	DisplayName string `json:"displayName"`

	// EmailAddress is the email address of the user
	EmailAddress string `json:"emailAddress"`

	// PhotoLink is a link to the user's profile photo
	PhotoLink string `json:"photoLink,omitempty"`
}

// Permission represents access permissions for a file
type Permission struct {
	// ID is the unique identifier for the permission
	ID string `json:"id"`

	// Type is the type of grantee (user, group, domain, anyone)
	Type string `json:"type"`

	// Role is the role granted by this permission (owner, organizer, fileOrganizer, writer, commenter, reader)
	Role string `json:"role"`

	// EmailAddress is the email address of the user or group (if type is user or group)
	EmailAddress string `json:"emailAddress,omitempty"`

	// Domain is the domain to which this permission refers (if type is domain)
	Domain string `json:"domain,omitempty"`

	// DisplayName is the display name of the user or group
	DisplayName string `json:"displayName,omitempty"`
}

// ListOptions contains options for listing files
type ListOptions struct {
	// Query is a query for filtering the file results using Google Drive's query language
	// See https://developers.google.com/drive/api/guides/search-files
	// Examples:
	//   "name contains 'report'"
	//   "mimeType='application/pdf'"
	//   "'me' in owners"
	//   "trashed=false and 'root' in parents"
	Query string

	// MaxResults is the maximum number of files to return (max: 1000)
	MaxResults int

	// OrderBy specifies the sort order of the result set
	// Examples: "folder,modifiedTime desc,name"
	OrderBy string

	// PageToken is a token for retrieving the next page of results
	PageToken string

	// IncludeTrashed includes trashed files in results
	IncludeTrashed bool

	// Spaces is a comma-separated list of spaces to query (drive, appDataFolder, photos)
	Spaces string
}

// UploadOptions contains options for uploading a file
type UploadOptions struct {
	// ParentFolders are the IDs of parent folders where the file should be placed
	ParentFolders []string

	// Description is a short description of the file
	Description string

	// MimeType is the MIME type of the file (e.g., "application/pdf", "image/png")
	// If not specified, Drive will attempt to detect it automatically
	MimeType string

	// ModifiedTime allows setting a custom modification time
	ModifiedTime *time.Time
}

// MoveOptions contains options for moving or renaming a file
type MoveOptions struct {
	// NewName is the new name for the file (leave empty to keep current name)
	NewName string

	// AddParents are folder IDs to add as parents
	AddParents []string

	// RemoveParents are folder IDs to remove as parents
	RemoveParents []string
}

// ShareOptions contains options for sharing a file
type ShareOptions struct {
	// Type is the type of grantee: "user", "group", "domain", or "anyone"
	Type string

	// Role is the role to grant: "owner", "organizer", "fileOrganizer", "writer", "commenter", or "reader"
	Role string

	// EmailAddress is the email address (required if Type is "user" or "group")
	EmailAddress string

	// Domain is the domain name (required if Type is "domain")
	Domain string

	// SendNotificationEmail indicates whether to send a notification email
	SendNotificationEmail bool

	// EmailMessage is a custom message to include in the notification email
	EmailMessage string
}
