package signal

import (
	"context"
	"os"
	"os/exec"
	"strings"
	"testing"
)

// mockCommand is a test helper to mock exec.Command
type mockCommand struct {
	stdout string
	stderr string
	err    error
}

// TestNewClient tests the creation of a new Signal client
func TestNewClient(t *testing.T) {
	tests := []struct {
		name      string
		userID    string
		wantErr   bool
		errString string
	}{
		{
			name:    "valid phone number",
			userID:  "+15551234567",
			wantErr: false,
		},
		{
			name:      "empty user ID",
			userID:    "",
			wantErr:   true,
			errString: "userID cannot be empty",
		},
		{
			name:      "missing plus sign",
			userID:    "15551234567",
			wantErr:   true,
			errString: "must be a phone number starting with +",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Skip signal-cli check if not installed (for CI/CD)
			if _, err := exec.LookPath("signal-cli"); err != nil && !tt.wantErr {
				t.Skip("signal-cli not installed")
			}

			ctx := context.Background()
			client, err := NewClient(ctx, tt.userID)

			if tt.wantErr {
				if err == nil {
					t.Errorf("NewClient() expected error containing %q, got nil", tt.errString)
				} else if !strings.Contains(err.Error(), tt.errString) {
					t.Errorf("NewClient() error = %v, want error containing %q", err, tt.errString)
				}
				return
			}

			if err != nil {
				t.Errorf("NewClient() unexpected error = %v", err)
				return
			}

			if client.UserID() != tt.userID {
				t.Errorf("NewClient() userID = %v, want %v", client.UserID(), tt.userID)
			}
		})
	}
}

// TestSendMessage tests sending direct messages
func TestSendMessage(t *testing.T) {
	// Skip if signal-cli is not installed
	if _, err := exec.LookPath("signal-cli"); err != nil {
		t.Skip("signal-cli not installed")
	}

	tests := []struct {
		name      string
		recipient string
		message   string
		wantErr   bool
		errString string
	}{
		{
			name:      "empty recipient",
			recipient: "",
			message:   "test message",
			wantErr:   true,
			errString: "recipient cannot be empty",
		},
		{
			name:      "empty message",
			recipient: "+15559876543",
			message:   "",
			wantErr:   true,
			errString: "message cannot be empty",
		},
		{
			name:      "invalid recipient format",
			recipient: "5559876543",
			message:   "test message",
			wantErr:   true,
			errString: "must be a phone number starting with +",
		},
	}

	ctx := context.Background()
	client, err := NewClient(ctx, "+15551234567")
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := client.SendMessage(tt.recipient, tt.message)

			if tt.wantErr {
				if err == nil {
					t.Errorf("SendMessage() expected error containing %q, got nil", tt.errString)
				} else if !strings.Contains(err.Error(), tt.errString) {
					t.Errorf("SendMessage() error = %v, want error containing %q", err, tt.errString)
				}
				return
			}

			// Note: actual sending will fail without proper signal-cli setup
			// We're primarily testing validation logic here
		})
	}
}

// TestSendGroupMessage tests sending group messages
func TestSendGroupMessage(t *testing.T) {
	// Skip if signal-cli is not installed
	if _, err := exec.LookPath("signal-cli"); err != nil {
		t.Skip("signal-cli not installed")
	}

	tests := []struct {
		name      string
		groupName string
		message   string
		wantErr   bool
		errString string
	}{
		{
			name:      "empty group name",
			groupName: "",
			message:   "test message",
			wantErr:   true,
			errString: "groupName cannot be empty",
		},
		{
			name:      "empty message",
			groupName: "Test Group",
			message:   "",
			wantErr:   true,
			errString: "message cannot be empty",
		},
	}

	ctx := context.Background()
	client, err := NewClient(ctx, "+15551234567")
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := client.SendGroupMessage(tt.groupName, tt.message)

			if tt.wantErr {
				if err == nil {
					t.Errorf("SendGroupMessage() expected error containing %q, got nil", tt.errString)
				} else if !strings.Contains(err.Error(), tt.errString) {
					t.Errorf("SendGroupMessage() error = %v, want error containing %q", err, tt.errString)
				}
				return
			}
		})
	}
}

// TestReceiveMessage tests receiving messages
func TestReceiveMessage(t *testing.T) {
	// Skip if signal-cli is not installed
	if _, err := exec.LookPath("signal-cli"); err != nil {
		t.Skip("signal-cli not installed")
	}

	tests := []struct {
		name           string
		timeoutSeconds int
		wantErr        bool
		errString      string
	}{
		{
			name:           "invalid timeout",
			timeoutSeconds: 0,
			wantErr:        true,
			errString:      "timeout must be positive",
		},
		{
			name:           "negative timeout",
			timeoutSeconds: -5,
			wantErr:        true,
			errString:      "timeout must be positive",
		},
	}

	ctx := context.Background()
	client, err := NewClient(ctx, "+15551234567")
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := client.ReceiveMessage(tt.timeoutSeconds)

			if tt.wantErr {
				if err == nil {
					t.Errorf("ReceiveMessage() expected error containing %q, got nil", tt.errString)
				} else if !strings.Contains(err.Error(), tt.errString) {
					t.Errorf("ReceiveMessage() error = %v, want error containing %q", err, tt.errString)
				}
				return
			}
		})
	}
}

