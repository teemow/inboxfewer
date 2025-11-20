package meet_tools

import (
	"context"
	"fmt"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"

	"github.com/teemow/inboxfewer/internal/google"
	"github.com/teemow/inboxfewer/internal/meet"
	"github.com/teemow/inboxfewer/internal/server"
)

// getAccountFromArgs extracts the account name from request arguments, defaulting to "default"
func getAccountFromArgs(args map[string]interface{}) string {
	account := "default"
	if accountVal, ok := args["account"].(string); ok && accountVal != "" {
		account = accountVal
	}
	return account
}

// getMeetClient retrieves or creates a meet client for the specified account
func getMeetClient(ctx context.Context, account string, sc *server.ServerContext) (*meet.Client, error) {
	client := sc.MeetClientForAccount(account)
	if client == nil {
		// Check if token exists before trying to create client
		if !meet.HasTokenForAccount(account) {
			authURL := google.GetAuthenticationErrorMessage(account)
			return nil, fmt.Errorf(`Google OAuth token not found for account "%s". To authorize access:

1. Visit this URL in your browser:
   %s

2. Sign in with your Google account
3. Grant access to Google services (Calendar, Gmail, Docs, Drive, Meet)
4. Copy the authorization code

5. Provide the authorization code to your AI agent
   The agent will use the google_save_auth_code tool with account="%s" to complete authentication.

Note: You only need to authorize once. The tokens will be automatically refreshed.`, account, authURL, account)
		}

		var err error
		client, err = meet.NewClientForAccount(ctx, account)
		if err != nil {
			return nil, fmt.Errorf("failed to create Meet client for account %s: %w", account, err)
		}
		sc.SetMeetClientForAccount(account, client)
	}

	return client, nil
}

