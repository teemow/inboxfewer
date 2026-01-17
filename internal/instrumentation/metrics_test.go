package instrumentation

import (
	"context"
	"testing"
	"time"
)

func TestMetrics_RecordHTTPRequest(t *testing.T) {
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

	metrics := provider.Metrics()
	if metrics == nil {
		t.Fatal("expected metrics to be non-nil")
	}

	// Should not panic
	metrics.RecordHTTPRequest(ctx, "GET", "/mcp", 200, 100*time.Millisecond)
	metrics.RecordHTTPRequest(ctx, "POST", "/mcp", 500, 50*time.Millisecond)
}

func TestMetrics_RecordGoogleAPIOperation(t *testing.T) {
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

	metrics := provider.Metrics()

	// Should not panic
	metrics.RecordGoogleAPIOperation(ctx, ServiceGmail, "list", StatusSuccess, 200*time.Millisecond)
	metrics.RecordGoogleAPIOperation(ctx, ServiceCalendar, "create", StatusError, 500*time.Millisecond)
	metrics.RecordGoogleAPIOperation(ctx, ServiceDrive, "get", StatusSuccess, 100*time.Millisecond)
}

func TestMetrics_RecordOAuthAuth(t *testing.T) {
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

	metrics := provider.Metrics()

	// Should not panic
	metrics.RecordOAuthAuth(ctx, OAuthResultSuccess)
	metrics.RecordOAuthAuth(ctx, OAuthResultFailure)
}

func TestMetrics_RecordOAuthTokenRefresh(t *testing.T) {
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

	metrics := provider.Metrics()

	// Should not panic
	metrics.RecordOAuthTokenRefresh(ctx, OAuthResultSuccess)
	metrics.RecordOAuthTokenRefresh(ctx, OAuthResultFailure)
	metrics.RecordOAuthTokenRefresh(ctx, OAuthResultExpired)
}

func TestMetrics_RecordOAuthCrossClientToken(t *testing.T) {
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

	metrics := provider.Metrics()

	// Should not panic
	metrics.RecordOAuthCrossClientToken(ctx, "accepted", "muster-client")
	metrics.RecordOAuthCrossClientToken(ctx, "rejected", "unknown-client")
}

func TestMetrics_RecordOAuthCrossClientToken_DetailedLabels(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Test with detailed labels enabled
	provider, err := NewProvider(ctx, Config{
		ServiceName:     "test-service",
		ServiceVersion:  "1.0.0",
		Enabled:         true,
		MetricsExporter: "prometheus",
		TracingExporter: "none",
		DetailedLabels:  true,
	})
	if err != nil {
		t.Fatalf("failed to create provider: %v", err)
	}
	defer func() { _ = provider.Shutdown(ctx) }()

	metrics := provider.Metrics()

	// Should not panic - audience should be included
	metrics.RecordOAuthCrossClientToken(ctx, "accepted", "muster-client")
}

func TestMetrics_RecordToolInvocation(t *testing.T) {
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

	metrics := provider.Metrics()

	// Should not panic
	metrics.RecordToolInvocation(ctx, "gmail_list_emails", StatusSuccess, 100*time.Millisecond)
	metrics.RecordToolInvocation(ctx, "calendar_create_event", StatusError, 500*time.Millisecond)
}

func TestMetrics_RecordToolInvocationWithAccount(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Test without detailed labels
	provider, err := NewProvider(ctx, Config{
		ServiceName:     "test-service",
		ServiceVersion:  "1.0.0",
		Enabled:         true,
		MetricsExporter: "prometheus",
		TracingExporter: "none",
		DetailedLabels:  false,
	})
	if err != nil {
		t.Fatalf("failed to create provider: %v", err)
	}
	defer func() { _ = provider.Shutdown(ctx) }()

	metrics := provider.Metrics()

	// Should not panic - account should be ignored
	metrics.RecordToolInvocationWithAccount(ctx, "gmail_list_emails", StatusSuccess, "user@example.com", 100*time.Millisecond)
}

func TestMetrics_RecordToolInvocationWithAccount_DetailedLabels(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Test with detailed labels
	provider, err := NewProvider(ctx, Config{
		ServiceName:     "test-service",
		ServiceVersion:  "1.0.0",
		Enabled:         true,
		MetricsExporter: "prometheus",
		TracingExporter: "none",
		DetailedLabels:  true,
	})
	if err != nil {
		t.Fatalf("failed to create provider: %v", err)
	}
	defer func() { _ = provider.Shutdown(ctx) }()

	metrics := provider.Metrics()

	// Should not panic - account should be included
	metrics.RecordToolInvocationWithAccount(ctx, "gmail_list_emails", StatusSuccess, "user@example.com", 100*time.Millisecond)
}

func TestMetrics_ActiveSessions(t *testing.T) {
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

	metrics := provider.Metrics()

	// Should not panic
	metrics.IncrementActiveSessions(ctx)
	metrics.IncrementActiveSessions(ctx)
	metrics.DecrementActiveSessions(ctx)
}

func TestMetrics_NoOp_WhenDisabled(t *testing.T) {
	ctx := context.Background()

	provider, err := NewProvider(ctx, Config{
		ServiceName:    "test-service",
		ServiceVersion: "1.0.0",
		Enabled:        false,
	})
	if err != nil {
		t.Fatalf("failed to create provider: %v", err)
	}

	metrics := provider.Metrics()
	if metrics == nil {
		t.Fatal("expected metrics to be non-nil even when disabled")
	}

	// All these should not panic even with nil underlying metrics
	metrics.RecordHTTPRequest(ctx, "GET", "/mcp", 200, 100*time.Millisecond)
	metrics.RecordGoogleAPIOperation(ctx, ServiceGmail, "list", StatusSuccess, 200*time.Millisecond)
	metrics.RecordOAuthAuth(ctx, OAuthResultSuccess)
	metrics.RecordOAuthTokenRefresh(ctx, OAuthResultSuccess)
	metrics.RecordOAuthCrossClientToken(ctx, "accepted", "muster-client")
	metrics.RecordToolInvocation(ctx, "test_tool", StatusSuccess, 100*time.Millisecond)
	metrics.RecordToolInvocationWithAccount(ctx, "test_tool", StatusSuccess, "user@example.com", 100*time.Millisecond)
	metrics.IncrementActiveSessions(ctx)
	metrics.DecrementActiveSessions(ctx)
}
