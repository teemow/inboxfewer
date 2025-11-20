# MCP OAuth 2.1 Server Authentication

This document describes the OAuth 2.1 authentication implementation for the MCP server according to the Model Context Protocol (MCP) specification dated 2025-06-18.

## Overview

The MCP server implements OAuth 2.1 authentication to secure access to its endpoints. This ensures that only authorized clients can access the server's capabilities.

**Note**: This is separate from Google OAuth, which the server uses to access Google services (Gmail, Drive, Calendar, etc.). MCP OAuth protects the MCP server itself, while Google OAuth allows the server to access external services on behalf of users.

## Features

- **OAuth 2.1 Compliance**: Full OAuth 2.1 support with security best practices
- **PKCE Support**: Proof Key for Code Exchange (PKCE) required for all authorization flows, especially public clients
- **Dynamic Client Registration** (RFC 7591): Clients can register dynamically without manual configuration
- **Authorization Server Metadata** (RFC 8414): Automatic discovery of endpoints and capabilities
- **Resource Indicators** (RFC 8707): Tokens are bound to the MCP server resource audience
- **Secure Token Storage**: In-memory token store with automatic expiration and cleanup
- **Support for Public and Confidential Clients**: Different security models for different client types

## Architecture

### Components

1. **OAuth Handler** (`internal/mcp/oauth/handler.go`): Implements the OAuth endpoints
   - Authorization endpoint (`/oauth/authorize`)
   - Token endpoint (`/oauth/token`)
   - Well-known metadata endpoint (`/.well-known/oauth-authorization-server`)
   - Dynamic client registration endpoint (`/oauth/register`)

2. **Token Store** (`internal/mcp/oauth/store.go`): Manages clients, tokens, and authorization codes in memory

3. **Middleware** (`internal/mcp/oauth/middleware.go`): Validates OAuth tokens and protects endpoints

4. **PKCE Utilities** (`internal/mcp/oauth/pkce.go`): Generates and validates PKCE parameters

## OAuth Flows

### 1. Dynamic Client Registration

Clients can register themselves dynamically:

```bash
curl -X POST https://mcp.example.com/oauth/register \
  -H "Content-Type: application/json" \
  -d '{
    "redirect_uris": ["https://client.example.com/callback"],
    "client_name": "My MCP Client",
    "token_endpoint_auth_method": "client_secret_post"
  }'
```

Response:
```json
{
  "client_id": "abc123...",
  "client_secret": "secret456...",
  "redirect_uris": ["https://client.example.com/callback"],
  "client_name": "My MCP Client",
  "grant_types": ["authorization_code", "refresh_token"],
  "response_types": ["code"],
  "token_endpoint_auth_method": "client_secret_post"
}
```

For public clients (e.g., mobile apps), use `"token_endpoint_auth_method": "none"` and no client secret will be issued.

### 2. Authorization Code Flow with PKCE

#### Step 1: Generate PKCE Parameters

```javascript
// Generate code verifier (random string)
const codeVerifier = generateRandomString(43); // Base64URL encoded, 43-128 chars

// Generate code challenge (SHA256 hash)
const codeChallenge = base64url(sha256(codeVerifier));
```

#### Step 2: Authorization Request

Redirect user to:

```
https://mcp.example.com/oauth/authorize?
  response_type=code&
  client_id=abc123&
  redirect_uri=https://client.example.com/callback&
  scope=mcp&
  state=random_state&
  code_challenge=sha256_hash&
  code_challenge_method=S256
```

#### Step 3: User Authorizes

The user is redirected back to your callback URL with an authorization code:

```
https://client.example.com/callback?
  code=auth_code_here&
  state=random_state
```

#### Step 4: Token Exchange

Exchange the authorization code for an access token:

```bash
curl -X POST https://mcp.example.com/oauth/token \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -d "grant_type=authorization_code" \
  -d "code=auth_code_here" \
  -d "redirect_uri=https://client.example.com/callback" \
  -d "client_id=abc123" \
  -d "client_secret=secret456" \  # Only for confidential clients
  -d "code_verifier=original_verifier"
```

Response:
```json
{
  "access_token": "access_token_here",
  "token_type": "Bearer",
  "expires_in": 3600,
  "refresh_token": "refresh_token_here",
  "scope": "mcp"
}
```

### 3. Refresh Token Flow

When the access token expires, use the refresh token:

```bash
curl -X POST https://mcp.example.com/oauth/token \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -d "grant_type=refresh_token" \
  -d "refresh_token=refresh_token_here" \
  -d "client_id=abc123" \
  -d "client_secret=secret456"  # Only for confidential clients
```

### 4. Using Access Tokens

Include the access token in the `Authorization` header when making requests to protected MCP endpoints:

```bash
curl -X POST https://mcp.example.com/mcp \
  -H "Authorization: Bearer access_token_here" \
  -H "Content-Type: application/json" \
  -d '{"method": "tools/list", "params": {}}'
```

## Authorization Server Metadata

Clients can discover OAuth endpoints and capabilities:

```bash
curl https://mcp.example.com/.well-known/oauth-authorization-server
```

Response:
```json
{
  "issuer": "https://mcp.example.com",
  "authorization_endpoint": "https://mcp.example.com/oauth/authorize",
  "token_endpoint": "https://mcp.example.com/oauth/token",
  "registration_endpoint": "https://mcp.example.com/oauth/register",
  "scopes_supported": ["mcp"],
  "response_types_supported": ["code"],
  "grant_types_supported": ["authorization_code", "refresh_token"],
  "token_endpoint_auth_methods_supported": ["client_secret_post", "client_secret_basic", "none"],
  "code_challenge_methods_supported": ["S256", "plain"]
}
```

## Security Considerations

1. **HTTPS Required**: All OAuth endpoints MUST be served over HTTPS in production (except localhost for development)

2. **PKCE Required for Public Clients**: Public clients (mobile apps, SPAs) MUST use PKCE to prevent authorization code interception

3. **Resource Binding**: Access tokens are bound to the MCP server resource using RFC 8707. Tokens issued for other resources are rejected.

4. **Short-Lived Tokens**: Access tokens have a default TTL of 1 hour (configurable)

5. **Authorization Code Expiration**: Authorization codes expire after 10 minutes (configurable)

6. **Automatic Cleanup**: Expired tokens and authorization codes are automatically cleaned up every minute

7. **Confidential Clients**: Confidential clients (server-side apps) MUST authenticate with their client secret

## Configuration

The OAuth handler is configured with the following options:

```go
config := &oauth.Config{
    Issuer:               "https://mcp.example.com",
    Resource:             "https://mcp.example.com",
    DefaultTokenTTL:      3600,  // Access token TTL in seconds (default: 3600)
    AuthorizationCodeTTL: 600,   // Authorization code TTL in seconds (default: 600)
    DefaultScopes:        []string{"mcp"},
    SupportedScopes:      []string{"mcp", "admin"},
}

handler, err := oauth.NewHandler(config)
if err != nil {
    log.Fatal(err)
}
```

## Integration with MCP Server

To protect MCP endpoints with OAuth, wrap the MCP handlers with the OAuth middleware:

```go
// Create OAuth handler
oauthHandler, err := oauth.NewHandler(oauthConfig)
if err != nil {
    return err
}

// Protect MCP endpoint
http.Handle("/mcp", oauthHandler.ValidateToken(mcpHandler))

// OAuth endpoints (unprotected)
http.HandleFunc("/.well-known/oauth-authorization-server", oauthHandler.ServeWellKnown)
http.HandleFunc("/oauth/register", oauthHandler.ServeDynamicRegistration)
http.HandleFunc("/oauth/authorize", oauthHandler.ServeAuthorize)
http.HandleFunc("/oauth/token", oauthHandler.ServeToken)
```

## Scopes

The default scope is `mcp`, which grants access to all MCP capabilities. You can define custom scopes for fine-grained access control:

```go
config.SupportedScopes = []string{"mcp", "mcp:read", "mcp:write", "admin"}
```

Use the `RequireScope` middleware to enforce scope requirements:

```go
http.Handle("/mcp/admin", 
    oauthHandler.ValidateToken(
        oauthHandler.RequireScope("admin")(adminHandler),
    ),
)
```

## Backward Compatibility

OAuth authentication is optional and can be disabled for backward compatibility. When OAuth is not configured, the MCP server operates without authentication (as before).

## Testing

The OAuth implementation includes comprehensive unit tests with >80% coverage:

```bash
go test ./internal/mcp/oauth/... -cover
```

## References

- [MCP Specification - Authorization](https://modelcontextprotocol.io/specification/2025-06-18/basic/authorization)
- [OAuth 2.1](https://datatracker.ietf.org/doc/html/draft-ietf-oauth-v2-1-10)
- [RFC 7591 - OAuth 2.0 Dynamic Client Registration Protocol](https://datatracker.ietf.org/doc/html/rfc7591)
- [RFC 8414 - OAuth 2.0 Authorization Server Metadata](https://datatracker.ietf.org/doc/html/rfc8414)
- [RFC 8707 - Resource Indicators for OAuth 2.0](https://datatracker.ietf.org/doc/html/rfc8707)
- [RFC 7636 - Proof Key for Code Exchange (PKCE)](https://datatracker.ietf.org/doc/html/rfc7636)

