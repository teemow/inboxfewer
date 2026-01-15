# MCP OAuth 2.1 Authentication with Google

This document describes the OAuth 2.1 authentication implementation for the MCP server according to the Model Context Protocol (MCP) specification dated 2025-06-18.

## Overview

The inboxfewer MCP server implements secure OAuth 2.1 authentication using an **OAuth Proxy architecture**. Inboxfewer acts as both an **OAuth Authorization Server** (for MCP clients) and an **OAuth Resource Server** (for Google services).

### Key Benefits

- **No Client Credentials Required**: MCP clients use Dynamic Client Registration (RFC 7591) - no need to create Google OAuth credentials
- **Single Sign-On**: Users authenticate with their existing Google accounts
- **Secure by Design**: The LLM never sees OAuth tokens - they're handled entirely by the MCP client
- **Integrated Access**: The same authentication flow grants access to both the MCP server and Google services (Gmail, Drive, Calendar, etc.)
- **Standards Compliant**: Full compliance with OAuth 2.1, PKCE, RFC 7591 (DCR), RFC 8414, and RFC 9728

## Architecture

### OAuth Proxy Pattern

Inboxfewer uses an OAuth Proxy architecture where:

1. **MCP Clients** authenticate with **inboxfewer** (not directly with Google)
2. **Inboxfewer** proxies the authentication to **Google**
3. **Inboxfewer** issues its own access tokens to MCP clients (backed by Google tokens)
4. **Google tokens** are managed internally by inboxfewer

This eliminates the need for every MCP client to have Google OAuth credentials.

### Roles

1. **MCP Client** (mcp-debug, Cursor, Claude Desktop, etc.): **OAuth 2.1 Client**
   - Connects to inboxfewer's OAuth server
   - Uses Dynamic Client Registration (no pre-configured credentials needed)
   - Receives inboxfewer access tokens
   - Includes Bearer tokens in MCP requests

2. **Inboxfewer MCP Server**: **OAuth 2.1 Authorization Server + Resource Server**
   - **As Authorization Server:**
     - Provides Dynamic Client Registration (RFC 7591)
     - Provides Authorization Server Metadata (RFC 8414)
     - Proxies authorization to Google
     - Issues inboxfewer access tokens
   - **As Resource Server:**
     - Validates inboxfewer Bearer tokens
     - Maps inboxfewer tokens to Google tokens
     - Provides Google services (Gmail, Drive, Calendar, etc.)

3. **Google**: **Upstream Identity Provider**
   - Authenticates users
   - Issues Google OAuth tokens to inboxfewer
   - Provides access to Google APIs

4. **LLM**: Never sees tokens
   - Only receives responses from MCP tools
   - No access to sensitive authentication credentials

### OAuth Proxy Flow

The complete authentication flow with the OAuth proxy:

```mermaid
sequenceDiagram
    participant Client as MCP Client
    participant Inboxfewer as Inboxfewer<br/>(OAuth Proxy)
    participant Google as Google OAuth
    participant Browser as User Browser

    Note over Client,Google: 1. Discovery Phase
    Client->>Inboxfewer: MCP request (no token)
    Inboxfewer-->>Client: 401 + WWW-Authenticate header
    
    Client->>Inboxfewer: GET /.well-known/oauth-protected-resource
    Inboxfewer-->>Client: Resource metadata (points to inboxfewer)
    
    Client->>Inboxfewer: GET /.well-known/oauth-authorization-server
    Inboxfewer-->>Client: Authorization server metadata + DCR endpoint
    
    Note over Client,Google: 2. Dynamic Client Registration
    Client->>Inboxfewer: POST /oauth/register
    Inboxfewer-->>Client: client_id + client_secret
    
    Note over Client,Google: 3. Authorization Flow (Proxied to Google)
    Client->>Inboxfewer: GET /oauth/authorize (with PKCE)
    Inboxfewer->>Google: Redirect to Google OAuth
    Browser->>Google: User signs in & grants consent
    Google-->>Browser: Google authorization code
    Browser->>Inboxfewer: Google authorization code
    Inboxfewer->>Google: Exchange code for Google token
    Google-->>Inboxfewer: Google access token + refresh token
    Inboxfewer->>Inboxfewer: Generate inboxfewer authorization code
    Inboxfewer-->>Client: Inboxfewer authorization code
    
    Note over Client,Google: 4. Token Exchange
    Client->>Inboxfewer: POST /oauth/token (code + PKCE verifier)
    Inboxfewer->>Inboxfewer: Validate PKCE + map tokens
    Inboxfewer-->>Client: Inboxfewer access token
    
    Note over Client,Google: 5. Authenticated Requests
    Client->>Inboxfewer: MCP request + inboxfewer Bearer token
    Inboxfewer->>Inboxfewer: Map inboxfewer token to Google token
    Inboxfewer->>Google: API request with Google token
    Google-->>Inboxfewer: API response
    Inboxfewer-->>Client: MCP response
```