// TestParseReceiveOutput tests parsing of signal-cli receive output
func TestParseReceiveOutput(t *testing.T) {
	tests := []struct {
		name       string
		output     string
		wantMsg    string
		wantSender string
		wantGroup  string
		wantErr    bool
	}{
		{
			name: "simple direct message",
			output: `Envelope from: "Alice" +11234567890 (device: 1) to +15551234567
Timestamp: 1234567890000
Body: Hello, world!`,
			wantMsg:    "Hello, world!",
			wantSender: "+11234567890",
			wantGroup:  "",
			wantErr:    false,
		},
		{
			name: "group message",
			output: `Envelope from: "Bob" +19876543210 (device: 1) to +15551234567
Group info:
  Name: Test Group
  Id: groupIdHere
Timestamp: 1234567890000
Body: Group message here`,
			wantMsg:    "Group message here",
			wantSender: "+19876543210",
			wantGroup:  "Test Group",
			wantErr:    false,
		},
		{
			name:    "no message body",
			output:  `Envelope from: "Carol" +15559999999 (device: 1) to +15551234567`,
			wantErr: true,
		},
		{
			name:    "empty output",
			output:  "",
			wantErr: true,
		},
	}

	ctx := context.Background()
	client, err := NewClient(ctx, "+15551234567")
	if err != nil {
		// Skip if signal-cli is not installed
		t.Skip("signal-cli not installed")
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			response, err := client.parseReceiveOutput(tt.output)

			if tt.wantErr {
				if err == nil {
					t.Errorf("parseReceiveOutput() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("parseReceiveOutput() unexpected error = %v", err)
				return
			}

			if response.Message != tt.wantMsg {
				t.Errorf("parseReceiveOutput() message = %v, want %v", response.Message, tt.wantMsg)
			}

			if response.SenderID != tt.wantSender {
				t.Errorf("parseReceiveOutput() senderID = %v, want %v", response.SenderID, tt.wantSender)
			}

			if response.GroupName != tt.wantGroup {
				t.Errorf("parseReceiveOutput() groupName = %v, want %v", response.GroupName, tt.wantGroup)
			}
		})
	}
}

// TestListGroups tests group listing
func TestListGroups(t *testing.T) {
	// Skip if signal-cli is not installed
	if _, err := exec.LookPath("signal-cli"); err != nil {
		t.Skip("signal-cli not installed")
	}

	ctx := context.Background()
	client, err := NewClient(ctx, "+15551234567")
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	// This test will likely fail without proper signal-cli setup,
	// but it verifies the method signature and basic error handling
	groups, err := client.ListGroups()

	// We expect either success or an error, but no panic
	if err != nil {
		// This is expected if signal-cli isn't properly configured
		t.Logf("ListGroups() error (expected without signal-cli setup): %v", err)
	} else {
		// If successful, verify return type
		if groups == nil {
			t.Error("ListGroups() returned nil instead of empty slice")
		}
	}
}

// TestSignalError tests the SignalError type
func TestSignalError(t *testing.T) {
	tests := []struct {
		name     string
		err      *SignalError
		wantStr  string
		contains []string
	}{
		{
			name: "error with userID",
			err: &SignalError{
				Op:     "send",
				UserID: "+15551234567",
				Err:    os.ErrNotExist,
			},
			contains: []string{"signal send", "+15551234567", "does not exist"},
		},
		{
			name: "error without userID",
			err: &SignalError{
				Op:  "receive",
				Err: os.ErrPermission,
			},
			contains: []string{"signal receive", "permission denied"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errStr := tt.err.Error()

			for _, substr := range tt.contains {
				if !strings.Contains(errStr, substr) {
					t.Errorf("SignalError.Error() = %v, want to contain %v", errStr, substr)
				}
			}

			// Test Unwrap
			if tt.err.Unwrap() != tt.err.Err {
				t.Errorf("SignalError.Unwrap() = %v, want %v", tt.err.Unwrap(), tt.err.Err)
			}
		})
	}
}

// TestUserID tests the UserID getter
func TestUserID(t *testing.T) {
	// Skip if signal-cli is not installed
	if _, err := exec.LookPath("signal-cli"); err != nil {
		t.Skip("signal-cli not installed")
	}

	ctx := context.Background()
	userID := "+15551234567"
	client, err := NewClient(ctx, userID)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	if client.UserID() != userID {
		t.Errorf("UserID() = %v, want %v", client.UserID(), userID)
	}
}
