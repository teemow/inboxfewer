// Package gmail provides a client for interacting with the Gmail API.
//
// This package offers comprehensive Gmail functionality including:
//   - Thread management (list, archive, iterate)
//   - Email operations (send, reply, forward)
//   - Attachment handling
//   - Contact search across personal, directory, and other contacts
//   - Gmail filters and classification
//   - Unsubscribe link detection
//   - Google Docs link extraction from emails
//
// The client supports multi-account authentication using the Google OAuth2 flow
// and can manage emails across multiple Google accounts. It integrates with both
// the Gmail API (for email operations) and the People API (for contact management).
//
// Authentication:
// This package uses the unified Google OAuth token from the google package.
// For CLI usage, it will prompt for authorization if no token exists.
// For MCP server usage, it will return an error with instructions to use the
// google_get_auth_url and google_save_auth_code tools.
//
// Example usage:
//
//	ctx := context.Background()
//	client, err := gmail.NewClient(ctx)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	// List threads matching a query
//	threads, err := client.ListThreads("in:inbox", 10)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	// Send an email
//	msg := &gmail.EmailMessage{
//	    To:      []string{"recipient@example.com"},
//	    Subject: "Hello",
//	    Body:    "This is a test email",
//	    IsHTML:  false,
//	}
//	msgID, err := client.SendEmail(msg)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	// Search contacts
//	contacts, err := client.SearchContacts("john", 10)
//	if err != nil {
//	    log.Fatal(err)
//	}
package gmail
