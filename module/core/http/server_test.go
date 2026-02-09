package http

import (
	"bytes"
	"context"
	"fmt"
	"net"
	stdhttp "net/http"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
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

// TestRegisterAndMount verifies route registration for app-level and grouped handlers.
func TestRegisterAndMount(t *testing.T) {
	server, err := New(Config{Host: "127.0.0.1", Port: 8081}, zap.NewNop())
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	server.Register(func(app *fiber.App) {
		app.Get("/health", func(ctx *fiber.Ctx) error {
			return ctx.Status(fiber.StatusOK).SendString("ok")
		})
	})
	server.Mount("/v1", func(router fiber.Router) {
		router.Get("/status", func(ctx *fiber.Ctx) error {
			return ctx.Status(fiber.StatusOK).SendString("ready")
		})
	})

	healthReq, _ := stdhttp.NewRequest(stdhttp.MethodGet, "/health", nil)
	healthResp, testErr := server.App().Test(healthReq)
	if testErr != nil {
		t.Fatalf("App().Test(/health) error = %v", testErr)
	}
	if healthResp.StatusCode != stdhttp.StatusOK {
		t.Fatalf("/health status = %d, want %d", healthResp.StatusCode, stdhttp.StatusOK)
	}

	statusReq, _ := stdhttp.NewRequest(stdhttp.MethodGet, "/v1/status", nil)
	statusResp, statusErr := server.App().Test(statusReq)
	if statusErr != nil {
		t.Fatalf("App().Test(/v1/status) error = %v", statusErr)
	}
	if statusResp.StatusCode != stdhttp.StatusOK {
		t.Fatalf("/v1/status status = %d, want %d", statusResp.StatusCode, stdhttp.StatusOK)
	}
}

// TestRegisterAndMountNilCallbacks verifies nil callbacks are safely ignored.
func TestRegisterAndMountNilCallbacks(t *testing.T) {
	server, err := New(Config{Host: "127.0.0.1", Port: 8082}, zap.NewNop())
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	server.Register(nil)
	server.Mount("/v1", nil)
}

// TestZapFiberMiddlewareLogs verifies zapfiber request logs are emitted through provided logger.
func TestZapFiberMiddlewareLogs(t *testing.T) {
	var output bytes.Buffer
	core := zapcore.NewCore(zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig()), zapcore.AddSync(&output), zapcore.DebugLevel)
	logger := zap.New(core)

	server, err := New(Config{Host: "127.0.0.1", Port: 8083}, logger)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	server.Register(func(app *fiber.App) {
		app.Get("/logged", func(ctx *fiber.Ctx) error {
			return ctx.Status(fiber.StatusOK).JSON(fiber.Map{"status": "ok"})
		})
	})

	req, _ := stdhttp.NewRequest(stdhttp.MethodGet, "/logged", nil)
	resp, testErr := server.App().Test(req)
	if testErr != nil {
		t.Fatalf("App().Test() error = %v", testErr)
	}
	if resp.StatusCode != stdhttp.StatusOK {
		t.Fatalf("status = %d, want %d", resp.StatusCode, stdhttp.StatusOK)
	}

	payload := output.String()
	if !strings.Contains(payload, "/logged") {
		t.Fatalf("expected zapfiber log payload to include route, got %q", payload)
	}
}

// TestStartWithListenerAndShutdown verifies server start on custom listener and graceful shutdown.
func TestStartWithListenerAndShutdown(t *testing.T) {
	server, err := New(Config{Host: "127.0.0.1", Port: 8084}, zap.NewNop())
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	server.Register(func(app *fiber.App) {
		app.Get("/alive", func(ctx *fiber.Ctx) error {
			return ctx.SendStatus(fiber.StatusOK)
		})
	})

	listener, listenErr := net.Listen("tcp", "127.0.0.1:0")
	if listenErr != nil {
		t.Fatalf("net.Listen() error = %v", listenErr)
	}
	t.Cleanup(func() {
		_ = listener.Close()
	})

	done := make(chan error, 1)
	go func() {
		done <- server.StartWithListener(listener)
	}()

	baseURL := fmt.Sprintf("http://%s/alive", listener.Addr().String())
	waitForHTTPReady(t, baseURL, 40, 25*time.Millisecond)

	shutdownErr := server.Shutdown(context.Background())
	if shutdownErr != nil {
		t.Fatalf("Shutdown() error = %v", shutdownErr)
	}

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("StartWithListener() error = %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatalf("server did not stop after shutdown")
	}
}

// TestStartReturnsListenError verifies start returns address binding failures.
func TestStartReturnsListenError(t *testing.T) {
	occupied, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("net.Listen() error = %v", err)
	}
	t.Cleanup(func() {
		_ = occupied.Close()
	})

	port := occupied.Addr().(*net.TCPAddr).Port
	server, newErr := New(Config{Host: "127.0.0.1", Port: port}, zap.NewNop())
	if newErr != nil {
		t.Fatalf("New() error = %v", newErr)
	}

	startErr := server.Start()
	if startErr == nil {
		t.Fatalf("expected Start() listen error for occupied port")
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

// waitForHTTPReady retries a GET request until endpoint becomes available.
func waitForHTTPReady(t *testing.T, url string, attempts int, interval time.Duration) {
	t.Helper()

	client := &stdhttp.Client{Timeout: 250 * time.Millisecond}
	for index := 0; index < attempts; index++ {
		resp, err := client.Get(url)
		if err == nil {
			_ = resp.Body.Close()
			if resp.StatusCode == stdhttp.StatusOK {
				return
			}
		}

		time.Sleep(interval)
	}

	t.Fatalf("endpoint %s did not become ready after %d attempts", url, attempts)
}
