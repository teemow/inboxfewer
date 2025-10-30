package calendar

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"regexp"
	"strings"
	"time"

	calendar "google.golang.org/api/calendar/v3"
	"google.golang.org/api/option"

	"github.com/teemow/inboxfewer/internal/google"
)

// Client wraps the Google Calendar service
type Client struct {
	svc     *calendar.Service
	account string // The account this client is associated with
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

// NewClientForAccount creates a new Calendar client with OAuth2 authentication for a specific account
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

	svc, err := calendar.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return nil, fmt.Errorf("failed to create Calendar service: %w", err)
	}

	return &Client{
		svc:     svc,
		account: account,
	}, nil
}

// NewClient creates a new Calendar client with OAuth2 authentication for the default account
// For CLI usage, it will prompt for auth code via stdin if no token exists
// For MCP usage, it will return an error if no token exists
func NewClient(ctx context.Context) (*Client, error) {
	return NewClientForAccount(ctx, "default")
}

// isTerminal checks if stdin is connected to a terminal (CLI mode)
func isTerminal() bool {
	fileInfo, _ := os.Stdin.Stat()
	return (fileInfo.Mode() & os.ModeCharDevice) != 0
}

// ListEvents lists events in a calendar within a time range
func (c *Client) ListEvents(calendarID string, timeMin, timeMax time.Time, query string) ([]EventSummary, error) {
	call := c.svc.Events.List(calendarID).
		TimeMin(timeMin.Format(time.RFC3339)).
		TimeMax(timeMax.Format(time.RFC3339)).
		SingleEvents(true).
		OrderBy("startTime")

	if query != "" {
		call = call.Q(query)
	}

	events, err := call.Do()
	if err != nil {
		return nil, fmt.Errorf("failed to list events: %w", err)
	}

	var summaries []EventSummary
	for _, event := range events.Items {
		summaries = append(summaries, toEventSummary(event))
	}

	return summaries, nil
}

// GetEvent retrieves a specific event by ID
func (c *Client) GetEvent(calendarID, eventID string) (*EventSummary, error) {
	event, err := c.svc.Events.Get(calendarID, eventID).Do()
	if err != nil {
		return nil, fmt.Errorf("failed to get event: %w", err)
	}

	summary := toEventSummary(event)
	return &summary, nil
}

// CreateEvent creates a new calendar event
func (c *Client) CreateEvent(calendarID string, input EventInput) (*EventSummary, error) {
	event := &calendar.Event{
		Summary:     input.Summary,
		Description: input.Description,
		Location:    input.Location,
		EventType:   input.EventType,
	}

	// Set start and end times
	if input.TimeZone == "" {
		input.TimeZone = "UTC"
	}

	event.Start = &calendar.EventDateTime{
		DateTime: input.Start.Format(time.RFC3339),
		TimeZone: input.TimeZone,
	}
	event.End = &calendar.EventDateTime{
		DateTime: input.End.Format(time.RFC3339),
		TimeZone: input.TimeZone,
	}

	// Set attendees
	if len(input.Attendees) > 0 {
		var attendees []*calendar.EventAttendee
		for _, email := range input.Attendees {
			attendees = append(attendees, &calendar.EventAttendee{
				Email: email,
			})
		}
		event.Attendees = attendees
	}

	// Set recurrence rules
	if len(input.Recurrence) > 0 {
		event.Recurrence = input.Recurrence
	}

	// Set guest permissions
	event.GuestsCanModify = input.GuestsCanModify
	if input.GuestsCanInviteOthers {
		guestsCanInvite := true
		event.GuestsCanInviteOthers = &guestsCanInvite
	}
	if input.GuestsCanSeeOtherGuests {
		guestsCanSee := true
		event.GuestsCanSeeOtherGuests = &guestsCanSee
	}

	// Add conference data (Google Meet)
	call := c.svc.Events.Insert(calendarID, event)
	if input.UseDefaultConferenceData {
		call = call.ConferenceDataVersion(1)
		event.ConferenceData = &calendar.ConferenceData{
			CreateRequest: &calendar.CreateConferenceRequest{
				RequestId: fmt.Sprintf("meet-%d", time.Now().Unix()),
			},
		}
	}

	created, err := call.Do()
	if err != nil {
		return nil, fmt.Errorf("failed to create event: %w", err)
	}

	summary := toEventSummary(created)
	return &summary, nil
}

