package instrumentation

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

// Metric attribute keys - using constants for consistency and DRY
const (
	// Common attributes (reused across metrics)
	attrMethod    = "method"
	attrPath      = "path"
	attrStatus    = "status"
	attrOperation = "operation"
	attrService   = "service"
	attrResult    = "result"
	attrTool      = "tool"
	attrAccount   = "account"
)

// Metrics provides methods for recording observability metrics.
type Metrics struct {
	// HTTP metrics
	httpRequestsTotal   metric.Int64Counter
	httpRequestDuration metric.Float64Histogram
	activeSessions      metric.Int64UpDownCounter

	// Google API metrics
	googleAPIOperationsTotal   metric.Int64Counter
	googleAPIOperationDuration metric.Float64Histogram

	// OAuth metrics
	oauthAuthTotal         metric.Int64Counter
	oauthTokenRefreshTotal metric.Int64Counter

	// MCP Tool metrics
	toolInvocationsTotal metric.Int64Counter
	toolDuration         metric.Float64Histogram

	// Configuration
	// detailedLabels controls whether high-cardinality labels are included
	detailedLabels bool
}

// NewMetrics creates a new Metrics instance with all metrics initialized.
// The detailedLabels parameter controls whether high-cardinality labels are included.
func NewMetrics(meter metric.Meter, detailedLabels bool) (*Metrics, error) {
	m := &Metrics{
		detailedLabels: detailedLabels,
	}

	var err error

	// HTTP Metrics
	m.httpRequestsTotal, err = meter.Int64Counter(
		"http_requests_total",
		metric.WithDescription("Total number of HTTP requests"),
		metric.WithUnit("{request}"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create http_requests_total counter: %w", err)
	}

	m.httpRequestDuration, err = meter.Float64Histogram(
		"http_request_duration_seconds",
		metric.WithDescription("HTTP request duration in seconds"),
		metric.WithUnit("s"),
		metric.WithExplicitBucketBoundaries(0.001, 0.01, 0.1, 0.5, 1.0, 2.5, 5.0, 10.0),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create http_request_duration_seconds histogram: %w", err)
	}

	m.activeSessions, err = meter.Int64UpDownCounter(
		"active_sessions",
		metric.WithDescription("Number of active user sessions"),
		metric.WithUnit("{session}"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create active_sessions gauge: %w", err)
	}

	// Google API Metrics
	m.googleAPIOperationsTotal, err = meter.Int64Counter(
		"google_api_operations_total",
		metric.WithDescription("Total number of Google API operations"),
		metric.WithUnit("{operation}"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create google_api_operations_total counter: %w", err)
	}

	m.googleAPIOperationDuration, err = meter.Float64Histogram(
		"google_api_operation_duration_seconds",
		metric.WithDescription("Google API operation duration in seconds"),
		metric.WithUnit("s"),
		metric.WithExplicitBucketBoundaries(0.01, 0.05, 0.1, 0.25, 0.5, 1.0, 2.5, 5.0, 10.0, 30.0),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create google_api_operation_duration_seconds histogram: %w", err)
	}

	// OAuth Metrics
	m.oauthAuthTotal, err = meter.Int64Counter(
		"oauth_auth_total",
		metric.WithDescription("Total number of OAuth authentication attempts"),
		metric.WithUnit("{attempt}"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create oauth_auth_total counter: %w", err)
	}

	m.oauthTokenRefreshTotal, err = meter.Int64Counter(
		"oauth_token_refresh_total",
		metric.WithDescription("Total number of OAuth token refresh attempts"),
		metric.WithUnit("{attempt}"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create oauth_token_refresh_total counter: %w", err)
	}

	// MCP Tool Metrics
	m.toolInvocationsTotal, err = meter.Int64Counter(
		"mcp_tool_invocations_total",
		metric.WithDescription("Total number of MCP tool invocations"),
		metric.WithUnit("{invocation}"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create mcp_tool_invocations_total counter: %w", err)
	}

	m.toolDuration, err = meter.Float64Histogram(
		"mcp_tool_duration_seconds",
		metric.WithDescription("MCP tool execution duration in seconds"),
		metric.WithUnit("s"),
		metric.WithExplicitBucketBoundaries(0.01, 0.05, 0.1, 0.25, 0.5, 1.0, 2.5, 5.0, 10.0, 30.0),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create mcp_tool_duration_seconds histogram: %w", err)
	}

	return m, nil
}

// RecordHTTPRequest records an HTTP request with method, path, status code, and duration.
func (m *Metrics) RecordHTTPRequest(ctx context.Context, method, path string, statusCode int, duration time.Duration) {
	if m.httpRequestsTotal == nil || m.httpRequestDuration == nil {
		return // Instrumentation not initialized
	}

	attrs := []attribute.KeyValue{
		attribute.String(attrMethod, method),
		attribute.String(attrPath, path),
		attribute.String(attrStatus, strconv.Itoa(statusCode)),
	}

	m.httpRequestsTotal.Add(ctx, 1, metric.WithAttributes(attrs...))
	m.httpRequestDuration.Record(ctx, duration.Seconds(), metric.WithAttributes(attrs...))
}

// RecordGoogleAPIOperation records a Google API operation with service, operation,
// status, and duration.
//
// Parameters:
//   - service: Google service name (gmail, calendar, drive, docs, meet, tasks)
//   - operation: Operation type (list, get, create, update, delete, send, etc.)
//   - status: Result status ("success" or "error")
//   - duration: Time taken for the operation
func (m *Metrics) RecordGoogleAPIOperation(ctx context.Context, service, operation, status string, duration time.Duration) {
	if m.googleAPIOperationsTotal == nil || m.googleAPIOperationDuration == nil {
		return // Instrumentation not initialized
	}

	attrs := []attribute.KeyValue{
		attribute.String(attrService, service),
		attribute.String(attrOperation, operation),
		attribute.String(attrStatus, status),
	}

	m.googleAPIOperationsTotal.Add(ctx, 1, metric.WithAttributes(attrs...))
	m.googleAPIOperationDuration.Record(ctx, duration.Seconds(), metric.WithAttributes(attrs...))
}

// RecordOAuthAuth records an OAuth authentication attempt with result.
// Result should be one of: "success", "failure"
func (m *Metrics) RecordOAuthAuth(ctx context.Context, result string) {
	if m.oauthAuthTotal == nil {
		return // Instrumentation not initialized
	}

	attrs := []attribute.KeyValue{
		attribute.String(attrResult, result),
	}

	m.oauthAuthTotal.Add(ctx, 1, metric.WithAttributes(attrs...))
}

// RecordOAuthTokenRefresh records an OAuth token refresh attempt with result.
// Result should be one of: "success", "failure", "expired"
func (m *Metrics) RecordOAuthTokenRefresh(ctx context.Context, result string) {
	if m.oauthTokenRefreshTotal == nil {
		return // Instrumentation not initialized
	}

	attrs := []attribute.KeyValue{
		attribute.String(attrResult, result),
	}

	m.oauthTokenRefreshTotal.Add(ctx, 1, metric.WithAttributes(attrs...))
}

// RecordToolInvocation records an MCP tool invocation with tool name, status, and duration.
//
// Parameters:
//   - toolName: Name of the MCP tool (e.g., "gmail_list_emails", "calendar_create_event")
//   - status: Result status ("success" or "error")
//   - duration: Time taken for the tool execution
func (m *Metrics) RecordToolInvocation(ctx context.Context, toolName, status string, duration time.Duration) {
	if m.toolInvocationsTotal == nil || m.toolDuration == nil {
		return // Instrumentation not initialized
	}

	attrs := []attribute.KeyValue{
		attribute.String(attrTool, toolName),
		attribute.String(attrStatus, status),
	}

	m.toolInvocationsTotal.Add(ctx, 1, metric.WithAttributes(attrs...))
	m.toolDuration.Record(ctx, duration.Seconds(), metric.WithAttributes(attrs...))
}

// IncrementActiveSessions increments the active sessions counter.
func (m *Metrics) IncrementActiveSessions(ctx context.Context) {
	if m.activeSessions == nil {
		return // Instrumentation not initialized
	}

	m.activeSessions.Add(ctx, 1)
}

// DecrementActiveSessions decrements the active sessions counter.
func (m *Metrics) DecrementActiveSessions(ctx context.Context) {
	if m.activeSessions == nil {
		return // Instrumentation not initialized
	}

	m.activeSessions.Add(ctx, -1)
}

// RecordToolInvocationWithAccount records an MCP tool invocation with account info.
// This is the detailed version that includes account information when detailedLabels is enabled.
//
// Parameters:
//   - toolName: Name of the MCP tool
//   - status: Result status ("success" or "error")
//   - account: User account/email (only included if detailedLabels is true)
//   - duration: Time taken for the tool execution
func (m *Metrics) RecordToolInvocationWithAccount(ctx context.Context, toolName, status, account string, duration time.Duration) {
	if m.toolInvocationsTotal == nil || m.toolDuration == nil {
		return // Instrumentation not initialized
	}

	attrs := []attribute.KeyValue{
		attribute.String(attrTool, toolName),
		attribute.String(attrStatus, status),
	}

	// Only add high-cardinality labels if explicitly enabled
	if m.detailedLabels && account != "" {
		attrs = append(attrs, attribute.String(attrAccount, account))
	}

	m.toolInvocationsTotal.Add(ctx, 1, metric.WithAttributes(attrs...))
	m.toolDuration.Record(ctx, duration.Seconds(), metric.WithAttributes(attrs...))
}
