package logger

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	wmlog "github.com/ThreeDotsLabs/watermill"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
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

// TestZapAdapterDowngradesNoSubscribersMessage verifies no-subscriber Watermill messages are logged at debug level.
func TestZapAdapterDowngradesNoSubscribersMessage(t *testing.T) {
	core, observed := observer.New(zapcore.DebugLevel)
	adapter := NewZapAdapter(zap.New(core))

	adapter.Info("No subscribers to send message", wmlog.LogFields{"topic": "events"})

	entries := observed.All()
	if len(entries) != 1 {
		t.Fatalf("len(entries) = %d, want %d", len(entries), 1)
	}
	if entries[0].Level != zapcore.DebugLevel {
		t.Fatalf("entry.Level = %v, want %v", entries[0].Level, zapcore.DebugLevel)
	}
}

// TestIsNoSubscribersMessage verifies no-subscriber message matching behavior.
func TestIsNoSubscribersMessage(t *testing.T) {
	if !isNoSubscribersMessage("No subscribers to send message") {
		t.Fatalf("expected exact message match")
	}
	if isNoSubscribersMessage("other") {
		t.Fatalf("unexpected match for unrelated message")
	}
}
