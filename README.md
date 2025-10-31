# inboxfewer

Archives Gmail threads for closed GitHub issues and pull requests.

## Features

- **Gmail Integration**: Automatically archives emails related to closed GitHub issues and PRs
- **Contact Search**: Search for contacts in Google Contacts by name, email, or phone number
- **Email Sending**: Send emails through Gmail API with support for CC, BCC, and HTML formatting
- **Google Docs Integration**: Extract and retrieve Google Docs content from email messages, with full support for multi-tab documents (Oct 2024 feature)
- **Google Calendar Integration**: Full calendar management including event creation, modification, availability checking, and meeting scheduling with Google Meet support
- **Google Meet Integration**: Retrieve meeting artifacts including recordings, transcripts, and Gemini notes from completed Google Meet sessions
- **MCP Server**: Provides Model Context Protocol server for AI assistant integration
- **Multiple Transports**: Supports stdio, SSE, and streamable HTTP transports
- **Flexible Usage**: Can run as a CLI tool or as an MCP server

## Installation

```bash
go install github.com/teemow/inboxfewer@latest
```

## Configuration

### GitHub Token

Create a file at `~/keys/github-inboxfewer.token` with two space-separated values:
```
<github-username> <github-personal-access-token>
```

### Google Services OAuth

On first run, you'll be prompted to authenticate with Google services (Gmail, Google Docs, Google Drive). OAuth tokens are cached per account at:
- Linux/Unix: `~/.cache/inboxfewer/google-{account}.token`
- macOS: `~/Library/Caches/inboxfewer/google-{account}.token`
- Windows: `%TEMP%/inboxfewer/google-{account}.token`

**Note:** Each OAuth token provides access to Gmail, Google Docs, Google Drive, Google Contacts, Google Calendar, and Google Meet APIs with the following scopes:
- Gmail: Read, modify, and send messages
- Google Docs: Read document content
- Google Drive: Read file metadata
- Google Contacts: Read contact information (personal contacts, interaction history, and directory)
- Google Calendar: Read and write calendar events, check availability, and manage calendars
- Google Meet: Read meeting artifacts (recordings, transcripts) and configure meeting spaces (enable/disable auto-recording, transcription, note-taking)

### Multi-Account Support

inboxfewer supports managing multiple Google accounts (e.g., work and personal). Each account is identified by a unique name and has its own OAuth token.

**Default Account:** If no account name is specified, the `default` account is used. Existing users' tokens will be automatically migrated to the `default` account on first run.

**Account Names:** Account names must contain only alphanumeric characters, hyphens, and underscores (e.g., `work`, `personal`, `work-email`).

## Usage

### CLI Mode (Cleanup)

Archive Gmail threads related to closed GitHub issues/PRs:

```bash
# Run cleanup with default account
inboxfewer

# Or explicitly
inboxfewer cleanup

# Use a specific account
inboxfewer cleanup --account work

# Use personal account
inboxfewer cleanup --account personal
```

### MCP Server Mode

Start the MCP server to provide Gmail/GitHub tools for AI assistants:

#### Standard I/O (Default)
```bash
inboxfewer serve
# or
inboxfewer serve --transport stdio
```

#### Server-Sent Events (SSE)
```bash
inboxfewer serve --transport sse --http-addr :8080
```

The SSE server will expose:
- SSE endpoint: `http://localhost:8080/sse`
- Message endpoint: `http://localhost:8080/message`

#### Streamable HTTP
```bash
inboxfewer serve --transport streamable-http --http-addr :8080
```

The HTTP server will expose:
- HTTP endpoint: `http://localhost:8080/mcp`

### Options

```bash
--debug           Enable debug logging
--transport       Transport type: stdio, sse, or streamable-http (default: stdio)
--http-addr       HTTP server address for sse/http transports (default: :8080)
```

## MCP Server Tools

When running as an MCP server, the following tools are available:

### OAuth Authentication Flow

Before using any Google services (Gmail, Docs, Drive), you need to authenticate each account:

1. **Check if authenticated:** The server will automatically check for an existing token for the specified account
2. **Get authorization URL:** If not authenticated, use `google_get_auth_url` (optionally with `account` parameter) to get the OAuth URL
3. **Authorize access:** Visit the URL in your browser and grant permissions
4. **Save the code:** Copy the authorization code and use `google_save_auth_code` (with matching `account` parameter) to save it
5. **Use the tools:** All Google-related tools will now work with the saved token for that account

Each token is stored in `~/.cache/inboxfewer/google-{account}.token` and provides access to all Google APIs (Gmail, Docs, Drive, Contacts).

### Multi-Account Support in MCP Tools

All Google-related MCP tools support an optional `account` parameter to specify which Google account to use:

- **Default behavior:** If `account` is not specified, the `default` account is used
- **Multiple accounts:** You can manage multiple Google accounts (e.g., `work`, `personal`, `company-email`)
- **Per-tool specification:** Each tool call can use a different account

**Example:**
```javascript
// Use default account
gmail_list_threads({query: "in:inbox"})

// Use work account
gmail_list_threads({account: "work", query: "in:inbox"})

// Use personal account
gmail_send_email({account: "personal", to: "friend@example.com", subject: "Hello", body: "Hi!"})
```

### Gmail Tools

**Note:** All Gmail tools support an optional `account` parameter to specify which Google account to use (default: 'default').

#### `gmail_list_threads`
List Gmail threads matching a query.

**Arguments:**
- `account` (optional): Account name (default: 'default')
- `query` (required): Gmail search query (e.g., 'in:inbox', 'from:user@example.com')
- `maxResults` (optional): Maximum number of results (default: 10)

#### `gmail_archive_thread`
Archive a Gmail thread by removing it from the inbox.

**Arguments:**
- `threadId` (required): The ID of the thread to archive

#### `gmail_classify_thread`
Classify a Gmail thread to determine if it's related to GitHub issues or PRs.

**Arguments:**
- `threadId` (required): The ID of the thread to classify

#### `gmail_check_stale`
Check if a Gmail thread is stale (GitHub issue/PR is closed).

**Arguments:**
- `threadId` (required): The ID of the thread to check

#### `gmail_archive_stale_threads`
Archive all Gmail threads in inbox that are related to closed GitHub issues/PRs.

**Arguments:**
- `query` (optional): Gmail search query (default: 'in:inbox')

#### `gmail_list_attachments`
List all attachments in a Gmail message.

**Arguments:**
- `messageId` (required): The ID of the Gmail message

**Returns:** JSON array of attachment metadata including attachmentId, filename, mimeType, size, and human-readable size.

#### `gmail_get_attachment`
Get the content of an attachment from a Gmail message.

**Arguments:**
- `messageId` (required): The ID of the Gmail message
- `attachmentId` (required): The ID of the attachment
- `encoding` (optional): Encoding format - 'base64' (default) or 'text'

**Returns:** Attachment content in the specified encoding.

**Note:** Use 'text' encoding for text-based attachments (.txt, .ics, .csv, etc.) and 'base64' for binary files (.pdf, .png, .zip, etc.).

**Security:** Attachments are limited to 25MB in size.

#### `gmail_get_message_body`
Extract text or HTML body from a Gmail message.

**Arguments:**
- `messageId` (required): The ID of the Gmail message
- `format` (optional): Body format - 'text' (default) or 'html'

**Returns:** Message body content in the specified format.

**Use Case:** Useful for extracting Google Docs/Drive links from email bodies, since Google Meet notes are typically shared as links rather than attachments.

#### `gmail_extract_doc_links`
Extract Google Docs/Drive links from a Gmail message.

**Arguments:**
- `messageId` (required): The ID of the Gmail message
- `format` (optional): Body format to search - 'text' (default) or 'html'

**Returns:** JSON array of Google Docs/Drive links found in the message, including documentId, url, and type (document, spreadsheet, presentation, or drive).

**Use Case:** Extracts Google Docs, Sheets, Slides, and Drive file links from email bodies. Particularly useful for finding meeting notes shared via Google Docs links.

#### `gmail_search_contacts`
Search for contacts across all Google contact sources.

**Arguments:**
- `query` (required): Search query to find contacts (e.g., name, email, phone number)
- `maxResults` (optional): Maximum number of results to return (default: 10)

