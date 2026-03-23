package manual

import (
	"context"
	"testing"

	"mannaiah/module/shipping/domain"
)

// TestGenerateMark verifies manual mark generation behavior.
func TestGenerateMark(t *testing.T) {
	provider := NewProvider()
	mark := &domain.ShippingMark{ID: "mark-1", CarrierID: "manual", OrderID: "order-1", Sender: domain.Address{Name: "S"}, Recipient: domain.Address{Name: "R"}, Units: []domain.PackageUnit{{Dimensions: domain.Dimensions{HeightCM: 1, WidthCM: 1, DepthCM: 1}}}}
	if err := provider.GenerateMark(context.Background(), mark); err != nil {
		t.Fatalf("GenerateMark() error = %v", err)
	}
	if mark.TrackingNumber == "" || mark.Status != domain.MarkStatusGenerated {
		t.Fatalf("unexpected mark = %#v", mark)
	}
}
