package tcc

import (
	"testing"

	"mannaiah/module/shipping/domain"
)

// TestBuildQuoteRequest verifies quotation request mapping behavior.
func TestBuildQuoteRequest(t *testing.T) {
	request := BuildQuoteRequest("7000880", domain.QuotationRequest{
		CarrierID:      "tcc",
		OriginCityCode: "11001",
		DestCityCode:   "76001",
		DeclaredValue:  50000,
		ShipmentMode:   domain.ShipmentModeParcel,
		Units:          []domain.PackageUnit{{Dimensions: domain.Dimensions{HeightCM: 10, WidthCM: 10, DepthCM: 10, RealWeightKG: 2}}},
	})
	if request.Account != "7000880" {
		t.Fatalf("request.Account = %q", request.Account)
	}
	if len(request.Units) != 1 {
		t.Fatalf("units len = %d", len(request.Units))
	}
	if request.OriginCityCode != "11001000" {
		t.Fatalf("request.OriginCityCode = %q", request.OriginCityCode)
	}
	if request.DestCityCode != "76001000" {
		t.Fatalf("request.DestCityCode = %q", request.DestCityCode)
	}
}

// TestNormalizeCityCode verifies TCC city-code normalization behavior.
func TestNormalizeCityCode(t *testing.T) {
	if got := NormalizeCityCode("11001"); got != "11001000" {
		t.Fatalf("NormalizeCityCode(11001) = %q", got)
	}
	if got := NormalizeCityCode("05001000"); got != "05001000" {
		t.Fatalf("NormalizeCityCode(05001000) = %q", got)
	}
	if got := NormalizeCityCode("ABC01"); got != "ABC01" {
		t.Fatalf("NormalizeCityCode(ABC01) = %q", got)
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