## Quick Start: Connecting with mcp-debug

With the OAuth proxy architecture, connecting to inboxfewer is simple - **no Google OAuth credentials needed**:

```bash
# Connect with Dynamic Client Registration (recommended)
mcp-debug --oauth --endpoint https://inboxfewer.example.com/mcp --repl

# With OIDC features
mcp-debug --oauth --oauth-oidc --endpoint https://inboxfewer.example.com/mcp --repl
```

The OAuth proxy handles everything:
1. ✅ **Automatic client registration** - no need to create OAuth credentials
2. ✅ **Browser-based authentication** - opens Google OAuth in your browser
3. ✅ **Secure token management** - tokens are handled by mcp-debug, never exposed to LLM
4. ✅ **Automatic token refresh** - seamless experience

### What Happens Behind the Scenes

When you run the command above:

1. **mcp-debug** connects to inboxfewer and discovers it needs OAuth
2. **Dynamic registration**: mcp-debug automatically registers with inboxfewer (gets client_id/secret)
3. **Authorization**: mcp-debug opens your browser to Google OAuth
4. **You authenticate** with your Google account
5. **Google redirects** back to inboxfewer with authorization
6. **Inboxfewer** exchanges with Google and issues inboxfewer tokens
7. **mcp-debug** receives inboxfewer tokens and uses them for all MCP requests

**You're done!** No manual OAuth setup, no credential management, no token handling.

## OAuth Proxy Components

### 1. Protected Resource Metadata (RFC 9728)

The MCP server provides an endpoint that tells clients where to find the authorization server (inboxfewer):

**Endpoint**: `/.well-known/oauth-protected-resource`