// UpdateEvent updates an existing calendar event
func (c *Client) UpdateEvent(calendarID, eventID string, input EventInput) (*EventSummary, error) {
	// Get the existing event first
	existing, err := c.svc.Events.Get(calendarID, eventID).Do()
	if err != nil {
		return nil, fmt.Errorf("failed to get existing event: %w", err)
	}

	// Update fields
	if input.Summary != "" {
		existing.Summary = input.Summary
	}
	if input.Description != "" {
		existing.Description = input.Description
	}
	if input.Location != "" {
		existing.Location = input.Location
	}
	if input.EventType != "" {
		existing.EventType = input.EventType
	}

	// Update times if provided
	if !input.Start.IsZero() {
		if input.TimeZone == "" {
			input.TimeZone = "UTC"
		}
		existing.Start = &calendar.EventDateTime{
			DateTime: input.Start.Format(time.RFC3339),
			TimeZone: input.TimeZone,
		}
	}
	if !input.End.IsZero() {
		if input.TimeZone == "" {
			input.TimeZone = "UTC"
		}
		existing.End = &calendar.EventDateTime{
			DateTime: input.End.Format(time.RFC3339),
			TimeZone: input.TimeZone,
		}
	}

	// Update attendees if provided
	if len(input.Attendees) > 0 {
		var attendees []*calendar.EventAttendee
		for _, email := range input.Attendees {
			attendees = append(attendees, &calendar.EventAttendee{
				Email: email,
			})
		}
		existing.Attendees = attendees
	}

	// Update recurrence if provided
	if len(input.Recurrence) > 0 {
		existing.Recurrence = input.Recurrence
	}

	// Update guest permissions
	existing.GuestsCanModify = input.GuestsCanModify
	if input.GuestsCanInviteOthers {
		guestsCanInvite := true
		existing.GuestsCanInviteOthers = &guestsCanInvite
	}
	if input.GuestsCanSeeOtherGuests {
		guestsCanSee := true
		existing.GuestsCanSeeOtherGuests = &guestsCanSee
	}

	// Update the event
	updated, err := c.svc.Events.Update(calendarID, eventID, existing).Do()
	if err != nil {
		return nil, fmt.Errorf("failed to update event: %w", err)
	}

	summary := toEventSummary(updated)
	return &summary, nil
}

// DeleteEvent deletes a calendar event
func (c *Client) DeleteEvent(calendarID, eventID string) error {
	err := c.svc.Events.Delete(calendarID, eventID).Do()
	if err != nil {
		return fmt.Errorf("failed to delete event: %w", err)
	}
	return nil
}

// ListCalendars lists all calendars accessible to the user
func (c *Client) ListCalendars() ([]CalendarInfo, error) {
	list, err := c.svc.CalendarList.List().Do()
	if err != nil {
		return nil, fmt.Errorf("failed to list calendars: %w", err)
	}

	var calendars []CalendarInfo
	for _, entry := range list.Items {
		calendars = append(calendars, toCalendarInfo(entry))
	}

	return calendars, nil
}

// GetCalendar retrieves information about a specific calendar
func (c *Client) GetCalendar(calendarID string) (*CalendarInfo, error) {
	entry, err := c.svc.CalendarList.Get(calendarID).Do()
	if err != nil {
		return nil, fmt.Errorf("failed to get calendar: %w", err)
	}

	info := toCalendarInfo(entry)
	return &info, nil
}

// GetPrimaryCalendar retrieves information about the primary calendar
func (c *Client) GetPrimaryCalendar() (*CalendarInfo, error) {
	return c.GetCalendar("primary")
}

// QueryFreeBusy checks availability for calendars in a time range
func (c *Client) QueryFreeBusy(timeMin, timeMax time.Time, calendarIDs []string) ([]FreeBusyInfo, error) {
	items := make([]*calendar.FreeBusyRequestItem, len(calendarIDs))
	for i, id := range calendarIDs {
		items[i] = &calendar.FreeBusyRequestItem{Id: id}
	}

	query := &calendar.FreeBusyRequest{
		TimeMin: timeMin.Format(time.RFC3339),
		TimeMax: timeMax.Format(time.RFC3339),
		Items:   items,
	}

	result, err := c.svc.Freebusy.Query(query).Do()
	if err != nil {
		return nil, fmt.Errorf("failed to query freebusy: %w", err)
	}

	var infos []FreeBusyInfo
	for calID, cal := range result.Calendars {
		info := FreeBusyInfo{
			Calendar: calID,
		}

		// Add busy time ranges
		for _, busy := range cal.Busy {
			start, _ := time.Parse(time.RFC3339, busy.Start)
			end, _ := time.Parse(time.RFC3339, busy.End)
			info.Busy = append(info.Busy, TimeRange{Start: start, End: end})
		}

		// Add errors if any
		for _, err := range cal.Errors {
			info.Errors = append(info.Errors, err.Reason)
		}

		infos = append(infos, info)
	}

	return infos, nil
}