**Returns:** List of contacts matching the query, including display name, email address, and phone number.

**Contact Sources Searched:**
- **Personal Contacts**: Your saved contacts in Google Contacts
- **Other Contacts**: People you've interacted with via email but haven't saved
- **Directory Contacts**: Organizational directory (for Google Workspace accounts only)

**Use Case:** Find contact information from all your contact sources before sending an email or when looking up someone's contact details. The search automatically de-duplicates contacts across sources.

#### `gmail_send_email`
Send an email through Gmail.

**Arguments:**
- `to` (required): Recipient email address(es), comma-separated for multiple recipients
- `subject` (required): Email subject
- `body` (required): Email body content
- `cc` (optional): CC email address(es), comma-separated for multiple recipients
- `bcc` (optional): BCC email address(es), comma-separated for multiple recipients
- `isHTML` (optional): Whether the body is HTML (default: false for plain text)

**Returns:** Confirmation message with the sent message ID.

**Use Case:** Send emails programmatically through your Gmail account. Supports both plain text and HTML emails, with CC and BCC options.

#### `gmail_reply_to_email`
Reply to an existing email message in a thread.

**Arguments:**
- `messageId` (required): The ID of the message to reply to
- `threadId` (required): The ID of the email thread
- `body` (required): Reply body content
- `cc` (optional): CC email address(es), comma-separated for multiple recipients
- `bcc` (optional): BCC email address(es), comma-separated for multiple recipients
- `isHTML` (optional): Whether the body is HTML (default: false for plain text)

**Returns:** Confirmation message with the reply message ID and thread ID.

**Use Case:** Reply to emails while maintaining proper email threading. The tool automatically extracts the original sender, subject, and threading headers (In-Reply-To, References) to ensure the reply appears correctly in the email thread. Adds "Re:" prefix to the subject if not already present.

#### `gmail_forward_email`
Forward an existing email message to new recipients.

**Arguments:**
- `messageId` (required): The ID of the message to forward
- `to` (required): Recipient email address(es), comma-separated for multiple recipients
- `additionalBody` (optional): Additional message to add before the forwarded content
- `cc` (optional): CC email address(es), comma-separated for multiple recipients
- `bcc` (optional): BCC email address(es), comma-separated for multiple recipients
- `isHTML` (optional): Whether the body is HTML (default: false for plain text)

**Returns:** Confirmation message with the forwarded message ID.

**Use Case:** Forward emails to other recipients. The tool automatically includes the original message content with proper formatting, including the original sender, date, subject, and body. Adds "Fwd:" prefix to the subject if not already present. You can optionally add your own message before the forwarded content.

### Google OAuth Tools

#### `google_get_auth_url`
Get the OAuth authorization URL for Google services.

**Arguments:**
- `account` (optional): Account name (default: 'default')

**Returns:** Authorization URL that the user should visit to grant access to Gmail, Google Docs, and Google Drive for the specified account.

**Use Case:** When the OAuth token is missing or expired for an account, use this to get the authorization URL. After visiting the URL and authorizing access, use `google_save_auth_code` with the same account name and the provided code.

#### `google_save_auth_code`
Save the OAuth authorization code to complete authentication.

**Arguments:**
- `account` (optional): Account name (default: 'default')
- `authCode` (required): The authorization code obtained from the Google OAuth flow

**Returns:** Success message indicating the token has been saved for the specified account.

**Use Case:** After visiting the authorization URL from `google_get_auth_url`, Google provides an authorization code. Pass this code (along with the matching account name) to complete the OAuth flow and save the access token.

### Google Docs Tools

**Note:** All Google Docs tools support an optional `account` parameter to specify which Google account to use (default: 'default').

#### `docs_get_document`
Get Google Docs content by document ID.

**Arguments:**
- `account` (optional): Account name (default: 'default')
- `documentId` (required): The ID of the Google Doc (extracted from URL)
- `format` (optional): Output format - 'markdown' (default), 'text', or 'json'

**Returns:** Document content in the specified format. Markdown format preserves headings, lists, formatting, and links. **Fully supports documents with multiple tabs** (introduced October 2024) - all tabs and nested child tabs are automatically fetched and included in the output.

