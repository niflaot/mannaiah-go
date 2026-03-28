package runtime

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/getkin/kin-openapi/openapi3"
	"go.uber.org/zap"
	"gorm.io/gorm"

	corehttp "mannaiah/module/core/http"
	shippingcarrier "mannaiah/module/shipping/adapter/carrier"
	"mannaiah/module/shipping/adapter/carrier/manual"
	"mannaiah/module/shipping/adapter/carrier/tcc"
	shippinghttp "mannaiah/module/shipping/adapter/http"
	shippingstore "mannaiah/module/shipping/adapter/store"
	dispatchservice "mannaiah/module/shipping/application/dispatch/service"
	markservice "mannaiah/module/shipping/application/mark/service"
	quotationservice "mannaiah/module/shipping/application/quotation/service"
	trackingservice "mannaiah/module/shipping/application/tracking/service"
	"mannaiah/module/shipping/domain"
	"mannaiah/module/shipping/port"
)

// Loader defines bootstrap hooks required by shipping modules.
type Loader interface {
	// RegisterRoutes registers module route handlers.
	RegisterRoutes(register func(router corehttp.Router))
	// AddOpenAPISpec merges module OpenAPI specs.
	AddOpenAPISpec(spec *openapi3.T) error
}

// Module defines composition-root wiring for shipping endpoints.
type Module struct {
	// cfg defines runtime configuration values.
	cfg Config
	// handler defines HTTP route adapter dependencies.
	handler *shippinghttp.Handler
	// quotationService defines quotation orchestration dependencies.
	quotationService *quotationservice.Service
	// markService defines mark orchestration dependencies.
	markService *markservice.Service
	// dispatchService defines batch orchestration dependencies.
	dispatchService *dispatchservice.Service
	// trackingService defines tracking orchestration dependencies.
	trackingService *trackingservice.Service
	// carrierService defines carrier listing dependencies.
	carrierService *carrierService
}

// CarrierService defines carrier listing/lookup behavior exposed for integration consumers.
type CarrierService interface {
	// List returns configured carriers.
	List(ctx context.Context) ([]domain.Carrier, error)
	// Get returns one configured carrier by identifier.
	Get(ctx context.Context, id string) (*domain.Carrier, error)
}

// carrierService defines carrier listing behavior.
type carrierService struct {
	// registry defines provider registry dependencies.
	registry port.ProviderRegistry
}

// New creates shipping modules with adapter wiring.
func New(cfg Config, db *gorm.DB, publishers ...port.IntegrationEventPublisher) (*Module, error) {
	markRepository, batchRepository, quotationRepository, err := shippingstore.NewRepositories(db)
	if err != nil {
		return nil, err
	}

	providers := make([]port.CarrierProvider, 0)
	trackingProviders := make([]port.TrackingProvider, 0)
	manualProvider := manual.NewProvider()
	providers = append(providers, manualProvider)
	trackingProviders = append(trackingProviders, manualProvider)

	if cfg.TCC.Enabled {
		tccAccessToken := resolveTCCAccessToken(cfg.TCC)
		tccProvider, providerErr := tcc.NewProvider(tcc.ProviderConfig{
			Enabled:              cfg.TCC.Enabled,
			IsSandbox:            cfg.TCC.Sandbox,
			AccessToken:          tccAccessToken,
			ParcelAccountNumber:  strings.TrimSpace(cfg.TCC.ParcelAccountNumber),
			ExpressAccountNumber: strings.TrimSpace(cfg.TCC.ExpressAccountNumber),
			Declaration:          strings.TrimSpace(cfg.TCC.Declaration),
			PaymentForm:          cfg.TCC.PaymentForm,
			CODFeePercent:        cfg.TCC.CODFeePercent,
			Sender: domain.Address{
				Name:        strings.TrimSpace(cfg.DefaultSender.Name),
				ID:          strings.TrimSpace(cfg.DefaultSender.ID),
				IDType:      strings.TrimSpace(cfg.DefaultSender.IDType),
				AddressLine: strings.TrimSpace(cfg.DefaultSender.Address),
				CityCode:    strings.TrimSpace(cfg.DefaultSender.CityCode),
				Phone:       strings.TrimSpace(cfg.DefaultSender.Phone),
				Email:       strings.TrimSpace(cfg.DefaultSender.Email),
			},
			RequestTimeout: time.Duration(cfg.TCC.RequestTimeoutMS) * time.Millisecond,
		})
		if providerErr != nil {
			return nil, fmt.Errorf("create tcc provider: %w", providerErr)
		}
		providers = append(providers, tccProvider)
		trackingProviders = append(trackingProviders, tccProvider)
	}

	registry := shippingcarrier.NewRegistry(providers, trackingProviders)
	publisher := resolvePublisher(publishers)
	quotationSvc := quotationservice.NewService(quotationRepository, registry, quotationservice.Config{
		ExpirationTTLMinutes: cfg.Quotation.ExpirationTTLMinutes,
	})
	markSvc := markservice.NewService(markRepository, registry, publisher)
	dispatchSvc := dispatchservice.NewService(batchRepository, markRepository, publisher, markSvc)
	dispatchSvc.SetQuotationRepository(quotationRepository)
	dispatchSvc.SetBatchManifestDocumentCacheTTL(time.Duration(cfg.BatchManifestCacheTTLSeconds) * time.Second)
	if err := dispatchSvc.SetBatchManifestDocumentCoverTemplateFromFile(cfg.BatchManifestTemplatePath); err != nil {
		return nil, fmt.Errorf("configure batch manifest cover template: %w", err)
	}
	trackingSvc := trackingservice.NewService(registry, publisher)
	carrierSvc := &carrierService{registry: registry}

	handler, err := shippinghttp.NewHandler(quotationSvc, markSvc, dispatchSvc, trackingSvc, carrierSvc)
	if err != nil {
		return nil, err
	}

	return &Module{
		cfg:              cfg,
		handler:          handler,
		quotationService: quotationSvc,
		markService:      markSvc,
		dispatchService:  dispatchSvc,
		trackingService:  trackingSvc,
		carrierService:   carrierSvc,
	}, nil
}

