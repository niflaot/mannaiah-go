package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"mannaiah/module/analytics"
	analyticsport "mannaiah/module/analytics/port"
	"mannaiah/module/assets"
	assetevent "mannaiah/module/assets/adapter/event"
	assetstorage "mannaiah/module/assets/adapter/storage"
	assetsport "mannaiah/module/assets/port"
	"mannaiah/module/auth"
	"mannaiah/module/contacts"
	contactevent "mannaiah/module/contacts/adapter/event"
	contactsearch "mannaiah/module/contacts/adapter/search"
	contactapplication "mannaiah/module/contacts/application"
	contactdomain "mannaiah/module/contacts/domain"
	contactport "mannaiah/module/contacts/port"
	coreconfig "mannaiah/module/core/config"
	corecron "mannaiah/module/core/cron"
	coredatabase "mannaiah/module/core/database"
	coredatabasemigration "mannaiah/module/core/database/migration"
	corehttp "mannaiah/module/core/http"
	corelogger "mannaiah/module/core/logger"
	coremsgplatform "mannaiah/module/core/messaging/platform"
	corewatermill "mannaiah/module/core/messaging/watermill"
	coreredis "mannaiah/module/core/redis"
	coresearch "mannaiah/module/core/search"
	"mannaiah/module/core/startup"
	corestorage "mannaiah/module/core/storage"
	"mannaiah/module/core/swagger"
	coretelemetry "mannaiah/module/core/telemetry"
	"mannaiah/module/coupons"
	"mannaiah/module/email"
	"mannaiah/module/exports"
	exportsstorage "mannaiah/module/exports/adapter/storage"
	exportsport "mannaiah/module/exports/port"
	"mannaiah/module/falabella"
	falabellaproducts "mannaiah/module/falabella/adapter/products"
	falabellaport "mannaiah/module/falabella/port"
	"mannaiah/module/membership"
	membershipevent "mannaiah/module/membership/adapter/event"
	membershipdomain "mannaiah/module/membership/domain"
	membershipport "mannaiah/module/membership/port"
	"mannaiah/module/orders"
	ordercontacts "mannaiah/module/orders/adapter/contacts"
	orderevent "mannaiah/module/orders/adapter/event"
	orderproducts "mannaiah/module/orders/adapter/products"
	ordersearch "mannaiah/module/orders/adapter/search"
	"mannaiah/module/products"
	categorysearch "mannaiah/module/products/adapter/search/category"
	productsearch "mannaiah/module/products/adapter/search/product"
	tagsearch "mannaiah/module/products/adapter/search/tag"
	variationsearch "mannaiah/module/products/adapter/search/variation"
	"mannaiah/module/shipping"
	shippingevent "mannaiah/module/shipping/adapter/event"
	shippingsearch "mannaiah/module/shipping/adapter/search"
	"mannaiah/module/shopify"
	shopifyport "mannaiah/module/shopify/port"
	"mannaiah/module/syncrecord"
	syncrecorddomain "mannaiah/module/syncrecord/domain"
	syncrecordport "mannaiah/module/syncrecord/port"

	"github.com/gofiber/adaptor/v2"
	"github.com/gofiber/fiber/v2"

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
	var assetsCfg assets.Config
	var falabellaCfg falabella.Config
	var shopifyCfg shopify.Config
	var analyticsCfg analytics.Config
	var emailCfg email.Config
	var membershipCfg membership.Config
	var syncRecordCfg syncrecord.Config
	var shippingCfg shipping.Config
	var telemetryCfg coretelemetry.Config
	var redisCfg coreredis.Config
	var productsCfg products.Config

	if err := coreconfig.Load(
		envFile,
		zap.NewNop(),
		&coreCfg,
		&httpCfg,
		&dbCfg,
		&storageCfg,
		&messagingCfg,
		&authCfg,
		&cronCfg,
		&assetsCfg,
		&falabellaCfg,
		&shopifyCfg,
		&analyticsCfg,
		&emailCfg,
		&membershipCfg,
		&syncRecordCfg,
		&shippingCfg,
		&telemetryCfg,
		&redisCfg,
		&productsCfg,
	); err != nil {
		return fmt.Errorf("load startup configuration: %w", err)
	}

	logger, err := corelogger.New(coreCfg.Logging)
	if err != nil {
		return fmt.Errorf("build logger: %w", err)
	}
	zap.ReplaceGlobals(logger)
	defer func() {
		_ = logger.Sync()
	}()

	var sharedRedisCacheStore *coreredis.Store
	sharedRedisCacheStore, err = coreredis.New(redisCfg, logger)
	if err != nil {
		logger.Warn("redis cache disabled", zap.Error(err))
		sharedRedisCacheStore = nil
	} else {
		if pingErr := sharedRedisCacheStore.Ping(ctx); pingErr != nil {
			logger.Warn("redis cache ping failed; continuing with fail-open cache behavior", zap.Error(pingErr))
		}
		defer func() {
			if closeErr := sharedRedisCacheStore.Close(); closeErr != nil {
				logger.Warn("redis cache shutdown failed", zap.Error(closeErr))
			}
		}()
	}

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
		Version:     "3.0.0",
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

	// --- Search infrastructure (registered before module routes for path priority) ---
	contactSearchRepo, err := contactsearch.NewRepository(db)
	if err != nil {
		return fmt.Errorf("create contacts search repo: %w", err)
	}
	orderSearchRepo, err := ordersearch.NewRepository(db)
	if err != nil {
		return fmt.Errorf("create orders search repo: %w", err)
	}
	productSearchRepo, err := productsearch.NewRepository(db)
	if err != nil {
		return fmt.Errorf("create products search repo: %w", err)
	}
	categorySearchRepo, err := categorysearch.NewRepository(db)
	if err != nil {
		return fmt.Errorf("create categories search repo: %w", err)
	}
	variationSearchRepo, err := variationsearch.NewRepository(db)
	if err != nil {
		return fmt.Errorf("create variations search repo: %w", err)
	}
	tagSearchRepo, err := tagsearch.NewRepository(db)
	if err != nil {
		return fmt.Errorf("create tags search repo: %w", err)
	}
	shippingSearchRepo, err := shippingsearch.NewRepository(db)
	if err != nil {
		return fmt.Errorf("create shipping search repo: %w", err)
	}
	spotlightService := coresearch.NewSpotlightService(
		2*time.Second,
		contactSearchRepo,
		orderSearchRepo,
		productSearchRepo,
		categorySearchRepo,
		variationSearchRepo,
		tagSearchRepo,
		shippingSearchRepo,
	)

	if err := runtime.AddOpenAPISpec(coresearch.OpenAPISpec()); err != nil {
		return fmt.Errorf("add search openapi spec: %w", err)
	}

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
	membershipPublisher, err := membershipevent.NewPublisher(messaging.Publisher())
	if err != nil {
		return fmt.Errorf("create membership integration publisher: %w", err)
	}
	shippingPublisher, err := shippingevent.NewPublisher(messaging.Publisher())
	if err != nil {
		return fmt.Errorf("create shipping integration publisher: %w", err)
	}

	authModule, err := auth.New(authCfg, coreCfg.Environment, logger)
	if err != nil {
		return fmt.Errorf("initialize auth module: %w", err)
	}
	if err := authModule.Load(runtime); err != nil {
		return fmt.Errorf("load auth module: %w", err)
	}

	searchProtect := func(permission string, next corehttp.Handler) corehttp.Handler {
		return func(ctx corehttp.Context) error {
			if err := authModule.Require(ctx.Context(), ctx.GetHeader("Authorization"), permission); err != nil {
				if authModule.IsUnauthorized(err) {
					return corehttp.NewAppError(401, "unauthorized", err)
				}
				if authModule.IsForbidden(err) {
					return corehttp.NewAppError(403, "forbidden", err)
				}
				return corehttp.NewAppError(500, "auth_failed", err)
			}
			return next(ctx)
		}
	}

	runtime.RegisterRoutes(func(router corehttp.Router) {
		router.Get("/search/contacts", searchProtect("contact:view", coresearch.SearchHandlerFunc(contactSearchRepo)))
		router.Get("/search/orders", searchProtect("order:view", coresearch.SearchHandlerFunc(orderSearchRepo)))
		router.Get("/search/products", searchProtect("product:view", coresearch.SearchHandlerFunc(productSearchRepo)))
		router.Get("/search/categories", searchProtect("product:view", coresearch.SearchHandlerFunc(categorySearchRepo)))
		router.Get("/search/variations", searchProtect("product:view", coresearch.SearchHandlerFunc(variationSearchRepo)))
		router.Get("/search/tags", searchProtect("product:view", coresearch.SearchHandlerFunc(tagSearchRepo)))
		router.Get("/search/shipping", searchProtect("shipping:quotations", coresearch.SearchHandlerFunc(shippingSearchRepo)))
		router.Get("/search", searchProtect("contact:view", coresearch.SpotlightHandlerFunc(spotlightService)))
	})

	var syncRecordScheduler corecron.Scheduler
	if syncRecordCfg.Enabled && syncRecordCfg.CleanupEnabled {
		syncRecordScheduler, err = corecron.NewScheduler(cronCfg, logger)
		if err != nil {
			return fmt.Errorf("create sync record scheduler: %w", err)
		}
	}

	syncRecordModule, err := syncrecord.New(syncRecordCfg, db)
	if err != nil {
		return fmt.Errorf("initialize sync record module: %w", err)
	}
	syncRecordModule.ConfigureScheduler(syncRecordScheduler)
	syncRecordModule.SetAuthorizer(authModule)
	if err := syncRecordModule.Load(runtime); err != nil {
		return fmt.Errorf("load sync record module: %w", err)
	}
	if err := syncRecordModule.Start(ctx); err != nil {
		return fmt.Errorf("start sync record module: %w", err)
	}
	defer func() {
		stopCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = syncRecordModule.Stop(stopCtx)
	}()

	recorderAdapter := syncRecordRecorderAdapter{recorder: syncRecordModule.Recorder()}

	contactsModule, err := contacts.New(db, contactPublisher)
	if err != nil {
		return fmt.Errorf("initialize contacts module: %w", err)
	}
	contactsModule.SetAuthorizer(authModule)
	if err := contactsModule.Load(runtime); err != nil {
		return fmt.Errorf("load contacts module: %w", err)
	}

	membershipModule, err := membership.New(
		membershipCfg,
		db,
		membershipContactLookupAdapter{service: contactsModule.Service()},
		membershipPublisher,
	)
	if err != nil {
		return fmt.Errorf("initialize membership module: %w", err)
	}
	membershipModule.SetSyncRecorder(membershipSyncRecorderAdapter{recorder: recorderAdapter})
	membershipModule.SetAuthorizer(authModule)
	if err := membershipModule.Load(runtime); err != nil {
		return fmt.Errorf("load membership module: %w", err)
	}
	analyticsModule, err := analytics.New(analyticsCfg, db, messaging.Registrar())
	if err != nil {
		return fmt.Errorf("initialize analytics module: %w", err)
	}
	analyticsModule.SetSyncRecorder(analyticsSyncRecorderAdapter{recorder: recorderAdapter})
	analyticsModule.SetAuthorizer(authModule)
	if err := analyticsModule.Load(runtime); err != nil {
		return fmt.Errorf("load analytics module: %w", err)
	}
	if err := analyticsModule.Start(ctx); err != nil {
		return fmt.Errorf("start analytics module: %w", err)
	}
	defer func() {
		_ = analyticsModule.Stop()
	}()

	emailModule, err := email.New(emailCfg, db)
	if err != nil {
		return fmt.Errorf("initialize email module: %w", err)
	}
	emailModule.SetMembershipStamper(emailMembershipStamperAdapter{service: membershipModule.Service()})
	emailModule.SetAuthorizer(authModule)
	if err := emailModule.Load(runtime); err != nil {
		return fmt.Errorf("load email module: %w", err)
	}
	shippingEmailTemplateRenderer, err := newShippingTemplateRenderer()
	if err != nil {
		return fmt.Errorf("initialize shipping transactional template renderer: %w", err)
	}

	shippingModule, err := shipping.New(shippingCfg, db, shippingPublisher)
	if err != nil {
		return fmt.Errorf("initialize shipping module: %w", err)
	}
	shippingModule.DispatchService().SetBatchManifestDocumentCacheStore(sharedRedisCacheStore)
	shippingModule.MarkService().SetRotulusDocumentCacheStore(sharedRedisCacheStore)
	shippingModule.SetAuthorizer(authModule)
	if err := shippingModule.Load(runtime); err != nil {
		return fmt.Errorf("load shipping module: %w", err)
	}
	if err := shippingModule.Start(ctx); err != nil {
		return fmt.Errorf("start shipping module: %w", err)
	}

	var assetsScheduler corecron.Scheduler
	if assetsCfg.JPGWorkerEnabled {
		assetsScheduler, err = corecron.NewScheduler(cronCfg, logger)
		if err != nil {
			return fmt.Errorf("create assets scheduler: %w", err)
		}
	}

	assetsModule, err := assets.NewWithConfig(assetsCfg, db, assetStorage, logger, assetPublisher)
	if err != nil {
		return fmt.Errorf("initialize assets module: %w", err)
	}
	assetsModule.SetSyncRecorder(assetSyncRecorderAdapter{recorder: recorderAdapter})
	assetsModule.ConfigureScheduler(assetsScheduler)
	assetsModule.SetAuthorizer(authModule)
	if err := assetsModule.Load(runtime); err != nil {
		return fmt.Errorf("load assets module: %w", err)
	}
	if err := assetsModule.Start(ctx); err != nil {
		return fmt.Errorf("start assets module: %w", err)
	}
	defer func() {
		stopCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = assetsModule.Stop(stopCtx)
	}()

	productsModule, err := products.NewWithConfig(db, assetsModule.Service(), productsCfg, sharedRedisCacheStore, logger)
	if err != nil {
		return fmt.Errorf("initialize products module: %w", err)
	}
	productsScheduler, err := corecron.NewScheduler(cronCfg, logger)
	if err != nil {
		return fmt.Errorf("create products scheduler: %w", err)
	}
	productsModule.ConfigureScheduler(productsScheduler)
	productsModule.SetAuthorizer(authModule)
	if err := productsModule.Load(runtime); err != nil {
		return fmt.Errorf("load products module: %w", err)
	}
	if err := productsModule.Start(ctx); err != nil {
		return fmt.Errorf("start products module: %w", err)
	}
	defer func() {
		stopCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = productsModule.Stop(stopCtx)
	}()

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
	falabellaModule.SetSyncRecorder(falabellaSyncRecorderAdapter{recorder: recorderAdapter})
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

	exportStorage, err := exportsstorage.NewCoreStoreAdapter(storageStore)
	if err != nil {
		return fmt.Errorf("initialize exports storage adapter: %w", err)
	}
	exportsModule, err := exports.New(
		db,
		exportStorage,
		contactsModule.Service(),
		ordersModule.Service(),
		exportMembershipConsentAdapter{service: membershipModule.Service()},
	)
	if err != nil {
		return fmt.Errorf("initialize exports module: %w", err)
	}
	exportsModule.SetAuthorizer(authModule)
	if err := exportsModule.Load(runtime); err != nil {
		return fmt.Errorf("load exports module: %w", err)
	}

	shippingOrderSummaryAdapter := shippingBatchManifestOrderSummaryAdapter{
		orders: ordersModule.Service(),
	}
	shippingModule.DispatchService().SetBatchManifestOrderSummaryResolver(shippingOrderSummaryAdapter)
	shippingModule.MarkService().SetRotulusOrderSummaryResolver(shippingOrderSummaryAdapter)
	shippingModule.SetQuotationOrderSource(shippingOrderQuotationSourceAdapter{
		orders:   ordersModule.Service(),
		contacts: contactsModule.Service(),
	})
	shippingModule.SetQuotationProductSource(shippingProductQuotationSourceAdapter{products: productsModule.Service()})

	couponsModule, err := coupons.New(db, messaging.Publisher())
	if err != nil {
		return fmt.Errorf("initialize coupons module: %w", err)
	}
	couponsModule.SetAuthorizer(authModule)
	if err := couponsModule.Load(runtime); err != nil {
		return fmt.Errorf("load coupons module: %w", err)
	}

	shopifyModule, err := shopify.New(
		shopifyCfg,
		db,
		contactsModule.Service(),
		ordersModule.Service(),
		logger,
		messaging.Registrar(),
	)
	if err != nil {
		return fmt.Errorf("initialize shopify module: %w", err)
	}
	if shopifyCfg.SyncContacts || shopifyCfg.SyncOrders {
		shopifyScheduler, schedulerErr := corecron.NewScheduler(cronCfg, logger)
		if schedulerErr != nil {
			return fmt.Errorf("create shopify scheduler: %w", schedulerErr)
		}
		shopifyModule.ConfigureScheduler(shopifyScheduler)
	}
	shopifyModule.SetSyncRecorder(shopifySyncRecorderAdapter{recorder: recorderAdapter})
	shopifyModule.SetAuthorizer(authModule)
	if err := shopifyModule.Load(runtime); err != nil {
		return fmt.Errorf("load shopify module: %w", err)
	}
	if err := shopifyModule.Start(ctx); err != nil {
		return fmt.Errorf("start shopify module: %w", err)
	}
	defer func() {
		stopCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = shopifyModule.Stop(stopCtx)
	}()

	if err := registerShippingMarkOrderCompletionConsumer(
		messaging.Registrar(),
		ordersModule.Service(),
		logger,
	); err != nil {
		return fmt.Errorf("register shipping mark order completion consumer: %w", err)
	}
	if err := registerShippingMarkTransactionalEmailConsumer(
		messaging.Registrar(),
		shippingEmailConsumerDependencies{
			marks:      shippingModule.MarkService(),
			carriers:   shippingModule.CarrierService(),
			orders:     ordersModule.Service(),
			contacts:   contactsModule.Service(),
			products:   productsModule.Service(),
			variations: productsModule.VariationService(),
			emails:     emailModule.Service(),
			assetResolver: analyticsAssetURLResolver{
				assetService: assetsModule.Service(),
				assetBaseURL: resolveMarketingAssetBaseURL(
					falabellaCfg.ProductImageBaseURL,
					buildStorageBucketBaseURL(storageCfg.Endpoint, storageCfg.BucketName),
				),
			},
			templateRenderer: shippingEmailTemplateRenderer,
			trackingBaseURL:  shippingCfg.TransactionalTrackingBaseURL,
			helpPhoneURL:     shippingCfg.TransactionalHelpPhoneURL,
		},
		logger,
	); err != nil {
		return fmt.Errorf("register shipping mark transactional email consumer: %w", err)
	}

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

