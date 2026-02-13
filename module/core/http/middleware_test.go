package http

import (
	"encoding/json"
	"errors"
	"fmt"
	stdhttp "net/http"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

// TestRayIDHeaderGeneratedOnSuccess verifies tracing header injection on successful responses.
func TestRayIDHeaderGeneratedOnSuccess(t *testing.T) {
	server, err := New(Config{Host: "127.0.0.1", Port: 8090}, zap.NewNop())
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	server.Register(func(app *fiber.App) {
		app.Get("/ok", func(ctx *fiber.Ctx) error {
			return ctx.JSON(fiber.Map{"status": "ok"})
		})
	})

	req, _ := stdhttp.NewRequest(stdhttp.MethodGet, "/ok", nil)
	resp, testErr := server.App().Test(req)
	if testErr != nil {
		t.Fatalf("App().Test() error = %v", testErr)
	}

	rayID := resp.Header.Get(HeaderRayID)
	if rayID == "" {
		t.Fatalf("expected %s header", HeaderRayID)
	}
}

// TestRayIDHeaderUsesIncomingHeader verifies inbound tracing ids are propagated to responses.
func TestRayIDHeaderUsesIncomingHeader(t *testing.T) {
	server, err := New(Config{Host: "127.0.0.1", Port: 8091}, zap.NewNop())
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	server.Register(func(app *fiber.App) {
		app.Get("/trace", func(ctx *fiber.Ctx) error {
			return ctx.SendStatus(stdhttp.StatusNoContent)
		})
	})

	req, _ := stdhttp.NewRequest(stdhttp.MethodGet, "/trace", nil)
	req.Header.Set(HeaderRayID, "external-trace")
	resp, testErr := server.App().Test(req)
	if testErr != nil {
		t.Fatalf("App().Test() error = %v", testErr)
	}

	if resp.Header.Get(HeaderRayID) != "external-trace" {
		t.Fatalf("%s = %q, want %q", HeaderRayID, resp.Header.Get(HeaderRayID), "external-trace")
	}
}

// TestErrorHandlerFormatsGenericError verifies generic errors use standard payload format.
func TestErrorHandlerFormatsGenericError(t *testing.T) {
	server, err := New(Config{Host: "127.0.0.1", Port: 8092}, zap.NewNop())
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	server.Register(func(app *fiber.App) {
		app.Get("/boom", func(ctx *fiber.Ctx) error {
			return errors.New("db exploded")
		})
	})

	req, _ := stdhttp.NewRequest(stdhttp.MethodGet, "/boom", nil)
	resp, testErr := server.App().Test(req)
	if testErr != nil {
		t.Fatalf("App().Test() error = %v", testErr)
	}

	if resp.StatusCode != stdhttp.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", resp.StatusCode, stdhttp.StatusInternalServerError)
	}

	payload := decodeErrorPayload(t, resp)
	if payload.Message != "internal_server_error" {
		t.Fatalf("message = %q, want %q", payload.Message, "internal_server_error")
	}
	if payload.Error != "db exploded" {
		t.Fatalf("error = %q, want %q", payload.Error, "db exploded")
	}
	if resp.Header.Get(HeaderRayID) == "" {
		t.Fatalf("expected %s header", HeaderRayID)
	}
}

// TestErrorHandlerLogsServerErrorCause verifies 5xx responses emit structured root-cause logs.
func TestErrorHandlerLogsServerErrorCause(t *testing.T) {
	logCore, observed := observer.New(zapcore.ErrorLevel)
	logger := zap.New(logCore)

	server, err := New(Config{Host: "127.0.0.1", Port: 8096}, logger)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	server.Register(func(app *fiber.App) {
		app.Get("/boom-log", func(ctx *fiber.Ctx) error {
			low := errors.New("sql duplicate key")
			return NewAppError(stdhttp.StatusInternalServerError, "internal_server_error", fmt.Errorf("create asset metadata: %w", low))
		})
	})

	req, _ := stdhttp.NewRequest(stdhttp.MethodGet, "/boom-log", nil)
	resp, testErr := server.App().Test(req)
	if testErr != nil {
		t.Fatalf("App().Test() error = %v", testErr)
	}
	if resp.StatusCode != stdhttp.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", resp.StatusCode, stdhttp.StatusInternalServerError)
	}

	entries := observed.FilterMessage("http request failed").All()
	if len(entries) == 0 {
		t.Fatalf("expected structured error log entry")
	}

	entry := entries[0]
	fields := entry.ContextMap()
	if fields["status"] != int64(stdhttp.StatusInternalServerError) {
		t.Fatalf("status field = %v, want %d", fields["status"], stdhttp.StatusInternalServerError)
	}
	if fields["url"] != "/boom-log" {
		t.Fatalf("url field = %v, want %q", fields["url"], "/boom-log")
	}
	if fields["ray_id"] == "" {
		t.Fatalf("expected ray_id field in error log")
	}

	errorChain, _ := fields["error_chain"].(string)
	if errorChain == "" || !containsAll(errorChain, []string{"create asset metadata", "sql duplicate key"}) {
		t.Fatalf("error_chain = %q, want wrapped cause details", errorChain)
	}
}