// RegisterMeetTools registers all Meet-related tools with the MCP server
func RegisterMeetTools(s *mcpserver.MCPServer, sc *server.ServerContext, readOnly bool) error {
	// Get conference record (read-only, always available)
	getConferenceTool := mcp.NewTool("meet_get_conference",
		mcp.WithDescription("Get details about a Google Meet conference record"),
		mcp.WithString("account",
			mcp.Description("Account name (default: 'default'). Used to manage multiple Google accounts."),
		),
		mcp.WithString("conference_record",
			mcp.Required(),
			mcp.Description("The resource name of the conference record (e.g., 'spaces/SPACE_ID/conferenceRecords/CONF_ID')"),
		),
	)

	s.AddTool(getConferenceTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleGetConference(ctx, request, sc)
	})

	// List recordings
	listRecordingsTool := mcp.NewTool("meet_list_recordings",
		mcp.WithDescription("List all recordings for a Google Meet conference"),
		mcp.WithString("account",
			mcp.Description("Account name (default: 'default'). Used to manage multiple Google accounts."),
		),
		mcp.WithString("conference_record",
			mcp.Required(),
			mcp.Description("The resource name of the conference record (e.g., 'spaces/SPACE_ID/conferenceRecords/CONF_ID')"),
		),
	)

	s.AddTool(listRecordingsTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleListRecordings(ctx, request, sc)
	})

	// Get recording
	getRecordingTool := mcp.NewTool("meet_get_recording",
		mcp.WithDescription("Get details about a specific Google Meet recording, including download link"),
		mcp.WithString("account",
			mcp.Description("Account name (default: 'default'). Used to manage multiple Google accounts."),
		),
		mcp.WithString("recording_name",
			mcp.Required(),
			mcp.Description("The resource name of the recording (e.g., 'spaces/SPACE_ID/conferenceRecords/CONF_ID/recordings/REC_ID')"),
		),
	)

	s.AddTool(getRecordingTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleGetRecording(ctx, request, sc)
	})

	// List transcripts
	listTranscriptsTool := mcp.NewTool("meet_list_transcripts",
		mcp.WithDescription("List all transcripts for a Google Meet conference"),
		mcp.WithString("account",
			mcp.Description("Account name (default: 'default'). Used to manage multiple Google accounts."),
		),
		mcp.WithString("conference_record",
			mcp.Required(),
			mcp.Description("The resource name of the conference record (e.g., 'spaces/SPACE_ID/conferenceRecords/CONF_ID')"),
		),
	)

	s.AddTool(listTranscriptsTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleListTranscripts(ctx, request, sc)
	})

	// Get transcript
	getTranscriptTool := mcp.NewTool("meet_get_transcript",
		mcp.WithDescription("Get details about a specific Google Meet transcript"),
		mcp.WithString("account",
			mcp.Description("Account name (default: 'default'). Used to manage multiple Google accounts."),
		),
		mcp.WithString("transcript_name",
			mcp.Required(),
			mcp.Description("The resource name of the transcript (e.g., 'spaces/SPACE_ID/conferenceRecords/CONF_ID/transcripts/TRANS_ID')"),
		),
	)

	s.AddTool(getTranscriptTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleGetTranscript(ctx, request, sc)
	})

	// Get transcript text
	getTranscriptTextTool := mcp.NewTool("meet_get_transcript_text",
		mcp.WithDescription("Get the full text content of a Google Meet transcript with timestamps and speakers"),
		mcp.WithString("account",
			mcp.Description("Account name (default: 'default'). Used to manage multiple Google accounts."),
		),
		mcp.WithString("transcript_name",
			mcp.Required(),
			mcp.Description("The resource name of the transcript (e.g., 'spaces/SPACE_ID/conferenceRecords/CONF_ID/transcripts/TRANS_ID')"),
		),
	)

	s.AddTool(getTranscriptTextTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleGetTranscriptText(ctx, request, sc)
	})

	// Meet space configuration tools (safe operations)
	// Create space
	createSpaceTool := mcp.NewTool("meet_create_space",
		mcp.WithDescription("Create a new Google Meet space with optional auto-recording, transcription, and note-taking configuration"),
		mcp.WithString("account",
			mcp.Description("Account name (default: 'default'). Used to manage multiple Google accounts."),
		),
		mcp.WithString("access_type",
			mcp.Description("Who can join without knocking: 'OPEN', 'TRUSTED', 'RESTRICTED' (optional)"),
		),
		mcp.WithBoolean("enable_recording",
			mcp.Description("Enable automatic recording (default: false)"),
		),
		mcp.WithBoolean("enable_transcription",
			mcp.Description("Enable automatic transcription (default: false)"),
		),
		mcp.WithBoolean("enable_smart_notes",
			mcp.Description("Enable automatic note-taking with Gemini (default: false). Requires Gemini add-on."),
		),
	)

	s.AddTool(createSpaceTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleCreateSpace(ctx, request, sc)
	})

	// Get space (read-only, always available)
	getSpaceTool := mcp.NewTool("meet_get_space",
		mcp.WithDescription("Get details about a Google Meet space including its configuration"),
		mcp.WithString("account",
			mcp.Description("Account name (default: 'default'). Used to manage multiple Google accounts."),
		),
		mcp.WithString("space_name",
			mcp.Required(),
			mcp.Description("The resource name of the space (e.g., 'spaces/SPACE_ID')"),
		),
	)

	s.AddTool(getSpaceTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleGetSpace(ctx, request, sc)
	})

	// Update space configuration (safe operation)
	updateSpaceConfigTool := mcp.NewTool("meet_update_space_config",
		mcp.WithDescription("Update the configuration of an existing Google Meet space (enable/disable auto-recording, transcription, notes)"),
		mcp.WithString("account",
			mcp.Description("Account name (default: 'default'). Used to manage multiple Google accounts."),
		),
		mcp.WithString("space_name",
			mcp.Required(),
			mcp.Description("The resource name of the space to update (e.g., 'spaces/SPACE_ID')"),
		),
		mcp.WithString("access_type",
			mcp.Description("Who can join without knocking: 'OPEN', 'TRUSTED', 'RESTRICTED' (optional)"),
		),
		mcp.WithBoolean("enable_recording",
			mcp.Description("Enable automatic recording"),
		),
		mcp.WithBoolean("enable_transcription",
			mcp.Description("Enable automatic transcription"),
		),
		mcp.WithBoolean("enable_smart_notes",
			mcp.Description("Enable automatic note-taking with Gemini. Requires Gemini add-on."),
		),
	)

	s.AddTool(updateSpaceConfigTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleUpdateSpaceConfig(ctx, request, sc)
	})

	return nil
}

