package server

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/teemow/inboxfewer/internal/instrumentation"
)

const (
	// DefaultMetricsAddr is the default address for the metrics server.
	DefaultMetricsAddr = ":9090"

	// DefaultMetricsReadTimeout is the default read timeout for the metrics server.
	DefaultMetricsReadTimeout = 10 * time.Second

	// DefaultMetricsWriteTimeout is the default write timeout for the metrics server.
	DefaultMetricsWriteTimeout = 10 * time.Second

	// DefaultMetricsIdleTimeout is the default idle timeout for the metrics server.
	DefaultMetricsIdleTimeout = 60 * time.Second

	// DefaultShutdownTimeout is the default timeout for graceful server shutdown.
	DefaultShutdownTimeout = 30 * time.Second
)

// MetricsServerConfig holds configuration for the metrics server.
type MetricsServerConfig struct {
	// Addr is the address to bind the metrics server to (e.g., ":9090").
	Addr string

	// Enabled determines whether the metrics server should be started.
	Enabled bool

	// InstrumentationProvider provides the Prometheus metrics handler.
	InstrumentationProvider *instrumentation.Provider
}

// MetricsServer serves Prometheus metrics on a dedicated port.
// This isolates metrics from the main application traffic for security,
// preventing unauthorized access to operational metrics.
type MetricsServer struct {
	httpServer *http.Server
	addr       string
	listener   net.Listener // Stored listener for actual bound address
}

// NewMetricsServer creates a new metrics server with the given configuration.
// The server exposes a /metrics endpoint for Prometheus scraping.
func NewMetricsServer(config MetricsServerConfig) (*MetricsServer, error) {
	if config.Addr == "" {
		config.Addr = DefaultMetricsAddr
	}

	if config.InstrumentationProvider == nil {
		return nil, fmt.Errorf("instrumentation provider is required for metrics server")
	}

	if !config.InstrumentationProvider.Enabled() {
		return nil, fmt.Errorf("instrumentation provider is not enabled")
	}

	return &MetricsServer{
		addr: config.Addr,
	}, nil
}

// Start starts the metrics server in a blocking manner.
// Call this in a goroutine if you need non-blocking operation.
func (s *MetricsServer) Start() error {
	return s.StartWithReadySignal(nil)
}

// StartWithReadySignal starts the metrics server and signals readiness via the channel.
// The ready channel is closed when the server is listening and ready to accept connections.
// If ready is nil, the server starts without signaling.
// Call this in a goroutine if you need non-blocking operation.
func (s *MetricsServer) StartWithReadySignal(ready chan<- struct{}) error {
	mux := http.NewServeMux()

	// Register /metrics endpoint using promhttp.Handler()
	// The OpenTelemetry prometheus exporter registers metrics to the global
	// Prometheus registry, which promhttp.Handler() exposes.
	mux.Handle("/metrics", promhttp.Handler())

	// Add a basic health check for the metrics server itself
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	s.httpServer = &http.Server{
		Addr:              s.addr,
		Handler:           mux,
		ReadHeaderTimeout: DefaultMetricsReadTimeout,
		WriteTimeout:      DefaultMetricsWriteTimeout,
		IdleTimeout:       DefaultMetricsIdleTimeout,
	}

	// Create listener first to detect bind errors before signaling ready
	ln, err := net.Listen("tcp", s.addr)
	if err != nil {
		return fmt.Errorf("failed to bind metrics server to %s: %w", s.addr, err)
	}
	s.listener = ln

	slog.Info("metrics server listening", "addr", ln.Addr().String())

	// Signal readiness if channel provided
	if ready != nil {
		close(ready)
	}

	return s.httpServer.Serve(ln)
}

// Shutdown gracefully shuts down the metrics server.
func (s *MetricsServer) Shutdown(ctx context.Context) error {
	if s.httpServer != nil {
		slog.Info("shutting down metrics server")
		return s.httpServer.Shutdown(ctx)
	}
	return nil
}

// Addr returns the address the metrics server is listening on.
// If the server was started with ":0", this returns the actual bound address.
// Otherwise, returns the configured address.
func (s *MetricsServer) Addr() string {
	if s.listener != nil {
		return s.listener.Addr().String()
	}
	return s.addr
}
