package logging

import (
	"errors"
	"log/slog"
	"testing"
)

func TestWithOperation(t *testing.T) {
	logger := slog.Default()
	result := WithOperation(logger, "test_operation")
	if result == nil {
		t.Error("WithOperation returned nil")
	}
}

func TestWithTool(t *testing.T) {
	logger := slog.Default()
	result := WithTool(logger, "test_tool")
	if result == nil {
		t.Error("WithTool returned nil")
	}
}

func TestWithService(t *testing.T) {
	logger := slog.Default()
	result := WithService(logger, "gmail")
	if result == nil {
		t.Error("WithService returned nil")
	}
}

func TestWithAccount(t *testing.T) {
	logger := slog.Default()
	result := WithAccount(logger, "work")
	if result == nil {
		t.Error("WithAccount returned nil")
	}
}

func TestOperationAttr(t *testing.T) {
	attr := Operation("test_op")
	if attr.Key != KeyOperation {
		t.Errorf("Operation key = %q, want %q", attr.Key, KeyOperation)
	}
	if attr.Value.String() != "test_op" {
		t.Errorf("Operation value = %q, want %q", attr.Value.String(), "test_op")
	}
}

func TestServiceAttr(t *testing.T) {
	attr := Service("gmail")
	if attr.Key != KeyService {
		t.Errorf("Service key = %q, want %q", attr.Key, KeyService)
	}
	if attr.Value.String() != "gmail" {
		t.Errorf("Service value = %q, want %q", attr.Value.String(), "gmail")
	}
}

func TestAccountAttr(t *testing.T) {
	attr := Account("work")
	if attr.Key != KeyAccount {
		t.Errorf("Account key = %q, want %q", attr.Key, KeyAccount)
	}
	if attr.Value.String() != "work" {
		t.Errorf("Account value = %q, want %q", attr.Value.String(), "work")
	}
}

func TestToolAttr(t *testing.T) {
	attr := Tool("gmail_send_email")
	if attr.Key != KeyTool {
		t.Errorf("Tool key = %q, want %q", attr.Key, KeyTool)
	}
	if attr.Value.String() != "gmail_send_email" {
		t.Errorf("Tool value = %q, want %q", attr.Value.String(), "gmail_send_email")
	}
}

func TestStatusAttr(t *testing.T) {
	attr := Status(StatusSuccess)
	if attr.Key != KeyStatus {
		t.Errorf("Status key = %q, want %q", attr.Key, KeyStatus)
	}
	if attr.Value.String() != StatusSuccess {
		t.Errorf("Status value = %q, want %q", attr.Value.String(), StatusSuccess)
	}
}

func TestErr(t *testing.T) {
	// Test with error
	err := errors.New("test error")
	attr := Err(err)
	if attr.Key != KeyError {
		t.Errorf("Err key = %q, want %q", attr.Key, KeyError)
	}
	if attr.Value.String() != "test error" {
		t.Errorf("Err value = %q, want %q", attr.Value.String(), "test error")
	}

	// Test with nil - should return an empty group that slog will omit
	attr = Err(nil)
	// Empty Group has empty key
	if attr.Key != "" {
		t.Errorf("Err(nil) key = %q, want empty string (empty group)", attr.Key)
	}
}

func TestAnonymizeEmail(t *testing.T) {
	tests := []struct {
		email    string
		wantLen  int  // Expected length of result (0 for empty)
		hasValue bool // Whether result should have a value
	}{
		{"jane@example.com", 21, true}, // "user:" + 16 hex chars
		{"user@gmail.com", 21, true},
		{"", 0, false},
	}

	for _, tt := range tests {
		t.Run(tt.email, func(t *testing.T) {
			result := AnonymizeEmail(tt.email)
			if tt.hasValue {
				if len(result) != tt.wantLen {
					t.Errorf("AnonymizeEmail(%q) length = %d, want %d", tt.email, len(result), tt.wantLen)
				}
				if result[:5] != "user:" {
					t.Errorf("AnonymizeEmail(%q) should start with 'user:', got %q", tt.email, result)
				}
			} else {
				if result != "" {
					t.Errorf("AnonymizeEmail(%q) = %q, want empty string", tt.email, result)
				}
			}
		})
	}

	// Test deterministic hashing
	hash1 := AnonymizeEmail("test@example.com")
	hash2 := AnonymizeEmail("test@example.com")
	if hash1 != hash2 {
		t.Error("AnonymizeEmail should return deterministic results")
	}

	// Test different emails produce different hashes
	hash3 := AnonymizeEmail("other@example.com")
	if hash1 == hash3 {
		t.Error("Different emails should produce different hashes")
	}
}

func TestUserHash(t *testing.T) {
	attr := UserHash("jane@example.com")
	if attr.Key != KeyUserHash {
		t.Errorf("UserHash key = %q, want %q", attr.Key, KeyUserHash)
	}
	if len(attr.Value.String()) != 21 {
		t.Errorf("UserHash value length = %d, want 21", len(attr.Value.String()))
	}
}

func TestSanitizeToken(t *testing.T) {
	tests := []struct {
		token    string
		expected string
	}{
		{"", "<empty>"},
		{"abc123", "[token:6 chars]"},
		{"a_very_long_token_string", "[token:24 chars]"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := SanitizeToken(tt.token)
			if result != tt.expected {
				t.Errorf("SanitizeToken(%q) = %q, want %q", tt.token, result, tt.expected)
			}
		})
	}
}

func TestExtractDomain(t *testing.T) {
	tests := []struct {
		email    string
		expected string
	}{
		{"jane@example.com", "example.com"},
		{"user@gmail.com", "gmail.com"},
		{"invalid", ""},
		{"", ""},
		{"@", ""},
		{"user@", ""},
	}

	for _, tt := range tests {
		t.Run(tt.email, func(t *testing.T) {
			result := ExtractDomain(tt.email)
			if result != tt.expected {
				t.Errorf("ExtractDomain(%q) = %q, want %q", tt.email, result, tt.expected)
			}
		})
	}
}

func TestDomain(t *testing.T) {
	attr := Domain("jane@example.com")
	if attr.Key != "user_domain" {
		t.Errorf("Domain key = %q, want %q", attr.Key, "user_domain")
	}
	if attr.Value.String() != "example.com" {
		t.Errorf("Domain value = %q, want %q", attr.Value.String(), "example.com")
	}
}

func TestStatusConstants(t *testing.T) {
	if StatusSuccess != "success" {
		t.Errorf("StatusSuccess = %q, want %q", StatusSuccess, "success")
	}
	if StatusError != "error" {
		t.Errorf("StatusError = %q, want %q", StatusError, "error")
	}
}
