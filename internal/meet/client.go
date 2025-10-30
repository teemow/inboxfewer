package meet

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"time"

	meet "google.golang.org/api/meet/v2"
	"google.golang.org/api/option"

	"github.com/teemow/inboxfewer/internal/google"
)

// Client wraps the Google Meet service
type Client struct {
	svc     *meet.Service
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

// NewClientForAccount creates a new Meet client with OAuth2 authentication for a specific account
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

	svc, err := meet.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return nil, fmt.Errorf("failed to create Meet service: %w", err)
	}

	return &Client{
		svc:     svc,
		account: account,
	}, nil
}

// NewClient creates a new Meet client with OAuth2 authentication for the default account
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

// GetConferenceRecord retrieves a conference record by name
func (c *Client) GetConferenceRecord(name string) (*ConferenceRecordSummary, error) {
	record, err := c.svc.ConferenceRecords.Get(name).Do()
	if err != nil {
		return nil, fmt.Errorf("failed to get conference record: %w", err)
	}

	summary := &ConferenceRecordSummary{
		Name:        record.Name,
		SpaceID:     record.Space,
		MeetingCode: extractMeetingCode(record.Name),
	}

	// Parse timestamps
	if record.StartTime != "" {
		if t, err := time.Parse(time.RFC3339, record.StartTime); err == nil {
			summary.StartTime = t
		}
	}
	if record.EndTime != "" {
		if t, err := time.Parse(time.RFC3339, record.EndTime); err == nil {
			summary.EndTime = t
		}
	}

	return summary, nil
}

