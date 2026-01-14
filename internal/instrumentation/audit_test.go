package instrumentation

import (
	"context"
	"errors"
	"log/slog"
	"testing"
)

// Test constants to reduce string repetition and satisfy goconst
const (
	testEmail        = "jane@example.com"
	testDomain       = "example.com"
	testAccount      = "work"
	testTraceID      = "abc123def456"
	testSpanID       = "span789"
	testToolGmail    = "gmail_list_emails"
	testToolCalendar = "calendar_create_event"
	testToolDrive    = "drive_list_files"
)

func TestToolInvocation_NewAndComplete(t *testing.T) {
	ti := NewToolInvocation(testToolGmail)

	// Verify initial state
	if ti.Tool != testToolGmail {
		t.Errorf("Tool = %q, want %q", ti.Tool, testToolGmail)
	}
	if ti.StartTime.IsZero() {
		t.Error("StartTime should not be zero")
	}

	// Complete the invocation - duration should be calculated from StartTime
	ti.CompleteSuccess()

	if !ti.Success {
		t.Error("Success should be true")
	}
	// Duration is calculated from StartTime, so it should be >= 0
	// We don't check for > 0 as the test may complete instantly
	if ti.Duration < 0 {
		t.Error("Duration should not be negative")
	}
	if ti.Error != "" {
		t.Errorf("Error should be empty, got %q", ti.Error)
	}
}

func TestToolInvocation_CompleteWithError(t *testing.T) {
	ti := NewToolInvocation(testToolCalendar)
	err := errors.New("permission denied")

	ti.CompleteWithError(err)

	if ti.Success {
		t.Error("Success should be false")
	}
	if ti.Error != "permission denied" {
		t.Errorf("Error = %q, want %q", ti.Error, "permission denied")
	}
}

func TestToolInvocation_WithUser(t *testing.T) {
	ti := NewToolInvocation(testToolGmail)
	ti.WithUser(testEmail)

	if ti.UserEmail != testEmail {
		t.Errorf("UserEmail = %q, want %q", ti.UserEmail, testEmail)
	}
}

func TestToolInvocation_WithAccount(t *testing.T) {
	ti := NewToolInvocation(testToolGmail)
	ti.WithAccount(testAccount)

	if ti.Account != testAccount {
		t.Errorf("Account = %q, want %q", ti.Account, testAccount)
	}
}

func TestToolInvocation_WithService(t *testing.T) {
	ti := NewToolInvocation(testToolGmail)
	ti.WithService(ServiceGmail, OperationList)

	if ti.ServiceName != ServiceGmail {
		t.Errorf("ServiceName = %q, want %q", ti.ServiceName, ServiceGmail)
	}
	if ti.Operation != OperationList {
		t.Errorf("Operation = %q, want %q", ti.Operation, OperationList)
	}
}

func TestToolInvocation_UserDomain(t *testing.T) {
	ti := NewToolInvocation("test")
	ti.UserEmail = testEmail

	if domain := ti.UserDomain(); domain != testDomain {
		t.Errorf("UserDomain() = %q, want %q", domain, testDomain)
	}
}

func TestToolInvocation_Status(t *testing.T) {
	ti := NewToolInvocation("test")

	ti.Success = true
	if status := ti.Status(); status != StatusSuccess {
		t.Errorf("Status() = %q, want %q", status, StatusSuccess)
	}

	ti.Success = false
	if status := ti.Status(); status != StatusError {
		t.Errorf("Status() = %q, want %q", status, StatusError)
	}
}

func TestToolInvocation_LogAttrs(t *testing.T) {
	ti := NewToolInvocation(testToolDrive)
	ti.WithUser(testEmail).
		WithAccount(testAccount).
		WithService(ServiceDrive, OperationList).
		CompleteSuccess()
	ti.TraceID = testTraceID

	attrs := ti.LogAttrs()

	// Verify we have the expected attributes
	attrMap := make(map[string]slog.Attr)
	for _, attr := range attrs {
		attrMap[attr.Key] = attr
	}

	// Check required attributes
	requiredKeys := []string{"tool", "user_domain", "duration", "success"}
	for _, key := range requiredKeys {
		if _, ok := attrMap[key]; !ok {
			t.Errorf("Missing required attribute: %s", key)
		}
	}

	// Check cardinality-controlled values
	if domain := attrMap["user_domain"].Value.String(); domain != testDomain {
		t.Errorf("user_domain = %q, want %q", domain, testDomain)
	}

	// Check service-related attributes
	if service := attrMap["service"].Value.String(); service != ServiceDrive {
		t.Errorf("service = %q, want %q", service, ServiceDrive)
	}
	if operation := attrMap["operation"].Value.String(); operation != OperationList {
		t.Errorf("operation = %q, want %q", operation, OperationList)
	}
}

