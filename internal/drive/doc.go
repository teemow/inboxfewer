// Package drive provides a client for interacting with the Google Drive API.
//
// This package enables comprehensive Google Drive file management operations including:
//   - Uploading files with metadata
//   - Listing and searching files and folders
//   - Downloading file content
//   - Deleting files
//   - Creating folders
//   - Moving and renaming files
//   - Managing file sharing and permissions
//
// The client supports multi-account functionality, allowing management of multiple
// Google accounts simultaneously. Each client instance is bound to a specific account.
//
// OAuth Authentication:
// This package uses the unified Google OAuth token from the google package.
// The OAuth scope includes full Google Drive access (drive scope), allowing read
// and write operations on all files in the user's Drive.
//
// Example usage:
//
//	ctx := context.Background()
//	client, err := drive.NewClient(ctx)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	// Upload a file
//	file, err := client.UploadFile(ctx, "document.pdf", bytes.NewReader(content), "application/pdf", nil)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	// List files
//	files, err := client.ListFiles(ctx, &drive.ListOptions{
//	    Query: "mimeType='application/pdf'",
//	    MaxResults: 10,
//	})
package drive
