package oauth

import (
	oauth "github.com/giantswarm/mcp-oauth"
	"github.com/giantswarm/mcp-oauth/providers"
)

// AuthorizationURLOptions contains optional OIDC parameters for the authorization request.
// These parameters enable advanced authentication flows like silent re-authentication
// and user hints per OpenID Connect Core 1.0 Section 3.1.2.1.
//
// Common usage for silent authentication:
//
//	opts := &oauth.AuthorizationURLOptions{
//	    Prompt:    "none",       // Silent authentication - no UI displayed
//	    LoginHint: "user@example.com", // Pre-fill email field
//	}
//
// See: https://openid.net/specs/openid-connect-core-1_0.html#AuthRequest
type AuthorizationURLOptions = providers.AuthorizationURLOptions

// SilentAuthError represents an error from a silent authentication attempt.
// These errors indicate the IdP requires user interaction and the client
// should fall back to interactive login.
//
// Silent authentication fails when:
//   - No active session at the IdP (login_required)
//   - User hasn't granted required scopes (consent_required)
//   - IdP needs user interaction for other reasons (interaction_required)
//   - Multiple accounts and none selected (account_selection_required)
//
// See: https://openid.net/specs/openid-connect-core-1_0.html#AuthError
type SilentAuthError = oauth.SilentAuthError

// CallbackResult represents the result of an OAuth authorization callback.
// It parses and holds the query parameters from the OAuth redirect.
//
// The callback may contain either:
//   - Success: Code and State parameters
//   - Error: Error, ErrorDescription, and optionally ErrorURI parameters
//
// Use Err() to get a typed error for error responses, including SilentAuthError
// for silent authentication failures.
type CallbackResult = oauth.CallbackResult

// IsSilentAuthError returns true if the error indicates silent authentication failed
// and interactive login is required. This checks for:
//   - *SilentAuthError type (including wrapped errors)
//   - Error strings containing known silent auth error codes
//
// Silent authentication fails when the IdP requires user interaction but the
// authorization request used prompt=none. Common error codes:
//   - login_required: No active session at the IdP
//   - consent_required: User hasn't granted required scopes
//   - interaction_required: IdP needs user interaction
//   - account_selection_required: Multiple accounts, none selected
//
// Example usage:
//
//	result := handleCallback(r)
//	if err := result.Err(); err != nil {
//	    if oauth.IsSilentAuthError(err) {
//	        // Fall back to interactive login
//	        return startInteractiveLogin(w, r)
//	    }
//	    // Handle other errors
//	    return handleError(w, err)
//	}
func IsSilentAuthError(err error) bool {
	return oauth.IsSilentAuthError(err)
}

// ParseOAuthError parses an OAuth error response and returns the appropriate error type.
// For silent auth failure codes (login_required, consent_required, interaction_required,
// account_selection_required), returns a *SilentAuthError.
// For other errors, returns a generic error with the code and description.
// Returns nil if errorCode is empty.
//
// Example usage:
//
//	err := oauth.ParseOAuthError(r.URL.Query().Get("error"), r.URL.Query().Get("error_description"))
//	if err != nil {
//	    if oauth.IsSilentAuthError(err) {
//	        // Handle silent auth failure - fall back to interactive login
//	    }
//	}
func ParseOAuthError(errorCode, errorDescription string) error {
	return oauth.ParseOAuthError(errorCode, errorDescription)
}

// ParseCallbackQuery creates a CallbackResult from URL query parameters.
// This is a convenience function for parsing OAuth callback query strings.
//
// Parameters:
//   - code: The authorization code (from "code" query param)
//   - state: The state parameter (from "state" query param)
//   - errorCode: The error code (from "error" query param)
//   - errorDescription: The error description (from "error_description" query param)
//   - errorURI: The error URI (from "error_uri" query param)
//
// Example usage:
//
//	q := r.URL.Query()
//	result := oauth.ParseCallbackQuery(
//	    q.Get("code"),
//	    q.Get("state"),
//	    q.Get("error"),
//	    q.Get("error_description"),
//	    q.Get("error_uri"),
//	)
//	if err := result.Err(); err != nil {
//	    if oauth.IsSilentAuthError(err) {
//	        // Fall back to interactive login
//	    }
//	}
func ParseCallbackQuery(code, state, errorCode, errorDescription, errorURI string) *CallbackResult {
	return oauth.ParseCallbackQuery(code, state, errorCode, errorDescription, errorURI)
}

// OAuth error codes for silent authentication failures.
// These are defined per OIDC Core Section 3.1.2.6.
const (
	// ErrorCodeLoginRequired indicates no active session at the IdP.
	// The user must log in interactively.
	ErrorCodeLoginRequired = oauth.ErrorCodeLoginRequired

	// ErrorCodeConsentRequired indicates the user hasn't granted required scopes.
	// The consent screen must be displayed.
	ErrorCodeConsentRequired = oauth.ErrorCodeConsentRequired

	// ErrorCodeInteractionRequired indicates the IdP needs user interaction for other reasons.
	// Interactive login is required.
	ErrorCodeInteractionRequired = oauth.ErrorCodeInteractionRequired

	// ErrorCodeAccountSelectionRequired indicates multiple accounts are available
	// and the user must select one.
	ErrorCodeAccountSelectionRequired = oauth.ErrorCodeAccountSelectionRequired
)

// OIDC Prompt values for AuthorizationURLOptions.Prompt field.
const (
	// PromptNone requests silent authentication - no UI displayed.
	// Returns error if login or consent is required.
	PromptNone = "none"

	// PromptLogin forces re-authentication even if session exists.
	PromptLogin = "login"

	// PromptConsent forces consent screen even if previously granted.
	PromptConsent = "consent"

	// PromptSelectAccount forces account selection even if only one account.
	PromptSelectAccount = "select_account"
)
