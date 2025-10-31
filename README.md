# inboxfewer

Archives Gmail threads for closed GitHub issues and pull requests.

## Features

- **Gmail Integration**: Automatically archives emails related to closed GitHub issues and PRs
- **Contact Search**: Search for contacts in Google Contacts by name, email, or phone number
- **Email Sending**: Send emails through Gmail API with support for CC, BCC, and HTML formatting
- **Newsletter Unsubscribe**: Extract and use List-Unsubscribe headers to easily unsubscribe from newsletters
- **Email Filters**: Create, list, and manage Gmail filters programmatically to automatically organize incoming emails
- **Google Docs Integration**: Extract and retrieve Google Docs content from email messages, with full support for multi-tab documents (Oct 2024 feature)
- **Google Drive Integration**: Full file and folder management including upload, download, search, organize, and share files with granular permissions
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

**Note:** Each OAuth token provides access to Gmail, Google Docs, Google Drive, Google Contacts, Google Calendar, Google Meet, and Google Tasks APIs with the following scopes:
- Gmail: Read, modify, and send messages
- Google Docs: Read document content
- Google Drive: Full read/write access to files and folders
- Google Contacts: Read contact information (personal contacts, interaction history, and directory)
- Google Calendar: Read and write calendar events, check availability, and manage calendars
- Google Meet: Read meeting artifacts (recordings, transcripts) and configure meeting spaces (enable/disable auto-recording, transcription, note-taking)
- Google Tasks: Read and write tasks and task lists

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

### Safety Mode (Read-Only by Default)

**By default, the MCP server operates in read-only mode** for AI safety. Only safe, non-destructive operations are available:

**Always Available (Safe Operations):**
- List, get, search, and query operations
- Archive threads (safe cleanup)
- Create tasks and task lists (safe planning)
- Create calendar entries (safe scheduling)
- Unsubscribe from emails (safe inbox cleanup)
- Create and delete Gmail filters (safe email organization)
- Create and configure Meet spaces (safe meeting setup)

**Requires `--yolo` Flag (Write Operations):**
- Email sending, replying, and forwarding
- Drive file operations (upload, delete, move, share)
- Calendar event deletion and updates
- Task deletion and updates

#### Enable Write Operations

To enable all write operations, use the `--yolo` flag:

```bash
# Enable write operations with stdio
inboxfewer serve --yolo

# Enable write operations with SSE
inboxfewer serve --transport sse --http-addr :8080 --yolo

# Enable write operations with HTTP
inboxfewer serve --transport streamable-http --http-addr :8080 --yolo
```

### Options

