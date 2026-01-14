package common

import (
	"context"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"go.opentelemetry.io/otel/attribute"

	"github.com/teemow/inboxfewer/internal/instrumentation"
	"github.com/teemow/inboxfewer/internal/server"
)

// InstrumentedToolHandler wraps a tool handler with tracing, metrics and audit logging.
// It creates a trace span, records tool invocation metrics, and logs the invocation
// for audit purposes.
//
// Usage:
//
//	s.AddTool(myTool, common.InstrumentedToolHandler("my_tool", sc, handler))
func InstrumentedToolHandler(
	toolName string,
	sc *server.ServerContext,
	handler func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error),
) func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		// Start trace span for this tool invocation
		ctx, span := instrumentation.StartToolSpan(ctx, toolName)
		defer span.End()

		// Get metrics and audit logger (may be nil if not configured)
		metrics := sc.Metrics()
		auditLogger := sc.AuditLogger()

		// Start timing and create invocation record
		// Note: We create the invocation after starting the span so it captures the new span context
		start := time.Now()
		invocation := instrumentation.NewToolInvocation(toolName).
			WithSpanContext(ctx)

		// Extract account from request arguments
		args := request.GetArguments()
		account := GetAccountFromArgs(ctx, args)
		if account != "" {
			invocation.WithAccount(account)
			span.SetAttributes(attribute.String(instrumentation.SpanAttrAccount, account))
		}

		// Call the actual handler
		result, err := handler(ctx, request)
		duration := time.Since(start)

		// Determine status and update span
		status := instrumentation.StatusSuccess
		if err != nil || (result != nil && result.IsError) {
			status = instrumentation.StatusError
			if err != nil {
				invocation.CompleteWithError(err)
				instrumentation.SetSpanError(span, err)
			} else {
				invocation.Complete(false, nil)
				// For error results without Go errors, still mark span as error
				span.SetAttributes(attribute.String("mcp.status", "error"))
			}
		} else {
			invocation.CompleteSuccess()
			instrumentation.SetSpanSuccess(span)
		}

		// Record metrics
		if metrics != nil {
			metrics.RecordToolInvocation(ctx, toolName, status, duration)
		}

		// Log audit
		if auditLogger != nil {
			auditLogger.LogToolInvocation(invocation)
		}

		return result, err
	}
}

// InstrumentedToolHandlerWithService is like InstrumentedToolHandler but also
// records the Google service and operation type for more detailed metrics and tracing.
//
// This handler creates a trace span with service attributes and records both:
// - MCP tool invocation metrics (mcp_tool_invocations_total, mcp_tool_duration_seconds)
// - Google API operation metrics (google_api_operations_total, google_api_operation_duration_seconds)
//
// Usage:
//
//	s.AddTool(myTool, common.InstrumentedToolHandlerWithService("my_tool", "gmail", "list", sc, handler))
func InstrumentedToolHandlerWithService(
	toolName string,
	serviceName string,
	operation string,
	sc *server.ServerContext,
	handler func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error),
) func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		// Build span attributes for service-specific tracing
		spanAttrs := instrumentation.NewSpanAttributeBuilder().
			WithService(serviceName).
			WithOperation(operation).
			Build()

		// Start trace span for this tool invocation with service attributes
		ctx, span := instrumentation.StartToolSpan(ctx, toolName, spanAttrs...)
		defer span.End()

		// Get metrics and audit logger (may be nil if not configured)
		metrics := sc.Metrics()
		auditLogger := sc.AuditLogger()

		// Start timing and create invocation record
		// Note: We create the invocation after starting the span so it captures the new span context
		start := time.Now()
		invocation := instrumentation.NewToolInvocation(toolName).
			WithSpanContext(ctx).
			WithService(serviceName, operation)

		// Extract account from request arguments
		args := request.GetArguments()
		account := GetAccountFromArgs(ctx, args)
		if account != "" {
			invocation.WithAccount(account)
			span.SetAttributes(attribute.String(instrumentation.SpanAttrAccount, account))
		}

		// Call the actual handler
		result, err := handler(ctx, request)
		duration := time.Since(start)

		// Determine status and update span
		status := instrumentation.StatusSuccess
		if err != nil || (result != nil && result.IsError) {
			status = instrumentation.StatusError
			if err != nil {
				invocation.CompleteWithError(err)
				instrumentation.SetSpanError(span, err)
			} else {
				invocation.Complete(false, nil)
				// For error results without Go errors, still mark span as error
				span.SetAttributes(attribute.String("mcp.status", "error"))
			}
		} else {
			invocation.CompleteSuccess()
			instrumentation.SetSpanSuccess(span)
		}

		// Record metrics
		if metrics != nil {
			// Record MCP tool invocation metrics
			metrics.RecordToolInvocation(ctx, toolName, status, duration)

			// Record Google API operation metrics for detailed service-level observability
			// This provides insight into which Google services/operations are used most
			// and their performance characteristics
			metrics.RecordGoogleAPIOperation(ctx, serviceName, operation, status, duration)
		}

		// Log audit
		if auditLogger != nil {
			auditLogger.LogToolInvocation(invocation)
		}

		return result, err
	}
}
