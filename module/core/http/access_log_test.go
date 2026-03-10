package http

import (
	"context"
	stdhttp "net/http"
	"testing"

	"github.com/gofiber/fiber/v2"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

// TestShouldSkipAccessLogCoreRoutes verifies access-log suppression for core infrastructure routes.
func TestShouldSkipAccessLogCoreRoutes(t *testing.T) {
	server, err := New(Config{Host: "127.0.0.1", Port: 8120}, nil)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	server.Register(func(app *fiber.App) {
		app.Get("/metrics", func(ctx *fiber.Ctx) error { return ctx.SendStatus(stdhttp.StatusOK) })
		app.Get("/status", func(ctx *fiber.Ctx) error { return ctx.SendStatus(stdhttp.StatusOK) })
		app.Get("/openapi.json", func(ctx *fiber.Ctx) error { return ctx.SendStatus(stdhttp.StatusOK) })
		app.Get("/docs", func(ctx *fiber.Ctx) error { return ctx.SendStatus(stdhttp.StatusOK) })
		app.Get("/orders", func(ctx *fiber.Ctx) error { return ctx.SendStatus(stdhttp.StatusOK) })
	})

	cases := []string{"/metrics", "/status", "/openapi.json", "/docs"}
	for _, path := range cases {
		request, _ := stdhttp.NewRequest(stdhttp.MethodGet, path, nil)
		response, testErr := server.App().Test(request)
		if testErr != nil {
			t.Fatalf("App().Test(%s) error = %v", path, testErr)
		}
		if response.StatusCode != stdhttp.StatusOK {
			t.Fatalf("%s status = %d, want %d", path, response.StatusCode, stdhttp.StatusOK)
		}
	}
}

// TestAccessLogIncludesRayID verifies access logs include ray_id correlation fields.
func TestAccessLogIncludesRayID(t *testing.T) {
	logCore, observed := observer.New(zapcore.InfoLevel)
	logger := zap.New(logCore)
	server, err := New(Config{Host: "127.0.0.1", Port: 8121}, logger)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	server.Register(func(app *fiber.App) {
		app.Get("/ok", func(ctx *fiber.Ctx) error { return ctx.SendStatus(stdhttp.StatusOK) })
	})

	request, _ := stdhttp.NewRequest(stdhttp.MethodGet, "/ok", nil)
	request.Header.Set(HeaderRayID, "external-ray-id")
	response, testErr := server.App().Test(request)
	if testErr != nil {
		t.Fatalf("App().Test() error = %v", testErr)
	}
	if response.StatusCode != stdhttp.StatusOK {
		t.Fatalf("status = %d, want %d", response.StatusCode, stdhttp.StatusOK)
	}

	entries := observed.FilterMessage("Success").All()
	if len(entries) == 0 {
		t.Fatalf("expected access log entry")
	}

	fields := entries[0].ContextMap()
	if fields["ray_id"] != "external-ray-id" {
		t.Fatalf("ray_id = %v, want %q", fields["ray_id"], "external-ray-id")
	}
	if fields["trace_id"] != "" {
		t.Fatalf("trace_id = %v, want empty value", fields["trace_id"])
	}
}

// TestAccessLogUsesOTelTraceID verifies access logs prefer active OTel trace identifiers.
func TestAccessLogUsesOTelTraceID(t *testing.T) {
	logCore, observed := observer.New(zapcore.InfoLevel)
	logger := zap.New(logCore)
	server, err := New(Config{Host: "127.0.0.1", Port: 8122}, logger)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	traceID, traceErr := trace.TraceIDFromHex("0123456789abcdef0123456789abcdef")
	if traceErr != nil {
		t.Fatalf("TraceIDFromHex() error = %v", traceErr)
	}
	spanID, spanErr := trace.SpanIDFromHex("0123456789abcdef")
	if spanErr != nil {
		t.Fatalf("SpanIDFromHex() error = %v", spanErr)
	}
	spanContext := trace.NewSpanContext(trace.SpanContextConfig{
		TraceID:    traceID,
		SpanID:     spanID,
		TraceFlags: trace.FlagsSampled,
		Remote:     true,
	})

	server.Register(func(app *fiber.App) {
		app.Use(func(ctx *fiber.Ctx) error {
			ctx.SetUserContext(trace.ContextWithSpanContext(context.Background(), spanContext))
			return ctx.Next()
		})
		app.Get("/trace", func(ctx *fiber.Ctx) error { return ctx.SendStatus(stdhttp.StatusOK) })
	})

	request, _ := stdhttp.NewRequest(stdhttp.MethodGet, "/trace", nil)
	request.Header.Set(HeaderRayID, "external-ray-id")
	response, testErr := server.App().Test(request)
	if testErr != nil {
		t.Fatalf("App().Test() error = %v", testErr)
	}
	if response.StatusCode != stdhttp.StatusOK {
		t.Fatalf("status = %d, want %d", response.StatusCode, stdhttp.StatusOK)
	}

	entries := observed.FilterMessage("Success").All()
	if len(entries) == 0 {
		t.Fatalf("expected access log entry")
	}

	fields := entries[0].ContextMap()
	if fields["ray_id"] != "external-ray-id" {
		t.Fatalf("ray_id = %v, want %q", fields["ray_id"], "external-ray-id")
	}
	if fields["trace_id"] != traceID.String() {
		t.Fatalf("trace_id = %v, want %q", fields["trace_id"], traceID.String())
	}
}
