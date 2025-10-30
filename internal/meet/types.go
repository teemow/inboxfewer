package meet

import "time"

// ConferenceRecordSummary represents a summary of a conference record
type ConferenceRecordSummary struct {
	// Name is the resource name of the conference record
	// Format: spaces/{space}/conferenceRecords/{conferenceRecord}
	Name string

	// StartTime is when the conference started
	StartTime time.Time

	// EndTime is when the conference ended
	EndTime time.Time

	// SpaceID is the ID of the Meet space
	SpaceID string

	// MeetingCode is the meeting code (e.g., "abc-defg-hij")
	MeetingCode string

	// RecordingCount is the number of recordings available
	RecordingCount int

	// TranscriptCount is the number of transcripts available
	TranscriptCount int
}

// Recording represents a Google Meet recording
type Recording struct {
	// Name is the resource name of the recording
	// Format: spaces/{space}/conferenceRecords/{conferenceRecord}/recordings/{recording}
	Name string

	// State is the current state of the recording (e.g., "FILE_GENERATED")
	State string

	// StartTime is when the recording started
	StartTime time.Time

	// EndTime is when the recording ended
	EndTime time.Time

	// DriveDestination contains Google Drive file information
	DriveDestination *DriveDestination
}

// DriveDestination contains information about a file stored in Google Drive
type DriveDestination struct {
	// File is the resource name of the Drive file
	File string

	// ExportURI is the URI to download/export the file
	ExportURI string
}

// Transcript represents a Google Meet transcript
type Transcript struct {
	// Name is the resource name of the transcript
	// Format: spaces/{space}/conferenceRecords/{conferenceRecord}/transcripts/{transcript}
	Name string

	// State is the current state of the transcript (e.g., "FILE_GENERATED")
	State string

	// StartTime is when transcription started
	StartTime time.Time

	// EndTime is when transcription ended
	EndTime time.Time

	// Language is the language of the transcript (BCP 47 code, e.g., "en-US")
	Language string

	// DriveDestination contains Google Drive file information
	DriveDestination *DriveDestination
}

// TranscriptEntry represents a single entry in a transcript
type TranscriptEntry struct {
	// Name is the resource name of the transcript entry
	// Format: spaces/{space}/conferenceRecords/{conferenceRecord}/transcripts/{transcript}/entries/{entry}
	Name string

	// Participant is the name of the participant who spoke
	Participant string

	// Text is the transcribed text
	Text string

	// Language is the language of this entry (BCP 47 code)
	Language string

	// StartTime is when the participant started speaking
	StartTime time.Time

	// EndTime is when the participant finished speaking
	EndTime time.Time
}

// Space represents a Google Meet space
type Space struct {
	// Name is the resource name of the space
	// Format: spaces/{space}
	Name string

	// MeetingURI is the URI to join the meeting
	MeetingURI string

	// MeetingCode is the meeting code (e.g., "abc-defg-hij")
	MeetingCode string

	// Config is the configuration for the space
	Config *SpaceConfig

	// ActiveConference is the resource name of the active conference in this space
	ActiveConference string
}

// SpaceConfig represents the configuration for a Google Meet space
type SpaceConfig struct {
	// AccessType defines who can join without knocking
	AccessType string

	// EntryPointAccess defines which entry points can be used
	EntryPointAccess string

	// ArtifactConfig contains auto artifact generation settings
	ArtifactConfig *ArtifactConfig
}

// ArtifactConfig contains settings for automatic artifact generation
type ArtifactConfig struct {
	// RecordingConfig controls automatic recording
	RecordingConfig *ArtifactGenerationConfig

	// TranscriptionConfig controls automatic transcription
	TranscriptionConfig *ArtifactGenerationConfig

	// SmartNotesConfig controls automatic note-taking (Gemini)
	SmartNotesConfig *ArtifactGenerationConfig
}

// ArtifactGenerationConfig controls whether an artifact type is auto-generated
type ArtifactGenerationConfig struct {
	// Enabled indicates whether this artifact should be auto-generated
	Enabled bool
}

// SpaceInput represents input for creating or updating a space
type SpaceInput struct {
	// Config is the configuration for the space
	Config *SpaceConfigInput
}

// SpaceConfigInput represents input for space configuration
type SpaceConfigInput struct {
	// AccessType defines who can join without knocking
	// Values: "OPEN", "TRUSTED", "RESTRICTED"
	AccessType string

	// EntryPointAccess defines which entry points can be used
	// Values: "ALL", "CREATOR_APP_ONLY"
	EntryPointAccess string

	// ArtifactConfig contains auto artifact generation settings
	ArtifactConfig *ArtifactConfigInput
}

// ArtifactConfigInput represents input for artifact configuration
type ArtifactConfigInput struct {
	// EnableRecording enables automatic recording
	EnableRecording bool

	// EnableTranscription enables automatic transcription
	EnableTranscription bool

	// EnableSmartNotes enables automatic note-taking (Gemini)
	EnableSmartNotes bool
}