**Response**:
```json
{
  "resource": "https://inboxfewer.example.com",
  "authorization_servers": [
    "https://inboxfewer.example.com"
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

**Key Difference**: `authorization_servers` now points to **inboxfewer** (not Google), enabling the OAuth proxy pattern.

### 2. Authorization Server Metadata (RFC 8414)

Inboxfewer provides OAuth Authorization Server metadata:

**Endpoint**: `/.well-known/oauth-authorization-server`

**Response**:
```json
{
  "issuer": "https://inboxfewer.example.com",
  "authorization_endpoint": "https://inboxfewer.example.com/oauth/authorize",
  "token_endpoint": "https://inboxfewer.example.com/oauth/token",
  "registration_endpoint": "https://inboxfewer.example.com/oauth/register",
  "scopes_supported": [
    "https://www.googleapis.com/auth/gmail.readonly",
    ...
  ],
  "response_types_supported": ["code"],
  "grant_types_supported": ["authorization_code", "refresh_token"],
  "token_endpoint_auth_methods_supported": [
    "client_secret_basic",
    "client_secret_post",
    "none"
  ],
  "code_challenge_methods_supported": ["S256", "plain"]
}
```

### 3. Dynamic Client Registration (RFC 7591)

MCP clients can register dynamically without pre-configured credentials:

**Endpoint**: `POST /oauth/register`

**Request**:
```json
{
  "redirect_uris": ["http://localhost:8765/callback"],
  "client_name": "mcp-debug",
  "grant_types": ["authorization_code"],
  "response_types": ["code"]
}
```

**Response**:
```json
{
  "client_id": "abc123...",
  "client_secret": "secret456...",
  "client_id_issued_at": 1700000000,
  "client_secret_expires_at": 0,
  "redirect_uris": ["http://localhost:8765/callback"],
  "client_name": "mcp-debug",
  "grant_types": ["authorization_code", "refresh_token"],
  "response_types": ["code"],
  "token_endpoint_auth_method": "client_secret_basic"
}
```

**Security**: Client secrets are bcrypt-hashed for storage.

### 4. Authorization Endpoint

MCP clients redirect users here to start the OAuth flow:

**Endpoint**: `GET /oauth/authorize`

**Parameters**:
- `client_id`: Client ID from registration
- `redirect_uri`: Where to redirect after authorization
- `state`: Client state for CSRF protection
- `scope`: Requested Google scopes
- `code_challenge`: PKCE code challenge (S256 or plain)
- `code_challenge_method`: PKCE method (S256 recommended)

**Flow**:
1. Validates client_id and redirect_uri
2. Redirects to Google OAuth for user authentication
3. Google redirects back to `/oauth/google/callback`
4. Inboxfewer exchanges Google code for Google tokens
5. Generates inboxfewer authorization code
6. Redirects to client's redirect_uri with inboxfewer code

### 5. Token Endpoint

MCP clients exchange authorization codes for access tokens:

**Endpoint**: `POST /oauth/token`

**Request** (authorization_code grant):
```
grant_type=authorization_code
&code=<authorization_code>
&redirect_uri=<same_as_authorization>
&client_id=<client_id>
&code_verifier=<pkce_verifier>
```

**Response**:
```json
{
  "access_token": "inboxfewer_token_xyz...",
  "token_type": "Bearer",
  "expires_in": 3600,
  "scope": "https://www.googleapis.com/auth/gmail.readonly ..."
}
```

**PKCE Validation**: The server validates the `code_verifier` against the `code_challenge` from the authorization request.

### 6. Google OAuth Callback

Internal endpoint where Google redirects after user authentication:

**Endpoint**: `GET /oauth/google/callback`

This endpoint:
1. Receives Google authorization code
2. Exchanges it for Google access + refresh tokens
3. Validates tokens with Google's userinfo endpoint
4. Stores Google tokens internally
5. Generates inboxfewer authorization code
6. Redirects back to the MCP client

### 7. Token Validation Middleware

The middleware (`internal/mcp/oauth/middleware.go`) validates inboxfewer Bearer tokens:

**Token Resolution**:
1. Extract Bearer token from `Authorization` header
2. Check if it's an inboxfewer token (look up in token store)
3. If found, retrieve mapped Google token
4. If not found, try validating directly with Google (backward compatibility)
5. Validate Google token with Google's userinfo endpoint
6. Store user info in request context

**Token Mapping**:
- Inboxfewer tokens → Google tokens (bidirectional mapping)
- User email → Google tokens (for API access)
- Automatic token refresh when Google tokens expire

**Error Handling**:
- Returns 401 with `WWW-Authenticate` header for invalid/missing tokens
- Provides actionable error messages for common failure scenarios

### 3. Token Store & Provider

The token store (`internal/mcp/oauth/store.go`):
- Stores validated Google OAuth tokens in memory
- Associates tokens with user email addresses
- Automatic cleanup of expired tokens (runs every minute)
- No custom token generation - all tokens come from Google

The token provider interface (`internal/google/token_provider.go`):
- Abstracts token retrieval for Google API clients
- File-based provider for STDIO transport (reads from `~/.cache/inboxfewer/`)
- OAuth store provider for HTTP transport (reads from OAuth middleware cache)
- Automatic selection based on transport type

## Authentication for Different Transports

### STDIO Transport (Default)

For STDIO transport (local execution), OAuth is **not used**. Instead, follow the traditional approach:

1. Run the initial setup (if needed) to authenticate with Google
2. Tokens are stored in `~/.cache/inboxfewer/google-{account}.token`
3. The MCP server reads tokens from the filesystem

This follows the MCP specification recommendation: "Implementations using an STDIO transport SHOULD NOT follow this specification, and instead retrieve credentials from the environment."

### HTTP Transport (OAuth Proxy Mode)

For HTTP-based transport (remote servers), OAuth authentication is **required**. Inboxfewer runs as an **OAuth proxy**, so clients don't need Google OAuth credentials:

1. **Start server with HTTP transport**:
   ```bash
   # Development (localhost) - base URL auto-detected
   inboxfewer serve --transport streamable-http --http-addr :8080 \
     --google-client-id "your-id.apps.googleusercontent.com" \
     --google-client-secret "your-secret"
   
   # Production (deployed instance) - MUST specify base URL
   inboxfewer serve --transport streamable-http --http-addr :8080 \
     --base-url "https://inboxfewer.example.com" \
     --google-client-id "your-id.apps.googleusercontent.com" \
     --google-client-secret "your-secret"
   
   # Or use environment variables (recommended for containers/K8s)
   export MCP_BASE_URL="https://inboxfewer.example.com"
   export GOOGLE_CLIENT_ID="your-id.apps.googleusercontent.com"
   export GOOGLE_CLIENT_SECRET="your-secret"
   inboxfewer serve --transport streamable-http --http-addr :8080
   ```
   
   **⚠️ Critical Configuration:**
   - **Base URL**: For deployed instances (Kubernetes, Docker, cloud), you **MUST** set `--base-url` or `MCP_BASE_URL` to the public URL where clients connect
   - **Google OAuth Credentials**: Required for OAuth proxy mode (single set of credentials for all clients)
   - **Redirect URL**: Automatically configured as `{base-url}/oauth/google/callback`

2. **MCP Client discovers OAuth proxy**:
   - Client makes unauthenticated request
   - Server returns `401 Unauthorized` with `WWW-Authenticate` header
   - Client fetches `/.well-known/oauth-protected-resource`
   - **Client discovers inboxfewer as the authorization server** (not Google!)
   - Client fetches `/.well-known/oauth-authorization-server`
   - Client discovers Dynamic Client Registration endpoint

3. **MCP Client registers dynamically**:
   - Client POSTs to `/oauth/register` with redirect_uri
   - Inboxfewer returns `client_id` and `client_secret`
   - **No pre-configured credentials needed!**

4. **MCP Client handles OAuth flow**:
   - Client opens browser to **inboxfewer's** `/oauth/authorize`
   - Inboxfewer redirects to Google OAuth consent page
   - User signs in with Google account and grants permissions
   - Google redirects back to **inboxfewer** with authorization code
   - Inboxfewer exchanges Google code for Google tokens
   - Inboxfewer generates inboxfewer authorization code
   - Client redirected back with inboxfewer authorization code
   - Client exchanges inboxfewer code for **inboxfewer access token** (using PKCE)
   - All of this happens **without LLM involvement**

5. **MCP Client makes authenticated requests**:
   ```
   Authorization: Bearer {inboxfewer_access_token}
   ```

6. **Inboxfewer validates and maps tokens**:
   - Looks up inboxfewer token in token store
   - Retrieves mapped Google token
   - Uses Google token to access Google APIs
   - Processes MCP request
   - Returns response to client

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

- **HTTP Transport**: Google tokens are cached in memory (per user session) by the OAuth middleware
- **STDIO Transport**: Tokens are stored in `~/.cache/inboxfewer/google-{account}.token`
- **No token leakage**: Tokens are never written to server logs or exposed to the LLM
- **Client-side**: The MCP client manages token persistence and refresh for HTTP transport
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

**⚠️ CRITICAL for Deployed Instances:** The `Resource` field MUST match the public URL where clients connect. For deployed instances, use the `--base-url` flag or `MCP_BASE_URL` environment variable.

```go
config := &oauth.Config{
    Resource: "https://mcp.example.com",  // MCP server URL (MUST match public URL!)
    
    SupportedScopes: []string{
        "https://www.googleapis.com/auth/gmail.readonly",
        "https://www.googleapis.com/auth/gmail.modify",
        "https://www.googleapis.com/auth/drive",
        // ... other Google scopes
    },
    
    // Google OAuth Credentials (REQUIRED for OAuth proxy mode)
    // These credentials are used by inboxfewer to authenticate with Google
    // on behalf of all MCP clients (OAuth proxy pattern).
    // 
    // MCP clients do NOT need their own Google credentials - they register
    // dynamically with inboxfewer using RFC 7591 (Dynamic Client Registration).
    // 
    // Setup instructions:
    // 1. Create a Google Cloud Project at https://console.cloud.google.com
    // 2. Enable the APIs you need (Gmail, Drive, Calendar, etc.)
    // 3. Create OAuth 2.0 credentials (Web application type)
    // 4. Add authorized redirect URI: https://mcp.example.com/oauth/google/callback
    // 5. Use the Client ID and Client Secret here
    GoogleClientID:     "your-client-id.apps.googleusercontent.com",
    GoogleClientSecret: "your-client-secret",
    
    // Google Redirect URL (OPTIONAL - auto-configured if not provided)
    // Default: {Resource}/oauth/google/callback
    GoogleRedirectURL: "https://mcp.example.com/oauth/google/callback",
    
    // Rate Limiting (protects against abuse)
    RateLimitRate:              10,             // 10 requests per second per IP
    RateLimitBurst:             20,             // Allow burst of 20 requests
    RateLimitCleanupInterval:   5 * time.Minute, // Cleanup inactive rate limiters
    
    // Security (defaults shown - all optional)
    TrustProxy:      false,          // Only trust proxy headers if behind trusted proxy (secure by default)
    
    // Token Management
    CleanupInterval: 1 * time.Minute, // Cleanup expired tokens every minute
    
    // Logging (optional)
    Logger: slog.Default(),          // Structured logging
}

