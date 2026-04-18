package service

import (
	"context"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	"mannaiah/module/shipping/domain"
	"mannaiah/module/shipping/port"
)

// Service defines tracking orchestration behavior.
type Service struct {
	// repository defines shipping mark lookup/list dependencies.
	repository port.ShippingMarkRepository
	// registry defines tracking provider lookup dependencies.
	registry port.ProviderRegistry
	// publisher defines integration event publisher dependencies.
	publisher port.IntegrationEventPublisher
	// trackingHistoryCache guards short-lived tracking lookups reused across list filter changes.
	trackingHistoryCache map[string]trackingHistoryCacheEntry
	// trackingHistoryCacheMu guards trackingHistoryCache access.
	trackingHistoryCacheMu sync.RWMutex
}

// TrackingUpdatedPayload defines tracking-updated event payload values.
type TrackingUpdatedPayload struct {
	// TrackingNumber defines tracking-number values.
	TrackingNumber string `json:"trackingNumber"`
	// CarrierID defines carrier identifier values.
	CarrierID string `json:"carrierId"`
	// GlobalStatus defines normalized status values.
	GlobalStatus domain.TrackingStatus `json:"globalStatus"`
	// LatestEvent defines latest history event values.
	LatestEvent *domain.TrackingEvent `json:"latestEvent,omitempty"`
}

// NewService creates tracking services.
func NewService(repository port.ShippingMarkRepository, registry port.ProviderRegistry, publisher port.IntegrationEventPublisher) *Service {
	return &Service{
		repository:           repository,
		registry:             registry,
		publisher:            publisher,
		trackingHistoryCache: map[string]trackingHistoryCacheEntry{},
	}
}

// Get resolves normalized tracking history for one tracking number and carrier identifier.
func (s *Service) Get(ctx context.Context, carrierID string, trackingNumber string) (*domain.TrackingHistory, error) {
	if s == nil || s.registry == nil {
		return nil, domain.ErrTrackingNotSupported
	}
	provider, exists := s.registry.TrackingProvider(strings.TrimSpace(carrierID))
	if !exists || provider == nil {
		return nil, domain.ErrTrackingNotSupported
	}
	resolved, err := provider.GetTrackingHistory(ctx, strings.TrimSpace(trackingNumber))
	if err != nil {
		return nil, err
	}
	if resolved == nil {
		return nil, domain.ErrTrackingNotSupported
	}

	s.publish(ctx, *resolved)

	return resolved, nil
}

// publish publishes tracking-updated integration events and suppresses publication errors.
func (s *Service) publish(ctx context.Context, history domain.TrackingHistory) {
	if s == nil || s.publisher == nil {
		return
	}
	var latest *domain.TrackingEvent
	if len(history.History) > 0 {
		last := history.History[len(history.History)-1]
		latest = &last
	}
	_ = s.publisher.Publish(ctx, port.IntegrationEvent{
		ID:            uuid.NewString(),
		Topic:         port.TopicTrackingUpdated,
		SchemaVersion: "v1",
		OccurredAt:    time.Now().UTC(),
		Payload: TrackingUpdatedPayload{
			TrackingNumber: history.TrackingNumber,
			CarrierID:      history.CarrierID,
			GlobalStatus:   history.GlobalStatus,
			LatestEvent:    latest,
		},
	})
}
