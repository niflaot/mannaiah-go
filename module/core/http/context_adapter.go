package http

import (
	"context"
	"mime/multipart"
	"strings"

	"github.com/gofiber/fiber/v2"
)

// fiberContextAdapter adapts Fiber contexts to abstract request context contracts.
type fiberContextAdapter struct {
	// ctx is the wrapped Fiber context.
	ctx *fiber.Ctx
}

// Context returns request-scoped context values.
func (a *fiberContextAdapter) Context() context.Context {
	return a.ctx.UserContext()
}

// GetHeader reads request header values.
func (a *fiberContextAdapter) GetHeader(key string, defaultValue ...string) string {
	return a.ctx.Get(key, defaultValue...)
}

// Status sets the response status code.
func (a *fiberContextAdapter) Status(code int) Context {
	a.ctx.Status(code)
	return a
}

// JSON writes a JSON response payload.
func (a *fiberContextAdapter) JSON(body any) error {
	return a.ctx.JSON(body)
}

// SendString writes a plain-text response payload.
func (a *fiberContextAdapter) SendString(body string) error {
	return a.ctx.SendString(body)
}

// SendStatus writes a response with status code only.
func (a *fiberContextAdapter) SendStatus(status int) error {
	return a.ctx.SendStatus(status)
}

// Params reads path parameter values.
func (a *fiberContextAdapter) Params(key string, defaultValue ...string) string {
	return a.ctx.Params(key, defaultValue...)
}

// Query reads query string values.
func (a *fiberContextAdapter) Query(key string, defaultValue ...string) string {
	return a.ctx.Query(key, defaultValue...)
}

// BodyParser decodes request body into output.
func (a *fiberContextAdapter) BodyParser(out any) error {
	return a.ctx.BodyParser(out)
}

// FormFile reads multipart file payload values.
func (a *fiberContextAdapter) FormFile(key string) (*multipart.FileHeader, error) {
	return a.ctx.FormFile(key)
}

// FormValue reads multipart/form values.
func (a *fiberContextAdapter) FormValue(key string, defaultValue ...string) string {
	if len(defaultValue) == 0 {
		return a.ctx.FormValue(key)
	}

	value := a.ctx.FormValue(key)
	if strings.TrimSpace(value) == "" {
		return defaultValue[0]
	}

	return value
}

// Locals reads or sets request-local values.
func (a *fiberContextAdapter) Locals(key string, value ...any) any {
	if len(value) > 0 {
		return a.ctx.Locals(key, value[0])
	}

	return a.ctx.Locals(key)
}