// syncRecordRecorderAdapter adapts syncrecord recorder behavior for module-specific recorder ports.
type syncRecordRecorderAdapter struct {
	// recorder defines base sync record recorder dependencies.
	recorder syncrecordport.Recorder
}

// StartRun starts one synchronization run and returns a run identifier.
func (a syncRecordRecorderAdapter) StartRun(ctx context.Context, kind string, trigger string) (string, error) {
	if a.recorder == nil {
		return "", nil
	}

	return a.recorder.StartRun(ctx, syncrecordport.StartRunInput{
		Kind:    syncrecorddomain.SyncKind(kind),
		Trigger: syncrecorddomain.SyncTrigger(trigger),
	})
}

// CompleteRun marks one synchronization run as completed.
func (a syncRecordRecorderAdapter) CompleteRun(ctx context.Context, runID string, processed int, succeeded int, failed int, skipped int) error {
	if a.recorder == nil {
		return nil
	}

	return a.recorder.CompleteRun(ctx, syncrecordport.FinishRunInput{
		RunID:     runID,
		Processed: processed,
		Succeeded: succeeded,
		Failed:    failed,
		Skipped:   skipped,
	})
}

// FailRun marks one synchronization run as failed.
func (a syncRecordRecorderAdapter) FailRun(ctx context.Context, runID string, processed int, succeeded int, failed int, skipped int, syncErrors []errorPayload) error {
	if a.recorder == nil {
		return nil
	}

	errorsPayload := make([]syncrecorddomain.SyncRunError, 0, len(syncErrors))
	for _, syncErr := range syncErrors {
		if strings.TrimSpace(syncErr.Message) == "" {
			continue
		}
		errorsPayload = append(errorsPayload, syncrecorddomain.SyncRunError{
			ErrorType: syncErr.Type,
			ErrorCode: syncErr.Code,
			Message:   syncErr.Message,
		})
	}

	return a.recorder.FailRun(ctx, syncrecordport.FinishRunInput{
		RunID:     runID,
		Processed: processed,
		Succeeded: succeeded,
		Failed:    failed,
		Skipped:   skipped,
		Errors:    errorsPayload,
	})
}

