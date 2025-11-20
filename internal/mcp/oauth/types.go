package oauth

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

// ErrorResponse represents an OAuth error response
type ErrorResponse struct {
	// Error is the error code
	Error string `json:"error"`

	// ErrorDescription provides additional information
	ErrorDescription string `json:"error_description,omitempty"`

	// ErrorURI points to error documentation
	ErrorURI string `json:"error_uri,omitempty"`
}
