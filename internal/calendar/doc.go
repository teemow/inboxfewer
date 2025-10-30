// Package calendar provides a client for interacting with the Google Calendar API.
//
// This package offers functionality for managing calendars and calendar events,
// including creating, reading, updating, and deleting events, as well as
// checking availability and finding available time slots for scheduling meetings.
//
// The client supports multi-account authentication using the Google OAuth2 flow
// and can manage events across multiple Google accounts.
//
// Example usage:
//
//	ctx := context.Background()
//	client, err := calendar.NewClient(ctx)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	// List upcoming events
//	events, err := client.ListEvents("primary", time.Now(), time.Now().AddDate(0, 0, 7), "")
//	if err != nil {
//	    log.Fatal(err)
//	}
package calendar
