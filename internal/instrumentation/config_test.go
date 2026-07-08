package instrumentation

import (
	"os"
	"strings"
	"testing"
)

// unsetenv clears an environment variable for the duration of the test.
// t.Setenv registers restoration of the original value on cleanup.
func unsetenv(t *testing.T, key string) {
	t.Helper()
	t.Setenv(key, "")
	if err := os.Unsetenv(key); err != nil {
		t.Fatalf("failed to unset %s: %v", key, err)
	}
}

func TestDefaultConfig(t *testing.T) {
	// Clear environment to get defaults
	unsetenv(t, "OTEL_SERVICE_NAME")
	unsetenv(t, "INSTRUMENTATION_ENABLED")
	unsetenv(t, "METRICS_EXPORTER")
	unsetenv(t, "TRACING_EXPORTER")

	config := DefaultConfig()

	if config.ServiceName != "inboxfewer" {
		t.Errorf("expected ServiceName 'inboxfewer', got %q", config.ServiceName)
	}

	if !config.Enabled {
		t.Error("expected Enabled to be true by default")
	}

	if config.MetricsExporter != ExporterPrometheus {
		t.Errorf("expected MetricsExporter 'prometheus', got %q", config.MetricsExporter)
	}

	if config.TracingExporter != ExporterNone {
		t.Errorf("expected TracingExporter 'none', got %q", config.TracingExporter)
	}

	if config.TraceSamplingRate != 0.1 {
		t.Errorf("expected TraceSamplingRate 0.1, got %f", config.TraceSamplingRate)
	}
}

func TestDefaultConfig_FromEnv(t *testing.T) {
	// Set environment variables (restored automatically on cleanup)
	t.Setenv("OTEL_SERVICE_NAME", "test-service")
	t.Setenv("INSTRUMENTATION_ENABLED", "false")
	t.Setenv("METRICS_EXPORTER", "stdout")
	t.Setenv("TRACING_EXPORTER", "stdout")
	t.Setenv("OTEL_TRACES_SAMPLER_ARG", "0.5")

	config := DefaultConfig()

	if config.ServiceName != "test-service" {
		t.Errorf("expected ServiceName 'test-service', got %q", config.ServiceName)
	}

	if config.Enabled {
		t.Error("expected Enabled to be false")
	}

	if config.MetricsExporter != "stdout" {
		t.Errorf("expected MetricsExporter 'stdout', got %q", config.MetricsExporter)
	}

	if config.TracingExporter != "stdout" {
		t.Errorf("expected TracingExporter 'stdout', got %q", config.TracingExporter)
	}

	if config.TraceSamplingRate != 0.5 {
		t.Errorf("expected TraceSamplingRate 0.5, got %f", config.TraceSamplingRate)
	}
}

func TestConfig_Validate(t *testing.T) {
	tests := []struct {
		name        string
		config      Config
		expectError bool
		errContains string
	}{
		{
			name: "valid config with prometheus",
			config: Config{
				ServiceName:     "test",
				Enabled:         true,
				MetricsExporter: ExporterPrometheus,
				TracingExporter: ExporterNone,
			},
			expectError: false,
		},
		{
			name: "valid config with otlp",
			config: Config{
				ServiceName:     "test",
				Enabled:         true,
				MetricsExporter: ExporterPrometheus,
				TracingExporter: ExporterOTLP,
				OTLPEndpoint:    "localhost:4318",
			},
			expectError: false,
		},
		{
			name: "invalid sampling rate negative",
			config: Config{
				TraceSamplingRate: -0.5,
			},
			expectError: true,
			errContains: "sampling rate",
		},
		{
			name: "invalid sampling rate above 1",
			config: Config{
				TraceSamplingRate: 1.5,
			},
			expectError: true,
			errContains: "sampling rate",
		},
		{
			name: "invalid metrics exporter",
			config: Config{
				MetricsExporter: "invalid",
			},
			expectError: true,
			errContains: "invalid metrics exporter",
		},
		{
			name: "invalid tracing exporter",
			config: Config{
				TracingExporter: "invalid",
			},
			expectError: true,
			errContains: "invalid tracing exporter",
		},
		{
			name: "otlp tracing without endpoint",
			config: Config{
				TracingExporter: ExporterOTLP,
				OTLPEndpoint:    "",
			},
			expectError: true,
			errContains: "OTLP endpoint is required",
		},
		{
			name: "otlp metrics without endpoint",
			config: Config{
				MetricsExporter: ExporterOTLP,
				OTLPEndpoint:    "",
			},
			expectError: true,
			errContains: "OTLP endpoint is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.expectError {
				if err == nil {
					t.Error("expected error, got nil")
				} else if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("expected error containing %q, got %q", tt.errContains, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}

func TestGetEnvOrDefault(t *testing.T) {
	t.Setenv("TEST_VAR", "test_value")

	if v := getEnvOrDefault("TEST_VAR", "default"); v != "test_value" {
		t.Errorf("expected 'test_value', got %q", v)
	}

	if v := getEnvOrDefault("NONEXISTENT_VAR", "default"); v != "default" {
		t.Errorf("expected 'default', got %q", v)
	}
}

func TestGetEnvBoolOrDefault(t *testing.T) {
	t.Setenv("TEST_BOOL_TRUE", "true")
	t.Setenv("TEST_BOOL_FALSE", "false")
	t.Setenv("TEST_BOOL_INVALID", "not_a_bool")

	if v := getEnvBoolOrDefault("TEST_BOOL_TRUE", false); !v {
		t.Error("expected true")
	}

	if v := getEnvBoolOrDefault("TEST_BOOL_FALSE", true); v {
		t.Error("expected false")
	}

	if v := getEnvBoolOrDefault("TEST_BOOL_INVALID", true); !v {
		t.Error("expected default value true for invalid bool")
	}

	if v := getEnvBoolOrDefault("NONEXISTENT", true); !v {
		t.Error("expected default value true")
	}
}

func TestGetEnvFloatOrDefault(t *testing.T) {
	t.Setenv("TEST_FLOAT", "0.75")
	t.Setenv("TEST_FLOAT_INVALID", "not_a_float")

	if v := getEnvFloatOrDefault("TEST_FLOAT", 0.5); v != 0.75 {
		t.Errorf("expected 0.75, got %f", v)
	}

	if v := getEnvFloatOrDefault("TEST_FLOAT_INVALID", 0.5); v != 0.5 {
		t.Errorf("expected default 0.5 for invalid float, got %f", v)
	}

	if v := getEnvFloatOrDefault("NONEXISTENT", 0.5); v != 0.5 {
		t.Errorf("expected default 0.5, got %f", v)
	}
}