handler, err := oauth.NewHandler(config)

// Check if token refresh is enabled
if handler.CanRefreshTokens() {
    log.Info("Token refresh is enabled - tokens will be automatically refreshed before expiration")
} else {
    log.Warn("Token refresh is disabled - users will need to re-authenticate when tokens expire")
}
```

### Security Configuration (OAuth 2.1 Hardening)

The OAuth handler is **secure by default** with enterprise-grade security features. Each security feature can be optionally disabled for backward compatibility with weaker client implementations (not recommended for production).

#### Security Features Overview

| Feature | Default | Security Impact | Disable At Your Own Risk |
|---------|---------|-----------------|--------------------------|
| **Mandatory State Parameter** | ✅ Enabled | Prevents CSRF attacks | `AllowInsecureAuthWithoutState = true` |
| **Refresh Token Rotation** | ✅ Enabled | Prevents token replay attacks | `DisableRefreshTokenRotation = true` |
| **Authenticated Client Registration** | ✅ Enabled | Prevents DoS via mass registration | `AllowPublicClientRegistration = true` |
| **Refresh Token Expiry** | ✅ 90 days | Limits stolen token lifetime | `RefreshTokenTTL = 0` (never expire) |
| **Custom URI Scheme Validation** | ✅ Enabled | Blocks dangerous redirect schemes | `AllowCustomRedirectSchemes = false` |
| **Per-IP Client Limits** | ✅ 10 clients | Prevents registration DoS | `MaxClientsPerIP = 0` (unlimited) |
| **Per-User Rate Limiting** | ⚙️  Optional | Prevents user-based abuse | `UserRateLimitRate > 0` to enable |
| **CIMD Private IP Protection** | ✅ Enabled | Blocks private IP CIMD metadata | `CIMDAllowPrivateIPs = true` to disable |

#### Complete Security Configuration Example

```go
config := &oauth.Config{
    Resource: "https://mcp.example.com",
    
    // ============================================================
    // Security Configuration (Secure by Default)
    // ============================================================
    
    // 1. CSRF Protection (OAuth 2.1 Requirement)
    // ⚠️  WARNING: Disabling this weakens CSRF protection
    // State parameter is REQUIRED for security unless you know what you're doing
    AllowInsecureAuthWithoutState: false,  // DEFAULT: false (state is REQUIRED)
    
    // 2. Refresh Token Rotation (OAuth 2.1 Best Practice)
    // ⚠️  WARNING: Disabling this violates OAuth 2.1 security guidelines
    // Without rotation, stolen refresh tokens can be used indefinitely
    DisableRefreshTokenRotation: false,    // DEFAULT: false (rotation ENABLED)
    
    // 3. Authenticated Dynamic Client Registration (DoS Protection)
    // ⚠️  WARNING: Enabling public registration allows unlimited client creation
    // This can lead to DoS attacks via mass registration
    AllowPublicClientRegistration: false,  // DEFAULT: false (auth REQUIRED)
    RegistrationAccessToken: "your-secure-random-token-here",  // Required if AllowPublicClientRegistration = false
    
    // 4. Refresh Token Expiry (Security vs Usability Balance)
    // Recommended: 30-90 days for balance between security and user experience
    // Set to 0 to disable expiry (not recommended)
    RefreshTokenTTL: 90 * 24 * time.Hour,  // DEFAULT: 90 days
    
    // 5. Per-IP Client Registration Limits (DoS Protection)
    // Prevents attackers from registering unlimited clients from same IP
    // Set to 0 to disable (not recommended)
    MaxClientsPerIP: 10,                   // DEFAULT: 10 clients per IP
    
    // 6. Custom Redirect URI Scheme Validation (Injection Protection)
    // When enabled, custom schemes (myapp://) are validated against patterns
    // Dangerous schemes (javascript:, data:, file:) are always blocked
    AllowCustomRedirectSchemes: true,      // DEFAULT: true (for native apps)
    AllowedCustomSchemes: []string{
        "^[a-z][a-z0-9+.-]*$",            // DEFAULT: RFC 3986 compliant schemes
    },
    
    // 7. Per-User Rate Limiting (After Authentication)
    // Protects against authenticated users abusing the system
    // Set to 0 to disable (default)
    UserRateLimitRate: 100,                // OPTIONAL: 100 requests per second per user
    UserRateLimitBurst: 200,               // OPTIONAL: Allow burst of 200 requests
    
    // ... other config ...
}
```

#### Security Feature Details

##### 1. Mandatory State Parameter (CSRF Protection)

**Default:** `AllowInsecureAuthWithoutState = false` (state is REQUIRED)

The `state` parameter is a critical CSRF protection mechanism in OAuth 2.1. When enabled (default), the authorization server **rejects** authorization requests without a `state` parameter.

**To disable (NOT recommended):**
```go
AllowInsecureAuthWithoutState: true  // ⚠️  SECURITY RISK: CSRF attacks possible
```

**When you might need to disable:**
- Working with broken OAuth clients that don't support `state`
- This is a **client bug** and should be fixed at the client level

**Security impact when disabled:**
- Attackers can trick users into authorizing access to attacker's application
- Cross-Site Request Forgery (CSRF) attacks become possible

---

##### 2. Refresh Token Rotation (OAuth 2.1 Compliance)

**Default:** `DisableRefreshTokenRotation = false` (rotation is ENABLED)

OAuth 2.1 requires refresh token rotation. When enabled (default), the server issues a **new refresh token** on every refresh and **invalidates the old one**.

**To disable (NOT recommended):**
```go
DisableRefreshTokenRotation: true  // ⚠️  SECURITY RISK: Token replay attacks possible
```

**When you might need to disable:**
- Testing legacy clients that don't handle token rotation
- This violates OAuth 2.1 specification

**Security impact when disabled:**
- Stolen refresh tokens can be used indefinitely
- No detection of token theft
- Attackers can refresh tokens even after user changes password

---

##### 3. Authenticated Dynamic Client Registration (DoS Protection)

**Default:** `AllowPublicClientRegistration = false` (authentication REQUIRED)

Dynamic Client Registration (RFC 7591) is protected by requiring a **registration access token**. This prevents attackers from registering unlimited clients.

**To enable public registration (NOT recommended):**
```go
AllowPublicClientRegistration: true  // ⚠️  SECURITY RISK: DoS via mass registration
```

**Setting up authenticated registration (recommended):**
```go
AllowPublicClientRegistration: false
RegistrationAccessToken: "your-long-random-token-here"  // Generate with: openssl rand -base64 48
```

Clients must include the token when registering:
```bash
curl -X POST https://mcp.example.com/oauth/register \
  -H "Authorization: Bearer your-long-random-token-here" \
  -H "Content-Type: application/json" \
  -d '{"redirect_uris": ["http://localhost:8080/callback"]}'
