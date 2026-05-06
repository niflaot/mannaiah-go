package http

import (
	"context"
	"mime/multipart"

	"go.uber.org/zap"
)

// Handler defines an abstract HTTP route handler signature.
type Handler func(ctx Context) error

// Context defines an abstract HTTP request context contract.
type Context interface {
	// Context returns request-scoped context values.
	Context() context.Context
	// GetHeader reads request header values.
	GetHeader(key string, defaultValue ...string) string
	// Queries reads all query-string values.
	Queries() map[string]string
	// Status sets the response status code.
	Status(code int) Context
	// JSON writes a JSON response payload.
	JSON(body any) error
	// SendString writes a plain-text response payload.
	SendString(body string) error
	// SendStatus writes a response with status code only.
	SendStatus(status int) error
	// SetHeader writes response header values.
	SetHeader(key string, value string)
	// SendBytes writes binary response payload values.
	SendBytes(body []byte) error
	// Params reads path parameter values.
	Params(key string, defaultValue ...string) string
	// Query reads query string values.
	Query(key string, defaultValue ...string) string
	// BodyParser decodes request body into output.
	BodyParser(out any) error
	// Body returns raw request body bytes.
	Body() []byte
	// FormFile reads multipart file payload values.
	FormFile(key string) (*multipart.FileHeader, error)
	// FormValue reads multipart/form values.
	FormValue(key string, defaultValue ...string) string
	// Locals reads or sets request-local values.
	Locals(key string, value ...any) any
}

// Router defines an abstract HTTP router contract for module route registration.
type Router interface {
	// Get registers a GET route handler.
	Get(path string, handler Handler)
	// Options registers an OPTIONS route handler.
	Options(path string, handler Handler)
	// Post registers a POST route handler.
	Post(path string, handler Handler)
	// Put registers a PUT route handler.
	Put(path string, handler Handler)
	// Patch registers a PATCH route handler.
	Patch(path string, handler Handler)
	// Delete registers a DELETE route handler.
	Delete(path string, handler Handler)
}

// Engine defines the abstract HTTP server contract exposed by this package.
type Engine interface {
	// Address returns the resolved server bind address.
	Address() string
	// RegisterRoutes registers app-level routes using abstract router interfaces.
	RegisterRoutes(register func(router Router))
	// MountRoutes mounts grouped routes under a prefix using abstract router interfaces.
	MountRoutes(prefix string, register func(router Router))
	// Start begins listening on the resolved address.
	Start() error
	// Shutdown gracefully stops the server using the provided context.
	Shutdown(ctx context.Context) error
	// Logger returns the resolved server logger instance.
	Logger() *zap.Logger
}
