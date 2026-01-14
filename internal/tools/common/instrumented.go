package common

import (
	"context"
	"time"

	"github.com/mark3labs/mcp-go/mcp"

	"github.com/teemow/inboxfewer/internal/instrumentation"
	"github.com/teemow/inboxfewer/internal/server"
)

// InstrumentedToolHandler wraps a tool handler with metrics and audit logging.
// It records tool invocation metrics and logs the invocation for audit purposes.
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
		// Get metrics and audit logger (may be nil if not configured)
		metrics := sc.Metrics()
		auditLogger := sc.AuditLogger()

		// If no instrumentation configured, just call the handler
		if metrics == nil && auditLogger == nil {
			return handler(ctx, request)
		}

		// Start timing and create invocation record
		start := time.Now()
		invocation := instrumentation.NewToolInvocation(toolName).
			WithSpanContext(ctx)

		// Extract account from request arguments
		args := request.GetArguments()
		account := GetAccountFromArgs(ctx, args)
		if account != "" {
			invocation.WithAccount(account)
		}

		// Call the actual handler
		result, err := handler(ctx, request)
		duration := time.Since(start)

		// Determine status
		status := instrumentation.StatusSuccess
		if err != nil || (result != nil && result.IsError) {
			status = instrumentation.StatusError
			if err != nil {
				invocation.CompleteWithError(err)
			} else {
				invocation.Complete(false, nil)
			}
		} else {
			invocation.CompleteSuccess()
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
// records the Google service and operation type for more detailed metrics.
//
// This handler records both:
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
		// Get metrics and audit logger (may be nil if not configured)
		metrics := sc.Metrics()
		auditLogger := sc.AuditLogger()

		// If no instrumentation configured, just call the handler
		if metrics == nil && auditLogger == nil {
			return handler(ctx, request)
		}

		// Start timing and create invocation record
		start := time.Now()
		invocation := instrumentation.NewToolInvocation(toolName).
			WithSpanContext(ctx).
			WithService(serviceName, operation)

		// Extract account from request arguments
		args := request.GetArguments()
		account := GetAccountFromArgs(ctx, args)
		if account != "" {
			invocation.WithAccount(account)
		}

		// Call the actual handler
		result, err := handler(ctx, request)
		duration := time.Since(start)

		// Determine status
		status := instrumentation.StatusSuccess
		if err != nil || (result != nil && result.IsError) {
			status = instrumentation.StatusError
			if err != nil {
				invocation.CompleteWithError(err)
			} else {
				invocation.Complete(false, nil)
			}
		} else {
			invocation.CompleteSuccess()
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