```

**When you might need public registration:**
- Development/testing environments
- Internal networks with other security controls

**Security impact when enabled:**
- Anyone can register unlimited OAuth clients
- Potential memory exhaustion via mass registration
- Resource exhaustion attacks

---

##### 4. Refresh Token Expiry (Time-Limited Tokens)

**Default:** `RefreshTokenTTL = 90 * 24 * time.Hour` (90 days)

Refresh tokens automatically expire after the configured TTL. This limits the window of opportunity for stolen tokens.

**To disable expiry (NOT recommended):**
```go
RefreshTokenTTL: 0  // ⚠️  SECURITY RISK: Tokens never expire
```

**Recommended values:**
- **High security:** 30 days
- **Balanced:** 90 days (default)
- **User convenience:** 180 days

**Security impact when disabled:**
- Stolen refresh tokens are valid forever
- No forced re-authentication
- Compromised tokens remain active indefinitely

---

##### 5. Per-IP Client Registration Limits (DoS Protection)

**Default:** `MaxClientsPerIP = 10`

Limits the number of OAuth clients that can be registered from a single IP address.

**To disable (NOT recommended):**
```go
MaxClientsPerIP: 0  // ⚠️  SECURITY RISK: Unlimited client registration
```

**Security impact when disabled:**
- Attackers can register unlimited clients from same IP
- Memory exhaustion possible
- Denial of Service via resource consumption

---

##### 6. Custom Redirect URI Scheme Validation (Injection Protection)

**Default:** `AllowCustomRedirectSchemes = true` with RFC 3986 validation

Custom URI schemes (e.g., `myapp://callback`) are validated against security patterns. Dangerous schemes are always blocked.

