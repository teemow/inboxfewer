package gmail

import (
	"testing"
)

func TestExtractDocLinks(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		expected []*DocLink
	}{
		{
			name: "Single Google Doc link",
			text: "Check out this document: https://docs.google.com/document/d/1ABC123xyz/edit",
			expected: []*DocLink{
				{
					URL:        "https://docs.google.com/document/d/1ABC123xyz",
					DocumentID: "1ABC123xyz",
					Type:       "document",
				},
			},
		},
		{
			name: "Google Doc with query parameters",
			text: "Here's the link: https://docs.google.com/document/d/1XYZ789abc/edit?usp=sharing",
			expected: []*DocLink{
				{
					URL:        "https://docs.google.com/document/d/1XYZ789abc",
					DocumentID: "1XYZ789abc",
					Type:       "document",
				},
			},
		},
		{
			name: "Google Sheets link",
			text: "See the spreadsheet: https://docs.google.com/spreadsheets/d/1Sheet123/edit",
			expected: []*DocLink{
				{
					URL:        "https://docs.google.com/spreadsheets/d/1Sheet123",
					DocumentID: "1Sheet123",
					Type:       "spreadsheet",
				},
			},
		},
		{
			name: "Google Slides link",
			text: "View presentation: https://docs.google.com/presentation/d/1Slide456/edit",
			expected: []*DocLink{
				{
					URL:        "https://docs.google.com/presentation/d/1Slide456",
					DocumentID: "1Slide456",
					Type:       "presentation",
				},
			},
		},
		{
			name: "Google Drive file link",
			text: "File is here: https://drive.google.com/file/d/1File789/view",
			expected: []*DocLink{
				{
					URL:        "https://drive.google.com/file/d/1File789",
					DocumentID: "1File789",
					Type:       "drive",
				},
			},
		},
		{
			name: "Google Drive open link with query param",
			text: "Open this: https://drive.google.com/open?id=1DriveABC",
			expected: []*DocLink{
				{
					URL:        "https://drive.google.com/open?id=1DriveABC",
					DocumentID: "1DriveABC",
					Type:       "drive",
				},
			},
		},
		{
			name: "Multiple links of different types",
			text: `Meeting notes: https://docs.google.com/document/d/1Doc1/edit
Budget sheet: https://docs.google.com/spreadsheets/d/1Sheet1/edit
Presentation: https://docs.google.com/presentation/d/1Slide1/edit`,
			expected: []*DocLink{
				{
					URL:        "https://docs.google.com/document/d/1Doc1",
					DocumentID: "1Doc1",
					Type:       "document",
				},
				{
					URL:        "https://docs.google.com/spreadsheets/d/1Sheet1",
					DocumentID: "1Sheet1",
					Type:       "spreadsheet",
				},
				{
					URL:        "https://docs.google.com/presentation/d/1Slide1",
					DocumentID: "1Slide1",
					Type:       "presentation",
				},
			},
		},
		{
			name: "Link with special characters in document ID",
			text: "Document: https://docs.google.com/document/d/1ABC-def_GHI123/edit",
			expected: []*DocLink{
				{
					URL:        "https://docs.google.com/document/d/1ABC-def_GHI123",
					DocumentID: "1ABC-def_GHI123",
					Type:       "document",
				},
			},
		},
		{
			name: "Duplicate document IDs (should return only one)",
			text: `First: https://docs.google.com/document/d/1ABC123/edit
Second: https://docs.google.com/document/d/1ABC123/edit?usp=sharing`,
			expected: []*DocLink{
				{
					URL:        "https://docs.google.com/document/d/1ABC123",
					DocumentID: "1ABC123",
					Type:       "document",
				},
			},
		},
		{
			name:     "No Google links",
			text:     "This is just plain text with no links.",
			expected: []*DocLink{},
		},
		{
			name:     "Empty text",
			text:     "",
			expected: []*DocLink{},
		},
		{
			name: "Link in HTML anchor tag",
			text: `<a href="https://docs.google.com/document/d/1HTMLDoc/edit">Click here</a>`,
			expected: []*DocLink{
				{
					URL:        "https://docs.google.com/document/d/1HTMLDoc",
					DocumentID: "1HTMLDoc",
					Type:       "document",
				},
			},
		},
		{
			name: "HTTP and HTTPS links",
			text: "HTTP: http://docs.google.com/document/d/1HTTP/edit HTTPS: https://docs.google.com/document/d/1HTTPS/edit",
			expected: []*DocLink{
				{
					URL:        "http://docs.google.com/document/d/1HTTP",
					DocumentID: "1HTTP",
					Type:       "document",
				},
				{
					URL:        "https://docs.google.com/document/d/1HTTPS",
					DocumentID: "1HTTPS",
					Type:       "document",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ExtractDocLinks(tt.text)

			if len(result) != len(tt.expected) {
				t.Errorf("ExtractDocLinks() returned %d links, expected %d", len(result), len(tt.expected))
				return
			}

			for i, link := range result {
				if i >= len(tt.expected) {
					break
				}
				expected := tt.expected[i]

				if link.URL != expected.URL {
					t.Errorf("Link %d: URL = %v, want %v", i, link.URL, expected.URL)
				}
				if link.DocumentID != expected.DocumentID {
					t.Errorf("Link %d: DocumentID = %v, want %v", i, link.DocumentID, expected.DocumentID)
				}
				if link.Type != expected.Type {
					t.Errorf("Link %d: Type = %v, want %v", i, link.Type, expected.Type)
				}
			}
		})
	}
}

