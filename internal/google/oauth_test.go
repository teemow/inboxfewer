package google

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestValidateAccountName(t *testing.T) {
	tests := []struct {
		name    string
		account string
		wantErr bool
	}{
		{"valid default", "default", false},
		{"valid work", "work", false},
		{"valid with hyphen", "work-email", false},
		{"valid with underscore", "personal_email", false},
		{"valid alphanumeric", "account123", false},
		{"empty", "", true},
		{"with spaces", "my account", true},
		{"with special chars", "account@work", true},
		{"with slash", "work/personal", true},
		{"with dot", "work.email", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateAccountName(tt.account)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateAccountName() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestGetTokenFilePath(t *testing.T) {
	tests := []struct {
		name    string
		account string
		want    string
	}{
		{"default account", "default", "google-default.token"},
		{"work account", "work", "google-work.token"},
		{"personal account", "personal", "google-personal.token"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getTokenFilePath(tt.account)
			if filepath.Base(got) != tt.want {
				t.Errorf("getTokenFilePath() = %v, want base %v", got, tt.want)
			}
		})
	}
}

func TestHasTokenForAccount(t *testing.T) {
	// Test with invalid account name
	if HasTokenForAccount("invalid account") {
		t.Error("HasTokenForAccount() should return false for invalid account name")
	}

	// Test with empty account name
	if HasTokenForAccount("") {
		t.Error("HasTokenForAccount() should return false for empty account name")
	}

	// Note: We can't easily test with actual token files without mocking,
	// but we've validated the account name validation logic
}

func TestMigrateDefaultToken(t *testing.T) {
	// Get the actual cache directory
	cacheDir := filepath.Join(userCacheDir(), "inboxfewer")

	// Create directory if it doesn't exist
	if err := os.MkdirAll(cacheDir, 0700); err != nil {
		t.Fatal(err)
	}

	// Clean up any existing test files
	oldTokenFile := filepath.Join(cacheDir, "google.token")
	newTokenFile := filepath.Join(cacheDir, "google-default.token")
	defer func() {
		os.Remove(oldTokenFile)
		os.Remove(newTokenFile)
	}()

	// Create old token file for testing
	tokenData := []byte("test_access_token test_refresh_token")
	if err := os.WriteFile(oldTokenFile, tokenData, 0600); err != nil {
		t.Fatal(err)
	}

	// Run migration
	if err := MigrateDefaultToken(); err != nil {
		t.Fatalf("MigrateDefaultToken() error = %v", err)
	}

	// Check that new token file exists
	if _, err := os.Stat(newTokenFile); os.IsNotExist(err) {
		t.Error("New token file should exist after migration")
	}

	// Check that old token file was removed
	if _, err := os.Stat(oldTokenFile); !os.IsNotExist(err) {
		t.Error("Old token file should be removed after migration")
	}

	// Verify token data was preserved
	newData, err := os.ReadFile(newTokenFile)
	if err != nil {
		t.Fatal(err)
	}
	if string(newData) != string(tokenData) {
		t.Errorf("Token data should be preserved during migration, got %s, want %s", string(newData), string(tokenData))
	}

	// Run migration again (should be idempotent)
	if err := MigrateDefaultToken(); err != nil {
		t.Fatalf("Second MigrateDefaultToken() error = %v", err)
	}
}

func TestSaveTokenForAccount(t *testing.T) {
	// This test requires a valid OAuth setup, so we'll just test validation
	ctx := context.Background()

	tests := []struct {
		name    string
		account string
		wantErr bool
	}{
		{"empty account", "", true},
		{"invalid account", "invalid account", true},
		{"valid account (will fail on exchange)", "test", true}, // Will fail because no valid auth code
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := SaveTokenForAccount(ctx, tt.account, "dummy_code")
			if (err != nil) != tt.wantErr {
				t.Errorf("SaveTokenForAccount() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestGetAuthURLForAccount(t *testing.T) {
	tests := []struct {
		name    string
		account string
	}{
		{"default account", "default"},
		{"work account", "work"},
		{"personal account", "personal"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url := GetAuthURLForAccount(tt.account)
			if url == "" {
				t.Error("GetAuthURLForAccount() should return non-empty URL")
			}
			// Check that URL contains state parameter with account
			if tt.account != "" {
				expectedState := "state-" + tt.account
				if !contains(url, expectedState) {
					t.Errorf("GetAuthURLForAccount() URL should contain state=%s", expectedState)
				}
			}
		})
	}
}

func TestDefaultAccountFunctions(t *testing.T) {
	// Test that legacy functions use default account

	// Test HasToken
	defaultResult := HasTokenForAccount("default")
	legacyResult := HasToken()
	if defaultResult != legacyResult {
		t.Error("HasToken() should return same result as HasTokenForAccount('default')")
	}

	// Test GetAuthURL
	defaultURL := GetAuthURLForAccount("default")
	legacyURL := GetAuthURL()
	if defaultURL != legacyURL {
		t.Error("GetAuthURL() should return same URL as GetAuthURLForAccount('default')")
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
