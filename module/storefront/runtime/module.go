package runtime

import (
	"errors"

	"github.com/getkin/kin-openapi/openapi3"
	"gorm.io/gorm"
	corehttp "mannaiah/module/core/http"
	storefronthttp "mannaiah/module/storefront/adapter/http"
	storefrontstore "mannaiah/module/storefront/adapter/store"
	pageservice "mannaiah/module/storefront/application/page/service"
	renderableservice "mannaiah/module/storefront/application/renderable/service"
)

var (
	// ErrNilDB is returned when required database dependencies are nil.
	ErrNilDB = errors.New("storefront module: db must not be nil")
)

// Loader defines bootstrap hooks required by storefront modules.
type Loader interface {
	// RegisterRoutes registers module route handlers.
	RegisterRoutes(register func(router corehttp.Router))
	// AddOpenAPISpec merges module OpenAPI specs.
	AddOpenAPISpec(spec *openapi3.T) error
}

// Module defines storefront composition-root wiring.
type Module struct {
	// renderables defines renderable application-service dependencies.
	renderables *renderableservice.Service
	// pages defines static-page application-service dependencies.
	pages *pageservice.Service
	// handler defines HTTP adapter dependencies.
	handler *storefronthttp.Handler
}

// New creates storefront modules with adapter wiring.
func New(db *gorm.DB) (*Module, error) {
	if db == nil {
		return nil, ErrNilDB
	}

	renderableRepository, err := storefrontstore.NewRenderableRepository(db)
	if err != nil {
		return nil, err
	}
	pageRepository, err := storefrontstore.NewStaticPageRepository(db)
	if err != nil {
		return nil, err
	}

	renderableService, err := renderableservice.NewService(renderableRepository)
	if err != nil {
		return nil, err
	}

	pageService, err := pageservice.NewService(pageRepository, renderableRepository)
	if err != nil {
		return nil, err
	}

	handler, err := storefronthttp.NewHandler(renderableService, pageService)
	if err != nil {
		return nil, err
	}

	return &Module{renderables: renderableService, pages: pageService, handler: handler}, nil
}

// SetAuthorizer configures endpoint authentication dependencies.
func (m *Module) SetAuthorizer(authorizer storefronthttp.Authorizer) {
	if m == nil || m.handler == nil {
		return
	}

	m.handler.SetAuthorizer(authorizer)
}

// RegisterRoutes registers storefront routes on the provided router.
func (m *Module) RegisterRoutes(router corehttp.Router) {
	if m == nil || m.handler == nil {
		return
	}

	m.handler.RegisterRoutes(router)
}

// RenderableService returns storefront renderable application services.
func (m *Module) RenderableService() *renderableservice.Service {
	if m == nil {
		return nil
	}

	return m.renderables
}

// PageService returns storefront static-page application services.
func (m *Module) PageService() *pageservice.Service {
	if m == nil {
		return nil
	}

	return m.pages
}

// OpenAPISpec returns storefront-module OpenAPI documentation.
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
