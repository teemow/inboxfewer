package gmail

import (
	"bufio"
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"log"
	"os"
	"strings"

	gmail "google.golang.org/api/gmail/v1"
	"google.golang.org/api/option"
	"google.golang.org/api/people/v1"

	"github.com/teemow/inboxfewer/internal/google"
)

// Client wraps the Gmail Users service and People service
type Client struct {
	svc       *gmail.UsersService
	peopleSvc *people.Service
}

// HasToken checks if a valid OAuth token exists
func HasToken() bool {
	return google.HasToken()
}

// GetAuthURL returns the OAuth URL for user authorization
func GetAuthURL() string {
	return google.GetAuthURL()
}

// SaveToken exchanges an authorization code for tokens and saves them
func SaveToken(ctx context.Context, authCode string) error {
	return google.SaveToken(ctx, authCode)
}

// NewClient creates a new Gmail client with OAuth2 authentication
// For CLI usage, it will prompt for auth code via stdin if no token exists
// For MCP usage, it will return an error if no token exists
func NewClient(ctx context.Context) (*Client, error) {
	// Try to get existing token
	client, err := google.GetHTTPClient(ctx)
	if err != nil {
		// Check if we're in a terminal (CLI mode)
		if isTerminal() {
			authURL := google.GetAuthURL()
			log.Printf("Go to %v", authURL)
			io.WriteString(os.Stdout, "Enter code> ")

			bs := bufio.NewScanner(os.Stdin)
			if !bs.Scan() {
				return nil, io.EOF
			}
			code := bs.Text()
			if err := google.SaveToken(ctx, code); err != nil {
				return nil, err
			}
			// Try again with the new token
			client, err = google.GetHTTPClient(ctx)
			if err != nil {
				return nil, err
			}
		} else {
			// MCP mode - return error with instructions
			return nil, fmt.Errorf("no valid Google OAuth token found. Use gmail_get_auth_url and gmail_save_auth_code tools to authenticate")
		}
	}

	svc, err := gmail.New(client)
	if err != nil {
		return nil, err
	}

	peopleSvc, err := people.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return nil, fmt.Errorf("failed to create People service: %w", err)
	}

	return &Client{
		svc:       svc.Users,
		peopleSvc: peopleSvc,
	}, nil
}

// ArchiveThread archives a thread by removing the INBOX label
func (c *Client) ArchiveThread(tid string) error {
	_, err := c.svc.Threads.Modify("me", tid, &gmail.ModifyThreadRequest{
		RemoveLabelIds: []string{"INBOX"},
	}).Do()
	return err
}

// ForeachThread iterates over all threads matching the query
func (c *Client) ForeachThread(q string, fn func(*gmail.Thread) error) error {
	pageToken := ""
	for {
		req := c.svc.Threads.List("me").Q(q)
		if pageToken != "" {
			req.PageToken(pageToken)
		}
		res, err := req.Do()
		if err != nil {
			return err
		}
		for _, t := range res.Threads {
			if err := fn(t); err != nil {
				return err
			}
		}
		if res.NextPageToken == "" {
			return nil
		}
		pageToken = res.NextPageToken
	}
}

// PopulateThread populates t with its full data. t.Id must be set initially.
func (c *Client) PopulateThread(t *gmail.Thread) error {
	req := c.svc.Threads.Get("me", t.Id).Format("full")
	tfull, err := req.Do()
	if err != nil {
		return err
	}
	*t = *tfull
	return nil
}

// ListThreads lists threads matching the query with pagination
func (c *Client) ListThreads(q string, maxResults int64) ([]*gmail.Thread, error) {
	req := c.svc.Threads.List("me").Q(q).MaxResults(maxResults)
	res, err := req.Do()
	if err != nil {
		return nil, err
	}
	return res.Threads, nil
}

// isTerminal checks if stdin is connected to a terminal (CLI mode)
func isTerminal() bool {
	fileInfo, _ := os.Stdin.Stat()
	return (fileInfo.Mode() & os.ModeCharDevice) != 0
}

