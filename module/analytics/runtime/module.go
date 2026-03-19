package runtime

import (
	"context"
	"fmt"
	"time"

	"github.com/getkin/kin-openapi/openapi3"
	"gorm.io/gorm"
	"mannaiah/module/analytics/adapter/clickhouse"
	analyticshttp "mannaiah/module/analytics/adapter/http"
	analyticsstore "mannaiah/module/analytics/adapter/store"
	"mannaiah/module/analytics/application"
	affinityapp "mannaiah/module/analytics/application/affinity"
	rfmapp "mannaiah/module/analytics/application/rfm"
	"mannaiah/module/analytics/port"
	corehttp "mannaiah/module/core/http"
	"mannaiah/module/core/messaging/bus"
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
	// rfmService defines RFM use-case dependencies.
	rfmService *rfmapp.RFMService
	// affinityService defines affinity use-case dependencies.
	affinityService *affinityapp.AffinityService
	// handler defines HTTP route adapter dependencies.
	handler *analyticshttp.Handler
	// rfmHandler defines RFM HTTP route adapter dependencies.
	rfmHandler *analyticshttp.RFMHandler
	// affinityHandler defines affinity HTTP route adapter dependencies.
	affinityHandler *analyticshttp.AffinityHandler
	// clickhouseClient defines optional clickhouse dependencies.
	clickhouseClient *clickhouse.Client
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

	if err := module.wireRFM(db); err != nil {
		return nil, err
	}
	if err := module.wireAffinity(store); err != nil {
		return nil, err
	}

	if err := module.registerIntegrationHandlers(registrar); err != nil {
		return nil, err
	}

	return module, nil
}

// wireRFM wires RFM use-cases and HTTP handler.
func (m *Module) wireRFM(db *gorm.DB) error {
	groupRepo, err := analyticsstore.NewRFMGroupRepository(db)
	if err != nil {
		return fmt.Errorf("create rfm group repository: %w", err)
	}

	if m.cfg.Enabled && m.cfg.MigrationEnabled {
		if seedErr := groupRepo.SeedDefaultBands(context.Background()); seedErr != nil {
			return fmt.Errorf("seed default rfm bands: %w", seedErr)
		}
	}

	var rfmStore port.RFMStore
	if m.clickhouseClient != nil {
		rfmStore = clickhouse.NewStoreAdapter(m.clickhouseClient)
	} else {
		rfmStore = &noopRFMStore{}
	}

	rfmSvc, err := rfmapp.NewService(rfmStore, groupRepo)
	if err != nil {
		return fmt.Errorf("create rfm service: %w", err)
	}
	m.rfmService = rfmSvc
	m.rfmHandler = analyticshttp.NewRFMHandler(rfmSvc)

	return nil
}

// wireAffinity wires affinity use-cases and HTTP handler.
func (m *Module) wireAffinity(store port.Store) error {
	var affinityStore port.AffinityStore
	if m.clickhouseClient != nil {
		affinityStore = clickhouse.NewStoreAdapter(m.clickhouseClient)
	} else {
		affinityStore = &noopAffinityStore{}
	}

	affinitySvc, err := affinityapp.NewService(affinityStore)
	if err != nil {
		return fmt.Errorf("create affinity service: %w", err)
	}
	m.affinityService = affinitySvc
	m.affinityHandler = analyticshttp.NewAffinityHandler(affinitySvc)

	return nil
}

// RegisterRoutes registers analytics routes on the provided router.
func (m *Module) RegisterRoutes(router corehttp.Router) {
	if m == nil {
		return
	}
	if m.handler != nil {
		m.handler.RegisterRoutes(router)
	}
	if m.rfmHandler != nil {
		m.rfmHandler.RegisterRoutes(router)
	}
	if m.affinityHandler != nil {
		m.affinityHandler.RegisterRoutes(router)
	}
}

// SetAuthorizer configures endpoint authentication and permission dependencies.
func (m *Module) SetAuthorizer(authorizer analyticshttp.Authorizer) {
	if m == nil {
		return
	}
	if m.handler != nil {
		m.handler.SetAuthorizer(authorizer)
	}
	if m.rfmHandler != nil {
		m.rfmHandler.SetAuthorizer(authorizer)
	}
	if m.affinityHandler != nil {
		m.affinityHandler.SetAuthorizer(authorizer)
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
	if m.affinityService != nil {
		m.affinityService.SetSyncRecorder(recorder)
	}
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

// Stop closes analytics backend connections.
func (m *Module) Stop() error {
	if m == nil || m.clickhouseClient == nil {
		return nil
	}

	return m.clickhouseClient.Close()
}
