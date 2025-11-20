package oauth

import (
	"time"
)

// Token represents an OAuth 2.1 access token
type Token struct {
	// AccessToken is the token value
	AccessToken string `json:"access_token"`

	// TokenType is typically "Bearer"
	TokenType string `json:"token_type"`

	// ExpiresIn is the lifetime in seconds
	ExpiresIn int64 `json:"expires_in"`

	// RefreshToken for obtaining new access tokens
	RefreshToken string `json:"refresh_token,omitempty"`

	// Scope is the space-separated list of granted scopes
	Scope string `json:"scope,omitempty"`

	// IssuedAt is when the token was created
	IssuedAt time.Time `json:"-"`

	// ClientID is the ID of the client this token was issued to
	ClientID string `json:"-"`

	// Resource is the audience/resource this token is for (RFC 8707)
	Resource string `json:"-"`
}

// IsExpired returns true if the token has expired
func (t *Token) IsExpired() bool {
	if t.ExpiresIn <= 0 {
		return false // No expiration
	}
	expiryTime := t.IssuedAt.Add(time.Duration(t.ExpiresIn) * time.Second)
	return time.Now().After(expiryTime)
}

// ClientInfo represents an OAuth client registration
type ClientInfo struct {
	// ClientID is the unique identifier for the client
	ClientID string `json:"client_id"`

	// ClientSecret for confidential clients (empty for public clients)
	ClientSecret string `json:"client_secret,omitempty"`

	// RedirectURIs are the allowed redirect URIs
	RedirectURIs []string `json:"redirect_uris"`

	// ClientName is the human-readable name
	ClientName string `json:"client_name,omitempty"`

	// ClientURI is the URL of the client's homepage
	ClientURI string `json:"client_uri,omitempty"`

	// GrantTypes are the OAuth grant types this client may use
	GrantTypes []string `json:"grant_types,omitempty"`

	// ResponseTypes are the OAuth response types this client may use
	ResponseTypes []string `json:"response_types,omitempty"`

	// TokenEndpointAuthMethod specifies how the client authenticates
	TokenEndpointAuthMethod string `json:"token_endpoint_auth_method,omitempty"`

	// CreatedAt is when the client was registered
	CreatedAt time.Time `json:"-"`

	// IsPublic indicates if this is a public client (no secret)
	IsPublic bool `json:"-"`
}

// AuthorizationRequest represents an OAuth authorization request
type AuthorizationRequest struct {
	// ResponseType must be "code" for authorization code flow
	ResponseType string

	// ClientID of the requesting client
	ClientID string

	// RedirectURI where to send the authorization code
	RedirectURI string

	// Scope requested by the client
	Scope string

	// State for CSRF protection
	State string

	// CodeChallenge for PKCE
	CodeChallenge string

	// CodeChallengeMethod is typically "S256" for PKCE
	CodeChallengeMethod string

	// Resource is the target audience (RFC 8707)
	Resource string
}

// AuthorizationCode represents a pending authorization
type AuthorizationCode struct {
	// Code is the authorization code value
	Code string

	// ClientID of the requesting client
	ClientID string

	// RedirectURI that was used in the authorization request
	RedirectURI string

	// Scope that was granted
	Scope string

	// CodeChallenge for PKCE verification
	CodeChallenge string

	// CodeChallengeMethod used (typically "S256")
	CodeChallengeMethod string

	// Resource is the target audience
	Resource string

	// ExpiresAt is when this code expires
	ExpiresAt time.Time

	// Used indicates if this code has been exchanged
	Used bool
}

// IsExpired returns true if the authorization code has expired
func (a *AuthorizationCode) IsExpired() bool {
	return time.Now().After(a.ExpiresAt)
}

// TokenRequest represents a token exchange request
type TokenRequest struct {
	// GrantType must be "authorization_code" or "refresh_token"
	GrantType string

	// Code is the authorization code (for authorization_code grant)
	Code string

	// RedirectURI must match the one from authorization request
	RedirectURI string

	// ClientID of the requesting client
	ClientID string

	// ClientSecret for confidential clients
	ClientSecret string

	// CodeVerifier for PKCE
	CodeVerifier string

	// RefreshToken for refresh_token grant
	RefreshToken string

	// Resource is the target audience (RFC 8707)
	Resource string
}