**OAuth:** Uses the unified Google OAuth token (see Configuration section above). If not already authenticated, you'll be prompted to authorize access.

**Use Case:** Retrieve the actual content of Google Meet notes, shared documents, or any Google Doc accessible to your account. Works seamlessly with both legacy single-tab documents and new multi-tab documents.

#### `docs_get_document_metadata`
Get metadata about a Google Doc or Drive file.

**Arguments:**
- `documentId` (required): The ID of the Google Doc or Drive file

**Returns:** JSON with document metadata including id, name, mimeType, createdTime, modifiedTime, size, and owners.

**Use Case:** Get information about a document without downloading its full content.

### Google Calendar Tools

**Note:** All Google Calendar tools support an optional `account` parameter to specify which Google account to use (default: 'default').

#### `calendar_list_events`
List/search calendar events within a time range.

**Arguments:**
- `account` (optional): Account name (default: 'default')
- `calendarId` (optional): Calendar ID (use 'primary' for primary calendar, default: 'primary')
- `timeMin` (required): Start time for the range (RFC3339 format, e.g., '2025-01-01T00:00:00Z')
- `timeMax` (required): End time for the range (RFC3339 format, e.g., '2025-01-31T23:59:59Z')
- `query` (optional): Search query to filter events

**Returns:** List of events with details including ID, summary, start/end times, location, attendees, and Google Meet link if present.

#### `calendar_get_event`
Get details of a specific calendar event.

**Arguments:**
- `account` (optional): Account name (default: 'default')
- `calendarId` (optional): Calendar ID (use 'primary' for primary calendar, default: 'primary')
- `eventId` (required): The ID of the event to retrieve

**Returns:** Full event details including description, attendees, status, and conference data.

#### `calendar_create_event`
Create a new calendar event (supports recurring, out-of-office, and Google Meet).

**Arguments:**
- `account` (optional): Account name (default: 'default')
- `calendarId` (optional): Calendar ID (use 'primary' for primary calendar, default: 'primary')
- `summary` (required): Event title/summary
- `description` (optional): Event description
- `location` (optional): Event location
- `start` (required): Start time (RFC3339 format, e.g., '2025-01-15T14:00:00Z')
- `end` (required): End time (RFC3339 format, e.g., '2025-01-15T15:00:00Z')
- `timeZone` (optional): Time zone (e.g., 'America/New_York'). Defaults to UTC.
- `attendees` (optional): Comma-separated list of attendee email addresses
- `recurrence` (optional): Recurrence rule (e.g., 'RRULE:FREQ=WEEKLY;BYDAY=MO,WE,FR')
- `eventType` (optional): Event type: 'default', 'outOfOffice', 'focusTime', 'workingLocation'
- `addGoogleMeet` (optional): Automatically add a Google Meet link to the event
- `guestsCanModify` (optional): Allow guests to modify the event
- `guestsCanInviteOthers` (optional): Allow guests to invite others
- `guestsCanSeeOtherGuests` (optional): Allow guests to see other guests

**Returns:** Created event details including ID and Google Meet link if added.

**Use Case:** Create meetings, out-of-office blocks, recurring events, or focus time with full control over guest permissions.

#### `calendar_update_event`
Update an existing calendar event.

**Arguments:**
- `account` (optional): Account name (default: 'default')
- `calendarId` (optional): Calendar ID (use 'primary' for primary calendar, default: 'primary')
- `eventId` (required): The ID of the event to update
- `summary` (optional): New event title/summary
- `description` (optional): New event description
- `location` (optional): New event location
- `start` (optional): New start time (RFC3339 format)
- `end` (optional): New end time (RFC3339 format)
- `timeZone` (optional): Time zone (e.g., 'America/New_York')
- `attendees` (optional): New comma-separated list of attendee email addresses
- `eventType` (optional): New event type
- `guestsCanModify` (optional): Allow guests to modify the event
- `guestsCanInviteOthers` (optional): Allow guests to invite others
- `guestsCanSeeOtherGuests` (optional): Allow guests to see other guests

**Returns:** Updated event details.

**Use Case:** Modify event details, change times, update attendees, or adjust guest permissions.

