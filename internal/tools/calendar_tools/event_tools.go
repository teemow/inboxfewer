package calendar_tools

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"

	"github.com/teemow/inboxfewer/internal/calendar"
	"github.com/teemow/inboxfewer/internal/server"
)

// RegisterEventTools registers event-related tools with the MCP server
func RegisterEventTools(s *mcpserver.MCPServer, sc *server.ServerContext, readOnly bool) error {
	// List events tool (read-only, always available)
	listEventsTool := mcp.NewTool("calendar_list_events",
		mcp.WithDescription("List/search calendar events within a time range"),
		mcp.WithString("account",
			mcp.Description("Account name (default: 'default'). Used to manage multiple Google accounts."),
		),
		mcp.WithString("calendarId",
			mcp.Description("Calendar ID (use 'primary' for primary calendar)"),
		),
		mcp.WithString("timeMin",
			mcp.Required(),
			mcp.Description("Start time for the range (RFC3339 format, e.g., '2025-01-01T00:00:00Z')"),
		),
		mcp.WithString("timeMax",
			mcp.Required(),
			mcp.Description("End time for the range (RFC3339 format, e.g., '2025-01-31T23:59:59Z')"),
		),
		mcp.WithString("query",
			mcp.Description("Optional search query to filter events"),
		),
	)

	s.AddTool(listEventsTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleListEvents(ctx, request, sc)
	})

	// Get event tool
	getEventTool := mcp.NewTool("calendar_get_event",
		mcp.WithDescription("Get details of a specific calendar event"),
		mcp.WithString("account",
			mcp.Description("Account name (default: 'default'). Used to manage multiple Google accounts."),
		),
		mcp.WithString("calendarId",
			mcp.Description("Calendar ID (use 'primary' for primary calendar)"),
		),
		mcp.WithString("eventId",
			mcp.Required(),
			mcp.Description("The ID of the event to retrieve"),
		),
	)

	s.AddTool(getEventTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleGetEvent(ctx, request, sc)
	})

	// Create event tool
	createEventTool := mcp.NewTool("calendar_create_event",
		mcp.WithDescription("Create a new calendar event (supports recurring, out-of-office, and Google Meet)"),
		mcp.WithString("account",
			mcp.Description("Account name (default: 'default'). Used to manage multiple Google accounts."),
		),
		mcp.WithString("calendarId",
			mcp.Description("Calendar ID (use 'primary' for primary calendar)"),
		),
		mcp.WithString("summary",
			mcp.Required(),
			mcp.Description("Event title/summary"),
		),
		mcp.WithString("description",
			mcp.Description("Event description"),
		),
		mcp.WithString("location",
			mcp.Description("Event location"),
		),
		mcp.WithString("start",
			mcp.Required(),
			mcp.Description("Start time (RFC3339 format, e.g., '2025-01-15T14:00:00Z')"),
		),
		mcp.WithString("end",
			mcp.Required(),
			mcp.Description("End time (RFC3339 format, e.g., '2025-01-15T15:00:00Z')"),
		),
		mcp.WithString("timeZone",
			mcp.Description("Time zone (e.g., 'America/New_York'). Defaults to UTC."),
		),
		mcp.WithString("attendees",
			mcp.Description("Comma-separated list of attendee email addresses"),
		),
		mcp.WithString("recurrence",
			mcp.Description("Recurrence rule (e.g., 'RRULE:FREQ=WEEKLY;BYDAY=MO,WE,FR')"),
		),
		mcp.WithString("eventType",
			mcp.Description("Event type: 'default', 'outOfOffice', 'focusTime', 'workingLocation'"),
		),
		mcp.WithBoolean("allDay",
			mcp.Description("Create as all-day event (ignores time portion of start/end)"),
		),
		mcp.WithBoolean("addGoogleMeet",
			mcp.Description("Automatically add a Google Meet link to the event"),
		),
		mcp.WithBoolean("guestsCanModify",
			mcp.Description("Allow guests to modify the event"),
		),
		mcp.WithBoolean("guestsCanInviteOthers",
			mcp.Description("Allow guests to invite others"),
		),
		mcp.WithBoolean("guestsCanSeeOtherGuests",
			mcp.Description("Allow guests to see other guests"),
		),
	)

	s.AddTool(createEventTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleCreateEvent(ctx, request, sc)
	})

	// Register update/delete tools only if not in read-only mode
	if !readOnly {
		// Update event tool
		updateEventTool := mcp.NewTool("calendar_update_event",
			mcp.WithDescription("Update an existing calendar event"),
			mcp.WithString("account",
				mcp.Description("Account name (default: 'default'). Used to manage multiple Google accounts."),
			),
			mcp.WithString("calendarId",
				mcp.Description("Calendar ID (use 'primary' for primary calendar)"),
			),
			mcp.WithString("eventId",
				mcp.Required(),
				mcp.Description("The ID of the event to update"),
			),
			mcp.WithString("summary",
				mcp.Description("New event title/summary"),
			),
			mcp.WithString("description",
				mcp.Description("New event description"),
			),
			mcp.WithString("location",
				mcp.Description("New event location"),
			),
			mcp.WithString("start",
				mcp.Description("New start time (RFC3339 format)"),
			),
			mcp.WithString("end",
				mcp.Description("New end time (RFC3339 format)"),
			),
			mcp.WithString("timeZone",
				mcp.Description("Time zone (e.g., 'America/New_York')"),
			),
			mcp.WithString("attendees",
				mcp.Description("New comma-separated list of attendee email addresses"),
			),
			mcp.WithString("eventType",
				mcp.Description("New event type: 'default', 'outOfOffice', 'focusTime', 'workingLocation'"),
			),
			mcp.WithBoolean("allDay",
				mcp.Description("Update to be an all-day event (ignores time portion of start/end)"),
			),
			mcp.WithBoolean("guestsCanModify",
				mcp.Description("Allow guests to modify the event"),
			),
			mcp.WithBoolean("guestsCanInviteOthers",
				mcp.Description("Allow guests to invite others"),
			),
			mcp.WithBoolean("guestsCanSeeOtherGuests",
				mcp.Description("Allow guests to see other guests"),
			),
		)

		s.AddTool(updateEventTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return handleUpdateEvent(ctx, request, sc)
		})

		// Delete event tool
		deleteEventTool := mcp.NewTool("calendar_delete_event",
			mcp.WithDescription("Delete a calendar event"),
			mcp.WithString("account",
				mcp.Description("Account name (default: 'default'). Used to manage multiple Google accounts."),
			),
			mcp.WithString("calendarId",
				mcp.Description("Calendar ID (use 'primary' for primary calendar)"),
			),
			mcp.WithString("eventId",
				mcp.Required(),
				mcp.Description("The ID of the event to delete"),
			),
		)

		s.AddTool(deleteEventTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return handleDeleteEvent(ctx, request, sc)
		})
	}

	// Extract docs links tool (read-only, always available)
	extractDocsLinksTool := mcp.NewTool("calendar_extract_docs_links",
		mcp.WithDescription("Extract Google Docs/Drive links from a calendar event"),
		mcp.WithString("account",
			mcp.Description("Account name (default: 'default'). Used to manage multiple Google accounts."),
		),
		mcp.WithString("calendarId",
			mcp.Description("Calendar ID (use 'primary' for primary calendar)"),
		),
		mcp.WithString("eventId",
			mcp.Required(),
			mcp.Description("The ID of the event"),
		),
	)

	s.AddTool(extractDocsLinksTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleExtractDocsLinks(ctx, request, sc)
	})

	// Get Meet link tool
	getMeetLinkTool := mcp.NewTool("calendar_get_meet_link",
		mcp.WithDescription("Get the Google Meet link from a calendar event"),
		mcp.WithString("account",
			mcp.Description("Account name (default: 'default'). Used to manage multiple Google accounts."),
		),
		mcp.WithString("calendarId",
			mcp.Description("Calendar ID (use 'primary' for primary calendar)"),
		),
		mcp.WithString("eventId",
			mcp.Required(),
			mcp.Description("The ID of the event"),
		),
	)

	s.AddTool(getMeetLinkTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleGetMeetLink(ctx, request, sc)
	})

	return nil
}

