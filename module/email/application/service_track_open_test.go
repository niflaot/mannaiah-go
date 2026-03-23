package application

import (
	"context"
	"errors"
	"testing"

	"mannaiah/module/email/domain"
)

// TestTrackOpenUpdatesCurrentStatus verifies open tracking updates delivery current status and appends history.
func TestTrackOpenUpdatesCurrentStatus(t *testing.T) {
	t.Parallel()

	repository := &snsWebhookRepositoryStub{
		deliveriesByID: map[string]*domain.Delivery{
			"delivery-1": {
				ID:     "delivery-1",
				Email:  "coccostoreco@gmail.com",
				Status: domain.StatusSubmitted,
			},
		},
		providerToID: map[string]string{},
	}
	service, err := NewService(repository, snsWebhookProviderStub{})
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}

	trackErr := service.TrackOpen(context.Background(), "delivery-1")
	if trackErr != nil {
		t.Fatalf("TrackOpen() error = %v", trackErr)
	}
	delivery, getErr := repository.GetByID(context.Background(), "delivery-1")
	if getErr != nil {
		t.Fatalf("GetByID() error = %v", getErr)
	}
	if delivery.Status != domain.StatusOpened {
		t.Fatalf("delivery.Status = %q, want %q", delivery.Status, domain.StatusOpened)
	}

	entries := repository.statusEntriesByID["delivery-1"]
	if len(entries) != 1 {
		t.Fatalf("status entry count = %d, want 1", len(entries))
	}
	if entries[0].Status != domain.StatusOpened {
		t.Fatalf("status entry status = %q, want %q", entries[0].Status, domain.StatusOpened)
	}
}

// TestTrackOpenMissingDelivery verifies open tracking returns not-found when the delivery does not exist.
func TestTrackOpenMissingDelivery(t *testing.T) {
	t.Parallel()

	service, err := NewService(&snsWebhookRepositoryStub{
		deliveriesByID: map[string]*domain.Delivery{},
		providerToID:   map[string]string{},
	}, snsWebhookProviderStub{})
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}

	trackErr := service.TrackOpen(context.Background(), "missing-delivery")
	if trackErr == nil {
		t.Fatalf("TrackOpen() error = nil, want non-nil")
	}
	if !errors.Is(trackErr, domain.ErrNotFound) {
		t.Fatalf("TrackOpen() error = %v, want %v", trackErr, domain.ErrNotFound)
	}
}
