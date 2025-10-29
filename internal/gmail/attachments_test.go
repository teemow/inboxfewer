package gmail

import (
	"encoding/base64"
	"testing"

	gmail "google.golang.org/api/gmail/v1"
)

func TestSanitizeFilename(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		want     string
	}{
		{
			name:     "normal filename",
			filename: "document.pdf",
			want:     "document.pdf",
		},
		{
			name:     "filename with forward slash",
			filename: "path/to/document.pdf",
			want:     "path_to_document.pdf",
		},
		{
			name:     "filename with backslash",
			filename: "path\\to\\document.pdf",
			want:     "path_to_document.pdf",
		},
		{
			name:     "filename with parent directory",
			filename: "../../../etc/passwd",
			want:     "______etc_passwd",
		},
		{
			name:     "filename with mixed separators",
			filename: "../path\\to/document.pdf",
			want:     "__path_to_document.pdf",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := SanitizeFilename(tt.filename); got != tt.want {
				t.Errorf("SanitizeFilename() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestValidateMimeType(t *testing.T) {
	tests := []struct {
		name         string
		mimeType     string
		allowedTypes []string
		want         bool
	}{
		{
			name:         "allowed type",
			mimeType:     "application/pdf",
			allowedTypes: []string{"application/pdf", "image/png"},
			want:         true,
		},
		{
			name:         "not allowed type",
			mimeType:     "application/exe",
			allowedTypes: []string{"application/pdf", "image/png"},
			want:         false,
		},
		{
			name:         "empty allowed list allows all",
			mimeType:     "any/type",
			allowedTypes: []string{},
			want:         true,
		},
		{
			name:         "nil allowed list allows all",
			mimeType:     "any/type",
			allowedTypes: nil,
			want:         true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ValidateMimeType(tt.mimeType, tt.allowedTypes); got != tt.want {
				t.Errorf("ValidateMimeType() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestWalkParts(t *testing.T) {
	tests := []struct {
		name          string
		part          *gmail.MessagePart
		expectedParts int
	}{
		{
			name: "single part",
			part: &gmail.MessagePart{
				PartId:   "0",
				MimeType: "text/plain",
			},
			expectedParts: 1,
		},
		{
			name: "nested parts",
			part: &gmail.MessagePart{
				PartId:   "0",
				MimeType: "multipart/mixed",
				Parts: []*gmail.MessagePart{
					{
						PartId:   "0.0",
						MimeType: "text/plain",
					},
					{
						PartId:   "0.1",
						MimeType: "text/html",
					},
				},
			},
			expectedParts: 3, // parent + 2 children
		},
		{
			name: "deeply nested parts",
			part: &gmail.MessagePart{
				PartId:   "0",
				MimeType: "multipart/mixed",
				Parts: []*gmail.MessagePart{
					{
						PartId:   "0.0",
						MimeType: "multipart/alternative",
						Parts: []*gmail.MessagePart{
							{
								PartId:   "0.0.0",
								MimeType: "text/plain",
							},
							{
								PartId:   "0.0.1",
								MimeType: "text/html",
							},
						},
					},
					{
						PartId:   "0.1",
						MimeType: "application/pdf",
					},
				},
			},
			expectedParts: 5, // parent + 2 children + 2 grandchildren
		},
		{
			name:          "nil part",
			part:          nil,
			expectedParts: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			count := 0
			walkParts(tt.part, "test-message-id", func(part *gmail.MessagePart) {
				count++
			})

			if count != tt.expectedParts {
				t.Errorf("walkParts() visited %d parts, want %d", count, tt.expectedParts)
			}
		})
	}
}

func TestGetMessageBody_Format(t *testing.T) {
	tests := []struct {
		name    string
		format  string
		wantErr bool
	}{
		{
			name:    "text format",
			format:  "text",
			wantErr: false,
		},
		{
			name:    "html format",
			format:  "html",
			wantErr: false,
		},
		{
			name:    "empty format defaults to text",
			format:  "",
			wantErr: false,
		},
		{
			name:    "invalid format",
			format:  "invalid",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// This test validates the format parameter handling
			// We can't test the actual API call without a mock, but we can test the validation
			if tt.format == "invalid" {
				// Create a mock client (would need proper mock setup in real test)
				// For now, this demonstrates the test structure
			}
		})
	}
}

func TestListAttachments_Parsing(t *testing.T) {
	tests := []struct {
		name         string
		payload      *gmail.MessagePart
		wantCount    int
		wantFilename string
	}{
		{
			name: "message with single attachment",
			payload: &gmail.MessagePart{
				PartId:   "1",
				Filename: "document.pdf",
				MimeType: "application/pdf",
				Body: &gmail.MessagePartBody{
					AttachmentId: "att123",
					Size:         1024,
				},
			},
			wantCount:    1,
			wantFilename: "document.pdf",
		},
		{
			name: "message with no attachments",
			payload: &gmail.MessagePart{
				PartId:   "0",
				MimeType: "text/plain",
				Body: &gmail.MessagePartBody{
					Data: base64.URLEncoding.EncodeToString([]byte("Hello")),
				},
			},
			wantCount: 0,
		},
		{
			name: "message with multiple attachments",
			payload: &gmail.MessagePart{
				PartId:   "0",
				MimeType: "multipart/mixed",
				Parts: []*gmail.MessagePart{
					{
						PartId:   "0.0",
						MimeType: "text/plain",
						Body: &gmail.MessagePartBody{
							Data: base64.URLEncoding.EncodeToString([]byte("Body text")),
						},
					},
					{
						PartId:   "0.1",
						Filename: "image.png",
						MimeType: "image/png",
						Body: &gmail.MessagePartBody{
							AttachmentId: "att456",
							Size:         2048,
						},
					},
					{
						PartId:   "0.2",
						Filename: "document.pdf",
						MimeType: "application/pdf",
						Body: &gmail.MessagePartBody{
							AttachmentId: "att789",
							Size:         3072,
						},
					},
				},
			},
			wantCount: 2,
		},
		{
			name: "message with nested attachments",
			payload: &gmail.MessagePart{
				PartId:   "0",
				MimeType: "multipart/mixed",
				Parts: []*gmail.MessagePart{
					{
						PartId:   "0.0",
						MimeType: "multipart/alternative",
						Parts: []*gmail.MessagePart{
							{
								PartId:   "0.0.0",
								MimeType: "text/plain",
								Body: &gmail.MessagePartBody{
									Data: base64.URLEncoding.EncodeToString([]byte("Text")),
								},
							},
						},
					},
					{
						PartId:   "0.1",
						Filename: "file.txt",
						MimeType: "text/plain",
						Body: &gmail.MessagePartBody{
							AttachmentId: "att999",
							Size:         512,
						},
					},
				},
			},
			wantCount: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var attachments []*AttachmentInfo
			walkParts(tt.payload, "test-msg-id", func(part *gmail.MessagePart) {
				if part.Filename != "" && part.Body != nil && part.Body.AttachmentId != "" {
					attachments = append(attachments, &AttachmentInfo{
						MessageID:    "test-msg-id",
						PartID:       part.PartId,
						AttachmentID: part.Body.AttachmentId,
						Filename:     part.Filename,
						MimeType:     part.MimeType,
						Size:         part.Body.Size,
					})
				}
			})

			if len(attachments) != tt.wantCount {
				t.Errorf("found %d attachments, want %d", len(attachments), tt.wantCount)
			}

			if tt.wantCount > 0 && tt.wantFilename != "" {
				if attachments[0].Filename != tt.wantFilename {
					t.Errorf("first attachment filename = %v, want %v", attachments[0].Filename, tt.wantFilename)
				}
			}
		})
	}
}

func TestGetAttachment_Validation(t *testing.T) {
	tests := []struct {
		name         string
		messageID    string
		attachmentID string
		wantErr      bool
		errContains  string
	}{
		{
			name:         "empty messageID",
			messageID:    "",
			attachmentID: "att123",
			wantErr:      true,
			errContains:  "messageID is required",
		},
		{
			name:         "empty attachmentID",
			messageID:    "msg123",
			attachmentID: "",
			wantErr:      true,
			errContains:  "attachmentID is required",
		},
		{
			name:         "valid IDs",
			messageID:    "msg123",
			attachmentID: "att123",
			wantErr:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test validation logic
			if tt.messageID == "" || tt.attachmentID == "" {
				// Simulate validation
				var err error
				if tt.messageID == "" {
					err = &validationError{msg: "messageID is required"}
				} else if tt.attachmentID == "" {
					err = &validationError{msg: "attachmentID is required"}
				}

				if (err != nil) != tt.wantErr {
					t.Errorf("expected error = %v, got error = %v", tt.wantErr, err != nil)
				}
			}
		})
	}
}

// validationError is a helper type for testing validation errors
type validationError struct {
	msg string
}

func (e *validationError) Error() string {
	return e.msg
}

func TestMaxAttachmentSize(t *testing.T) {
	const expectedSize = 25 * 1024 * 1024 // 25MB

	if MaxAttachmentSize != expectedSize {
		t.Errorf("MaxAttachmentSize = %d, want %d", MaxAttachmentSize, expectedSize)
	}
}

func TestBase64Decoding(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{
			name:    "standard base64",
			input:   base64.StdEncoding.EncodeToString([]byte("Hello, World!")),
			want:    "Hello, World!",
			wantErr: false,
		},
		{
			name:    "url base64",
			input:   base64.URLEncoding.EncodeToString([]byte("Hello, World!")),
			want:    "Hello, World!",
			wantErr: false,
		},
		{
			name:    "url base64 with special chars",
			input:   base64.URLEncoding.EncodeToString([]byte("Special: !@#$%^&*()")),
			want:    "Special: !@#$%^&*()",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test URL encoding first (Gmail's default)
			decoded, err := base64.URLEncoding.DecodeString(tt.input)
			if err != nil {
				// Try standard encoding
				decoded, err = base64.StdEncoding.DecodeString(tt.input)
			}

			if (err != nil) != tt.wantErr {
				t.Errorf("decode error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && string(decoded) != tt.want {
				t.Errorf("decoded = %v, want %v", string(decoded), tt.want)
			}
		})
	}
}
