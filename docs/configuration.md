# Configuration Guide

This document describes how to configure inboxfewer for use with GitHub and Google services.

## GitHub Token

inboxfewer requires a GitHub personal access token to check the status of issues and pull requests.

### Setup

Create a file at `~/keys/github-inboxfewer.token` with two space-separated values:

```
<github-username> <github-personal-access-token>
```

### Example

```
octocat ghp_abcdefghijklmnopqrstuvwxyz123456
```

### Token Permissions

The token needs the following scopes:
- `repo` - Access to repository data
- `read:user` - Read user profile data

### Creating a Token

1. Go to GitHub Settings > Developer settings > Personal access tokens
2. Click "Generate new token"
3. Select the required scopes
4. Copy the generated token
5. Create the file at `~/keys/github-inboxfewer.token`

## Google Services OAuth

inboxfewer uses OAuth 2.0 to authenticate with Google services (Gmail, Google Docs, Google Drive, Google Calendar, Google Meet, Google Tasks).

### First Run

On first run, you'll be prompted to authenticate:

1. The application displays an authorization URL
2. Visit the URL in your browser
3. Grant the requested permissions
4. Copy the authorization code
5. Paste it back into the application

### Token Storage

OAuth tokens are cached per account at:
- Linux/Unix: `~/.cache/inboxfewer/google-{account}.token`
- macOS: `~/Library/Caches/inboxfewer/google-{account}.token`
- Windows: `%TEMP%/inboxfewer/google-{account}.token`

### OAuth Scopes

Each token provides access to the following Google APIs with these scopes:

**Gmail:**
- `https://mail.google.com/` - Full Gmail access (read, compose, send, and permanently delete all your email)
- `https://www.googleapis.com/auth/gmail.settings.basic` - Manage basic mail settings (create/delete filters and labels)

**Google Docs:**
- `https://www.googleapis.com/auth/documents.readonly` - Read document content

**Google Drive:**
- `https://www.googleapis.com/auth/drive` - Full read/write access to files and folders

**Google Contacts:**
- `https://www.googleapis.com/auth/contacts.readonly` - Read personal contacts
- `https://www.googleapis.com/auth/contacts.other.readonly` - Read interaction history
- `https://www.googleapis.com/auth/directory.readonly` - Read directory (Workspace)

**Google Calendar:**
- `https://www.googleapis.com/auth/calendar` - Full calendar access (read, write, share, and permanently delete all calendars)

**Google Meet:**
- `https://www.googleapis.com/auth/meetings.space.readonly` - Read meeting spaces, recordings, and transcripts
- `https://www.googleapis.com/auth/meetings.space.settings` - Edit and see settings for all your Google Meet calls

**Google Tasks:**
- `https://www.googleapis.com/auth/tasks` - Read and write tasks

## Signal Messaging

inboxfewer provides Signal messaging capabilities through integration with signal-cli. This allows AI assistants to send and receive Signal messages.

### Prerequisites

Before using Signal features, you must:

1. Install signal-cli on your system
2. Register your Signal account with signal-cli
3. Verify your account

### Installing signal-cli

Follow the official installation instructions at: https://github.com/AsamK/signal-cli

For most systems:

**macOS (Homebrew):**
```bash
brew install signal-cli
```

**Linux (Manual):**
```bash
# Download the latest release from GitHub
wget https://github.com/AsamK/signal-cli/releases/latest/download/signal-cli-*.tar.gz
tar xf signal-cli-*.tar.gz -C /opt
sudo ln -s /opt/signal-cli-*/bin/signal-cli /usr/local/bin/
```

### Registering Your Signal Account

1. **Register with your phone number:**
```bash
signal-cli -u +15551234567 register
```

2. **Verify with the code received via SMS:**
```bash
signal-cli -u +15551234567 verify CODE_RECEIVED
```

3. **Test the registration:**
```bash
signal-cli -u +15551234567 receive --timeout 5
```

### Configuration

Unlike Google services, Signal does not require OAuth authentication. The signal-cli stores its configuration in:
- Linux/Unix: `~/.local/share/signal-cli/`
- macOS: `~/.local/share/signal-cli/`
- Windows: `%APPDATA%\signal-cli\`

### Using Signal Tools

Signal tools are available through the MCP server. Like other services, they support multi-account configuration:

```javascript
// Send a direct message using default account
signal_send_message({
  recipient: "+15559876543",
  message: "Hello from Signal!"
})

// Send a group message
signal_send_group_message({
  group_name: "Family Chat",
  message: "Meeting at 3pm"
})

// Receive messages with 30 second timeout
signal_receive_message({
  timeout: 30
})

// List all groups
signal_list_groups()
```

### Multi-Account Signal Support

You can manage multiple Signal accounts (phone numbers) by specifying the `account` parameter:

```javascript
// Use work phone number
signal_send_message({
  account: "work",
  recipient: "+15559876543",
  message: "Work message"
})