func TestToolInvocation_LogAttrs_WithError(t *testing.T) {
	ti := NewToolInvocation(testToolCalendar)
	ti.WithUser(testEmail).
		WithAccount(testAccount).
		CompleteWithError(errors.New("test error"))

	attrs := ti.LogAttrs()

	attrMap := make(map[string]slog.Attr)
	for _, attr := range attrs {
		attrMap[attr.Key] = attr
	}

	// Check error attribute is present
	if _, ok := attrMap["error"]; !ok {
		t.Error("Missing error attribute")
	}
	if errVal := attrMap["error"].Value.String(); errVal != "test error" {
		t.Errorf("error = %q, want %q", errVal, "test error")
	}
}

func TestToolInvocation_LogAttrs_MinimalFields(t *testing.T) {
	ti := NewToolInvocation(testToolGmail)
	ti.CompleteSuccess()

	attrs := ti.LogAttrs()

	// Verify minimal attributes are present
	attrMap := make(map[string]slog.Attr)
	for _, attr := range attrs {
		attrMap[attr.Key] = attr
	}

	// These should NOT be present when not set
	if _, ok := attrMap["service"]; ok {
		t.Error("service should not be present when empty")
	}
	if _, ok := attrMap["operation"]; ok {
		t.Error("operation should not be present when empty")
	}
	if _, ok := attrMap["trace_id"]; ok {
		t.Error("trace_id should not be present when empty")
	}
}

func TestToolInvocation_LogAttrs_DefaultAccount(t *testing.T) {
	ti := NewToolInvocation(testToolGmail)
	ti.WithAccount("default").CompleteSuccess()

	attrs := ti.LogAttrs()

	attrMap := make(map[string]slog.Attr)
	for _, attr := range attrs {
		attrMap[attr.Key] = attr
	}

	// "default" account should NOT be in attributes to reduce noise
	if _, ok := attrMap["account"]; ok {
		t.Error("account should not be present when set to 'default'")
	}
}

func TestToolInvocation_LogAuditAttrs(t *testing.T) {
	ti := NewToolInvocation(testToolDrive)
	ti.WithUser(testEmail).
		WithAccount(testAccount).
		WithService(ServiceDrive, OperationList).
		CompleteSuccess()
	ti.TraceID = testTraceID
	ti.SpanID = testSpanID

	attrs := ti.LogAuditAttrs()

	// Verify we have the expected attributes
	attrMap := make(map[string]slog.Attr)
	for _, attr := range attrs {
		attrMap[attr.Key] = attr
	}

	// Check that full values are present (not cardinality-controlled)
	if user := attrMap["user"].Value.String(); user != testEmail {
		t.Errorf("user = %q, want %q", user, testEmail)
	}
	if account := attrMap["account"].Value.String(); account != testAccount {
		t.Errorf("account = %q, want %q", account, testAccount)
	}

	// Check trace context
	if traceID := attrMap["trace_id"].Value.String(); traceID != testTraceID {
		t.Errorf("trace_id = %q, want %q", traceID, testTraceID)
	}
	if spanID := attrMap["span_id"].Value.String(); spanID != testSpanID {
		t.Errorf("span_id = %q, want %q", spanID, testSpanID)
	}
}

func TestToolInvocation_LogAuditAttrs_WithError(t *testing.T) {
	ti := NewToolInvocation(testToolCalendar)
	ti.WithUser(testEmail).
		WithAccount(testAccount).
		CompleteWithError(errors.New("audit error"))

	attrs := ti.LogAuditAttrs()

	attrMap := make(map[string]slog.Attr)
	for _, attr := range attrs {
		attrMap[attr.Key] = attr
	}

	// Check error attribute is present
	if _, ok := attrMap["error"]; !ok {
		t.Error("Missing error attribute")
	}
}

