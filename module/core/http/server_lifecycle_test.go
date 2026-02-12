package http

import (
	"context"
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

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

// TestStartLogsStartupMessage verifies Start emits custom startup logs through zap.
func TestStartLogsStartupMessage(t *testing.T) {
	occupied, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("net.Listen() error = %v", err)
	}
	t.Cleanup(func() {
		_ = occupied.Close()
	})

	logCore, observed := observer.New(zapcore.InfoLevel)
	logger := zap.New(logCore)
	port := occupied.Addr().(*net.TCPAddr).Port
	server, newErr := New(Config{Host: "127.0.0.1", Port: port}, logger)
	if newErr != nil {
		t.Fatalf("New() error = %v", newErr)
	}

	startErr := server.Start()
	if startErr == nil {
		t.Fatalf("expected Start() listen error for occupied port")
	}

	entries := observed.FilterMessage("http server starting").All()
	if len(entries) != 1 {
		t.Fatalf("startup log count = %d, want 1", len(entries))
	}
	context := entries[0].ContextMap()
	if context["address"] != fmt.Sprintf("127.0.0.1:%d", port) {
		t.Fatalf("startup address = %v, want %q", context["address"], fmt.Sprintf("127.0.0.1:%d", port))
	}
	if context["app_name"] != "mannaiah-http" {
		t.Fatalf("startup app_name = %v, want %q", context["app_name"], "mannaiah-http")
	}
}

// TestStartWithListenerLogsStartupMessage verifies StartWithListener emits custom startup logs through zap.
func TestStartWithListenerLogsStartupMessage(t *testing.T) {
	logCore, observed := observer.New(zapcore.InfoLevel)
	logger := zap.New(logCore)
	server, err := New(Config{Host: "127.0.0.1", Port: 8091}, logger)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
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

	waitForCondition(t, 40, 25*time.Millisecond, func() bool {
		return observed.FilterMessage("http server starting").Len() > 0
	})

	if err := server.Shutdown(context.Background()); err != nil {
		t.Fatalf("Shutdown() error = %v", err)
	}
	select {
	case startErr := <-done:
		if startErr != nil {
			t.Fatalf("StartWithListener() error = %v", startErr)
		}
	case <-time.After(2 * time.Second):
		t.Fatalf("server did not stop after shutdown")
	}

	entries := observed.FilterMessage("http server starting").All()
	if len(entries) == 0 {
		t.Fatalf("expected startup log entry")
	}
	context := entries[0].ContextMap()
	if context["address"] != listener.Addr().String() {
		t.Fatalf("startup address = %v, want %q", context["address"], listener.Addr().String())
	}
}