// Use personal phone number  
signal_send_message({
  account: "personal",
  recipient: "+15551112222",
  message: "Personal message"
})
```

Each account corresponds to a different phone number registered with signal-cli.

### Safety Mode and Signal

Signal tools respect the server's safety mode:

- **Read-only mode (default):** Only `signal_receive_message` and `signal_list_groups` are available
- **Write mode (--yolo):** All Signal tools are available, including `signal_send_message` and `signal_send_group_message`

### Troubleshooting Signal

**signal-cli not found:**
- Ensure signal-cli is installed and in your PATH
- Try running `signal-cli --version` to verify installation

**Phone number not registered:**
- Complete the registration and verification steps
- Verify with `signal-cli -u +15551234567 receive --timeout 5`

**Message sending fails:**
- Check that the recipient's phone number is correct (include country code with +)
- Ensure both parties have Signal installed
- Verify your registration is still valid

**Group not found:**
- Use `signal-cli -u +15551234567 listGroups` to see available groups
- Ensure you're a member of the group
- Group names are case-sensitive

## Multi-Account Support

inboxfewer supports managing multiple Google accounts simultaneously (e.g., work and personal email).

### Account Names

Each account is identified by a unique name that must contain only:
- Alphanumeric characters (a-z, A-Z, 0-9)
- Hyphens (-)
- Underscores (_)

Valid examples: `default`, `work`, `personal`, `work-email`, `personal_gmail`

### Default Account

If no account name is specified, the `default` account is used. Existing users' tokens will be automatically migrated to the `default` account on first run.

### Using Multiple Accounts

#### CLI Mode

Use the `--account` flag:

```bash
# Use default account
inboxfewer cleanup

# Use work account
inboxfewer cleanup --account work

# Use personal account
inboxfewer cleanup --account personal
```

#### MCP Server Mode

In MCP server mode, each tool accepts an optional `account` parameter:

```javascript
// Use default account
gmail_list_threads({query: "in:inbox"})

// Use work account
gmail_list_threads({account: "work", query: "in:inbox"})

// Use personal account
gmail_send_email({
  account: "personal",
  to: "friend@example.com",
  subject: "Hello",
  body: "Hi!"
})
```

### OAuth Authentication per Account

Each account requires separate OAuth authentication:

```bash
# Authenticate default account
inboxfewer serve
# Follow OAuth flow when prompted

# Authenticate work account
# In MCP client, use google_get_auth_url with account: "work"
# Then use google_save_auth_code with account: "work"

# Authenticate personal account
# In MCP client, use google_get_auth_url with account: "personal"
# Then use google_save_auth_code with account: "personal"
```

## MCP Server Configuration

### Claude Desktop

Add to `~/Library/Application Support/Claude/claude_desktop_config.json` (macOS) or `%APPDATA%\Claude\claude_desktop_config.json` (Windows):

```json
{
  "mcpServers": {
    "inboxfewer": {
      "command": "/usr/local/bin/inboxfewer",
      "args": ["serve"]
    }
  }
}
```

For read-write access:

```json
{
  "mcpServers": {
    "inboxfewer": {
      "command": "/usr/local/bin/inboxfewer",
      "args": ["serve", "--yolo"]
    }
  }
}
```

### Other MCP Clients

For SSE transport:

```bash
inboxfewer serve --transport sse --http-addr :8080
```

Configure your MCP client to connect to:
- SSE endpoint: `http://localhost:8080/sse`
- Message endpoint: `http://localhost:8080/message`

For HTTP transport:

```bash
inboxfewer serve --transport streamable-http --http-addr :8080
```

Configure your MCP client to connect to:
- HTTP endpoint: `http://localhost:8080/mcp`

## Safety Mode (Read-Only by Default)

By default, the MCP server operates in read-only mode for AI safety. Only safe, non-destructive operations are available.

### Always Available (Safe Operations)

- List, get, search, and query operations
- Archive threads (safe cleanup)
- Create tasks and task lists (safe planning)
- Create calendar entries (safe scheduling)
- Unsubscribe from emails (safe inbox cleanup)
- Create and delete Gmail filters (safe email organization)
- Create and configure Meet spaces (safe meeting setup)

### Requires --yolo Flag (Write Operations)

- Email sending, replying, and forwarding
- Drive file operations (upload, delete, move, share)
- Calendar event deletion and updates
- Task deletion and updates

### Enabling Write Operations

Use the `--yolo` flag to enable all operations:

```bash
inboxfewer serve --yolo
```

## Troubleshooting

### Token Refresh

OAuth tokens automatically refresh when they expire. If you encounter authentication errors:

1. Delete the token file for the affected account
2. Restart the application
3. Complete the OAuth flow again

### Multiple Account Issues

If you're having trouble with multiple accounts:

1. Verify each account name is unique and valid
2. Check that each account has a separate token file
3. Ensure you're using the correct account name in CLI flags or MCP tool calls

### Permission Denied

If you see "Permission Denied" errors:

1. Verify the OAuth token has the required scopes
2. Re-authenticate to grant additional permissions (see "Reauthorization" below)
3. Check that you've enabled `--yolo` if using write operations

### Reauthorization for New Scopes

When new features are added that require additional OAuth scopes (e.g., Gmail filter management), you'll need to reauthorize:

1. Delete your existing token file:
   ```bash
   # For default account
   rm ~/.cache/inboxfewer/google-default.token
   
   # For specific account
   rm ~/.cache/inboxfewer/google-{account}.token
   ```

2. Restart the application or server

3. Complete the OAuth flow again to grant the new permissions

**Note:** The new authorization will include all previously granted scopes plus any newly added ones.

