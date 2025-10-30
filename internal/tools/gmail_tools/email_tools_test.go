package gmail_tools

import (
	"strings"
	"testing"
)

func TestHandleSendEmail_MissingTo(t *testing.T) {
	args := map[string]interface{}{
		"subject": "Test Subject",
		"body":    "Test Body",
	}

	request := &mockCallToolRequest{args: args}
	result, err := handleSendEmail(nil, request, nil)

	if err != nil {
		t.Errorf("handleSendEmail() error = %v, want nil", err)
	}

	if result == nil {
		t.Fatal("handleSendEmail() result is nil")
	}

	if !result.IsError {
		t.Error("handleSendEmail() should return error for missing 'to' field")
	}

	if !strings.Contains(result.Content[0].Text, "'to' field is required") {
		t.Errorf("handleSendEmail() error message = %s, want \"'to' field is required\"", result.Content[0].Text)
	}
}

func TestHandleSendEmail_MissingSubject(t *testing.T) {
	args := map[string]interface{}{
		"to":   "user@example.com",
		"body": "Test Body",
	}

	request := &mockCallToolRequest{args: args}
	result, err := handleSendEmail(nil, request, nil)

	if err != nil {
		t.Errorf("handleSendEmail() error = %v, want nil", err)
	}

	if result == nil {
		t.Fatal("handleSendEmail() result is nil")
	}

	if !result.IsError {
		t.Error("handleSendEmail() should return error for missing 'subject' field")
	}

	if !strings.Contains(result.Content[0].Text, "'subject' field is required") {
		t.Errorf("handleSendEmail() error message = %s, want \"'subject' field is required\"", result.Content[0].Text)
	}
}

func TestHandleSendEmail_MissingBody(t *testing.T) {
	args := map[string]interface{}{
		"to":      "user@example.com",
		"subject": "Test Subject",
	}

	request := &mockCallToolRequest{args: args}
	result, err := handleSendEmail(nil, request, nil)

	if err != nil {
		t.Errorf("handleSendEmail() error = %v, want nil", err)
	}

	if result == nil {
		t.Fatal("handleSendEmail() result is nil")
	}

	if !result.IsError {
		t.Error("handleSendEmail() should return error for missing 'body' field")
	}

	if !strings.Contains(result.Content[0].Text, "'body' field is required") {
		t.Errorf("handleSendEmail() error message = %s, want \"'body' field is required\"", result.Content[0].Text)
	}
}

func TestHandleSendEmail_EmptyTo(t *testing.T) {
	args := map[string]interface{}{
		"to":      "",
		"subject": "Test Subject",
		"body":    "Test Body",
	}

	request := &mockCallToolRequest{args: args}
	result, err := handleSendEmail(nil, request, nil)

	if err != nil {
		t.Errorf("handleSendEmail() error = %v, want nil", err)
	}

	if result == nil {
		t.Fatal("handleSendEmail() result is nil")
	}

	if !result.IsError {
		t.Error("handleSendEmail() should return error for empty 'to' field")
	}

	if !strings.Contains(result.Content[0].Text, "'to' field is required") {
		t.Errorf("handleSendEmail() error message = %s, want \"'to' field is required\"", result.Content[0].Text)
	}
}
