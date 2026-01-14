package calendar_tools

import (
	"context"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"

	"github.com/teemow/inboxfewer/internal/server"
	"github.com/teemow/inboxfewer/internal/tools/common"
)

// RegisterCalendarListTools registers calendar list tools with the MCP server
func RegisterCalendarListTools(s *mcpserver.MCPServer, sc *server.ServerContext) error {
	// List calendars tool
	listCalendarsTool := mcp.NewTool("calendar_list_calendars",
		mcp.WithDescription("List all calendars accessible to the user"),
		mcp.WithString("account",
			mcp.Description("Account name (default: 'default'). Used to manage multiple Google accounts."),
		),
	)

	s.AddTool(listCalendarsTool, common.InstrumentedToolHandlerWithService(
		"calendar_list_calendars", "calendar", "list", sc,
		func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return handleListCalendars(ctx, request, sc)
		}))

	// Get calendar tool
	getCalendarTool := mcp.NewTool("calendar_get_calendar",
		mcp.WithDescription("Get information about a specific calendar"),
		mcp.WithString("account",
			mcp.Description("Account name (default: 'default'). Used to manage multiple Google accounts."),
		),
		mcp.WithString("calendarId",
			mcp.Required(),
			mcp.Description("Calendar ID (use 'primary' for primary calendar)"),
		),
	)

	s.AddTool(getCalendarTool, common.InstrumentedToolHandlerWithService(
		"calendar_get_calendar", "calendar", "get", sc,
		func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return handleGetCalendar(ctx, request, sc)
		}))

	return nil
}

func handleListCalendars(ctx context.Context, request mcp.CallToolRequest, sc *server.ServerContext) (*mcp.CallToolResult, error) {
	args := request.GetArguments()
	account := common.GetAccountFromArgs(ctx, args)

	client, err := getCalendarClient(ctx, account, sc)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	calendars, err := client.ListCalendars()
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to list calendars: %v", err)), nil
	}

	result := fmt.Sprintf("Found %d calendar(s):\n\n", len(calendars))
	for i, cal := range calendars {
		result += fmt.Sprintf("%d. %s\n", i+1, cal.Summary)
		result += fmt.Sprintf("   ID: %s\n", cal.ID)
		result += fmt.Sprintf("   Access Role: %s\n", cal.AccessRole)
		if cal.Primary {
			result += "   [PRIMARY]\n"
		}
		if cal.Description != "" {
			result += fmt.Sprintf("   Description: %s\n", cal.Description)
		}
		if cal.TimeZone != "" {
			result += fmt.Sprintf("   Time Zone: %s\n", cal.TimeZone)
		}
		result += "\n"
	}

	return mcp.NewToolResultText(result), nil
}

func handleGetCalendar(ctx context.Context, request mcp.CallToolRequest, sc *server.ServerContext) (*mcp.CallToolResult, error) {
	args := request.GetArguments()
	account := common.GetAccountFromArgs(ctx, args)

	calendarID, ok := args["calendarId"].(string)
	if !ok || calendarID == "" {
		return mcp.NewToolResultError("calendarId is required"), nil
	}

	client, err := getCalendarClient(ctx, account, sc)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	cal, err := client.GetCalendar(calendarID)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to get calendar: %v", err)), nil
	}

	result := fmt.Sprintf("Calendar: %s\n", cal.Summary)
	result += fmt.Sprintf("ID: %s\n", cal.ID)
	result += fmt.Sprintf("Access Role: %s\n", cal.AccessRole)
	if cal.Primary {
		result += "Type: PRIMARY\n"
	}
	if cal.Description != "" {
		result += fmt.Sprintf("Description: %s\n", cal.Description)
	}
	if cal.TimeZone != "" {
		result += fmt.Sprintf("Time Zone: %s\n", cal.TimeZone)
	}

	return mcp.NewToolResultText(result), nil
}
