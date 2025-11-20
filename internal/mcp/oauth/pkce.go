package oauth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
)

// GenerateCodeVerifier generates a random code verifier for PKCE
// The code verifier is a cryptographically random string using the characters [A-Z] / [a-z] / [0-9] / "-" / "." / "_" / "~"
// with a minimum length of 43 characters and a maximum length of 128 characters.
func GenerateCodeVerifier() (string, error) {
	// Use 32 bytes (256 bits) which will result in 43 characters when base64url encoded
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("failed to generate random bytes: %w", err)
	}

	// Use base64 URL encoding without padding as per RFC 7636
	verifier := base64.RawURLEncoding.EncodeToString(b)
	return verifier, nil
}

// GenerateCodeChallenge generates the code challenge from a code verifier using S256 method
// S256: code_challenge = BASE64URL(SHA256(ASCII(code_verifier)))
func GenerateCodeChallenge(verifier string) string {
	h := sha256.New()
	h.Write([]byte(verifier))
	challenge := base64.RawURLEncoding.EncodeToString(h.Sum(nil))
	return challenge
}

// ValidateCodeChallenge validates that the code verifier matches the code challenge
// using the specified method (plain or S256)
func ValidateCodeChallenge(verifier, challenge, method string) bool {
	switch method {
	case "S256":
		computed := GenerateCodeChallenge(verifier)
		return computed == challenge
	case "plain":
		return verifier == challenge
	case "":
		// If no method specified, default to plain (though S256 is recommended)
		return verifier == challenge
	default:
		// Unknown method
		return false
	}
}

// GenerateAuthorizationCode generates a random authorization code
func GenerateAuthorizationCode() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("failed to generate random bytes: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// GenerateClientSecret generates a random client secret for confidential clients
func GenerateClientSecret() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("failed to generate random bytes: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// GenerateClientID generates a random client ID
func GenerateClientID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("failed to generate random bytes: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// GenerateAccessToken generates a random access token
func GenerateAccessToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("failed to generate random bytes: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// GenerateRefreshToken generates a random refresh token
func GenerateRefreshToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("failed to generate random bytes: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// GenerateState generates a random state parameter for CSRF protection
func GenerateState() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("failed to generate random bytes: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}