// RegisterRoutes registers shipping routes on the provided router.
func (m *Module) RegisterRoutes(router corehttp.Router) {
	if m == nil || m.handler == nil || !m.cfg.Enabled {
		return
	}

	m.handler.RegisterRoutes(router)
}

// SetAuthorizer configures endpoint authentication and permission dependencies.
func (m *Module) SetAuthorizer(authorizer shippinghttp.Authorizer) {
	if m == nil || m.handler == nil {
		return
	}

	m.handler.SetAuthorizer(authorizer)
}

// QuotationService returns quotation service dependencies.
func (m *Module) QuotationService() *quotationservice.Service {
	if m == nil {
		return nil
	}

	return m.quotationService
}

// SetQuotationOrderSource configures the order data source for order-based quotation workflows.
func (m *Module) SetQuotationOrderSource(source port.OrderQuotationSource) {
	if m == nil || m.quotationService == nil {
		return
	}

	m.quotationService.SetOrderSource(source)
	if m.dispatchService != nil {
		m.dispatchService.SetOrderSource(source)
	}
}

// SetQuotationProductSource configures the product shipping attribute source for box-packing.
func (m *Module) SetQuotationProductSource(source port.OrderProductSource) {
	if m == nil || m.quotationService == nil {
		return
	}

	m.quotationService.SetProductSource(source)
}

// MarkService returns mark service dependencies.
func (m *Module) MarkService() *markservice.Service {
	if m == nil {
		return nil
	}

	return m.markService
}

// DispatchService returns dispatch service dependencies.
func (m *Module) DispatchService() *dispatchservice.Service {
	if m == nil {
		return nil
	}

	return m.dispatchService
}

// TrackingService returns tracking service dependencies.
func (m *Module) TrackingService() *trackingservice.Service {
	if m == nil {
		return nil
	}

	return m.trackingService
}

// CarrierService returns carrier listing dependencies.
func (m *Module) CarrierService() CarrierService {
	if m == nil {
		return nil
	}

	return m.carrierService
}

// OpenAPISpec returns shipping-module OpenAPI documentation.
func (m *Module) OpenAPISpec() *openapi3.T {
	return OpenAPISpec()
}

// Load mounts all module routes/specs into the provided startup loader.
func (m *Module) Load(loader Loader) error {
	if m == nil || loader == nil {
		return nil
	}
	if m.cfg.Enabled {
		loader.RegisterRoutes(m.RegisterRoutes)
	}
	if err := loader.AddOpenAPISpec(m.OpenAPISpec()); err != nil {
		return err
	}

	return nil
}

// Start begins background maintenance tasks for the shipping module.
// The cleanup goroutine terminates when ctx is cancelled.
func (m *Module) Start(ctx context.Context) error {
	if m == nil || m.quotationService == nil {
		return nil
	}
	go func() {
		ticker := time.NewTicker(time.Hour)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				deleted, err := m.quotationService.PurgeExpired(ctx)
				if err != nil {
					zap.L().Error("quotation purge failed", zap.Error(err))
				} else if deleted > 0 {
					zap.L().Info("purged expired quotations", zap.Int64("count", deleted))
				}
			}
		}
	}()

	return nil
}

// List returns configured carriers.
func (s *carrierService) List(ctx context.Context) ([]domain.Carrier, error) {
	if s == nil || s.registry == nil {
		return []domain.Carrier{}, nil
	}
	rows := s.registry.Carriers()
	sort.SliceStable(rows, func(i int, j int) bool {
		return strings.ToLower(rows[i].ID) < strings.ToLower(rows[j].ID)
	})

	return rows, nil
}

// Get returns one configured carrier by identifier.
func (s *carrierService) Get(ctx context.Context, id string) (*domain.Carrier, error) {
	if s == nil || s.registry == nil {
		return nil, domain.ErrCarrierNotSupported
	}
	trimmedID := strings.TrimSpace(id)
	for _, carrier := range s.registry.Carriers() {
		if strings.EqualFold(strings.TrimSpace(carrier.ID), trimmedID) {
			copy := carrier

			return &copy, nil
		}
	}

	return nil, domain.ErrNotFound
}

// resolvePublisher resolves optional integration event publisher dependencies.
func resolvePublisher(publishers []port.IntegrationEventPublisher) port.IntegrationEventPublisher {
	if len(publishers) == 0 {
		return nil
	}

	return publishers[0]
}

// resolveTCCAccessToken resolves TCC access tokens according to configured runtime mode.
func resolveTCCAccessToken(config TCCConfig) string {
	if config.Sandbox {
		return strings.TrimSpace(config.SandboxAccessToken)
	}

	return strings.TrimSpace(config.ProductionAccessToken)
}
