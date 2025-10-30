// Package meet_tools provides MCP tools for interacting with the Google Meet API v2.
//
// This package registers tools that allow MCP clients to create and configure Google Meet
// spaces with automatic recording, transcription, and Gemini note-taking, as well as
// retrieve meeting artifacts from completed sessions.
//
// Available tools:
//
// Space Management (Write):
//   - meet_create_space - Create a new Meet space with optional auto-recording/transcription/notes
//   - meet_get_space - Get details about a space including its configuration
//   - meet_update_space_config - Update space settings (enable/disable auto-artifacts)
//
// Meeting Artifacts (Read):
//   - meet_get_conference - Retrieve conference record metadata
//   - meet_list_recordings - List all recordings for a meeting
//   - meet_get_recording - Get details about a specific recording
//   - meet_list_transcripts - List all transcripts for a meeting
//   - meet_get_transcript - Get details about a specific transcript
//   - meet_get_transcript_text - Get the full text of a transcript
//
// All tools support multi-account authentication via the "account" parameter.
//
// Example usage:
//
//	# Create a space with auto-recording enabled
//	meet_create_space(
//	    account="work",
//	    enable_recording=true,
//	    enable_transcription=true
//	)
//
//	# List recordings for a conference
//	meet_list_recordings(
//	    account="work",
//	    conference_record="spaces/SPACE_ID/conferenceRecords/CONF_ID"
//	)
//
//	# Get transcript text
//	meet_get_transcript_text(
//	    account="default",
//	    transcript_name="spaces/SPACE_ID/conferenceRecords/CONF_ID/transcripts/TRANS_ID"
//	)
package meet_tools