func handleGetConference(ctx context.Context, request mcp.CallToolRequest, sc *server.ServerContext) (*mcp.CallToolResult, error) {
	args := request.GetArguments()
	account := getAccountFromArgs(args)

	conferenceRecord, ok := args["conference_record"].(string)
	if !ok || conferenceRecord == "" {
		return mcp.NewToolResultError("conference_record is required"), nil
	}

	client, err := getMeetClient(ctx, account, sc)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	record, err := client.GetConferenceRecord(conferenceRecord)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to get conference record: %v", err)), nil
	}

	result := fmt.Sprintf("Conference Record: %s\n", record.Name)
	result += fmt.Sprintf("Space ID: %s\n", record.SpaceID)
	result += fmt.Sprintf("Meeting Code: %s\n", record.MeetingCode)
	if !record.StartTime.IsZero() {
		result += fmt.Sprintf("Start Time: %s\n", record.StartTime.Format("2006-01-02 15:04:05 MST"))
	}
	if !record.EndTime.IsZero() {
		result += fmt.Sprintf("End Time: %s\n", record.EndTime.Format("2006-01-02 15:04:05 MST"))
	}
	result += fmt.Sprintf("Recordings: %d\n", record.RecordingCount)
	result += fmt.Sprintf("Transcripts: %d\n", record.TranscriptCount)

	return mcp.NewToolResultText(result), nil
}

func handleListRecordings(ctx context.Context, request mcp.CallToolRequest, sc *server.ServerContext) (*mcp.CallToolResult, error) {
	args := request.GetArguments()
	account := getAccountFromArgs(args)

	conferenceRecord, ok := args["conference_record"].(string)
	if !ok || conferenceRecord == "" {
		return mcp.NewToolResultError("conference_record is required"), nil
	}

	client, err := getMeetClient(ctx, account, sc)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	recordings, err := client.ListRecordings(conferenceRecord)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to list recordings: %v", err)), nil
	}

	if len(recordings) == 0 {
		return mcp.NewToolResultText("No recordings found for this conference"), nil
	}

	result := fmt.Sprintf("Found %d recording(s):\n\n", len(recordings))
	for i, rec := range recordings {
		result += fmt.Sprintf("%d. %s\n", i+1, rec.Name)
		result += fmt.Sprintf("   State: %s\n", rec.State)
		if !rec.StartTime.IsZero() {
			result += fmt.Sprintf("   Start: %s\n", rec.StartTime.Format("2006-01-02 15:04:05 MST"))
		}
		if !rec.EndTime.IsZero() {
			result += fmt.Sprintf("   End: %s\n", rec.EndTime.Format("2006-01-02 15:04:05 MST"))
		}
		if rec.DriveDestination != nil {
			result += fmt.Sprintf("   Drive File: %s\n", rec.DriveDestination.File)
			if rec.DriveDestination.ExportURI != "" {
				result += fmt.Sprintf("   Download: %s\n", rec.DriveDestination.ExportURI)
			}
		}
		result += "\n"
	}

	return mcp.NewToolResultText(result), nil
}