// FindAvailableSlots finds available time slots for scheduling a meeting
// It checks the availability of all specified attendees and returns slots where everyone is free
func (c *Client) FindAvailableSlots(attendees []string, duration time.Duration, timeMin, timeMax time.Time) ([]AvailableSlot, error) {
	// Query freebusy for all attendees
	freeBusyInfos, err := c.QueryFreeBusy(timeMin, timeMax, attendees)
	if err != nil {
		return nil, err
	}

	// Merge all busy times into a single list
	var allBusyTimes []TimeRange
	for _, info := range freeBusyInfos {
		allBusyTimes = append(allBusyTimes, info.Busy...)
	}

	// Sort busy times by start time
	// For simplicity, we'll use a basic algorithm to find gaps
	// This could be optimized with better interval merging

	// Find gaps in busy times
	var availableSlots []AvailableSlot

	currentTime := timeMin
	for currentTime.Add(duration).Before(timeMax) || currentTime.Add(duration).Equal(timeMax) {
		slotEnd := currentTime.Add(duration)

		// Check if this slot overlaps with any busy time
		isFree := true
		for _, busy := range allBusyTimes {
			// Check for overlap
			if (currentTime.Before(busy.End) || currentTime.Equal(busy.End)) &&
				(slotEnd.After(busy.Start) || slotEnd.Equal(busy.Start)) {
				isFree = false
				// Skip to the end of this busy period
				if busy.End.After(currentTime) {
					currentTime = busy.End
				}
				break
			}
		}

		if isFree {
			availableSlots = append(availableSlots, AvailableSlot{
				Start:    currentTime,
				End:      slotEnd,
				Duration: duration,
			})
			// Move to next potential slot (15-minute increments)
			currentTime = currentTime.Add(15 * time.Minute)
		}
	}

	return availableSlots, nil
}

// ExtractGoogleDocsLinks extracts Google Docs links from an event
func (c *Client) ExtractGoogleDocsLinks(calendarID, eventID string) ([]string, error) {
	event, err := c.svc.Events.Get(calendarID, eventID).Do()
	if err != nil {
		return nil, fmt.Errorf("failed to get event: %w", err)
	}

	var docsLinks []string

	// Check attachments
	for _, attachment := range event.Attachments {
		if attachment.FileUrl != "" && isGoogleDocsLink(attachment.FileUrl) {
			docsLinks = append(docsLinks, attachment.FileUrl)
		}
	}

	// Check description for embedded links
	if event.Description != "" {
		links := extractLinksFromText(event.Description)
		for _, link := range links {
			if isGoogleDocsLink(link) {
				docsLinks = append(docsLinks, link)
			}
		}
	}

	return docsLinks, nil
}

// GetGoogleMeetLink retrieves the Google Meet link from an event
func (c *Client) GetGoogleMeetLink(calendarID, eventID string) (string, error) {
	event, err := c.svc.Events.Get(calendarID, eventID).Do()
	if err != nil {
		return "", fmt.Errorf("failed to get event: %w", err)
	}

	if event.ConferenceData != nil {
		for _, ep := range event.ConferenceData.EntryPoints {
			if ep.EntryPointType == "video" {
				return ep.Uri, nil
			}
		}
	}

	return "", nil
}

// isGoogleDocsLink checks if a URL is a Google Docs link
func isGoogleDocsLink(url string) bool {
	return strings.Contains(url, "docs.google.com") ||
		strings.Contains(url, "drive.google.com")
}

// extractLinksFromText extracts URLs from text
func extractLinksFromText(text string) []string {
	// Simple regex for URLs
	urlRegex := regexp.MustCompile(`https?://[^\s<>"]+`)
	matches := urlRegex.FindAllString(text, -1)
	return matches
}
