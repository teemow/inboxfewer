package google

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestHasToken(t *testing.T) {
	// Save original environment variables
	origHome := os.Getenv("HOME")
	origXDG := os.Getenv("XDG_CACHE_HOME")
	defer func() {
		os.Setenv("HOME", origHome)
		os.Setenv("XDG_CACHE_HOME", origXDG)
	}()

	// Create temp directory for testing
	tmpDir := t.TempDir()
	os.Setenv("HOME", tmpDir)
	os.Unsetenv("XDG_CACHE_HOME") // Make sure we use HOME/.cache

	// Initially should not have token
	if HasToken() {
		t.Error("Expected HasToken() to return false when no token exists")
	}

	// Create token file
	cacheDir := filepath.Join(tmpDir, ".cache", "inboxfewer")
	if err := os.MkdirAll(cacheDir, 0700); err != nil {
		t.Fatalf("Failed to create cache dir: %v", err)
	}

	tokenFile := filepath.Join(cacheDir, "google.token")
	if err := os.WriteFile(tokenFile, []byte("access_token refresh_token"), 0600); err != nil {
		t.Fatalf("Failed to write token file: %v", err)
	}

	// Now should have token
	if !HasToken() {
		t.Error("Expected HasToken() to return true when token exists")
	}
}

func TestGetAuthURL(t *testing.T) {
	url := GetAuthURL()

	// Check that URL contains expected components
	expectedComponents := []string{
		"accounts.google.com",
		"client_id=615260903473",
		"scope=https",
		"mail.google.com",
		"documents.readonly",
		"drive.readonly",
	}

	for _, component := range expectedComponents {
		if !contains(url, component) {
			t.Errorf("Expected auth URL to contain '%s', got: %s", component, url)
		}
	}
}

func TestSaveToken(t *testing.T) {
	// Save original environment variables
	origHome := os.Getenv("HOME")
	origXDG := os.Getenv("XDG_CACHE_HOME")
	defer func() {
		os.Setenv("HOME", origHome)
		os.Setenv("XDG_CACHE_HOME", origXDG)
	}()

	// Create temp directory for testing
	tmpDir := t.TempDir()
	os.Setenv("HOME", tmpDir)
	os.Unsetenv("XDG_CACHE_HOME")

	ctx := context.Background()

	// Note: This test won't actually exchange the auth code since we'd need a valid one
	// We're just testing that the function attempts to save properly
	err := SaveToken(ctx, "invalid_auth_code")
	if err == nil {
		t.Error("Expected error when saving invalid auth code")
	}

	// Error should mention failed exchange
	if !contains(err.Error(), "failed to exchange") {
		t.Errorf("Expected error about failed exchange, got: %v", err)
	}
}

func TestGetTokenSource_NoToken(t *testing.T) {
	// Save original environment variables
	origHome := os.Getenv("HOME")
	origXDG := os.Getenv("XDG_CACHE_HOME")
	defer func() {
		os.Setenv("HOME", origHome)
		os.Setenv("XDG_CACHE_HOME", origXDG)
	}()

	// Create temp directory for testing
	tmpDir := t.TempDir()
	os.Setenv("HOME", tmpDir)
	os.Unsetenv("XDG_CACHE_HOME")

	ctx := context.Background()

	_, err := GetTokenSource(ctx)
	if err == nil {
		t.Error("Expected error when no token exists")
	}

	if !contains(err.Error(), "no valid Google OAuth token") {
		t.Errorf("Expected error about missing token, got: %v", err)
	}
}

func TestGetHTTPClient_NoToken(t *testing.T) {
	// Save original environment variables
	origHome := os.Getenv("HOME")
	origXDG := os.Getenv("XDG_CACHE_HOME")
	defer func() {
		os.Setenv("HOME", origHome)
		os.Setenv("XDG_CACHE_HOME", origXDG)
	}()

	// Create temp directory for testing
	tmpDir := t.TempDir()
	os.Setenv("HOME", tmpDir)
	os.Unsetenv("XDG_CACHE_HOME")

	ctx := context.Background()

	_, err := GetHTTPClient(ctx)
	if err == nil {
		t.Error("Expected error when no token exists")
	}
}

func TestGetTokenSource_InvalidTokenFormat(t *testing.T) {
	// Save original environment variables
	origHome := os.Getenv("HOME")
	origXDG := os.Getenv("XDG_CACHE_HOME")
	defer func() {
		os.Setenv("HOME", origHome)
		os.Setenv("XDG_CACHE_HOME", origXDG)
	}()

	// Create temp directory for testing
	tmpDir := t.TempDir()
	os.Setenv("HOME", tmpDir)
	os.Unsetenv("XDG_CACHE_HOME")

	// Create token file with invalid format (only one field)
	cacheDir := filepath.Join(tmpDir, ".cache", "inboxfewer")
	if err := os.MkdirAll(cacheDir, 0700); err != nil {
		t.Fatalf("Failed to create cache dir: %v", err)
	}

	tokenFile := filepath.Join(cacheDir, "google.token")
	if err := os.WriteFile(tokenFile, []byte("only_one_field"), 0600); err != nil {
		t.Fatalf("Failed to write token file: %v", err)
	}

	ctx := context.Background()

	_, err := GetTokenSource(ctx)
	if err == nil {
		t.Error("Expected error when token has invalid format")
	}

	if !contains(err.Error(), "invalid token format") {
		t.Errorf("Expected error about invalid token format, got: %v", err)
	}
}

func TestUserCacheDir_XDG(t *testing.T) {
	// Save original environment variables
	origHome := os.Getenv("HOME")
	origXDG := os.Getenv("XDG_CACHE_HOME")
	defer func() {
		os.Setenv("HOME", origHome)
		os.Setenv("XDG_CACHE_HOME", origXDG)
	}()

	// Create temp directory for XDG
	tmpDir := t.TempDir()
	xdgCache := filepath.Join(tmpDir, "xdg-cache")
	os.Setenv("XDG_CACHE_HOME", xdgCache)
	os.Setenv("HOME", tmpDir)

	// Create token file in XDG location
	if err := os.MkdirAll(filepath.Join(xdgCache, "inboxfewer"), 0700); err != nil {
		t.Fatalf("Failed to create XDG cache dir: %v", err)
	}

	tokenFile := filepath.Join(xdgCache, "inboxfewer", "google.token")
	if err := os.WriteFile(tokenFile, []byte("access refresh"), 0600); err != nil {
		t.Fatalf("Failed to write token file: %v", err)
	}

	// HasToken should find the token in XDG location
	if !HasToken() {
		t.Error("Expected HasToken() to find token in XDG_CACHE_HOME location")
	}
}

func TestSaveToken_FailsWithInvalidCode(t *testing.T) {
	// Save original environment variables
	origHome := os.Getenv("HOME")
	origXDG := os.Getenv("XDG_CACHE_HOME")
	defer func() {
		os.Setenv("HOME", origHome)
		os.Setenv("XDG_CACHE_HOME", origXDG)
	}()

	// Create temp directory for testing
	tmpDir := t.TempDir()
	os.Setenv("HOME", tmpDir)
	os.Unsetenv("XDG_CACHE_HOME")

	ctx := context.Background()

	// Try to save with invalid auth code
	err := SaveToken(ctx, "invalid_code")
	if err == nil {
		t.Error("Expected error when saving invalid auth code")
	}

	// Error should mention failed exchange (which happens before directory creation)
	if !contains(err.Error(), "failed to exchange") {
		t.Errorf("Expected error about failed exchange, got: %v", err)
	}
}

// Helper function to check if string contains substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && containsAt(s, substr))
}

func containsAt(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