func handleListEvents(ctx context.Context, request mcp.CallToolRequest, sc *server.ServerContext) (*mcp.CallToolResult, error) {
	args := request.GetArguments()
	account := getAccountFromArgs(args)

	calendarID := "primary"
	if calIDVal, ok := args["calendarId"].(string); ok && calIDVal != "" {
		calendarID = calIDVal
	}

	timeMinStr, ok := args["timeMin"].(string)
	if !ok || timeMinStr == "" {
		return mcp.NewToolResultError("timeMin is required"), nil
	}
	timeMin, err := time.Parse(time.RFC3339, timeMinStr)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Invalid timeMin format: %v", err)), nil
	}

	timeMaxStr, ok := args["timeMax"].(string)
	if !ok || timeMaxStr == "" {
		return mcp.NewToolResultError("timeMax is required"), nil
	}
	timeMax, err := time.Parse(time.RFC3339, timeMaxStr)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Invalid timeMax format: %v", err)), nil
	}

	query := ""
	if queryVal, ok := args["query"].(string); ok {
		query = queryVal
	}

	client, err := getCalendarClient(ctx, account, sc)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	events, err := client.ListEvents(calendarID, timeMin, timeMax, query)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to list events: %v", err)), nil
	}

	result := fmt.Sprintf("Found %d events:\n\n", len(events))
	for i, event := range events {
		result += fmt.Sprintf("%d. %s\n", i+1, event.Summary)
		result += fmt.Sprintf("   ID: %s\n", event.ID)
		result += fmt.Sprintf("   Start: %s\n", event.Start.Format(time.RFC3339))
		result += fmt.Sprintf("   End: %s\n", event.End.Format(time.RFC3339))
		if event.Location != "" {
			result += fmt.Sprintf("   Location: %s\n", event.Location)
		}
		if event.MeetLink != "" {
			result += fmt.Sprintf("   Meet: %s\n", event.MeetLink)
		}
		if len(event.Attendees) > 0 {
			result += fmt.Sprintf("   Attendees: %d\n", len(event.Attendees))
		}
		result += "\n"
	}

	return mcp.NewToolResultText(result), nil
}

