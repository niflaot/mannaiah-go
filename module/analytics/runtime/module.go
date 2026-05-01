package runtime

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/getkin/kin-openapi/openapi3"
	"gorm.io/gorm"
	"mannaiah/module/analytics/adapter/clickhouse"
	analyticshttp "mannaiah/module/analytics/adapter/http"
	analyticsstore "mannaiah/module/analytics/adapter/store"
	"mannaiah/module/analytics/application"
	recommendationapp "mannaiah/module/analytics/application/recommendation"
	"mannaiah/module/analytics/port"
	corecron "mannaiah/module/core/cron"
	corehttp "mannaiah/module/core/http"
	"mannaiah/module/core/messaging/bus"
)

var (
	// ErrModuleNotInitialized is returned when module lifecycle methods are called on nil receivers.
	ErrModuleNotInitialized = errors.New("analytics module is not initialized")
)

// Loader defines bootstrap hooks required by analytics modules.
type Loader interface {
	// RegisterRoutes registers module route handlers.
	RegisterRoutes(register func(router corehttp.Router))
	// AddOpenAPISpec merges module OpenAPI specs.
	AddOpenAPISpec(spec *openapi3.T) error
}

// Module defines composition-root wiring for analytics endpoints.
type Module struct {
	// cfg defines runtime configuration values.
	cfg Config
	// service defines analytics use-case dependencies.
	service *application.AnalyticsService
	// handler defines HTTP route adapter dependencies.
	handler *analyticshttp.Handler
	// recommendationService defines recommendation use-case dependencies.
	recommendationService *recommendationapp.RecommendationService
	// recommendationHandler defines recommendation HTTP route adapter dependencies.
	recommendationHandler *analyticshttp.RecommendationHandler
	// clickhouseClient defines optional clickhouse dependencies.
	clickhouseClient *clickhouse.Client
	// scheduler defines optional cron scheduler dependencies.
	scheduler corecron.Scheduler
	// affinityRefreshEntryID defines optional scheduled affinity-refresh entry identifiers.
	affinityRefreshEntryID corecron.EntryID
	// mutex guards scheduler lifecycle state.
	mutex sync.Mutex
	// started reports whether scheduler lifecycle start logic has completed.
	started bool
}

// New creates analytics modules with adapter wiring.
func New(cfg Config, db *gorm.DB, registrar bus.Registrar) (*Module, error) {
	var (
		store            port.Store
		clickhouseClient *clickhouse.Client
	)
	if cfg.Enabled {
		client, err := clickhouse.NewClient(clickhouse.Config{
			DSN:             cfg.ClickHouseDSN,
			MaxOpenConns:    cfg.MaxOpenConns,
			MaxIdleConns:    cfg.MaxIdleConns,
			ConnMaxLifetime: time.Duration(cfg.ConnMaxLifetimeMS) * time.Millisecond,
		})
		if err != nil {
			return nil, fmt.Errorf("create clickhouse client: %w", err)
		}
		clickhouseClient = client
		store = clickhouse.NewStoreAdapter(client)
	}

	service, err := application.NewService(cfg.Enabled, db, store)
	if err != nil {
		return nil, err
	}
	if cfg.Enabled && cfg.MigrationEnabled && store != nil {
		if err := store.EnsureSchema(context.Background()); err != nil {
			return nil, fmt.Errorf("ensure analytics clickhouse schema: %w", err)
		}
	}

	if cfg.Enabled && store != nil {
		storeAdapter := store.(*clickhouse.StoreAdapter)
		service.SetTaxonomyStore(storeAdapter)
	}

	handler, err := analyticshttp.NewHandler(service)
	if err != nil {
		return nil, err
	}

	module := &Module{cfg: cfg, service: service, handler: handler, clickhouseClient: clickhouseClient}

	if err := module.wireRecommendation(db); err != nil {
		return nil, err
	}

	if err := module.registerIntegrationHandlers(registrar); err != nil {
		return nil, err
	}

	return module, nil
}

