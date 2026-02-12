package http

import (
	"strings"
	"testing"

	"go.uber.org/zap"
	coreconfig "mannaiah/module/core/config"
)

// TestNewWithCoreUsesFallbackHostPort verifies HTTP address fallback from core config.
func TestNewWithCoreUsesFallbackHostPort(t *testing.T) {
	coreCfg := coreconfig.Core{
		Host: "127.0.0.1",
		Port: 9099,
	}

	server, err := NewWithCore(Config{}, &coreCfg, zap.NewNop())
	if err != nil {
		t.Fatalf("NewWithCore() error = %v", err)
	}

	if server.Address() != "127.0.0.1:9099" {
		t.Fatalf("Address() = %q, want %q", server.Address(), "127.0.0.1:9099")
	}
}

// TestNewWithCoreUsesCoreHostAndPortWhenHTTPDiffers verifies core host and port are authoritative when core config is provided.
func TestNewWithCoreUsesCoreHostAndPortWhenHTTPDiffers(t *testing.T) {
	coreCfg := coreconfig.Core{
		Host: "127.0.0.1",
		Port: 9099,
	}

	server, err := NewWithCore(Config{
		Host: "10.10.10.10",
		Port: 7070,
	}, &coreCfg, zap.NewNop())
	if err != nil {
		t.Fatalf("NewWithCore() error = %v", err)
	}

	if server.Address() != "127.0.0.1:9099" {
		t.Fatalf("Address() = %q, want %q", server.Address(), "127.0.0.1:9099")
	}
}

// TestNewUsesHTTPConfigOverrides verifies explicit HTTP host and port override core fallback.
func TestNewUsesHTTPConfigOverrides(t *testing.T) {
	server, err := New(
		Config{
			Host: "0.0.0.0",
			Port: 7070,
		},
		zap.NewNop(),
	)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	if server.Address() != "0.0.0.0:7070" {
		t.Fatalf("Address() = %q, want %q", server.Address(), "0.0.0.0:7070")
	}
}

// TestAddressFromInvalidPort verifies invalid port values return wrapped errors.
func TestAddressFromInvalidPort(t *testing.T) {
	_, err := AddressFrom(
		Config{
			Host: "127.0.0.1",
			Port: 70000,
		},
		nil,
	)
	if err == nil {
		t.Fatalf("expected AddressFrom() error for invalid port")
	}
	if !strings.Contains(err.Error(), "resolve http address") {
		t.Fatalf("expected wrapped address error, got %q", err.Error())
	}
}

// TestAddressFromSuccess verifies successful address construction from provided HTTP config values.
func TestAddressFromSuccess(t *testing.T) {
	address, err := AddressFrom(
		Config{
			Host: "localhost",
			Port: 8088,
		},
		nil,
	)
	if err != nil {
		t.Fatalf("AddressFrom() error = %v", err)
	}
	if address != "localhost:8088" {
		t.Fatalf("AddressFrom() = %q, want %q", address, "localhost:8088")
	}
}

// TestAddressFromUsesCoreHostAndPortWhenProvided verifies resolution prioritizes core host and core port in merged values.
func TestAddressFromUsesCoreHostAndPortWhenProvided(t *testing.T) {
	address, err := AddressFrom(
		Config{
			Host: "192.168.1.100",
			Port: 8088,
		},
		&coreconfig.Core{
			Host: "localhost",
			Port: 9999,
		},
	)
	if err != nil {
		t.Fatalf("AddressFrom() error = %v", err)
	}
	if address != "localhost:9999" {
		t.Fatalf("AddressFrom() = %q, want %q", address, "localhost:9999")
	}
}

// TestLoggerAccess verifies Logger returns the resolved logger instance.
func TestLoggerAccess(t *testing.T) {
	logger := zap.NewNop()
	server, err := New(Config{Host: "127.0.0.1", Port: 8085}, logger)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	if server.Logger() != logger {
		t.Fatalf("expected Logger() to return provided logger instance")
	}
}

// TestNewDisablesFiberStartupMessage verifies Fiber startup banner output is disabled by default.
func TestNewDisablesFiberStartupMessage(t *testing.T) {
	server, err := New(Config{Host: "127.0.0.1", Port: 8090}, zap.NewNop())
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	if !server.App().Config().DisableStartupMessage {
		t.Fatalf("expected DisableStartupMessage to be true")
	}
}

// TestNewWithCoreInvalidPort verifies constructor validation for out-of-range HTTP ports.
func TestNewWithCoreInvalidPort(t *testing.T) {
	_, err := NewWithCore(
		Config{
			Host: "127.0.0.1",
			Port: 70000,
		},
		nil,
		nil,
	)
	if err == nil {
		t.Fatalf("expected NewWithCore() error for invalid port")
	}
}

// TestNewWithCoreNilLoggerFallback verifies nil logger inputs are replaced with a non-nil fallback logger.
func TestNewWithCoreNilLoggerFallback(t *testing.T) {
	server, err := NewWithCore(
		Config{
			Host: "127.0.0.1",
			Port: 8086,
		},
		nil,
		nil,
	)
	if err != nil {
		t.Fatalf("NewWithCore() error = %v", err)
	}
	if server.Logger() == nil {
		t.Fatalf("expected resolved fallback logger instance")
	}
}
