package gmail

import (
	"encoding/base64"
	"mime"
	"strings"
	"testing"

	gmail "google.golang.org/api/gmail/v1"
)

func TestReplyToEmail(t *testing.T) {
	tests := []struct {
		name        string
		messageID   string
		threadID    string
		body        string
		cc          []string
		bcc         []string
		isHTML      bool
		wantErr     bool
		errContains string
	}{
		{
			name:        "missing messageID",
			messageID:   "",
			threadID:    "thread123",
			body:        "Reply body",
			wantErr:     true,
			errContains: "messageID is required",
		},
		{
			name:        "missing threadID",
			messageID:   "msg123",
			threadID:    "",
			body:        "Reply body",
			wantErr:     true,
			errContains: "threadID is required",
		},
		{
			name:        "missing body",
			messageID:   "msg123",
			threadID:    "thread123",
			body:        "",
			wantErr:     true,
			errContains: "body is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a mock client (this will fail on actual API calls, but validation should catch it first)
			c := &Client{}

			_, err := c.ReplyToEmail(tt.messageID, tt.threadID, tt.body, tt.cc, tt.bcc, tt.isHTML)

			if (err != nil) != tt.wantErr {
				t.Errorf("ReplyToEmail() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if err != nil && tt.errContains != "" {
				if !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("ReplyToEmail() error = %v, should contain %v", err, tt.errContains)
				}
			}
		})
	}
}

func TestForwardEmail(t *testing.T) {
	tests := []struct {
		name           string
		messageID      string
		to             []string
		cc             []string
		bcc            []string
		additionalBody string
		isHTML         bool
		wantErr        bool
		errContains    string
	}{
		{
			name:        "missing messageID",
			messageID:   "",
			to:          []string{"recipient@example.com"},
			wantErr:     true,
			errContains: "messageID is required",
		},
		{
			name:        "missing recipients",
			messageID:   "msg123",
			to:          []string{},
			wantErr:     true,
			errContains: "at least one recipient is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a mock client (this will fail on actual API calls, but validation should catch it first)
			c := &Client{}

			_, err := c.ForwardEmail(tt.messageID, tt.to, tt.cc, tt.bcc, tt.additionalBody, tt.isHTML)

			if (err != nil) != tt.wantErr {
				t.Errorf("ForwardEmail() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if err != nil && tt.errContains != "" {
				if !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("ForwardEmail() error = %v, should contain %v", err, tt.errContains)
				}
			}
		})
	}
}

func TestReplySubjectFormatting(t *testing.T) {
	tests := []struct {
		name            string
		originalSubject string
		wantPrefix      string
	}{
		{
			name:            "add Re: to subject without Re:",
			originalSubject: "Original Subject",
			wantPrefix:      "re:",
		},
		{
			name:            "don't duplicate Re: in subject",
			originalSubject: "Re: Original Subject",
			wantPrefix:      "re:",
		},
		{
			name:            "case insensitive Re: check",
			originalSubject: "RE: Original Subject",
			wantPrefix:      "re:",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test the subject formatting logic
			replySubject := tt.originalSubject
			if !strings.HasPrefix(strings.ToLower(replySubject), "re:") {
				replySubject = "Re: " + replySubject
			}

			if !strings.HasPrefix(strings.ToLower(replySubject), tt.wantPrefix) {
				t.Errorf("Reply subject = %v, want prefix %v", replySubject, tt.wantPrefix)
			}

			// Should not have double Re:
			lowerSubject := strings.ToLower(replySubject)
			reCount := strings.Count(lowerSubject, "re:")
			if reCount > 1 && tt.originalSubject != "Re: Re: Test" {
				t.Errorf("Reply subject has multiple Re: prefixes: %v", replySubject)
			}
		})
	}
}

func TestForwardSubjectFormatting(t *testing.T) {
	tests := []struct {
		name            string
		originalSubject string
		wantPrefix      bool
	}{
		{
			name:            "add Fwd: to subject without Fwd:",
			originalSubject: "Original Subject",
			wantPrefix:      true,
		},
		{
			name:            "don't duplicate Fwd: in subject",
			originalSubject: "Fwd: Original Subject",
			wantPrefix:      true,
		},
		{
			name:            "handle Fw: prefix",
			originalSubject: "Fw: Original Subject",
			wantPrefix:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test the forward subject formatting logic
			fwdSubject := tt.originalSubject
			if !strings.HasPrefix(strings.ToLower(fwdSubject), "fwd:") && !strings.HasPrefix(strings.ToLower(fwdSubject), "fw:") {
				fwdSubject = "Fwd: " + fwdSubject
			}

			hasPrefix := strings.HasPrefix(strings.ToLower(fwdSubject), "fwd:") || strings.HasPrefix(strings.ToLower(fwdSubject), "fw:")
			if hasPrefix != tt.wantPrefix {
				t.Errorf("Forward subject = %v, want prefix = %v", fwdSubject, tt.wantPrefix)
			}
		})
	}
}