**Always blocked schemes:** `javascript:`, `data:`, `file:`, `vbscript:`, `about:`

**To disable custom schemes (most restrictive):**
```go
AllowCustomRedirectSchemes: false  // Only http/https allowed
```

**To customize allowed patterns:**
```go
AllowCustomRedirectSchemes: true
AllowedCustomSchemes: []string{
    "^myapp$",           // Exact match: myapp://
    "^com\\.example\\.", // Pattern: com.example.*://
}
```

**Security impact:**
- Custom schemes validated against RFC 3986
- Dangerous schemes blocked by default
- XSS and redirect attacks prevented

---

##### 7. Per-User Rate Limiting (Authenticated Abuse Protection)

**Default:** Disabled (no per-user limits)

Optional feature to rate-limit authenticated users independently of IP-based limits.

**To enable:**
```go
UserRateLimitRate: 100,   // 100 requests per second per user
UserRateLimitBurst: 200,  // Allow burst of 200 requests
```

**Use cases:**
- Multi-tenant deployments
- Preventing authenticated user abuse
- Fair usage enforcement

**When to enable:**
- Production deployments with multiple users
- When IP-based limiting isn't sufficient (NAT, VPNs)
- Enforcing API quotas per user

---

##### 8. CIMD Private IP Protection (SSRF Protection)

**Default:** `CIMDAllowPrivateIPs = false` (private IPs BLOCKED)

Client ID Metadata Documents (CIMD) allow clients to use HTTPS URLs as client identifiers. When fetching metadata from these URLs, the server validates that the URL does not resolve to private IP addresses.

**To allow private IPs (NOT recommended for public deployments):**
```go
CIMDAllowPrivateIPs: true  // WARNING: Reduces SSRF protection
```

**When you might need to enable:**
- Home lab deployments where MCP servers are on the same internal network
- Air-gapped environments
- Internal enterprise networks with MCP aggregators
- Any deployment where MCP servers communicate over private networks

**Example scenario:**
```
muster.k8s-internal.home.example.com -> 192.168.1.231
```

Without this option, the server would block fetching metadata from this URL because `192.168.1.231` is a private IP address (RFC 1918).

**Security impact when enabled:**
- SSRF protection is reduced
- Server will make HTTP requests to internal network addresses
- Only enable for deployments that are not exposed to the public internet
- PKCE and other OAuth security measures still apply

**CLI/Environment Configuration:**
```bash
# CLI flag
inboxfewer serve --cimd-allow-private-ips

# Environment variable
CIMD_ALLOW_PRIVATE_IPS=true
```

**Helm values:**
```yaml
oauthSecurity:
  cimd:
    allowPrivateIPs: true
```

---

#### Security Warnings and Logging