func TestToolInvocation_LogAuditAttrs_MinimalFields(t *testing.T) {
	ti := NewToolInvocation(testToolGmail)
	ti.CompleteSuccess()

	attrs := ti.LogAuditAttrs()

	attrMap := make(map[string]slog.Attr)
	for _, attr := range attrs {
		attrMap[attr.Key] = attr
	}

	// These should NOT be present when not set
	if _, ok := attrMap["service"]; ok {
		t.Error("service should not be present when empty")
	}
	if _, ok := attrMap["operation"]; ok {
		t.Error("operation should not be present when empty")
	}
}

func TestToolInvocation_MethodChaining(t *testing.T) {
	ti := NewToolInvocation(testToolGmail).
		WithUser("user@example.com").
		WithAccount("personal").
		WithService(ServiceGmail, OperationSend).
		CompleteSuccess()

	if ti.Tool != testToolGmail {
		t.Errorf("Tool = %q, want %q", ti.Tool, testToolGmail)
	}
	if ti.UserEmail != "user@example.com" {
		t.Errorf("UserEmail = %q, want %q", ti.UserEmail, "user@example.com")
	}
	if ti.Account != "personal" {
		t.Errorf("Account = %q, want %q", ti.Account, "personal")
	}
	if ti.ServiceName != ServiceGmail {
		t.Errorf("ServiceName = %q, want %q", ti.ServiceName, ServiceGmail)
	}
	if !ti.Success {
		t.Error("Success should be true")
	}
}

func TestAuditLogger_New(t *testing.T) {
	// Test with nil logger (should use default)
	al := NewAuditLogger(nil)
	if al.logger == nil {
		t.Error("logger should not be nil when created with nil")
	}

	// Test with custom logger
	logger := slog.Default()
	al = NewAuditLogger(logger)
	if al.logger != logger {
		t.Error("logger should be the provided logger")
	}
}

func TestAuditLogger_LogToolInvocation_Success(t *testing.T) {
	// This test verifies the method runs without panic
	al := NewAuditLogger(slog.Default())
	ti := NewToolInvocation(testToolGmail).
		WithUser(testEmail).
		WithAccount(testAccount).
		CompleteSuccess()

	// Should not panic
	al.LogToolInvocation(ti)
}

func TestAuditLogger_LogToolInvocation_Failure(t *testing.T) {
	// This test verifies the method runs without panic for failures
	al := NewAuditLogger(slog.Default())
	ti := NewToolInvocation(testToolCalendar).
		WithUser(testEmail).
		WithAccount(testAccount).
		CompleteWithError(errors.New("test error"))

	// Should not panic
	al.LogToolInvocation(ti)
}

func TestAuditLogger_LogToolAudit(t *testing.T) {
	// This test verifies the method runs without panic
	al := NewAuditLogger(slog.Default())
	ti := NewToolInvocation(testToolDrive).
		WithUser(testEmail).
		WithAccount(testAccount).
		WithService(ServiceDrive, OperationList).
		CompleteSuccess()
	ti.TraceID = testTraceID

	// Should not panic
	al.LogToolAudit(ti)
}

func TestTraceIDFromContext_NoSpan(t *testing.T) {
	ctx := context.Background()
	traceID := TraceIDFromContext(ctx)

	if traceID != "" {
		t.Errorf("TraceIDFromContext with no span = %q, want empty string", traceID)
	}
}

func TestToolInvocation_WithSpanContext_NoSpan(t *testing.T) {
	ctx := context.Background()
	ti := NewToolInvocation("test").WithSpanContext(ctx)

	if ti.TraceID != "" {
		t.Errorf("TraceID = %q, want empty string", ti.TraceID)
	}
	if ti.SpanID != "" {
		t.Errorf("SpanID = %q, want empty string", ti.SpanID)
	}
}

func TestToolInvocation_Complete_NilError(t *testing.T) {
	ti := NewToolInvocation("test")
	ti.Complete(true, nil)

	if ti.Error != "" {
		t.Errorf("Error = %q, want empty string", ti.Error)
	}
}

func TestToolInvocation_Complete_WithError(t *testing.T) {
	ti := NewToolInvocation("test")
	ti.Complete(false, errors.New("some error"))

	if ti.Success {
		t.Error("Success should be false")
	}
	if ti.Error != "some error" {
		t.Errorf("Error = %q, want %q", ti.Error, "some error")
	}
}
