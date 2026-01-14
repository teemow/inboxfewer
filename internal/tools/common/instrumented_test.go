package common

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/metric/noop"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"

	"github.com/teemow/inboxfewer/internal/instrumentation"
	"github.com/teemow/inboxfewer/internal/server"
)

func TestInstrumentedToolHandler_Success(t *testing.T) {
	ctx := context.Background()

	// Create a server context without metrics (nil metrics)
	sc, err := server.NewServerContext(ctx, "", "")
	if err != nil {
		t.Fatalf("failed to create server context: %v", err)
	}
	defer sc.Shutdown()

	// Create a handler that returns success
	called := false
	handler := func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		called = true
		return mcp.NewToolResultText("success"), nil
	}

	// Wrap with instrumentation
	wrapped := InstrumentedToolHandler("test_tool", sc, handler)

	// Create a test request
	req := mcp.CallToolRequest{}

	// Call the wrapped handler
	result, err := wrapped(ctx, req)

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if !called {
		t.Error("expected handler to be called")
	}
	if result == nil {
		t.Error("expected result, got nil")
	}
}

func TestInstrumentedToolHandler_Error(t *testing.T) {
	ctx := context.Background()

	// Create a server context without metrics
	sc, err := server.NewServerContext(ctx, "", "")
	if err != nil {
		t.Fatalf("failed to create server context: %v", err)
	}
	defer sc.Shutdown()

	// Create a handler that returns an error
	expectedErr := errors.New("test error")
	handler := func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return nil, expectedErr
	}

	// Wrap with instrumentation
	wrapped := InstrumentedToolHandler("test_tool", sc, handler)

	// Create a test request
	req := mcp.CallToolRequest{}

	// Call the wrapped handler
	_, err = wrapped(ctx, req)

	if err != expectedErr {
		t.Errorf("expected error %v, got %v", expectedErr, err)
	}
}

func TestInstrumentedToolHandler_ErrorResult(t *testing.T) {
	ctx := context.Background()

	// Create a server context without metrics
	sc, err := server.NewServerContext(ctx, "", "")
	if err != nil {
		t.Fatalf("failed to create server context: %v", err)
	}
	defer sc.Shutdown()

	// Create a handler that returns an error result (not Go error)
	handler := func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return mcp.NewToolResultError("error message"), nil
	}

	// Wrap with instrumentation
	wrapped := InstrumentedToolHandler("test_tool", sc, handler)

	// Create a test request
	req := mcp.CallToolRequest{}

	// Call the wrapped handler
	result, err := wrapped(ctx, req)

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if result == nil {
		t.Error("expected result, got nil")
	}
	if !result.IsError {
		t.Error("expected result.IsError to be true")
	}
}

func TestInstrumentedToolHandlerWithService_Success(t *testing.T) {
	ctx := context.Background()

	// Create a server context without metrics
	sc, err := server.NewServerContext(ctx, "", "")
	if err != nil {
		t.Fatalf("failed to create server context: %v", err)
	}
	defer sc.Shutdown()

	// Create a handler that returns success
	called := false
	handler := func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		called = true
		return mcp.NewToolResultText("success"), nil
	}

	// Wrap with instrumentation
	wrapped := InstrumentedToolHandlerWithService("test_tool", "gmail", "list", sc, handler)

	// Create a test request
	req := mcp.CallToolRequest{}

	// Call the wrapped handler
	result, err := wrapped(ctx, req)

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if !called {
		t.Error("expected handler to be called")
	}
	if result == nil {
		t.Error("expected result, got nil")
	}
}

func TestInstrumentedToolHandlerWithService_WithMetrics(t *testing.T) {
	ctx := context.Background()

	// Create a server context
	sc, err := server.NewServerContext(ctx, "", "")
	if err != nil {
		t.Fatalf("failed to create server context: %v", err)
	}
	defer sc.Shutdown()

	// Create metrics with noop meter (for testing)
	meter := noop.NewMeterProvider().Meter("test")
	metrics, err := instrumentation.NewMetrics(meter, false)
	if err != nil {
		t.Fatalf("failed to create metrics: %v", err)
	}

	// Set metrics on server context
	sc.SetMetrics(metrics)

	// Create a handler that simulates some work
	handler := func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		// Simulate some work
		time.Sleep(1 * time.Millisecond)
		return mcp.NewToolResultText("success"), nil
	}

	// Wrap with instrumentation including service info
	wrapped := InstrumentedToolHandlerWithService("gmail_list_emails", "gmail", "list", sc, handler)

	// Create a test request
	req := mcp.CallToolRequest{}

	// Call the wrapped handler
	result, err := wrapped(ctx, req)

	// Verify the call succeeded
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if result == nil {
		t.Error("expected result, got nil")
	}

	// Note: With noop meter, we can't verify actual metric values,
	// but we verify the code path executes without panics.
	// The metrics are recorded via:
	// - metrics.RecordToolInvocation(ctx, "gmail_list_emails", "success", duration)
	// - metrics.RecordGoogleAPIOperation(ctx, "gmail", "list", "success", duration)
}

