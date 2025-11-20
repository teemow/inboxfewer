package oauth

import (
	"encoding/base64"
	"testing"
	"time"
)

func TestGenerateCodeVerifier(t *testing.T) {
	verifier, err := GenerateCodeVerifier()
	if err != nil {
		t.Fatalf("GenerateCodeVerifier() error = %v", err)
	}

	// Check length (32 bytes base64url encoded = 43 characters)
	if len(verifier) < 43 {
		t.Errorf("GenerateCodeVerifier() length = %d, want >= 43", len(verifier))
	}
	if len(verifier) > 128 {
		t.Errorf("GenerateCodeVerifier() length = %d, want <= 128", len(verifier))
	}

	// Check that it's valid base64url
	if _, err := base64.RawURLEncoding.DecodeString(verifier); err != nil {
		t.Errorf("GenerateCodeVerifier() not valid base64url: %v", err)
	}

	// Generate multiple verifiers and ensure they're unique
	verifiers := make(map[string]bool)
	for i := 0; i < 100; i++ {
		v, err := GenerateCodeVerifier()
		if err != nil {
			t.Fatalf("GenerateCodeVerifier() iteration %d error = %v", i, err)
		}
		if verifiers[v] {
			t.Errorf("GenerateCodeVerifier() generated duplicate: %s", v)
		}
		verifiers[v] = true
	}
}

func TestGenerateCodeChallenge(t *testing.T) {
	// Test with known verifier
	verifier := "dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk"
	challenge := GenerateCodeChallenge(verifier)

	// Check that challenge is valid base64url
	if _, err := base64.RawURLEncoding.DecodeString(challenge); err != nil {
		t.Errorf("GenerateCodeChallenge() not valid base64url: %v", err)
	}

	// Challenge should be 43 characters (32 bytes SHA256 base64url encoded)
	if len(challenge) != 43 {
		t.Errorf("GenerateCodeChallenge() length = %d, want 43", len(challenge))
	}

	// Same verifier should produce same challenge
	challenge2 := GenerateCodeChallenge(verifier)
	if challenge != challenge2 {
		t.Errorf("GenerateCodeChallenge() not deterministic")
	}
}

