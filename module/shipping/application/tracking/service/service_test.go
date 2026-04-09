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

type trackingRepositoryStub struct {
	rows  []domain.ShippingMark
	total int64
}

func (trackingRepositoryStub) Create(ctx context.Context, mark *domain.ShippingMark) error {
	return nil
}

func (trackingRepositoryStub) GetByID(ctx context.Context, id string) (*domain.ShippingMark, error) {
	return nil, domain.ErrNotFound
}

func (trackingRepositoryStub) GetByTrackingNumber(ctx context.Context, trackingNumber string) (*domain.ShippingMark, error) {
	return nil, domain.ErrNotFound
}

func (trackingRepositoryStub) ListByOrderID(ctx context.Context, orderID string) ([]domain.ShippingMark, error) {
	return nil, nil
}

func (trackingRepositoryStub) ListByBatchID(ctx context.Context, batchID string) ([]domain.ShippingMark, error) {
	return nil, nil
}

func (trackingRepositoryStub) Update(ctx context.Context, mark *domain.ShippingMark) error {
	return nil
}

func (trackingRepositoryStub) Delete(ctx context.Context, id string) error {
	return nil
}

func (s trackingRepositoryStub) List(ctx context.Context, query port.MarkListQuery) ([]domain.ShippingMark, int64, error) {
	filtered := make([]domain.ShippingMark, 0, len(s.rows))
	for _, row := range s.rows {
		if query.RequireTracking && row.TrackingNumber == "" {
			continue
		}
		if len(query.ExcludedStatuses) > 0 {
			excluded := false
			for _, status := range query.ExcludedStatuses {
				if row.Status == status {
					excluded = true
					break
				}
			}
			if excluded {
				continue
			}
		}
		filtered = append(filtered, row)
	}
	total := s.total
	if total == 0 {
		total = int64(len(filtered))
	}
	page := query.Page
	if page <= 0 {
		page = 1
	}
	limit := query.Limit
	if limit <= 0 {
		limit = len(filtered)
	}
	start := (page - 1) * limit
	if start >= len(filtered) {
		return []domain.ShippingMark{}, total, nil
	}
	end := start + limit
	if end > len(filtered) {
		end = len(filtered)
	}

	return filtered[start:end], total, nil
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
	service := NewService(trackingRepositoryStub{}, trackingRegistryStub{}, publisher)

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

// TestList marks manual rows with MANUAL status and manual carrier aliases.
func TestList(t *testing.T) {
	now := time.Now().UTC()
	service := NewService(trackingRepositoryStub{
		rows: []domain.ShippingMark{
			{
				ID:             "mark-manual",
				OrderID:        "order-1",
				CarrierID:      "manual",
				TrackingNumber: "MANUAL-1",
				Status:         domain.MarkStatusCreated,
				Observations:   "interrapidisimo",
				Recipient:      domain.Address{Name: "Ian Castano"},
				CreatedAt:      now,
			},
			{
				ID:             "mark-tcc",
				OrderID:        "order-2",
				CarrierID:      "tcc",
				TrackingNumber: "TRACK-2",
				Status:         domain.MarkStatusCreated,
				Recipient:      domain.Address{Name: "Kevin Cardenas"},
				CreatedAt:      now.Add(-time.Minute),
			},
			{
				ID:             "mark-draft",
				OrderID:        "order-3",
				CarrierID:      "tcc",
				TrackingNumber: "TRACK-3",
				Status:         domain.MarkStatusQuoted,
				Recipient:      domain.Address{Name: "Draft"},
				CreatedAt:      now.Add(-2 * time.Minute),
			},
		},
	}, trackingRegistryStub{}, &trackingPublisherStub{})

	rows, total, err := service.List(context.Background(), ListQuery{Page: 1, Limit: 10})
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if total != 2 {
		t.Fatalf("List() total = %d, want 2", total)
	}
	if len(rows) != 2 {
		t.Fatalf("List() len = %d, want 2", len(rows))
	}
	if rows[0].CarrierID != "manual_interrapidisimo" || rows[0].LastStatus != trackingStatusManual {
		t.Fatalf("manual row = %#v", rows[0])
	}
	if rows[1].LastStatus != string(domain.TrackingStatusProcessing) {
		t.Fatalf("tcc row lastStatus = %q", rows[1].LastStatus)
	}
}

// TestList filters rows by last tracking status.
func TestListByStatus(t *testing.T) {
	service := NewService(trackingRepositoryStub{
		rows: []domain.ShippingMark{
			{ID: "mark-manual", CarrierID: "manual", TrackingNumber: "MANUAL-1", Status: domain.MarkStatusCreated, Observations: "inter"},
			{ID: "mark-tcc", CarrierID: "tcc", TrackingNumber: "TRACK-2", Status: domain.MarkStatusCreated},
		},
	}, trackingRegistryStub{}, &trackingPublisherStub{})

	rows, total, err := service.List(context.Background(), ListQuery{Status: "MANUAL", Page: 1, Limit: 10})
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if total != 1 || len(rows) != 1 {
		t.Fatalf("filtered manual rows = %d total=%d", len(rows), total)
	}
	if rows[0].ID != "mark-manual" {
		t.Fatalf("filtered row id = %q", rows[0].ID)
	}
}
