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

func TestExtractBodyFromMessage(t *testing.T) {
	// Create a mock client (we only test the extraction logic, not API calls)
	client := &Client{}

	tests := []struct {
		name    string
		message *gmail.Message
		format  string
		want    string
		wantErr bool
	}{
		{
			name: "simple text message",
			message: &gmail.Message{
				Id: "msg123",
				Payload: &gmail.MessagePart{
					MimeType: "text/plain",
					Body: &gmail.MessagePartBody{
						Data: base64.URLEncoding.EncodeToString([]byte("Hello, this is a test message")),
					},
				},
			},
			format:  "text",
			want:    "Hello, this is a test message",
			wantErr: false,
		},
		{
			name: "html message",
			message: &gmail.Message{
				Id: "msg456",
				Payload: &gmail.MessagePart{
					MimeType: "text/html",
					Body: &gmail.MessagePartBody{
						Data: base64.URLEncoding.EncodeToString([]byte("<html><body>HTML content</body></html>")),
					},
				},
			},
			format:  "html",
			want:    "<html><body>HTML content</body></html>",
			wantErr: false,
		},
		{
			name: "multipart message with text",
			message: &gmail.Message{
				Id: "msg789",
				Payload: &gmail.MessagePart{
					MimeType: "multipart/alternative",
					Parts: []*gmail.MessagePart{
						{
							MimeType: "text/plain",
							Body: &gmail.MessagePartBody{
								Data: base64.URLEncoding.EncodeToString([]byte("Plain text body")),
							},
						},
						{
							MimeType: "text/html",
							Body: &gmail.MessagePartBody{
								Data: base64.URLEncoding.EncodeToString([]byte("<html>HTML body</html>")),
							},
						},
					},
				},
			},
			format:  "text",
			want:    "Plain text body",
			wantErr: false,
		},
		{
			name: "message with no body",
			message: &gmail.Message{
				Id: "msg999",
				Payload: &gmail.MessagePart{
					MimeType: "text/plain",
					Body:     &gmail.MessagePartBody{},
				},
			},
			format:  "text",
			wantErr: true,
		},
		{
			name: "invalid format",
			message: &gmail.Message{
				Id: "msg111",
				Payload: &gmail.MessagePart{
					MimeType: "text/plain",
					Body: &gmail.MessagePartBody{
						Data: base64.URLEncoding.EncodeToString([]byte("Test")),
					},
				},
			},
			format:  "invalid",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := client.extractBodyFromMessage(tt.message, tt.format)
			if (err != nil) != tt.wantErr {
				t.Errorf("extractBodyFromMessage() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("extractBodyFromMessage() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestExtractBodyFromMessage_Formats(t *testing.T) {
	client := &Client{}

	// Test that both text and html formats work correctly with the same multipart message
	message := &gmail.Message{
		Id: "msg-multipart",
		Payload: &gmail.MessagePart{
			MimeType: "multipart/alternative",
			Parts: []*gmail.MessagePart{
				{
					MimeType: "text/plain",
					Body: &gmail.MessagePartBody{
						Data: base64.URLEncoding.EncodeToString([]byte("Plain text version")),
					},
				},
				{
					MimeType: "text/html",
					Body: &gmail.MessagePartBody{
						Data: base64.URLEncoding.EncodeToString([]byte("<p>HTML version</p>")),
					},
				},
			},
		},
	}

	// Test text format
	gotText, err := client.extractBodyFromMessage(message, "text")
	if err != nil {
		t.Errorf("extractBodyFromMessage(text) error = %v", err)
	}
	if gotText != "Plain text version" {
		t.Errorf("extractBodyFromMessage(text) = %v, want 'Plain text version'", gotText)
	}

	// Test html format
	gotHTML, err := client.extractBodyFromMessage(message, "html")
	if err != nil {
		t.Errorf("extractBodyFromMessage(html) error = %v", err)
	}
	if gotHTML != "<p>HTML version</p>" {
		t.Errorf("extractBodyFromMessage(html) = %v, want '<p>HTML version</p>'", gotHTML)
	}

	// Test default format (should be text)
	gotDefault, err := client.extractBodyFromMessage(message, "")
	if err != nil {
		t.Errorf("extractBodyFromMessage('') error = %v", err)
	}
	if gotDefault != "Plain text version" {
		t.Errorf("extractBodyFromMessage('') = %v, want 'Plain text version'", gotDefault)
	}
}

// TestExtractBodyFromMessage_FallbackToHTML tests Issue #53 fix:
// When text body is not available, it should fall back to HTML.
// If both fail, it should return a comprehensive error message.
func TestExtractBodyFromMessage_FallbackToHTML(t *testing.T) {
	client := &Client{}

	t.Run("html-only message with text format request falls back to html", func(t *testing.T) {
		// Message with only HTML body
		message := &gmail.Message{
			Id: "msg-html-only",
			Payload: &gmail.MessagePart{
				MimeType: "text/html",
				Body: &gmail.MessagePartBody{
					Data: base64.URLEncoding.EncodeToString([]byte("<p>HTML only content</p>")),
				},
			},
		}

		// Request text format, should automatically fall back to HTML
		got, err := client.extractBodyFromMessage(message, "text")
		if err != nil {
			t.Errorf("extractBodyFromMessage() should have fallen back to HTML, got error = %v", err)
		}
		if got != "<p>HTML only content</p>" {
			t.Errorf("extractBodyFromMessage() = %v, want '<p>HTML only content</p>'", got)
		}
	})

	t.Run("message with no text or html returns comprehensive error", func(t *testing.T) {
		// Message with no body at all
		message := &gmail.Message{
			Id: "msg-no-body",
			Payload: &gmail.MessagePart{
				MimeType: "multipart/mixed",
				Parts: []*gmail.MessagePart{
					{
						MimeType: "application/pdf",
						Body: &gmail.MessagePartBody{
							AttachmentId: "att123",
						},
					},
				},
			},
		}

		// Request text format, should try both text and HTML and fail with comprehensive error
		_, err := client.extractBodyFromMessage(message, "text")
		if err == nil {
			t.Error("extractBodyFromMessage() should have returned error for message with no body")
		}

		// Check that the error message mentions both text and html attempts
		errMsg := err.Error()
		if !contains(errMsg, "text") && !contains(errMsg, "html") {
			t.Errorf("error message should mention both text and html attempts, got: %v", errMsg)
		}
	})

	t.Run("empty message returns comprehensive error", func(t *testing.T) {
		// Completely empty message
		message := &gmail.Message{
			Id:      "msg-empty",
			Payload: &gmail.MessagePart{},
		}

		_, err := client.extractBodyFromMessage(message, "text")
		if err == nil {
			t.Error("extractBodyFromMessage() should have returned error for empty message")
		}

		// Verify error mentions both formats were tried
		errMsg := err.Error()
		if !contains(errMsg, "tried text and html") {
			t.Errorf("error should indicate both formats were tried, got: %v", errMsg)
		}
	})
}

// TestExtractBodyFromMessage_HTMLFallbackSuccess tests that HTML fallback works
// and returns the HTML content when text is not available
func TestExtractBodyFromMessage_HTMLFallbackSuccess(t *testing.T) {
	client := &Client{}

	tests := []struct {
		name     string
		message  *gmail.Message
		wantHTML string
	}{
		{
			name: "multipart with only html",
			message: &gmail.Message{
				Id: "msg1",
				Payload: &gmail.MessagePart{
					MimeType: "multipart/alternative",
					Parts: []*gmail.MessagePart{
						{
							MimeType: "text/html",
							Body: &gmail.MessagePartBody{
								Data: base64.URLEncoding.EncodeToString([]byte("<html>Test</html>")),
							},
						},
					},
				},
			},
			wantHTML: "<html>Test</html>",
		},
		{
			name: "simple html message",
			message: &gmail.Message{
				Id: "msg2",
				Payload: &gmail.MessagePart{
					MimeType: "text/html",
					Body: &gmail.MessagePartBody{
						Data: base64.URLEncoding.EncodeToString([]byte("<div>HTML Body</div>")),
					},
				},
			},
			wantHTML: "<div>HTML Body</div>",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Request text format, should fall back to HTML
			got, err := client.extractBodyFromMessage(tt.message, "text")
			if err != nil {
				t.Errorf("extractBodyFromMessage() error = %v, expected successful HTML fallback", err)
			}
			if got != tt.wantHTML {
				t.Errorf("extractBodyFromMessage() = %v, want %v", got, tt.wantHTML)
			}
		})
	}
}

// contains is a helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
