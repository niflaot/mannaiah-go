package http

import (
	"context"
	"net"
	"strings"
	"time"

	fiberzap "github.com/gofiber/contrib/fiberzap/v2"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/limiter"
	"go.uber.org/zap"
	coreconfig "mannaiah/module/core/config"
)

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
		ErrorHandler:          errorHandlerWithLogger(logger),
	})

	if origins := strings.TrimSpace(resolvedCfg.CORSAllowedOrigins); origins != "" {
		app.Use(cors.New(cors.Config{
			AllowOrigins:     origins,
			AllowMethods:     "GET,POST,PATCH,DELETE,OPTIONS",
			AllowHeaders:     "Authorization,Content-Type,Accept",
			AllowCredentials: false,
			MaxAge:           600,
		}))
	}

	if resolvedCfg.RateLimitMax > 0 {
		window := time.Duration(resolvedCfg.RateLimitWindowMS) * time.Millisecond
		if window <= 0 {
			window = time.Minute
		}
		app.Use(limiter.New(limiter.Config{
			Max:        resolvedCfg.RateLimitMax,
			Expiration: window,
		}))
	}

	app.Use(rayIDMiddleware)
	app.Use(fiberzap.New(fiberzap.Config{
		Logger:     logger,
		Next:       shouldSkipAccessLog,
		FieldsFunc: accessLogFields,
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
	if s == nil || s.logger == nil || s.app == nil {
		return
	}

	config := s.app.Config()
	s.logger.Info("http server starting",
		zap.String("address", address),
		zap.String("app_name", config.AppName),
		zap.Bool("prefork", config.Prefork),
	)
}