#### `calendar_delete_event`
Delete a calendar event.

**Arguments:**
- `account` (optional): Account name (default: 'default')
- `calendarId` (optional): Calendar ID (use 'primary' for primary calendar, default: 'primary')
- `eventId` (required): The ID of the event to delete

**Returns:** Confirmation message.

#### `calendar_extract_docs_links`
Extract Google Docs/Drive links from a calendar event.

**Arguments:**
- `account` (optional): Account name (default: 'default')
- `calendarId` (optional): Calendar ID (use 'primary' for primary calendar, default: 'primary')
- `eventId` (required): The ID of the event

**Returns:** List of Google Docs/Drive links found in event attachments and description.

**Use Case:** Extract meeting notes, agendas, or documents attached to calendar events.

#### `calendar_get_meet_link`
Get the Google Meet link from a calendar event.

**Arguments:**
- `account` (optional): Account name (default: 'default')
- `calendarId` (optional): Calendar ID (use 'primary' for primary calendar, default: 'primary')
- `eventId` (required): The ID of the event

**Returns:** Google Meet video conference link if present.

#### `calendar_list_calendars`
List all calendars accessible to the user.

**Arguments:**
- `account` (optional): Account name (default: 'default')

**Returns:** List of calendars with ID, name, access role, time zone, and primary calendar indication.

**Use Case:** Discover available calendars including shared calendars, team calendars, and room resources.

#### `calendar_get_calendar`
Get information about a specific calendar.

**Arguments:**
- `account` (optional): Account name (default: 'default')
- `calendarId` (required): Calendar ID (use 'primary' for primary calendar)

**Returns:** Calendar information including name, description, time zone, and access permissions.

#### `calendar_query_freebusy`
Check availability for one or more calendars/attendees in a time range.

**Arguments:**
- `account` (optional): Account name (default: 'default')
- `timeMin` (required): Start time for the range (RFC3339 format)
- `timeMax` (required): End time for the range (RFC3339 format)
- `calendars` (required): Comma-separated list of calendar IDs or email addresses to check

**Returns:** Free/busy information showing when each calendar has scheduled events.

**Use Case:** Check if colleagues are available before scheduling a meeting.

#### `calendar_find_available_time`
Find available time slots for scheduling a meeting with one or more attendees.

**Arguments:**
- `account` (optional): Account name (default: 'default')
- `attendees` (required): Comma-separated list of attendee email addresses
- `durationMinutes` (required): Meeting duration in minutes
- `timeMin` (required): Start time for search range (RFC3339 format)
- `timeMax` (required): End time for search range (RFC3339 format)
- `maxResults` (optional): Maximum number of available slots to return (default: 10)

**Returns:** List of available time slots where all attendees are free.

**Use Case:** Automatically find the best meeting times when scheduling with multiple participants.

### Google Meet Tools

**Note:** All Google Meet tools support an optional `account` parameter to specify which Google account to use (default: 'default').

**Space Configuration:** You can now programmatically create Google Meet spaces with automatic recording, transcription, and Gemini note-taking enabled! Use the space management tools below to configure these features.

#### `meet_create_space`
Create a new Google Meet space with optional auto-recording, transcription, and note-taking configuration.

**Arguments:**
- `account` (optional): Account name (default: 'default')
- `access_type` (optional): Who can join without knocking: 'OPEN', 'TRUSTED', 'RESTRICTED'
- `enable_recording` (optional): Enable automatic recording (default: false)
- `enable_transcription` (optional): Enable automatic transcription (default: false)
- `enable_smart_notes` (optional): Enable automatic note-taking with Gemini (default: false). Requires Gemini add-on.

**Returns:** New space details including meeting URI, meeting code, and configuration settings.

**Use Case:** Create a meeting space with automatic recording and transcription enabled for important meetings.

**Example:**
```bash
meet_create_space(
  enable_recording: true,
  enable_transcription: true,
  enable_smart_notes: true
)
```

#### `meet_get_space`
Get details about a Google Meet space including its configuration.

**Arguments:**
- `account` (optional): Account name (default: 'default')
- `space_name` (required): The resource name of the space (e.g., 'spaces/SPACE_ID')

