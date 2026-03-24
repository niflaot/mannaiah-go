package service

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/google/uuid"

	markevent "mannaiah/module/shipping/application/mark/event"
	"mannaiah/module/shipping/domain"
	"mannaiah/module/shipping/port"
)

// GenerateCommand defines shipping mark generation input values.
type GenerateCommand struct {
	// OrderID defines order identifier values.
	OrderID string
	// CarrierID defines carrier identifier values.
	CarrierID string
	// Sender defines sender address values.
	Sender domain.Address
	// Recipient defines recipient address values.
	Recipient domain.Address
	// Units defines shipment package units.
	Units []domain.PackageUnit
	// DeclaredValue defines declared shipment value amounts.
	DeclaredValue float64
	// PaymentForm defines payment arrangement values.
	PaymentForm string
	// CollectOnDeliveryAmount defines requested cash-on-delivery collection amounts.
	CollectOnDeliveryAmount float64
	// Observations defines optional observation values.
	Observations string
	// TrackingNumber defines optional manual tracking-number values.
	TrackingNumber string
	// DocumentType defines optional manual document-type values.
	DocumentType domain.MarkDocumentType
	// DocumentRef defines optional manual document reference values.
	DocumentRef string
}

// ListQuery defines mark listing query values.
type ListQuery struct {
	// OrderID filters marks by order identifier.
	OrderID string
	// BatchID filters marks by batch identifier.
	BatchID string
	// Page defines 1-based page values.
	Page int
	// Limit defines page-size values.
	Limit int
}

// Service defines shipping-mark orchestration behavior.
type Service struct {
	// repository defines mark repository dependencies.
	repository port.ShippingMarkRepository
	// registry defines carrier registry dependencies.
	registry port.ProviderRegistry
	// publisher defines integration event publisher dependencies.
	publisher port.IntegrationEventPublisher
}

// NewService creates shipping-mark services.
func NewService(repository port.ShippingMarkRepository, registry port.ProviderRegistry, publisher port.IntegrationEventPublisher) *Service {
	return &Service{repository: repository, registry: registry, publisher: publisher}
}

// Generate creates one shipping mark through the configured carrier provider.
func (s *Service) Generate(ctx context.Context, command GenerateCommand) (*domain.ShippingMark, error) {
	if s == nil || s.repository == nil || s.registry == nil {
		return nil, domain.ErrCarrierNotSupported
	}
	provider, exists := s.registry.CarrierProvider(command.CarrierID)
	if !exists || provider == nil {
		return nil, domain.ErrCarrierNotSupported
	}

	mark := domain.ShippingMark{
		ID:                      uuid.NewString(),
		OrderID:                 strings.TrimSpace(command.OrderID),
		CarrierID:               strings.TrimSpace(command.CarrierID),
		Status:                  domain.MarkStatusPending,
		DocumentType:            command.DocumentType,
		DocumentRef:             strings.TrimSpace(command.DocumentRef),
		TrackingNumber:          strings.TrimSpace(command.TrackingNumber),
		Sender:                  command.Sender,
		Recipient:               command.Recipient,
		Units:                   command.Units,
		DeclaredValue:           command.DeclaredValue,
		PaymentForm:             strings.TrimSpace(command.PaymentForm),
		CollectOnDeliveryAmount: command.CollectOnDeliveryAmount,
		Observations:            strings.TrimSpace(command.Observations),
		CreatedAt:               time.Now().UTC(),
		UpdatedAt:               time.Now().UTC(),
	}.Normalize()
	if err := mark.Validate(); err != nil {
		return nil, err
	}
	if provider.Carrier().RequiresBalanceCheck {
		if err := provider.CheckBalance(ctx); err != nil {
			return nil, domain.ErrInsufficientBalance
		}
	}

	if err := provider.GenerateMark(ctx, &mark); err != nil {
		mark.Status = domain.MarkStatusFailed
		mark.UpdatedAt = time.Now().UTC()
		if createErr := s.repository.Create(ctx, &mark); createErr != nil {
			return nil, createErr
		}
		s.publish(ctx, markevent.BuildMarkFailed(mark, err.Error()))

		return nil, err
	}
	if mark.Status == "" {
		mark.Status = domain.MarkStatusGenerated
	}
	mark.UpdatedAt = time.Now().UTC()
	if err := s.repository.Create(ctx, &mark); err != nil {
		return nil, err
	}
	s.publish(ctx, markevent.BuildMarkGenerated(mark))

	return &mark, nil
}

