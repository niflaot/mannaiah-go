package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gofiber/adaptor/v2"
	"github.com/gofiber/fiber/v2"
	"mannaiah/module/assets"
	assetevent "mannaiah/module/assets/adapter/event"
	assetstorage "mannaiah/module/assets/adapter/storage"
	"mannaiah/module/auth"
	"mannaiah/module/contacts"
	contactevent "mannaiah/module/contacts/adapter/event"
	coreconfig "mannaiah/module/core/config"
	corecron "mannaiah/module/core/cron"
	coredatabase "mannaiah/module/core/database"
	coredatabasemigration "mannaiah/module/core/database/migration"
	corehttp "mannaiah/module/core/http"
	corelogger "mannaiah/module/core/logger"
	coremsgplatform "mannaiah/module/core/messaging/platform"
	corewatermill "mannaiah/module/core/messaging/watermill"
	"mannaiah/module/core/startup"
	corestorage "mannaiah/module/core/storage"
	"mannaiah/module/core/swagger"
	coretelemetry "mannaiah/module/core/telemetry"
	"mannaiah/module/falabella"
	falabellaproducts "mannaiah/module/falabella/adapter/products"
	"mannaiah/module/orders"
	ordercontacts "mannaiah/module/orders/adapter/contacts"
	orderevent "mannaiah/module/orders/adapter/event"
	orderproducts "mannaiah/module/orders/adapter/products"
	"mannaiah/module/products"
	"mannaiah/module/woocommerce"
	wooevent "mannaiah/module/woocommerce/adapter/event"

	"go.uber.org/zap"
	"gorm.io/gorm"
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
	var storageCfg corestorage.Config
	var messagingCfg coremsgplatform.Config
	var authCfg auth.Config
	var cronCfg corecron.Config
	var falabellaCfg falabella.Config
	var wooCfg woocommerce.Config
	var telemetryCfg coretelemetry.Config

	if err := coreconfig.Load(envFile, zap.NewNop(), &coreCfg, &httpCfg, &dbCfg, &storageCfg, &messagingCfg, &authCfg, &cronCfg, &falabellaCfg, &wooCfg, &telemetryCfg); err != nil {
		return fmt.Errorf("load startup configuration: %w", err)
	}

	logger, err := corelogger.New(coreCfg.Logging)
	if err != nil {
		return fmt.Errorf("build logger: %w", err)
	}
	defer func() {
		_ = logger.Sync()
	}()

	telemetryProvider, telemetryErr := coretelemetry.Init(ctx, telemetryCfg, logger)
	if telemetryErr != nil {
		return fmt.Errorf("initialize telemetry: %w", telemetryErr)
	}
	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if shutdownErr := telemetryProvider.Shutdown(shutdownCtx); shutdownErr != nil {
			logger.Warn("telemetry shutdown failed", zap.Error(shutdownErr))
		}
	}()

	db, err := coredatabase.Open(dbCfg, logger)
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}
	if migrationErr := coredatabasemigration.Apply(ctx, db, coredatabasemigration.FromDatabaseConfig(dbCfg), logger); migrationErr != nil {
		return fmt.Errorf("apply database migrations: %w", migrationErr)
	}
	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("access sql db handle: %w", err)
	}
	telemetryProvider.StartSQLStatsCollector(sqlDB)
	defer func() {
		_ = sqlDB.Close()
	}()

	storageStore := corestorage.NewS3(storageCfg, logger)
	assetStorage, err := assetstorage.NewCoreStoreAdapter(storageStore)
	if err != nil {
		return fmt.Errorf("create asset storage adapter: %w", err)
	}
	if availabilityErr := assetStorage.AvailabilityError(); availabilityErr != nil {
		return fmt.Errorf("storage is mandatory: %w", availabilityErr)
	}

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
	httpServer.Register(func(app *fiber.App) {
		app.Use(telemetryProvider.HTTPMiddleware())
		app.Get(telemetryProvider.MetricsPath(), adaptor.HTTPHandler(telemetryProvider.MetricsHandler()))
	})

	document := swagger.NewDocument(swagger.Info{
		Title:       "Mannaiah API",
		Version:     "1.2.2",
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
	assetPublisher, err := assetevent.NewPublisher(messaging.Publisher())
	if err != nil {
		return fmt.Errorf("create assets integration publisher: %w", err)
	}
	orderPublisher, err := orderevent.NewPublisher(messaging.Publisher())
	if err != nil {
		return fmt.Errorf("create orders integration publisher: %w", err)
	}
	wooPublisher, err := wooevent.NewPublisher(messaging.Publisher())
	if err != nil {
		return fmt.Errorf("create woocommerce integration publisher: %w", err)
	}

	authModule, err := auth.New(authCfg, coreCfg.Environment, logger)
	if err != nil {
		return fmt.Errorf("initialize auth module: %w", err)
	}
	if err := authModule.Load(runtime); err != nil {
		return fmt.Errorf("load auth module: %w", err)
	}

	contactsModule, err := contacts.New(db, contactPublisher)
	if err != nil {
		return fmt.Errorf("initialize contacts module: %w", err)
	}
	contactsModule.SetAuthorizer(authModule)
	if err := contactsModule.Load(runtime); err != nil {
		return fmt.Errorf("load contacts module: %w", err)
	}

	assetsModule, err := assets.New(db, assetStorage, assetPublisher)
	if err != nil {
		return fmt.Errorf("initialize assets module: %w", err)
	}
	assetsModule.SetAuthorizer(authModule)
	if err := assetsModule.Load(runtime); err != nil {
		return fmt.Errorf("load assets module: %w", err)
	}

	productsModule, err := products.New(db, assetsModule.Service())
	if err != nil {
		return fmt.Errorf("initialize products module: %w", err)
	}
	productsModule.SetAuthorizer(authModule)
	if err := productsModule.Load(runtime); err != nil {
		return fmt.Errorf("load products module: %w", err)
	}

	falabellaCatalog, err := falabellaproducts.NewCatalog(
		productsModule.Service(),
		falabellaproducts.WithVariationService(productsModule.VariationService()),
		falabellaproducts.WithAssetService(assetsModule.Service()),
		falabellaproducts.WithAssetBaseURL(falabellaCfg.ProductImageBaseURL),
	)
	if err != nil {
		return fmt.Errorf("initialize falabella products catalog: %w", err)
	}
	falabellaModule, err := falabella.New(falabellaCfg, logger, falabellaCatalog)
	if err != nil {
		return fmt.Errorf("initialize falabella module: %w", err)
	}
	if syncStatusErr := falabellaModule.ConfigureSyncStatus(db); syncStatusErr != nil {
		return fmt.Errorf("configure falabella sync status: %w", syncStatusErr)
	}
	falabellaScheduler, err := corecron.NewScheduler(cronCfg, logger)
	if err != nil {
		return fmt.Errorf("create falabella scheduler: %w", err)
	}
	falabellaModule.ConfigureScheduler(falabellaScheduler)
	falabellaModule.SetAuthorizer(authModule)
	if err := falabellaModule.Load(runtime); err != nil {
		return fmt.Errorf("load falabella module: %w", err)
	}
	if err := falabellaModule.Start(ctx); err != nil {
		return fmt.Errorf("start falabella module: %w", err)
	}
	defer func() {
		stopCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = falabellaModule.Stop(stopCtx)
	}()

	orderCustomerSource, err := ordercontacts.NewSource(contactsModule.Service())
	if err != nil {
		return fmt.Errorf("create order customer source: %w", err)
	}
	orderProductResolver, err := orderproducts.NewResolver(db)
	if err != nil {
		return fmt.Errorf("create order product resolver: %w", err)
	}
	ordersModule, err := orders.NewWithPublisher(db, orderCustomerSource, orderPublisher, orderProductResolver)
	if err != nil {
		return fmt.Errorf("initialize orders module: %w", err)
	}
	ordersModule.SetAuthorizer(authModule)
	if err := ordersModule.Load(runtime); err != nil {
		return fmt.Errorf("load orders module: %w", err)
	}

	var wooScheduler corecron.Scheduler
	if wooCfg.SyncContacts || wooCfg.SyncOrders {
		wooScheduler, err = corecron.NewScheduler(cronCfg, logger)
		if err != nil {
			return fmt.Errorf("create woocommerce scheduler: %w", err)
		}
	}

	wooModule, err := woocommerce.NewWithMessaging(
		wooCfg,
		contactsModule.Service(),
		ordersModule.Service(),
		wooScheduler,
		logger,
		messaging.Registrar(),
		wooPublisher,
	)
	if err != nil {
		return fmt.Errorf("initialize woocommerce module: %w", err)
	}
	wooModule.SetAuthorizer(authModule)
	if err := wooModule.Load(runtime); err != nil {
		return fmt.Errorf("load woocommerce module: %w", err)
	}
	if err := wooModule.Start(ctx); err != nil {
		return fmt.Errorf("start woocommerce module: %w", err)
	}
	defer func() {
		stopCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = wooModule.Stop(stopCtx)
	}()

	runtime.ExposeOpenAPI("/openapi.json")
	runtime.ExposeOpenAPIUI("/docs", "/openapi.json", "Mannaiah API Docs")

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
