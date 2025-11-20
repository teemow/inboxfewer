package oauth

import (
	"testing"
	"time"

	"golang.org/x/oauth2"
)

func TestNewStore(t *testing.T) {
	store := NewStore()
	if store == nil {
		t.Fatal("NewStore() returned nil")
	}

	stats := store.Stats()
	if stats["clients"] != 0 {
		t.Errorf("New store should have 0 clients, got %d", stats["clients"])
	}
	if stats["tokens"] != 0 {
		t.Errorf("New store should have 0 tokens, got %d", stats["tokens"])
	}
}

func TestStore_SaveAndGetClient(t *testing.T) {
	store := NewStore()

	client := &ClientInfo{
		ClientID:     "test-client",
		ClientSecret: "test-secret",
		RedirectURIs: []string{"http://localhost:8080/callback"},
		ClientName:   "Test Client",
		IsPublic:     false,
	}

	// Save client
	if err := store.SaveClient(client); err != nil {
		t.Fatalf("SaveClient() error = %v", err)
	}

	// Get client
	retrieved, err := store.GetClient("test-client")
	if err != nil {
		t.Fatalf("GetClient() error = %v", err)
	}

	if retrieved.ClientID != client.ClientID {
		t.Errorf("GetClient() ClientID = %s, want %s", retrieved.ClientID, client.ClientID)
	}
	if retrieved.ClientSecret != client.ClientSecret {
		t.Errorf("GetClient() ClientSecret = %s, want %s", retrieved.ClientSecret, client.ClientSecret)
	}
}

func TestStore_SaveClientEmptyID(t *testing.T) {
	store := NewStore()

	client := &ClientInfo{
		ClientID: "",
	}

	err := store.SaveClient(client)
	if err == nil {
		t.Error("SaveClient() with empty ID should return error")
	}
}

func TestStore_GetNonExistentClient(t *testing.T) {
	store := NewStore()

	_, err := store.GetClient("non-existent")
	if err == nil {
		t.Error("GetClient() for non-existent client should return error")
	}
}

func TestStore_DeleteClient(t *testing.T) {
	store := NewStore()

	client := &ClientInfo{
		ClientID:     "test-client",
		ClientSecret: "test-secret",
	}

	if err := store.SaveClient(client); err != nil {
		t.Fatalf("SaveClient() error = %v", err)
	}

	if err := store.DeleteClient("test-client"); err != nil {
		t.Fatalf("DeleteClient() error = %v", err)
	}

	_, err := store.GetClient("test-client")
	if err == nil {
		t.Error("GetClient() after DeleteClient() should return error")
	}
}

