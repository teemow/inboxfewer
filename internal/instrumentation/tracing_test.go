package instrumentation

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestSpanAttributeBuilder(t *testing.T) {
	builder := NewSpanAttributeBuilder().
		WithTool("gmail_list_emails").
		WithService("gmail").
		WithOperation("list").
		WithAccount("user@example.com").
		WithResource("email", "12345").
		WithReadOnly(true)

	attrs := builder.Build()

	if len(attrs) != 7 {
		t.Errorf("expected 7 attributes, got %d", len(attrs))
	}

	// Verify attributes are present
	attrMap := make(map[string]interface{})
	for _, attr := range attrs {
		attrMap[string(attr.Key)] = attr.Value.AsInterface()
	}

	if attrMap[SpanAttrTool] != "gmail_list_emails" {
		t.Errorf("expected tool 'gmail_list_emails', got %v", attrMap[SpanAttrTool])
	}
	if attrMap[SpanAttrService] != "gmail" {
		t.Errorf("expected service 'gmail', got %v", attrMap[SpanAttrService])
	}
	if attrMap[SpanAttrOperation] != "list" {
		t.Errorf("expected operation 'list', got %v", attrMap[SpanAttrOperation])
	}
	if attrMap[SpanAttrAccount] != "user@example.com" {
		t.Errorf("expected account 'user@example.com', got %v", attrMap[SpanAttrAccount])
	}
	if attrMap[SpanAttrResourceType] != "email" {
		t.Errorf("expected resource type 'email', got %v", attrMap[SpanAttrResourceType])
	}
	if attrMap[SpanAttrResourceID] != "12345" {
		t.Errorf("expected resource id '12345', got %v", attrMap[SpanAttrResourceID])
	}
	if attrMap[SpanAttrReadOnly] != true {
		t.Errorf("expected read_only true, got %v", attrMap[SpanAttrReadOnly])
	}
}

func TestSpanAttributeBuilder_EmptyValues(t *testing.T) {
	// Empty account should not be added
	builder := NewSpanAttributeBuilder().
		WithTool("test_tool").
		WithAccount("").
		WithResource("", "")

	attrs := builder.Build()

	// Only tool should be present
	if len(attrs) != 1 {
		t.Errorf("expected 1 attribute (only tool), got %d", len(attrs))
	}
}

func TestStartSpan(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Initialize provider to set global tracer
	provider, err := NewProvider(ctx, Config{
		ServiceName:     "test-service",
		ServiceVersion:  "1.0.0",
		Enabled:         true,
		MetricsExporter: "prometheus",
		TracingExporter: "none",
	})
	if err != nil {
		t.Fatalf("failed to create provider: %v", err)
	}
	defer func() { _ = provider.Shutdown(ctx) }()

	spanCtx, span := StartSpan(ctx, "test-span")
	defer span.End()

	if spanCtx == nil {
		t.Error("expected context to be non-nil")
	}
	if span == nil {
		t.Error("expected span to be non-nil")
	}
}

func TestStartToolSpan(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	provider, err := NewProvider(ctx, Config{
		ServiceName:     "test-service",
		ServiceVersion:  "1.0.0",
		Enabled:         true,
		MetricsExporter: "prometheus",
		TracingExporter: "none",
	})
	if err != nil {
		t.Fatalf("failed to create provider: %v", err)
	}
	defer func() { _ = provider.Shutdown(ctx) }()

	spanCtx, span := StartToolSpan(ctx, "gmail_list_emails")
	defer span.End()

	if spanCtx == nil {
		t.Error("expected context to be non-nil")
	}
	if span == nil {
		t.Error("expected span to be non-nil")
	}
}

func TestStartGoogleAPISpan(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	provider, err := NewProvider(ctx, Config{
		ServiceName:     "test-service",
		ServiceVersion:  "1.0.0",
		Enabled:         true,
		MetricsExporter: "prometheus",
		TracingExporter: "none",
	})
	if err != nil {
		t.Fatalf("failed to create provider: %v", err)
	}
	defer func() { _ = provider.Shutdown(ctx) }()

	spanCtx, span := StartGoogleAPISpan(ctx, "gmail", "list")
	defer span.End()

	if spanCtx == nil {
		t.Error("expected context to be non-nil")
	}
	if span == nil {
		t.Error("expected span to be non-nil")
	}
}

func TestSetSpanError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	provider, err := NewProvider(ctx, Config{
		ServiceName:     "test-service",
		ServiceVersion:  "1.0.0",
		Enabled:         true,
		MetricsExporter: "prometheus",
		TracingExporter: "none",
	})
	if err != nil {
		t.Fatalf("failed to create provider: %v", err)
	}
	defer func() { _ = provider.Shutdown(ctx) }()

	_, span := StartSpan(ctx, "test-span")

	// Should not panic
	SetSpanError(span, errors.New("test error"))
	SetSpanError(span, nil) // nil error should be safe
	span.End()
}

func TestSetSpanSuccess(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	provider, err := NewProvider(ctx, Config{
		ServiceName:     "test-service",
		ServiceVersion:  "1.0.0",
		Enabled:         true,
		MetricsExporter: "prometheus",
		TracingExporter: "none",
	})
	if err != nil {
		t.Fatalf("failed to create provider: %v", err)
	}
	defer func() { _ = provider.Shutdown(ctx) }()

	_, span := StartSpan(ctx, "test-span")

	// Should not panic
	SetSpanSuccess(span)
	span.End()
}

func TestAddSpanEvent(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	provider, err := NewProvider(ctx, Config{
		ServiceName:     "test-service",
		ServiceVersion:  "1.0.0",
		Enabled:         true,
		MetricsExporter: "prometheus",
		TracingExporter: "none",
	})
	if err != nil {
		t.Fatalf("failed to create provider: %v", err)
	}
	defer func() { _ = provider.Shutdown(ctx) }()

	_, span := StartSpan(ctx, "test-span")

	// Should not panic
	AddSpanEvent(span, "test-event")
	span.End()
}

func TestGetTraceID_NoSpan(t *testing.T) {
	ctx := context.Background()
	traceID := GetTraceID(ctx)
	if traceID != "" {
		t.Errorf("expected empty trace ID for context without span, got %q", traceID)
	}
}

func TestGetSpanID_NoSpan(t *testing.T) {
	ctx := context.Background()
	spanID := GetSpanID(ctx)
	if spanID != "" {
		t.Errorf("expected empty span ID for context without span, got %q", spanID)
	}
}

func TestSpanContextString_NoSpan(t *testing.T) {
	ctx := context.Background()
	ctxStr := SpanContextString(ctx)
	if ctxStr != "" {
		t.Errorf("expected empty context string for context without span, got %q", ctxStr)
	}
}
