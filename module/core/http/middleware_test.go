package http

import (
	"encoding/json"
	"errors"
	stdhttp "net/http"
	"testing"

	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"
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