// errorPayload defines module-agnostic sync error payload values.
type errorPayload struct {
	// Type defines high-level error category values.
	Type string
	// Code defines machine-readable error code values.
	Code string
	// Message defines error message values.
	Message string
}

// shopifySyncRecorderAdapter adapts sync recorders for Shopify sync recorder ports.
type shopifySyncRecorderAdapter struct {
	// recorder defines base sync recorder dependencies.
	recorder syncRecordRecorderAdapter
}

// StartRun starts one synchronization run and returns a run identifier.
func (a shopifySyncRecorderAdapter) StartRun(ctx context.Context, kind string, trigger string) (string, error) {
	return a.recorder.StartRun(ctx, kind, trigger)
}

// CompleteRun marks one synchronization run as completed.
func (a shopifySyncRecorderAdapter) CompleteRun(ctx context.Context, runID string, processed int, succeeded int, failed int, skipped int) error {
	return a.recorder.CompleteRun(ctx, runID, processed, succeeded, failed, skipped)
}

// FailRun marks one synchronization run as failed.
func (a shopifySyncRecorderAdapter) FailRun(ctx context.Context, runID string, processed int, succeeded int, failed int, skipped int, syncErrors []shopifyport.SyncError) error {
	adapted := make([]errorPayload, 0, len(syncErrors))
	for _, syncErr := range syncErrors {
		adapted = append(adapted, errorPayload{Type: syncErr.Type, Code: syncErr.Code, Message: syncErr.Message})
	}

	return a.recorder.FailRun(ctx, runID, processed, succeeded, failed, skipped, adapted)
}