**Returns:** Space details including meeting URI, meeting code, access settings, and artifact configuration (recording, transcription, smart notes status).

**Use Case:** Check the current configuration of a meeting space.

#### `meet_update_space_config`
Update the configuration of an existing Google Meet space to enable/disable auto-recording, transcription, and notes.

**Arguments:**
- `account` (optional): Account name (default: 'default')
- `space_name` (required): The resource name of the space to update (e.g., 'spaces/SPACE_ID')
- `access_type` (optional): Who can join without knocking: 'OPEN', 'TRUSTED', 'RESTRICTED'
- `enable_recording` (optional): Enable automatic recording
- `enable_transcription` (optional): Enable automatic transcription
- `enable_smart_notes` (optional): Enable automatic note-taking with Gemini. Requires Gemini add-on.

**Returns:** Updated space configuration.

**Use Case:** Enable recording and transcription for an existing meeting space.

**Example:**
```bash
meet_update_space_config(
  space_name: "spaces/abc123",
  enable_recording: true,
  enable_transcription: true
)
```

#### `meet_get_conference`
Get details about a Google Meet conference record.

**Arguments:**
- `account` (optional): Account name (default: 'default')
- `conference_record` (required): The resource name of the conference record (e.g., 'spaces/SPACE_ID/conferenceRecords/CONF_ID')

**Returns:** Conference metadata including space ID, meeting code, start/end times, and counts of recordings and transcripts.

**Use Case:** Retrieve information about a completed Google Meet session.

#### `meet_list_recordings`
List all recordings for a Google Meet conference.

**Arguments:**
- `account` (optional): Account name (default: 'default')
- `conference_record` (required): The resource name of the conference record

**Returns:** List of recordings with details including state, start/end times, and Google Drive file locations with download links.

**Use Case:** Find and access recorded meetings for review or sharing.

#### `meet_get_recording`
Get details about a specific Google Meet recording, including download link.

**Arguments:**
- `account` (optional): Account name (default: 'default')
- `recording_name` (required): The resource name of the recording (e.g., 'spaces/SPACE_ID/conferenceRecords/CONF_ID/recordings/REC_ID')

**Returns:** Recording details including state, timestamps, Drive file location, and export URI for downloading.

**Use Case:** Retrieve download link for a specific meeting recording.

#### `meet_list_transcripts`
List all transcripts for a Google Meet conference.

**Arguments:**
- `account` (optional): Account name (default: 'default')
- `conference_record` (required): The resource name of the conference record

**Returns:** List of transcripts with details including state, language, start/end times, and Google Docs file locations.

**Use Case:** Find available transcripts for a meeting.

#### `meet_get_transcript`
Get details about a specific Google Meet transcript.

**Arguments:**
- `account` (optional): Account name (default: 'default')
- `transcript_name` (required): The resource name of the transcript (e.g., 'spaces/SPACE_ID/conferenceRecords/CONF_ID/transcripts/TRANS_ID')

**Returns:** Transcript details including state, language, timestamps, and Docs file location.

**Use Case:** Get metadata about a specific meeting transcript.

#### `meet_get_transcript_text`
Get the full text content of a Google Meet transcript with timestamps and speakers.

**Arguments:**
- `account` (optional): Account name (default: 'default')
- `transcript_name` (required): The resource name of the transcript

**Returns:** Full transcript text with timestamps and speaker names for each entry.

**Use Case:** Retrieve the complete conversation from a meeting for review or analysis.

### Workflow Examples

#### Extracting Meeting Notes

```bash
# 1. Find emails with Google Docs links
gmail_list_threads(query: "meeting notes")

# 2. Extract doc links from an email
gmail_extract_doc_links(messageId: "msg123")
# Returns: [{"documentId": "1ABC...", "url": "https://docs.google.com/...", "type": "document"}]

# 3. Fetch the document content
docs_get_document(documentId: "1ABC...", format: "markdown")
# Returns the meeting notes in Markdown format
```

#### Searching Contacts and Sending Email

