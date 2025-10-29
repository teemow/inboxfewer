package gmail

import (
	"encoding/base64"
	"fmt"
	"strings"

	gmail "google.golang.org/api/gmail/v1"
)

const (
	// MaxAttachmentSize defines the maximum attachment size in bytes (25MB)
	MaxAttachmentSize = 25 * 1024 * 1024
)

// AttachmentInfo represents an attachment's metadata
type AttachmentInfo struct {
	MessageID    string
	PartID       string
	AttachmentID string
	Filename     string
	MimeType     string
	Size         int64
}

// GetMessage retrieves a full Gmail message
func (c *Client) GetMessage(messageID string) (*gmail.Message, error) {
	msg, err := c.svc.Messages.Get("me", messageID).Format("full").Do()
	if err != nil {
		return nil, fmt.Errorf("failed to get message %s: %w", messageID, err)
	}
	return msg, nil
}

// ListAttachments extracts all attachments from a message
func (c *Client) ListAttachments(messageID string) ([]*AttachmentInfo, error) {
	msg, err := c.GetMessage(messageID)
	if err != nil {
		return nil, err
	}

	var attachments []*AttachmentInfo
	walkParts(msg.Payload, messageID, func(part *gmail.MessagePart) {
		if part.Filename != "" && part.Body != nil && part.Body.AttachmentId != "" {
			attachments = append(attachments, &AttachmentInfo{
				MessageID:    messageID,
				PartID:       part.PartId,
				AttachmentID: part.Body.AttachmentId,
				Filename:     part.Filename,
				MimeType:     part.MimeType,
				Size:         part.Body.Size,
			})
		}
	})

	return attachments, nil
}

// GetAttachment retrieves the content of an attachment (returns []byte)
func (c *Client) GetAttachment(messageID, attachmentID string) ([]byte, error) {
	if messageID == "" {
		return nil, fmt.Errorf("messageID is required")
	}
	if attachmentID == "" {
		return nil, fmt.Errorf("attachmentID is required")
	}

	attachment, err := c.svc.Messages.Attachments.Get("me", messageID, attachmentID).Do()
	if err != nil {
		return nil, fmt.Errorf("failed to get attachment %s: %w", attachmentID, err)
	}

	// Check size limit
	if attachment.Size > MaxAttachmentSize {
		return nil, fmt.Errorf("attachment size %d exceeds maximum size %d", attachment.Size, MaxAttachmentSize)
	}

	// Decode base64url-encoded data (Gmail API uses RFC 4648 base64url encoding)
	data, err := base64.URLEncoding.DecodeString(attachment.Data)
	if err != nil {
		// Try with standard base64 if URLEncoding fails
		data, err = base64.StdEncoding.DecodeString(attachment.Data)
		if err != nil {
			return nil, fmt.Errorf("failed to decode attachment data: %w", err)
		}
	}

	return data, nil
}

// GetAttachmentAsString retrieves attachment content as string (for text files)
func (c *Client) GetAttachmentAsString(messageID, attachmentID string) (string, error) {
	data, err := c.GetAttachment(messageID, attachmentID)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// GetMessageBody extracts text/HTML body from a message
func (c *Client) GetMessageBody(messageID string, format string) (string, error) {
	if format == "" {
		format = "text"
	}

	msg, err := c.GetMessage(messageID)
	if err != nil {
		return "", err
	}

	// Find the appropriate body part based on format
	var body string
	var targetMimeType string

	switch format {
	case "text":
		targetMimeType = "text/plain"
	case "html":
		targetMimeType = "text/html"
	default:
		return "", fmt.Errorf("invalid format %s, must be 'text' or 'html'", format)
	}

	// First, try to find the body in the main payload
	if msg.Payload != nil {
		if msg.Payload.MimeType == targetMimeType && msg.Payload.Body != nil && msg.Payload.Body.Data != "" {
			body = msg.Payload.Body.Data
		} else {
			// Walk through parts to find the body
			walkParts(msg.Payload, messageID, func(part *gmail.MessagePart) {
				if body == "" && part.MimeType == targetMimeType && part.Body != nil && part.Body.Data != "" {
					body = part.Body.Data
				}
			})
		}
	}

	if body == "" {
		return "", fmt.Errorf("no %s body found in message", format)
	}

	// Decode base64url-encoded body data
	decoded, err := base64.URLEncoding.DecodeString(body)
	if err != nil {
		// Try with standard base64 if URLEncoding fails
		decoded, err = base64.StdEncoding.DecodeString(body)
		if err != nil {
			return "", fmt.Errorf("failed to decode message body: %w", err)
		}
	}

	return string(decoded), nil
}

// walkParts recursively walks through message parts
func walkParts(part *gmail.MessagePart, messageID string, fn func(*gmail.MessagePart)) {
	if part == nil {
		return
	}

	fn(part)

	for _, subpart := range part.Parts {
		walkParts(subpart, messageID, fn)
	}
}

// SanitizeFilename sanitizes a filename to prevent path traversal attacks
func SanitizeFilename(filename string) string {
	// Remove path separators and other potentially dangerous characters
	filename = strings.ReplaceAll(filename, "/", "_")
	filename = strings.ReplaceAll(filename, "\\", "_")
	filename = strings.ReplaceAll(filename, "..", "_")
	return filename
}

// ValidateMimeType checks if a MIME type is in the allowed list
func ValidateMimeType(mimeType string, allowedTypes []string) bool {
	if len(allowedTypes) == 0 {
		return true // No restrictions if list is empty
	}

	for _, allowed := range allowedTypes {
		if mimeType == allowed {
			return true
		}
	}
	return false
}