func TestInstrumentedToolHandlerWithService_ErrorWithMetrics(t *testing.T) {
	ctx := context.Background()

	// Create a server context
	sc, err := server.NewServerContext(ctx, "", "")
	if err != nil {
		t.Fatalf("failed to create server context: %v", err)
	}
	defer sc.Shutdown()

	// Create metrics with noop meter
	meter := noop.NewMeterProvider().Meter("test")
	metrics, err := instrumentation.NewMetrics(meter, false)
	if err != nil {
		t.Fatalf("failed to create metrics: %v", err)
	}

	// Set metrics on server context
	sc.SetMetrics(metrics)

	// Create a handler that returns an error
	expectedErr := errors.New("calendar API error")
	handler := func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return nil, expectedErr
	}

	// Wrap with instrumentation including service info
	wrapped := InstrumentedToolHandlerWithService("calendar_create_event", "calendar", "create", sc, handler)

	// Create a test request
	req := mcp.CallToolRequest{}

	// Call the wrapped handler
	_, err = wrapped(ctx, req)

	// Verify the error is propagated
	if err != expectedErr {
		t.Errorf("expected error %v, got %v", expectedErr, err)
	}

	// Note: With noop meter, we can't verify actual metric values,
	// but we verify the code path executes without panics.
	// The metrics are recorded with status "error" via:
	// - metrics.RecordToolInvocation(ctx, "calendar_create_event", "error", duration)
	// - metrics.RecordGoogleAPIOperation(ctx, "calendar", "create", "error", duration)
}

// setupTestTracer creates a tracer provider with an in-memory span exporter for testing.
// Returns the exporter (for verification) and a cleanup function.
func setupTestTracer() (*tracetest.InMemoryExporter, func()) {
	exporter := tracetest.NewInMemoryExporter()
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithSyncer(exporter),
	)
	oldTP := otel.GetTracerProvider()
	otel.SetTracerProvider(tp)

	return exporter, func() {
		otel.SetTracerProvider(oldTP)
		_ = tp.Shutdown(context.Background())
	}
}

func TestInstrumentedToolHandler_CreatesSpan(t *testing.T) {
	exporter, cleanup := setupTestTracer()
	defer cleanup()

	ctx := context.Background()

	// Create a server context
	sc, err := server.NewServerContext(ctx, "", "")
	if err != nil {
		t.Fatalf("failed to create server context: %v", err)
	}
	defer sc.Shutdown()

	// Create a handler that returns success
	handler := func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return mcp.NewToolResultText("success"), nil
	}

	// Wrap with instrumentation
	wrapped := InstrumentedToolHandler("test_tool", sc, handler)

	// Call the wrapped handler
	req := mcp.CallToolRequest{}
	_, err = wrapped(ctx, req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify span was created
	spans := exporter.GetSpans()
	if len(spans) != 1 {
		t.Fatalf("expected 1 span, got %d", len(spans))
	}

	span := spans[0]

	// Verify span name follows convention: tool.<tool_name>
	expectedName := "tool.test_tool"
	if span.Name != expectedName {
		t.Errorf("expected span name %q, got %q", expectedName, span.Name)
	}

	// Verify span has success status
	if span.Status.Code != codes.Ok {
		t.Errorf("expected span status Ok, got %v", span.Status.Code)
	}

	// Verify mcp.tool attribute is present
	found := false
	for _, attr := range span.Attributes {
		if string(attr.Key) == instrumentation.SpanAttrTool {
			if attr.Value.AsString() != "test_tool" {
				t.Errorf("expected mcp.tool='test_tool', got %v", attr.Value.AsString())
			}
			found = true
			break
		}
	}
	if !found {
		t.Error("expected mcp.tool attribute to be present")
	}
}

