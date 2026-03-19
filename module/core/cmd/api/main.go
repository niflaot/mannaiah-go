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

	"github.com/gofiber/adaptor/v2"
	"github.com/gofiber/fiber/v2"
	"mannaiah/module/analytics"
	analyticsport "mannaiah/module/analytics/port"
	"mannaiah/module/assets"
	assetevent "mannaiah/module/assets/adapter/event"
	assetstorage "mannaiah/module/assets/adapter/storage"
	assetsport "mannaiah/module/assets/port"
	"mannaiah/module/auth"
	"mannaiah/module/campaign"
	campaignevent "mannaiah/module/campaign/adapter/event"
	campaignport "mannaiah/module/campaign/port"
	"mannaiah/module/contacts"
	contactevent "mannaiah/module/contacts/adapter/event"
	contactapplication "mannaiah/module/contacts/application"
	contactport "mannaiah/module/contacts/port"
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
	"mannaiah/module/email"
	emailapplication "mannaiah/module/email/application"
	emailport "mannaiah/module/email/port"
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
	"mannaiah/module/products"
	"mannaiah/module/segment"
	segmentapplication "mannaiah/module/segment/application"
	"mannaiah/module/syncrecord"
	syncrecorddomain "mannaiah/module/syncrecord/domain"
	syncrecordport "mannaiah/module/syncrecord/port"
	"mannaiah/module/woocommerce"
	wooevent "mannaiah/module/woocommerce/adapter/event"
	woocommerceport "mannaiah/module/woocommerce/port"

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
	var wooCfg woocommerce.Config
	var analyticsCfg analytics.Config
	var segmentCfg segment.Config
	var emailCfg email.Config
	var campaignCfg campaign.Config
	var membershipCfg membership.Config
	var syncRecordCfg syncrecord.Config
	var telemetryCfg coretelemetry.Config

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
		&wooCfg,
		&analyticsCfg,
		&segmentCfg,
		&emailCfg,
		&campaignCfg,
		&membershipCfg,
		&syncRecordCfg,
		&telemetryCfg,
	); err != nil {
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
		Version:     "2.4.7",
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
	membershipPublisher, err := membershipevent.NewPublisher(messaging.Publisher())
	if err != nil {
		return fmt.Errorf("create membership integration publisher: %w", err)
	}
	campaignPublisher, err := campaignevent.NewPublisher(messaging.Publisher())
	if err != nil {
		return fmt.Errorf("create campaign integration publisher: %w", err)
	}

	authModule, err := auth.New(authCfg, coreCfg.Environment, logger)
	if err != nil {
		return fmt.Errorf("initialize auth module: %w", err)
	}
	if err := authModule.Load(runtime); err != nil {
		return fmt.Errorf("load auth module: %w", err)
	}

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
	if segmentCfg.Enabled && !analyticsCfg.Enabled {
		return errors.New("segment module requires analytics to be enabled")
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
	defer func() {
		_ = analyticsModule.Stop()
	}()

	segmentModule, err := segment.New(segmentCfg, db, analyticsModule.QueryService())
	if err != nil {
		return fmt.Errorf("initialize segment module: %w", err)
	}
	segmentModule.SetAuthorizer(authModule)
	if err := segmentModule.Load(runtime); err != nil {
		return fmt.Errorf("load segment module: %w", err)
	}

	emailModule, err := email.New(emailCfg, db)
	if err != nil {
		return fmt.Errorf("initialize email module: %w", err)
	}
	emailModule.SetMembershipStamper(emailMembershipStamperAdapter{service: membershipModule.Service()})
	emailModule.SetAuthorizer(authModule)
	if err := emailModule.Load(runtime); err != nil {
		return fmt.Errorf("load email module: %w", err)
	}

	campaignModule, err := campaign.New(
		campaignCfg,
		db,
		campaignSegmentResolverAdapter{segments: segmentModule.Service(), contacts: contactsModule.Service()},
		campaignEmailSenderAdapter{service: emailModule.Service()},
		campaignDeliveryReaderAdapter{repository: emailModule.Repository()},
		campaignPublisher,
	)
	if err != nil {
		return fmt.Errorf("initialize campaign module: %w", err)
	}
	campaignModule.SetSyncRecorder(campaignSyncRecorderAdapter{recorder: recorderAdapter})
	campaignModule.SetAuthorizer(authModule)
	if err := campaignModule.Load(runtime); err != nil {
		return fmt.Errorf("load campaign module: %w", err)
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
	wooModule.SetSyncRecorder(wooSyncRecorderAdapter{recorder: recorderAdapter})
	wooModule.SetMembershipStamper(membershipWooStamperAdapter{service: membershipModule.Service()})
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

// wooSyncRecorderAdapter adapts sync recorders for WooCommerce sync recorder ports.
type wooSyncRecorderAdapter struct {
	// recorder defines base sync recorder dependencies.
	recorder syncRecordRecorderAdapter
}

// StartRun starts one synchronization run and returns a run identifier.
func (a wooSyncRecorderAdapter) StartRun(ctx context.Context, kind string, trigger string) (string, error) {
	return a.recorder.StartRun(ctx, kind, trigger)
}

// CompleteRun marks one synchronization run as completed.
func (a wooSyncRecorderAdapter) CompleteRun(ctx context.Context, runID string, processed int, succeeded int, failed int, skipped int) error {
	return a.recorder.CompleteRun(ctx, runID, processed, succeeded, failed, skipped)
}

// FailRun marks one synchronization run as failed.
func (a wooSyncRecorderAdapter) FailRun(ctx context.Context, runID string, processed int, succeeded int, failed int, skipped int, syncErrors []woocommerceport.SyncError) error {
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

// membershipWooStamperAdapter adapts membership stamping behavior for WooCommerce sync flows.
type membershipWooStamperAdapter struct {
	// service defines membership stamper dependencies.
	service membershipport.Stamper
}

// StampByEmail stamps membership state by contact email.
func (a membershipWooStamperAdapter) StampByEmail(ctx context.Context, email string, channel string, action woocommerceport.MembershipAction, source string, occurredAt *time.Time) error {
	if a.service == nil {
		return nil
	}

	_, err := a.service.Stamp(ctx, membershipport.StampCommand{
		Email:      email,
		Channel:    membershipdomain.Channel(channel),
		Action:     membershipdomain.Action(action),
		Source:     source,
		OccurredAt: occurredAt,
	})
	return err
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

// campaignSyncRecorderAdapter adapts sync recorders for campaign sync recorder ports.
type campaignSyncRecorderAdapter struct {
	// recorder defines base sync recorder dependencies.
	recorder syncRecordRecorderAdapter
}

// StartRun starts one synchronization run and returns a run identifier.
func (a campaignSyncRecorderAdapter) StartRun(ctx context.Context, kind string, trigger string) (string, error) {
	return a.recorder.StartRun(ctx, kind, trigger)
}

// CompleteRun marks one synchronization run as completed.
func (a campaignSyncRecorderAdapter) CompleteRun(ctx context.Context, runID string, processed int, succeeded int, failed int, skipped int) error {
	return a.recorder.CompleteRun(ctx, runID, processed, succeeded, failed, skipped)
}

// FailRun marks one synchronization run as failed.
func (a campaignSyncRecorderAdapter) FailRun(ctx context.Context, runID string, processed int, succeeded int, failed int, skipped int, syncErrors []campaignport.SyncError) error {
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

// campaignSegmentResolverAdapter adapts segment and contacts services for campaign resolution.
type campaignSegmentResolverAdapter struct {
	// segments defines segment resolution dependencies.
	segments segmentapplication.Service
	// contacts defines contact lookup dependencies.
	contacts contactapplication.Service
}

// ResolveSegment resolves contact ids for a segment.
func (a campaignSegmentResolverAdapter) ResolveSegment(ctx context.Context, segmentID string, page int, limit int) ([]string, error) {
	if a.segments == nil {
		return nil, errors.New("segment service is not configured")
	}

	resolved, err := a.segments.Resolve(ctx, strings.TrimSpace(segmentID), page, limit)
	if err != nil {
		return nil, err
	}
	if resolved == nil {
		return nil, nil
	}

	return resolved.ContactIDs, nil
}

// ResolveEmails resolves recipient emails by contact ids.
func (a campaignSegmentResolverAdapter) ResolveEmails(ctx context.Context, contactIDs []string) (map[string]string, error) {
	if a.contacts == nil {
		return nil, errors.New("contacts service is not configured")
	}

	result := make(map[string]string, len(contactIDs))
	for _, contactID := range contactIDs {
		contact, err := a.contacts.Get(ctx, strings.TrimSpace(contactID))
		if err != nil || contact == nil {
			continue
		}

		result[strings.TrimSpace(contactID)] = strings.TrimSpace(contact.Email)
	}

	return result, nil
}

// campaignDeliveryReaderAdapter adapts email repository behavior for campaign delivery reads.
type campaignDeliveryReaderAdapter struct {
	// repository defines email delivery read dependencies.
	repository emailport.Repository
}

// ListByCampaignID returns paginated delivery rows for one campaign.
func (a campaignDeliveryReaderAdapter) ListByCampaignID(ctx context.Context, campaignID string, page int, limit int) ([]campaignport.DeliveryRow, int64, error) {
	if a.repository == nil {
		return nil, 0, nil
	}

	rows, total, err := a.repository.ListByCampaignID(ctx, campaignID, page, limit)
	if err != nil {
		return nil, 0, err
	}

	result := make([]campaignport.DeliveryRow, 0, len(rows))
	for _, d := range rows {
		if d == nil {
			continue
		}
		result = append(result, campaignport.DeliveryRow{
			ContactID: d.ContactID,
			Email:     d.Email,
			Status:    string(d.Status),
			CreatedAt: d.CreatedAt,
			UpdatedAt: d.UpdatedAt,
		})
	}

	return result, total, nil
}

// campaignEmailSenderAdapter adapts email service behavior for campaign send flows.
type campaignEmailSenderAdapter struct {
	// service defines email send dependencies.
	service emailapplication.Service
}

// SendCampaignEmail sends one campaign email.
func (a campaignEmailSenderAdapter) SendCampaignEmail(ctx context.Context, contactID string, email string, subject string, htmlBody string, textBody string, idempotencyKey string) error {
	if a.service == nil {
		return errors.New("email service is not configured")
	}

	_, err := a.service.Send(ctx, emailapplication.SendCommand{
		ContactID:      strings.TrimSpace(contactID),
		Email:          strings.TrimSpace(email),
		Subject:        strings.TrimSpace(subject),
		HTMLBody:       htmlBody,
		TextBody:       textBody,
		IdempotencyKey: strings.TrimSpace(idempotencyKey),
	})
	return err
}