func handleGetEvent(ctx context.Context, request mcp.CallToolRequest, sc *server.ServerContext) (*mcp.CallToolResult, error) {
	args := request.GetArguments()
	account := getAccountFromArgs(args)

	calendarID := "primary"
	if calIDVal, ok := args["calendarId"].(string); ok && calIDVal != "" {
		calendarID = calIDVal
	}

	eventID, ok := args["eventId"].(string)
	if !ok || eventID == "" {
		return mcp.NewToolResultError("eventId is required"), nil
	}

	client, err := getCalendarClient(ctx, account, sc)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	event, err := client.GetEvent(calendarID, eventID)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to get event: %v", err)), nil
	}

	result := fmt.Sprintf("Event: %s\n", event.Summary)
	result += fmt.Sprintf("ID: %s\n", event.ID)
	result += fmt.Sprintf("Start: %s\n", event.Start.Format(time.RFC3339))
	result += fmt.Sprintf("End: %s\n", event.End.Format(time.RFC3339))
	result += fmt.Sprintf("Status: %s\n", event.Status)
	if event.Description != "" {
		result += fmt.Sprintf("Description: %s\n", event.Description)
	}
	if event.Location != "" {
		result += fmt.Sprintf("Location: %s\n", event.Location)
	}
	if event.Creator != "" {
		result += fmt.Sprintf("Creator: %s\n", event.Creator)
	}
	if event.Organizer != "" {
		result += fmt.Sprintf("Organizer: %s\n", event.Organizer)
	}
	if event.MeetLink != "" {
		result += fmt.Sprintf("Google Meet: %s\n", event.MeetLink)
	}
	if event.EventType != "" {
		result += fmt.Sprintf("Type: %s\n", event.EventType)
	}

	if len(event.Attendees) > 0 {
		result += fmt.Sprintf("\nAttendees (%d):\n", len(event.Attendees))
		for _, att := range event.Attendees {
			result += fmt.Sprintf("  - %s (%s)", att.Email, att.ResponseStatus)
			if att.DisplayName != "" {
				result += fmt.Sprintf(" - %s", att.DisplayName)
			}
			if att.Optional {
				result += " [optional]"
			}
			result += "\n"
		}
	}

	return mcp.NewToolResultText(result), nil
}