// AuthorizationServerMetadata represents OAuth 2.0 Authorization Server Metadata (RFC 8414)
type AuthorizationServerMetadata struct {
	// Issuer is the authorization server's identifier
	Issuer string `json:"issuer"`

	// AuthorizationEndpoint for obtaining authorization codes
	AuthorizationEndpoint string `json:"authorization_endpoint"`

	// TokenEndpoint for exchanging codes for tokens
	TokenEndpoint string `json:"token_endpoint"`

	// RegistrationEndpoint for dynamic client registration
	RegistrationEndpoint string `json:"registration_endpoint,omitempty"`

	// ScopesSupported lists available scopes
	ScopesSupported []string `json:"scopes_supported,omitempty"`

	// ResponseTypesSupported lists supported response types
	ResponseTypesSupported []string `json:"response_types_supported"`

	// GrantTypesSupported lists supported grant types
	GrantTypesSupported []string `json:"grant_types_supported"`

	// TokenEndpointAuthMethodsSupported lists supported auth methods
	TokenEndpointAuthMethodsSupported []string `json:"token_endpoint_auth_methods_supported"`

	// CodeChallengeMethodsSupported lists PKCE methods
	CodeChallengeMethodsSupported []string `json:"code_challenge_methods_supported"`
}

// DynamicClientRegistrationRequest represents a client registration request (RFC 7591)
type DynamicClientRegistrationRequest struct {
	// RedirectURIs are the client's redirect URIs
	RedirectURIs []string `json:"redirect_uris"`

	// ClientName is the human-readable name
	ClientName string `json:"client_name,omitempty"`

	// ClientURI is the URL of the client's homepage
	ClientURI string `json:"client_uri,omitempty"`

	// GrantTypes are the OAuth grant types requested
	GrantTypes []string `json:"grant_types,omitempty"`

	// ResponseTypes are the OAuth response types requested
	ResponseTypes []string `json:"response_types,omitempty"`

	// TokenEndpointAuthMethod specifies how the client will authenticate
	TokenEndpointAuthMethod string `json:"token_endpoint_auth_method,omitempty"`
}

// ErrorResponse represents an OAuth error response
type ErrorResponse struct {
	// Error is the error code
	Error string `json:"error"`

	// ErrorDescription provides additional information
	ErrorDescription string `json:"error_description,omitempty"`

	// ErrorURI points to error documentation
	ErrorURI string `json:"error_uri,omitempty"`
}

// ProtectedResourceMetadata represents OAuth 2.0 Protected Resource Metadata (RFC 9728)
type ProtectedResourceMetadata struct {
	// Resource is the identifier for the protected resource
	Resource string `json:"resource"`

	// AuthorizationServers lists the authorization servers that can issue tokens for this resource
	AuthorizationServers []string `json:"authorization_servers"`

	// BearerMethodsSupported lists the ways Bearer tokens can be sent (RFC 6750)
	BearerMethodsSupported []string `json:"bearer_methods_supported,omitempty"`

	// ResourceSigningAlgValuesSupported lists supported signing algorithms
	ResourceSigningAlgValuesSupported []string `json:"resource_signing_alg_values_supported,omitempty"`

	// ScopesSupported lists the scopes understood by this resource
	ScopesSupported []string `json:"scopes_supported,omitempty"`
}

// GoogleUserInfo represents the user information from Google's userinfo endpoint
type GoogleUserInfo struct {
	// Sub is the unique Google user ID
	Sub string `json:"sub"`

	// Email is the user's email address
	Email string `json:"email"`

	// EmailVerified indicates if the email is verified
	EmailVerified bool `json:"email_verified"`

	// Name is the user's full name
	Name string `json:"name"`

	// Picture is the URL of the user's profile picture
	Picture string `json:"picture"`

	// GivenName is the user's first name
	GivenName string `json:"given_name"`

	// FamilyName is the user's last name
	FamilyName string `json:"family_name"`

	// Locale is the user's preferred locale
	Locale string `json:"locale"`
}
