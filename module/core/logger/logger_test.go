package logger

import (
	"bytes"
	"strings"
	"testing"

	"go.uber.org/zap"
)

// TestNewWithWritersJSONFormat verifies JSON encoder output includes structured fields.
func TestNewWithWritersJSONFormat(t *testing.T) {
	var out bytes.Buffer

	log, err := NewWithWriters(
		Settings{
			Format: "json",
			Level:  "info",
		},
		&out,
		&out,
	)
	if err != nil {
		t.Fatalf("NewWithWriters() error = %v", err)
	}

	log.Info("json-line", zap.String("kind", "test"))

	got := out.String()
	if !strings.Contains(got, `"msg":"json-line"`) {
		t.Fatalf("expected JSON message field in output, got %q", got)
	}
	if !strings.Contains(got, `"kind":"test"`) {
		t.Fatalf("expected JSON structured field in output, got %q", got)
	}
	if !strings.HasPrefix(strings.TrimSpace(got), "{") {
		t.Fatalf("expected JSON object output, got %q", got)
	}
}

// TestNewWithWritersPrettyFormat verifies pretty format uses console encoding.
func TestNewWithWritersPrettyFormat(t *testing.T) {
	var out bytes.Buffer

	log, err := NewWithWriters(
		Settings{
			Format: "pretty",
			Level:  "info",
		},
		&out,
		&out,
	)
	if err != nil {
		t.Fatalf("NewWithWriters() error = %v", err)
	}

	log.Info("pretty-line", zap.String("kind", "test"))

	got := out.String()
	if strings.HasPrefix(strings.TrimSpace(got), "{") {
		t.Fatalf("expected console output, got %q", got)
	}
	if !strings.Contains(got, "pretty-line") {
		t.Fatalf("expected message in output, got %q", got)
	}
}

// TestNewWithWritersLevelFiltering verifies log level thresholds filter lower-severity entries.
func TestNewWithWritersLevelFiltering(t *testing.T) {
	var out bytes.Buffer

	log, err := NewWithWriters(
		Settings{
			Format: "json",
			Level:  "error",
		},
		&out,
		&out,
	)
	if err != nil {
		t.Fatalf("NewWithWriters() error = %v", err)
	}

	log.Info("should-not-appear")
	log.Error("must-appear")

	got := out.String()
	if strings.Contains(got, "should-not-appear") {
		t.Fatalf("expected info logs filtered out at error level, got %q", got)
	}
	if !strings.Contains(got, "must-appear") {
		t.Fatalf("expected error log to be emitted, got %q", got)
	}
}

// TestResolveUsesProvidedLogger verifies Resolve prefers caller-provided logger instances.
func TestResolveUsesProvidedLogger(t *testing.T) {
	provided := zap.NewNop()

	got, err := Resolve(
		provided,
		Settings{
			Format: "json",
			Level:  "debug",
		},
	)
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}

	if got != provided {
		t.Fatalf("expected Resolve() to return provided logger instance")
	}
}

// TestNewWithWritersInvalidFormat verifies unsupported log formats return an error.
func TestNewWithWritersInvalidFormat(t *testing.T) {
	var out bytes.Buffer

	_, err := NewWithWriters(
		Settings{
			Format: "xml",
			Level:  "info",
		},
		&out,
		&out,
	)
	if err == nil {
		t.Fatalf("expected error for unsupported format")
	}
}

// TestNewWithWritersInvalidLevel verifies unsupported log levels return an error.
func TestNewWithWritersInvalidLevel(t *testing.T) {
	var out bytes.Buffer

	_, err := NewWithWriters(
		Settings{
			Format: "json",
			Level:  "silent",
		},
		&out,
		&out,
	)
	if err == nil {
		t.Fatalf("expected error for unsupported level")
	}
}
