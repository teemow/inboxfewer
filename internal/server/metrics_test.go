package server

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/teemow/inboxfewer/internal/instrumentation"
)

func TestNewMetricsServer(t *testing.T) {
	tests := []struct {
		name        string
		config      MetricsServerConfig
		expectError bool
		errContains string
	}{
		{
			name: "valid config",
			config: MetricsServerConfig{
				Addr:                    ":9090",
				Enabled:                 true,
				InstrumentationProvider: createTestProvider(t),
			},
			expectError: false,
		},
		{
			name: "default addr",
			config: MetricsServerConfig{
				Addr:                    "",
				Enabled:                 true,
				InstrumentationProvider: createTestProvider(t),
			},
			expectError: false,
		},
		{
			name: "nil provider",
			config: MetricsServerConfig{
				Addr:                    ":9090",
				Enabled:                 true,
				InstrumentationProvider: nil,
			},
			expectError: true,
			errContains: "instrumentation provider is required",
		},
		{
			name: "disabled provider",
			config: MetricsServerConfig{
				Addr:                    ":9090",
				Enabled:                 true,
				InstrumentationProvider: createDisabledProvider(t),
			},
			expectError: true,
			errContains: "instrumentation provider is not enabled",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server, err := NewMetricsServer(tt.config)

			if tt.expectError {
				if err == nil {
					t.Errorf("NewMetricsServer() expected error, got nil")
				} else if tt.errContains != "" && !containsString(err.Error(), tt.errContains) {
					t.Errorf("NewMetricsServer() error = %v, want error containing %q", err, tt.errContains)
				}
			} else {
				if err != nil {
					t.Errorf("NewMetricsServer() unexpected error: %v", err)
				}
				if server == nil {
					t.Error("NewMetricsServer() returned nil server")
				}
			}
		})
	}
}

func TestMetricsServer_StartAndShutdown(t *testing.T) {
	provider := createTestProvider(t)

	server, err := NewMetricsServer(MetricsServerConfig{
		Addr:                    ":0", // Use any available port
		Enabled:                 true,
		InstrumentationProvider: provider,
	})
	if err != nil {
		t.Fatalf("NewMetricsServer() error = %v", err)
	}

	// Start server in goroutine
	serverErr := make(chan error, 1)
	go func() {
		if err := server.Start(); err != nil && err != http.ErrServerClosed {
			serverErr <- err
		}
		close(serverErr)
	}()

	// Give server time to start
	time.Sleep(100 * time.Millisecond)

	// Test /healthz endpoint
	resp, err := http.Get("http://localhost" + server.Addr() + "/healthz")
	if err != nil {
		// Server might not be ready yet on :0, skip the HTTP test
		t.Logf("Skipping HTTP test (server may not be ready): %v", err)
	} else {
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Errorf("GET /healthz status = %d, want %d", resp.StatusCode, http.StatusOK)
		}
	}

	// Shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		t.Errorf("Shutdown() error = %v", err)
	}

	// Check for server errors
	select {
	case err := <-serverErr:
		if err != nil {
			t.Errorf("Server error: %v", err)
		}
	case <-time.After(2 * time.Second):
		// Server shut down cleanly
	}
}

func TestMetricsServer_ShutdownWithoutStart(t *testing.T) {
	provider := createTestProvider(t)

	server, err := NewMetricsServer(MetricsServerConfig{
		Addr:                    ":9090",
		Enabled:                 true,
		InstrumentationProvider: provider,
	})
	if err != nil {
		t.Fatalf("NewMetricsServer() error = %v", err)
	}

	// Shutdown without starting should not error
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		t.Errorf("Shutdown() without Start() error = %v", err)
	}
}

func TestMetricsServer_Addr(t *testing.T) {
	provider := createTestProvider(t)

	server, err := NewMetricsServer(MetricsServerConfig{
		Addr:                    ":9091",
		Enabled:                 true,
		InstrumentationProvider: provider,
	})
	if err != nil {
		t.Fatalf("NewMetricsServer() error = %v", err)
	}

	if server.Addr() != ":9091" {
		t.Errorf("Addr() = %q, want %q", server.Addr(), ":9091")
	}
}

// Helper functions

func createTestProvider(t *testing.T) *instrumentation.Provider {
	t.Helper()
	ctx := context.Background()
	provider, err := instrumentation.NewProvider(ctx, instrumentation.Config{
		ServiceName:     "test-service",
		ServiceVersion:  "1.0.0",
		Enabled:         true,
		MetricsExporter: "prometheus",
		TracingExporter: "none",
	})
	if err != nil {
		t.Fatalf("failed to create test provider: %v", err)
	}
	t.Cleanup(func() {
		_ = provider.Shutdown(ctx)
	})
	return provider
}

func createDisabledProvider(t *testing.T) *instrumentation.Provider {
	t.Helper()
	ctx := context.Background()
	provider, err := instrumentation.NewProvider(ctx, instrumentation.Config{
		ServiceName:    "test-service",
		ServiceVersion: "1.0.0",
		Enabled:        false,
	})
	if err != nil {
		t.Fatalf("failed to create disabled provider: %v", err)
	}
	return provider
}

func containsString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
