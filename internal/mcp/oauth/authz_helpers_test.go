package oauth

import (
	"strings"
	"testing"
)

// contains checks if a string contains a substring
func contains(s, substr string) bool {
	return strings.Contains(s, substr)
}

// TestValidatePKCE_CharacterValidation tests RFC 7636 character validation
func TestValidatePKCE_CharacterValidation(t *testing.T) {
	handler, _ := NewHandler(&Config{
		Resource: "https://mcp.example.com",
	})

	tests := []struct {
		name         string
		codeVerifier string
		wantError    bool
		errorMsg     string
	}{
		{
			name:         "Valid code_verifier - alphanumeric",
			codeVerifier: "dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk", // 43 chars
			wantError:    false,
		},
		{
			name:         "Valid code_verifier - all allowed chars",
			codeVerifier: "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789-._~", // 66 chars
			wantError:    false,
		},
		{
			name:         "Valid code_verifier - minimum length (43 chars)",
			codeVerifier: "0123456789012345678901234567890123456789012", // Exactly 43 chars
			wantError:    false,
		},
		{
			name:         "Valid code_verifier - maximum length (128 chars)",
			codeVerifier: "01234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567", // Exactly 128 chars
			wantError:    false,
		},
		{
			name:         "Invalid - too short (42 chars)",
			codeVerifier: "012345678901234567890123456789012345678901",
			wantError:    true,
			errorMsg:     "code_verifier must be at least 43 characters",
		},
		{
			name:         "Invalid - too long (129 chars)",
			codeVerifier: "012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678",
			wantError:    true,
			errorMsg:     "code_verifier must be at most 128 characters",
		},
		{
			name:         "Invalid - contains spaces",
			codeVerifier: "dBjftJeZ4CVP mB92K27uhbUJU1p1r wW1gFWFOEjXk",
			wantError:    true,
			errorMsg:     "code_verifier contains invalid characters",
		},
		{
			name:         "Invalid - contains null byte",
			codeVerifier: "dBjftJeZ4CVP\x00mB92K27uhbUJU1p1r_wW1gFWFOEjXk",
			wantError:    true,
			errorMsg:     "code_verifier contains invalid characters",
		},
		{
			name:         "Invalid - contains control characters",
			codeVerifier: "dBjftJeZ4CVP\nmB92K27uhbUJU1p1r_wW1gFWFOEjXk",
			wantError:    true,
			errorMsg:     "code_verifier contains invalid characters",
		},
		{
			name:         "Invalid - contains Unicode",
			codeVerifier: "dBjftJeZ4CVP–mB92K27uhbUJU1p1r_wW1gFWFOEjXk",
			wantError:    true,
			errorMsg:     "code_verifier contains invalid characters",
		},
		{
			name:         "Invalid - contains special chars not in RFC 7636",
			codeVerifier: "dBjftJeZ4CVP+mB92K27uhbUJU1p1r_wW1gFWFOEjXk",
			wantError:    true,
			errorMsg:     "code_verifier contains invalid characters",
		},
		{
			name:         "Invalid - contains equals sign (base64 padding)",
			codeVerifier: "dBjftJeZ4CVP=mB92K27uhbUJU1p1r_wW1gFWFOEjXk",
			wantError:    true,
			errorMsg:     "code_verifier contains invalid characters",
		},
		{
			name:         "Invalid - contains forward slash",
			codeVerifier: "dBjftJeZ4CVP/mB92K27uhbUJU1p1r_wW1gFWFOEjXk",
			wantError:    true,
			errorMsg:     "code_verifier contains invalid characters",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a mock authorization code with plain challenge for character validation tests
			// This allows us to test character validation without worrying about hash matching
			authCode := &AuthorizationCode{
				CodeChallenge:       tt.codeVerifier, // Use plain method so challenge = verifier
				CodeChallengeMethod: "plain",
			}

			err := handler.validatePKCE(authCode, tt.codeVerifier, "test-client")

			if tt.wantError {
				if err == nil {
					t.Errorf("Expected error containing '%s', got nil", tt.errorMsg)
				} else {
					// Check if the error description contains the expected message
					errStr := err.Error()
					if !contains(errStr, tt.errorMsg) && !contains(err.Description, tt.errorMsg) {
						t.Errorf("Expected error containing '%s', got: %v (code: %s, description: %s)",
							tt.errorMsg, errStr, err.Code, err.Description)
					}
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error, got: %v", err)
				}
			}
		})
	}
}

// TestValidatePKCE_VerificationSuccess tests successful PKCE verification
func TestValidatePKCE_VerificationSuccess(t *testing.T) {
	handler, _ := NewHandler(&Config{
		Resource: "https://mcp.example.com",
	})

	tests := []struct {
		name                string
		codeVerifier        string
		codeChallenge       string
		codeChallengeMethod string
		wantError           bool
	}{
		{
			name:                "S256 method - correct verifier",
			codeVerifier:        "dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk",
			codeChallenge:       "E9Melhoa2OwvFrEMTJguCHaoeK1t8URWbuGJSstw-cM",
			codeChallengeMethod: "S256",
			wantError:           false,
		},
		{
			name:                "plain method - correct verifier",
			codeVerifier:        "dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk",
			codeChallenge:       "dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk",
			codeChallengeMethod: "plain",
			wantError:           false,
		},
		{
			name:                "S256 method - wrong verifier",
			codeVerifier:        "wrong-verifier-value-here-that-is-long-enough",
			codeChallenge:       "E9Melhoa2OwvFrEMTJguCHaoeK1t8URWbuGJSstw-cM",
			codeChallengeMethod: "S256",
			wantError:           true,
		},
		{
			name:                "No PKCE required",
			codeVerifier:        "",
			codeChallenge:       "",
			codeChallengeMethod: "",
			wantError:           false,
		},
		{
			name:                "PKCE required but verifier missing",
			codeVerifier:        "",
			codeChallenge:       "E9Melhoa2OwvFrEMTJguCHaoeK1t8URWbuGJSstw-cM",
			codeChallengeMethod: "S256",
			wantError:           true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			authCode := &AuthorizationCode{
				CodeChallenge:       tt.codeChallenge,
				CodeChallengeMethod: tt.codeChallengeMethod,
			}

			err := handler.validatePKCE(authCode, tt.codeVerifier, "test-client")

			if tt.wantError && err == nil {
				t.Error("Expected error, got nil")
			} else if !tt.wantError && err != nil {
				t.Errorf("Expected no error, got: %v", err)
			}
		})
	}
}