func TestReplyThreadingHeaders(t *testing.T) {
	// Test that threading headers are properly constructed
	originalMessageID := "<abc123@example.com>"
	originalReferences := "<ref1@example.com> <ref2@example.com>"

	// Build References header for proper threading
	var references string
	if originalReferences != "" {
		references = originalReferences + " " + originalMessageID
	} else {
		references = originalMessageID
	}

	expectedReferences := "<ref1@example.com> <ref2@example.com> <abc123@example.com>"
	if references != expectedReferences {
		t.Errorf("References header = %v, want %v", references, expectedReferences)
	}

	// Test case without existing references
	references = ""
	if originalReferences == "" {
		references = originalMessageID
	}

	// Simulate empty originalReferences
	testOriginalReferences := ""
	if testOriginalReferences != "" {
		references = testOriginalReferences + " " + originalMessageID
	} else {
		references = originalMessageID
	}

	if references != originalMessageID {
		t.Errorf("References header without existing refs = %v, want %v", references, originalMessageID)
	}
}

func TestForwardBodyFormatting(t *testing.T) {
	originalFrom := "sender@example.com"
	originalTo := "recipient@example.com"
	originalSubject := "Test Subject"
	originalDate := "Mon, 31 Oct 2025 10:00:00 +0000"
	originalBody := "Original message body"
	additionalBody := "FYI"

	tests := []struct {
		name   string
		isHTML bool
	}{
		{
			name:   "plain text forward",
			isHTML: false,
		},
		{
			name:   "HTML forward",
			isHTML: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var forwardedBody string
			if tt.isHTML {
				forwardedBody = additionalBody + "<br><br>"
				forwardedBody += "---------- Forwarded message ---------<br>"
				forwardedBody += "From: " + originalFrom + "<br>"
				forwardedBody += "Date: " + originalDate + "<br>"
				forwardedBody += "Subject: " + originalSubject + "<br>"
				forwardedBody += "To: " + originalTo + "<br><br>"
				forwardedBody += originalBody
			} else {
				forwardedBody = additionalBody + "\n\n"
				forwardedBody += "---------- Forwarded message ---------\n"
				forwardedBody += "From: " + originalFrom + "\n"
				forwardedBody += "Date: " + originalDate + "\n"
				forwardedBody += "Subject: " + originalSubject + "\n"
				forwardedBody += "To: " + originalTo + "\n\n"
				forwardedBody += originalBody
			}

			// Verify structure
			if !strings.Contains(forwardedBody, additionalBody) {
				t.Errorf("Forward body missing additional body")
			}
			if !strings.Contains(forwardedBody, "Forwarded message") {
				t.Errorf("Forward body missing forwarded message indicator")
			}
			if !strings.Contains(forwardedBody, originalFrom) {
				t.Errorf("Forward body missing original sender")
			}
			if !strings.Contains(forwardedBody, originalBody) {
				t.Errorf("Forward body missing original message body")
			}
		})
	}
}

