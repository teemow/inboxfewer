// Package meet provides a client for the Google Meet API v2.
//
// This package enables creation and configuration of Google Meet spaces with automatic
// recording, transcription, and Gemini note-taking, as well as retrieval of meeting
// artifacts from completed sessions. It follows the same account-based authentication
// pattern as the Gmail and Calendar clients.
//
// Space Configuration: As of April 2025, Google Meet API v2 supports programmatic
// configuration of meeting spaces, allowing you to enable automatic recording,
// transcription, and Gemini note-taking when creating or updating spaces.
//
// Key features:
//   - Create new Meet spaces with optional auto-recording, transcription, and note-taking
//   - Configure existing spaces to enable/disable automatic artifacts
//   - Retrieve conference record metadata
//   - List and access meeting recordings
//   - List and access meeting transcripts
//   - Multi-account support for managing multiple Google Workspace accounts
//
// Authentication:
// The client uses OAuth2 with the meetings.space.readonly and meetings.space.settings
// scopes. Users must authenticate via the google_get_auth_url and google_save_auth_code
// tools before using the Meet API.
//
// Example usage:
//
//	ctx := context.Background()
//	client, err := meet.NewClient(ctx)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	// Create a space with auto-recording enabled
//	space, err := client.CreateSpace(meet.SpaceInput{
//	    Config: &meet.SpaceConfigInput{
//	        ArtifactConfig: &meet.ArtifactConfigInput{
//	            EnableRecording: true,
//	            EnableTranscription: true,
//	        },
//	    },
//	})
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	// List recordings for a conference
//	recordings, err := client.ListRecordings("spaces/SPACE_ID/conferences/CONFERENCE_ID")
//	if err != nil {
//	    log.Fatal(err)
//	}
package meet
