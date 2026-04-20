package http

import (
	stdhttp "net/http"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2"
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

// TestCORSHeadersInjectedWhenOriginsConfigured verifies that CORS response headers are added when allowed origins are set.
func TestCORSHeadersInjectedWhenOriginsConfigured(t *testing.T) {
	server, err := New(Config{
		Host:               "127.0.0.1",
		Port:               8092,
		CORSAllowedOrigins: "https://app.example.com",
	}, zap.NewNop())
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	server.Register(func(app *fiber.App) {
		app.Get("/ping", func(ctx *fiber.Ctx) error { return ctx.SendStatus(200) })
	})

	req, _ := stdhttp.NewRequest(stdhttp.MethodGet, "/ping", nil)
	req.Header.Set("Origin", "https://app.example.com")
	resp, testErr := server.App().Test(req)
	if testErr != nil {
		t.Fatalf("App().Test() error = %v", testErr)
	}

	if got := resp.Header.Get("Access-Control-Allow-Origin"); got != "https://app.example.com" {
		t.Fatalf("Access-Control-Allow-Origin = %q, want %q", got, "https://app.example.com")
	}
}

// TestCORSHeadersAbsentWhenOriginsNotConfigured verifies no CORS headers are emitted when origins are empty.
func TestCORSHeadersAbsentWhenOriginsNotConfigured(t *testing.T) {
	server, err := New(Config{Host: "127.0.0.1", Port: 8093}, zap.NewNop())
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	server.Register(func(app *fiber.App) {
		app.Get("/ping", func(ctx *fiber.Ctx) error { return ctx.SendStatus(200) })
	})

	req, _ := stdhttp.NewRequest(stdhttp.MethodGet, "/ping", nil)
	req.Header.Set("Origin", "https://attacker.example.com")
	resp, testErr := server.App().Test(req)
	if testErr != nil {
		t.Fatalf("App().Test() error = %v", testErr)
	}

	if got := resp.Header.Get("Access-Control-Allow-Origin"); got != "" {
		t.Fatalf("Access-Control-Allow-Origin = %q, want empty when CORS is disabled", got)
	}
}

// TestRateLimiterRejects429WhenMaxExceeded verifies the rate limiter returns 429 after the request limit is reached.
func TestRateLimiterRejects429WhenMaxExceeded(t *testing.T) {
	server, err := New(Config{
		Host:              "127.0.0.1",
		Port:              8094,
		RateLimitMax:      1,
		RateLimitWindowMS: 60000,
	}, zap.NewNop())
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	server.Register(func(app *fiber.App) {
		app.Get("/ping", func(ctx *fiber.Ctx) error { return ctx.SendStatus(200) })
	})

	first, _ := stdhttp.NewRequest(stdhttp.MethodGet, "/ping", nil)
	resp1, err1 := server.App().Test(first)
	if err1 != nil {
		t.Fatalf("first request error = %v", err1)
	}
	if resp1.StatusCode != 200 {
		t.Fatalf("first request status = %d, want 200", resp1.StatusCode)
	}

	second, _ := stdhttp.NewRequest(stdhttp.MethodGet, "/ping", nil)
	resp2, err2 := server.App().Test(second)
	if err2 != nil {
		t.Fatalf("second request error = %v", err2)
	}
	if resp2.StatusCode != stdhttp.StatusTooManyRequests {
		t.Fatalf("second request status = %d, want 429", resp2.StatusCode)
	}
}

// TestRateLimiterDisabledWhenMaxIsZero verifies that rate limiting is not applied when RateLimitMax is zero.
func TestRateLimiterDisabledWhenMaxIsZero(t *testing.T) {
	server, err := New(Config{Host: "127.0.0.1", Port: 8095, RateLimitMax: 0}, zap.NewNop())
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	server.Register(func(app *fiber.App) {
		app.Get("/ping", func(ctx *fiber.Ctx) error { return ctx.SendStatus(200) })
	})

	for i := range 5 {
		req, _ := stdhttp.NewRequest(stdhttp.MethodGet, "/ping", nil)
		resp, testErr := server.App().Test(req)
		if testErr != nil {
			t.Fatalf("request %d error = %v", i+1, testErr)
		}
		if resp.StatusCode != 200 {
			t.Fatalf("request %d status = %d, want 200 (rate limiting should be disabled)", i+1, resp.StatusCode)
		}
	}
}