// ListRecordings lists all recordings for a conference record
func (c *Client) ListRecordings(conferenceRecordName string) ([]Recording, error) {
	var recordings []Recording

	call := c.svc.ConferenceRecords.Recordings.List(conferenceRecordName)

	err := call.Pages(context.Background(), func(resp *meet.ListRecordingsResponse) error {
		for _, rec := range resp.Recordings {
			recording := Recording{
				Name:  rec.Name,
				State: rec.State,
			}

			// Parse timestamps
			if rec.StartTime != "" {
				if t, err := time.Parse(time.RFC3339, rec.StartTime); err == nil {
					recording.StartTime = t
				}
			}
			if rec.EndTime != "" {
				if t, err := time.Parse(time.RFC3339, rec.EndTime); err == nil {
					recording.EndTime = t
				}
			}

			// Parse drive destination
			if rec.DriveDestination != nil {
				recording.DriveDestination = &DriveDestination{
					File:      rec.DriveDestination.File,
					ExportURI: rec.DriveDestination.ExportUri,
				}
			}

			recordings = append(recordings, recording)
		}
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to list recordings: %w", err)
	}

	return recordings, nil
}

// GetRecording retrieves a specific recording by name
func (c *Client) GetRecording(name string) (*Recording, error) {
	rec, err := c.svc.ConferenceRecords.Recordings.Get(name).Do()
	if err != nil {
		return nil, fmt.Errorf("failed to get recording: %w", err)
	}

	recording := &Recording{
		Name:  rec.Name,
		State: rec.State,
	}

	// Parse timestamps
	if rec.StartTime != "" {
		if t, err := time.Parse(time.RFC3339, rec.StartTime); err == nil {
			recording.StartTime = t
		}
	}
	if rec.EndTime != "" {
		if t, err := time.Parse(time.RFC3339, rec.EndTime); err == nil {
			recording.EndTime = t
		}
	}

	// Parse drive destination
	if rec.DriveDestination != nil {
		recording.DriveDestination = &DriveDestination{
			File:      rec.DriveDestination.File,
			ExportURI: rec.DriveDestination.ExportUri,
		}
	}

	return recording, nil
}

// ListTranscripts lists all transcripts for a conference record
func (c *Client) ListTranscripts(conferenceRecordName string) ([]Transcript, error) {
	var transcripts []Transcript

	call := c.svc.ConferenceRecords.Transcripts.List(conferenceRecordName)

	err := call.Pages(context.Background(), func(resp *meet.ListTranscriptsResponse) error {
		for _, trans := range resp.Transcripts {
			transcript := Transcript{
				Name:  trans.Name,
				State: trans.State,
				// Language field not available in Transcript type
			}

			// Parse timestamps
			if trans.StartTime != "" {
				if t, err := time.Parse(time.RFC3339, trans.StartTime); err == nil {
					transcript.StartTime = t
				}
			}
			if trans.EndTime != "" {
				if t, err := time.Parse(time.RFC3339, trans.EndTime); err == nil {
					transcript.EndTime = t
				}
			}

			// Parse drive destination
			if trans.DocsDestination != nil {
				transcript.DriveDestination = &DriveDestination{
					File:      trans.DocsDestination.Document,
					ExportURI: "", // Not provided for transcripts
				}
			}

			transcripts = append(transcripts, transcript)
		}
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to list transcripts: %w", err)
	}

	return transcripts, nil
}

// GetTranscript retrieves a specific transcript by name
func (c *Client) GetTranscript(name string) (*Transcript, error) {
	trans, err := c.svc.ConferenceRecords.Transcripts.Get(name).Do()
	if err != nil {
		return nil, fmt.Errorf("failed to get transcript: %w", err)
	}

	transcript := &Transcript{
		Name:  trans.Name,
		State: trans.State,
		// Language field not available in Transcript type
	}

	// Parse timestamps
	if trans.StartTime != "" {
		if t, err := time.Parse(time.RFC3339, trans.StartTime); err == nil {
			transcript.StartTime = t
		}
	}
	if trans.EndTime != "" {
		if t, err := time.Parse(time.RFC3339, trans.EndTime); err == nil {
			transcript.EndTime = t
		}
	}

	// Parse drive destination
	if trans.DocsDestination != nil {
		transcript.DriveDestination = &DriveDestination{
			File:      trans.DocsDestination.Document,
			ExportURI: "", // Not provided for transcripts
		}
	}

	return transcript, nil
}

// GetTranscriptEntries retrieves all entries from a transcript
func (c *Client) GetTranscriptEntries(transcriptName string) ([]TranscriptEntry, error) {
	var entries []TranscriptEntry

	call := c.svc.ConferenceRecords.Transcripts.Entries.List(transcriptName)

	err := call.Pages(context.Background(), func(resp *meet.ListTranscriptEntriesResponse) error {
		for _, entry := range resp.TranscriptEntries {
			transcriptEntry := TranscriptEntry{
				Name:        entry.Name,
				Participant: entry.Participant,
				Text:        entry.Text,
				Language:    entry.LanguageCode,
			}

			// Parse timestamps
			if entry.StartTime != "" {
				if t, err := time.Parse(time.RFC3339, entry.StartTime); err == nil {
					transcriptEntry.StartTime = t
				}
			}
			if entry.EndTime != "" {
				if t, err := time.Parse(time.RFC3339, entry.EndTime); err == nil {
					transcriptEntry.EndTime = t
				}
			}

			entries = append(entries, transcriptEntry)
		}
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to get transcript entries: %w", err)
	}

	return entries, nil
}

// CreateSpace creates a new Google Meet space with optional configuration
func (c *Client) CreateSpace(input SpaceInput) (*Space, error) {
	space := &meet.Space{}

	// Build the space configuration
	if input.Config != nil {
		spaceConfig := &meet.SpaceConfig{}

		// Set access type
		if input.Config.AccessType != "" {
			spaceConfig.AccessType = input.Config.AccessType
		}

		// Set entry point access
		if input.Config.EntryPointAccess != "" {
			spaceConfig.EntryPointAccess = input.Config.EntryPointAccess
		}

		// Set artifact configuration
		if input.Config.ArtifactConfig != nil {
			artifactConfig := &meet.ArtifactConfig{}

			if input.Config.ArtifactConfig.EnableRecording {
				artifactConfig.RecordingConfig = &meet.RecordingConfig{
					AutoRecordingGeneration: "ON",
				}
			}

			if input.Config.ArtifactConfig.EnableTranscription {
				artifactConfig.TranscriptionConfig = &meet.TranscriptionConfig{
					AutoTranscriptionGeneration: "ON",
				}
			}

			if input.Config.ArtifactConfig.EnableSmartNotes {
				artifactConfig.SmartNotesConfig = &meet.SmartNotesConfig{
					AutoSmartNotesGeneration: "ON",
				}
			}

			spaceConfig.ArtifactConfig = artifactConfig
		}

		space.Config = spaceConfig
	}

	created, err := c.svc.Spaces.Create(space).Do()
	if err != nil {
		return nil, fmt.Errorf("failed to create space: %w", err)
	}

	return toSpace(created), nil
}

// GetSpace retrieves a Google Meet space by name
func (c *Client) GetSpace(name string) (*Space, error) {
	space, err := c.svc.Spaces.Get(name).Do()
	if err != nil {
		return nil, fmt.Errorf("failed to get space: %w", err)
	}

	return toSpace(space), nil
}

// UpdateSpaceConfig updates the configuration of an existing Google Meet space
func (c *Client) UpdateSpaceConfig(spaceName string, input SpaceConfigInput) (*Space, error) {
	// First get the existing space
	existing, err := c.svc.Spaces.Get(spaceName).Do()
	if err != nil {
		return nil, fmt.Errorf("failed to get existing space: %w", err)
	}

	// Update the configuration
	if existing.Config == nil {
		existing.Config = &meet.SpaceConfig{}
	}

	// Update access type
	if input.AccessType != "" {
		existing.Config.AccessType = input.AccessType
	}

	// Update entry point access
	if input.EntryPointAccess != "" {
		existing.Config.EntryPointAccess = input.EntryPointAccess
	}

	// Update artifact configuration
	if input.ArtifactConfig != nil {
		if existing.Config.ArtifactConfig == nil {
			existing.Config.ArtifactConfig = &meet.ArtifactConfig{}
		}

		// Update recording config
		if input.ArtifactConfig.EnableRecording {
			existing.Config.ArtifactConfig.RecordingConfig = &meet.RecordingConfig{
				AutoRecordingGeneration: "ON",
			}
		} else {
			existing.Config.ArtifactConfig.RecordingConfig = &meet.RecordingConfig{
				AutoRecordingGeneration: "OFF",
			}
		}

		// Update transcription config
		if input.ArtifactConfig.EnableTranscription {
			existing.Config.ArtifactConfig.TranscriptionConfig = &meet.TranscriptionConfig{
				AutoTranscriptionGeneration: "ON",
			}
		} else {
			existing.Config.ArtifactConfig.TranscriptionConfig = &meet.TranscriptionConfig{
				AutoTranscriptionGeneration: "OFF",
			}
		}

		// Update smart notes config
		if input.ArtifactConfig.EnableSmartNotes {
			existing.Config.ArtifactConfig.SmartNotesConfig = &meet.SmartNotesConfig{
				AutoSmartNotesGeneration: "ON",
			}
		} else {
			existing.Config.ArtifactConfig.SmartNotesConfig = &meet.SmartNotesConfig{
				AutoSmartNotesGeneration: "OFF",
			}
		}
	}

	// Update the space with the configuration
	updated, err := c.svc.Spaces.Patch(spaceName, existing).UpdateMask("config").Do()
	if err != nil {
		return nil, fmt.Errorf("failed to update space config: %w", err)
	}

	return toSpace(updated), nil
}

// toSpace converts a Meet API Space to our Space type
func toSpace(s *meet.Space) *Space {
	space := &Space{
		Name:        s.Name,
		MeetingURI:  s.MeetingUri,
		MeetingCode: s.MeetingCode,
	}

	// Extract ActiveConference record name if present
	if s.ActiveConference != nil {
		space.ActiveConference = s.ActiveConference.ConferenceRecord
	}

	if s.Config != nil {
		config := &SpaceConfig{
			AccessType:       s.Config.AccessType,
			EntryPointAccess: s.Config.EntryPointAccess,
		}

		if s.Config.ArtifactConfig != nil {
			artifactConfig := &ArtifactConfig{}

			if s.Config.ArtifactConfig.RecordingConfig != nil {
				artifactConfig.RecordingConfig = &ArtifactGenerationConfig{
					Enabled: s.Config.ArtifactConfig.RecordingConfig.AutoRecordingGeneration == "ON",
				}
			}

			if s.Config.ArtifactConfig.TranscriptionConfig != nil {
				artifactConfig.TranscriptionConfig = &ArtifactGenerationConfig{
					Enabled: s.Config.ArtifactConfig.TranscriptionConfig.AutoTranscriptionGeneration == "ON",
				}
			}

			if s.Config.ArtifactConfig.SmartNotesConfig != nil {
				artifactConfig.SmartNotesConfig = &ArtifactGenerationConfig{
					Enabled: s.Config.ArtifactConfig.SmartNotesConfig.AutoSmartNotesGeneration == "ON",
				}
			}

			config.ArtifactConfig = artifactConfig
		}

		space.Config = config
	}

	return space
}

// extractMeetingCode extracts the meeting code from a conference record name
// e.g., "spaces/SPACE_ID/conferenceRecords/CONF_ID" -> "CONF_ID"
func extractMeetingCode(name string) string {
	// This is a simplified extraction; actual meeting codes may differ
	// The API doesn't directly provide the meeting code, so this is a best-effort extraction
	return name
}
