package tcc

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"mannaiah/module/shipping/domain"
)

// TestProviderLifecycle verifies quote, generate mark, and tracking behaviors.
func TestProviderLifecycle(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		switch request.URL.Path {
		case "/api/clientes/tarifas/v5/consultarliquidacion":
			_ = json.NewEncoder(writer).Encode(map[string]any{"codigoResultado": "0", "mensajeResultado": "OK", "total": map[string]any{"totaldespacho": 20000, "unidadnegocio": "PAQ"}})
		case "/api/clientes/remesas/grabardespacho8":
			_ = json.NewEncoder(writer).Encode(map[string]any{"codigoresultado": "0", "mensajeresultado": "OK", "numeroremesa": "1001", "urlguia": "https://carrier/guide/1001"})
		case "/api/clientes/remesas/consultarestatusremesasv3":
			_ = json.NewEncoder(writer).Encode(map[string]any{"codigoresultado": "0", "mensajeresultado": "OK", "estados": []any{map[string]any{"codigo": "3000", "descripcion": "Entregado", "fecha": "2026-03-22T10:00:00Z"}}})
		default:
			writer.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	provider, err := NewProvider(ProviderConfig{Enabled: true, BaseURL: server.URL, AccessToken: "token", AccountNumber: "7000880", BusinessUnit: 1, PaymentForm: 1, Sender: domain.Address{Name: "Sender", ID: "900", IDType: "NIT", AddressLine: "street", CityCode: "11001000"}})
	if err != nil {
		t.Fatalf("NewProvider() error = %v", err)
	}

	quote, err := provider.Quote(context.Background(), domain.QuotationRequest{CarrierID: "tcc", OriginCityCode: "11001000", DestCityCode: "76001000", DeclaredValue: 50000, Units: []domain.PackageUnit{{Dimensions: domain.Dimensions{HeightCM: 10, WidthCM: 10, DepthCM: 10, RealWeightKG: 2}}}})
	if err != nil {
		t.Fatalf("Quote() error = %v", err)
	}
	if quote.FreightCost <= 0 {
		t.Fatalf("invalid quote = %#v", quote)
	}

	mark := &domain.ShippingMark{ID: "mark-1", OrderID: "order-1", CarrierID: "tcc", Sender: domain.Address{Name: "Sender", ID: "900", IDType: "NIT", AddressLine: "street", CityCode: "11001000"}, Recipient: domain.Address{Name: "Recipient", ID: "800", IDType: "CC", AddressLine: "street", CityCode: "76001000"}, Units: []domain.PackageUnit{{Description: "box", PackageType: "CLEM_CAJA", Dimensions: domain.Dimensions{HeightCM: 10, WidthCM: 10, DepthCM: 10, RealWeightKG: 2}}}}
	if err := provider.GenerateMark(context.Background(), mark); err != nil {
		t.Fatalf("GenerateMark() error = %v", err)
	}
	if mark.TrackingNumber == "" || mark.Status != domain.MarkStatusGenerated {
		t.Fatalf("unexpected mark = %#v", mark)
	}

	history, err := provider.GetTrackingHistory(context.Background(), mark.TrackingNumber)
	if err != nil {
		t.Fatalf("GetTrackingHistory() error = %v", err)
	}
	if history.GlobalStatus != domain.TrackingStatusCompleted {
		t.Fatalf("history status = %q", history.GlobalStatus)
	}
}