// wireRecommendation wires recommendation use-cases and HTTP handler.
func (m *Module) wireRecommendation(db *gorm.DB) error {
	var affinityStore port.AffinityStore
	if m.clickhouseClient != nil {
		affinityStore = clickhouse.NewStoreAdapter(m.clickhouseClient)
	} else {
		affinityStore = &noopAffinityStore{}
	}

	var correlationStore port.TagCorrelationStore
	var catalogStore port.ProductCatalogStore
	if db != nil {
		corrRepo, err := analyticsstore.NewTagCorrelationRepository(db)
		if err != nil {
			return err
		}
		catalogRepo, err := analyticsstore.NewProductCatalogRepository(db)
		if err != nil {
			return err
		}
		correlationStore = corrRepo
		catalogStore = catalogRepo
	} else {
		correlationStore = &noopTagCorrelationStore{}
		catalogStore = &noopProductCatalogStore{}
	}

	svc, err := recommendationapp.NewService(affinityStore, correlationStore, catalogStore)
	if err != nil {
		return err
	}
	m.recommendationService = svc
	m.recommendationHandler = analyticshttp.NewRecommendationHandler(svc)

	return nil
}

// RegisterRoutes registers analytics routes on the provided router.
func (m *Module) RegisterRoutes(router corehttp.Router) {
	_ = m
	_ = router
}

// SetAuthorizer configures endpoint authentication and permission dependencies.
func (m *Module) SetAuthorizer(authorizer analyticshttp.Authorizer) {
	if m == nil {
		return
	}
	if m.handler != nil {
		m.handler.SetAuthorizer(authorizer)
	}
	if m.recommendationHandler != nil {
		m.recommendationHandler.SetAuthorizer(authorizer)
	}
}

// SetSyncRecorder configures optional sync run recording dependencies.
func (m *Module) SetSyncRecorder(recorder port.SyncRecorder) {
	if m == nil {
		return
	}
	if m.service != nil {
		m.service.SetSyncRecorder(recorder)
	}
}

// RecommendationService returns the recommendation service for use by other modules.
func (m *Module) RecommendationService() *recommendationapp.RecommendationService {
	if m == nil {
		return nil
	}

	return m.recommendationService
}

// QueryService returns analytics query resolver dependencies.
func (m *Module) QueryService() port.Resolver {
	if m == nil {
		return nil
	}

	return m.service
}

// OpenAPISpec returns analytics-module OpenAPI documentation.
func (m *Module) OpenAPISpec() *openapi3.T {
	return OpenAPISpec()
}

// Load mounts all module routes/specs into the provided startup loader.
func (m *Module) Load(loader Loader) error {
	if m == nil || loader == nil {
		return nil
	}

	loader.RegisterRoutes(m.RegisterRoutes)
	if err := loader.AddOpenAPISpec(m.OpenAPISpec()); err != nil {
		return err
	}

	return nil
}

// Stop closes analytics backend connections and scheduled jobs.
func (m *Module) Stop() error {
	if m == nil {
		return nil
	}
	m.mutex.Lock()
	started := m.started
	m.started = false
	entryID := m.affinityRefreshEntryID
	m.affinityRefreshEntryID = 0
	scheduler := m.scheduler
	m.mutex.Unlock()

	var stopErr error
	if started && scheduler != nil {
		if entryID != 0 {
			scheduler.Remove(entryID)
		}
		stopCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := scheduler.Stop(stopCtx); err != nil {
			stopErr = err
		}
	}
	if m.clickhouseClient != nil {
		if err := m.clickhouseClient.Close(); err != nil && stopErr == nil {
			stopErr = err
		}
	}

	return stopErr
}

// ConfigureScheduler configures the cron scheduler for periodic affinity refresh behavior.
func (m *Module) ConfigureScheduler(scheduler corecron.Scheduler) {
	if m == nil {
		return
	}

	m.scheduler = scheduler
}

// Start marks analytics runtime startup as complete.
func (m *Module) Start(_ context.Context) error {
	if m == nil {
		return ErrModuleNotInitialized
	}

	m.mutex.Lock()
	defer m.mutex.Unlock()

	if m.started {
		return nil
	}
	m.started = true
	return nil
}