// Contact represents a simplified contact entry
type Contact struct {
	ResourceName string
	DisplayName  string
	EmailAddress string
	PhoneNumber  string
}

// SearchContacts searches for contacts in Google Contacts using the query
func (c *Client) SearchContacts(query string, pageSize int) ([]*Contact, error) {
	if pageSize <= 0 {
		pageSize = 10
	}

	req := c.peopleSvc.People.SearchContacts().
		Query(query).
		ReadMask("names,emailAddresses,phoneNumbers").
		PageSize(int64(pageSize))

	resp, err := req.Do()
	if err != nil {
		return nil, fmt.Errorf("failed to search contacts: %w", err)
	}

	var contacts []*Contact
	for _, result := range resp.Results {
		person := result.Person
		if person == nil {
			continue
		}

		contact := &Contact{
			ResourceName: person.ResourceName,
		}

		// Get display name
		if len(person.Names) > 0 {
			contact.DisplayName = person.Names[0].DisplayName
		}

		// Get primary email
		if len(person.EmailAddresses) > 0 {
			contact.EmailAddress = person.EmailAddresses[0].Value
		}

		// Get primary phone number
		if len(person.PhoneNumbers) > 0 {
			contact.PhoneNumber = person.PhoneNumbers[0].Value
		}

		contacts = append(contacts, contact)
	}

	return contacts, nil
}

// EmailMessage represents an email to be sent
type EmailMessage struct {
	To      []string
	Cc      []string
	Bcc     []string
	Subject string
	Body    string
	IsHTML  bool
}

// SendEmail sends an email through Gmail API
func (c *Client) SendEmail(msg *EmailMessage) (string, error) {
	if len(msg.To) == 0 {
		return "", fmt.Errorf("at least one recipient is required")
	}
	if msg.Subject == "" {
		return "", fmt.Errorf("subject is required")
	}
	if msg.Body == "" {
		return "", fmt.Errorf("body is required")
	}

	// Build the email message in RFC 2822 format
	var emailBuilder strings.Builder

	// Add To header
	emailBuilder.WriteString("To: ")
	emailBuilder.WriteString(strings.Join(msg.To, ", "))
	emailBuilder.WriteString("\r\n")

	// Add Cc header if present
	if len(msg.Cc) > 0 {
		emailBuilder.WriteString("Cc: ")
		emailBuilder.WriteString(strings.Join(msg.Cc, ", "))
		emailBuilder.WriteString("\r\n")
	}

	// Add Bcc header if present
	if len(msg.Bcc) > 0 {
		emailBuilder.WriteString("Bcc: ")
		emailBuilder.WriteString(strings.Join(msg.Bcc, ", "))
		emailBuilder.WriteString("\r\n")
	}

	// Add Subject
	emailBuilder.WriteString("Subject: ")
	emailBuilder.WriteString(msg.Subject)
	emailBuilder.WriteString("\r\n")

	// Add Content-Type
	if msg.IsHTML {
		emailBuilder.WriteString("Content-Type: text/html; charset=\"UTF-8\"\r\n")
	} else {
		emailBuilder.WriteString("Content-Type: text/plain; charset=\"UTF-8\"\r\n")
	}
	emailBuilder.WriteString("MIME-Version: 1.0\r\n")
	emailBuilder.WriteString("\r\n")

	// Add body
	emailBuilder.WriteString(msg.Body)

	// Encode the message in base64url format
	rawMessage := base64.URLEncoding.EncodeToString([]byte(emailBuilder.String()))

	// Send the message
	gmailMsg := &gmail.Message{
		Raw: rawMessage,
	}

	sent, err := c.svc.Messages.Send("me", gmailMsg).Do()
	if err != nil {
		return "", fmt.Errorf("failed to send email: %w", err)
	}

	return sent.Id, nil
}
