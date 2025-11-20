package calendar

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

	"golang.org/x/oauth2"
	calendar "google.golang.org/api/calendar/v3"
	"google.golang.org/api/option"

	"github.com/teemow/inboxfewer/internal/google"
)

// Client wraps the Google Calendar service
type Client struct {
	svc           *calendar.Service
	account       string // The account this client is associated with
	tokenProvider google.TokenProvider
}

// Account returns the account name this client is associated with
func (c *Client) Account() string {
	return c.account
}

// HasTokenForAccountWithProvider checks if a valid OAuth token exists for the specified account
func HasTokenForAccountWithProvider(account string, provider google.TokenProvider) bool {
	if provider == nil {
		return false
	}
	return provider.HasTokenForAccount(account)
}

// HasTokenForAccount checks if a valid OAuth token exists for the specified account
func HasTokenForAccount(account string) bool {
	provider := google.NewFileTokenProvider()
	return HasTokenForAccountWithProvider(account, provider)
}

// HasToken checks if a valid OAuth token exists for the default account
func HasToken() bool {
	return HasTokenForAccount("default")
}

// NewClientForAccountWithProvider creates a new Calendar client with OAuth2 authentication for a specific account
// The OAuth token is retrieved from the provided token provider
func NewClientForAccountWithProvider(ctx context.Context, account string, tokenProvider google.TokenProvider) (*Client, error) {
	if tokenProvider == nil {
		return nil, fmt.Errorf("token provider cannot be nil")
	}

	// Get token from the provided provider
	token, err := tokenProvider.GetTokenForAccount(ctx, account)
	if err != nil {
		return nil, fmt.Errorf("failed to get Google OAuth token for account %s: %w", account, err)
	}

	// Create OAuth2 config and token source
	conf := google.GetOAuthConfig()
	tokenSource := conf.TokenSource(ctx, token)

	// Create HTTP client with the token
	client := oauth2.NewClient(ctx, tokenSource)

	// Force HTTP/1.1 by disabling HTTP/2
	transport := client.Transport.(*oauth2.Transport)
	baseTransport := &http.Transport{
		ForceAttemptHTTP2: false,
	}
	transport.Base = baseTransport

	svc, err := calendar.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return nil, fmt.Errorf("failed to create Calendar service: %w", err)
	}

	return &Client{
		svc:           svc,
		account:       account,
		tokenProvider: tokenProvider,
	}, nil
}

// NewClientForAccount creates a new Calendar client with OAuth2 authentication for a specific account
// Uses the default file-based token provider for backward compatibility
func NewClientForAccount(ctx context.Context, account string) (*Client, error) {
	provider := google.NewFileTokenProvider()
	return NewClientForAccountWithProvider(ctx, account, provider)
}

// NewClient creates a new Calendar client with OAuth2 authentication for the default account
// For CLI usage, it will prompt for auth code via stdin if no token exists
// For MCP usage, it will return an error if no token exists
func NewClient(ctx context.Context) (*Client, error) {
	return NewClientForAccount(ctx, "default")
}

// NewClientWithProvider creates a new Calendar client with OAuth2 authentication for the default account
// using the provided token provider
func NewClientWithProvider(ctx context.Context, provider google.TokenProvider) (*Client, error) {
	return NewClientForAccountWithProvider(ctx, "default", provider)
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
	// For all-day events, use Date instead of DateTime
	if input.AllDay {
		event.Start = &calendar.EventDateTime{
			Date: input.Start.Format("2006-01-02"),
		}
		event.End = &calendar.EventDateTime{
			Date: input.End.Format("2006-01-02"),
		}
	} else {
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
		if input.AllDay {
			existing.Start = &calendar.EventDateTime{
				Date: input.Start.Format("2006-01-02"),
			}
		} else {
			if input.TimeZone == "" {
				input.TimeZone = "UTC"
			}
			existing.Start = &calendar.EventDateTime{
				DateTime: input.Start.Format(time.RFC3339),
				TimeZone: input.TimeZone,
			}
		}
	}
	if !input.End.IsZero() {
		if input.AllDay {
			existing.End = &calendar.EventDateTime{
				Date: input.End.Format("2006-01-02"),
			}
		} else {
			if input.TimeZone == "" {
				input.TimeZone = "UTC"
			}
			existing.End = &calendar.EventDateTime{
				DateTime: input.End.Format(time.RFC3339),
				TimeZone: input.TimeZone,
			}
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