func TestInstrumentedToolHandler_SpanRecordsError(t *testing.T) {
	exporter, cleanup := setupTestTracer()
	defer cleanup()

	ctx := context.Background()

	// Create a server context
	sc, err := server.NewServerContext(ctx, "", "")
	if err != nil {
		t.Fatalf("failed to create server context: %v", err)
	}
	defer sc.Shutdown()

	// Create a handler that returns an error
	expectedErr := errors.New("test error")
	handler := func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return nil, expectedErr
	}

	// Wrap with instrumentation
	wrapped := InstrumentedToolHandler("test_tool", sc, handler)

	// Call the wrapped handler
	req := mcp.CallToolRequest{}
	_, _ = wrapped(ctx, req)

	// Verify span was created with error status
	spans := exporter.GetSpans()
	if len(spans) != 1 {
		t.Fatalf("expected 1 span, got %d", len(spans))
	}

	span := spans[0]

	// Verify span has error status
	if span.Status.Code != codes.Error {
		t.Errorf("expected span status Error, got %v", span.Status.Code)
	}

	// Verify error was recorded
	if len(span.Events) == 0 {
		t.Error("expected span to have error event recorded")
	}
}

func TestInstrumentedToolHandlerWithService_CreatesSpanWithServiceAttributes(t *testing.T) {
	exporter, cleanup := setupTestTracer()
	defer cleanup()

	ctx := context.Background()

	// Create a server context
	sc, err := server.NewServerContext(ctx, "", "")
	if err != nil {
		t.Fatalf("failed to create server context: %v", err)
	}
	defer sc.Shutdown()

	// Create a handler that returns success
	handler := func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return mcp.NewToolResultText("success"), nil
	}

	// Wrap with instrumentation including service info
	wrapped := InstrumentedToolHandlerWithService("gmail_list_emails", "gmail", "list", sc, handler)

	// Call the wrapped handler
	req := mcp.CallToolRequest{}
	_, err = wrapped(ctx, req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify span was created
	spans := exporter.GetSpans()
	if len(spans) != 1 {
		t.Fatalf("expected 1 span, got %d", len(spans))
	}

	span := spans[0]

	// Verify span name follows convention: tool.<tool_name>
	expectedName := "tool.gmail_list_emails"
	if span.Name != expectedName {
		t.Errorf("expected span name %q, got %q", expectedName, span.Name)
	}

	// Verify span has success status
	if span.Status.Code != codes.Ok {
		t.Errorf("expected span status Ok, got %v", span.Status.Code)
	}

	// Build a map of attributes for easier checking
	attrMap := make(map[string]string)
	for _, attr := range span.Attributes {
		attrMap[string(attr.Key)] = attr.Value.AsString()
	}

	// Verify required attributes are present
	if attrMap[instrumentation.SpanAttrTool] != "gmail_list_emails" {
		t.Errorf("expected mcp.tool='gmail_list_emails', got %v", attrMap[instrumentation.SpanAttrTool])
	}
	if attrMap[instrumentation.SpanAttrService] != "gmail" {
		t.Errorf("expected google.service='gmail', got %v", attrMap[instrumentation.SpanAttrService])
	}
	if attrMap[instrumentation.SpanAttrOperation] != "list" {
		t.Errorf("expected google.operation='list', got %v", attrMap[instrumentation.SpanAttrOperation])
	}
}