func handleCreateEvent(ctx context.Context, request mcp.CallToolRequest, sc *server.ServerContext) (*mcp.CallToolResult, error) {
	args := request.GetArguments()
	account := getAccountFromArgs(args)

	calendarID := "primary"
	if calIDVal, ok := args["calendarId"].(string); ok && calIDVal != "" {
		calendarID = calIDVal
	}

	summary, ok := args["summary"].(string)
	if !ok || summary == "" {
		return mcp.NewToolResultError("summary is required"), nil
	}

	startStr, ok := args["start"].(string)
	if !ok || startStr == "" {
		return mcp.NewToolResultError("start is required"), nil
	}
	start, err := time.Parse(time.RFC3339, startStr)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Invalid start format: %v", err)), nil
	}

	endStr, ok := args["end"].(string)
	if !ok || endStr == "" {
		return mcp.NewToolResultError("end is required"), nil
	}
	end, err := time.Parse(time.RFC3339, endStr)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Invalid end format: %v", err)), nil
	}

	input := calendar.EventInput{
		Summary: summary,
		Start:   start,
		End:     end,
	}

	if desc, ok := args["description"].(string); ok {
		input.Description = desc
	}
	if loc, ok := args["location"].(string); ok {
		input.Location = loc
	}
	if tz, ok := args["timeZone"].(string); ok {
		input.TimeZone = tz
	}
	if eventType, ok := args["eventType"].(string); ok {
		input.EventType = eventType
	}

	if attendeesStr, ok := args["attendees"].(string); ok && attendeesStr != "" {
		input.Attendees = strings.Split(attendeesStr, ",")
		for i := range input.Attendees {
			input.Attendees[i] = strings.TrimSpace(input.Attendees[i])
		}
	}

	if recurrence, ok := args["recurrence"].(string); ok && recurrence != "" {
		input.Recurrence = []string{recurrence}
	}

	if allDay, ok := args["allDay"].(bool); ok {
		input.AllDay = allDay
	}
	if addMeet, ok := args["addGoogleMeet"].(bool); ok {
		input.UseDefaultConferenceData = addMeet
	}
	if guestsCanModify, ok := args["guestsCanModify"].(bool); ok {
		input.GuestsCanModify = guestsCanModify
	}
	if guestsCanInvite, ok := args["guestsCanInviteOthers"].(bool); ok {
		input.GuestsCanInviteOthers = guestsCanInvite
	}
	if guestsCanSee, ok := args["guestsCanSeeOtherGuests"].(bool); ok {
		input.GuestsCanSeeOtherGuests = guestsCanSee
	}

	client, err := getCalendarClient(ctx, account, sc)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	event, err := client.CreateEvent(calendarID, input)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to create event: %v", err)), nil
	}

	result := fmt.Sprintf("Successfully created event: %s\n", event.Summary)
	result += fmt.Sprintf("ID: %s\n", event.ID)
	result += fmt.Sprintf("Start: %s\n", event.Start.Format(time.RFC3339))
	result += fmt.Sprintf("End: %s\n", event.End.Format(time.RFC3339))
	if event.MeetLink != "" {
		result += fmt.Sprintf("Google Meet: %s\n", event.MeetLink)
	}

	return mcp.NewToolResultText(result), nil
}