func TestParseDocumentID(t *testing.T) {
	tests := []struct {
		name      string
		url       string
		expected  string
		expectErr bool
	}{
		{
			name:     "Google Doc URL",
			url:      "https://docs.google.com/document/d/1ABC123xyz/edit",
			expected: "1ABC123xyz",
		},
		{
			name:     "Google Doc with query params",
			url:      "https://docs.google.com/document/d/1XYZ789/edit?usp=sharing",
			expected: "1XYZ789",
		},
		{
			name:     "Google Sheets URL",
			url:      "https://docs.google.com/spreadsheets/d/1Sheet456/edit",
			expected: "1Sheet456",
		},
		{
			name:     "Google Slides URL",
			url:      "https://docs.google.com/presentation/d/1Slide789/edit",
			expected: "1Slide789",
		},
		{
			name:     "Google Drive file URL",
			url:      "https://drive.google.com/file/d/1File123/view",
			expected: "1File123",
		},
		{
			name:     "Google Drive open URL",
			url:      "https://drive.google.com/open?id=1DriveXYZ",
			expected: "1DriveXYZ",
		},
		{
			name:     "Document ID with hyphens and underscores",
			url:      "https://docs.google.com/document/d/1ABC-def_GHI/edit",
			expected: "1ABC-def_GHI",
		},
		{
			name:      "Empty URL",
			url:       "",
			expectErr: true,
		},
		{
			name:      "Invalid URL",
			url:       "not a url",
			expectErr: true,
		},
		{
			name:      "Non-Google URL",
			url:       "https://example.com/document/123",
			expectErr: true,
		},
		{
			name:      "Google URL without document ID",
			url:       "https://docs.google.com/",
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseDocumentID(tt.url)

			if tt.expectErr {
				if err == nil {
					t.Errorf("ParseDocumentID() expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("ParseDocumentID() unexpected error: %v", err)
				return
			}

			if result != tt.expected {
				t.Errorf("ParseDocumentID() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestValidateDocumentID(t *testing.T) {
	tests := []struct {
		name     string
		docID    string
		expected bool
	}{
		{
			name:     "Valid document ID",
			docID:    "1ABC123xyz",
			expected: true,
		},
		{
			name:     "Valid with hyphens",
			docID:    "1ABC-123-xyz",
			expected: true,
		},
		{
			name:     "Valid with underscores",
			docID:    "1ABC_123_xyz",
			expected: true,
		},
		{
			name:     "Valid long ID",
			docID:    "1ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789",
			expected: true,
		},
		{
			name:     "Empty string",
			docID:    "",
			expected: false,
		},
		{
			name:     "Too short",
			docID:    "1ABC",
			expected: false,
		},
		{
			name:     "Invalid characters (spaces)",
			docID:    "1ABC 123",
			expected: false,
		},
		{
			name:     "Invalid characters (special chars)",
			docID:    "1ABC@123",
			expected: false,
		},
		{
			name:     "Invalid characters (slashes)",
			docID:    "1ABC/123",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ValidateDocumentID(tt.docID)
			if result != tt.expected {
				t.Errorf("ValidateDocumentID(%q) = %v, want %v", tt.docID, result, tt.expected)
			}
		})
	}
}
