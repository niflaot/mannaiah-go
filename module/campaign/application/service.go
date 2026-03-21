package application

import (
	"context"
	"errors"
	"math"
	"sync"

	"mannaiah/module/campaign/application/template"
	"mannaiah/module/campaign/domain"
	"mannaiah/module/campaign/port"
)

var (
	// ErrNilRepository is returned when nil repository dependencies are provided.
	ErrNilRepository = errors.New("campaign repository must not be nil")
)

// CreateCommand defines campaign creation payload values.
type CreateCommand struct {
	// Name defines campaign names.
	Name string
	// Slug defines campaign slugs.
	Slug string
	// Channel defines target channel values.
	Channel string
	// SegmentID defines target segment identifier values.
	SegmentID string
	// Subject defines email subject values.
	Subject string
	// HTMLBody defines html content values.
	HTMLBody string
	// TextBody defines text content values.
	TextBody string
	// TemplateVars defines campaign-level custom variable values.
	TemplateVars map[string]string
	// ProductBlocks defines product recommendation block configurations.
	ProductBlocks []domain.ProductBlock
}

// UpdateCommand defines campaign update payload values.
type UpdateCommand struct {
	// Name defines optional campaign names.
	Name *string
	// Slug defines optional campaign slugs.
	Slug *string
	// Channel defines optional target channel values.
	Channel *string
	// SegmentID defines optional target segment identifier values.
	SegmentID *string
	// Subject defines optional email subject values.
	Subject *string
	// HTMLBody defines optional html content values.
	HTMLBody *string
	// TextBody defines optional text content values.
	TextBody *string
	// TemplateVars defines optional replacement campaign-level custom variable values.
	TemplateVars map[string]string
	// ProductBlocks defines optional replacement product recommendation block configurations.
	ProductBlocks []domain.ProductBlock
}

// ListResult defines paged campaign query output values.
type ListResult struct {
	// Data defines campaign rows in the current page.
	Data []domain.Campaign `json:"data"`
	// Page defines current page number.
	Page int `json:"page"`
	// Limit defines current page size.
	Limit int `json:"limit"`
	// Total defines total matching rows.
	Total int64 `json:"total"`
	// TotalPages defines total available pages.
	TotalPages int `json:"totalPages"`
}

// DeliveryEntry defines a single delivery line within a campaign delivery list result.
type DeliveryEntry struct {
	// ContactID defines recipient contact identifier values.
	ContactID string `json:"contactId"`
	// Email defines recipient email values.
	Email string `json:"email"`
	// Status defines current delivery status values.
	Status string `json:"status"`
	// CreatedAt defines delivery creation timestamps.
	CreatedAt string `json:"createdAt"`
	// UpdatedAt defines delivery last-update timestamps.
	UpdatedAt string `json:"updatedAt"`
}

// DeliveryListResult defines paged campaign delivery query output values.
type DeliveryListResult struct {
	// Data defines delivery rows in the current page.
	Data []DeliveryEntry `json:"data"`
	// Page defines current page number.
	Page int `json:"page"`
	// Limit defines current page size.
	Limit int `json:"limit"`
	// Total defines total matching rows.
	Total int64 `json:"total"`
	// TotalPages defines total available pages.
	TotalPages int `json:"totalPages"`
}

// Service defines campaign use-case behavior.
type Service interface {
	// Create persists campaign rows.
	Create(ctx context.Context, command CreateCommand) (*domain.Campaign, error)
	// Get retrieves one campaign by id.
	Get(ctx context.Context, id string) (*domain.Campaign, error)
	// List retrieves paged campaign rows.
	List(ctx context.Context, page int, limit int) (*ListResult, error)
	// Update persists campaign row updates.
	Update(ctx context.Context, id string, command UpdateCommand) (*domain.Campaign, error)
	// Delete removes one campaign by id.
	Delete(ctx context.Context, id string) error
	// Send starts asynchronous campaign fan-out and returns accepted campaign states.
	Send(ctx context.Context, id string) (*domain.Campaign, error)
	// ListDeliveries retrieves paged delivery rows for one campaign.
	ListDeliveries(ctx context.Context, id string, page int, limit int) (*DeliveryListResult, error)
}

// CampaignService implements campaign use-cases.
type CampaignService struct {
	// repository defines campaign persistence dependencies.
	repository port.Repository
	// resolver defines segment audience resolution dependencies.
	resolver port.SegmentResolver
	// sender defines outbound email sender dependencies.
	sender port.EmailSender
	// deliveryReader defines campaign delivery read dependencies.
	deliveryReader port.DeliveryReader
	// workers defines bounded fan-out worker counts.
	workers int
	// syncRecorder defines optional sync run recording dependencies.
	syncRecorder port.SyncRecorder
	// publisher defines integration event publication dependencies.
	publisher port.IntegrationEventPublisher
	// contactDataProvider defines optional per-contact personalization data dependencies.
	contactDataProvider port.ContactDataProvider
	// affinityProductProvider defines optional affinity product fetch dependencies.
	affinityProductProvider port.AffinityProductProvider
	// templateRenderer renders campaign HTML/text bodies with per-contact context.
	templateRenderer *template.Renderer
	// mutex guards active send guards.
	mutex sync.Mutex
	// activeSends prevents duplicate in-memory sends per campaign.
	activeSends map[string]struct{}
}

// NewService creates campaign services.
func NewService(repository port.Repository, resolver port.SegmentResolver, sender port.EmailSender, workers int, publisher port.IntegrationEventPublisher, readers ...port.DeliveryReader) (*CampaignService, error) {
	if repository == nil {
		return nil, ErrNilRepository
	}
	if workers <= 0 {
		workers = 8
	}

	var reader port.DeliveryReader
	if len(readers) > 0 {
		reader = readers[0]
	}

	return &CampaignService{
		repository:              repository,
		resolver:                resolver,
		sender:                  sender,
		deliveryReader:          reader,
		workers:                 workers,
		syncRecorder:            port.NoopSyncRecorder{},
		publisher:               resolvePublisher(publisher),
		contactDataProvider:     port.NoopContactDataProvider{},
		affinityProductProvider: port.NoopAffinityProductProvider{},
		templateRenderer:        template.NewRenderer(),
		activeSends:             map[string]struct{}{},
	}, nil
}

// SetContactDataProvider configures optional per-contact personalization data dependencies.
func (s *CampaignService) SetContactDataProvider(provider port.ContactDataProvider) {
	if s == nil {
		return
	}
	if provider == nil {
		s.contactDataProvider = port.NoopContactDataProvider{}
		return
	}
	s.contactDataProvider = provider
}

// SetAffinityProductProvider configures optional affinity product fetch dependencies.
func (s *CampaignService) SetAffinityProductProvider(provider port.AffinityProductProvider) {
	if s == nil {
		return
	}
	if provider == nil {
		s.affinityProductProvider = port.NoopAffinityProductProvider{}
		return
	}
	s.affinityProductProvider = provider
}

// SetSyncRecorder configures optional sync run recording dependencies.
func (s *CampaignService) SetSyncRecorder(recorder port.SyncRecorder) {
	if s == nil {
		return
	}
	if recorder == nil {
		s.syncRecorder = port.NoopSyncRecorder{}
		return
	}

	s.syncRecorder = recorder
}

// totalPages computes total page count from total rows and page size.
func totalPages(total int64, limit int) int {
	if total <= 0 || limit <= 0 {
		return 0
	}

	return int(math.Ceil(float64(total) / float64(limit)))
}
