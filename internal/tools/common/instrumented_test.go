package common

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"go.opentelemetry.io/otel/metric/noop"

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