// assetSyncRecorderAdapter adapts sync recorders for assets sync recorder ports.
type assetSyncRecorderAdapter struct {
	// recorder defines base sync recorder dependencies.
	recorder syncRecordRecorderAdapter
}

// StartRun starts one synchronization run and returns a run identifier.
func (a assetSyncRecorderAdapter) StartRun(ctx context.Context, kind string, trigger string) (string, error) {
	return a.recorder.StartRun(ctx, kind, trigger)
}

// CompleteRun marks one synchronization run as completed.
func (a assetSyncRecorderAdapter) CompleteRun(ctx context.Context, runID string, processed int, succeeded int, failed int, skipped int) error {
	return a.recorder.CompleteRun(ctx, runID, processed, succeeded, failed, skipped)
}

// FailRun marks one synchronization run as failed.
func (a assetSyncRecorderAdapter) FailRun(ctx context.Context, runID string, processed int, succeeded int, failed int, skipped int, syncErrors []assetsport.SyncError) error {
	adapted := make([]errorPayload, 0, len(syncErrors))
	for _, syncErr := range syncErrors {
		adapted = append(adapted, errorPayload{Type: syncErr.Type, Code: syncErr.Code, Message: syncErr.Message})
	}

	return a.recorder.FailRun(ctx, runID, processed, succeeded, failed, skipped, adapted)
}

