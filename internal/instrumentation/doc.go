// Package instrumentation provides comprehensive OpenTelemetry instrumentation
// for the inboxfewer MCP server.
//
// This package enables production-grade observability through:
//   - OpenTelemetry metrics for HTTP requests, OAuth operations, and Google API calls
//   - Distributed tracing for request flows and API calls
//   - Prometheus metrics export via /metrics endpoint on dedicated port
//   - OTLP export support for modern observability platforms
//
// # Metrics
//
// The package exposes the following metric categories:
//
// Server/HTTP Metrics:
//   - http_requests_total: Counter of HTTP requests by method, path, and status
//   - http_request_duration_seconds: Histogram of HTTP request durations
//   - active_sessions: Gauge of active user sessions
//
// Google API Metrics:
//   - google_api_operations_total: Counter of Google API operations by service, operation, status
//   - google_api_operation_duration_seconds: Histogram of Google API operation durations
//
// OAuth Authentication Metrics:
//   - oauth_auth_total: Counter of OAuth authentication events by result
//   - oauth_token_refresh_total: Counter of token refresh attempts by result
//
// MCP Tool Metrics:
//   - mcp_tool_invocations_total: Counter of MCP tool invocations by tool name and status
//   - mcp_tool_duration_seconds: Histogram of MCP tool execution durations
//
// # Tracing
//
// Distributed tracing spans are created for:
//   - HTTP request handling
//   - MCP tool invocations (tool.<name>)
//   - Google API calls (google.<service>.<operation>)
//   - OAuth token operations
//
// # Configuration
//
// Instrumentation can be configured via environment variables:
//   - INSTRUMENTATION_ENABLED: Enable/disable instrumentation (default: true)
//   - METRICS_EXPORTER: Metrics exporter type (prometheus, otlp, stdout, default: prometheus)
//   - TRACING_EXPORTER: Tracing exporter type (otlp, stdout, none, default: none)
//   - OTEL_EXPORTER_OTLP_ENDPOINT: OTLP endpoint for traces/metrics
//   - OTEL_TRACES_SAMPLER_ARG: Sampling rate (0.0 to 1.0, default: 0.1)
//   - OTEL_SERVICE_NAME: Service name (default: inboxfewer)
//
// # Example Usage
//
//	// Initialize instrumentation
//	provider, err := instrumentation.NewProvider(ctx, instrumentation.Config{
//		ServiceName:    "inboxfewer",
//		ServiceVersion: "0.1.0",
//		Enabled:        true,
//	})
//	if err != nil {
//		return err
//	}
//	defer provider.Shutdown(ctx)
//
//	// Get metrics recorder
//	recorder := provider.Metrics()
//
//	// Record an HTTP request
//	recorder.RecordHTTPRequest(ctx, "POST", "/mcp", 200, time.Since(start))
//
//	// Record a Google API operation
//	recorder.RecordGoogleAPIOperation(ctx, "gmail", "list", "success", time.Since(start))
//
//	// Record an MCP tool invocation
//	recorder.RecordToolInvocation(ctx, "gmail_list_emails", "success", time.Since(start))
package instrumentation