// TestErrorHandlerFormatsFiberError verifies Fiber errors are mapped with status-derived message keys.
func TestErrorHandlerFormatsFiberError(t *testing.T) {
	server, err := New(Config{Host: "127.0.0.1", Port: 8093}, zap.NewNop())
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	server.Register(func(app *fiber.App) {
		app.Get("/missing", func(ctx *fiber.Ctx) error {
			return fiber.NewError(stdhttp.StatusNotFound, "contact missing")
		})
	})

	req, _ := stdhttp.NewRequest(stdhttp.MethodGet, "/missing", nil)
	resp, testErr := server.App().Test(req)
	if testErr != nil {
		t.Fatalf("App().Test() error = %v", testErr)
	}

	if resp.StatusCode != stdhttp.StatusNotFound {
		t.Fatalf("status = %d, want %d", resp.StatusCode, stdhttp.StatusNotFound)
	}

	payload := decodeErrorPayload(t, resp)
	if payload.Message != "not_found" {
		t.Fatalf("message = %q, want %q", payload.Message, "not_found")
	}
	if payload.Error != "contact missing" {
		t.Fatalf("error = %q, want %q", payload.Error, "contact missing")
	}
}

// TestErrorHandlerFormatsAppError verifies custom application errors keep translatable message keys.
func TestErrorHandlerFormatsAppError(t *testing.T) {
	server, err := New(Config{Host: "127.0.0.1", Port: 8094}, zap.NewNop())
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	server.Register(func(app *fiber.App) {
		app.Get("/validation", func(ctx *fiber.Ctx) error {
			return NewAppError(stdhttp.StatusUnprocessableEntity, "validation_failed", errors.New("email invalid"))
		})
	})

	req, _ := stdhttp.NewRequest(stdhttp.MethodGet, "/validation", nil)
	resp, testErr := server.App().Test(req)
	if testErr != nil {
		t.Fatalf("App().Test() error = %v", testErr)
	}

	if resp.StatusCode != stdhttp.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want %d", resp.StatusCode, stdhttp.StatusUnprocessableEntity)
	}

	payload := decodeErrorPayload(t, resp)
	if payload.Message != "validation_failed" {
		t.Fatalf("message = %q, want %q", payload.Message, "validation_failed")
	}
	if payload.Error != "email invalid" {
		t.Fatalf("error = %q, want %q", payload.Error, "email invalid")
	}
}

// TestNewAppErrorFallbacks verifies constructor fallback behavior for invalid statuses and empty messages.
func TestNewAppErrorFallbacks(t *testing.T) {
	err := NewAppError(200, "", nil)
	appErr, ok := err.(*AppError)
	if !ok {
		t.Fatalf("expected *AppError, got %T", err)
	}
	if appErr.Status != stdhttp.StatusInternalServerError {
		t.Fatalf("Status = %d, want %d", appErr.Status, stdhttp.StatusInternalServerError)
	}
	if appErr.Message != "internal_server_error" {
		t.Fatalf("Message = %q, want %q", appErr.Message, "internal_server_error")
	}
}

// TestStatusMessageKey verifies status text normalization into translatable keys.
func TestStatusMessageKey(t *testing.T) {
	if statusMessageKey(stdhttp.StatusMethodNotAllowed) != "method_not_allowed" {
		t.Fatalf("unexpected method_not_allowed message key")
	}
	if statusMessageKey(999) != "internal_server_error" {
		t.Fatalf("unexpected fallback message key for unknown status")
	}
}

// TestFormatErrorChain verifies error-chain formatting behavior.
func TestFormatErrorChain(t *testing.T) {
	if value := formatErrorChain(nil); value != "internal_server_error" {
		t.Fatalf("formatErrorChain(nil) = %q, want %q", value, "internal_server_error")
	}

	base := errors.New("low")
	wrapped := fmt.Errorf("mid: %w", base)
	chain := formatErrorChain(wrapped)
	if !containsAll(chain, []string{"mid: low", "low"}) {
		t.Fatalf("formatErrorChain(wrapped) = %q, want full chain", chain)
	}
}

// decodeErrorPayload decodes a standard error response payload.
func decodeErrorPayload(t *testing.T, resp *stdhttp.Response) ErrorResponse {
	t.Helper()

	defer func() {
		_ = resp.Body.Close()
	}()

	var payload ErrorResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		t.Fatalf("Decode() error = %v", err)
	}

	return payload
}

// containsAll verifies all tokens exist within a target string.
func containsAll(target string, tokens []string) bool {
	for _, token := range tokens {
		if !strings.Contains(target, token) {
			return false
		}
	}

	return true
}