```bash
--debug           Enable debug logging
--transport       Transport type: stdio, sse, or streamable-http (default: stdio)
--http-addr       HTTP server address for sse/http transports (default: :8080)
--yolo            Enable write operations (default: false, read-only mode)
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

#### `gmail_get_unsubscribe_info`
Extract unsubscribe information from a Gmail message.

**Arguments:**
- `account` (optional): Account name (default: 'default')
- `messageId` (required): The ID of the Gmail message to check for unsubscribe information

**Returns:** Available unsubscribe methods (mailto or HTTP) extracted from the List-Unsubscribe header.

**Use Case:** Check if a newsletter or promotional email has unsubscribe information before attempting to unsubscribe. Many marketing emails include RFC 2369 compliant List-Unsubscribe headers that provide one-click unsubscribe options.

#### `gmail_unsubscribe_via_http`
Unsubscribe from a newsletter using an HTTP unsubscribe link.

**Arguments:**
- `account` (optional): Account name (default: 'default')
- `url` (required): The HTTP/HTTPS unsubscribe URL (obtained from `gmail_get_unsubscribe_info`)

**Returns:** Confirmation message indicating success or failure.

**Use Case:** Automatically unsubscribe from newsletters and promotional emails by visiting the HTTP unsubscribe link. Use `gmail_get_unsubscribe_info` first to get available unsubscribe methods. For mailto unsubscribe links, use `gmail_send_email` to send the unsubscribe request.

**Note:** This follows RFC 2369 List-Unsubscribe specification. Some senders may require email confirmation after visiting the unsubscribe link.

#### `gmail_create_filter`
Create a new Gmail filter to automatically organize incoming emails.

**Arguments:**
- `account` (optional): Account name (default: 'default')
- **Criteria** (at least one required):
  - `from` (optional): Filter emails from this sender (e.g., 'newsletter@example.com')
  - `to` (optional): Filter emails sent to this recipient (e.g., 'myalias@example.com')
  - `subject` (optional): Filter emails with this subject (e.g., 'Weekly Report')
  - `query` (optional): Gmail search query for advanced filtering (e.g., 'has:attachment larger:10M')
  - `hasAttachment` (optional): Filter emails that have attachments
- **Actions** (at least one required):
  - `addLabelIds` (optional): Comma-separated list of label IDs to add (use `gmail_list_labels` to get IDs)
  - `removeLabelIds` (optional): Comma-separated list of label IDs to remove
  - `archive` (optional): Remove from inbox (archive)
  - `markAsRead` (optional): Mark as read
  - `star` (optional): Add star
  - `markAsSpam` (optional): Mark as spam
  - `delete` (optional): Send to trash
  - `forward` (optional): Forward to this email address

**Returns:** Details of the created filter including its ID and configuration.

**Use Case:** Automatically organize emails by sender, subject, or content. Create filters to archive newsletters, label important emails, forward specific messages, or delete spam. Filters apply to both existing and future emails.

**Examples:**
- Archive all emails from a specific sender: `from="newsletter@example.com", archive=true`
- Label and star important emails: `subject="Urgent", addLabelIds="Label_1", star=true`
- Auto-delete spam: `from="spam@example.com", delete=true`

#### `gmail_list_filters`
List all existing Gmail filters for the account.

**Arguments:**
- `account` (optional): Account name (default: 'default')

**Returns:** List of all filters with their IDs, criteria, and actions.

**Use Case:** Review existing filters to understand current email organization rules. Get filter IDs for deletion. Audit and manage your email automation setup.

#### `gmail_delete_filter`
Delete a Gmail filter by its ID.

**Arguments:**
- `account` (optional): Account name (default: 'default')
- `filterId` (required): The ID of the filter to delete (obtained from `gmail_list_filters`)

**Returns:** Confirmation message.

**Use Case:** Remove outdated or incorrect filters. Clean up email automation rules that are no longer needed.

#### `gmail_list_labels`
List all Gmail labels for the account.

**Arguments:**
- `account` (optional): Account name (default: 'default')

**Returns:** List of all labels with their IDs, names, and types (system or user).

**Use Case:** Get label IDs needed for creating filters. Browse available labels before organizing emails. System labels include INBOX, SENT, DRAFT, SPAM, TRASH, UNREAD, STARRED, and IMPORTANT.

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

### Google Drive Tools

**Note:** All Google Drive tools support an optional `account` parameter to specify which Google account to use (default: 'default').

#### `drive_upload_file`
Upload a file to Google Drive.

**Arguments:**
- `account` (optional): Account name (default: 'default')
- `name` (required): The name of the file
- `content` (required): The file content (base64-encoded for binary files, or plain text)
- `mimeType` (optional): The MIME type of the file (e.g., 'application/pdf', 'text/plain', 'image/png')
- `parentFolders` (optional): Comma-separated list of parent folder IDs where the file should be placed
- `description` (optional): A short description of the file
- `isBase64` (optional): Whether the content is base64-encoded (default: true for binary files)

**Returns:** File metadata including ID, name, webViewLink, and other properties.

**Use Case:** Upload documents, images, PDFs, or any other files to your Google Drive programmatically.

**Example:**
```bash
drive_upload_file(
  account: "work",
  name: "report.pdf",
  content: "<base64-encoded-content>",
  mimeType: "application/pdf",
  parentFolders: "folder_id_123"
)
```

#### `drive_list_files`
List files in Google Drive with optional filtering.

**Arguments:**
- `account` (optional): Account name (default: 'default')
- `query` (optional): Query for filtering files using Google Drive's query language (e.g., "name contains 'report'", "mimeType='application/pdf'")
- `maxResults` (optional): Maximum number of files to return (default: 100, max: 1000)
- `orderBy` (optional): Sort order (e.g., 'folder,modifiedTime desc,name')
- `includeTrashed` (optional): Include trashed files in results (default: false)
- `pageToken` (optional): Page token for retrieving the next page of results

**Returns:** List of files with metadata and nextPageToken for pagination.

**Use Case:** Find files by name, type, owner, or other criteria. Search your entire Drive or specific folders.

**Query Examples:**
- Find PDFs: `"mimeType='application/pdf'"`
- Find files modified today: `"modifiedTime > '2025-10-31T00:00:00'"`
- Find files in a folder: `"'folder_id' in parents"`
- Combine filters: `"name contains 'invoice' and mimeType='application/pdf' and trashed=false"`

#### `drive_get_file`
Get metadata for a specific file in Google Drive.

**Arguments:**
- `account` (optional): Account name (default: 'default')
- `fileId` (required): The ID of the file

**Returns:** File metadata including name, mimeType, size, owners, sharing status, and permissions.

**Use Case:** Get detailed information about a file including who owns it, when it was modified, and who has access.

#### `drive_download_file`
Download the content of a file from Google Drive.

**Arguments:**
- `account` (optional): Account name (default: 'default')
- `fileId` (required): The ID of the file to download
- `asBase64` (optional): Return content as base64-encoded string (default: false for text)

**Returns:** File content as text or base64-encoded string.

**Use Case:** Download files to process their content, create backups, or transfer data.

**Note:** For binary files (images, PDFs, etc.), use `asBase64: true` to get base64-encoded content.

#### `drive_delete_file`
Delete a file from Google Drive.

**Arguments:**
- `account` (optional): Account name (default: 'default')
- `fileId` (required): The ID of the file to delete

**Returns:** Confirmation message.

**Use Case:** Remove files that are no longer needed. The file is moved to trash and can be restored from there.

#### `drive_create_folder`
Create a new folder in Google Drive.

**Arguments:**
- `account` (optional): Account name (default: 'default')
- `name` (required): The name of the folder
- `parentFolders` (optional): Comma-separated list of parent folder IDs where the folder should be created

**Returns:** Folder metadata including ID and webViewLink.

**Use Case:** Organize files by creating folder structures programmatically.

**Example:**
```bash
drive_create_folder(
  name: "Project Reports",
  parentFolders: "root_folder_id"
)
```

#### `drive_move_file`
Move or rename a file in Google Drive.

**Arguments:**
- `account` (optional): Account name (default: 'default')
- `fileId` (required): The ID of the file to move or rename
- `newName` (optional): The new name for the file (leave empty to keep current name)
- `addParents` (optional): Comma-separated list of folder IDs to add as parents
- `removeParents` (optional): Comma-separated list of folder IDs to remove as parents

**Returns:** Updated file metadata.

**Use Case:** Reorganize files by moving them between folders or renaming them. A file can have multiple parent folders in Drive.

**Example:**
```bash
# Move file to different folder
drive_move_file(
  fileId: "file_123",
  addParents: "new_folder_id",
  removeParents: "old_folder_id"
)

