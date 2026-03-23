package service

import (
	"context"
	"testing"
	"time"

	"mannaiah/module/shipping/domain"
	"mannaiah/module/shipping/port"
)

type trackingProviderStub struct{}

func (trackingProviderStub) SupportsCourier(carrierID string) bool { return true }
func (trackingProviderStub) GetTrackingHistory(ctx context.Context, trackingNumber string) (*domain.TrackingHistory, error) {
	return &domain.TrackingHistory{CarrierID: "manual", TrackingNumber: trackingNumber, GlobalStatus: domain.TrackingStatusProcessing, LastUpdate: time.Now().UTC(), History: []domain.TrackingEvent{{Date: time.Now().UTC(), Text: "ok", Status: domain.TrackingStatusProcessing}}}, nil
}

type trackingRegistryStub struct{}

func (trackingRegistryStub) CarrierProvider(carrierID string) (port.CarrierProvider, bool) {
	return nil, false
}
func (trackingRegistryStub) TrackingProvider(carrierID string) (port.TrackingProvider, bool) {
	return trackingProviderStub{}, true
}
func (trackingRegistryStub) Carriers() []domain.Carrier {
	return nil
}

type trackingPublisherStub struct {
	count int
}

func (s *trackingPublisherStub) Publish(ctx context.Context, event port.IntegrationEvent) error {
	s.count++

	return nil
}

// TestGet verifies tracking lookup and publication behavior.
func TestGet(t *testing.T) {
	publisher := &trackingPublisherStub{}
	service := NewService(trackingRegistryStub{}, publisher)

	history, err := service.Get(context.Background(), "manual", "TRACK-1")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if history == nil || history.TrackingNumber != "TRACK-1" {
		t.Fatalf("unexpected history = %#v", history)
	}
	if publisher.count != 1 {
		t.Fatalf("publish count = %d, want 1", publisher.count)
	}
}