func handleGetRecording(ctx context.Context, request mcp.CallToolRequest, sc *server.ServerContext) (*mcp.CallToolResult, error) {
	args := request.GetArguments()
	account := getAccountFromArgs(args)

	recordingName, ok := args["recording_name"].(string)
	if !ok || recordingName == "" {
		return mcp.NewToolResultError("recording_name is required"), nil
	}

	client, err := getMeetClient(ctx, account, sc)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	recording, err := client.GetRecording(recordingName)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to get recording: %v", err)), nil
	}

	result := fmt.Sprintf("Recording: %s\n", recording.Name)
	result += fmt.Sprintf("State: %s\n", recording.State)
	if !recording.StartTime.IsZero() {
		result += fmt.Sprintf("Start Time: %s\n", recording.StartTime.Format("2006-01-02 15:04:05 MST"))
	}
	if !recording.EndTime.IsZero() {
		result += fmt.Sprintf("End Time: %s\n", recording.EndTime.Format("2006-01-02 15:04:05 MST"))
	}
	if recording.DriveDestination != nil {
		result += fmt.Sprintf("\nDrive File: %s\n", recording.DriveDestination.File)
		if recording.DriveDestination.ExportURI != "" {
			result += fmt.Sprintf("Download URL: %s\n", recording.DriveDestination.ExportURI)
		}
	}

	return mcp.NewToolResultText(result), nil
}

func handleListTranscripts(ctx context.Context, request mcp.CallToolRequest, sc *server.ServerContext) (*mcp.CallToolResult, error) {
	args := request.GetArguments()
	account := getAccountFromArgs(args)

	conferenceRecord, ok := args["conference_record"].(string)
	if !ok || conferenceRecord == "" {
		return mcp.NewToolResultError("conference_record is required"), nil
	}

	client, err := getMeetClient(ctx, account, sc)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	transcripts, err := client.ListTranscripts(conferenceRecord)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to list transcripts: %v", err)), nil
	}

	if len(transcripts) == 0 {
		return mcp.NewToolResultText("No transcripts found for this conference"), nil
	}

	result := fmt.Sprintf("Found %d transcript(s):\n\n", len(transcripts))
	for i, trans := range transcripts {
		result += fmt.Sprintf("%d. %s\n", i+1, trans.Name)
		result += fmt.Sprintf("   State: %s\n", trans.State)
		result += fmt.Sprintf("   Language: %s\n", trans.Language)
		if !trans.StartTime.IsZero() {
			result += fmt.Sprintf("   Start: %s\n", trans.StartTime.Format("2006-01-02 15:04:05 MST"))
		}
		if !trans.EndTime.IsZero() {
			result += fmt.Sprintf("   End: %s\n", trans.EndTime.Format("2006-01-02 15:04:05 MST"))
		}
		if trans.DriveDestination != nil {
			result += fmt.Sprintf("   Docs File: %s\n", trans.DriveDestination.File)
		}
		result += "\n"
	}

	return mcp.NewToolResultText(result), nil
}

func handleGetTranscript(ctx context.Context, request mcp.CallToolRequest, sc *server.ServerContext) (*mcp.CallToolResult, error) {
	args := request.GetArguments()
	account := getAccountFromArgs(args)

	transcriptName, ok := args["transcript_name"].(string)
	if !ok || transcriptName == "" {
		return mcp.NewToolResultError("transcript_name is required"), nil
	}

	client, err := getMeetClient(ctx, account, sc)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	transcript, err := client.GetTranscript(transcriptName)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to get transcript: %v", err)), nil
	}

	result := fmt.Sprintf("Transcript: %s\n", transcript.Name)
	result += fmt.Sprintf("State: %s\n", transcript.State)
	result += fmt.Sprintf("Language: %s\n", transcript.Language)
	if !transcript.StartTime.IsZero() {
		result += fmt.Sprintf("Start Time: %s\n", transcript.StartTime.Format("2006-01-02 15:04:05 MST"))
	}
	if !transcript.EndTime.IsZero() {
		result += fmt.Sprintf("End Time: %s\n", transcript.EndTime.Format("2006-01-02 15:04:05 MST"))
	}
	if transcript.DriveDestination != nil {
		result += fmt.Sprintf("\nDocs File: %s\n", transcript.DriveDestination.File)
	}

	return mcp.NewToolResultText(result), nil
}

