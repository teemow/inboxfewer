package server

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"
	"go.opentelemetry.io/otel/metric/noop"

	"github.com/teemow/inboxfewer/internal/instrumentation"
)

func TestResponseWriterStreaming(t *testing.T) {
	t.Run("forwards Flush and commits the header", func(t *testing.T) {
		recorder := httptest.NewRecorder()
		rw := newResponseWriter(recorder)

		flusher, ok := any(rw).(http.Flusher)
		if !ok {
			t.Fatal("responseWriter does not implement http.Flusher")
		}
		flusher.Flush()

		if !recorder.Flushed {
			t.Error("Flush was not forwarded to the underlying writer")
		}
		if !rw.headerWritten || rw.statusCode != http.StatusOK {
			t.Errorf("headerWritten = %v, statusCode = %d, want true, %d", rw.headerWritten, rw.statusCode, http.StatusOK)
		}
	})

	t.Run("Unwrap returns the underlying writer", func(t *testing.T) {
		recorder := httptest.NewRecorder()
		rw := newResponseWriter(recorder)

		if rw.Unwrap() != recorder {
			t.Error("Unwrap did not return the underlying writer")
		}
	})
}

// TestMCPListenStreamThroughMiddleware sends a subscriptions/listen request
// (MCP 2026-07-28), as muster does on every connection, through the
// wrappers that sit in front of mcp-go on /mcp. mcp-go only streams when its
// writer implements http.Flusher; a wrapper that hides it turns every listen
// into HTTP 500 "Streaming unsupported" (teemow/inboxfewer#401).
func TestMCPListenStreamThroughMiddleware(t *testing.T) {
	metrics, err := instrumentation.NewMetrics(noop.NewMeterProvider().Meter("test"), false)
	if err != nil {
		t.Fatalf("NewMetrics: %v", err)
	}
	s := &OAuthHTTPServer{metrics: metrics}

	mcpSrv := mcpserver.NewMCPServer("inboxfewer", "test", mcpserver.WithToolCapabilities(true))
	handler := s.instrumentationMiddleware(s.oauthInstrumentationWrapper(mcpserver.NewStreamableHTTPServer(mcpSrv)))
	httpServer := httptest.NewServer(handler)
	t.Cleanup(httpServer.Close)

	body, err := json.Marshal(map[string]any{
		"jsonrpc": "2.0",
		"id":      2,
		"method":  mcp.MethodSubscriptionsListen,
		"params": map[string]any{
			"notifications": map[string]any{"toolsListChanged": true},
			"_meta": map[string]any{
				mcp.MetaKeyProtocolVersion:    mcp.ProtocolVersion20260728,
				mcp.MetaKeyClientInfo:         map[string]any{"name": "test-client", "version": "1.0.0"},
				mcp.MetaKeyClientCapabilities: map[string]any{},
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, httpServer.URL, bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")
	req.Header.Set(mcp.HeaderProtocolVersion, mcp.ProtocolVersion20260728)
	req.Header.Set(mcp.HeaderMethod, string(mcp.MethodSubscriptionsListen))

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("subscriptions/listen was not answered: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		var errBody bytes.Buffer
		_, _ = errBody.ReadFrom(resp.Body)
		t.Fatalf("status = %d, want %d; body: %s", resp.StatusCode, http.StatusOK, strings.TrimSpace(errBody.String()))
	}
	if ct := resp.Header.Get("Content-Type"); !strings.HasPrefix(ct, "text/event-stream") {
		t.Fatalf("Content-Type = %q, want text/event-stream", ct)
	}

	// The stream stays open; its first event must be the acknowledgement,
	// delivered while the request is still running.
	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		if data, ok := strings.CutPrefix(scanner.Text(), "data: "); ok {
			if !strings.Contains(data, mcp.MethodNotificationSubscriptionsAcknowledged) {
				t.Fatalf("first event = %s, want %s", data, mcp.MethodNotificationSubscriptionsAcknowledged)
			}
			return
		}
	}
	t.Fatalf("stream ended without an acknowledgement: %v", scanner.Err())
}
