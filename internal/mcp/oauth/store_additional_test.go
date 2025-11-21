package oauth

import (
	"testing"
	"time"

	"golang.org/x/oauth2"
)

func TestStore_SaveGoogleToken_EmptyEmail(t *testing.T) {
	store := NewStore()

	err := store.SaveGoogleToken("", &oauth2.Token{
		AccessToken: "test-token",
	})

	if err == nil {
		t.Error("Expected error when saving token with empty email")
	}
}

func TestStore_SaveGoogleToken_NilToken(t *testing.T) {
	store := NewStore()

	err := store.SaveGoogleToken("test@example.com", nil)

	if err == nil {
		t.Error("Expected error when saving nil token")
	}
}

func TestStore_GetGoogleToken_Expired(t *testing.T) {
	store := NewStore()

	// Save an expired token
	err := store.SaveGoogleToken("test@example.com", &oauth2.Token{
		AccessToken: "expired-token",
		Expiry:      time.Now().Add(-1 * time.Hour),
	})
	if err != nil {
		t.Fatalf("Failed to save token: %v", err)
	}

	// Try to get it - should fail because it's expired
	_, err = store.GetGoogleToken("test@example.com")
	if err == nil {
		t.Error("Expected error when getting expired token")
	}
}

func TestStore_SaveGoogleUserInfo_EmptyEmail(t *testing.T) {
	store := NewStore()

	err := store.SaveGoogleUserInfo("", &GoogleUserInfo{
		Email: "test@example.com",
	})

	if err == nil {
		t.Error("Expected error when saving user info with empty email")
	}
}

func TestStore_SaveGoogleUserInfo_NilUserInfo(t *testing.T) {
	store := NewStore()

	err := store.SaveGoogleUserInfo("test@example.com", nil)

	if err == nil {
		t.Error("Expected error when saving nil user info")
	}
}

func TestStore_GetGoogleUserInfo_NotFound(t *testing.T) {
	store := NewStore()

	_, err := store.GetGoogleUserInfo("nonexistent@example.com")
	if err == nil {
		t.Error("Expected error when getting non-existent user info")
	}
}

func TestStore_DeleteGoogleToken_RemovesUserInfo(t *testing.T) {
	store := NewStore()

	// Add token and user info
	email := "test@example.com"
	err := store.SaveGoogleToken(email, &oauth2.Token{
		AccessToken: "test-token",
		Expiry:      time.Now().Add(1 * time.Hour),
	})
	if err != nil {
		t.Fatalf("Failed to save token: %v", err)
	}

	err = store.SaveGoogleUserInfo(email, &GoogleUserInfo{
		Email: email,
		Name:  "Test User",
	})
	if err != nil {
		t.Fatalf("Failed to save user info: %v", err)
	}

	// Delete token
	err = store.DeleteGoogleToken(email)
	if err != nil {
		t.Fatalf("Failed to delete token: %v", err)
	}

	// Verify both token and user info are gone
	_, err = store.GetGoogleToken(email)
	if err == nil {
		t.Error("Expected error when getting deleted token")
	}

	_, err = store.GetGoogleUserInfo(email)
	if err == nil {
		t.Error("Expected error when getting deleted user info")
	}
}

func TestStore_CleanupExpiredTokens(t *testing.T) {
	// Create store with short cleanup interval for testing
	store := NewStoreWithInterval(100 * time.Millisecond)

	// Add an expired token
	store.SaveGoogleToken("expired@example.com", &oauth2.Token{
		AccessToken: "expired-token",
		Expiry:      time.Now().Add(-1 * time.Hour),
	})

	// Add a valid token
	store.SaveGoogleToken("valid@example.com", &oauth2.Token{
		AccessToken: "valid-token",
		Expiry:      time.Now().Add(1 * time.Hour),
	})

	// Wait for cleanup to run
	time.Sleep(200 * time.Millisecond)

	// Valid token should still exist
	_, err := store.GetGoogleToken("valid@example.com")
	if err != nil {
		t.Error("Valid token was cleaned up unexpectedly")
	}

	// Expired token should be gone (but GetGoogleToken might return error for expired token before cleanup)
	// So we check the stats instead
	stats := store.Stats()
	if stats["google_tokens"] != 1 {
		t.Errorf("After cleanup, google_tokens = %d, want 1", stats["google_tokens"])
	}
}
