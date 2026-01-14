package instrumentation

import (
	"os"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	// Clear environment to get defaults
	os.Unsetenv("OTEL_SERVICE_NAME")
	os.Unsetenv("INSTRUMENTATION_ENABLED")
	os.Unsetenv("METRICS_EXPORTER")
	os.Unsetenv("TRACING_EXPORTER")

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
	// Set environment variables
	os.Setenv("OTEL_SERVICE_NAME", "test-service")
	os.Setenv("INSTRUMENTATION_ENABLED", "false")
	os.Setenv("METRICS_EXPORTER", "stdout")
	os.Setenv("TRACING_EXPORTER", "stdout")
	os.Setenv("OTEL_TRACES_SAMPLER_ARG", "0.5")

	defer func() {
		os.Unsetenv("OTEL_SERVICE_NAME")
		os.Unsetenv("INSTRUMENTATION_ENABLED")
		os.Unsetenv("METRICS_EXPORTER")
		os.Unsetenv("TRACING_EXPORTER")
		os.Unsetenv("OTEL_TRACES_SAMPLER_ARG")
	}()

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
				} else if tt.errContains != "" && !containsString(err.Error(), tt.errContains) {
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

func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsSubstring(s, substr))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestGetEnvOrDefault(t *testing.T) {
	os.Setenv("TEST_VAR", "test_value")
	defer os.Unsetenv("TEST_VAR")

	if v := getEnvOrDefault("TEST_VAR", "default"); v != "test_value" {
		t.Errorf("expected 'test_value', got %q", v)
	}

	if v := getEnvOrDefault("NONEXISTENT_VAR", "default"); v != "default" {
		t.Errorf("expected 'default', got %q", v)
	}
}

func TestGetEnvBoolOrDefault(t *testing.T) {
	os.Setenv("TEST_BOOL_TRUE", "true")
	os.Setenv("TEST_BOOL_FALSE", "false")
	os.Setenv("TEST_BOOL_INVALID", "not_a_bool")
	defer func() {
		os.Unsetenv("TEST_BOOL_TRUE")
		os.Unsetenv("TEST_BOOL_FALSE")
		os.Unsetenv("TEST_BOOL_INVALID")
	}()

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
	os.Setenv("TEST_FLOAT", "0.75")
	os.Setenv("TEST_FLOAT_INVALID", "not_a_float")
	defer func() {
		os.Unsetenv("TEST_FLOAT")
		os.Unsetenv("TEST_FLOAT_INVALID")
	}()

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