// falabellaSyncRecorderAdapter adapts sync recorders for Falabella sync recorder ports.
type falabellaSyncRecorderAdapter struct {
	// recorder defines base sync recorder dependencies.
	recorder syncRecordRecorderAdapter
}

// StartRun starts one synchronization run and returns a run identifier.
func (a falabellaSyncRecorderAdapter) StartRun(ctx context.Context, kind string, trigger string) (string, error) {
	return a.recorder.StartRun(ctx, kind, trigger)
}

// CompleteRun marks one synchronization run as completed.
func (a falabellaSyncRecorderAdapter) CompleteRun(ctx context.Context, runID string, processed int, succeeded int, failed int, skipped int) error {
	return a.recorder.CompleteRun(ctx, runID, processed, succeeded, failed, skipped)
}

// FailRun marks one synchronization run as failed.
func (a falabellaSyncRecorderAdapter) FailRun(ctx context.Context, runID string, processed int, succeeded int, failed int, skipped int, syncErrors []falabellaport.SyncError) error {
	adapted := make([]errorPayload, 0, len(syncErrors))
	for _, syncErr := range syncErrors {
		adapted = append(adapted, errorPayload{Type: syncErr.Type, Code: syncErr.Code, Message: syncErr.Message})
	}

	return a.recorder.FailRun(ctx, runID, processed, succeeded, failed, skipped, adapted)
}

