// Package docs provides functionality for interacting with Google Docs API.
//
// This package includes a client for authenticating with Google Docs API using OAuth2,
// retrieving document content, and converting documents to various formats (Markdown, plain text).
//
// The package handles:
//   - OAuth2 authentication with token caching
//   - Document retrieval via Google Docs API
//   - Document metadata retrieval via Google Drive API
//   - Document content conversion to Markdown and plain text formats
//
// Example usage:
//
//	client, err := docs.NewClient(context.Background())
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	doc, err := client.GetDocument("1ABC123xyz")
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	markdown, err := DocumentToMarkdown(doc)
//	if err != nil {
//	    log.Fatal(err)
//	}
package docs