func TestValidateCodeChallenge(t *testing.T) {
	tests := []struct {
		name      string
		verifier  string
		challenge string
		method    string
		want      bool
	}{
		{
			name:      "valid S256",
			verifier:  "dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk",
			challenge: GenerateCodeChallenge("dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk"),
			method:    "S256",
			want:      true,
		},
		{
			name:      "invalid S256",
			verifier:  "wrong_verifier",
			challenge: GenerateCodeChallenge("dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk"),
			method:    "S256",
			want:      false,
		},
		{
			name:      "valid plain",
			verifier:  "test_verifier",
			challenge: "test_verifier",
			method:    "plain",
			want:      true,
		},
		{
			name:      "invalid plain",
			verifier:  "test_verifier",
			challenge: "wrong_challenge",
			method:    "plain",
			want:      false,
		},
		{
			name:      "empty method defaults to plain",
			verifier:  "test_verifier",
			challenge: "test_verifier",
			method:    "",
			want:      true,
		},
		{
			name:      "unknown method",
			verifier:  "test_verifier",
			challenge: "test_verifier",
			method:    "unknown",
			want:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ValidateCodeChallenge(tt.verifier, tt.challenge, tt.method)
			if got != tt.want {
				t.Errorf("ValidateCodeChallenge() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGenerateAuthorizationCode(t *testing.T) {
	code, err := GenerateAuthorizationCode()
	if err != nil {
		t.Fatalf("GenerateAuthorizationCode() error = %v", err)
	}

	// Check that it's not empty
	if code == "" {
		t.Error("GenerateAuthorizationCode() returned empty string")
	}

	// Check that it's valid base64url
	if _, err := base64.RawURLEncoding.DecodeString(code); err != nil {
		t.Errorf("GenerateAuthorizationCode() not valid base64url: %v", err)
	}

	// Generate multiple codes and ensure they're unique
	codes := make(map[string]bool)
	for i := 0; i < 100; i++ {
		c, err := GenerateAuthorizationCode()
		if err != nil {
			t.Fatalf("GenerateAuthorizationCode() iteration %d error = %v", i, err)
		}
		if codes[c] {
			t.Errorf("GenerateAuthorizationCode() generated duplicate: %s", c)
		}
		codes[c] = true
	}
}

func TestGenerateClientSecret(t *testing.T) {
	secret, err := GenerateClientSecret()
	if err != nil {
		t.Fatalf("GenerateClientSecret() error = %v", err)
	}

	// Check that it's not empty
	if secret == "" {
		t.Error("GenerateClientSecret() returned empty string")
	}

	// Check that it's valid base64url
	if _, err := base64.RawURLEncoding.DecodeString(secret); err != nil {
		t.Errorf("GenerateClientSecret() not valid base64url: %v", err)
	}

	// Generate multiple secrets and ensure they're unique
	secrets := make(map[string]bool)
	for i := 0; i < 100; i++ {
		s, err := GenerateClientSecret()
		if err != nil {
			t.Fatalf("GenerateClientSecret() iteration %d error = %v", i, err)
		}
		if secrets[s] {
			t.Errorf("GenerateClientSecret() generated duplicate: %s", s)
		}
		secrets[s] = true
	}
}

func TestGenerateClientID(t *testing.T) {
	clientID, err := GenerateClientID()
	if err != nil {
		t.Fatalf("GenerateClientID() error = %v", err)
	}

	// Check that it's not empty
	if clientID == "" {
		t.Error("GenerateClientID() returned empty string")
	}

	// Check that it's valid base64url
	if _, err := base64.RawURLEncoding.DecodeString(clientID); err != nil {
		t.Errorf("GenerateClientID() not valid base64url: %v", err)
	}

	// Generate multiple IDs and ensure they're unique
	ids := make(map[string]bool)
	for i := 0; i < 100; i++ {
		id, err := GenerateClientID()
		if err != nil {
			t.Fatalf("GenerateClientID() iteration %d error = %v", i, err)
		}
		if ids[id] {
			t.Errorf("GenerateClientID() generated duplicate: %s", id)
		}
		ids[id] = true
	}
}

func TestGenerateAccessToken(t *testing.T) {
	token, err := GenerateAccessToken()
	if err != nil {
		t.Fatalf("GenerateAccessToken() error = %v", err)
	}

	// Check that it's not empty
	if token == "" {
		t.Error("GenerateAccessToken() returned empty string")
	}

	// Check that it's valid base64url
	if _, err := base64.RawURLEncoding.DecodeString(token); err != nil {
		t.Errorf("GenerateAccessToken() not valid base64url: %v", err)
	}

	// Generate multiple tokens and ensure they're unique
	tokens := make(map[string]bool)
	for i := 0; i < 100; i++ {
		tok, err := GenerateAccessToken()
		if err != nil {
			t.Fatalf("GenerateAccessToken() iteration %d error = %v", i, err)
		}
		if tokens[tok] {
			t.Errorf("GenerateAccessToken() generated duplicate: %s", tok)
		}
		tokens[tok] = true
	}
}

func TestGenerateRefreshToken(t *testing.T) {
	token, err := GenerateRefreshToken()
	if err != nil {
		t.Fatalf("GenerateRefreshToken() error = %v", err)
	}

	// Check that it's not empty
	if token == "" {
		t.Error("GenerateRefreshToken() returned empty string")
	}

	// Check that it's valid base64url
	if _, err := base64.RawURLEncoding.DecodeString(token); err != nil {
		t.Errorf("GenerateRefreshToken() not valid base64url: %v", err)
	}

	// Generate multiple tokens and ensure they're unique
	tokens := make(map[string]bool)
	for i := 0; i < 100; i++ {
		tok, err := GenerateRefreshToken()
		if err != nil {
			t.Fatalf("GenerateRefreshToken() iteration %d error = %v", i, err)
		}
		if tokens[tok] {
			t.Errorf("GenerateRefreshToken() generated duplicate: %s", tok)
		}
		tokens[tok] = true
	}
}

func TestGenerateState(t *testing.T) {
	state, err := GenerateState()
	if err != nil {
		t.Fatalf("GenerateState() error = %v", err)
	}

	// Check that it's not empty
	if state == "" {
		t.Error("GenerateState() returned empty string")
	}

	// Check that it's valid base64url
	if _, err := base64.RawURLEncoding.DecodeString(state); err != nil {
		t.Errorf("GenerateState() not valid base64url: %v", err)
	}

	// Generate multiple states and ensure they're unique
	states := make(map[string]bool)
	for i := 0; i < 100; i++ {
		s, err := GenerateState()
		if err != nil {
			t.Fatalf("GenerateState() iteration %d error = %v", i, err)
		}
		if states[s] {
			t.Errorf("GenerateState() generated duplicate: %s", s)
		}
		states[s] = true
	}
}

func TestTokenIsExpired(t *testing.T) {
	tests := []struct {
		name     string
		token    *Token
		expected bool
	}{
		{
			name: "not expired",
			token: &Token{
				AccessToken: "test",
				ExpiresIn:   3600,
				IssuedAt:    time.Now(),
			},
			expected: false,
		},
		{
			name: "expired",
			token: &Token{
				AccessToken: "test",
				ExpiresIn:   1,
				IssuedAt:    time.Now().Add(-2 * time.Second),
			},
			expected: true,
		},
		{
			name: "no expiration",
			token: &Token{
				AccessToken: "test",
				ExpiresIn:   0,
				IssuedAt:    time.Now().Add(-24 * time.Hour),
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.token.IsExpired()
			if got != tt.expected {
				t.Errorf("Token.IsExpired() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestAuthorizationCodeIsExpired(t *testing.T) {
	tests := []struct {
		name     string
		code     *AuthorizationCode
		expected bool
	}{
		{
			name: "not expired",
			code: &AuthorizationCode{
				Code:      "test",
				ExpiresAt: time.Now().Add(5 * time.Minute),
			},
			expected: false,
		},
		{
			name: "expired",
			code: &AuthorizationCode{
				Code:      "test",
				ExpiresAt: time.Now().Add(-1 * time.Second),
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.code.IsExpired()
			if got != tt.expected {
				t.Errorf("AuthorizationCode.IsExpired() = %v, want %v", got, tt.expected)
			}
		})
	}
}
