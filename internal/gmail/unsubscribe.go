package gmail

import (
	"fmt"
	"io"
	"net/http"
	"strings"
)

// UnsubscribeInfo contains information about how to unsubscribe from a sender
type UnsubscribeInfo struct {
	MessageID      string
	HasUnsubscribe bool
	Methods        []UnsubscribeMethod
}

// UnsubscribeMethod represents a single unsubscribe method
type UnsubscribeMethod struct {
	Type string // "mailto" or "http"
	URL  string
}

// GetUnsubscribeInfo extracts List-Unsubscribe information from a message
func (c *Client) GetUnsubscribeInfo(messageID string) (*UnsubscribeInfo, error) {
	msg, err := c.GetMessage(messageID)
	if err != nil {
		return nil, fmt.Errorf("failed to get message: %w", err)
	}

	info := &UnsubscribeInfo{
		MessageID:      messageID,
		HasUnsubscribe: false,
		Methods:        []UnsubscribeMethod{},
	}

	// Extract List-Unsubscribe header
	listUnsubscribe := HeaderValue(msg, "List-Unsubscribe")
	if listUnsubscribe == "" {
		return info, nil
	}

	info.HasUnsubscribe = true

	// Parse the List-Unsubscribe header
	// Format: <mailto:unsub@example.com>, <http://example.com/unsub>
	// Or: <http://example.com/unsub>
	// Or: <mailto:unsub@example.com?subject=unsubscribe>
	methods := parseListUnsubscribe(listUnsubscribe)
	info.Methods = methods

	return info, nil
}

// parseListUnsubscribe parses the List-Unsubscribe header value
func parseListUnsubscribe(header string) []UnsubscribeMethod {
	var methods []UnsubscribeMethod

	// Split by angle brackets
	parts := strings.Split(header, "<")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		// Find closing bracket
		endIdx := strings.Index(part, ">")
		if endIdx == -1 {
			continue
		}

		url := part[:endIdx]
		url = strings.TrimSpace(url)

		if strings.HasPrefix(url, "mailto:") {
			methods = append(methods, UnsubscribeMethod{
				Type: "mailto",
				URL:  url,
			})
		} else if strings.HasPrefix(url, "http://") || strings.HasPrefix(url, "https://") {
			methods = append(methods, UnsubscribeMethod{
				Type: "http",
				URL:  url,
			})
		}
	}

	return methods
}

// UnsubscribeViaHTTP performs an HTTP GET request to the unsubscribe URL
// This follows the RFC 2369 List-Unsubscribe specification
func (c *Client) UnsubscribeViaHTTP(url string) error {
	if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
		return fmt.Errorf("invalid HTTP URL: %s", url)
	}

	// Create HTTP client
	client := &http.Client{}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Some unsubscribe links require a user agent
	req.Header.Set("User-Agent", "inboxfewer/1.0")

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send unsubscribe request: %w", err)
	}
	defer resp.Body.Close()

	// Read response body (but discard it, we just want the status)
	_, _ = io.ReadAll(resp.Body)

	if resp.StatusCode < 200 || resp.StatusCode >= 400 {
		return fmt.Errorf("unsubscribe request failed with status %d", resp.StatusCode)
	}

	return nil
}