func TestStore_ValidateClientCredentials(t *testing.T) {
	store := NewStore()

	// Confidential client
	confidentialClient := &ClientInfo{
		ClientID:     "confidential",
		ClientSecret: "secret123",
		IsPublic:     false,
	}

	// Public client
	publicClient := &ClientInfo{
		ClientID: "public",
		IsPublic: true,
	}

	if err := store.SaveClient(confidentialClient); err != nil {
		t.Fatalf("SaveClient() error = %v", err)
	}
	if err := store.SaveClient(publicClient); err != nil {
		t.Fatalf("SaveClient() error = %v", err)
	}

	tests := []struct {
		name         string
		clientID     string
		clientSecret string
		wantErr      bool
	}{
		{
			name:         "valid confidential client",
			clientID:     "confidential",
			clientSecret: "secret123",
			wantErr:      false,
		},
		{
			name:         "invalid secret for confidential client",
			clientID:     "confidential",
			clientSecret: "wrong-secret",
			wantErr:      true,
		},
		{
			name:         "public client without secret",
			clientID:     "public",
			clientSecret: "",
			wantErr:      false,
		},
		{
			name:         "non-existent client",
			clientID:     "non-existent",
			clientSecret: "",
			wantErr:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := store.ValidateClientCredentials(tt.clientID, tt.clientSecret)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateClientCredentials() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestStore_SaveAndGetAuthorizationCode(t *testing.T) {
	store := NewStore()

	code := &AuthorizationCode{
		Code:        "test-code",
		ClientID:    "test-client",
		RedirectURI: "http://localhost:8080/callback",
		Scope:       "read write",
		ExpiresAt:   time.Now().Add(10 * time.Minute),
		Used:        false,
	}

	// Save code
	if err := store.SaveAuthorizationCode(code); err != nil {
		t.Fatalf("SaveAuthorizationCode() error = %v", err)
	}

	// Get code
	retrieved, err := store.GetAuthorizationCode("test-code")
	if err != nil {
		t.Fatalf("GetAuthorizationCode() error = %v", err)
	}

	if retrieved.Code != code.Code {
		t.Errorf("GetAuthorizationCode() Code = %s, want %s", retrieved.Code, code.Code)
	}
	if retrieved.ClientID != code.ClientID {
		t.Errorf("GetAuthorizationCode() ClientID = %s, want %s", retrieved.ClientID, code.ClientID)
	}
}

func TestStore_GetExpiredAuthorizationCode(t *testing.T) {
	store := NewStore()

	code := &AuthorizationCode{
		Code:      "expired-code",
		ClientID:  "test-client",
		ExpiresAt: time.Now().Add(-1 * time.Minute), // Expired
		Used:      false,
	}

	if err := store.SaveAuthorizationCode(code); err != nil {
		t.Fatalf("SaveAuthorizationCode() error = %v", err)
	}

	_, err := store.GetAuthorizationCode("expired-code")
	if err == nil {
		t.Error("GetAuthorizationCode() for expired code should return error")
	}
}

func TestStore_MarkAuthorizationCodeUsed(t *testing.T) {
	store := NewStore()

	code := &AuthorizationCode{
		Code:      "test-code",
		ClientID:  "test-client",
		ExpiresAt: time.Now().Add(10 * time.Minute),
		Used:      false,
	}

	if err := store.SaveAuthorizationCode(code); err != nil {
		t.Fatalf("SaveAuthorizationCode() error = %v", err)
	}

	if err := store.MarkAuthorizationCodeUsed("test-code"); err != nil {
		t.Fatalf("MarkAuthorizationCodeUsed() error = %v", err)
	}

	_, err := store.GetAuthorizationCode("test-code")
	if err == nil {
		t.Error("GetAuthorizationCode() for used code should return error")
	}
}

func TestStore_SaveAndGetToken(t *testing.T) {
	store := NewStore()

	token := &Token{
		AccessToken:  "access-token",
		TokenType:    "Bearer",
		ExpiresIn:    3600,
		RefreshToken: "refresh-token",
		Scope:        "read write",
		IssuedAt:     time.Now(),
		ClientID:     "test-client",
	}

	// Save token
	if err := store.SaveToken(token); err != nil {
		t.Fatalf("SaveToken() error = %v", err)
	}

	// Get token by access token
	retrieved, err := store.GetToken("access-token")
	if err != nil {
		t.Fatalf("GetToken() error = %v", err)
	}

	if retrieved.AccessToken != token.AccessToken {
		t.Errorf("GetToken() AccessToken = %s, want %s", retrieved.AccessToken, token.AccessToken)
	}

	// Get token by refresh token
	retrievedByRefresh, err := store.GetTokenByRefreshToken("refresh-token")
	if err != nil {
		t.Fatalf("GetTokenByRefreshToken() error = %v", err)
	}

	if retrievedByRefresh.AccessToken != token.AccessToken {
		t.Errorf("GetTokenByRefreshToken() AccessToken = %s, want %s", retrievedByRefresh.AccessToken, token.AccessToken)
	}
}

func TestStore_GetExpiredToken(t *testing.T) {
	store := NewStore()

	token := &Token{
		AccessToken: "expired-token",
		TokenType:   "Bearer",
		ExpiresIn:   1,
		IssuedAt:    time.Now().Add(-2 * time.Second), // Expired
		ClientID:    "test-client",
	}

	if err := store.SaveToken(token); err != nil {
		t.Fatalf("SaveToken() error = %v", err)
	}

	_, err := store.GetToken("expired-token")
	if err == nil {
		t.Error("GetToken() for expired token should return error")
	}
}

func TestStore_DeleteToken(t *testing.T) {
	store := NewStore()

	token := &Token{
		AccessToken:  "access-token",
		TokenType:    "Bearer",
		ExpiresIn:    3600,
		RefreshToken: "refresh-token",
		IssuedAt:     time.Now(),
		ClientID:     "test-client",
	}

	if err := store.SaveToken(token); err != nil {
		t.Fatalf("SaveToken() error = %v", err)
	}

	if err := store.DeleteToken("access-token"); err != nil {
		t.Fatalf("DeleteToken() error = %v", err)
	}

	_, err := store.GetToken("access-token")
	if err == nil {
		t.Error("GetToken() after DeleteToken() should return error")
	}

	_, err = store.GetTokenByRefreshToken("refresh-token")
	if err == nil {
		t.Error("GetTokenByRefreshToken() after DeleteToken() should return error")
	}
}

func TestStore_DeleteClientCascade(t *testing.T) {
	store := NewStore()

	client := &ClientInfo{
		ClientID: "test-client",
	}

	token := &Token{
		AccessToken:  "access-token",
		TokenType:    "Bearer",
		ExpiresIn:    3600,
		RefreshToken: "refresh-token",
		IssuedAt:     time.Now(),
		ClientID:     "test-client",
	}

	if err := store.SaveClient(client); err != nil {
		t.Fatalf("SaveClient() error = %v", err)
	}
	if err := store.SaveToken(token); err != nil {
		t.Fatalf("SaveToken() error = %v", err)
	}

	// Delete client should also remove its tokens
	if err := store.DeleteClient("test-client"); err != nil {
		t.Fatalf("DeleteClient() error = %v", err)
	}

	stats := store.Stats()
	if stats["tokens"] != 0 {
		t.Errorf("After DeleteClient(), tokens should be 0, got %d", stats["tokens"])
	}
	if stats["refresh_tokens"] != 0 {
		t.Errorf("After DeleteClient(), refresh_tokens should be 0, got %d", stats["refresh_tokens"])
	}
}

func TestStore_ListClients(t *testing.T) {
	store := NewStore()

	clients := []*ClientInfo{
		{ClientID: "client1"},
		{ClientID: "client2"},
		{ClientID: "client3"},
	}

	for _, client := range clients {
		if err := store.SaveClient(client); err != nil {
			t.Fatalf("SaveClient() error = %v", err)
		}
	}

	listed := store.ListClients()
	if len(listed) != 3 {
		t.Errorf("ListClients() returned %d clients, want 3", len(listed))
	}
}

func TestStore_Stats(t *testing.T) {
	store := NewStore()

	client := &ClientInfo{ClientID: "test-client"}
	token := &Token{
		AccessToken:  "access-token",
		RefreshToken: "refresh-token",
		ExpiresIn:    3600,
		IssuedAt:     time.Now(),
		ClientID:     "test-client",
	}
	code := &AuthorizationCode{
		Code:      "auth-code",
		ClientID:  "test-client",
		ExpiresAt: time.Now().Add(10 * time.Minute),
	}

	if err := store.SaveClient(client); err != nil {
		t.Fatalf("SaveClient() error = %v", err)
	}
	if err := store.SaveToken(token); err != nil {
		t.Fatalf("SaveToken() error = %v", err)
	}
	if err := store.SaveAuthorizationCode(code); err != nil {
		t.Fatalf("SaveAuthorizationCode() error = %v", err)
	}

	stats := store.Stats()

	if stats["clients"] != 1 {
		t.Errorf("Stats() clients = %d, want 1", stats["clients"])
	}
	if stats["tokens"] != 1 {
		t.Errorf("Stats() tokens = %d, want 1", stats["tokens"])
	}
	if stats["refresh_tokens"] != 1 {
		t.Errorf("Stats() refresh_tokens = %d, want 1", stats["refresh_tokens"])
	}
	if stats["authorization_codes"] != 1 {
		t.Errorf("Stats() authorization_codes = %d, want 1", stats["authorization_codes"])
	}
}

func TestStore_DeleteAuthorizationCode(t *testing.T) {
	store := NewStore()

	code := &AuthorizationCode{
		Code:      "test-code",
		ClientID:  "test-client",
		ExpiresAt: time.Now().Add(10 * time.Minute),
		Used:      false,
	}

	if err := store.SaveAuthorizationCode(code); err != nil {
		t.Fatalf("SaveAuthorizationCode() error = %v", err)
	}

	if err := store.DeleteAuthorizationCode("test-code"); err != nil {
		t.Fatalf("DeleteAuthorizationCode() error = %v", err)
	}

	// Should not be able to retrieve deleted code
	_, err := store.GetAuthorizationCode("test-code")
	if err == nil {
		t.Error("GetAuthorizationCode() after DeleteAuthorizationCode() should return error")
	}
}

func TestStore_DeleteTokenByRefreshToken(t *testing.T) {
	store := NewStore()

	token := &Token{
		AccessToken:  "access-token",
		TokenType:    "Bearer",
		ExpiresIn:    3600,
		RefreshToken: "refresh-token",
		IssuedAt:     time.Now(),
		ClientID:     "test-client",
	}

	if err := store.SaveToken(token); err != nil {
		t.Fatalf("SaveToken() error = %v", err)
	}

	if err := store.DeleteTokenByRefreshToken("refresh-token"); err != nil {
		t.Fatalf("DeleteTokenByRefreshToken() error = %v", err)
	}

	// Both access and refresh tokens should be deleted
	_, err := store.GetToken("access-token")
	if err == nil {
		t.Error("GetToken() after DeleteTokenByRefreshToken() should return error")
	}

	_, err = store.GetTokenByRefreshToken("refresh-token")
	if err == nil {
		t.Error("GetTokenByRefreshToken() after DeleteTokenByRefreshToken() should return error")
	}
}

func TestStore_SaveAndGetGoogleToken(t *testing.T) {
	store := NewStore()

	token := &oauth2.Token{
		AccessToken: "google-access-token",
		TokenType:   "Bearer",
		Expiry:      time.Now().Add(1 * time.Hour),
	}

	// Save Google token
	if err := store.SaveGoogleToken("user@example.com", token); err != nil {
		t.Fatalf("SaveGoogleToken() error = %v", err)
	}

	// Get Google token
	retrieved, err := store.GetGoogleToken("user@example.com")
	if err != nil {
		t.Fatalf("GetGoogleToken() error = %v", err)
	}

	if retrieved.AccessToken != token.AccessToken {
		t.Errorf("GetGoogleToken() AccessToken = %s, want %s", retrieved.AccessToken, token.AccessToken)
	}
}

func TestStore_SaveGoogleTokenEmptyEmail(t *testing.T) {
	store := NewStore()

	token := &oauth2.Token{
		AccessToken: "google-access-token",
	}

	err := store.SaveGoogleToken("", token)
	if err == nil {
		t.Error("SaveGoogleToken() with empty email should return error")
	}
}

func TestStore_SaveGoogleTokenNil(t *testing.T) {
	store := NewStore()

	err := store.SaveGoogleToken("user@example.com", nil)
	if err == nil {
		t.Error("SaveGoogleToken() with nil token should return error")
	}
}

func TestStore_GetGoogleTokenNotFound(t *testing.T) {
	store := NewStore()

	_, err := store.GetGoogleToken("nonexistent@example.com")
	if err == nil {
		t.Error("GetGoogleToken() for non-existent user should return error")
	}
}

func TestStore_GetGoogleTokenExpired(t *testing.T) {
	store := NewStore()

	token := &oauth2.Token{
		AccessToken: "google-access-token",
		TokenType:   "Bearer",
		Expiry:      time.Now().Add(-1 * time.Hour), // Expired
	}

	if err := store.SaveGoogleToken("user@example.com", token); err != nil {
		t.Fatalf("SaveGoogleToken() error = %v", err)
	}

	_, err := store.GetGoogleToken("user@example.com")
	if err == nil {
		t.Error("GetGoogleToken() for expired token should return error")
	}
}

func TestStore_DeleteGoogleToken(t *testing.T) {
	store := NewStore()

	token := &oauth2.Token{
		AccessToken: "google-access-token",
		TokenType:   "Bearer",
		Expiry:      time.Now().Add(1 * time.Hour),
	}

	if err := store.SaveGoogleToken("user@example.com", token); err != nil {
		t.Fatalf("SaveGoogleToken() error = %v", err)
	}

	if err := store.DeleteGoogleToken("user@example.com"); err != nil {
		t.Fatalf("DeleteGoogleToken() error = %v", err)
	}

	_, err := store.GetGoogleToken("user@example.com")
	if err == nil {
		t.Error("GetGoogleToken() after DeleteGoogleToken() should return error")
	}
}

func TestStore_SaveAndGetGoogleUserInfo(t *testing.T) {
	store := NewStore()

	userInfo := &GoogleUserInfo{
		Sub:           "12345",
		Email:         "user@example.com",
		EmailVerified: true,
		Name:          "Test User",
	}

	// Save user info
	if err := store.SaveGoogleUserInfo("user@example.com", userInfo); err != nil {
		t.Fatalf("SaveGoogleUserInfo() error = %v", err)
	}

	// Get user info
	retrieved, err := store.GetGoogleUserInfo("user@example.com")
	if err != nil {
		t.Fatalf("GetGoogleUserInfo() error = %v", err)
	}

	if retrieved.Email != userInfo.Email {
		t.Errorf("GetGoogleUserInfo() Email = %s, want %s", retrieved.Email, userInfo.Email)
	}
	if retrieved.Name != userInfo.Name {
		t.Errorf("GetGoogleUserInfo() Name = %s, want %s", retrieved.Name, userInfo.Name)
	}
}

func TestStore_SaveGoogleUserInfoEmptyEmail(t *testing.T) {
	store := NewStore()

	userInfo := &GoogleUserInfo{
		Sub:   "12345",
		Email: "user@example.com",
	}

	err := store.SaveGoogleUserInfo("", userInfo)
	if err == nil {
		t.Error("SaveGoogleUserInfo() with empty email should return error")
	}
}

func TestStore_SaveGoogleUserInfoNil(t *testing.T) {
	store := NewStore()

	err := store.SaveGoogleUserInfo("user@example.com", nil)
	if err == nil {
		t.Error("SaveGoogleUserInfo() with nil userInfo should return error")
	}
}

func TestStore_GetGoogleUserInfoNotFound(t *testing.T) {
	store := NewStore()

	_, err := store.GetGoogleUserInfo("nonexistent@example.com")
	if err == nil {
		t.Error("GetGoogleUserInfo() for non-existent user should return error")
	}
}
