package oauth

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsSilentAuthError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "nil error",
			err:      nil,
			expected: false,
		},
		{
			name:     "generic error",
			err:      errors.New("something went wrong"),
			expected: false,
		},
		{
			name:     "login_required in error message",
			err:      errors.New("oauth error: login_required - user must log in"),
			expected: true,
		},
		{
			name:     "consent_required in error message",
			err:      errors.New("oauth error: consent_required - user must consent"),
			expected: true,
		},
		{
			name:     "interaction_required in error message",
			err:      errors.New("oauth error: interaction_required - interaction needed"),
			expected: true,
		},
		{
			name:     "account_selection_required in error message",
			err:      errors.New("oauth error: account_selection_required - select an account"),
			expected: true,
		},
		{
			name:     "SilentAuthError type",
			err:      &SilentAuthError{Code: "login_required", Description: "No session"},
			expected: true,
		},
		{
			name:     "access_denied is not silent auth error",
			err:      errors.New("oauth error: access_denied - user denied access"),
			expected: false,
		},
		{
			name:     "invalid_request is not silent auth error",
			err:      errors.New("oauth error: invalid_request - bad request"),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsSilentAuthError(tt.err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestParseOAuthError(t *testing.T) {
	tests := []struct {
		name             string
		errorCode        string
		errorDescription string
		expectNil        bool
		expectSilentAuth bool
	}{
		{
			name:             "empty error code returns nil",
			errorCode:        "",
			errorDescription: "",
			expectNil:        true,
			expectSilentAuth: false,
		},
		{
			name:             "login_required returns SilentAuthError",
			errorCode:        "login_required",
			errorDescription: "User must log in",
			expectNil:        false,
			expectSilentAuth: true,
		},
		{
			name:             "consent_required returns SilentAuthError",
			errorCode:        "consent_required",
			errorDescription: "User must consent",
			expectNil:        false,
			expectSilentAuth: true,
		},
		{
			name:             "interaction_required returns SilentAuthError",
			errorCode:        "interaction_required",
			errorDescription: "Interaction needed",
			expectNil:        false,
			expectSilentAuth: true,
		},
		{
			name:             "account_selection_required returns SilentAuthError",
			errorCode:        "account_selection_required",
			errorDescription: "Select account",
			expectNil:        false,
			expectSilentAuth: true,
		},
		{
			name:             "access_denied returns generic error",
			errorCode:        "access_denied",
			errorDescription: "User denied access",
			expectNil:        false,
			expectSilentAuth: false,
		},
		{
			name:             "invalid_request returns generic error",
			errorCode:        "invalid_request",
			errorDescription: "Bad request",
			expectNil:        false,
			expectSilentAuth: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ParseOAuthError(tt.errorCode, tt.errorDescription)

			if tt.expectNil {
				assert.Nil(t, err)
				return
			}

			require.NotNil(t, err)
			assert.Equal(t, tt.expectSilentAuth, IsSilentAuthError(err))
		})
	}
}

func TestParseCallbackQuery(t *testing.T) {
	tests := []struct {
		name             string
		code             string
		state            string
		errorCode        string
		errorDescription string
		errorURI         string
		expectError      bool
		expectSilentAuth bool
	}{
		{
			name:        "successful callback",
			code:        "auth_code_123",
			state:       "state_456",
			expectError: false,
		},
		{
			name:             "login_required error",
			state:            "state_456",
			errorCode:        "login_required",
			errorDescription: "User session expired",
			expectError:      true,
			expectSilentAuth: true,
		},
		{
			name:             "consent_required error",
			state:            "state_456",
			errorCode:        "consent_required",
			errorDescription: "Consent needed",
			expectError:      true,
			expectSilentAuth: true,
		},
		{
			name:             "access_denied error",
			state:            "state_456",
			errorCode:        "access_denied",
			errorDescription: "User denied access",
			expectError:      true,
			expectSilentAuth: false,
		},
		{
			name:             "error with URI",
			state:            "state_456",
			errorCode:        "server_error",
			errorDescription: "Internal server error",
			errorURI:         "https://example.com/error",
			expectError:      true,
			expectSilentAuth: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ParseCallbackQuery(tt.code, tt.state, tt.errorCode, tt.errorDescription, tt.errorURI)

			require.NotNil(t, result)
			assert.Equal(t, tt.code, result.Code)
			assert.Equal(t, tt.state, result.State)
			assert.Equal(t, tt.errorCode, result.Error)
			assert.Equal(t, tt.errorDescription, result.ErrorDescription)
			assert.Equal(t, tt.errorURI, result.ErrorURI)

			err := result.Err()
			if tt.expectError {
				require.NotNil(t, err)
				assert.Equal(t, tt.expectSilentAuth, IsSilentAuthError(err))
			} else {
				assert.Nil(t, err)
			}
		})
	}
}

func TestCallbackResultIsError(t *testing.T) {
	tests := []struct {
		name     string
		result   *CallbackResult
		expected bool
	}{
		{
			name:     "no error",
			result:   &CallbackResult{Code: "abc", State: "xyz"},
			expected: false,
		},
		{
			name:     "has error",
			result:   &CallbackResult{Error: "access_denied", State: "xyz"},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.result.IsError())
		})
	}
}

func TestSilentAuthErrorMessage(t *testing.T) {
	tests := []struct {
		name        string
		err         *SilentAuthError
		expectedMsg string
	}{
		{
			name:        "with description",
			err:         &SilentAuthError{Code: "login_required", Description: "User session expired"},
			expectedMsg: "silent authentication failed: login_required - User session expired",
		},
		{
			name:        "without description",
			err:         &SilentAuthError{Code: "login_required"},
			expectedMsg: "silent authentication failed: login_required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expectedMsg, tt.err.Error())
		})
	}
}

func TestErrorCodeConstants(t *testing.T) {
	// Verify error code constants match expected values
	assert.Equal(t, "login_required", ErrorCodeLoginRequired)
	assert.Equal(t, "consent_required", ErrorCodeConsentRequired)
	assert.Equal(t, "interaction_required", ErrorCodeInteractionRequired)
	assert.Equal(t, "account_selection_required", ErrorCodeAccountSelectionRequired)
}

func TestPromptConstants(t *testing.T) {
	// Verify prompt constants match expected OIDC values
	assert.Equal(t, "none", PromptNone)
	assert.Equal(t, "login", PromptLogin)
	assert.Equal(t, "consent", PromptConsent)
	assert.Equal(t, "select_account", PromptSelectAccount)
}

func TestAuthorizationURLOptionsUsage(t *testing.T) {
	// Test that AuthorizationURLOptions can be constructed and used
	opts := &AuthorizationURLOptions{
		Prompt:      PromptNone,
		LoginHint:   "user@example.com",
		IDTokenHint: "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9...",
	}

	assert.Equal(t, "none", opts.Prompt)
	assert.Equal(t, "user@example.com", opts.LoginHint)
	assert.NotEmpty(t, opts.IDTokenHint)
}

func TestAuthorizationURLOptionsWithMaxAge(t *testing.T) {
	maxAge := 3600 // 1 hour
	opts := &AuthorizationURLOptions{
		Prompt: PromptNone,
		MaxAge: &maxAge,
		Extra:  map[string]string{"custom_param": "custom_value"},
	}

	assert.Equal(t, "none", opts.Prompt)
	require.NotNil(t, opts.MaxAge)
	assert.Equal(t, 3600, *opts.MaxAge)
	assert.Equal(t, "custom_value", opts.Extra["custom_param"])
}
