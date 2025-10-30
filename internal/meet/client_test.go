package meet

import (
	"testing"
	"time"
)

func TestConferenceRecordSummary(t *testing.T) {
	now := time.Now()
	summary := ConferenceRecordSummary{
		Name:            "spaces/test/conferenceRecords/conf123",
		StartTime:       now,
		EndTime:         now.Add(1 * time.Hour),
		SpaceID:         "spaces/test",
		MeetingCode:     "abc-defg-hij",
		RecordingCount:  2,
		TranscriptCount: 1,
	}

	if summary.Name == "" {
		t.Error("Name should not be empty")
	}
	if summary.StartTime.IsZero() {
		t.Error("StartTime should not be zero")
	}
	if summary.RecordingCount != 2 {
		t.Errorf("Expected 2 recordings, got %d", summary.RecordingCount)
	}
}

func TestRecording(t *testing.T) {
	now := time.Now()
	recording := Recording{
		Name:      "spaces/test/conferenceRecords/conf123/recordings/rec456",
		State:     "FILE_GENERATED",
		StartTime: now,
		EndTime:   now.Add(1 * time.Hour),
		DriveDestination: &DriveDestination{
			File:      "files/file123",
			ExportURI: "https://drive.google.com/export/file123",
		},
	}

	if recording.State != "FILE_GENERATED" {
		t.Errorf("Expected state FILE_GENERATED, got %s", recording.State)
	}
	if recording.DriveDestination == nil {
		t.Error("DriveDestination should not be nil")
	}
	if recording.DriveDestination.File != "files/file123" {
		t.Errorf("Expected file files/file123, got %s", recording.DriveDestination.File)
	}
}

func TestTranscript(t *testing.T) {
	now := time.Now()
	transcript := Transcript{
		Name:      "spaces/test/conferenceRecords/conf123/transcripts/trans789",
		State:     "FILE_GENERATED",
		StartTime: now,
		EndTime:   now.Add(1 * time.Hour),
		Language:  "en-US",
		DriveDestination: &DriveDestination{
			File: "docs/doc123",
		},
	}

	if transcript.Language != "en-US" {
		t.Errorf("Expected language en-US, got %s", transcript.Language)
	}
	if transcript.DriveDestination == nil {
		t.Error("DriveDestination should not be nil")
	}
}

func TestTranscriptEntry(t *testing.T) {
	now := time.Now()
	entry := TranscriptEntry{
		Name:        "spaces/test/conferenceRecords/conf123/transcripts/trans789/entries/entry1",
		Participant: "users/user123",
		Text:        "Hello, this is a test transcript.",
		Language:    "en-US",
		StartTime:   now,
		EndTime:     now.Add(5 * time.Second),
	}

	if entry.Text == "" {
		t.Error("Text should not be empty")
	}
	if entry.Participant == "" {
		t.Error("Participant should not be empty")
	}
	if entry.Language != "en-US" {
		t.Errorf("Expected language en-US, got %s", entry.Language)
	}
}

func TestExtractMeetingCode(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "conference record name",
			input:    "spaces/SPACE_ID/conferenceRecords/CONF_ID",
			expected: "spaces/SPACE_ID/conferenceRecords/CONF_ID",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractMeetingCode(tt.input)
			if result != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestSpace(t *testing.T) {
	space := Space{
		Name:             "spaces/test-space",
		MeetingURI:       "https://meet.google.com/abc-defg-hij",
		MeetingCode:      "abc-defg-hij",
		ActiveConference: "spaces/test-space/conferences/conf123",
		Config: &SpaceConfig{
			AccessType:       "OPEN",
			EntryPointAccess: "ALL",
			ArtifactConfig: &ArtifactConfig{
				RecordingConfig: &ArtifactGenerationConfig{
					Enabled: true,
				},
				TranscriptionConfig: &ArtifactGenerationConfig{
					Enabled: true,
				},
				SmartNotesConfig: &ArtifactGenerationConfig{
					Enabled: false,
				},
			},
		},
	}

	if space.Name != "spaces/test-space" {
		t.Errorf("Expected name spaces/test-space, got %s", space.Name)
	}

	if space.Config == nil {
		t.Fatal("Config should not be nil")
	}

	if space.Config.ArtifactConfig == nil {
		t.Fatal("ArtifactConfig should not be nil")
	}

	if !space.Config.ArtifactConfig.RecordingConfig.Enabled {
		t.Error("Recording should be enabled")
	}

	if !space.Config.ArtifactConfig.TranscriptionConfig.Enabled {
		t.Error("Transcription should be enabled")
	}

	if space.Config.ArtifactConfig.SmartNotesConfig.Enabled {
		t.Error("Smart notes should be disabled")
	}
}

func TestSpaceInput(t *testing.T) {
	input := SpaceInput{
		Config: &SpaceConfigInput{
			AccessType:       "TRUSTED",
			EntryPointAccess: "CREATOR_APP_ONLY",
			ArtifactConfig: &ArtifactConfigInput{
				EnableRecording:     true,
				EnableTranscription: false,
				EnableSmartNotes:    true,
			},
		},
	}

	if input.Config == nil {
		t.Fatal("Config should not be nil")
	}

	if input.Config.AccessType != "TRUSTED" {
		t.Errorf("Expected access type TRUSTED, got %s", input.Config.AccessType)
	}

	if input.Config.ArtifactConfig == nil {
		t.Fatal("ArtifactConfig should not be nil")
	}

	if !input.Config.ArtifactConfig.EnableRecording {
		t.Error("Recording should be enabled")
	}

	if input.Config.ArtifactConfig.EnableTranscription {
		t.Error("Transcription should be disabled")
	}

	if !input.Config.ArtifactConfig.EnableSmartNotes {
		t.Error("Smart notes should be enabled")
	}
}
