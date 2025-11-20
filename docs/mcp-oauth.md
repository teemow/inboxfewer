# MCP OAuth 2.1 Authentication with Google

This document describes the OAuth 2.1 authentication implementation for the MCP server according to the Model Context Protocol (MCP) specification dated 2025-06-18.

## Overview

The inboxfewer MCP server implements secure OAuth 2.1 authentication using **Google as the authorization server**. This architecture provides several benefits:

- **Single Sign-On**: Users authenticate with their existing Google accounts
- **Secure by Design**: The LLM never sees OAuth tokens - they're handled entirely by the MCP client
- **Integrated Access**: The same Google credentials grant both access to the MCP server and Google services (Gmail, Drive, Calendar, etc.)
- **Standards Compliant**: Full compliance with OAuth 2.1, PKCE, and RFC 9728 (Protected Resource Metadata)

## Architecture

### Roles

1. **MCP Server** (inboxfewer): Acts as an **OAuth 2.1 Resource Server**
   - Validates Google OAuth tokens
   - Provides Google services (Gmail, Drive, Calendar, etc.)
   - Protected by Bearer token authentication

2. **Google**: Acts as the **OAuth 2.1 Authorization Server**
   - Issues access tokens
   - Handles user authentication and consent
   - Provides token introspection/validation

3. **MCP Client** (Cursor, Claude Desktop, etc.): Acts as the **OAuth 2.1 Client**
   - Discovers the authorization server via Protected Resource Metadata
   - Handles the OAuth flow (opening browser, PKCE, token exchange)
   - Includes Bearer tokens in requests to the MCP server

4. **LLM**: Never sees tokens
   - Only receives responses from MCP tools
   - No access to sensitive authentication credentials

### Security Flow

According to the MCP specification, the authentication flow is:

```mermaid
sequenceDiagram
    participant Client as MCP Client
    participant Server as MCP Server
    participant Google as Google OAuth
    participant Browser as User Browser

    Client->>Server: Request without token
    Server-->>Client: 401 + WWW-Authenticate header
    
    Client->>Server: GET /.well-known/oauth-protected-resource
    Server-->>Client: Resource metadata (points to Google)
    
    Client->>Google: GET /.well-known/openid-configuration
    Google-->>Client: Authorization server metadata
    
    Client->>Browser: Open Google OAuth consent page
    Browser->>Google: User signs in & grants consent
    Google-->>Browser: Authorization code
    Browser->>Client: Authorization code
    
    Client->>Google: Exchange code for token (with PKCE)
    Google-->>Client: Access token + refresh token
    
    Client->>Server: MCP request with Bearer token
    Server->>Google: Validate token (userinfo endpoint)
    Google-->>Server: User info
    Server-->>Client: MCP response
```

## Components

### 1. Protected Resource Metadata (RFC 9728)

The MCP server provides an endpoint that tells clients where to find the authorization server:

**Endpoint**: `/.well-known/oauth-protected-resource`

**Response**:
```json
{
  "resource": "https://mcp.example.com",
  "authorization_servers": [
    "https://accounts.google.com"
  ],
  "bearer_methods_supported": [
    "header"
  ],
  "scopes_supported": [
    "https://www.googleapis.com/auth/gmail.readonly",
    "https://www.googleapis.com/auth/gmail.modify",
    "https://www.googleapis.com/auth/drive",
    "https://www.googleapis.com/auth/calendar",
    ...
  ]
}
```

### 2. OAuth Middleware

The middleware (`internal/mcp/oauth/middleware.go`):
- Validates Bearer tokens from the `Authorization` header
- Calls Google's userinfo endpoint to validate tokens and extract user identity
- Stores user identity and token in request context
- Caches Google tokens for accessing Google APIs
- Returns 401 with `WWW-Authenticate` header for invalid/missing tokens

### 3. Token Store & Provider

The token store (`internal/mcp/oauth/store.go`):
- Stores validated Google OAuth tokens in memory
- Associates tokens with user email addresses
- Automatic cleanup of expired tokens (runs every minute)
- No custom token generation - all tokens come from Google

The token provider interface (`internal/google/token_provider.go`):
- Abstracts token retrieval for Google API clients
- File-based provider for STDIO transport (reads from `~/.cache/inboxfewer/`)
- OAuth store provider for HTTP/SSE transports (reads from OAuth middleware cache)
- Automatic selection based on transport type

## Authentication for Different Transports

### STDIO Transport (Default)

For STDIO transport (local execution), OAuth is **not used**. Instead, follow the traditional approach:

1. Run the initial setup (if needed) to authenticate with Google
2. Tokens are stored in `~/.cache/inboxfewer/google-{account}.token`
3. The MCP server reads tokens from the filesystem

This follows the MCP specification recommendation: "Implementations using an STDIO transport SHOULD NOT follow this specification, and instead retrieve credentials from the environment."

### HTTP/SSE Transports

