package http

import (
	"context"
	"errors"
	"fmt"
	"mime/multipart"
	"net"
	"strconv"
	"strings"
	"time"

	fiberzap "github.com/gofiber/contrib/fiberzap/v2"
	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"
	coreconfig "mannaiah/module/core/config"
)

var (
	// ErrInvalidPort is returned when the configured port is out of valid range.
	ErrInvalidPort = errors.New("http port must be between 1 and 65535")
)

// Handler defines an abstract HTTP route handler signature.
type Handler func(ctx Context) error

// Context defines an abstract HTTP request context contract.
type Context interface {
	// Context returns request-scoped context values.
	Context() context.Context
	// GetHeader reads request header values.
	GetHeader(key string, defaultValue ...string) string
	// Status sets the response status code.
	Status(code int) Context
	// JSON writes a JSON response payload.
	JSON(body any) error
	// SendString writes a plain-text response payload.
	SendString(body string) error
	// SendStatus writes a response with status code only.
	SendStatus(status int) error
	// Params reads path parameter values.
	Params(key string, defaultValue ...string) string
	// Query reads query string values.
	Query(key string, defaultValue ...string) string
	// BodyParser decodes request body into output.
	BodyParser(out any) error
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

var (
	// _ ensures Server satisfies the abstract Engine contract.
	_ Engine = (*Server)(nil)
)

// Server defines the Fiber HTTP server wrapper used by modules.
type Server struct {
	// app is the underlying Fiber application.
	app *fiber.App
	// address is the resolved bind address in host:port format.
	address string
	// logger is the request logger used by zapfiber middleware.
	logger *zap.Logger
}

// New creates a server using HTTP config values only.
func New(cfg Config, providedLogger *zap.Logger) (*Server, error) {
	return NewWithCore(cfg, nil, providedLogger)
}

// NewWithCore creates a server using HTTP config and optional core config fallbacks.
func NewWithCore(cfg Config, coreCfg *coreconfig.Core, providedLogger *zap.Logger) (*Server, error) {
	resolvedCfg := mergeConfig(cfg, coreCfg)
	address, err := buildAddress(resolvedCfg.Host, resolvedCfg.Port)
	if err != nil {
		return nil, err
	}

	logger := resolveLogger(providedLogger)
	app := fiber.New(fiber.Config{
		AppName:               resolvedCfg.AppName,
		Prefork:               resolvedCfg.Prefork,
		ServerHeader:          resolvedCfg.ServerHeader,
		ReadTimeout:           time.Duration(resolvedCfg.ReadTimeoutMS) * time.Millisecond,
		WriteTimeout:          time.Duration(resolvedCfg.WriteTimeoutMS) * time.Millisecond,
		IdleTimeout:           time.Duration(resolvedCfg.IdleTimeoutMS) * time.Millisecond,
		DisableStartupMessage: true,
		ErrorHandler:          errorHandler,
	})

	app.Use(rayIDMiddleware)
	app.Use(fiberzap.New(fiberzap.Config{
		Logger: logger,
	}))

	return &Server{
		app:     app,
		address: address,
		logger:  logger,
	}, nil
}

// App returns the underlying Fiber app for route and middleware composition.
func (s *Server) App() *fiber.App {
	return s.app
}

// Address returns the resolved server bind address.
func (s *Server) Address() string {
	return s.address
}

// Register allows future modules to register app-level routes and middleware using Fiber types.
func (s *Server) Register(register func(app *fiber.App)) {
	if register == nil {
		return
	}

	register(s.app)
}

// Mount allows future modules to mount grouped routes under a prefix using Fiber types.
func (s *Server) Mount(prefix string, register func(router fiber.Router)) {
	if register == nil {
		return
	}

	register(s.app.Group(prefix))
}

// RegisterRoutes allows future modules to register app-level routes using abstract router interfaces.
func (s *Server) RegisterRoutes(register func(router Router)) {
	if register == nil {
		return
	}

	register(newFiberRouterAdapter(s.app))
}

// MountRoutes allows future modules to mount grouped routes using abstract router interfaces.
func (s *Server) MountRoutes(prefix string, register func(router Router)) {
	if register == nil {
		return
	}

	register(newFiberRouterAdapter(s.app.Group(prefix)))
}

// Start begins listening on the resolved address.
func (s *Server) Start() error {
	s.logStartup(s.address)
	return s.app.Listen(s.address)
}

// StartWithListener begins serving using a provided listener.
func (s *Server) StartWithListener(listener net.Listener) error {
	s.logStartup(listener.Addr().String())
	return s.app.Listener(listener)
}

// Shutdown gracefully stops the server using the provided context.
func (s *Server) Shutdown(ctx context.Context) error {
	return s.app.ShutdownWithContext(ctx)
}

// Logger returns the resolved server logger instance.
func (s *Server) Logger() *zap.Logger {
	return s.logger
}

// logStartup writes a startup message using the server logger.
func (s *Server) logStartup(address string) {
	if s == nil || s.logger == nil {
		return
	}

	config := s.app.Config()
	s.logger.Info("http server starting",
		zap.String("address", address),
		zap.String("app_name", config.AppName),
		zap.Bool("prefork", config.Prefork),
	)
}

// mergeConfig resolves HTTP config values using optional core config fallbacks.
func mergeConfig(cfg Config, coreCfg *coreconfig.Core) Config {
	resolved := cfg

	if coreCfg != nil && strings.TrimSpace(coreCfg.Host) != "" {
		resolved.Host = strings.TrimSpace(coreCfg.Host)
	} else if strings.TrimSpace(resolved.Host) == "" {
		resolved.Host = "0.0.0.0"
	}

	if coreCfg != nil && coreCfg.Port > 0 {
		resolved.Port = coreCfg.Port
	} else if resolved.Port <= 0 {
		resolved.Port = 8080
	}

	if strings.TrimSpace(resolved.AppName) == "" {
		resolved.AppName = "mannaiah-http"
	}
	if strings.TrimSpace(resolved.ServerHeader) == "" {
		resolved.ServerHeader = "mannaiah"
	}
	if resolved.ReadTimeoutMS <= 0 {
		resolved.ReadTimeoutMS = 30000
	}
	if resolved.WriteTimeoutMS <= 0 {
		resolved.WriteTimeoutMS = 30000
	}
	if resolved.IdleTimeoutMS <= 0 {
		resolved.IdleTimeoutMS = 120000
	}

	return resolved
}

// buildAddress validates host and port and returns host:port format.
func buildAddress(host string, port int) (string, error) {
	if port <= 0 || port > 65535 {
		return "", ErrInvalidPort
	}

	return net.JoinHostPort(strings.TrimSpace(host), strconv.Itoa(port)), nil
}

// resolveLogger returns the provided logger or a no-op logger fallback.
func resolveLogger(providedLogger *zap.Logger) *zap.Logger {
	if providedLogger != nil {
		return providedLogger
	}

	return zap.NewNop()
}

// AddressFrom resolves host and port values from HTTP config and optional core config.
func AddressFrom(cfg Config, coreCfg *coreconfig.Core) (string, error) {
	resolved := mergeConfig(cfg, coreCfg)
	address, err := buildAddress(resolved.Host, resolved.Port)
	if err != nil {
		return "", fmt.Errorf("resolve http address: %w", err)
	}

	return address, nil
}

// fiberRouterAdapter adapts Fiber router registration to abstract router contracts.
type fiberRouterAdapter struct {
	// router is the wrapped Fiber router.
	router fiber.Router
}

// newFiberRouterAdapter creates a router adapter over a Fiber router.
func newFiberRouterAdapter(router fiber.Router) Router {
	return &fiberRouterAdapter{router: router}
}

// Get registers a GET route handler.
func (a *fiberRouterAdapter) Get(path string, handler Handler) {
	a.router.Get(path, adaptHandler(handler))
}

// Post registers a POST route handler.
func (a *fiberRouterAdapter) Post(path string, handler Handler) {
	a.router.Post(path, adaptHandler(handler))
}

// Put registers a PUT route handler.
func (a *fiberRouterAdapter) Put(path string, handler Handler) {
	a.router.Put(path, adaptHandler(handler))
}

// Patch registers a PATCH route handler.
func (a *fiberRouterAdapter) Patch(path string, handler Handler) {
	a.router.Patch(path, adaptHandler(handler))
}

// Delete registers a DELETE route handler.
func (a *fiberRouterAdapter) Delete(path string, handler Handler) {
	a.router.Delete(path, adaptHandler(handler))
}

// adaptHandler wraps abstract handlers into Fiber handlers.
func adaptHandler(handler Handler) fiber.Handler {
	return func(ctx *fiber.Ctx) error {
		if handler == nil {
			return nil
		}

		return handler(&fiberContextAdapter{ctx: ctx})
	}
}

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
