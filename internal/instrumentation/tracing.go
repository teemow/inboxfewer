package instrumentation

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// TracerName is the default tracer name for the inboxfewer package.
const TracerName = "github.com/teemow/inboxfewer"

// Span attribute keys for operations.
const (
	// SpanAttrTool is the MCP tool name attribute.
	SpanAttrTool = "mcp.tool"

	// SpanAttrService is the Google service name attribute.
	SpanAttrService = "google.service"

	// SpanAttrOperation is the operation type attribute.
	SpanAttrOperation = "google.operation"

	// SpanAttrAccount is the user account/email attribute.
	SpanAttrAccount = "mcp.account"

	// SpanAttrStatus is the operation status attribute.
	SpanAttrStatus = "mcp.status"

	// SpanAttrResourceID is the resource identifier (email ID, event ID, etc.).
	SpanAttrResourceID = "mcp.resource_id"

	// SpanAttrResourceType is the resource type (email, event, file, etc.).
	SpanAttrResourceType = "mcp.resource_type"

	// SpanAttrReadOnly indicates if the operation is read-only.
	SpanAttrReadOnly = "mcp.read_only"
)

// SpanAttributeBuilder helps construct OpenTelemetry span attributes
// with consistent naming.
type SpanAttributeBuilder struct {
	attrs []attribute.KeyValue
}

// NewSpanAttributeBuilder creates a new SpanAttributeBuilder.
func NewSpanAttributeBuilder() *SpanAttributeBuilder {
	return &SpanAttributeBuilder{
		attrs: make([]attribute.KeyValue, 0, 10),
	}
}

// WithTool adds the MCP tool name attribute.
func (b *SpanAttributeBuilder) WithTool(tool string) *SpanAttributeBuilder {
	b.attrs = append(b.attrs, attribute.String(SpanAttrTool, tool))
	return b
}

// WithService adds the Google service name attribute.
func (b *SpanAttributeBuilder) WithService(service string) *SpanAttributeBuilder {
	b.attrs = append(b.attrs, attribute.String(SpanAttrService, service))
	return b
}

// WithOperation adds the operation type attribute.
func (b *SpanAttributeBuilder) WithOperation(operation string) *SpanAttributeBuilder {
	b.attrs = append(b.attrs, attribute.String(SpanAttrOperation, operation))
	return b
}

// WithAccount adds the user account attribute.
func (b *SpanAttributeBuilder) WithAccount(account string) *SpanAttributeBuilder {
	if account != "" {
		b.attrs = append(b.attrs, attribute.String(SpanAttrAccount, account))
	}
	return b
}

// WithResource adds resource attributes.
func (b *SpanAttributeBuilder) WithResource(resourceType, resourceID string) *SpanAttributeBuilder {
	if resourceType != "" {
		b.attrs = append(b.attrs, attribute.String(SpanAttrResourceType, resourceType))
	}
	if resourceID != "" {
		b.attrs = append(b.attrs, attribute.String(SpanAttrResourceID, resourceID))
	}
	return b
}

// WithReadOnly adds the read-only indicator attribute.
func (b *SpanAttributeBuilder) WithReadOnly(readOnly bool) *SpanAttributeBuilder {
	b.attrs = append(b.attrs, attribute.Bool(SpanAttrReadOnly, readOnly))
	return b
}

// Build returns the constructed attributes.
func (b *SpanAttributeBuilder) Build() []attribute.KeyValue {
	return b.attrs
}

// StartSpan starts a new span with the given name and attributes.
// Returns the context with the span and the span itself.
// The caller is responsible for ending the span with defer span.End().
func StartSpan(ctx context.Context, name string, attrs ...attribute.KeyValue) (context.Context, trace.Span) {
	tracer := otel.GetTracerProvider().Tracer(TracerName)
	return tracer.Start(ctx, name, trace.WithAttributes(attrs...))
}

// StartToolSpan starts a span for an MCP tool invocation.
// Automatically adds tool name and sets appropriate span kind.
func StartToolSpan(ctx context.Context, toolName string, attrs ...attribute.KeyValue) (context.Context, trace.Span) {
	allAttrs := make([]attribute.KeyValue, 0, len(attrs)+1)
	allAttrs = append(allAttrs, attribute.String(SpanAttrTool, toolName))
	allAttrs = append(allAttrs, attrs...)

	tracer := otel.GetTracerProvider().Tracer(TracerName)
	return tracer.Start(ctx, "tool."+toolName,
		trace.WithAttributes(allAttrs...),
		trace.WithSpanKind(trace.SpanKindServer),
	)
}

// StartGoogleAPISpan starts a span for Google API operations.
// Includes service and operation attributes.
func StartGoogleAPISpan(ctx context.Context, service, operation string, attrs ...attribute.KeyValue) (context.Context, trace.Span) {
	allAttrs := make([]attribute.KeyValue, 0, len(attrs)+2)
	allAttrs = append(allAttrs,
		attribute.String(SpanAttrService, service),
		attribute.String(SpanAttrOperation, operation),
	)
	allAttrs = append(allAttrs, attrs...)

	tracer := otel.GetTracerProvider().Tracer(TracerName)
	return tracer.Start(ctx, "google."+service+"."+operation,
		trace.WithAttributes(allAttrs...),
		trace.WithSpanKind(trace.SpanKindClient),
	)
}

// SetSpanError records an error on the span and sets the status to error.
func SetSpanError(span trace.Span, err error) {
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	}
}

// SetSpanSuccess sets the span status to OK.
func SetSpanSuccess(span trace.Span) {
	span.SetStatus(codes.Ok, "")
}

// AddSpanEvent adds an event to the span with optional attributes.
func AddSpanEvent(span trace.Span, name string, attrs ...attribute.KeyValue) {
	span.AddEvent(name, trace.WithAttributes(attrs...))
}

// GetTraceID returns the trace ID from the current span in context.
// Returns empty string if no valid span is present.
func GetTraceID(ctx context.Context) string {
	span := trace.SpanFromContext(ctx)
	if span.SpanContext().IsValid() {
		return span.SpanContext().TraceID().String()
	}
	return ""
}

// GetSpanID returns the span ID from the current span in context.
// Returns empty string if no valid span is present.
func GetSpanID(ctx context.Context) string {
	span := trace.SpanFromContext(ctx)
	if span.SpanContext().IsValid() {
		return span.SpanContext().SpanID().String()
	}
	return ""
}

// SpanContextString returns a human-readable trace context string.
// Format: "trace_id=X span_id=Y" or empty string if no valid context.
func SpanContextString(ctx context.Context) string {
	span := trace.SpanFromContext(ctx)
	if !span.SpanContext().IsValid() {
		return ""
	}
	return "trace_id=" + span.SpanContext().TraceID().String() +
		" span_id=" + span.SpanContext().SpanID().String()
}