func handleGetTranscriptText(ctx context.Context, request mcp.CallToolRequest, sc *server.ServerContext) (*mcp.CallToolResult, error) {
	args := request.GetArguments()
	account := getAccountFromArgs(args)

	transcriptName, ok := args["transcript_name"].(string)
	if !ok || transcriptName == "" {
		return mcp.NewToolResultError("transcript_name is required"), nil
	}

	client, err := getMeetClient(ctx, account, sc)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	entries, err := client.GetTranscriptEntries(transcriptName)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to get transcript entries: %v", err)), nil
	}

	if len(entries) == 0 {
		return mcp.NewToolResultText("No transcript entries found"), nil
	}

	var result strings.Builder
	result.WriteString(fmt.Sprintf("Transcript (%d entries):\n\n", len(entries)))

	for _, entry := range entries {
		timestamp := ""
		if !entry.StartTime.IsZero() {
			timestamp = entry.StartTime.Format("15:04:05")
		}

		// Extract participant name (remove "users/" prefix if present)
		participantName := entry.Participant
		if strings.HasPrefix(participantName, "users/") {
			participantName = strings.TrimPrefix(participantName, "users/")
		}

		result.WriteString(fmt.Sprintf("[%s] %s: %s\n", timestamp, participantName, entry.Text))
	}

	return mcp.NewToolResultText(result.String()), nil
}

func handleCreateSpace(ctx context.Context, request mcp.CallToolRequest, sc *server.ServerContext) (*mcp.CallToolResult, error) {
	args := request.GetArguments()
	account := getAccountFromArgs(args)

	client, err := getMeetClient(ctx, account, sc)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	// Build the space input
	input := meet.SpaceInput{
		Config: &meet.SpaceConfigInput{},
	}

	// Set access type
	if accessType, ok := args["access_type"].(string); ok && accessType != "" {
		input.Config.AccessType = accessType
	}

	// Set artifact configuration
	input.Config.ArtifactConfig = &meet.ArtifactConfigInput{}

	if enableRecording, ok := args["enable_recording"].(bool); ok {
		input.Config.ArtifactConfig.EnableRecording = enableRecording
	}

	if enableTranscription, ok := args["enable_transcription"].(bool); ok {
		input.Config.ArtifactConfig.EnableTranscription = enableTranscription
	}

	if enableSmartNotes, ok := args["enable_smart_notes"].(bool); ok {
		input.Config.ArtifactConfig.EnableSmartNotes = enableSmartNotes
	}

	space, err := client.CreateSpace(input)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to create space: %v", err)), nil
	}

	result := fmt.Sprintf("Successfully created Google Meet space: %s\n", space.Name)
	result += fmt.Sprintf("Meeting URI: %s\n", space.MeetingURI)
	result += fmt.Sprintf("Meeting Code: %s\n", space.MeetingCode)

	if space.Config != nil {
		result += "\nConfiguration:\n"
		result += fmt.Sprintf("  Access Type: %s\n", space.Config.AccessType)

		if space.Config.ArtifactConfig != nil {
			result += "\nAuto-Artifacts:\n"
			if space.Config.ArtifactConfig.RecordingConfig != nil {
				result += fmt.Sprintf("  Recording: %v\n", space.Config.ArtifactConfig.RecordingConfig.Enabled)
			}
			if space.Config.ArtifactConfig.TranscriptionConfig != nil {
				result += fmt.Sprintf("  Transcription: %v\n", space.Config.ArtifactConfig.TranscriptionConfig.Enabled)
			}
			if space.Config.ArtifactConfig.SmartNotesConfig != nil {
				result += fmt.Sprintf("  Smart Notes (Gemini): %v\n", space.Config.ArtifactConfig.SmartNotesConfig.Enabled)
			}
		}
	}

	return mcp.NewToolResultText(result), nil
}

