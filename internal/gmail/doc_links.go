package gmail

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"
)

// DocLink represents a Google Docs/Drive link found in an email
type DocLink struct {
	URL        string `json:"url"`
	DocumentID string `json:"documentId"`
	Type       string `json:"type"` // "document", "spreadsheet", "presentation", "drive"
}

// Regular expressions for matching Google Docs/Drive URLs
var (
	// Google Docs: https://docs.google.com/document/d/{documentId}/...
	docsDocumentRegex = regexp.MustCompile(`https?://docs\.google\.com/document/d/([a-zA-Z0-9_-]+)`)

	// Google Sheets: https://docs.google.com/spreadsheets/d/{documentId}/...
	docsSpreadsheetRegex = regexp.MustCompile(`https?://docs\.google\.com/spreadsheets/d/([a-zA-Z0-9_-]+)`)

	// Google Slides: https://docs.google.com/presentation/d/{documentId}/...
	docsPresentationRegex = regexp.MustCompile(`https?://docs\.google\.com/presentation/d/([a-zA-Z0-9_-]+)`)

	// Google Drive file: https://drive.google.com/file/d/{fileId}/...
	driveFileRegex = regexp.MustCompile(`https?://drive\.google\.com/file/d/([a-zA-Z0-9_-]+)`)

	// Google Drive open: https://drive.google.com/open?id={fileId}
	driveOpenRegex = regexp.MustCompile(`https?://drive\.google\.com/open\?id=([a-zA-Z0-9_-]+)`)
)

// ExtractDocLinks parses Google Docs/Drive URLs from text
func ExtractDocLinks(text string) []*DocLink {
	var links []*DocLink
	seen := make(map[string]bool) // Track seen document IDs to avoid duplicates

	// Try to match each URL pattern
	patterns := []struct {
		regex   *regexp.Regexp
		docType string
	}{
		{docsDocumentRegex, "document"},
		{docsSpreadsheetRegex, "spreadsheet"},
		{docsPresentationRegex, "presentation"},
		{driveFileRegex, "drive"},
		{driveOpenRegex, "drive"},
	}

	for _, pattern := range patterns {
		matches := pattern.regex.FindAllStringSubmatch(text, -1)
		for _, match := range matches {
			if len(match) >= 2 {
				docID := match[1]
				fullURL := match[0]

				// Skip if we've already seen this document ID
				if seen[docID] {
					continue
				}
				seen[docID] = true

				links = append(links, &DocLink{
					URL:        fullURL,
					DocumentID: docID,
					Type:       pattern.docType,
				})
			}
		}
	}

	return links
}

// ParseDocumentID extracts the document ID from a Google Docs URL
func ParseDocumentID(urlStr string) (string, error) {
	if urlStr == "" {
		return "", fmt.Errorf("URL is empty")
	}

	// Try to parse as a standard URL first
	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		return "", fmt.Errorf("invalid URL: %w", err)
	}

	// Check for docs.google.com URLs
	if strings.Contains(parsedURL.Host, "docs.google.com") {
		// Pattern: /document/d/{id}/... or /spreadsheets/d/{id}/... or /presentation/d/{id}/...
		parts := strings.Split(parsedURL.Path, "/")
		for i, part := range parts {
			if part == "d" && i+1 < len(parts) {
				return parts[i+1], nil
			}
		}
	}

	// Check for drive.google.com URLs
	if strings.Contains(parsedURL.Host, "drive.google.com") {
		// Pattern: /file/d/{id}/... or /open?id={id}
		parts := strings.Split(parsedURL.Path, "/")
		for i, part := range parts {
			if part == "d" && i+1 < len(parts) {
				return parts[i+1], nil
			}
		}

		// Check query parameter: ?id={id}
		if id := parsedURL.Query().Get("id"); id != "" {
			return id, nil
		}
	}

	return "", fmt.Errorf("could not extract document ID from URL: %s", urlStr)
}

// ValidateDocumentID checks if a document ID has a valid format
func ValidateDocumentID(docID string) bool {
	if docID == "" {
		return false
	}

	// Google document IDs are typically alphanumeric with hyphens and underscores
	// Length is typically 30-50 characters
	validIDRegex := regexp.MustCompile(`^[a-zA-Z0-9_-]{10,100}$`)
	return validIDRegex.MatchString(docID)
}
