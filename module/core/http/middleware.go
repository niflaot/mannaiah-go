package http

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"
)

const (
	// HeaderRayID defines the tracing header key injected in HTTP responses.
	HeaderRayID = "X-Ray-ID"
	// rayIDLocalsKey defines the request-local key storing the tracing id.
	rayIDLocalsKey = "ray_id"
)

var (
	// defaultErrorMessage defines the fallback translatable error message key.
	defaultErrorMessage = "internal_server_error"
)

// ErrorResponse defines the standard HTTP error payload.
type ErrorResponse struct {
	// Message is the translatable error message key.
	Message string `json:"message"`
	// Error is the concrete low-level error message.
	Error string `json:"error"`
}

// AppError defines standardized application errors mapped by the HTTP error handler.
type AppError struct {
	// Status defines the HTTP status code.
	Status int
	// Message defines the translatable error message key.
	Message string
	// Cause defines the wrapped low-level error.
	Cause error
}

// Error returns the low-level cause when available, otherwise the message key.
func (e *AppError) Error() string {
	if e == nil {
		return defaultErrorMessage
	}
	if e.Cause != nil {
		return e.Cause.Error()
	}
	if strings.TrimSpace(e.Message) != "" {
		return e.Message
	}

	return defaultErrorMessage
}

// Unwrap returns the wrapped low-level cause.
func (e *AppError) Unwrap() error {
	if e == nil {
		return nil
	}

	return e.Cause
}

// NewAppError creates an application error with HTTP status, translatable key, and optional cause.
func NewAppError(status int, message string, cause error) error {
	resolvedStatus := status
	if resolvedStatus < 400 || resolvedStatus > 599 {
		resolvedStatus = fiber.StatusInternalServerError
	}

	resolvedMessage := strings.TrimSpace(message)
	if resolvedMessage == "" {
		resolvedMessage = statusMessageKey(resolvedStatus)
	}

	return &AppError{
		Status:  resolvedStatus,
		Message: resolvedMessage,
		Cause:   cause,
	}
}

// rayIDMiddleware ensures every response has a tracing id header.
func rayIDMiddleware(ctx *fiber.Ctx) error {
	rayID := readOrCreateRayID(ctx)
	ctx.Locals(rayIDLocalsKey, rayID)
	ctx.Set(HeaderRayID, rayID)

	return ctx.Next()
}

// errorHandler maps all handler errors to a consistent JSON payload format.
func errorHandler(ctx *fiber.Ctx, err error) error {
	return errorHandlerWithLogger(nil)(ctx, err)
}

// errorHandlerWithLogger maps errors to JSON and emits structured 5xx logs when logger is provided.
func errorHandlerWithLogger(logger *zap.Logger) fiber.ErrorHandler {
	resolvedLogger := logger
	if resolvedLogger == nil {
		resolvedLogger = zap.NewNop()
	}

	return func(ctx *fiber.Ctx, err error) error {
		rayID := readOrCreateRayID(ctx)
		ctx.Locals(rayIDLocalsKey, rayID)
		ctx.Set(HeaderRayID, rayID)

		status := fiber.StatusInternalServerError
		message := defaultErrorMessage
		errorValue := defaultErrorMessage

		var appErr *AppError
		if errors.As(err, &appErr) && appErr != nil {
			status = appErr.Status
			message = appErr.Message
			errorValue = appErr.Error()
		} else {
			var fiberErr *fiber.Error
			if errors.As(err, &fiberErr) && fiberErr != nil {
				status = fiberErr.Code
				message = statusMessageKey(status)
				errorValue = strings.TrimSpace(fiberErr.Message)
			} else if err != nil {
				errorValue = strings.TrimSpace(err.Error())
			}
		}

		if strings.TrimSpace(message) == "" {
			message = statusMessageKey(status)
		}
		if strings.TrimSpace(errorValue) == "" {
			errorValue = message
		}

		payload := ErrorResponse{
			Message: message,
			Error:   errorValue,
		}

		if status >= fiber.StatusInternalServerError {
			resolvedLogger.Error("http request failed",
				zap.String("ray_id", rayID),
				zap.Int("status", status),
				zap.String("method", ctx.Method()),
				zap.String("url", ctx.OriginalURL()),
				zap.String("message", message),
				zap.String("error_chain", formatErrorChain(err)),
			)
		}

		return ctx.Status(status).JSON(payload)
	}
}

// formatErrorChain returns a flattened error chain for faster root-cause inspection in logs.
func formatErrorChain(err error) string {
	if err == nil {
		return defaultErrorMessage
	}

	parts := make([]string, 0, 4)
	for current := err; current != nil; current = errors.Unwrap(current) {
		text := strings.TrimSpace(current.Error())
		if text == "" {
			continue
		}
		parts = append(parts, text)
	}
	if len(parts) == 0 {
		return defaultErrorMessage
	}

	return strings.Join(parts, " -> ")
}

// readOrCreateRayID resolves tracing id from headers, locals, or generated value.
func readOrCreateRayID(ctx *fiber.Ctx) string {
	if ctx == nil {
		return newRayID()
	}

	if value := strings.TrimSpace(ctx.Get(HeaderRayID)); value != "" {
		return value
	}
	if local, ok := ctx.Locals(rayIDLocalsKey).(string); ok && strings.TrimSpace(local) != "" {
		return strings.TrimSpace(local)
	}

	return newRayID()
}

// newRayID generates a new tracing id.
func newRayID() string {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return fmt.Sprintf("fallback-%d", randByteFallback())
	}

	return hex.EncodeToString(bytes)
}

// randByteFallback provides a deterministic fallback when random generation fails.
func randByteFallback() int {
	bytes := make([]byte, 1)
	if _, err := rand.Read(bytes); err != nil {
		return http.StatusInternalServerError
	}

	return int(bytes[0])
}

// statusMessageKey converts HTTP status text into a translatable snake_case key.
func statusMessageKey(status int) string {
	text := strings.TrimSpace(strings.ToLower(http.StatusText(status)))
	if text == "" {
		return defaultErrorMessage
	}

	text = strings.ReplaceAll(text, "-", " ")
	parts := strings.Fields(text)
	if len(parts) == 0 {
		return defaultErrorMessage
	}

	return strings.Join(parts, "_")
}
