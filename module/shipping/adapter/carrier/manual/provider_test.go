package manual

import (
	"context"
	"testing"
	"time"

	"mannaiah/module/shipping/domain"
	"mannaiah/module/shipping/port"
)

type markRepositoryStub struct {
	mark *domain.ShippingMark
	err  error
}

// Create mocks shipping mark creation.
func (s *markRepositoryStub) Create(ctx context.Context, mark *domain.ShippingMark) error {
	return nil
}

// GetByID mocks shipping mark lookup by identifier.
func (s *markRepositoryStub) GetByID(ctx context.Context, id string) (*domain.ShippingMark, error) {
	return nil, domain.ErrNotFound
}

// GetByTrackingNumber mocks shipping mark lookup by tracking number.
func (s *markRepositoryStub) GetByTrackingNumber(ctx context.Context, trackingNumber string) (*domain.ShippingMark, error) {
	if s.err != nil {
		return nil, s.err
	}
	if s.mark == nil {
		return nil, domain.ErrNotFound
	}

	return s.mark, nil
}

// ListByOrderID mocks shipping mark listing by order identifier.
func (s *markRepositoryStub) ListByOrderID(ctx context.Context, orderID string) ([]domain.ShippingMark, error) {
	return nil, nil
}

// ListByBatchID mocks shipping mark listing by batch identifier.
func (s *markRepositoryStub) ListByBatchID(ctx context.Context, batchID string) ([]domain.ShippingMark, error) {
	return nil, nil
}

// Update mocks shipping mark updates.
func (s *markRepositoryStub) Update(ctx context.Context, mark *domain.ShippingMark) error {
	return nil
}

// Delete mocks shipping mark deletion.
func (s *markRepositoryStub) Delete(ctx context.Context, id string) error {
	return nil
}

// List mocks paginated shipping mark listing.
func (s *markRepositoryStub) List(ctx context.Context, query port.MarkListQuery) ([]domain.ShippingMark, int64, error) {
	return nil, 0, nil
}

// TestGenerateMark verifies manual mark generation behavior.
func TestGenerateMark(t *testing.T) {
	provider := NewProvider(nil)
	mark := &domain.ShippingMark{ID: "mark-1", CarrierID: "manual", OrderID: "order-1", Sender: domain.Address{Name: "S"}, Recipient: domain.Address{Name: "R"}, Units: []domain.PackageUnit{{Dimensions: domain.Dimensions{HeightCM: 1, WidthCM: 1, DepthCM: 1}}}}
	if err := provider.GenerateMark(context.Background(), mark); err != nil {
		t.Fatalf("GenerateMark() error = %v", err)
	}
	if mark.TrackingNumber == "" || mark.Status != domain.MarkStatusGenerated {
		t.Fatalf("unexpected mark = %#v", mark)
	}
}

// TestGetTrackingHistory uses manual carrier labels in tracking responses.
func TestGetTrackingHistory(t *testing.T) {
	provider := NewProvider(&markRepositoryStub{mark: &domain.ShippingMark{
		CarrierID:      "manual",
		TrackingNumber: "TRACK-1",
		Observations:   "interrapidisimo",
		UpdatedAt:      time.Now().UTC(),
	}})
	history, err := provider.GetTrackingHistory(context.Background(), "TRACK-1")
	if err != nil {
		t.Fatalf("GetTrackingHistory() error = %v", err)
	}
	if history.CarrierID != "manual_interrapidisimo" {
		t.Fatalf("history.CarrierID = %q", history.CarrierID)
	}
}

// TestSupportsCourier accepts manual carrier aliases.
func TestSupportsCourier(t *testing.T) {
	provider := NewProvider(nil)
	if !provider.SupportsCourier("manual_interrapidisimo") {
		t.Fatal("SupportsCourier(manual_interrapidisimo) = false")
	}
}
