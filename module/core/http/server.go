package http

import (
	"context"
	"errors"
	"fmt"
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
		AppName:      resolvedCfg.AppName,
		Prefork:      resolvedCfg.Prefork,
		ServerHeader: resolvedCfg.ServerHeader,
		ReadTimeout:  time.Duration(resolvedCfg.ReadTimeoutMS) * time.Millisecond,
		WriteTimeout: time.Duration(resolvedCfg.WriteTimeoutMS) * time.Millisecond,
		IdleTimeout:  time.Duration(resolvedCfg.IdleTimeoutMS) * time.Millisecond,
	})

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

// Register allows future modules to register app-level routes and middleware.
func (s *Server) Register(register func(app *fiber.App)) {
	if register == nil {
		return
	}

	register(s.app)
}

// Mount allows future modules to mount grouped routes under a prefix.
func (s *Server) Mount(prefix string, register func(router fiber.Router)) {
	if register == nil {
		return
	}

	register(s.app.Group(prefix))
}

// Start begins listening on the resolved address.
func (s *Server) Start() error {
	return s.app.Listen(s.address)
}

// StartWithListener begins serving using a provided listener.
func (s *Server) StartWithListener(listener net.Listener) error {
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

// mergeConfig resolves HTTP config values using optional core config fallbacks.
func mergeConfig(cfg Config, coreCfg *coreconfig.Core) Config {
	resolved := cfg

	if strings.TrimSpace(resolved.Host) == "" {
		if coreCfg != nil && strings.TrimSpace(coreCfg.Host) != "" {
			resolved.Host = strings.TrimSpace(coreCfg.Host)
		} else {
			resolved.Host = "0.0.0.0"
		}
	}

	if resolved.Port <= 0 {
		if coreCfg != nil && coreCfg.Port > 0 {
			resolved.Port = coreCfg.Port
		} else {
			resolved.Port = 8080
		}
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
