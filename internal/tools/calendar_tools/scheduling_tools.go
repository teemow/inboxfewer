package calendar_tools

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"

	"github.com/teemow/inboxfewer/internal/server"
	"github.com/teemow/inboxfewer/internal/tools/common"
)

// RegisterSchedulingTools registers scheduling and availability tools with the MCP server
func RegisterSchedulingTools(s *mcpserver.MCPServer, sc *server.ServerContext) error {
	// Query free/busy tool
	queryFreeBusyTool := mcp.NewTool("calendar_query_freebusy",
		mcp.WithDescription("Check availability for one or more calendars/attendees in a time range"),
		mcp.WithString("account",
			mcp.Description("Account name (default: 'default'). Used to manage multiple Google accounts."),
		),
		mcp.WithString("timeMin",
			mcp.Required(),
			mcp.Description("Start time for the range (RFC3339 format, e.g., '2025-01-01T00:00:00Z')"),
		),
		mcp.WithString("timeMax",
			mcp.Required(),
			mcp.Description("End time for the range (RFC3339 format, e.g., '2025-01-31T23:59:59Z')"),
		),
		mcp.WithString("calendars",
			mcp.Required(),
			mcp.Description("Comma-separated list of calendar IDs or email addresses to check"),
		),
	)

	s.AddTool(queryFreeBusyTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleQueryFreeBusy(ctx, request, sc)
	})

	// Find available time tool
	findAvailableTimeTool := mcp.NewTool("calendar_find_available_time",
		mcp.WithDescription("Find available time slots for scheduling a meeting with one or more attendees"),
		mcp.WithString("account",
			mcp.Description("Account name (default: 'default'). Used to manage multiple Google accounts."),
		),
		mcp.WithString("attendees",
			mcp.Required(),
			mcp.Description("Comma-separated list of attendee email addresses"),
		),
		mcp.WithNumber("durationMinutes",
			mcp.Required(),
			mcp.Description("Meeting duration in minutes"),
		),
		mcp.WithString("timeMin",
			mcp.Required(),
			mcp.Description("Start time for search range (RFC3339 format, e.g., '2025-01-01T09:00:00Z')"),
		),
		mcp.WithString("timeMax",
			mcp.Required(),
			mcp.Description("End time for search range (RFC3339 format, e.g., '2025-01-01T17:00:00Z')"),
		),
		mcp.WithNumber("maxResults",
			mcp.Description("Maximum number of available slots to return (default: 10)"),
		),
	)

	s.AddTool(findAvailableTimeTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleFindAvailableTime(ctx, request, sc)
	})

	return nil
}

func handleQueryFreeBusy(ctx context.Context, request mcp.CallToolRequest, sc *server.ServerContext) (*mcp.CallToolResult, error) {
	args := request.GetArguments()
	account := common.GetAccountFromArgs(ctx, args)

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

	calendarsStr, ok := args["calendars"].(string)
	if !ok || calendarsStr == "" {
		return mcp.NewToolResultError("calendars is required"), nil
	}

	calendars := strings.Split(calendarsStr, ",")
	for i := range calendars {
		calendars[i] = strings.TrimSpace(calendars[i])
	}

	client, err := getCalendarClient(ctx, account, sc)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	freeBusyInfos, err := client.QueryFreeBusy(timeMin, timeMax, calendars)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to query free/busy: %v", err)), nil
	}

	result := fmt.Sprintf("Free/Busy information for %d calendar(s):\n\n", len(freeBusyInfos))
	for _, info := range freeBusyInfos {
		result += fmt.Sprintf("Calendar: %s\n", info.Calendar)

		if len(info.Errors) > 0 {
			result += fmt.Sprintf("  Errors: %s\n", strings.Join(info.Errors, ", "))
		}

		if len(info.Busy) == 0 {
			result += "  Status: FREE for entire range\n"
		} else {
			result += fmt.Sprintf("  Busy periods: %d\n", len(info.Busy))
			for i, busy := range info.Busy {
				result += fmt.Sprintf("  %d. %s to %s\n",
					i+1,
					busy.Start.Format("2006-01-02 15:04"),
					busy.End.Format("2006-01-02 15:04"))
			}
		}
		result += "\n"
	}

	return mcp.NewToolResultText(result), nil
}

func handleFindAvailableTime(ctx context.Context, request mcp.CallToolRequest, sc *server.ServerContext) (*mcp.CallToolResult, error) {
	args := request.GetArguments()
	account := common.GetAccountFromArgs(ctx, args)

	attendeesStr, ok := args["attendees"].(string)
	if !ok || attendeesStr == "" {
		return mcp.NewToolResultError("attendees is required"), nil
	}

	attendees := strings.Split(attendeesStr, ",")
	for i := range attendees {
		attendees[i] = strings.TrimSpace(attendees[i])
	}

	durationMinutes, ok := args["durationMinutes"].(float64)
	if !ok || durationMinutes <= 0 {
		return mcp.NewToolResultError("durationMinutes is required and must be positive"), nil
	}
	duration := time.Duration(durationMinutes) * time.Minute

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

	maxResults := 10
	if maxResultsVal, ok := args["maxResults"].(float64); ok && maxResultsVal > 0 {
		maxResults = int(maxResultsVal)
	}

	client, err := getCalendarClient(ctx, account, sc)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	slots, err := client.FindAvailableSlots(attendees, duration, timeMin, timeMax)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to find available time: %v", err)), nil
	}

	if len(slots) == 0 {
		return mcp.NewToolResultText("No available time slots found for the specified criteria"), nil
	}

	// Limit results
	if len(slots) > maxResults {
		slots = slots[:maxResults]
	}

	result := fmt.Sprintf("Found %d available time slot(s) for %d minute meeting:\n\n",
		len(slots), int(durationMinutes))

	for i, slot := range slots {
		result += fmt.Sprintf("%d. %s to %s (%s)\n",
			i+1,
			slot.Start.Format("Mon, Jan 2 at 3:04 PM"),
			slot.End.Format("3:04 PM MST"),
			slot.Start.Weekday())
	}

	return mcp.NewToolResultText(result), nil
}
