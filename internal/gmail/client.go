package gmail

import (
	"context"
	"encoding/base64"
	"fmt"
	"mime"
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
	signature string // Cached signature for this account
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

// NewClientForAccount creates a new Gmail client with OAuth2 authentication for a specific account
// The OAuth token must be provided by the MCP client through the OAuth middleware
func NewClientForAccount(ctx context.Context, account string) (*Client, error) {
	// Get HTTP client with OAuth token
	client, err := google.GetHTTPClientForAccount(ctx, account)
	if err != nil {
		return nil, fmt.Errorf("no valid Google OAuth token found for account %s: %w. Please authenticate with Google through your MCP client", account, err)
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

// UnarchiveThread moves a thread back to inbox by adding the INBOX label
func (c *Client) UnarchiveThread(tid string) error {
	_, err := c.svc.Threads.Modify("me", tid, &gmail.ModifyThreadRequest{
		AddLabelIds: []string{"INBOX"},
	}).Do()
	return err
}

// MarkThreadAsSpam marks a thread as spam by adding the SPAM label and removing the INBOX label
func (c *Client) MarkThreadAsSpam(tid string) error {
	_, err := c.svc.Threads.Modify("me", tid, &gmail.ModifyThreadRequest{
		AddLabelIds:    []string{"SPAM"},
		RemoveLabelIds: []string{"INBOX"},
	}).Do()
	return err
}

// UnmarkThreadAsSpam removes the spam label from a thread, moving it back to inbox
func (c *Client) UnmarkThreadAsSpam(tid string) error {
	_, err := c.svc.Threads.Modify("me", tid, &gmail.ModifyThreadRequest{
		AddLabelIds:    []string{"INBOX"},
		RemoveLabelIds: []string{"SPAM"},
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

// GetThread retrieves a full Gmail thread with all its messages
func (c *Client) GetThread(threadID string) (*gmail.Thread, error) {
	thread, err := c.svc.Threads.Get("me", threadID).Format("full").Do()
	if err != nil {
		return nil, fmt.Errorf("failed to get thread %s: %w", threadID, err)
	}
	return thread, nil
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
// It will fetch up to maxResults threads, making multiple API calls if necessary
func (c *Client) ListThreads(q string, maxResults int64) ([]*gmail.Thread, error) {
	var allThreads []*gmail.Thread
	pageToken := ""

	for {
		// Request the remaining number of threads needed
		remaining := maxResults - int64(len(allThreads))
		if remaining <= 0 {
			break
		}

		// Gmail API has a max page size, typically 100
		pageSize := remaining
		if pageSize > 100 {
			pageSize = 100
		}

		req := c.svc.Threads.List("me").Q(q).MaxResults(pageSize)
		if pageToken != "" {
			req = req.PageToken(pageToken)
		}

		res, err := req.Do()
		if err != nil {
			return nil, err
		}

		allThreads = append(allThreads, res.Threads...)

		// If there's no next page or we have enough results, stop
		if res.NextPageToken == "" || int64(len(allThreads)) >= maxResults {
			break
		}

		pageToken = res.NextPageToken
	}

	// Trim to exact maxResults if we got more
	if int64(len(allThreads)) > maxResults {
		allThreads = allThreads[:maxResults]
	}

	return allThreads, nil
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

// encodeRFC2047 encodes a string for use in email headers according to RFC 2047
// This is necessary for non-ASCII characters (like German umlauts) in subjects
func encodeRFC2047(s string) string {
	// Check if the string contains only ASCII characters
	needsEncoding := false
	for _, r := range s {
		if r > 127 {
			needsEncoding = true
			break
		}
	}

	// If it's all ASCII, return as-is
	if !needsEncoding {
		return s
	}

	// Use Go's mime package which implements RFC 2047 encoding
	return mime.BEncoding.Encode("UTF-8", s)
}

// GetSignature fetches the user's Gmail signature (primary send-as address)
// The signature is cached after the first fetch
func (c *Client) GetSignature() (string, error) {
	// Return cached signature if available
	if c.signature != "" {
		return c.signature, nil
	}

	// Fetch send-as settings to get the signature
	sendAs, err := c.svc.Settings.SendAs.Get("me", "me").Do()
	if err != nil {
		// If we can't fetch the signature, return empty string (not an error)
		// This allows emails to be sent even if signature fetching fails
		return "", nil
	}

	// Cache the signature
	if sendAs.Signature != "" {
		c.signature = sendAs.Signature
	}

	return c.signature, nil
}

// appendSignature adds the user's signature to the email body
func (c *Client) appendSignature(body string, isHTML bool) string {
	signature, err := c.GetSignature()
	if err != nil || signature == "" {
		// No signature or error fetching it, return body as-is
		return body
	}

	// Append signature with appropriate formatting
	if isHTML {
		// Add signature with line breaks for HTML
		return body + "<br><br>-- <br>" + signature
	}

	// Add signature with line breaks for plain text
	return body + "\n\n-- \n" + signature
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

	// Add Subject (encode for non-ASCII characters like umlauts)
	emailBuilder.WriteString("Subject: ")
	emailBuilder.WriteString(encodeRFC2047(msg.Subject))
	emailBuilder.WriteString("\r\n")

	// Add Content-Type
	if msg.IsHTML {
		emailBuilder.WriteString("Content-Type: text/html; charset=\"UTF-8\"\r\n")
	} else {
		emailBuilder.WriteString("Content-Type: text/plain; charset=\"UTF-8\"\r\n")
	}
	emailBuilder.WriteString("MIME-Version: 1.0\r\n")
	emailBuilder.WriteString("\r\n")

	// Add body with signature
	bodyWithSignature := c.appendSignature(msg.Body, msg.IsHTML)
	emailBuilder.WriteString(bodyWithSignature)

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

// ReplyToEmail sends a reply to an existing email message
func (c *Client) ReplyToEmail(messageID, threadID, body string, cc, bcc []string, isHTML bool) (string, error) {
	if messageID == "" {
		return "", fmt.Errorf("messageID is required")
	}
	if threadID == "" {
		return "", fmt.Errorf("threadID is required")
	}
	if body == "" {
		return "", fmt.Errorf("body is required")
	}

	// Get the original message to extract headers
	msg, err := c.GetMessage(messageID)
	if err != nil {
		return "", fmt.Errorf("failed to get original message: %w", err)
	}

	// Extract necessary headers
	originalFrom := HeaderValue(msg, "From")
	originalSubject := HeaderValue(msg, "Subject")
	originalMessageID := HeaderValue(msg, "Message-ID")
	originalReferences := HeaderValue(msg, "References")

	if originalFrom == "" {
		return "", fmt.Errorf("original message has no From header")
	}

	// Build reply subject (add "Re: " if not already present)
	replySubject := originalSubject
	if !strings.HasPrefix(strings.ToLower(replySubject), "re:") {
		replySubject = "Re: " + replySubject
	}

	// Build References header for proper threading
	var references string
	if originalReferences != "" {
		references = originalReferences + " " + originalMessageID
	} else {
		references = originalMessageID
	}

	// Build the email message in RFC 2822 format
	var emailBuilder strings.Builder

	// Add To header (reply to original sender)
	emailBuilder.WriteString("To: ")
	emailBuilder.WriteString(originalFrom)
	emailBuilder.WriteString("\r\n")

	// Add Cc header if present
	if len(cc) > 0 {
		emailBuilder.WriteString("Cc: ")
		emailBuilder.WriteString(strings.Join(cc, ", "))
		emailBuilder.WriteString("\r\n")
	}

	// Add Bcc header if present
	if len(bcc) > 0 {
		emailBuilder.WriteString("Bcc: ")
		emailBuilder.WriteString(strings.Join(bcc, ", "))
		emailBuilder.WriteString("\r\n")
	}

	// Add Subject (encode for non-ASCII characters like umlauts)
	emailBuilder.WriteString("Subject: ")
	emailBuilder.WriteString(encodeRFC2047(replySubject))
	emailBuilder.WriteString("\r\n")

	// Add threading headers for proper email threading
	if originalMessageID != "" {
		emailBuilder.WriteString("In-Reply-To: ")
		emailBuilder.WriteString(originalMessageID)
		emailBuilder.WriteString("\r\n")
	}

	if references != "" {
		emailBuilder.WriteString("References: ")
		emailBuilder.WriteString(references)
		emailBuilder.WriteString("\r\n")
	}

	// Add Content-Type
	if isHTML {
		emailBuilder.WriteString("Content-Type: text/html; charset=\"UTF-8\"\r\n")
	} else {
		emailBuilder.WriteString("Content-Type: text/plain; charset=\"UTF-8\"\r\n")
	}
	emailBuilder.WriteString("MIME-Version: 1.0\r\n")
	emailBuilder.WriteString("\r\n")

	// Add body with signature
	bodyWithSignature := c.appendSignature(body, isHTML)
	emailBuilder.WriteString(bodyWithSignature)

	// Encode the message in base64url format
	rawMessage := base64.URLEncoding.EncodeToString([]byte(emailBuilder.String()))

	// Send the reply with threadID to maintain threading
	gmailMsg := &gmail.Message{
		Raw:      rawMessage,
		ThreadId: threadID,
	}

	sent, err := c.svc.Messages.Send("me", gmailMsg).Do()
	if err != nil {
		return "", fmt.Errorf("failed to send reply: %w", err)
	}

	return sent.Id, nil
}

// ForwardEmail forwards an existing email message to new recipients
func (c *Client) ForwardEmail(messageID string, to, cc, bcc []string, additionalBody string, isHTML bool) (string, error) {
	if messageID == "" {
		return "", fmt.Errorf("messageID is required")
	}
	if len(to) == 0 {
		return "", fmt.Errorf("at least one recipient is required")
	}

	// Get the original message
	msg, err := c.GetMessage(messageID)
	if err != nil {
		return "", fmt.Errorf("failed to get original message: %w", err)
	}

	// Extract headers from original message
	originalFrom := HeaderValue(msg, "From")
	originalTo := HeaderValue(msg, "To")
	originalSubject := HeaderValue(msg, "Subject")
	originalDate := HeaderValue(msg, "Date")

	// Build forwarded subject (add "Fwd: " if not already present)
	fwdSubject := originalSubject
	if !strings.HasPrefix(strings.ToLower(fwdSubject), "fwd:") && !strings.HasPrefix(strings.ToLower(fwdSubject), "fw:") {
		fwdSubject = "Fwd: " + fwdSubject
	}

	// Get the original message body
	var originalBody string
	// Try to get HTML body first, then fall back to text
	if isHTML {
		originalBody, _ = c.GetMessageBody(messageID, "html")
		if originalBody == "" {
			originalBody, _ = c.GetMessageBody(messageID, "text")
		}
	} else {
		originalBody, _ = c.GetMessageBody(messageID, "text")
	}

	// Add signature to additional body (the part before the forwarded content)
	additionalBodyWithSignature := c.appendSignature(additionalBody, isHTML)

	// Build the forwarded message body
	var forwardedBody string
	if isHTML {
		forwardedBody = additionalBodyWithSignature + "<br><br>"
		forwardedBody += "---------- Forwarded message ---------<br>"
		forwardedBody += fmt.Sprintf("From: %s<br>", originalFrom)
		forwardedBody += fmt.Sprintf("Date: %s<br>", originalDate)
		forwardedBody += fmt.Sprintf("Subject: %s<br>", originalSubject)
		forwardedBody += fmt.Sprintf("To: %s<br><br>", originalTo)
		forwardedBody += originalBody
	} else {
		forwardedBody = additionalBodyWithSignature + "\n\n"
		forwardedBody += "---------- Forwarded message ---------\n"
		forwardedBody += fmt.Sprintf("From: %s\n", originalFrom)
		forwardedBody += fmt.Sprintf("Date: %s\n", originalDate)
		forwardedBody += fmt.Sprintf("Subject: %s\n", originalSubject)
		forwardedBody += fmt.Sprintf("To: %s\n\n", originalTo)
		forwardedBody += originalBody
	}

	// Build the email message in RFC 2822 format
	var emailBuilder strings.Builder

	// Add To header
	emailBuilder.WriteString("To: ")
	emailBuilder.WriteString(strings.Join(to, ", "))
	emailBuilder.WriteString("\r\n")

	// Add Cc header if present
	if len(cc) > 0 {
		emailBuilder.WriteString("Cc: ")
		emailBuilder.WriteString(strings.Join(cc, ", "))
		emailBuilder.WriteString("\r\n")
	}

	// Add Bcc header if present
	if len(bcc) > 0 {
		emailBuilder.WriteString("Bcc: ")
		emailBuilder.WriteString(strings.Join(bcc, ", "))
		emailBuilder.WriteString("\r\n")
	}

	// Add Subject (encode for non-ASCII characters like umlauts)
	emailBuilder.WriteString("Subject: ")
	emailBuilder.WriteString(encodeRFC2047(fwdSubject))
	emailBuilder.WriteString("\r\n")

	// Add Content-Type
	if isHTML {
		emailBuilder.WriteString("Content-Type: text/html; charset=\"UTF-8\"\r\n")
	} else {
		emailBuilder.WriteString("Content-Type: text/plain; charset=\"UTF-8\"\r\n")
	}
	emailBuilder.WriteString("MIME-Version: 1.0\r\n")
	emailBuilder.WriteString("\r\n")

	// Add forwarded body
	emailBuilder.WriteString(forwardedBody)

	// Encode the message in base64url format
	rawMessage := base64.URLEncoding.EncodeToString([]byte(emailBuilder.String()))

	// Send the forwarded message
	gmailMsg := &gmail.Message{
		Raw: rawMessage,
	}

	sent, err := c.svc.Messages.Send("me", gmailMsg).Do()
	if err != nil {
		return "", fmt.Errorf("failed to forward email: %w", err)
	}

	return sent.Id, nil
}