When security features are disabled, the OAuth handler logs **warning messages** at startup:

```
⚠️  SECURITY WARNING: State parameter is OPTIONAL (CSRF protection weakened)
    recommendation: Set AllowInsecureAuthWithoutState=false for production

⚠️  SECURITY WARNING: Refresh token rotation is DISABLED
    recommendation: Set DisableRefreshTokenRotation=false for production

⚠️  SECURITY WARNING: Public client registration is ENABLED (DoS risk)
    recommendation: Set AllowPublicClientRegistration=false and use RegistrationAccessToken
```

These warnings help developers identify security misconfigurations before deploying to production.

---

#### Security Best Practices

1. **Never disable security features in production** without understanding the risks
2. **Use authenticated client registration** with a strong registration token
3. **Set reasonable refresh token TTL** (30-90 days recommended)
4. **Enable per-user rate limiting** for multi-tenant deployments
5. **Monitor security warnings** in logs during development
6. **Regularly rotate** the `RegistrationAccessToken`
7. **Audit registered clients** periodically and remove unused ones

---

### Token Refresh Requirements

**Google OAuth credentials are REQUIRED for OAuth proxy mode.**

#### OAuth Proxy Mode (Default for HTTP Transport)
- **Requires:** `GoogleClientID` and `GoogleClientSecret` configuration
- Inboxfewer uses these credentials to authenticate with Google on behalf of all MCP clients
- MCP clients do NOT need their own Google credentials
- MCP clients register dynamically using RFC 7591 (Dynamic Client Registration)
- Tokens are automatically refreshed 5 minutes before expiration
- Seamless user experience with no interruptions
- Suitable for: all HTTP deployments (development, production)

#### Direct Google OAuth (Backward Compatibility)
- If a client sends a Google token directly (not an inboxfewer token)
- Inboxfewer will validate it with Google and accept it
- Maintains backward compatibility with older clients
- Not recommended for new deployments

#### How to Enable Token Refresh

1. **Create Google Cloud Project:**
   - Visit https://console.cloud.google.com
   - Create a new project or select existing one

2. **Enable APIs:**
   - Navigate to "APIs & Services" > "Library"
   - Enable: Gmail API, Drive API, Calendar API, etc.

3. **Create OAuth 2.0 Credentials:**
   - Go to "APIs & Services" > "Credentials"
   - Click "Create Credentials" > "OAuth 2.0 Client ID"
   - Application type: "Web application"
   - Add authorized redirect URIs (your MCP server endpoints)
   - Note the Client ID and Client Secret

4. **Configure Handler:**
   ```go
   config := &oauth.Config{
       Resource:           "https://mcp.example.com",
       GoogleClientID:     "YOUR_CLIENT_ID.apps.googleusercontent.com",
       GoogleClientSecret: "YOUR_CLIENT_SECRET",
       // ... other config
   }
   ```

5. **Verify:**
   ```go
   handler, _ := oauth.NewHandler(config)
   if !handler.CanRefreshTokens() {
       log.Fatal("Token refresh not enabled - check credentials")
   }
   ```

#### Token Refresh Behavior

- **Automatic:** Middleware checks token expiry on every request
- **Proactive:** Refreshes tokens 5 minutes before they expire
- **Transparent:** Users never see "token expired" errors
- **Fallback:** If refresh fails, returns 401 with actionable error message
- **Logging:** All refresh attempts are logged for debugging

### Integration with HTTP Server

```go
// Create OAuth-enabled HTTP server
oauthServer, err := server.NewOAuthHTTPServer(mcpServer, "streamable-http", baseURL)
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
- For HTTP: The MCP client will automatically initiate OAuth flow
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

## OAuth Proxy Benefits

The OAuth proxy architecture provides significant advantages over the traditional approach:

### For Users

- ✅ **Zero Setup**: No need to create Google OAuth credentials
- ✅ **Single Authentication**: One login flow works everywhere
- ✅ **Automatic Registration**: MCP clients register themselves dynamically
- ✅ **Better Security**: Client credentials are ephemeral and client-specific
- ✅ **Seamless Experience**: Same workflow as any OAuth-protected service

### For Administrators

- ✅ **Centralized Management**: One set of Google OAuth credentials for all clients
- ✅ **Better Control**: Monitor and revoke client registrations
- ✅ **Scalability**: No quota limits from Google per-client
- ✅ **Standards Compliant**: Full RFC compliance (7591, 8414, 9728)
- ✅ **Audit Trail**: All OAuth flows logged centrally

### For Developers

- ✅ **Standard Protocol**: Works with any OAuth 2.1 client
- ✅ **No Custom Code**: Use existing OAuth libraries (like mcp-go)
- ✅ **PKCE Support**: Enhanced security for public clients
- ✅ **Token Refresh**: Automatic token refresh handled by inboxfewer

### Architecture Comparison

**Before (Direct Google OAuth)**:
```
MCP Client --[Google Token]--> Inboxfewer --[Same Token]--> Google APIs
          ↑
    Each client needs Google OAuth credentials
