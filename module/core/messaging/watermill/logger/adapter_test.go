package logger

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	wmlog "github.com/ThreeDotsLabs/watermill"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// TestZapAdapterLogs verifies Watermill logger adapter writes logs through Zap.
func TestZapAdapterLogs(t *testing.T) {
	var output bytes.Buffer
	core := zapcore.NewCore(zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig()), zapcore.AddSync(&output), zapcore.DebugLevel)
	adapter := NewZapAdapter(zap.New(core))

	adapter.Info("info-msg", wmlog.LogFields{"component": "test"})
	adapter.Debug("debug-msg", wmlog.LogFields{"component": "test"})
	adapter.Trace("trace-msg", wmlog.LogFields{"component": "test"})
	adapter.Error("error-msg", errors.New("boom"), wmlog.LogFields{"component": "test"})

	payload := output.String()
	if !strings.Contains(payload, "info-msg") {
		t.Fatalf("expected info message in log payload, got %q", payload)
	}
	if !strings.Contains(payload, "error-msg") {
		t.Fatalf("expected error message in log payload, got %q", payload)
	}
	if !strings.Contains(payload, "component") {
		t.Fatalf("expected structured fields in log payload, got %q", payload)
	}
}

// TestZapAdapterWith verifies field propagation through With contexts.
func TestZapAdapterWith(t *testing.T) {
	var output bytes.Buffer
	core := zapcore.NewCore(zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig()), zapcore.AddSync(&output), zapcore.DebugLevel)
	adapter := NewZapAdapter(zap.New(core))

	child := adapter.With(wmlog.LogFields{"module": "messaging"})
	child.Info("child-log", nil)

	if !strings.Contains(output.String(), `"module":"messaging"`) {
		t.Fatalf("expected inherited field in child logs, got %q", output.String())
	}
}

// TestZapAdapterNilLoggerFallback verifies nil logger input uses no-op fallback.
func TestZapAdapterNilLoggerFallback(t *testing.T) {
	adapter := NewZapAdapter(nil)
	adapter.Info("noop", nil)
}
