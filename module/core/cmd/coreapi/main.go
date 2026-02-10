package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"
	"mannaiah/module/contacts"
	contactevent "mannaiah/module/contacts/adapter/event"
	coreconfig "mannaiah/module/core/config"
	coredatabase "mannaiah/module/core/database"
	corehttp "mannaiah/module/core/http"
	corelogger "mannaiah/module/core/logger"
	coremsgplatform "mannaiah/module/core/messaging/platform"
	corewatermill "mannaiah/module/core/messaging/watermill"
	"mannaiah/module/core/startup"
	"mannaiah/module/core/swagger"
)

// main executes startup bootstrap and blocks until process shutdown.
func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := run(ctx, ".env"); err != nil {
		fmt.Fprintf(os.Stderr, "startup failed: %v\n", err)
		os.Exit(1)
	}
}

// run bootstraps infrastructure, modules, and HTTP serving lifecycle.
func run(ctx context.Context, envFile string) error {
	var coreCfg coreconfig.Core
	var httpCfg corehttp.Config
	var dbCfg coredatabase.Config
	var messagingCfg coremsgplatform.Config

	if err := coreconfig.Load(envFile, zap.NewNop(), &coreCfg, &httpCfg, &dbCfg, &messagingCfg); err != nil {
		return fmt.Errorf("load startup configuration: %w", err)
	}

	logger, err := corelogger.New(coreCfg.Logging)
	if err != nil {
		return fmt.Errorf("build logger: %w", err)
	}
	defer func() {
		_ = logger.Sync()
	}()

	db, err := coredatabase.Open(dbCfg, logger)
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("access sql db handle: %w", err)
	}
	defer func() {
		_ = sqlDB.Close()
	}()

	messaging, err := corewatermill.NewInMemoryPlatform(messagingCfg, logger)
	if err != nil {
		return fmt.Errorf("create messaging platform: %w", err)
	}
	defer func() {
		_ = messaging.Close()
	}()

	httpServer, err := corehttp.NewWithCore(httpCfg, &coreCfg, logger)
	if err != nil {
		return fmt.Errorf("create http server: %w", err)
	}

	document := swagger.NewDocument(swagger.Info{
		Title:       "Mannaiah API",
		Version:     "0.0.1",
		Description: "Mannaiah modular monolith API",
	})
	runtime, err := startup.NewRuntime(httpServer, document)
	if err != nil {
		return fmt.Errorf("create startup runtime: %w", err)
	}

	if err := runtime.AddOpenAPISpec(startup.CoreSpec()); err != nil {
		return fmt.Errorf("add core openapi spec: %w", err)
	}
	runtime.RegisterRoutes(registerCoreStatusRoute)

	contactPublisher, err := contactevent.NewPublisher(messaging.Publisher())
	if err != nil {
		return fmt.Errorf("create contacts integration publisher: %w", err)
	}

	contactsModule, err := contacts.New(db, contactPublisher)
	if err != nil {
		return fmt.Errorf("initialize contacts module: %w", err)
	}
	if err := contactsModule.Load(runtime); err != nil {
		return fmt.Errorf("load contacts module: %w", err)
	}

	runtime.ExposeOpenAPI("/openapi.json")

	serverErrors := make(chan error, 1)
	go func() {
		serverErrors <- httpServer.Start()
	}()

	messagingErrors := make(chan error, 1)
	go func() {
		messagingErrors <- messaging.Run(ctx)
	}()

	return waitForShutdown(ctx, db, httpServer, messaging, serverErrors, messagingErrors)
}

// registerCoreStatusRoute registers core status endpoints.
func registerCoreStatusRoute(router corehttp.Router) {
	router.Get("/status", func(ctx corehttp.Context) error {
		return ctx.Status(200).JSON(map[string]string{"status": "ok"})
	})
}

// waitForShutdown waits for process shutdown signals or runtime errors.
func waitForShutdown(
	ctx context.Context,
	db *gorm.DB,
	httpServer *corehttp.Server,
	messaging *corewatermill.InMemoryPlatform,
	serverErrors <-chan error,
	messagingErrors <-chan error,
) error {
	select {
	case <-ctx.Done():
		return shutdownResources(db, httpServer, messaging)
	case err := <-serverErrors:
		if err != nil {
			return fmt.Errorf("http server stopped: %w", err)
		}
		return shutdownResources(db, httpServer, messaging)
	case err := <-messagingErrors:
		if err != nil && !errors.Is(err, context.Canceled) {
			return fmt.Errorf("messaging router stopped: %w", err)
		}
		return shutdownResources(db, httpServer, messaging)
	}
}

// shutdownResources gracefully stops HTTP, messaging, and DB resources.
func shutdownResources(db *gorm.DB, httpServer *corehttp.Server, messaging *corewatermill.InMemoryPlatform) error {
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("shutdown http server: %w", err)
	}
	if err := messaging.Close(); err != nil {
		return fmt.Errorf("shutdown messaging platform: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("access sql db handle on shutdown: %w", err)
	}
	if err := sqlDB.Close(); err != nil {
		return fmt.Errorf("close sql db handle: %w", err)
	}

	return nil
}