func TestInstrumentedToolHandlerWithService_SpanRecordsErrorWithService(t *testing.T) {
	exporter, cleanup := setupTestTracer()
	defer cleanup()

	ctx := context.Background()

	// Create a server context
	sc, err := server.NewServerContext(ctx, "", "")
	if err != nil {
		t.Fatalf("failed to create server context: %v", err)
	}
	defer sc.Shutdown()

	// Create a handler that returns an error
	expectedErr := errors.New("gmail API error")
	handler := func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return nil, expectedErr
	}

	// Wrap with instrumentation including service info
	wrapped := InstrumentedToolHandlerWithService("gmail_send_email", "gmail", "send", sc, handler)

	// Call the wrapped handler
	req := mcp.CallToolRequest{}
	_, _ = wrapped(ctx, req)

	// Verify span was created with error status
	spans := exporter.GetSpans()
	if len(spans) != 1 {
		t.Fatalf("expected 1 span, got %d", len(spans))
	}

	span := spans[0]

	// Verify span has error status
	if span.Status.Code != codes.Error {
		t.Errorf("expected span status Error, got %v", span.Status.Code)
	}

	// Verify service attributes are still present
	attrMap := make(map[string]string)
	for _, attr := range span.Attributes {
		attrMap[string(attr.Key)] = attr.Value.AsString()
	}

	if attrMap[instrumentation.SpanAttrService] != "gmail" {
		t.Errorf("expected google.service='gmail', got %v", attrMap[instrumentation.SpanAttrService])
	}
	if attrMap[instrumentation.SpanAttrOperation] != "send" {
		t.Errorf("expected google.operation='send', got %v", attrMap[instrumentation.SpanAttrOperation])
	}

	// Verify error was recorded
	if len(span.Events) == 0 {
		t.Error("expected span to have error event recorded")
	}
}

func TestInstrumentedToolHandler_SpanContextPassedToHandler(t *testing.T) {
	exporter, cleanup := setupTestTracer()
	defer cleanup()

	ctx := context.Background()

	// Create a server context
	sc, err := server.NewServerContext(ctx, "", "")
	if err != nil {
		t.Fatalf("failed to create server context: %v", err)
	}
	defer sc.Shutdown()

	// Create a handler that verifies trace context is present
	var traceID, spanID string
	handler := func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		traceID = instrumentation.GetTraceID(ctx)
		spanID = instrumentation.GetSpanID(ctx)
		return mcp.NewToolResultText("success"), nil
	}

	// Wrap with instrumentation
	wrapped := InstrumentedToolHandler("test_tool", sc, handler)

	// Call the wrapped handler
	req := mcp.CallToolRequest{}
	_, err = wrapped(ctx, req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify trace context was available inside the handler
	if traceID == "" {
		t.Error("expected trace ID to be available inside handler")
	}
	if spanID == "" {
		t.Error("expected span ID to be available inside handler")
	}

	// Verify the span in the exporter matches
	spans := exporter.GetSpans()
	if len(spans) != 1 {
		t.Fatalf("expected 1 span, got %d", len(spans))
	}

	span := spans[0]
	if span.SpanContext.TraceID().String() != traceID {
		t.Errorf("expected trace ID %q, got %q", span.SpanContext.TraceID().String(), traceID)
	}
	if span.SpanContext.SpanID().String() != spanID {
		t.Errorf("expected span ID %q, got %q", span.SpanContext.SpanID().String(), spanID)
	}
}

func TestInstrumentedToolHandler_SpanRecordsErrorResultStatus(t *testing.T) {
	exporter, cleanup := setupTestTracer()
	defer cleanup()

	ctx := context.Background()

	// Create a server context
	sc, err := server.NewServerContext(ctx, "", "")
	if err != nil {
		t.Fatalf("failed to create server context: %v", err)
	}
	defer sc.Shutdown()

	// Create a handler that returns an error result (not Go error)
	handler := func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return mcp.NewToolResultError("error message"), nil
	}

	// Wrap with instrumentation
	wrapped := InstrumentedToolHandler("test_tool", sc, handler)

	// Call the wrapped handler
	req := mcp.CallToolRequest{}
	_, err = wrapped(ctx, req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify span was created with error status
	spans := exporter.GetSpans()
	if len(spans) != 1 {
		t.Fatalf("expected 1 span, got %d", len(spans))
	}

	span := spans[0]

	// Verify span has error status (even though no Go error was returned)
	if span.Status.Code != codes.Error {
		t.Errorf("expected span status Error, got %v", span.Status.Code)
	}

	// Verify status message
	expectedMsg := "tool returned error result"
	if span.Status.Description != expectedMsg {
		t.Errorf("expected status description %q, got %q", expectedMsg, span.Status.Description)
	}

	// Verify mcp.status attribute is set
	attrMap := make(map[string]string)
	for _, attr := range span.Attributes {
		attrMap[string(attr.Key)] = attr.Value.AsString()
	}

	if attrMap[instrumentation.SpanAttrStatus] != "error" {
		t.Errorf("expected mcp.status='error', got %v", attrMap[instrumentation.SpanAttrStatus])
	}
}