# Rename file
drive_move_file(
  fileId: "file_123",
  newName: "Updated Report.pdf"
)
```

#### `drive_share_file`
Share a file in Google Drive by granting permissions.

**Arguments:**
- `account` (optional): Account name (default: 'default')
- `fileId` (required): The ID of the file to share
- `type` (required): The type of grantee: 'user', 'group', 'domain', or 'anyone'
- `role` (required): The role to grant: 'owner', 'organizer', 'fileOrganizer', 'writer', 'commenter', or 'reader'
- `emailAddress` (optional): Email address (required if type is 'user' or 'group')
- `domain` (optional): Domain name (required if type is 'domain')
- `sendNotificationEmail` (optional): Send a notification email to the grantee (default: false)
- `emailMessage` (optional): Custom message to include in the notification email

**Returns:** Permission details including ID and role.

**Use Case:** Share files with specific people, groups, or make them publicly accessible.

**Examples:**
```bash
# Share with specific user
drive_share_file(
  fileId: "file_123",
  type: "user",
  role: "reader",
  emailAddress: "colleague@example.com",
  sendNotificationEmail: true
)

# Make file public
drive_share_file(
  fileId: "file_123",
  type: "anyone",
  role: "reader"
)

# Share with entire domain
drive_share_file(
  fileId: "file_123",
  type: "domain",
  role: "reader",
  domain: "example.com"
)
```

#### `drive_list_permissions`
List all permissions for a file in Google Drive.

**Arguments:**
- `account` (optional): Account name (default: 'default')
- `fileId` (required): The ID of the file

**Returns:** List of permissions showing who has access to the file and their roles.

**Use Case:** Audit file access, see who has permissions before removing them, or check sharing status.

#### `drive_remove_permission`
Remove a permission from a file in Google Drive.

**Arguments:**
- `account` (optional): Account name (default: 'default')
- `fileId` (required): The ID of the file
- `permissionId` (required): The ID of the permission to remove (get this from `drive_list_permissions`)

**Returns:** Confirmation message.

**Use Case:** Revoke access from users, groups, or remove public sharing.

**Example:**
```bash
# First, list permissions to get the permission ID
permissions = drive_list_permissions(fileId: "file_123")