// membershipContactLookupAdapter adapts contacts service behavior for membership lookups.
type membershipContactLookupAdapter struct {
	// service defines contacts lookup dependencies.
	service contactapplication.Service
}

// exportMembershipStatusService defines membership status behavior required by exports.
type exportMembershipStatusService interface {
	// GetStatuses retrieves current statuses by contact across all channels.
	GetStatuses(ctx context.Context, contactID string) ([]membershipdomain.Status, error)
}

// exportMembershipConsentAdapter adapts membership status lookups for contact exports.
type exportMembershipConsentAdapter struct {
	// service defines membership status dependencies.
	service exportMembershipStatusService
}

// GetContactStatuses returns latest consent statuses for a contact.
func (a exportMembershipConsentAdapter) GetContactStatuses(ctx context.Context, contactID string) ([]exportsport.ContactConsentStatus, error) {
	if a.service == nil {
		return nil, nil
	}

	statuses, err := a.service.GetStatuses(ctx, contactID)
	if errors.Is(err, membershipdomain.ErrStatusNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	result := make([]exportsport.ContactConsentStatus, 0, len(statuses))
	for _, status := range statuses {
		result = append(result, exportsport.ContactConsentStatus{
			Channel:    string(status.Channel),
			Action:     string(status.Action),
			OccurredAt: status.OccurredAt,
		})
	}

	return result, nil
}

// FindByEmail resolves one contact by normalized email values.
func (a membershipContactLookupAdapter) FindByEmail(ctx context.Context, email string) (*membershipport.ContactSnapshot, error) {
	if a.service == nil {
		return nil, membershipdomain.ErrContactNotFound
	}

	page, err := a.service.List(ctx, contactport.ListQuery{
		Page:  1,
		Limit: 1,
		Email: strings.ToLower(strings.TrimSpace(email)),
	})
	if err != nil {
		return nil, err
	}
	if page == nil || len(page.Data) == 0 {
		return nil, membershipdomain.ErrContactNotFound
	}

	contact := page.Data[0]
	return &membershipport.ContactSnapshot{ID: contact.ID, Email: contact.Email, Metadata: contact.Metadata}, nil
}

// ListByMetadata resolves contacts by metadata key/value filters.
func (a membershipContactLookupAdapter) ListByMetadata(ctx context.Context, metadataKey string, metadataValue string, page int, limit int) ([]membershipport.ContactSnapshot, int64, error) {
	if a.service == nil {
		return nil, 0, nil
	}

	rows, err := a.service.List(ctx, contactport.ListQuery{
		Page:          page,
		Limit:         limit,
		MetadataKey:   metadataKey,
		MetadataValue: metadataValue,
	})
	if err != nil {
		return nil, 0, err
	}

	result := make([]membershipport.ContactSnapshot, 0, len(rows.Data))
	for _, row := range rows.Data {
		result = append(result, membershipport.ContactSnapshot{ID: row.ID, Email: row.Email, Metadata: row.Metadata})
	}

	return result, rows.Total, nil
}

// analyticsSyncRecorderAdapter adapts sync recorders for analytics sync recorder ports.
type analyticsSyncRecorderAdapter struct {
	// recorder defines base sync recorder dependencies.
	recorder syncRecordRecorderAdapter
}

// StartRun starts one synchronization run and returns a run identifier.
func (a analyticsSyncRecorderAdapter) StartRun(ctx context.Context, kind string, trigger string) (string, error) {
	return a.recorder.StartRun(ctx, kind, trigger)
}

// CompleteRun marks one synchronization run as completed.
func (a analyticsSyncRecorderAdapter) CompleteRun(ctx context.Context, runID string, processed int, succeeded int, failed int, skipped int) error {
	return a.recorder.CompleteRun(ctx, runID, processed, succeeded, failed, skipped)
}

// FailRun marks one synchronization run as failed.
func (a analyticsSyncRecorderAdapter) FailRun(ctx context.Context, runID string, processed int, succeeded int, failed int, skipped int, syncErrors []analyticsport.SyncError) error {
	adapted := make([]errorPayload, 0, len(syncErrors))
	for _, syncErr := range syncErrors {
		adapted = append(adapted, errorPayload{Type: syncErr.Type, Code: syncErr.Code, Message: syncErr.Message})
	}

	return a.recorder.FailRun(ctx, runID, processed, succeeded, failed, skipped, adapted)
}

// membershipSyncRecorderAdapter adapts sync recorders for membership sync recorder ports.
type membershipSyncRecorderAdapter struct {
	// recorder defines base sync recorder dependencies.
	recorder syncRecordRecorderAdapter
}

// StartRun starts one synchronization run and returns a run identifier.
func (a membershipSyncRecorderAdapter) StartRun(ctx context.Context, kind string, trigger string) (string, error) {
	return a.recorder.StartRun(ctx, kind, trigger)
}

// CompleteRun marks one synchronization run as completed.
func (a membershipSyncRecorderAdapter) CompleteRun(ctx context.Context, runID string, processed int, succeeded int, failed int, skipped int) error {
	return a.recorder.CompleteRun(ctx, runID, processed, succeeded, failed, skipped)
}

// FailRun marks one synchronization run as failed.
func (a membershipSyncRecorderAdapter) FailRun(ctx context.Context, runID string, processed int, succeeded int, failed int, skipped int, syncErrors []membershipport.SyncError) error {
	adapted := make([]errorPayload, 0, len(syncErrors))
	for _, syncErr := range syncErrors {
		adapted = append(adapted, errorPayload{Type: syncErr.Type, Code: syncErr.Code, Message: syncErr.Message})
	}

	return a.recorder.FailRun(ctx, runID, processed, succeeded, failed, skipped, adapted)
}

// emailMembershipStamperAdapter adapts membership stamping behavior for email complaint handling.
type emailMembershipStamperAdapter struct {
	// service defines membership stamper dependencies.
	service membershipport.Stamper
}

// OptOutByEmail stamps email opt-out for one recipient.
func (a emailMembershipStamperAdapter) OptOutByEmail(ctx context.Context, email string, source string) error {
	if a.service == nil {
		return nil
	}

	_, err := a.service.Stamp(ctx, membershipport.StampCommand{
		Email:   strings.TrimSpace(email),
		Channel: membershipdomain.ChannelEmail,
		Action:  membershipdomain.ActionOptOut,
		Source:  strings.TrimSpace(source),
	})
	return err
}

// resolveContactDisplayName resolves one contact display name from legal/personal name fields.
func resolveContactDisplayName(contact *contactdomain.Contact) string {
	if contact == nil {
		return ""
	}

	legalName := strings.TrimSpace(contact.LegalName)
	if legalName != "" {
		return legalName
	}

	firstName := strings.TrimSpace(contact.FirstName)
	lastName := strings.TrimSpace(contact.LastName)
	return strings.TrimSpace(firstName + " " + lastName)
}
