# Observability Guide for inboxfewer

This document provides a comprehensive guide to the observability features of `inboxfewer`, including metrics, tracing, and example queries for monitoring production deployments.

## Overview

`inboxfewer` uses OpenTelemetry for comprehensive instrumentation, providing:

- **Metrics**: Prometheus-compatible metrics for HTTP requests, Google API operations, OAuth tokens, and MCP tools
- **Distributed Tracing**: OpenTelemetry traces for request flows (see [issue #74](https://github.com/teemow/inboxfewer/issues/74))
- **Prometheus Integration**: Dedicated metrics server on port 9090
- **Health Checks**: `/healthz` (liveness) and `/readyz` (readiness) endpoints
- **ServiceMonitor**: Optional Prometheus Operator ServiceMonitor CRD

## Configuration

Instrumentation is configured via environment variables:

```bash
# Enable/disable instrumentation (default: true)
INSTRUMENTATION_ENABLED=true

# Metrics exporter type (default: prometheus)
# Options: prometheus, otlp, stdout
METRICS_EXPORTER=prometheus

# Tracing exporter type (default: none)
# Options: otlp, stdout, none
TRACING_EXPORTER=otlp

# OTLP endpoint for traces/metrics (required for otlp exporters)
# Format: hostname:port (without protocol prefix)
OTEL_EXPORTER_OTLP_ENDPOINT=localhost:4318

# Use insecure (HTTP) transport for OTLP (default: false, uses HTTPS)
# WARNING: Only enable for local development/testing
OTEL_EXPORTER_OTLP_INSECURE=false

# Sampling rate (0.0 to 1.0, default: 0.1)
OTEL_TRACES_SAMPLER_ARG=0.1

# Service name (default: inboxfewer)
OTEL_SERVICE_NAME=inboxfewer

# Enable detailed labels (account info) in metrics
# WARNING: Can cause high cardinality with many users
METRICS_DETAILED_LABELS=false

# Metrics server configuration
METRICS_ENABLED=true
METRICS_ADDR=:9090

# Kubernetes metadata (automatically set by Helm chart)
K8S_NAMESPACE=default
K8S_POD_NAME=inboxfewer-abc123
```

## Available Metrics

### HTTP Server Metrics

#### `http_requests_total`
Counter of HTTP requests.

**Labels:**
- `method`: HTTP method (GET, POST, etc.)
- `path`: Request path (/mcp, /oauth/*, etc.)
- `status`: HTTP status code

**Example:**
```promql
# Total requests to /mcp endpoint
http_requests_total{path="/mcp"}

# Error rate (5xx responses)
rate(http_requests_total{status=~"5.."}[5m])
```

#### `http_request_duration_seconds`
Histogram of HTTP request durations.

**Labels:**
- `method`: HTTP method
- `path`: Request path
- `status`: HTTP status code

**Buckets:** 0.001, 0.01, 0.1, 0.5, 1.0, 2.5, 5.0, 10.0 seconds

**Example:**
```promql
# 95th percentile request duration
histogram_quantile(0.95, 
  rate(http_request_duration_seconds_bucket[5m])
)

# Average request duration for MCP endpoint
rate(http_request_duration_seconds_sum{path="/mcp"}[5m])
/ rate(http_request_duration_seconds_count{path="/mcp"}[5m])
```

### Google API Operation Metrics

#### `google_api_operations_total`
Counter of Google API operations.

**Labels:**
- `service`: Google service name (gmail, calendar, drive, docs, meet, tasks)
- `operation`: Operation type (list, get, create, update, delete, send, etc.)
- `status`: Operation result (success, error)

**Example:**
```promql
# Total Gmail operations
google_api_operations_total{service="gmail"}

# Error rate for Calendar API
rate(google_api_operations_total{service="calendar", status="error"}[5m])
/ rate(google_api_operations_total{service="calendar"}[5m])

# Operations by service
sum by (service) (rate(google_api_operations_total[1m]))
```

#### `google_api_operation_duration_seconds`
Histogram of Google API operation durations.

**Labels:**
- `service`: Google service name
- `operation`: Operation type
- `status`: Operation result

**Buckets:** 0.01, 0.05, 0.1, 0.25, 0.5, 1.0, 2.5, 5.0, 10.0, 30.0 seconds

**Example:**
```promql
# 99th percentile Gmail API duration
histogram_quantile(0.99, 
  rate(google_api_operation_duration_seconds_bucket{service="gmail"}[5m])
)

# Slow operations (>2s)
histogram_quantile(0.95, 
  rate(google_api_operation_duration_seconds_bucket[5m])
) > 2
```

### MCP Tool Metrics

#### `mcp_tool_invocations_total`
Counter of MCP tool invocations.

**Labels:**
- `tool`: Tool name (gmail_list_emails, calendar_create_event, etc.)
- `status`: Invocation result (success, error)

**Example:**
```promql
# Most used tools
topk(10, sum by (tool) (rate(mcp_tool_invocations_total[1h])))

# Tool error rate
sum by (tool) (rate(mcp_tool_invocations_total{status="error"}[5m]))
/ sum by (tool) (rate(mcp_tool_invocations_total[5m]))
```

#### `mcp_tool_duration_seconds`
Histogram of MCP tool execution durations.

**Labels:**
- `tool`: Tool name
- `status`: Execution result

**Buckets:** 0.01, 0.05, 0.1, 0.25, 0.5, 1.0, 2.5, 5.0, 10.0, 30.0 seconds

**Example:**
```promql
# P95 tool duration by tool
histogram_quantile(0.95, 
  sum by (le, tool) (rate(mcp_tool_duration_seconds_bucket[5m]))
)

# Average tool duration
rate(mcp_tool_duration_seconds_sum[5m])
/ rate(mcp_tool_duration_seconds_count[5m])
```

### OAuth Metrics

#### `oauth_auth_total`
Counter of OAuth authentication attempts.

**Labels:**
- `result`: Authentication result (success, failure)

**Example:**
```promql
# Total successful authentications
oauth_auth_total{result="success"}

# Authentication failure rate
rate(oauth_auth_total{result="failure"}[5m])
/ rate(oauth_auth_total[5m])
```

#### `oauth_token_refresh_total`
Counter of OAuth token refresh attempts.

**Labels:**
- `result`: Refresh result (success, failure, expired)

**Example:**
```promql
# Token refresh success rate
oauth_token_refresh_total{result="success"}
/ sum(oauth_token_refresh_total)

# Expired token retrievals (indicates tokens needing refresh)
rate(oauth_token_refresh_total{result="expired"}[5m])
```

### Session Metrics

#### `active_sessions`
Gauge of active user sessions.

**Example:**
```promql
# Current active sessions
active_sessions

# Average active sessions over time
avg_over_time(active_sessions[1h])
```

## Example Prometheus Queries

### Service Health

```promql
# Request success rate (non-5xx responses)
sum(rate(http_requests_total{status!~"5.."}[5m]))
/ sum(rate(http_requests_total[5m]))

# Google API error rate
sum(rate(google_api_operations_total{status="error"}[5m]))
/ sum(rate(google_api_operations_total[5m]))

# Tool success rate
sum(rate(mcp_tool_invocations_total{status="success"}[5m]))
/ sum(rate(mcp_tool_invocations_total[5m]))
```

### Performance Monitoring

```promql
# P95 HTTP request duration
histogram_quantile(0.95, 
  sum by (le) (rate(http_request_duration_seconds_bucket[5m]))
)

# P95 Google API operation duration by service
histogram_quantile(0.95, 
  sum by (le, service) (
    rate(google_api_operation_duration_seconds_bucket[5m])
  )
)

# Slowest tools (P99 duration)
histogram_quantile(0.99, 
  sum by (le, tool) (rate(mcp_tool_duration_seconds_bucket[5m]))
)
```

### Resource Usage

```promql
# Requests per second
rate(http_requests_total[1m])

# Google API operations per second by service
sum by (service) (rate(google_api_operations_total[1m]))

# Tool invocations per minute
sum by (tool) (rate(mcp_tool_invocations_total[1m])) * 60
```

## Prometheus Scraping Configuration

Add the following to your Prometheus configuration:

```yaml
scrape_configs:
  - job_name: 'inboxfewer'
    static_configs:
      - targets: ['inboxfewer:9090']
    metrics_path: '/metrics'
    scrape_interval: 15s
```

## Kubernetes ServiceMonitor

For Prometheus Operator, enable the ServiceMonitor in your Helm values:

```yaml
serviceMonitor:
  enabled: true
  interval: 15s
  labels:
    release: prometheus  # Match your Prometheus Operator selector
```

The Helm chart will create a ServiceMonitor CRD:

```yaml
apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  name: inboxfewer
spec:
  selector:
    matchLabels:
      app.kubernetes.io/name: inboxfewer
  endpoints:
    - port: metrics
      path: /metrics
      interval: 15s
```

## Example Alerts

### High Error Rate

```yaml
groups:
  - name: inboxfewer
    rules:
      - alert: InboxfewerHighErrorRate
        expr: |
          (
            sum(rate(http_requests_total{status=~"5.."}[5m]))
            / sum(rate(http_requests_total[5m]))
          ) > 0.05
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "High error rate in inboxfewer"
          description: "Error rate is {{ $value | humanizePercentage }}"
```

### Slow Google API Operations

```yaml
      - alert: SlowGoogleAPIOperations
        expr: |
          histogram_quantile(0.95, 
            sum by (le, service) (
              rate(google_api_operation_duration_seconds_bucket[5m])
            )
          ) > 5
        for: 10m
        labels:
          severity: warning
        annotations:
          summary: "Slow Google API operations detected"
          description: "P95 duration for {{ $labels.service }} is {{ $value }}s"
```

### OAuth Token Issues

```yaml
      - alert: OAuthTokenRefreshFailures
        expr: |
          rate(oauth_token_refresh_total{result="failure"}[5m]) > 0.1
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "OAuth token refresh failures detected"
          description: "Failure rate: {{ $value | humanize }} per second"

      - alert: HighExpiredTokenRate
        expr: |
          (
            rate(oauth_token_refresh_total{result="expired"}[5m])
            / rate(oauth_token_refresh_total[5m])
          ) > 0.5
        for: 10m
        labels:
          severity: info
        annotations:
          summary: "High rate of expired token retrievals"
          description: "{{ $value | humanizePercentage }} of token retrievals are for expired tokens"
```

### Tool Errors

```yaml
      - alert: ToolHighErrorRate
        expr: |
          (
            sum by (tool) (rate(mcp_tool_invocations_total{status="error"}[5m]))
            / sum by (tool) (rate(mcp_tool_invocations_total[5m]))
          ) > 0.1
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "High error rate for MCP tool"
          description: "Tool {{ $labels.tool }} has {{ $value | humanizePercentage }} error rate"
```

## Grafana Dashboards

### Key Panels

1. **Request Rate**: `rate(http_requests_total[1m])`
2. **Error Rate**: `rate(http_requests_total{status=~"5.."}[1m])`
3. **Request Duration (P50, P95, P99)**:
   ```promql
   histogram_quantile(0.50, rate(http_request_duration_seconds_bucket[5m]))
   histogram_quantile(0.95, rate(http_request_duration_seconds_bucket[5m]))
   histogram_quantile(0.99, rate(http_request_duration_seconds_bucket[5m]))
   ```
4. **Active Sessions**: `active_sessions`
5. **Google API Operations by Service**:
   ```promql
   sum by (service) (rate(google_api_operations_total[1m]))
   ```
6. **Top Tools by Usage**:
   ```promql
   topk(10, sum by (tool) (rate(mcp_tool_invocations_total[5m])))
   ```
7. **OAuth Health**:
   ```promql
   sum by (result) (rate(oauth_token_refresh_total[5m]))
   ```

### Example Dashboard JSON

A basic Grafana dashboard can be created with these panels. See the [Grafana documentation](https://grafana.com/docs/grafana/latest/dashboards/) for creating dashboards from PromQL queries.

## Tracing

> **Note**: Full distributed tracing is planned for a future release. See [issue #74](https://github.com/teemow/inboxfewer/issues/74).

When `TRACING_EXPORTER=otlp` is set, distributed traces can be exported to an OTLP collector (Jaeger, Tempo, etc.).

### Planned Trace Attributes

Traces will include the following attributes:

- `mcp.tool`: MCP tool name being executed
- `mcp.service`: Google service (gmail, calendar, drive, etc.)
- `mcp.operation`: Operation type (list, get, create, delete)
- `mcp.account`: User account (when available)
- `mcp.status`: Operation result

### Span Naming Convention

Spans will follow a consistent naming convention:

- `tool.<tool_name>`: MCP tool invocations (e.g., `tool.gmail_list_emails`)
- `google.<service>.<operation>`: Google API calls (e.g., `google.gmail.list`)

## Health Endpoints

`inboxfewer` exposes standard Kubernetes health check endpoints:

### Liveness Probe (`/healthz`)

Returns `200 OK` if the server process is running. Use for Kubernetes liveness probes.

```yaml
livenessProbe:
  httpGet:
    path: /healthz
    port: 8080
  initialDelaySeconds: 5
  periodSeconds: 10
```

### Readiness Probe (`/readyz`)

Returns `200 OK` if the server is ready to receive traffic. Checks:
- Server is not shutting down
- Required components are initialized

```yaml
readinessProbe:
  httpGet:
    path: /readyz
    port: 8080
  initialDelaySeconds: 5
  periodSeconds: 10
```

## Structured Audit Logging

`inboxfewer` provides structured JSON logging for tool invocations to support security auditing.

### Log Format

Every tool invocation produces a structured log entry:

```json
{
  "level": "info",
  "msg": "tool_invocation",
  "tool": "gmail_send_email",
  "service": "gmail",
  "operation": "send",
  "account": "user@example.com",
  "duration_ms": 523,
  "success": true
}
```

### Audit Log Fields

**Standard fields (cardinality-controlled):**
- `tool`: MCP tool name
- `service`: Google service name
- `operation`: Operation type
- `duration_ms`: Execution duration in milliseconds
- `success`: Boolean success indicator

**Optional fields:**
- `account`: User account (when detailedLabels enabled)
- `error`: Error message (when failed)
- `trace_id`: OpenTelemetry trace ID (when tracing enabled)

## Cardinality Management

### Detailed Labels

By default, metrics only include low-cardinality labels to prevent cardinality explosion with many users.

To enable detailed labels (includes account information):

```bash
METRICS_DETAILED_LABELS=true
```

**Warning**: With many users, detailed labels can cause:
- Prometheus memory issues
- Slow queries
- Storage bloat

For production with many users, use traces instead of detailed metrics for per-user debugging.

## Best Practices

1. **Sampling**: Set `OTEL_TRACES_SAMPLER_ARG` to an appropriate value (e.g., 0.1 for 10% sampling)
2. **Cardinality**: Keep `METRICS_DETAILED_LABELS=false` for multi-user deployments
3. **Retention**: Configure appropriate retention policies for metrics and traces
4. **Alerting**: Set up alerts for critical metrics (error rates, latency, OAuth failures)
5. **Dashboards**: Create Grafana dashboards for key metrics
6. **Health Checks**: Use `/healthz` and `/readyz` for Kubernetes probes
7. **Separate Ports**: Metrics are served on port 9090, separate from the main service on 8080

## Troubleshooting

### Metrics Not Appearing

1. Check that `INSTRUMENTATION_ENABLED=true` (default)
2. Check that `METRICS_ENABLED=true` (default)
3. Verify Prometheus is scraping the metrics endpoint (port 9090)
4. Check logs for instrumentation initialization errors
5. Ensure the metrics exporter is correctly configured

### High Cardinality

If you see high cardinality warnings:

1. Disable detailed labels: `METRICS_DETAILED_LABELS=false`
2. Review tool usage patterns
3. Consider using recording rules to pre-aggregate metrics
4. Adjust Prometheus storage settings

### Performance Impact

If instrumentation impacts performance:

1. Reduce trace sampling rate (`OTEL_TRACES_SAMPLER_ARG`)
2. Use `prometheus` exporter instead of `otlp` for lower overhead
3. Consider disabling tracing (`TRACING_EXPORTER=none`)
4. Increase scrape interval in Prometheus config
