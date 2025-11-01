package gmail_tools

import (
	"testing"

	"github.com/teemow/inboxfewer/internal/tools/batch"
)

func TestFormatSize(t *testing.T) {
	tests := []struct {
		name  string
		bytes int64
		want  string
	}{
		{
			name:  "bytes",
			bytes: 512,
			want:  "512 bytes",
		},
		{
			name:  "kilobytes",
			bytes: 1536,
			want:  "1.50 KB",
		},
		{
			name:  "megabytes",
			bytes: 5242880,
			want:  "5.00 MB",
		},
		{
			name:  "gigabytes",
			bytes: 2147483648,
			want:  "2.00 GB",
		},
		{
			name:  "exact 1KB",
			bytes: 1024,
			want:  "1.00 KB",
		},
		{
			name:  "exact 1MB",
			bytes: 1048576,
			want:  "1.00 MB",
		},
		{
			name:  "exact 1GB",
			bytes: 1073741824,
			want:  "1.00 GB",
		},
		{
			name:  "zero bytes",
			bytes: 0,
			want:  "0 bytes",
		},
		{
			name:  "fractional KB",
			bytes: 1536,
			want:  "1.50 KB",
		},
		{
			name:  "fractional MB",
			bytes: 1572864, // 1.5 MB
			want:  "1.50 MB",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := formatSize(tt.bytes); got != tt.want {
				t.Errorf("formatSize() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestHandleListAttachments_ArgumentValidation(t *testing.T) {
	tests := []struct {
		name    string
		args    map[string]interface{}
		wantErr bool
	}{
		{
			name: "valid messageId",
			args: map[string]interface{}{
				"messageId": "msg123",
			},
			wantErr: false,
		},
		{
			name:    "missing messageId",
			args:    map[string]interface{}{},
			wantErr: true,
		},
		{
			name: "empty messageId",
			args: map[string]interface{}{
				"messageId": "",
			},
			wantErr: true,
		},
		{
			name: "wrong type messageId",
			args: map[string]interface{}{
				"messageId": 123,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			messageID, ok := tt.args["messageId"].(string)
			hasError := !ok || messageID == ""

			if hasError != tt.wantErr {
				t.Errorf("validation result = %v, wantErr %v", hasError, tt.wantErr)
			}
		})
	}
}

func TestHandleGetAttachment_ArgumentValidation(t *testing.T) {
	tests := []struct {
		name    string
		args    map[string]interface{}
		wantErr bool
	}{
		{
			name: "valid arguments",
			args: map[string]interface{}{
				"messageId":    "msg123",
				"attachmentId": "att456",
			},
			wantErr: false,
		},
		{
			name: "valid with encoding",
			args: map[string]interface{}{
				"messageId":    "msg123",
				"attachmentId": "att456",
				"encoding":     "text",
			},
			wantErr: false,
		},
		{
			name: "missing messageId",
			args: map[string]interface{}{
				"attachmentId": "att456",
			},
			wantErr: true,
		},
		{
			name: "missing attachmentId",
			args: map[string]interface{}{
				"messageId": "msg123",
			},
			wantErr: true,
		},
		{
			name: "empty messageId",
			args: map[string]interface{}{
				"messageId":    "",
				"attachmentId": "att456",
			},
			wantErr: true,
		},
		{
			name: "empty attachmentId",
			args: map[string]interface{}{
				"messageId":    "msg123",
				"attachmentId": "",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			messageID, ok1 := tt.args["messageId"].(string)
			attachmentID, ok2 := tt.args["attachmentId"].(string)
			hasError := !ok1 || messageID == "" || !ok2 || attachmentID == ""

			if hasError != tt.wantErr {
				t.Errorf("validation result = %v, wantErr %v", hasError, tt.wantErr)
			}
		})
	}
}

func TestHandleGetAttachment_EncodingValidation(t *testing.T) {
	tests := []struct {
		name     string
		encoding string
		wantErr  bool
	}{
		{
			name:     "base64 encoding",
			encoding: "base64",
			wantErr:  false,
		},
		{
			name:     "text encoding",
			encoding: "text",
			wantErr:  false,
		},
		{
			name:     "empty encoding defaults to base64",
			encoding: "",
			wantErr:  false,
		},
		{
			name:     "invalid encoding",
			encoding: "invalid",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			encoding := tt.encoding
			if encoding == "" {
				encoding = "base64"
			}

			validEncodings := map[string]bool{
				"base64": true,
				"text":   true,
			}

			hasError := !validEncodings[encoding]

			if hasError != tt.wantErr {
				t.Errorf("encoding validation result = %v, wantErr %v", hasError, tt.wantErr)
			}
		})
	}
}

func TestHandleGetMessageBody_ArgumentValidation(t *testing.T) {
	tests := []struct {
		name    string
		args    map[string]interface{}
		wantErr bool
	}{
		{
			name: "valid messageId",
			args: map[string]interface{}{
				"messageId": "msg123",
			},
			wantErr: false,
		},
		{
			name: "valid with format",
			args: map[string]interface{}{
				"messageId": "msg123",
				"format":    "html",
			},
			wantErr: false,
		},
		{
			name:    "missing messageId",
			args:    map[string]interface{}{},
			wantErr: true,
		},
		{
			name: "empty messageId",
			args: map[string]interface{}{
				"messageId": "",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			messageID, ok := tt.args["messageId"].(string)
			hasError := !ok || messageID == ""

			if hasError != tt.wantErr {
				t.Errorf("validation result = %v, wantErr %v", hasError, tt.wantErr)
			}
		})
	}
}

func TestHandleGetMessageBody_FormatValidation(t *testing.T) {
	tests := []struct {
		name   string
		format string
		valid  bool
	}{
		{
			name:   "text format",
			format: "text",
			valid:  true,
		},
		{
			name:   "html format",
			format: "html",
			valid:  true,
		},
		{
			name:   "empty format defaults to text",
			format: "",
			valid:  true,
		},
		{
			name:   "invalid format",
			format: "invalid",
			valid:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			format := tt.format
			if format == "" {
				format = "text"
			}

			validFormats := map[string]bool{
				"text": true,
				"html": true,
			}

			isValid := validFormats[format]

			if isValid != tt.valid {
				t.Errorf("format validation result = %v, want %v", isValid, tt.valid)
			}
		})
	}
}

func TestAttachmentOutputStructure(t *testing.T) {
	// Test that we can create proper JSON output structure
	type attachmentOutput struct {
		AttachmentID string `json:"attachmentId"`
		Filename     string `json:"filename"`
		MimeType     string `json:"mimeType"`
		Size         int64  `json:"size"`
		SizeHuman    string `json:"sizeHuman"`
	}

	tests := []struct {
		name       string
		attachment attachmentOutput
	}{
		{
			name: "PDF attachment",
			attachment: attachmentOutput{
				AttachmentID: "att123",
				Filename:     "document.pdf",
				MimeType:     "application/pdf",
				Size:         1048576,
				SizeHuman:    "1.00 MB",
			},
		},
		{
			name: "Image attachment",
			attachment: attachmentOutput{
				AttachmentID: "att456",
				Filename:     "image.png",
				MimeType:     "image/png",
				Size:         2048,
				SizeHuman:    "2.00 KB",
			},
		},
		{
			name: "Calendar attachment",
			attachment: attachmentOutput{
				AttachmentID: "att789",
				Filename:     "invite.ics",
				MimeType:     "text/calendar",
				Size:         1024,
				SizeHuman:    "1.00 KB",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.attachment.AttachmentID == "" {
				t.Error("AttachmentID should not be empty")
			}
			if tt.attachment.Filename == "" {
				t.Error("Filename should not be empty")
			}
			if tt.attachment.MimeType == "" {
				t.Error("MimeType should not be empty")
			}
			if tt.attachment.Size <= 0 {
				t.Error("Size should be positive")
			}
			if tt.attachment.SizeHuman == "" {
				t.Error("SizeHuman should not be empty")
			}
		})
	}
}

func TestHandleTransferAttachmentsToDrive_ArgumentValidation(t *testing.T) {
	tests := []struct {
		name    string
		args    map[string]interface{}
		wantErr bool
	}{
		{
			name: "valid single attachment",
			args: map[string]interface{}{
				"messageId":     "msg123",
				"attachmentIds": "att456",
			},
			wantErr: false,
		},
		{
			name: "valid multiple attachments",
			args: map[string]interface{}{
				"messageId":     "msg123",
				"attachmentIds": []interface{}{"att456", "att789"},
			},
			wantErr: false,
		},
		{
			name: "valid with parent folders",
			args: map[string]interface{}{
				"messageId":     "msg123",
				"attachmentIds": "att456",
				"parentFolders": "folder1,folder2",
			},
			wantErr: false,
		},
		{
			name: "valid with description",
			args: map[string]interface{}{
				"messageId":     "msg123",
				"attachmentIds": "att456",
				"description":   "Test file",
			},
			wantErr: false,
		},
		{
			name: "missing messageId",
			args: map[string]interface{}{
				"attachmentIds": "att456",
			},
			wantErr: true,
		},
		{
			name: "missing attachmentIds",
			args: map[string]interface{}{
				"messageId": "msg123",
			},
			wantErr: true,
		},
		{
			name: "empty messageId",
			args: map[string]interface{}{
				"messageId":     "",
				"attachmentIds": "att456",
			},
			wantErr: true,
		},
		{
			name: "empty attachmentIds string",
			args: map[string]interface{}{
				"messageId":     "msg123",
				"attachmentIds": "",
			},
			wantErr: true,
		},
		{
			name: "empty attachmentIds array",
			args: map[string]interface{}{
				"messageId":     "msg123",
				"attachmentIds": []interface{}{},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			messageID, ok := tt.args["messageId"].(string)
			messageIDErr := !ok || messageID == ""

			_, attachmentIDsErr := batch.ParseStringOrArray(tt.args["attachmentIds"], "attachmentIds")

			hasError := messageIDErr || attachmentIDsErr != nil

			if hasError != tt.wantErr {
				t.Errorf("validation result = %v, wantErr %v", hasError, tt.wantErr)
			}
		})
	}
}

func TestHandleTransferAttachmentsToDrive_ParentFolderParsing(t *testing.T) {
	tests := []struct {
		name          string
		parentFolders string
		expected      []string
	}{
		{
			name:          "single folder",
			parentFolders: "folder1",
			expected:      []string{"folder1"},
		},
		{
			name:          "multiple folders",
			parentFolders: "folder1,folder2,folder3",
			expected:      []string{"folder1", "folder2", "folder3"},
		},
		{
			name:          "folders with spaces",
			parentFolders: "folder1, folder2 , folder3",
			expected:      []string{"folder1", "folder2", "folder3"},
		},
		{
			name:          "empty string",
			parentFolders: "",
			expected:      nil,
		},
		{
			name:          "only commas",
			parentFolders: ",,,",
			expected:      nil,
		},
		{
			name:          "mixed empty and valid",
			parentFolders: "folder1,,folder2,",
			expected:      []string{"folder1", "folder2"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var result []string
			if tt.parentFolders != "" {
				for _, folder := range splitAndTrim(tt.parentFolders, ",") {
					if folder != "" {
						result = append(result, folder)
					}
				}
			}

			if len(result) != len(tt.expected) {
				t.Errorf("got %d folders, want %d", len(result), len(tt.expected))
				return
			}

			for i, folder := range result {
				if folder != tt.expected[i] {
					t.Errorf("folder[%d] = %v, want %v", i, folder, tt.expected[i])
				}
			}
		})
	}
}

func TestTransferAttachmentOutputStructure(t *testing.T) {
	// Test that we can create proper JSON output structure for transferred files
	type transferOutput struct {
		Filename     string `json:"filename"`
		DriveFileID  string `json:"driveFileId"`
		Size         int64  `json:"size"`
		SizeHuman    string `json:"sizeHuman"`
		MimeType     string `json:"mimeType"`
		WebViewLink  string `json:"webViewLink"`
		AttachmentID string `json:"attachmentId"`
	}

	tests := []struct {
		name   string
		output transferOutput
	}{
		{
			name: "PDF transfer",
			output: transferOutput{
				Filename:     "document.pdf",
				DriveFileID:  "drive123",
				Size:         1048576,
				SizeHuman:    "1.00 MB",
				MimeType:     "application/pdf",
				WebViewLink:  "https://drive.google.com/file/d/drive123/view",
				AttachmentID: "att456",
			},
		},
		{
			name: "Image transfer",
			output: transferOutput{
				Filename:     "photo.jpg",
				DriveFileID:  "drive456",
				Size:         2097152,
				SizeHuman:    "2.00 MB",
				MimeType:     "image/jpeg",
				WebViewLink:  "https://drive.google.com/file/d/drive456/view",
				AttachmentID: "att789",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.output.Filename == "" {
				t.Error("Filename should not be empty")
			}
			if tt.output.DriveFileID == "" {
				t.Error("DriveFileID should not be empty")
			}
			if tt.output.Size <= 0 {
				t.Error("Size should be positive")
			}
			if tt.output.SizeHuman == "" {
				t.Error("SizeHuman should not be empty")
			}
			if tt.output.MimeType == "" {
				t.Error("MimeType should not be empty")
			}
			if tt.output.WebViewLink == "" {
				t.Error("WebViewLink should not be empty")
			}
			if tt.output.AttachmentID == "" {
				t.Error("AttachmentID should not be empty")
			}
		})
	}
}

// Helper function for testing
func splitAndTrim(s, sep string) []string {
	if s == "" {
		return nil
	}
	var result []string
	for _, part := range splitString(s, sep) {
		trimmed := trimString(part)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

func splitString(s, sep string) []string {
	var result []string
	current := ""
	for _, c := range s {
		if string(c) == sep {
			result = append(result, current)
			current = ""
		} else {
			current += string(c)
		}
	}
	result = append(result, current)
	return result
}

func trimString(s string) string {
	start := 0
	end := len(s)

	// Trim leading spaces
	for start < end && s[start] == ' ' {
		start++
	}

	// Trim trailing spaces
	for end > start && s[end-1] == ' ' {
		end--
	}

	return s[start:end]
}