// Get resolves one shipping mark by identifier.
func (s *Service) Get(ctx context.Context, id string) (*domain.ShippingMark, error) {
	if s == nil || s.repository == nil {
		return nil, domain.ErrNotFound
	}

	return s.repository.GetByID(ctx, strings.TrimSpace(id))
}

// List resolves shipping marks with pagination and optional filters.
func (s *Service) List(ctx context.Context, query ListQuery) ([]domain.ShippingMark, int64, error) {
	if s == nil || s.repository == nil {
		return []domain.ShippingMark{}, 0, nil
	}

	return s.repository.List(ctx, port.MarkListQuery{
		OrderID: strings.TrimSpace(query.OrderID),
		BatchID: strings.TrimSpace(query.BatchID),
		Page:    query.Page,
		Limit:   query.Limit,
	})
}

// Void voids one shipping mark with a local status change only.
// No carrier API call is made; carrier-level void must be handled out-of-band when the carrier supports it.
func (s *Service) Void(ctx context.Context, id string, reason string) (*domain.ShippingMark, error) {
	if s == nil || s.repository == nil {
		return nil, domain.ErrNotFound
	}
	mark, err := s.repository.GetByID(ctx, strings.TrimSpace(id))
	if err != nil {
		return nil, err
	}
	mark.Status = domain.MarkStatusVoided
	mark.UpdatedAt = time.Now().UTC()
	if err := s.repository.Update(ctx, mark); err != nil {
		return nil, err
	}
	s.publish(ctx, markevent.BuildMarkVoided(*mark, strings.TrimSpace(reason)))

	return mark, nil
}

// Materialize submits one QUOTED draft mark to the carrier and updates its status to CREATED or FAILED.
// A JSON snapshot of the mark fields is captured before submission and stored in DraftSnapshot.
func (s *Service) Materialize(ctx context.Context, mark *domain.ShippingMark) error {
	if s == nil || s.repository == nil || s.registry == nil {
		return domain.ErrCarrierNotSupported
	}
	provider, exists := s.registry.CarrierProvider(mark.CarrierID)
	if !exists || provider == nil {
		return domain.ErrCarrierNotSupported
	}
	snapshot, _ := json.Marshal(mark)
	mark.DraftSnapshot = string(snapshot)
	if provider.Carrier().RequiresBalanceCheck {
		if err := provider.CheckBalance(ctx); err != nil {
			mark.Status = domain.MarkStatusFailed
			mark.UpdatedAt = time.Now().UTC()
			_ = s.repository.Update(ctx, mark)
			s.publish(ctx, markevent.BuildMarkFailed(*mark, domain.ErrInsufficientBalance.Error()))

			return domain.ErrInsufficientBalance
		}
	}
	if err := provider.GenerateMark(ctx, mark); err != nil {
		mark.Status = domain.MarkStatusFailed
		mark.UpdatedAt = time.Now().UTC()
		_ = s.repository.Update(ctx, mark)
		s.publish(ctx, markevent.BuildMarkFailed(*mark, err.Error()))

		return err
	}
	mark.Status = domain.MarkStatusCreated
	mark.UpdatedAt = time.Now().UTC()
	if err := s.repository.Update(ctx, mark); err != nil {
		return err
	}
	s.publish(ctx, markevent.BuildMarkGenerated(*mark))

	return nil
}

// publish publishes one integration event and suppresses publication errors.
func (s *Service) publish(ctx context.Context, event port.IntegrationEvent) {
	if s == nil || s.publisher == nil {
		return
	}
	_ = s.publisher.Publish(ctx, event)
}
