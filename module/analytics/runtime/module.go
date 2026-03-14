package runtime

import (
	"context"
	"fmt"
	"time"

	"github.com/getkin/kin-openapi/openapi3"
	"gorm.io/gorm"
	"mannaiah/module/analytics/adapter/clickhouse"
	analyticshttp "mannaiah/module/analytics/adapter/http"
	"mannaiah/module/analytics/application"
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
	// handler defines HTTP route adapter dependencies.
	handler *analyticshttp.Handler
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

	handler, err := analyticshttp.NewHandler(service)
	if err != nil {
		return nil, err
	}

	module := &Module{cfg: cfg, service: service, handler: handler, clickhouseClient: clickhouseClient}
	if err := module.registerIntegrationHandlers(registrar); err != nil {
		return nil, err
	}

	return module, nil
}

// RegisterRoutes registers analytics routes on the provided router.
func (m *Module) RegisterRoutes(router corehttp.Router) {
	if m == nil || m.handler == nil {
		return
	}

	m.handler.RegisterRoutes(router)
}

// SetAuthorizer configures endpoint authentication and permission dependencies.
func (m *Module) SetAuthorizer(authorizer analyticshttp.Authorizer) {
	if m == nil || m.handler == nil {
		return
	}

	m.handler.SetAuthorizer(authorizer)
}

// SetSyncRecorder configures optional sync run recording dependencies.
func (m *Module) SetSyncRecorder(recorder port.SyncRecorder) {
	if m == nil || m.service == nil {
		return
	}

	m.service.SetSyncRecorder(recorder)
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
