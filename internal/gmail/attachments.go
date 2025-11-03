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

// GetThreadMessageBodies extracts bodies from all messages in a thread
func (c *Client) GetThreadMessageBodies(threadID string, format string) (string, error) {
	if format == "" {
		format = "text"
	}

	thread, err := c.GetThread(threadID)
	if err != nil {
		return "", err
	}

	if len(thread.Messages) == 0 {
		return "", fmt.Errorf("thread %s contains no messages", threadID)
	}

	var allBodies strings.Builder
	for i, msg := range thread.Messages {
		// Extract body directly from the message we already have
		body, err := c.extractBodyFromMessage(msg, format)
		if err != nil {
			// If we can't get the body for this message, skip it with a note
			allBodies.WriteString(fmt.Sprintf("\n[Message %d/%d: Error extracting body: %v]\n", i+1, len(thread.Messages), err))
			continue
		}

		// Add message separator if not the first message
		if i > 0 {
			allBodies.WriteString("\n\n" + strings.Repeat("-", 80) + "\n\n")
		}

		allBodies.WriteString(fmt.Sprintf("Message %d/%d (ID: %s):\n\n", i+1, len(thread.Messages), msg.Id))
		allBodies.WriteString(body)
	}

	return allBodies.String(), nil
}

// extractBodyFromMessage extracts body from a message object we already have.
// When format is "text" and no text body is found, it automatically falls back to HTML.
// This eliminates the need for manual retries when dealing with HTML-only emails.
func (c *Client) extractBodyFromMessage(msg *gmail.Message, format string) (string, error) {
	// Default to text format if empty
	if format == "" {
		format = "text"
	}

	// Try to extract with the requested format
	body, err := c.extractBodyFromMessageInternal(msg, format)

	// Auto-fallback to HTML if text not available
	// Only fallback when format is "text" to prevent infinite loops
	if err != nil && format == "text" && strings.Contains(err.Error(), "no text body found") {
		return c.extractBodyFromMessageInternal(msg, "html")
	}

	return body, err
}

// extractBodyFromMessageInternal is the internal implementation that extracts a specific format
func (c *Client) extractBodyFromMessageInternal(msg *gmail.Message, format string) (string, error) {
	var targetMimeType string
	switch format {
	case "text":
		targetMimeType = "text/plain"
	case "html":
		targetMimeType = "text/html"
	default:
		return "", fmt.Errorf("invalid format %s, must be 'text' or 'html'", format)
	}

	var body string

	// Find the appropriate body part based on format
	if msg.Payload != nil {
		if msg.Payload.MimeType == targetMimeType && msg.Payload.Body != nil && msg.Payload.Body.Data != "" {
			body = msg.Payload.Body.Data
		} else {
			// Walk through parts to find the body
			walkParts(msg.Payload, msg.Id, func(part *gmail.MessagePart) {
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

// GetMessageBody extracts text/HTML body from a message or thread.
// It accepts both Message IDs and Thread IDs for convenience.
// When format is "text" and no text body is found, it automatically falls back to HTML.
func (c *Client) GetMessageBody(messageID string, format string) (string, error) {
	if format == "" {
		format = "text"
	}

	// First, try to get it as a message (most common case)
	msg, err := c.GetMessage(messageID)
	if err == nil {
		// Successfully got it as a message, extract the body
		return c.extractBodyFromMessage(msg, format)
	}

	// If it failed, check if the error suggests it might be a thread ID
	// Gmail API returns 404 for messages that don't exist, which includes thread IDs
	if strings.Contains(err.Error(), "404") || strings.Contains(err.Error(), "not found") {
		// Try as a thread ID
		threadBody, threadErr := c.GetThreadMessageBodies(messageID, format)
		if threadErr == nil {
			// Successfully got it as a thread
			return threadBody, nil
		}
		// If both failed, return the original message error
		return "", fmt.Errorf("failed to get body (tried as message ID and thread ID): %w", err)
	}

	// For other errors (not 404), just return the original error
	return "", err
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