func handleUpdateEvent(ctx context.Context, request mcp.CallToolRequest, sc *server.ServerContext) (*mcp.CallToolResult, error) {
	args := request.GetArguments()
	account := getAccountFromArgs(args)

	calendarID := "primary"
	if calIDVal, ok := args["calendarId"].(string); ok && calIDVal != "" {
		calendarID = calIDVal
	}

	eventID, ok := args["eventId"].(string)
	if !ok || eventID == "" {
		return mcp.NewToolResultError("eventId is required"), nil
	}

	input := calendar.EventInput{}

	if summary, ok := args["summary"].(string); ok {
		input.Summary = summary
	}
	if desc, ok := args["description"].(string); ok {
		input.Description = desc
	}
	if loc, ok := args["location"].(string); ok {
		input.Location = loc
	}
	if tz, ok := args["timeZone"].(string); ok {
		input.TimeZone = tz
	}
	if eventType, ok := args["eventType"].(string); ok {
		input.EventType = eventType
	}

	if startStr, ok := args["start"].(string); ok && startStr != "" {
		start, err := time.Parse(time.RFC3339, startStr)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Invalid start format: %v", err)), nil
		}
		input.Start = start
	}

	if endStr, ok := args["end"].(string); ok && endStr != "" {
		end, err := time.Parse(time.RFC3339, endStr)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Invalid end format: %v", err)), nil
		}
		input.End = end
	}

	if attendeesStr, ok := args["attendees"].(string); ok && attendeesStr != "" {
		input.Attendees = strings.Split(attendeesStr, ",")
		for i := range input.Attendees {
			input.Attendees[i] = strings.TrimSpace(input.Attendees[i])
		}
	}

	if allDay, ok := args["allDay"].(bool); ok {
		input.AllDay = allDay
	}
	if guestsCanModify, ok := args["guestsCanModify"].(bool); ok {
		input.GuestsCanModify = guestsCanModify
	}
	if guestsCanInvite, ok := args["guestsCanInviteOthers"].(bool); ok {
		input.GuestsCanInviteOthers = guestsCanInvite
	}
	if guestsCanSee, ok := args["guestsCanSeeOtherGuests"].(bool); ok {
		input.GuestsCanSeeOtherGuests = guestsCanSee
	}

	client, err := getCalendarClient(ctx, account, sc)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	event, err := client.UpdateEvent(calendarID, eventID, input)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to update event: %v", err)), nil
	}

	result := fmt.Sprintf("Successfully updated event: %s\n", event.Summary)
	result += fmt.Sprintf("ID: %s\n", event.ID)
	result += fmt.Sprintf("Start: %s\n", event.Start.Format(time.RFC3339))
	result += fmt.Sprintf("End: %s\n", event.End.Format(time.RFC3339))

	return mcp.NewToolResultText(result), nil
}

func handleDeleteEvent(ctx context.Context, request mcp.CallToolRequest, sc *server.ServerContext) (*mcp.CallToolResult, error) {
	args := request.GetArguments()
	account := getAccountFromArgs(args)

	calendarID := "primary"
	if calIDVal, ok := args["calendarId"].(string); ok && calIDVal != "" {
		calendarID = calIDVal
	}

	eventID, ok := args["eventId"].(string)
	if !ok || eventID == "" {
		return mcp.NewToolResultError("eventId is required"), nil
	}

	client, err := getCalendarClient(ctx, account, sc)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	if err := client.DeleteEvent(calendarID, eventID); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to delete event: %v", err)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Successfully deleted event %s", eventID)), nil
}

func handleExtractDocsLinks(ctx context.Context, request mcp.CallToolRequest, sc *server.ServerContext) (*mcp.CallToolResult, error) {
	args := request.GetArguments()
	account := getAccountFromArgs(args)

	calendarID := "primary"
	if calIDVal, ok := args["calendarId"].(string); ok && calIDVal != "" {
		calendarID = calIDVal
	}

	eventID, ok := args["eventId"].(string)
	if !ok || eventID == "" {
		return mcp.NewToolResultError("eventId is required"), nil
	}

	client, err := getCalendarClient(ctx, account, sc)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	links, err := client.ExtractGoogleDocsLinks(calendarID, eventID)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to extract docs links: %v", err)), nil
	}

	if len(links) == 0 {
		return mcp.NewToolResultText("No Google Docs/Drive links found in this event"), nil
	}

	result := fmt.Sprintf("Found %d Google Docs/Drive link(s):\n\n", len(links))
	for i, link := range links {
		result += fmt.Sprintf("%d. %s\n", i+1, link)
	}

	return mcp.NewToolResultText(result), nil
}

func handleGetMeetLink(ctx context.Context, request mcp.CallToolRequest, sc *server.ServerContext) (*mcp.CallToolResult, error) {
	args := request.GetArguments()
	account := getAccountFromArgs(args)

	calendarID := "primary"
	if calIDVal, ok := args["calendarId"].(string); ok && calIDVal != "" {
		calendarID = calIDVal
	}

	eventID, ok := args["eventId"].(string)
	if !ok || eventID == "" {
		return mcp.NewToolResultError("eventId is required"), nil
	}

	client, err := getCalendarClient(ctx, account, sc)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	meetLink, err := client.GetGoogleMeetLink(calendarID, eventID)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to get Meet link: %v", err)), nil
	}

	if meetLink == "" {
		return mcp.NewToolResultText("No Google Meet link found for this event"), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Google Meet link: %s", meetLink)), nil
}
