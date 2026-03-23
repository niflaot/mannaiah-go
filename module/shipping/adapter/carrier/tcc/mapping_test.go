package tcc

import (
	"testing"

	"mannaiah/module/shipping/domain"
)

// TestBuildQuoteRequest verifies quotation request mapping behavior.
func TestBuildQuoteRequest(t *testing.T) {
	request := BuildQuoteRequest("7000880", 1, domain.QuotationRequest{
		CarrierID:      "tcc",
		OriginCityCode: "11001000",
		DestCityCode:   "76001000",
		DeclaredValue:  50000,
		Units:          []domain.PackageUnit{{Dimensions: domain.Dimensions{HeightCM: 10, WidthCM: 10, DepthCM: 10, RealWeightKG: 2}}},
	})
	if request.Account != "7000880" {
		t.Fatalf("request.Account = %q", request.Account)
	}
	if len(request.Units) != 1 {
		t.Fatalf("units len = %d", len(request.Units))
	}
}

// TestMapTrackingStatus verifies TCC code mapping behavior.
func TestMapTrackingStatus(t *testing.T) {
	if status := MapTrackingStatus("3000", "Entregado"); status != domain.TrackingStatusCompleted {
		t.Fatalf("status = %q", status)
	}
	if status := MapTrackingStatus("500", "Origen"); status != domain.TrackingStatusOrigin {
		t.Fatalf("status = %q", status)
	}
}
