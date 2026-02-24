package telemetry

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
	coredatabase "mannaiah/module/core/database"
	corehttp "mannaiah/module/core/http"
)

// TestInitAndMetricsHandler verifies telemetry initialization and Prometheus exposition.
func TestInitAndMetricsHandler(t *testing.T) {
	provider, err := Init(context.Background(), Config{
		Enabled:        true,
		MetricsEnabled: true,
		TracesEnabled:  false,
	}, nil)
	if err != nil {
		t.Fatalf("Init() error = %v", err)
	}
	defer func() {
		_ = provider.Shutdown(context.Background())
		SetActive(nil)
	}()

	RecordDependency("redis", "get", time.Now(), nil)

	request := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	recorder := httptest.NewRecorder()
	provider.MetricsHandler().ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("metrics status = %d, want %d", recorder.Code, http.StatusOK)
	}
	body := recorder.Body.String()
	if !strings.Contains(body, "mannaiah_dependency_requests_total") {
		t.Fatalf("expected dependency metrics in exposition output")
	}
}

// TestHTTPMiddlewareRecordsMetrics verifies HTTP middleware tracing and metrics behavior.
func TestHTTPMiddlewareRecordsMetrics(t *testing.T) {
	provider, err := Init(context.Background(), Config{
		Enabled:        true,
		MetricsEnabled: true,
		TracesEnabled:  true,
		TracesExporter: "none",
	}, nil)
	if err != nil {
		t.Fatalf("Init() error = %v", err)
	}
	defer func() {
		_ = provider.Shutdown(context.Background())
		SetActive(nil)
	}()

	app := fiber.New()
	app.Use(provider.HTTPMiddleware())

	var capturedTraceID string
	app.Get("/hello/:id", func(ctx *fiber.Ctx) error {
		spanContext := trace.SpanContextFromContext(ctx.UserContext())
		capturedTraceID = spanContext.TraceID().String()
		return ctx.SendStatus(http.StatusOK)
	})

	parentCtx, parentSpan := StartSpan(context.Background(), "test", "root")
	parentTraceparent := TraceparentFromContext(parentCtx)
	EndSpan(parentSpan, nil)
	parentSpanContext := trace.SpanContextFromContext(ContextWithTraceparent(context.Background(), parentTraceparent))

	request := httptest.NewRequest(http.MethodGet, "/hello/123", nil)
	request.Header.Set("traceparent", parentTraceparent)
	response, testErr := app.Test(request)
	if testErr != nil {
		t.Fatalf("app.Test() error = %v", testErr)
	}
	if response.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.StatusCode, http.StatusOK)
	}
	if capturedTraceID == "" {
		t.Fatalf("expected handler trace context")
	}
	if capturedTraceID != parentSpanContext.TraceID().String() {
		t.Fatalf("captured trace_id = %q, want parent trace_id %q", capturedTraceID, parentSpanContext.TraceID().String())
	}
	if response.Header.Get(corehttp.HeaderRayID) != capturedTraceID {
		t.Fatalf("%s = %q, want %q", corehttp.HeaderRayID, response.Header.Get(corehttp.HeaderRayID), capturedTraceID)
	}

	metricsRequest := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	metricsRecorder := httptest.NewRecorder()
	provider.MetricsHandler().ServeHTTP(metricsRecorder, metricsRequest)
	metricsBody := metricsRecorder.Body.String()
	if !strings.Contains(metricsBody, "mannaiah_http_server_requests_total") {
		t.Fatalf("expected http request metrics in exposition output")
	}
	if !strings.Contains(metricsBody, "route=\"/hello/:id\"") {
		t.Fatalf("expected route template labels in exposition output")
	}
}

// TestStartSQLStatsCollector verifies SQL stats collection metrics.
func TestStartSQLStatsCollector(t *testing.T) {
	provider, err := Init(context.Background(), Config{
		Enabled:           true,
		MetricsEnabled:    true,
		TracesEnabled:     false,
		DBStatsIntervalMS: 5,
	}, nil)
	if err != nil {
		t.Fatalf("Init() error = %v", err)
	}
	defer func() {
		_ = provider.Shutdown(context.Background())
		SetActive(nil)
	}()

	db, dbErr := coredatabase.Open(coredatabase.Config{
		Driver: "sqlite",
		DSN:    "file::memory:?cache=shared",
	}, nil)
	if dbErr != nil {
		t.Fatalf("coredatabase.Open() error = %v", dbErr)
	}
	sqlDB, sqlDBErr := db.DB()
	if sqlDBErr != nil {
		t.Fatalf("db.DB() error = %v", sqlDBErr)
	}
	defer func() {
		_ = sqlDB.Close()
	}()

	provider.StartSQLStatsCollector(sqlDB)
	time.Sleep(20 * time.Millisecond)

	request := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	recorder := httptest.NewRecorder()
	provider.MetricsHandler().ServeHTTP(recorder, request)

	bodyBytes, readErr := io.ReadAll(recorder.Body)
	if readErr != nil {
		t.Fatalf("ReadAll() error = %v", readErr)
	}
	body := string(bodyBytes)
	if !strings.Contains(body, "mannaiah_db_pool_open_connections") {
		t.Fatalf("expected DB pool metrics in exposition output")
	}
}

// TestOTelZapErrorHandlerDeduplicates verifies repeated OTel errors are deduplicated and logged through Zap.
func TestOTelZapErrorHandlerDeduplicates(t *testing.T) {
	core, observed := observer.New(zapcore.WarnLevel)
	logger := zap.New(core)
	handler := newOTelZapErrorHandler(logger)
	handler.minInterval = time.Hour

	handler.Handle(errors.New("traces export: resolver produced zero addresses"))
	handler.Handle(errors.New("traces export: resolver produced zero addresses"))
	handler.Handle(nil)

	entries := observed.FilterMessage("opentelemetry runtime error").All()
	if len(entries) != 1 {
		t.Fatalf("log entry count = %d, want %d", len(entries), 1)
	}

	fields := entries[0].ContextMap()
	errorField, ok := fields["error"].(string)
	if !ok || !strings.Contains(errorField, "resolver produced zero addresses") {
		t.Fatalf("unexpected error field = %v", fields["error"])
	}
}
