package logging

import (
	"log/slog"
	"testing"
)

func TestNewSlogAdapter_WithNil(t *testing.T) {
	adapter := NewSlogAdapter(nil)
	if adapter == nil {
		t.Fatal("NewSlogAdapter returned nil")
	}
	if adapter.logger == nil {
		t.Error("adapter.logger should not be nil when created with nil")
	}
}

func TestNewSlogAdapter_WithLogger(t *testing.T) {
	logger := slog.Default()
	adapter := NewSlogAdapter(logger)
	if adapter == nil {
		t.Fatal("NewSlogAdapter returned nil")
	}
	if adapter.logger != logger {
		t.Error("adapter.logger should be the provided logger")
	}
}

func TestSlogAdapter_Debug(t *testing.T) {
	adapter := NewSlogAdapter(slog.Default())
	// Should not panic
	adapter.Debug("test message", "key", "value")
}

func TestSlogAdapter_Info(t *testing.T) {
	adapter := NewSlogAdapter(slog.Default())
	// Should not panic
	adapter.Info("test message", "key", "value")
}

func TestSlogAdapter_Warn(t *testing.T) {
	adapter := NewSlogAdapter(slog.Default())
	// Should not panic
	adapter.Warn("test message", "key", "value")
}

func TestSlogAdapter_Error(t *testing.T) {
	adapter := NewSlogAdapter(slog.Default())
	// Should not panic
	adapter.Error("test message", "key", "value")
}

func TestSlogAdapter_Logger(t *testing.T) {
	logger := slog.Default()
	adapter := NewSlogAdapter(logger)
	if adapter.Logger() != logger {
		t.Error("Logger() should return the underlying logger")
	}
}

func TestDefaultLogger(t *testing.T) {
	adapter := DefaultLogger()
	if adapter == nil {
		t.Fatal("DefaultLogger returned nil")
	}
	if adapter.logger == nil {
		t.Error("DefaultLogger().logger should not be nil")
	}
}

func TestLoggerInterface(t *testing.T) {
	// Verify SlogAdapter implements Logger interface
	var _ Logger = (*SlogAdapter)(nil)
}
