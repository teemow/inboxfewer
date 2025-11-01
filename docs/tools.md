# MCP Tools Reference

This document provides a complete reference of all tools available when running inboxfewer as an MCP server.

**Note:** This documentation is automatically generated from the tool definitions.

## Table of Contents

- [Gmail Tools](#gmail-tools)
- [Google Calendar Tools](#google-calendar-tools)
- [Google Docs Tools](#google-docs-tools)
- [Google Drive Tools](#google-drive-tools)
- [Google Meet Tools](#google-meet-tools)
- [Google Tasks Tools](#google-tasks-tools)
- [OAuth Authentication](#oauth-authentication)

## Multi-Account Support

All Google-related MCP tools support an optional `account` parameter to specify which Google account to use:

- **Default behavior:** If `account` is not specified, the `default` account is used
- **Multiple accounts:** You can manage multiple Google accounts (e.g., `work`, `personal`)
- **Per-tool specification:** Each tool call can use a different account

## Gmail Tools

### gmail_archive_stale_threads

Archive all Gmail threads in inbox that are related to closed GitHub issues/PRs

**Arguments:**
- `account` (optional): Account name (default: 'default'). Used to manage multiple Google accounts.
- `query` (optional): Gmail search query (default: 'in:inbox')


### gmail_archive_threads

Archive one or more Gmail threads by removing them from the inbox

**Arguments:**
- `account` (optional): Account name (default: 'default'). Used to manage multiple Google accounts.
- `threadIds` (required): Thread ID (string) or array of thread IDs to archive


### gmail_check_stale

Check if a Gmail thread is stale (GitHub issue/PR is closed)

**Arguments:**
- `account` (optional): Account name (default: 'default'). Used to manage multiple Google accounts.
- `threadId` (required): The ID of the thread to check


### gmail_classify_thread

Classify a Gmail thread to determine if it's related to GitHub issues or PRs

**Arguments:**
- `account` (optional): Account name (default: 'default'). Used to manage multiple Google accounts.
- `threadId` (required): The ID of the thread to classify


### gmail_create_filter

Create a new Gmail filter to automatically organize incoming emails. Filters can match on sender, recipient, subject, or custom queries, and perform actions like labeling, archiving, or marking as read.

**Arguments:**
- `account` (optional): Account name (default: 'default'). Used to manage multiple Google accounts.
- `addLabelIds` (optional): Comma-separated list of label IDs to add (e.g., 'Label_1,Label_2'). Use gmail_list_labels to get label IDs.
- `archive` (optional): Remove from inbox (archive)
- `delete` (optional): Send to trash
- `forward` (optional): Forward to this email address
- `from` (optional): Filter emails from this sender (e.g., 'newsletter@example.com')
- `hasAttachment` (optional): Filter emails that have attachments
- `markAsRead` (optional): Mark as read
- `markAsSpam` (optional): Mark as spam
- `query` (optional): Gmail search query for advanced filtering (e.g., 'has:attachment larger:10M')
- `removeLabelIds` (optional): Comma-separated list of label IDs to remove (e.g., 'INBOX,UNREAD')
- `star` (optional): Add star
- `subject` (optional): Filter emails with this subject (e.g., 'Weekly Report')
- `to` (optional): Filter emails sent to this recipient (e.g., 'myalias@example.com')


### gmail_delete_filter

Delete a Gmail filter by its ID (obtain ID from gmail_list_filters)

**Arguments:**
- `account` (optional): Account name (default: 'default'). Used to manage multiple Google accounts.
- `filterId` (required): The ID of the filter to delete (obtained from gmail_list_filters)


### gmail_extract_doc_links

Extract Google Docs/Drive links from a Gmail message

**Arguments:**
- `account` (optional): Account name (default: 'default'). Used to manage multiple Google accounts.
- `format` (optional): Body format to search: 'text' (default) or 'html'
- `messageId` (required): The ID of the Gmail message


### gmail_forward_email

Forward an existing email message to new recipients

**Arguments:**
- `account` (optional): Account name (default: 'default'). Used to manage multiple Google accounts.
- `additionalBody` (optional): Additional message to add before the forwarded content
- `bcc` (optional): BCC email address(es), comma-separated for multiple recipients
- `cc` (optional): CC email address(es), comma-separated for multiple recipients
- `isHTML` (optional): Whether the body is HTML (default: false for plain text)
- `messageId` (required): The ID of the message to forward
- `to` (required): Recipient email address(es), comma-separated for multiple recipients


### gmail_get_attachment

Get the content of an attachment

**Arguments:**
- `account` (optional): Account name (default: 'default'). Used to manage multiple Google accounts.
- `attachmentId` (required): The ID of the attachment
- `encoding` (optional): Encoding format: 'base64' (default) or 'text'
- `messageId` (required): The ID of the Gmail message


### gmail_get_message_body

Extract text or HTML body from a Gmail message

**Arguments:**
- `account` (optional): Account name (default: 'default'). Used to manage multiple Google accounts.
- `format` (optional): Body format: 'text' (default) or 'html'
- `messageId` (required): The ID of the Gmail message


### gmail_get_unsubscribe_info

Extract unsubscribe information from a Gmail message. Returns available unsubscribe methods (mailto or HTTP).

**Arguments:**
- `account` (optional): Account name (default: 'default'). Used to manage multiple Google accounts.
- `messageId` (required): The ID of the Gmail message to check for unsubscribe information


### gmail_list_attachments

List all attachments in a Gmail message

**Arguments:**
- `account` (optional): Account name (default: 'default'). Used to manage multiple Google accounts.
- `messageId` (required): The ID of the Gmail message


### gmail_list_filters

List all existing Gmail filters for the account

**Arguments:**
- `account` (optional): Account name (default: 'default'). Used to manage multiple Google accounts.


### gmail_list_labels

List all Gmail labels for the account. Use this to get label IDs for creating filters.

**Arguments:**
- `account` (optional): Account name (default: 'default'). Used to manage multiple Google accounts.


### gmail_list_threads

List Gmail threads matching a query

**Arguments:**
- `account` (optional): Account name (default: 'default'). Used to manage multiple Google accounts.
- `maxResults` (optional): Maximum number of results to return (default: 10)
- `query` (required): Gmail search query (e.g., 'in:inbox', 'from:user@example.com')


### gmail_reply_to_email

Reply to an existing email message in a thread

**Arguments:**
- `account` (optional): Account name (default: 'default'). Used to manage multiple Google accounts.
- `bcc` (optional): BCC email address(es), comma-separated for multiple recipients
- `body` (required): Reply body content
- `cc` (optional): CC email address(es), comma-separated for multiple recipients
- `isHTML` (optional): Whether the body is HTML (default: false for plain text)
- `messageId` (required): The ID of the message to reply to
- `threadId` (required): The ID of the email thread


### gmail_search_contacts

Search for contacts in Google Contacts

**Arguments:**
- `account` (optional): Account name (default: 'default'). Used to manage multiple Google accounts.
- `maxResults` (optional): Maximum number of results to return (default: 10)
- `query` (required): Search query to find contacts (e.g., name, email, phone number)


### gmail_send_email

Send an email through Gmail

**Arguments:**
- `account` (optional): Account name (default: 'default'). Used to manage multiple Google accounts.
- `bcc` (optional): BCC email address(es), comma-separated for multiple recipients
- `body` (required): Email body content
- `cc` (optional): CC email address(es), comma-separated for multiple recipients
- `isHTML` (optional): Whether the body is HTML (default: false for plain text)
- `subject` (required): Email subject
- `to` (required): Recipient email address(es), comma-separated for multiple recipients


### gmail_unsubscribe_via_http

Unsubscribe from a newsletter using an HTTP unsubscribe link. Use gmail_get_unsubscribe_info first to get available unsubscribe methods.

**Arguments:**
- `account` (optional): Account name (default: 'default'). Used to manage multiple Google accounts.
- `url` (required): The HTTP/HTTPS unsubscribe URL (obtained from gmail_get_unsubscribe_info)


## Google Calendar Tools

### calendar_create_event

Create a new calendar event (supports recurring, out-of-office, and Google Meet)

**Arguments:**
- `account` (optional): Account name (default: 'default'). Used to manage multiple Google accounts.
- `addGoogleMeet` (optional): Automatically add a Google Meet link to the event
- `allDay` (optional): Create as all-day event (ignores time portion of start/end)
- `attendees` (optional): Comma-separated list of attendee email addresses
- `calendarId` (optional): Calendar ID (use 'primary' for primary calendar)
- `description` (optional): Event description
- `end` (required): End time (RFC3339 format, e.g., '2025-01-15T15:00:00Z')
- `eventType` (optional): Event type: 'default', 'outOfOffice', 'focusTime', 'workingLocation'
- `guestsCanInviteOthers` (optional): Allow guests to invite others
- `guestsCanModify` (optional): Allow guests to modify the event
- `guestsCanSeeOtherGuests` (optional): Allow guests to see other guests
- `location` (optional): Event location
- `recurrence` (optional): Recurrence rule (e.g., 'RRULE:FREQ=WEEKLY;BYDAY=MO,WE,FR')
- `start` (required): Start time (RFC3339 format, e.g., '2025-01-15T14:00:00Z')
- `summary` (required): Event title/summary
- `timeZone` (optional): Time zone (e.g., 'America/New_York'). Defaults to UTC.


### calendar_delete_event

Delete a calendar event

**Arguments:**
- `account` (optional): Account name (default: 'default'). Used to manage multiple Google accounts.
- `calendarId` (optional): Calendar ID (use 'primary' for primary calendar)
- `eventId` (required): The ID of the event to delete


### calendar_extract_docs_links

Extract Google Docs/Drive links from a calendar event

**Arguments:**
- `account` (optional): Account name (default: 'default'). Used to manage multiple Google accounts.
- `calendarId` (optional): Calendar ID (use 'primary' for primary calendar)
- `eventId` (required): The ID of the event


### calendar_find_available_time

Find available time slots for scheduling a meeting with one or more attendees

**Arguments:**
- `account` (optional): Account name (default: 'default'). Used to manage multiple Google accounts.
- `attendees` (required): Comma-separated list of attendee email addresses
- `durationMinutes` (required): Meeting duration in minutes
- `maxResults` (optional): Maximum number of available slots to return (default: 10)
- `timeMax` (required): End time for search range (RFC3339 format, e.g., '2025-01-01T17:00:00Z')
- `timeMin` (required): Start time for search range (RFC3339 format, e.g., '2025-01-01T09:00:00Z')


### calendar_get_calendar

Get information about a specific calendar

**Arguments:**
- `account` (optional): Account name (default: 'default'). Used to manage multiple Google accounts.
- `calendarId` (required): Calendar ID (use 'primary' for primary calendar)


### calendar_get_event

Get details of a specific calendar event

**Arguments:**
- `account` (optional): Account name (default: 'default'). Used to manage multiple Google accounts.
- `calendarId` (optional): Calendar ID (use 'primary' for primary calendar)
- `eventId` (required): The ID of the event to retrieve


### calendar_get_meet_link

Get the Google Meet link from a calendar event

**Arguments:**
- `account` (optional): Account name (default: 'default'). Used to manage multiple Google accounts.
- `calendarId` (optional): Calendar ID (use 'primary' for primary calendar)
- `eventId` (required): The ID of the event


### calendar_list_calendars

List all calendars accessible to the user

**Arguments:**
- `account` (optional): Account name (default: 'default'). Used to manage multiple Google accounts.


### calendar_list_events

List/search calendar events within a time range

**Arguments:**
- `account` (optional): Account name (default: 'default'). Used to manage multiple Google accounts.
- `calendarId` (optional): Calendar ID (use 'primary' for primary calendar)
- `query` (optional): Optional search query to filter events
- `timeMax` (required): End time for the range (RFC3339 format, e.g., '2025-01-31T23:59:59Z')
- `timeMin` (required): Start time for the range (RFC3339 format, e.g., '2025-01-01T00:00:00Z')


### calendar_query_freebusy

Check availability for one or more calendars/attendees in a time range

**Arguments:**
- `account` (optional): Account name (default: 'default'). Used to manage multiple Google accounts.
- `calendars` (required): Comma-separated list of calendar IDs or email addresses to check
- `timeMax` (required): End time for the range (RFC3339 format, e.g., '2025-01-31T23:59:59Z')
- `timeMin` (required): Start time for the range (RFC3339 format, e.g., '2025-01-01T00:00:00Z')


### calendar_update_event

Update an existing calendar event

**Arguments:**
- `account` (optional): Account name (default: 'default'). Used to manage multiple Google accounts.
- `allDay` (optional): Update to be an all-day event (ignores time portion of start/end)
- `attendees` (optional): New comma-separated list of attendee email addresses
- `calendarId` (optional): Calendar ID (use 'primary' for primary calendar)
- `description` (optional): New event description
- `end` (optional): New end time (RFC3339 format)
- `eventId` (required): The ID of the event to update
- `eventType` (optional): New event type: 'default', 'outOfOffice', 'focusTime', 'workingLocation'
- `guestsCanInviteOthers` (optional): Allow guests to invite others
- `guestsCanModify` (optional): Allow guests to modify the event
- `guestsCanSeeOtherGuests` (optional): Allow guests to see other guests
- `location` (optional): New event location
- `start` (optional): New start time (RFC3339 format)
- `summary` (optional): New event title/summary
- `timeZone` (optional): Time zone (e.g., 'America/New_York')


## Google Docs Tools

### docs_get_document

Get Google Docs content by document ID

**Arguments:**
- `account` (optional): Account name (default: 'default'). Used to manage multiple Google accounts.
- `documentId` (required): The ID of the Google Doc
- `format` (optional): Output format: 'markdown' (default), 'text', or 'json'


### docs_get_document_metadata

Get metadata about a Google Doc or Drive file

**Arguments:**
- `account` (optional): Account name (default: 'default'). Used to manage multiple Google accounts.
- `documentId` (required): The ID of the Google Doc or Drive file


## Google Drive Tools

### drive_create_folder

Create a new folder in Google Drive

**Arguments:**
- `account` (optional): Account name (default: 'default'). Used to manage multiple Google accounts.
- `name` (required): The name of the folder
- `parentFolders` (optional): Comma-separated list of parent folder IDs where the folder should be created


### drive_delete_file

Delete a file from Google Drive

**Arguments:**
- `account` (optional): Account name (default: 'default'). Used to manage multiple Google accounts.
- `fileId` (required): The ID of the file to delete


### drive_download_file

Download the content of a file from Google Drive

**Arguments:**
- `account` (optional): Account name (default: 'default'). Used to manage multiple Google accounts.
- `asBase64` (optional): Return content as base64-encoded string (default: false for text)
- `fileId` (required): The ID of the file to download


### drive_get_file

Get metadata for a specific file in Google Drive

**Arguments:**
- `account` (optional): Account name (default: 'default'). Used to manage multiple Google accounts.
- `fileId` (required): The ID of the file


### drive_list_files

List files in Google Drive with optional filtering

**Arguments:**
- `account` (optional): Account name (default: 'default'). Used to manage multiple Google accounts.
- `includeTrashed` (optional): Include trashed files in results (default: false)
- `maxResults` (optional): Maximum number of files to return (default: 100, max: 1000)
- `orderBy` (optional): Sort order (e.g., 'folder,modifiedTime desc,name')
- `pageToken` (optional): Page token for retrieving the next page of results
- `query` (optional): Query for filtering files using Google Drive's query language (e.g., "name contains 'report'", "mimeType='application/pdf'")


### drive_list_permissions

List all permissions for a file in Google Drive

**Arguments:**
- `account` (optional): Account name (default: 'default'). Used to manage multiple Google accounts.
- `fileId` (required): The ID of the file


### drive_move_file

Move or rename a file in Google Drive

**Arguments:**
- `account` (optional): Account name (default: 'default'). Used to manage multiple Google accounts.
- `addParents` (optional): Comma-separated list of folder IDs to add as parents
- `fileId` (required): The ID of the file to move or rename
- `newName` (optional): The new name for the file (leave empty to keep current name)
- `removeParents` (optional): Comma-separated list of folder IDs to remove as parents


### drive_remove_permission

Remove a permission from a file in Google Drive

**Arguments:**
- `account` (optional): Account name (default: 'default'). Used to manage multiple Google accounts.
- `fileId` (required): The ID of the file
- `permissionId` (required): The ID of the permission to remove (get this from drive_list_permissions)


### drive_share_file

Share a file in Google Drive by granting permissions

**Arguments:**
- `account` (optional): Account name (default: 'default'). Used to manage multiple Google accounts.
- `domain` (optional): Domain name (required if type is 'domain')
- `emailAddress` (optional): Email address (required if type is 'user' or 'group')
- `emailMessage` (optional): Custom message to include in the notification email
- `fileId` (required): The ID of the file to share
- `role` (required): The role to grant: 'owner', 'organizer', 'fileOrganizer', 'writer', 'commenter', or 'reader'
- `sendNotificationEmail` (optional): Send a notification email to the grantee (default: false)
- `type` (required): The type of grantee: 'user', 'group', 'domain', or 'anyone'


### drive_upload_file

Upload a file to Google Drive

**Arguments:**
- `account` (optional): Account name (default: 'default'). Used to manage multiple Google accounts.
- `content` (required): The file content (base64-encoded for binary files, or plain text)
- `description` (optional): A short description of the file
- `isBase64` (optional): Whether the content is base64-encoded (default: true for binary files, false for text)
- `mimeType` (optional): The MIME type of the file (e.g., 'application/pdf', 'text/plain', 'image/png')
- `name` (required): The name of the file
- `parentFolders` (optional): Comma-separated list of parent folder IDs where the file should be placed


## Google Meet Tools

### meet_create_space

Create a new Google Meet space with optional auto-recording, transcription, and note-taking configuration

**Arguments:**
- `access_type` (optional): Who can join without knocking: 'OPEN', 'TRUSTED', 'RESTRICTED' (optional)
- `account` (optional): Account name (default: 'default'). Used to manage multiple Google accounts.
- `enable_recording` (optional): Enable automatic recording (default: false)
- `enable_smart_notes` (optional): Enable automatic note-taking with Gemini (default: false). Requires Gemini add-on.
- `enable_transcription` (optional): Enable automatic transcription (default: false)


### meet_get_conference

Get details about a Google Meet conference record

**Arguments:**
- `account` (optional): Account name (default: 'default'). Used to manage multiple Google accounts.
- `conference_record` (required): The resource name of the conference record (e.g., 'spaces/SPACE_ID/conferenceRecords/CONF_ID')


### meet_get_recording

Get details about a specific Google Meet recording, including download link

**Arguments:**
- `account` (optional): Account name (default: 'default'). Used to manage multiple Google accounts.
- `recording_name` (required): The resource name of the recording (e.g., 'spaces/SPACE_ID/conferenceRecords/CONF_ID/recordings/REC_ID')


### meet_get_space

Get details about a Google Meet space including its configuration

**Arguments:**
- `account` (optional): Account name (default: 'default'). Used to manage multiple Google accounts.
- `space_name` (required): The resource name of the space (e.g., 'spaces/SPACE_ID')


### meet_get_transcript

Get details about a specific Google Meet transcript

**Arguments:**
- `account` (optional): Account name (default: 'default'). Used to manage multiple Google accounts.
- `transcript_name` (required): The resource name of the transcript (e.g., 'spaces/SPACE_ID/conferenceRecords/CONF_ID/transcripts/TRANS_ID')


### meet_get_transcript_text

Get the full text content of a Google Meet transcript with timestamps and speakers

**Arguments:**
- `account` (optional): Account name (default: 'default'). Used to manage multiple Google accounts.
- `transcript_name` (required): The resource name of the transcript (e.g., 'spaces/SPACE_ID/conferenceRecords/CONF_ID/transcripts/TRANS_ID')


### meet_list_recordings

List all recordings for a Google Meet conference

**Arguments:**
- `account` (optional): Account name (default: 'default'). Used to manage multiple Google accounts.
- `conference_record` (required): The resource name of the conference record (e.g., 'spaces/SPACE_ID/conferenceRecords/CONF_ID')


### meet_list_transcripts

List all transcripts for a Google Meet conference

**Arguments:**
- `account` (optional): Account name (default: 'default'). Used to manage multiple Google accounts.
- `conference_record` (required): The resource name of the conference record (e.g., 'spaces/SPACE_ID/conferenceRecords/CONF_ID')


### meet_update_space_config

Update the configuration of an existing Google Meet space (enable/disable auto-recording, transcription, notes)

**Arguments:**
- `access_type` (optional): Who can join without knocking: 'OPEN', 'TRUSTED', 'RESTRICTED' (optional)
- `account` (optional): Account name (default: 'default'). Used to manage multiple Google accounts.
- `enable_recording` (optional): Enable automatic recording
- `enable_smart_notes` (optional): Enable automatic note-taking with Gemini. Requires Gemini add-on.
- `enable_transcription` (optional): Enable automatic transcription
- `space_name` (required): The resource name of the space to update (e.g., 'spaces/SPACE_ID')


## Google Tasks Tools

### tasks_clear_completed

Clear all completed tasks from a task list

**Arguments:**
- `account` (optional): Account name (default: 'default'). Used to manage multiple Google accounts.
- `taskListId` (required): The ID of the task list to clear completed tasks from


### tasks_complete_tasks

Mark one or more tasks as completed

**Arguments:**
- `account` (optional): Account name (default: 'default'). Used to manage multiple Google accounts.
- `taskIds` (required): Task ID (string) or array of task IDs to complete
- `taskListId` (required): The ID of the task list


### tasks_create_task

Create a new task in a task list

**Arguments:**
- `account` (optional): Account name (default: 'default'). Used to manage multiple Google accounts.
- `due` (optional): Due date for the task (RFC3339 format, e.g., '2025-11-07T09:00:00Z')
- `notes` (optional): Notes or description for the task
- `parent` (optional): Parent task ID to create a subtask
- `previous` (optional): Previous sibling task ID for positioning
- `taskListId` (required): The ID of the task list
- `title` (required): The title of the new task


### tasks_create_task_list

Create a new task list

**Arguments:**
- `account` (optional): Account name (default: 'default'). Used to manage multiple Google accounts.
- `title` (required): The title of the new task list


### tasks_delete_task_list

Delete a task list

**Arguments:**
- `account` (optional): Account name (default: 'default'). Used to manage multiple Google accounts.
- `taskListId` (required): The ID of the task list to delete


### tasks_delete_tasks

Delete one or more tasks

**Arguments:**
- `account` (optional): Account name (default: 'default'). Used to manage multiple Google accounts.
- `taskIds` (required): Task ID (string) or array of task IDs to delete
- `taskListId` (required): The ID of the task list


### tasks_get_task

Get details of a specific task

**Arguments:**
- `account` (optional): Account name (default: 'default'). Used to manage multiple Google accounts.
- `taskId` (required): The ID of the task to retrieve
- `taskListId` (required): The ID of the task list


### tasks_get_task_list

Get details of a specific task list

**Arguments:**
- `account` (optional): Account name (default: 'default'). Used to manage multiple Google accounts.
- `taskListId` (required): The ID of the task list to retrieve


### tasks_list_task_lists

List all task lists for the authenticated user

**Arguments:**
- `account` (optional): Account name (default: 'default'). Used to manage multiple Google accounts.


### tasks_list_tasks

List tasks in a task list with optional filters

**Arguments:**
- `account` (optional): Account name (default: 'default'). Used to manage multiple Google accounts.
- `dueMax` (optional): Only return tasks with due date before this time (RFC3339 format)
- `dueMin` (optional): Only return tasks with due date after this time (RFC3339 format)
- `showCompleted` (optional): Include completed tasks (default: false)
- `taskListId` (required): The ID of the task list


### tasks_move_task

Move a task to a different position or parent

**Arguments:**
- `account` (optional): Account name (default: 'default'). Used to manage multiple Google accounts.
- `parent` (optional): New parent task ID (empty string to move to root level)
- `previous` (optional): Previous sibling task ID for positioning
- `taskId` (required): The ID of the task to move
- `taskListId` (required): The ID of the task list


### tasks_update_task

Update an existing task

**Arguments:**
- `account` (optional): Account name (default: 'default'). Used to manage multiple Google accounts.
- `due` (optional): New due date for the task (RFC3339 format)
- `notes` (optional): New notes or description for the task
- `status` (optional): New status: 'needsAction' or 'completed'
- `taskId` (required): The ID of the task to update
- `taskListId` (required): The ID of the task list
- `title` (optional): New title for the task


### tasks_update_task_list

Update a task list's title

**Arguments:**
- `account` (optional): Account name (default: 'default'). Used to manage multiple Google accounts.
- `taskListId` (required): The ID of the task list to update
- `title` (required): The new title for the task list


## OAuth Authentication

### google_get_auth_url

Get the OAuth URL to authorize Google services access (Gmail, Docs, Drive) for a specific account

**Arguments:**
- `account` (optional): Account name (default: 'default'). Used to manage multiple Google accounts.


### google_save_auth_code

Save the OAuth authorization code to complete Google services authentication (Gmail, Docs, Drive) for a specific account

**Arguments:**
- `account` (optional): Account name (default: 'default'). Used to manage multiple Google accounts.
- `authCode` (required): The authorization code from Google OAuth


