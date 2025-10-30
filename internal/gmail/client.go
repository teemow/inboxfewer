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
	account   string // The account this client is associated with
}

// Account returns the account name this client is associated with
func (c *Client) Account() string {
	return c.account
}

// HasTokenForAccount checks if a valid OAuth token exists for the specified account
func HasTokenForAccount(account string) bool {
	return google.HasTokenForAccount(account)
}

// HasToken checks if a valid OAuth token exists for the default account
func HasToken() bool {
	return google.HasToken()
}

// GetAuthURLForAccount returns the OAuth URL for user authorization for a specific account
func GetAuthURLForAccount(account string) string {
	return google.GetAuthURLForAccount(account)
}

// GetAuthURL returns the OAuth URL for user authorization for the default account
func GetAuthURL() string {
	return google.GetAuthURL()
}

// SaveTokenForAccount exchanges an authorization code for tokens and saves them for a specific account
func SaveTokenForAccount(ctx context.Context, account string, authCode string) error {
	return google.SaveTokenForAccount(ctx, account, authCode)
}

// SaveToken exchanges an authorization code for tokens and saves them for the default account
func SaveToken(ctx context.Context, authCode string) error {
	return google.SaveToken(ctx, authCode)
}

// NewClientForAccount creates a new Gmail client with OAuth2 authentication for a specific account
// For CLI usage, it will prompt for auth code via stdin if no token exists
// For MCP usage, it will return an error if no token exists
func NewClientForAccount(ctx context.Context, account string) (*Client, error) {
	// Try to get existing token
	client, err := google.GetHTTPClientForAccount(ctx, account)
	if err != nil {
		// Check if we're in a terminal (CLI mode)
		if isTerminal() {
			authURL := google.GetAuthURLForAccount(account)
			log.Printf("Go to %v", authURL)
			log.Printf("Authorizing for account: %s", account)
			io.WriteString(os.Stdout, "Enter code> ")

			bs := bufio.NewScanner(os.Stdin)
			if !bs.Scan() {
				return nil, io.EOF
			}
			code := bs.Text()
			if err := google.SaveTokenForAccount(ctx, account, code); err != nil {
				return nil, err
			}
			// Try again with the new token
			client, err = google.GetHTTPClientForAccount(ctx, account)
			if err != nil {
				return nil, err
			}
		} else {
			// MCP mode - return error with instructions
			return nil, fmt.Errorf("no valid Google OAuth token found for account %s. Use google_get_auth_url and google_save_auth_code tools to authenticate", account)
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
		account:   account,
	}, nil
}

// NewClient creates a new Gmail client with OAuth2 authentication for the default account
// For CLI usage, it will prompt for auth code via stdin if no token exists
// For MCP usage, it will return an error if no token exists
func NewClient(ctx context.Context) (*Client, error) {
	return NewClientForAccount(ctx, "default")
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

// SearchContacts searches for contacts across all sources (personal, directory, and other contacts)
// using the query string to filter results
func (c *Client) SearchContacts(query string, pageSize int) ([]*Contact, error) {
	if pageSize <= 0 {
		pageSize = 10
	}

	var allContacts []*Contact
	seenEmails := make(map[string]bool) // Track seen emails to avoid duplicates
	queryLower := strings.ToLower(query)

	// We want to collect enough candidates from all sources before limiting
	// Set a higher target to ensure we get good coverage from each source
	targetResults := pageSize * 10 // Collect 10x the requested results before limiting

	// 1. Search personal contacts using SearchContacts
	req := c.peopleSvc.People.SearchContacts().
		Query(query).
		ReadMask("names,emailAddresses,phoneNumbers").
		PageSize(int64(pageSize * 2)) // Request more to get better coverage

	resp, err := req.Do()
	if err == nil { // Don't fail if one source fails
		for _, result := range resp.Results {
			if contact := extractContact(result.Person); contact != nil {
				if contact.EmailAddress != "" && !seenEmails[contact.EmailAddress] {
					seenEmails[contact.EmailAddress] = true
					allContacts = append(allContacts, contact)
				}
			}
		}
	}

	// 2. Search other contacts (people user has interacted with)
	// Need to paginate through all other contacts since API doesn't support search query
	// Keep searching until we have enough results OR we've exhausted the pages
	pageToken := ""
	maxPagesToFetch := 10 // Fetch up to 1000 contacts total
	pagesSearched := 0
	otherContactsFound := 0

	for pagesSearched < maxPagesToFetch {
		otherReq := c.peopleSvc.OtherContacts.List().
			ReadMask("names,emailAddresses,phoneNumbers").
			PageSize(100) // Fetch 100 at a time for efficiency

		if pageToken != "" {
			otherReq = otherReq.PageToken(pageToken)
		}

		otherResp, err := otherReq.Do()
		if err != nil {
			break // Stop if we hit an error
		}

		for _, person := range otherResp.OtherContacts {
			if contact := extractContact(person); contact != nil {
				// Filter by query
				if matchesQuery(contact, queryLower) {
					if contact.EmailAddress != "" && !seenEmails[contact.EmailAddress] {
						seenEmails[contact.EmailAddress] = true
						allContacts = append(allContacts, contact)
						otherContactsFound++
					}
				}
			}
		}

		// Check if there are more pages
		pageToken = otherResp.NextPageToken
		if pageToken == "" {
			break // No more pages
		}

		// Stop if we've collected enough total results
		if len(allContacts) >= targetResults {
			break
		}

		pagesSearched++
	}

	// 3. Try to search directory contacts (for Workspace accounts)
	// This will only work for Workspace accounts, will fail gracefully for consumer accounts
	dirReq := c.peopleSvc.People.SearchDirectoryPeople().
		Query(query).
		ReadMask("names,emailAddresses,phoneNumbers").
		PageSize(int64(pageSize * 2)) // Request more to get better coverage

	dirResp, err := dirReq.Do()
	if err == nil { // Will fail for non-Workspace accounts, that's OK
		for _, person := range dirResp.People {
			if contact := extractContact(person); contact != nil {
				if contact.EmailAddress != "" && !seenEmails[contact.EmailAddress] {
					seenEmails[contact.EmailAddress] = true
					allContacts = append(allContacts, contact)
				}
			}
		}
	}

	// Limit results to requested page size
	if len(allContacts) > pageSize {
		allContacts = allContacts[:pageSize]
	}

	return allContacts, nil
}

// extractContact extracts contact information from a Person object
func extractContact(person *people.Person) *Contact {
	if person == nil {
		return nil
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

	// Skip contacts without any useful information
	if contact.DisplayName == "" && contact.EmailAddress == "" && contact.PhoneNumber == "" {
		return nil
	}

	return contact
}

// matchesQuery checks if a contact matches the search query
func matchesQuery(contact *Contact, queryLower string) bool {
	if queryLower == "" {
		return true
	}

	// Check if query matches name, email, or phone
	if strings.Contains(strings.ToLower(contact.DisplayName), queryLower) {
		return true
	}
	if strings.Contains(strings.ToLower(contact.EmailAddress), queryLower) {
		return true
	}
	if strings.Contains(contact.PhoneNumber, queryLower) {
		return true
	}

	return false
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