# Then remove specific permission
drive_remove_permission(
  fileId: "file_123",
  permissionId: "permission_456"
)
```

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

### Google Tasks Tools

**Note:** All Google Tasks tools support an optional `account` parameter to specify which Google account to use (default: 'default').

#### `tasks_list_task_lists`
List all task lists for the authenticated user.

**Arguments:**
- `account` (optional): Account name (default: 'default')

**Returns:** List of all task lists with their IDs, titles, and last updated timestamps.

**Use Case:** Discover all your task lists to choose which one to work with.

#### `tasks_get_task_list`
Get details of a specific task list.

**Arguments:**
- `account` (optional): Account name (default: 'default')
- `taskListId` (required): The ID of the task list to retrieve

**Returns:** Task list details including ID, title, and updated timestamp.

**Use Case:** Retrieve metadata about a specific task list.

#### `tasks_create_task_list`
Create a new task list.

**Arguments:**
- `account` (optional): Account name (default: 'default')
- `title` (required): The title of the new task list

**Returns:** Created task list with its ID and details.

**Use Case:** Organize tasks by creating separate lists for different projects or categories (e.g., "Work", "Personal", "Shopping").

#### `tasks_update_task_list`
Update a task list's title.

**Arguments:**
- `account` (optional): Account name (default: 'default')
- `taskListId` (required): The ID of the task list to update
- `title` (required): The new title for the task list

**Returns:** Updated task list details.

**Use Case:** Rename a task list to better reflect its purpose.

#### `tasks_delete_task_list`
Delete a task list.

**Arguments:**
- `account` (optional): Account name (default: 'default')
- `taskListId` (required): The ID of the task list to delete

**Returns:** Confirmation message.

**Use Case:** Remove task lists that are no longer needed.

#### `tasks_list_tasks`
List tasks in a task list with optional filters.

**Arguments:**
- `account` (optional): Account name (default: 'default')
- `taskListId` (required): The ID of the task list
- `showCompleted` (optional): Include completed tasks (default: false)
- `dueMin` (optional): Only return tasks with due date after this time (RFC3339 format)
- `dueMax` (optional): Only return tasks with due date before this time (RFC3339 format)

**Returns:** List of tasks with full details including title, notes, status, due date, and completion time.

**Use Case:** View all pending tasks, or filter tasks by due date to focus on what's coming up soon.

#### `tasks_get_task`
Get details of a specific task.

**Arguments:**
- `account` (optional): Account name (default: 'default')
- `taskListId` (required): The ID of the task list
- `taskId` (required): The ID of the task to retrieve

**Returns:** Full task details including all fields and related links.

**Use Case:** Retrieve complete information about a specific task.

#### `tasks_create_task`
Create a new task in a task list.

**Arguments:**
- `account` (optional): Account name (default: 'default')
- `taskListId` (required): The ID of the task list
- `title` (required): The title of the new task
- `notes` (optional): Notes or description for the task
- `due` (optional): Due date for the task (RFC3339 format, e.g., '2025-11-07T09:00:00Z')
- `parent` (optional): Parent task ID to create a subtask
- `previous` (optional): Previous sibling task ID for positioning

**Returns:** Created task with its ID and details.

**Use Case:** Add new tasks to track work items, errands, or reminders. Create subtasks to break down larger tasks.

**Example:**
```bash
tasks_create_task(
  taskListId: "MTIzNDU2Nzg5MDEyMzQ1Njc4OTA",
  title: "Review pull request #123",
  notes: "Check tests and documentation",
  due: "2025-11-07T17:00:00Z"
)
```

#### `tasks_update_task`
Update an existing task.

**Arguments:**
- `account` (optional): Account name (default: 'default')
- `taskListId` (required): The ID of the task list
- `taskId` (required): The ID of the task to update
- `title` (optional): New title for the task
- `notes` (optional): New notes or description
- `status` (optional): New status: 'needsAction' or 'completed'
- `due` (optional): New due date (RFC3339 format)

**Returns:** Updated task details.

**Use Case:** Modify task details, change due dates, or update notes with progress information.

#### `tasks_delete_task`
Delete a task.

**Arguments:**
- `account` (optional): Account name (default: 'default')
- `taskListId` (required): The ID of the task list
- `taskId` (required): The ID of the task to delete

**Returns:** Confirmation message.

**Use Case:** Remove tasks that are no longer relevant or were created by mistake.

#### `tasks_complete_task`
Mark a task as completed.

**Arguments:**
- `account` (optional): Account name (default: 'default')
- `taskListId` (required): The ID of the task list
- `taskId` (required): The ID of the task to complete

**Returns:** Updated task with completed status and completion timestamp.

**Use Case:** Mark tasks as done when you finish them. The completion time is automatically recorded.

#### `tasks_move_task`
Move a task to a different position or parent.

**Arguments:**
- `account` (optional): Account name (default: 'default')
- `taskListId` (required): The ID of the task list
- `taskId` (required): The ID of the task to move
- `parent` (optional): New parent task ID (empty string to move to root level)
- `previous` (optional): Previous sibling task ID for positioning

**Returns:** Moved task with updated position information.

**Use Case:** Reorganize tasks by changing their order or making them subtasks of other tasks.

#### `tasks_clear_completed`
Clear all completed tasks from a task list.

**Arguments:**
- `account` (optional): Account name (default: 'default')
- `taskListId` (required): The ID of the task list to clear completed tasks from

**Returns:** Confirmation message.

**Use Case:** Clean up your task list by removing all completed tasks at once. This is useful for maintaining a clean, focused list.

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
│   ├── drive/             # Google Drive client and utilities
│   │   ├── client.go      # Drive API client
│   │   ├── types.go       # File and permission types
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
│       ├── drive_tools/   # Google Drive MCP tools
│       │   ├── tools.go         # Tool registration
│       │   ├── file_tools.go    # File operations
│       │   ├── folder_tools.go  # Folder operations
│       │   ├── share_tools.go   # Permission management
│       │   └── doc.go           # Package documentation
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