For HTTP-based transports (remote servers), OAuth authentication is **required**:

1. **Start server with HTTP transport**:
   ```bash
   inboxfewer serve --transport sse --http-addr :8080
   # or
   inboxfewer serve --transport streamable-http --http-addr :8080
   ```

2. **MCP Client discovers authentication**:
   - Client makes unauthenticated request
   - Server returns `401 Unauthorized` with `WWW-Authenticate` header
   - Client fetches `/.well-known/oauth-protected-resource`
   - Client discovers Google as the authorization server

3. **MCP Client handles OAuth flow**:
   - Client opens browser to Google OAuth consent page
   - User signs in with Google account
   - User grants requested permissions (Gmail, Drive, Calendar, etc.)
   - Client receives authorization code
   - Client exchanges code for access token using PKCE
   - All of this happens **without LLM involvement**

4. **MCP Client makes authenticated requests**:
   ```
   Authorization: Bearer {google_access_token}
   ```

5. **MCP Server validates token**:
   - Calls `https://www.googleapis.com/oauth2/v2/userinfo` with the token
   - Extracts user identity (email, name, etc.)
   - Caches token for accessing Google APIs
   - Processes MCP request

## Required Google OAuth Scopes

The MCP server requests the following Google OAuth scopes:

- `https://www.googleapis.com/auth/gmail.readonly` - Read Gmail messages
- `https://www.googleapis.com/auth/gmail.modify` - Modify Gmail (labels, archive, etc.)
- `https://www.googleapis.com/auth/gmail.send` - Send emails
- `https://www.googleapis.com/auth/gmail.settings.basic` - Manage filters and labels
- `https://www.googleapis.com/auth/documents.readonly` - Read Google Docs
- `https://www.googleapis.com/auth/drive` - Access Google Drive
- `https://www.googleapis.com/auth/calendar` - Manage Google Calendar
- `https://www.googleapis.com/auth/meetings.space.readonly` - Access Google Meet recordings
- `https://www.googleapis.com/auth/tasks` - Manage Google Tasks

## Advanced Features

### Rate Limiting

The MCP server includes built-in rate limiting to protect against abuse:

- **Per-IP Rate Limiting**: Each IP address has its own token bucket
- **Configurable Limits**: Set both rate (requests/second) and burst size
- **Automatic Cleanup**: Inactive rate limiters are removed after 10 minutes
- **Graceful Degradation**: Returns `429 Too Many Requests` with `Retry-After` header

**Configuration**:
```go
RateLimitRate:  10,  // 10 requests per second
RateLimitBurst: 20,  // Allow burst of 20 requests
```

**Rate limit disabled**: Set `RateLimitRate` to `0` to disable rate limiting entirely.

### Token Refresh

The server automatically handles token refresh for long-running sessions:

- **Proactive Refresh**: Tokens are refreshed 5 minutes before expiration
- **Transparent**: Happens automatically during Google API calls
- **Fallback**: If refresh fails, user is prompted to re-authenticate

**How it works**:
1. Google API clients check token expiry before each request
2. If token expires within 5 minutes, refresh is attempted
3. New token is stored in the OAuth store
4. Request proceeds with fresh token

### Token Revocation

Administrators can revoke tokens to force re-authentication:

**Revoke a specific user's token**:
```go
err := oauthHandler.RevokeToken("user@example.com")
```

**Via HTTP endpoint**:
```bash
curl -X POST https://mcp.example.com/oauth/revoke \
  -H "Content-Type: application/json" \
  -d '{"email": "user@example.com"}'
```

**Response**:
```json
{
  "status": "success",
  "message": "Token revoked for user@example.com"
}
```

After revocation, the user must re-authenticate through their MCP client.

### Enhanced Error Messages

The server provides actionable error messages for common issues:

| Error Type | User-Friendly Message |
|------------|----------------------|
| 401 Unauthorized | "Google token is invalid or expired. Please re-authenticate through your MCP client." |
| 403 Forbidden | "Access denied by Google. Please ensure your token has the required scopes." |
| Network errors | "Unable to verify token with Google due to network issues. Please try again." |
| Rate limits | "Google API rate limit exceeded. Please wait a moment and try again." |
| Server errors | "Google authentication service is temporarily unavailable. Please try again later." |

## Security Considerations

### Token Validation

The MCP server validates every Bearer token by:
1. Calling Google's userinfo endpoint: `https://www.googleapis.com/oauth2/v2/userinfo`
2. Verifying the response is successful (HTTP 200)
3. Extracting user identity (email, user ID, name)
4. Automatically refreshing tokens that are close to expiration

Invalid or expired tokens receive a `401 Unauthorized` response with actionable error messages.

### Rate Limiting Protection

Built-in rate limiting protects the server from:
- **Brute force attacks**: Limits authentication attempts per IP
- **Token validation floods**: Prevents excessive validation requests
- **Resource exhaustion**: Protects against DoS attacks

Each IP address gets an independent token bucket with configurable rate and burst limits.