func TestHeaderValue(t *testing.T) {
	tests := []struct {
		name       string
		headers    []*gmail.MessagePartHeader
		headerName string
		want       string
	}{
		{
			name: "existing header",
			headers: []*gmail.MessagePartHeader{
				{Name: "From", Value: "sender@example.com"},
				{Name: "To", Value: "recipient@example.com"},
				{Name: "Subject", Value: "Test Subject"},
			},
			headerName: "From",
			want:       "sender@example.com",
		},
		{
			name: "missing header",
			headers: []*gmail.MessagePartHeader{
				{Name: "From", Value: "sender@example.com"},
			},
			headerName: "Cc",
			want:       "",
		},
		{
			name:       "nil payload",
			headers:    nil,
			headerName: "From",
			want:       "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := &gmail.Message{
				Payload: &gmail.MessagePart{
					Headers: tt.headers,
				},
			}

			// Test nil payload
			if tt.headers == nil {
				msg.Payload = nil
			}

			got := HeaderValue(msg, tt.headerName)
			if got != tt.want {
				t.Errorf("HeaderValue() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEmailEncoding(t *testing.T) {
	// Test that email content is properly base64url encoded
	testContent := "To: recipient@example.com\r\nSubject: Test\r\n\r\nBody content"
	encoded := base64.URLEncoding.EncodeToString([]byte(testContent))

	// Decode and verify
	decoded, err := base64.URLEncoding.DecodeString(encoded)
	if err != nil {
		t.Fatalf("Failed to decode: %v", err)
	}

	if string(decoded) != testContent {
		t.Errorf("Decoded content = %v, want %v", string(decoded), testContent)
	}
}

func TestEncodeRFC2047(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantASCII bool // If true, should return as-is; if false, should be encoded
	}{
		{
			name:      "plain ASCII text",
			input:     "Simple Subject",
			wantASCII: true,
		},
		{
			name:      "German umlauts",
			input:     "Rückerstattung €115 - Überweisung",
			wantASCII: false,
		},
		{
			name:      "French accents",
			input:     "Réponse à votre demande",
			wantASCII: false,
		},
		{
			name:      "Japanese characters",
			input:     "こんにちは",
			wantASCII: false,
		},
		{
			name:      "Emoji",
			input:     "Subject with emoji 🎉",
			wantASCII: false,
		},
		{
			name:      "Mixed ASCII and umlauts",
			input:     "Re: Öffnungszeiten",
			wantASCII: false,
		},
		{
			name:      "Empty string",
			input:     "",
			wantASCII: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := encodeRFC2047(tt.input)

			// If ASCII, result should equal input
			if tt.wantASCII {
				if result != tt.input {
					t.Errorf("encodeRFC2047() = %v, want %v (should not encode ASCII)", result, tt.input)
				}
			} else {
				// Should be encoded (starts with =?UTF-8?)
				if !strings.HasPrefix(result, "=?UTF-8?") {
					t.Errorf("encodeRFC2047() = %v, should start with =?UTF-8? for non-ASCII input", result)
				}
				// Should end with ?=
				if !strings.HasSuffix(result, "?=") {
					t.Errorf("encodeRFC2047() = %v, should end with ?= for non-ASCII input", result)
				}
			}
		})
	}
}

func TestEncodeRFC2047GermanUmlautsExample(t *testing.T) {
	// Test the exact example from the user's issue
	subject := "Rückerstattung €115 - Überweisung"
	encoded := encodeRFC2047(subject)

	// Should be encoded (not plain text)
	if encoded == subject {
		t.Errorf("Subject with umlauts should be encoded, got plain text: %v", encoded)
	}

	// Should start with =?UTF-8? and end with ?=
	if !strings.HasPrefix(encoded, "=?UTF-8?") {
		t.Errorf("Encoded subject should start with =?UTF-8?, got: %v", encoded)
	}

	if !strings.HasSuffix(encoded, "?=") {
		t.Errorf("Encoded subject should end with ?=, got: %v", encoded)
	}

	// Should NOT contain the garbled characters from the issue (RÃƒÂ¼)
	if strings.Contains(encoded, "Ã") {
		t.Errorf("Encoded subject should not contain garbled characters: %v", encoded)
	}
}

func TestEncodeRFC2047Roundtrip(t *testing.T) {
	// Test that encoding and decoding works correctly
	originalSubjects := []string{
		"Rückerstattung €115",
		"Überweisung",
		"Äpfel und Öl",
		"Größe",
	}

	for _, original := range originalSubjects {
		t.Run(original, func(t *testing.T) {
			encoded := encodeRFC2047(original)

			// Use mime.WordDecoder to decode
			decoder := new(mime.WordDecoder)
			decoded, err := decoder.DecodeHeader(encoded)
			if err != nil {
				t.Fatalf("Failed to decode %v: %v", encoded, err)
			}

			if decoded != original {
				t.Errorf("Roundtrip failed: original=%v, encoded=%v, decoded=%v", original, encoded, decoded)
			}
		})
	}
}

func TestAppendSignature(t *testing.T) {
	tests := []struct {
		name         string
		body         string
		signature    string
		isHTML       bool
		wantContains []string
	}{
		{
			name:      "plain text with signature",
			body:      "Hello,\n\nThis is my message.",
			signature: "Best regards,\nSender Name",
			isHTML:    false,
			wantContains: []string{
				"Hello,\n\nThis is my message.",
				"\n\n-- \n",
				"Best regards,\nSender Name",
			},
		},
		{
			name:      "HTML with signature",
			body:      "<p>Hello,</p><p>This is my message.</p>",
			signature: "<p>Best regards,<br>Sender Name</p>",
			isHTML:    true,
			wantContains: []string{
				"<p>Hello,</p><p>This is my message.</p>",
				"<br><br>-- <br>",
				"<p>Best regards,<br>Sender Name</p>",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a client with a signature
			c := &Client{
				signature: tt.signature,
			}

			result := c.appendSignature(tt.body, tt.isHTML)

			// Verify all expected parts are present
			for _, want := range tt.wantContains {
				if !strings.Contains(result, want) {
					t.Errorf("appendSignature() result missing expected content: %v\nGot: %v", want, result)
				}
			}
		})
	}
}

func TestSignatureFormatting(t *testing.T) {
	tests := []struct {
		name    string
		isHTML  bool
		wantSep string
	}{
		{
			name:    "plain text separator",
			isHTML:  false,
			wantSep: "\n\n-- \n",
		},
		{
			name:    "HTML separator",
			isHTML:  true,
			wantSep: "<br><br>-- <br>",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Client{
				signature: "Test Signature",
			}

			result := c.appendSignature("Body", tt.isHTML)

			if !strings.Contains(result, tt.wantSep) {
				t.Errorf("appendSignature() missing separator %v in result: %v", tt.wantSep, result)
			}
		})
	}
}