func handleGetSpace(ctx context.Context, request mcp.CallToolRequest, sc *server.ServerContext) (*mcp.CallToolResult, error) {
	args := request.GetArguments()
	account := getAccountFromArgs(args)

	spaceName, ok := args["space_name"].(string)
	if !ok || spaceName == "" {
		return mcp.NewToolResultError("space_name is required"), nil
	}

	client, err := getMeetClient(ctx, account, sc)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	space, err := client.GetSpace(spaceName)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to get space: %v", err)), nil
	}

	result := fmt.Sprintf("Google Meet Space: %s\n", space.Name)
	result += fmt.Sprintf("Meeting URI: %s\n", space.MeetingURI)
	result += fmt.Sprintf("Meeting Code: %s\n", space.MeetingCode)

	if space.ActiveConference != "" {
		result += fmt.Sprintf("Active Conference: %s\n", space.ActiveConference)
	}

	if space.Config != nil {
		result += "\nConfiguration:\n"
		result += fmt.Sprintf("  Access Type: %s\n", space.Config.AccessType)
		result += fmt.Sprintf("  Entry Point Access: %s\n", space.Config.EntryPointAccess)

		if space.Config.ArtifactConfig != nil {
			result += "\nAuto-Artifacts:\n"
			if space.Config.ArtifactConfig.RecordingConfig != nil {
				result += fmt.Sprintf("  Recording: %v\n", space.Config.ArtifactConfig.RecordingConfig.Enabled)
			}
			if space.Config.ArtifactConfig.TranscriptionConfig != nil {
				result += fmt.Sprintf("  Transcription: %v\n", space.Config.ArtifactConfig.TranscriptionConfig.Enabled)
			}
			if space.Config.ArtifactConfig.SmartNotesConfig != nil {
				result += fmt.Sprintf("  Smart Notes (Gemini): %v\n", space.Config.ArtifactConfig.SmartNotesConfig.Enabled)
			}
		}
	}

	return mcp.NewToolResultText(result), nil
}

func handleUpdateSpaceConfig(ctx context.Context, request mcp.CallToolRequest, sc *server.ServerContext) (*mcp.CallToolResult, error) {
	args := request.GetArguments()
	account := getAccountFromArgs(args)

	spaceName, ok := args["space_name"].(string)
	if !ok || spaceName == "" {
		return mcp.NewToolResultError("space_name is required"), nil
	}

	client, err := getMeetClient(ctx, account, sc)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	// Build the configuration input
	input := meet.SpaceConfigInput{
		ArtifactConfig: &meet.ArtifactConfigInput{},
	}

	// Set access type
	if accessType, ok := args["access_type"].(string); ok && accessType != "" {
		input.AccessType = accessType
	}

	// Set artifact configuration
	if enableRecording, ok := args["enable_recording"].(bool); ok {
		input.ArtifactConfig.EnableRecording = enableRecording
	}

	if enableTranscription, ok := args["enable_transcription"].(bool); ok {
		input.ArtifactConfig.EnableTranscription = enableTranscription
	}

	if enableSmartNotes, ok := args["enable_smart_notes"].(bool); ok {
		input.ArtifactConfig.EnableSmartNotes = enableSmartNotes
	}

	space, err := client.UpdateSpaceConfig(spaceName, input)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to update space config: %v", err)), nil
	}

	result := fmt.Sprintf("Successfully updated space configuration: %s\n", space.Name)

	if space.Config != nil {
		result += "\nUpdated Configuration:\n"
		result += fmt.Sprintf("  Access Type: %s\n", space.Config.AccessType)

		if space.Config.ArtifactConfig != nil {
			result += "\nAuto-Artifacts:\n"
			if space.Config.ArtifactConfig.RecordingConfig != nil {
				result += fmt.Sprintf("  Recording: %v\n", space.Config.ArtifactConfig.RecordingConfig.Enabled)
			}
			if space.Config.ArtifactConfig.TranscriptionConfig != nil {
				result += fmt.Sprintf("  Transcription: %v\n", space.Config.ArtifactConfig.TranscriptionConfig.Enabled)
			}
			if space.Config.ArtifactConfig.SmartNotesConfig != nil {
				result += fmt.Sprintf("  Smart Notes (Gemini): %v\n", space.Config.ArtifactConfig.SmartNotesConfig.Enabled)
			}
		}
	}

	return mcp.NewToolResultText(result), nil
}