### Token Storage

- **HTTP/SSE Transport**: Google tokens are cached in memory (per user session) by the OAuth middleware
- **STDIO Transport**: Tokens are stored in `~/.cache/inboxfewer/google-{account}.token`
- **No token leakage**: Tokens are never written to server logs or exposed to the LLM
- **Client-side**: The MCP client manages token persistence and refresh for HTTP/SSE
- **Automatic cleanup**: Expired tokens are removed from memory every minute
- **Token provider pattern**: Google API clients automatically use the correct token source based on transport type

### HTTPS Requirement

According to OAuth 2.1 specification:
- **Production**: All OAuth endpoints MUST use HTTPS
- **Development**: Only true loopback addresses (`localhost`, `127.0.0.1`, `::1`) may use HTTP for testing
- **Validation**: Server performs strict URL parsing to prevent bypass attempts (e.g., `localhost.evil.com`)

### PKCE (Proof Key for Code Exchange)

The MCP client MUST use PKCE when exchanging authorization codes for tokens. This prevents authorization code interception attacks.

## Multi-Account Support

The MCP server supports multiple Google accounts:

```javascript
// Use specific account in MCP tool call
{
  "tool": "gmail_list_threads",
  "arguments": {
    "account": "work",
    "query": "in:inbox"
  }
}
```

Each account has its own cached Google token, identified by the account name (email address).

## Configuration

### OAuth Handler Configuration

```go
config := &oauth.Config{
    Resource: "https://mcp.example.com",  // MCP server URL
    SupportedScopes: []string{
        "https://www.googleapis.com/auth/gmail.readonly",
        "https://www.googleapis.com/auth/drive",
        // ... other Google scopes
    },
    
    // Rate Limiting (protects against abuse)
    RateLimitRate:   10,             // 10 requests per second per IP
    RateLimitBurst:  20,             // Allow burst of 20 requests
    
    // Token Management
    CleanupInterval: 1 * time.Minute, // Cleanup expired tokens every minute
}

handler, err := oauth.NewHandler(config)
```

### Integration with HTTP Server

```go
// Create OAuth-enabled HTTP server
oauthServer, err := server.NewOAuthHTTPServer(mcpServer, "sse", baseURL)
if err != nil {
    log.Fatal(err)
}

// Start server
if err := oauthServer.Start(":8080"); err != nil {
    log.Fatal(err)
}
```

The server automatically:
- Serves Protected Resource Metadata at `/.well-known/oauth-protected-resource`
- Wraps MCP endpoints with OAuth middleware
- Returns appropriate `WWW-Authenticate` headers

## Testing

Run tests with:

```bash
make test
```

The OAuth implementation includes tests for:
- Protected Resource Metadata generation
- Token validation (with mocked Google responses)
- Middleware behavior (missing tokens, invalid tokens, etc.)
- Token store operations

## Troubleshooting

### "Missing Authorization header"

**Cause**: MCP client is not sending Bearer token

**Solution**: Ensure your MCP client supports OAuth 2.1 and properly discovers the authorization server via Protected Resource Metadata

### "Token validation failed"

**Cause**: Invalid or expired Google OAuth token

**Solution**: The MCP client should automatically refresh the token. If it doesn't, re-authenticate through the client.

### "Please authenticate with Google through your MCP client"

**Cause**: No cached Google token for the requested account

**Solution**: 
- For HTTP/SSE: The MCP client will automatically initiate OAuth flow
- For STDIO: Run authentication setup or check token files in `~/.cache/inboxfewer/`

## Migrating from Old Authentication

The previous insecure authentication flow (where users pasted auth codes into the LLM) has been **removed** for security reasons.

**Old flow** (insecure, removed):
1. LLM calls `google_get_auth_url` tool → gets URL
2. User visits URL, gets auth code
3. User pastes code back to LLM
4. LLM calls `google_save_auth_code` tool → stores token
5. **Problem**: LLM sees the auth code!

**New flow** (secure):
1. MCP client detects missing authentication
2. Client opens browser with Google OAuth
3. User authenticates with Google
4. Client receives and stores token
5. Client includes token in requests to MCP server
6. **LLM never sees any tokens!**

## References

- [MCP Specification - Authorization](https://modelcontextprotocol.io/specification/2025-06-18/basic/authorization)
- [OAuth 2.1](https://datatracker.ietf.org/doc/html/draft-ietf-oauth-v2-1-13)
- [RFC 9728 - OAuth 2.0 Protected Resource Metadata](https://datatracker.ietf.org/doc/html/rfc9728)
- [RFC 8707 - Resource Indicators for OAuth 2.0](https://datatracker.ietf.org/doc/html/rfc8707)
- [RFC 7636 - Proof Key for Code Exchange (PKCE)](https://datatracker.ietf.org/doc/html/rfc7636)
- [Google OAuth 2.0](https://developers.google.com/identity/protocols/oauth2)