```bash
# 1. Search for a contact
gmail_search_contacts(query: "John Doe")
# Returns: List of contacts with name, email, and phone

# 2. Send an email to the contact
gmail_send_email(
  to: "john.doe@example.com",
  subject: "Follow up on meeting",
  body: "Hi John,\n\nThanks for the meeting today...",
  cc: "manager@example.com"
)
# Returns: Email sent successfully with message ID
```

#### Scheduling a Meeting with Multiple Attendees

```bash
# 1. Find available time slots for all attendees
calendar_find_available_time(
  attendees: "alice@example.com, bob@example.com, carol@example.com",
  durationMinutes: 60,
  timeMin: "2025-02-01T09:00:00Z",
  timeMax: "2025-02-01T17:00:00Z"
)
# Returns: List of available 1-hour slots

# 2. Create the meeting with Google Meet
calendar_create_event(
  summary: "Team Planning Session",
  start: "2025-02-01T14:00:00Z",
  end: "2025-02-01T15:00:00Z",
  attendees: "alice@example.com, bob@example.com, carol@example.com",
  addGoogleMeet: true,
  description: "Q1 2025 planning discussion"
)
# Returns: Event created with Google Meet link

# 3. Extract the Meet link from the event
calendar_get_meet_link(calendarId: "primary", eventId: "event123")
# Returns: https://meet.google.com/abc-defg-hij
```

#### Managing Out-of-Office and Focus Time

```bash
# 1. Create an out-of-office block
calendar_create_event(
  summary: "Out of Office - Vacation",
  start: "2025-03-15T00:00:00Z",
  end: "2025-03-22T00:00:00Z",
  eventType: "outOfOffice"
)

# 2. Schedule recurring focus time
calendar_create_event(
  summary: "Focus Time - Deep Work",
  start: "2025-02-03T09:00:00Z",
  end: "2025-02-03T11:00:00Z",
  recurrence: "RRULE:FREQ=WEEKLY;BYDAY=MO,WE,FR",
  eventType: "focusTime"
)
```

## MCP Server Configuration

### Using with Claude Desktop

Add to your Claude Desktop configuration (`~/Library/Application Support/Claude/claude_desktop_config.json` on macOS):

```json
{
  "mcpServers": {
    "inboxfewer": {
      "command": "/path/to/inboxfewer",
      "args": ["serve"]
    }
  }
}
```

### Using with Other MCP Clients

For SSE or HTTP transports, configure your MCP client to connect to:
- SSE: `http://localhost:8080/sse` (with message endpoint at `/message`)
- HTTP: `http://localhost:8080/mcp`

## Development

### Quick Start

```bash
# Clone the repository
git clone https://github.com/teemow/inboxfewer.git
cd inboxfewer

# Build the project
make build

# Run tests
make test

# See all available targets
make help
```

### Debugging

