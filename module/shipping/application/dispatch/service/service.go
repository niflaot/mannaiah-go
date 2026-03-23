package service

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"

	dispatchevent "mannaiah/module/shipping/application/dispatch/event"
	"mannaiah/module/shipping/domain"
	"mannaiah/module/shipping/port"
)

// CreateBatchCommand defines dispatch batch creation input values.
type CreateBatchCommand struct {
	// Name defines batch display-name values.
	Name string
	// CarrierID defines batch carrier identifier values.
	CarrierID string
}

// AddMarksCommand defines mark assignment input values.
type AddMarksCommand struct {
	// BatchID defines batch identifier values.
	BatchID string
	// MarkIDs defines mark identifier values.
	MarkIDs []string
}

// ListQuery defines dispatch batch listing query values.
type ListQuery struct {
	// CarrierID filters rows by carrier identifier.
	CarrierID string
	// Status filters rows by batch status.
	Status domain.BatchStatus
	// Page defines 1-based page values.
	Page int
	// Limit defines page-size values.
	Limit int
}

// Service defines dispatch batch orchestration behavior.
type Service struct {
	// batchRepository defines dispatch batch persistence dependencies.
	batchRepository port.DispatchBatchRepository
	// markRepository defines mark persistence dependencies.
	markRepository port.ShippingMarkRepository
	// publisher defines integration event publisher dependencies.
	publisher port.IntegrationEventPublisher
}

// NewService creates dispatch batch services.
func NewService(batchRepository port.DispatchBatchRepository, markRepository port.ShippingMarkRepository, publisher port.IntegrationEventPublisher) *Service {
	return &Service{batchRepository: batchRepository, markRepository: markRepository, publisher: publisher}
}

// Create creates one dispatch batch.
func (s *Service) Create(ctx context.Context, command CreateBatchCommand) (*domain.DispatchBatch, error) {
	if s == nil || s.batchRepository == nil {
		return nil, domain.ErrInvalidID
	}
	batch := domain.DispatchBatch{
		ID:        uuid.NewString(),
		Name:      strings.TrimSpace(command.Name),
		CarrierID: strings.TrimSpace(command.CarrierID),
		Status:    domain.BatchStatusOpen,
		CreatedAt: time.Now().UTC(),
	}.Normalize()
	if err := batch.Validate(); err != nil {
		return nil, err
	}
	if err := s.batchRepository.Create(ctx, &batch); err != nil {
		return nil, err
	}
	s.publish(ctx, dispatchevent.BuildBatchCreated(batch))

	return &batch, nil
}

// Get resolves one dispatch batch by identifier.
func (s *Service) Get(ctx context.Context, id string) (*domain.DispatchBatch, error) {
	if s == nil || s.batchRepository == nil {
		return nil, domain.ErrNotFound
	}

	return s.batchRepository.GetByID(ctx, strings.TrimSpace(id))
}

// List resolves dispatch batches with pagination and filters.
func (s *Service) List(ctx context.Context, query ListQuery) ([]domain.DispatchBatch, int64, error) {
	if s == nil || s.batchRepository == nil {
		return []domain.DispatchBatch{}, 0, nil
	}

	return s.batchRepository.List(ctx, port.BatchListQuery{
		CarrierID: strings.TrimSpace(query.CarrierID),
		Status:    query.Status,
		Page:      query.Page,
		Limit:     query.Limit,
	})
}

// AddMarks assigns one or more marks to one open dispatch batch.
func (s *Service) AddMarks(ctx context.Context, command AddMarksCommand) (*domain.DispatchBatch, error) {
	if s == nil || s.batchRepository == nil || s.markRepository == nil {
		return nil, domain.ErrInvalidID
	}
	batch, err := s.batchRepository.GetByID(ctx, strings.TrimSpace(command.BatchID))
	if err != nil {
		return nil, err
	}
	if batch.Status != domain.BatchStatusOpen {
		return nil, domain.ErrBatchClosed
	}
	for _, markID := range command.MarkIDs {
		trimmedMarkID := strings.TrimSpace(markID)
		if trimmedMarkID == "" {
			continue
		}
		mark, markErr := s.markRepository.GetByID(ctx, trimmedMarkID)
		if markErr != nil {
			return nil, markErr
		}
		if mark.Status != domain.MarkStatusGenerated {
			return nil, domain.ErrBatchMarkStatusMismatch
		}
		if !strings.EqualFold(strings.TrimSpace(mark.CarrierID), strings.TrimSpace(batch.CarrierID)) {
			return nil, domain.ErrBatchCarrierMismatch
		}
		if err := s.batchRepository.AddMark(ctx, batch.ID, mark.ID); err != nil {
			return nil, err
		}
	}

	return s.batchRepository.GetByID(ctx, batch.ID)
}

// RemoveMark removes one mark from one open dispatch batch.
func (s *Service) RemoveMark(ctx context.Context, batchID string, markID string) (*domain.DispatchBatch, error) {
	if s == nil || s.batchRepository == nil {
		return nil, domain.ErrInvalidID
	}
	if err := s.batchRepository.RemoveMark(ctx, strings.TrimSpace(batchID), strings.TrimSpace(markID)); err != nil {
		return nil, err
	}

	return s.batchRepository.GetByID(ctx, strings.TrimSpace(batchID))
}

// Close closes one dispatch batch and emits a batch-closed event.
func (s *Service) Close(ctx context.Context, batchID string) (*domain.DispatchBatch, error) {
	if s == nil || s.batchRepository == nil {
		return nil, domain.ErrInvalidID
	}
	if err := s.batchRepository.Close(ctx, strings.TrimSpace(batchID)); err != nil {
		return nil, err
	}
	batch, err := s.batchRepository.GetByID(ctx, strings.TrimSpace(batchID))
	if err != nil {
		return nil, err
	}
	s.publish(ctx, dispatchevent.BuildBatchClosed(*batch))

	return batch, nil
}

// publish publishes one integration event and suppresses publication errors.
func (s *Service) publish(ctx context.Context, event port.IntegrationEvent) {
	if s == nil || s.publisher == nil {
		return
	}
	_ = s.publisher.Publish(ctx, event)
}