```

**After (OAuth Proxy)**:
```
MCP Client --[Inboxfewer Token]--> Inboxfewer --[Google Token]--> Google APIs
          ↑                              ↑
    Dynamic Client Registration     Single Google OAuth app
```

## 🎉 Recent Improvements

### OAuth Proxy Architecture (Latest)

**Major architectural improvement**: Implemented OAuth proxy pattern with Dynamic Client Registration

1. **OAuth Authorization Server**:
   - Added Dynamic Client Registration (RFC 7591) at `/oauth/register`
   - Added Authorization Server Metadata (RFC 8414) at `/.well-known/oauth-authorization-server`
   - Added OAuth authorization endpoint at `/oauth/authorize`
   - Added OAuth token endpoint at `/oauth/token`
   - Added Google OAuth callback at `/oauth/google/callback`

2. **Token Management**:
   - Client registration store with bcrypt-hashed secrets
   - OAuth flow state management with automatic cleanup
   - Inboxfewer token ↔ Google token mapping
   - PKCE support (S256 and plain methods)
   - Automatic token expiration and cleanup

3. **Client Experience**:
   - **Zero configuration**: No Google OAuth credentials needed
   - **Automatic registration**: `mcp-debug --oauth --endpoint <url>` just works
   - **Standard compliant**: Works with any OAuth 2.1 client
   - **Secure**: PKCE required for public clients (OAuth 2.1 requirement)

4. **Testing & Quality**:
   - Comprehensive test suite (100+ tests)
   - Tests for client registration, OAuth flows, PKCE validation
   - Tests for token mapping and validation
   - All tests passing ✅

5. **Documentation**:
   - Updated architecture diagrams
   - Quick start guide for mcp-debug
   - Component documentation
   - Security best practices

### OAuth Token Provider Integration Fix (Latest)

**Critical fix**: Resolved OAuth token provider integration gap:

1. **Fixed Token Flow**: 
   - OAuth middleware now properly connects to Google API clients
   - HTTP transport creates OAuth handler first
   - ServerContext recreated with OAuth token provider
   - Eliminates "no valid token" errors after successful authentication

2. **CLI Enhancements**:
   - Added `--google-client-id` and `--google-client-secret` flags
   - Support for `GOOGLE_CLIENT_ID` and `GOOGLE_CLIENT_SECRET` env vars
   - Clear visual feedback for token refresh status on startup

3. **User Experience**:
   - Server displays token refresh capability on startup
   - Helpful messages guide users to enable automatic refresh
   - Seamless integration between OAuth authentication and Google API access

### Cleanup and Simplification

The OAuth implementation has been significantly cleaned up:

1. **Removed Obsolete Code**: 
   - Removed PKCE implementation (Google handles authorization)
   - Removed unused OAuth types (ClientInfo, AuthorizationCode, etc.)
   - Simplified Store to only manage Google tokens

2. **Token Refresh**:
   - Added `GoogleClientID` and `GoogleClientSecret` to Config
   - Automatic token refresh now fully functional when credentials provided
   - Middleware proactively refreshes tokens 5 minutes before expiration

3. **Dependency Injection**:
   - Removed global `TokenProvider` state
   - All Google API clients now accept `TokenProvider` via constructor
   - ServerContext properly injects token provider to clients

4. **Structured Logging**:
   - Replaced all `fmt.Printf` with `log/slog` structured logging
   - Better debugging and production logging
   - Configurable logger in Config

5. **Configurable Options**:
   - `RateLimitCleanupInterval` now configurable
   - `TrustProxy` defaults to false (secure by default)
   - Clear documentation for all config options

6. **Test Coverage**:
   - All tests updated and passing
   - Removed tests for obsolete functionality
   - Maintained 75.2% coverage for OAuth package

### Code Quality Improvements

- **KISS Principle**: Removed complexity by eliminating custom OAuth authorization server
- **Clean Architecture**: Proper dependency injection instead of global state
- **Least Surprise**: Clear, predictable behavior throughout
- **Production Ready**: Enterprise-grade logging and configuration

## References

- [MCP Specification - Authorization](https://modelcontextprotocol.io/specification/2025-06-18/basic/authorization)
- [OAuth 2.1](https://datatracker.ietf.org/doc/html/draft-ietf-oauth-v2-1-13)
- [RFC 9728 - OAuth 2.0 Protected Resource Metadata](https://datatracker.ietf.org/doc/html/rfc9728)
- [RFC 8707 - Resource Indicators for OAuth 2.0](https://datatracker.ietf.org/doc/html/rfc8707)
- [RFC 7636 - Proof Key for Code Exchange (PKCE)](https://datatracker.ietf.org/doc/html/rfc7636)
- [Google OAuth 2.0](https://developers.google.com/identity/protocols/oauth2)