To debug the MCP server with [mcp-debug](https://github.com/giantswarm/mcp-debug):

```bash
# Start the server
./scripts/start-mcp-server.sh

# In another terminal, use mcp-debug
mcp-debug --repl --endpoint http://localhost:8080/mcp
```

For development workflow (rebuild and restart):
```bash
./scripts/start-mcp-server.sh --restart
```

See [docs/debugging.md](docs/debugging.md) for details.

### Makefile Targets

The project includes a comprehensive Makefile with the following targets:

**Development:**
- `make build` - Build the binary
- `make install` - Install the binary to GOPATH/bin
- `make clean` - Clean build artifacts
- `make run` - Run the application

**Testing:**
- `make test` - Run tests
- `make test-coverage` - Run tests with coverage report
- `make vet` - Run go vet

**Code Quality:**
- `make fmt` - Run go fmt
- `make lint` - Run golangci-lint (requires golangci-lint installed)
- `make lint-yaml` - Run YAML linter (requires yamllint installed)
- `make tidy` - Run go mod tidy
- `make check` - Run all checks (fmt, vet, test, lint-yaml)

**Release:**
- `make release-dry-run` - Test the release process without publishing (requires goreleaser)
- `make release-local` - Create a release locally (requires goreleaser)

**Multi-platform Builds:**
- `make build-linux` - Build for Linux
- `make build-darwin` - Build for macOS
- `make build-windows` - Build for Windows
- `make build-all` - Build for all platforms

### Automated Releases

The project uses GitHub Actions to automatically create releases:

1. **CI Checks** (`ci.yaml`): Runs on every PR and push to master
   - Runs tests, linting, and formatting checks
   - Validates the release process with a dry-run

2. **Auto Release** (`auto-release.yaml`): Triggers on merged PRs to master
   - Automatically increments the patch version
   - Creates a git tag
   - Runs GoReleaser to build binaries for multiple platforms
   - Publishes a GitHub release with artifacts

Releases include pre-built binaries for:
- Linux (amd64, arm64)
- macOS/Darwin (amd64, arm64)
- Windows (amd64, arm64)

### Project Structure

```
inboxfewer/
├── cmd/                    # Command implementations
│   ├── root.go            # Root command
│   ├── cleanup.go         # Cleanup command (original functionality)
│   ├── serve.go           # MCP server command
│   └── version.go         # Version command
├── internal/
│   ├── gmail/             # Gmail client and utilities
│   │   ├── client.go      # Gmail API client
│   │   ├── attachments.go # Attachment retrieval
│   │   ├── doc_links.go   # Google Docs URL extraction
│   │   ├── classifier.go  # Thread classification
│   │   └── types.go       # GitHub issue/PR types
│   ├── docs/              # Google Docs client and utilities
│   │   ├── client.go      # Google Docs API client
│   │   ├── converter.go   # Document to Markdown/text conversion
│   │   ├── types.go       # Document metadata types
│   │   └── doc.go         # Package documentation
│   ├── calendar/          # Google Calendar client and utilities
│   │   ├── client.go      # Calendar API client
│   │   ├── types.go       # Event and calendar types
│   │   └── doc.go         # Package documentation
│   ├── meet/              # Google Meet client and utilities
│   │   ├── client.go      # Meet API client
│   │   ├── types.go       # Conference, recording, and transcript types
│   │   └── doc.go         # Package documentation
│   ├── google/            # Unified Google OAuth2 authentication
│   │   ├── oauth.go       # OAuth token management for all Google services
│   │   └── doc.go         # Package documentation
│   ├── github/            # GitHub types and utilities
│   │   └── types.go       # GitHub issue/PR types
│   ├── server/            # MCP server context
│   │   └── context.go     # Server context management
│   └── tools/             # MCP tool implementations
│       ├── google_tools/  # Google OAuth MCP tools
│       │   ├── tools.go   # OAuth authentication tools
│       │   └── doc.go     # Package documentation
│       ├── gmail_tools/   # Gmail-related MCP tools
│       │   ├── tools.go           # Thread tools
│       │   ├── attachment_tools.go # Attachment tools
│       │   └── doc.go             # Package documentation
│       ├── docs_tools/    # Google Docs MCP tools
│       │   ├── tools.go   # Docs retrieval tools
│       │   └── doc.go     # Package documentation
│       ├── calendar_tools/ # Google Calendar MCP tools
│       │   ├── tools.go            # Tool registration
│       │   ├── event_tools.go      # Event management tools
│       │   ├── calendar_list_tools.go # Calendar list tools
│       │   ├── scheduling_tools.go # Scheduling and availability tools
│       │   └── doc.go              # Package documentation
│       └── meet_tools/    # Google Meet MCP tools
│           ├── tools.go   # Meet artifact retrieval tools
│           └── doc.go     # Package documentation
├── docs/                  # Documentation
│   └── debugging.md       # Debugging guide
├── scripts/               # Utility scripts
│   └── start-mcp-server.sh # Development server script
├── .github/               # GitHub Actions workflows
│   └── workflows/
│       ├── ci.yaml        # Continuous integration
│       └── auto-release.yaml # Automated releases
├── main.go                # Application entry point
├── Makefile               # Build automation
├── go.mod                 # Go module definition
└── README.md              # This file
```

### Building

```bash
go build -o inboxfewer
```

### Testing

```bash
go test ./...
```

## License

See LICENSE file for details.

## Credits

Original concept and implementation by Brad Fitzpatrick.
MCP server integration added to provide AI assistant capabilities.

## Announcement

Original announcement: https://twitter.com/bradfitz/status/652973744302919680
