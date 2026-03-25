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
	var dispatchRequest DispatchRequest
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		switch request.URL.Path {
		case "/api/clientes/tarifas/v5/consultarliquidacion":
			_ = json.NewEncoder(writer).Encode(map[string]any{"codigoResultado": "0", "mensajeResultado": "OK", "total": map[string]any{"totaldespacho": 20000, "unidadnegocio": "PAQ"}})
		case "/api/clientes/remesas/grabardespacho7":
			if err := json.NewDecoder(request.Body).Decode(&dispatchRequest); err != nil {
				writer.WriteHeader(http.StatusBadRequest)
				return
			}
			_ = json.NewEncoder(writer).Encode(map[string]any{"codigoresultado": "0", "mensajeresultado": "OK", "numeroremesa": "1001", "urlguia": "https://carrier/guide/1001"})
		case "/api/clientes/remesas/consultarestatusremesasv3":
			_ = json.NewEncoder(writer).Encode(map[string]any{
				"remesas": []any{
					map[string]any{
						"numeroremesa": "1001",
						"estados": []any{
							map[string]any{"codigo": "3000", "descripcion": "Entregado", "fecha": "2026-03-22T10:00:00Z"},
						},
						"ciudadorigen":  map[string]any{"descripcion": "MEDELLIN"},
						"ciudaddestino": map[string]any{"descripcion": "BOGOTA"},
					},
				},
				"respuesta": map[string]any{"codigo": "0", "mensaje": "OK"},
			})
		default:
			writer.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	provider, err := NewProvider(ProviderConfig{
		Enabled:         true,
		IsSandbox:       true,
		BaseURLOverride: server.URL,
		AccessToken:          "token",
		ParcelAccountNumber:  "7000880",
		ExpressAccountNumber: "7000880",
		PaymentForm:          1,
		CODFeePercent:   4,
		Sender:          domain.Address{Name: "Sender", ID: "900", IDType: "NIT", AddressLine: "street", CityCode: "11001"},
	})
	if err != nil {
		t.Fatalf("NewProvider() error = %v", err)
	}

	quote, err := provider.Quote(context.Background(), domain.QuotationRequest{
		CarrierID:               "tcc",
		OriginCityCode:          "11001",
		DestCityCode:            "76001",
		DeclaredValue:           50000,
		CollectOnDeliveryAmount: 100000,
		ShipmentMode:            domain.ShipmentModeParcel,
		Units: []domain.PackageUnit{
			{Dimensions: domain.Dimensions{HeightCM: 10, WidthCM: 10, DepthCM: 10, RealWeightKG: 2}},
		},
	})
	if err != nil {
		t.Fatalf("Quote() error = %v", err)
	}
	if quote.FreightCost <= 0 {
		t.Fatalf("invalid quote = %#v", quote)
	}
	if quote.CollectOnDeliveryAmount != 100000 {
		t.Fatalf("quote.CollectOnDeliveryAmount = %v", quote.CollectOnDeliveryAmount)
	}
	if quote.CollectOnDeliveryFeePercent != 4 {
		t.Fatalf("quote.CollectOnDeliveryFeePercent = %v", quote.CollectOnDeliveryFeePercent)
	}
	if quote.CollectOnDeliveryFeeAmount != 4000 {
		t.Fatalf("quote.CollectOnDeliveryFeeAmount = %v", quote.CollectOnDeliveryFeeAmount)
	}
	if quote.CollectOnDeliveryChargedAmount != 104000 {
		t.Fatalf("quote.CollectOnDeliveryChargedAmount = %v", quote.CollectOnDeliveryChargedAmount)
	}

	mark := &domain.ShippingMark{
		ID:                      "mark-1",
		OrderID:                 "order-1",
		CarrierID:               "tcc",
		Sender:                  domain.Address{Name: "Sender", ID: "900", IDType: "NIT", AddressLine: "street", CityCode: "11001"},
		Recipient:               domain.Address{Name: "Recipient", ID: "800", IDType: "CC", AddressLine: "street", CityCode: "76001"},
		Units:                   []domain.PackageUnit{{Description: "box", PackageType: "CLEM_CAJA", Dimensions: domain.Dimensions{HeightCM: 10, WidthCM: 10, DepthCM: 10, RealWeightKG: 2}}},
		CollectOnDeliveryAmount: 100000,
		ShipmentMode:            domain.ShipmentModeParcel,
	}
	if err := provider.GenerateMark(context.Background(), mark); err != nil {
		t.Fatalf("GenerateMark() error = %v", err)
	}
	if mark.TrackingNumber == "" || mark.Status != domain.MarkStatusGenerated {
		t.Fatalf("unexpected mark = %#v", mark)
	}
	if dispatchRequest.OriginCityCode != "11001000" {
		t.Fatalf("dispatchRequest.OriginCityCode = %q", dispatchRequest.OriginCityCode)
	}
	if dispatchRequest.DestCityCode != "76001000" {
		t.Fatalf("dispatchRequest.DestCityCode = %q", dispatchRequest.DestCityCode)
	}
	if dispatchRequest.CollectOnDeliveryAmount != "104000" {
		t.Fatalf("dispatchRequest.CollectOnDeliveryAmount = %q", dispatchRequest.CollectOnDeliveryAmount)
	}
	if mark.CollectOnDeliveryChargedAmount != 104000 {
		t.Fatalf("mark.CollectOnDeliveryChargedAmount = %v", mark.CollectOnDeliveryChargedAmount)
	}

	history, err := provider.GetTrackingHistory(context.Background(), mark.TrackingNumber)
	if err != nil {
		t.Fatalf("GetTrackingHistory() error = %v", err)
	}
	if history.GlobalStatus != domain.TrackingStatusCompleted {
		t.Fatalf("history status = %q", history.GlobalStatus)
	}
}

// TestProviderGrabardespacho7RealFieldNames verifies that the provider correctly reads
// the real grabardespacho7 production response field names ("remesa", "respuesta", "mensaje").
func TestProviderGrabardespacho7RealFieldNames(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		switch request.URL.Path {
		case "/api/clientes/remesas/grabardespacho7":
			_ = json.NewEncoder(writer).Encode(map[string]any{
				"remesa":      "615093378",
				"respuesta":   "0",
				"mensaje":     "Se ha grabado con exito la remesa y la unidad",
				"urlrotulos":  "https://carrier/labels/615093378",
				"urlremesa":   " 60050612",
				"urlrelacionenvio": "https://carrier/relation/615093378",
			})
		default:
			writer.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	provider, err := NewProvider(ProviderConfig{
		Enabled:             true,
		IsSandbox:           true,
		BaseURLOverride:     server.URL,
		AccessToken:         "token",
		ParcelAccountNumber: "7000880",
		PaymentForm:         1,
		Sender:              domain.Address{Name: "Sender", ID: "900", IDType: "NIT", AddressLine: "street", CityCode: "11001"},
	})
	if err != nil {
		t.Fatalf("NewProvider() error = %v", err)
	}

	mark := &domain.ShippingMark{
		ID:           "mark-real",
		OrderID:      "order-real",
		CarrierID:    "tcc",
		Sender:       domain.Address{Name: "Sender", ID: "900", IDType: "NIT", AddressLine: "street", CityCode: "11001"},
		Recipient:    domain.Address{Name: "Recipient", ID: "800", IDType: "CC", AddressLine: "street", CityCode: "76001"},
		Units:        []domain.PackageUnit{{Description: "box", PackageType: "CLEM_CAJA", Dimensions: domain.Dimensions{HeightCM: 10, WidthCM: 10, DepthCM: 10, RealWeightKG: 2}}},
		ShipmentMode: domain.ShipmentModeParcel,
	}
	if err := provider.GenerateMark(context.Background(), mark); err != nil {
		t.Fatalf("GenerateMark() error = %v", err)
	}
	if mark.TrackingNumber != "615093378" {
		t.Fatalf("mark.TrackingNumber = %q, want %q", mark.TrackingNumber, "615093378")
	}
	if mark.DocumentRef == "" {
		t.Fatalf("mark.DocumentRef is empty")
	}
	if mark.Status != domain.MarkStatusGenerated {
		t.Fatalf("mark.Status = %q", mark.Status)
	}
}

// TestProviderGrabardespacho7URLFallback verifies that the tracking number is extracted
// from the urlguia "ti" query parameter when no explicit remesa field is returned.
func TestProviderGrabardespacho7URLFallback(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		switch request.URL.Path {
		case "/api/clientes/remesas/grabardespacho7":
			_ = json.NewEncoder(writer).Encode(map[string]any{
				"respuesta": "0",
				"mensaje":   "Se ha grabado con exito la remesa y la unidad",
				"urlguia":   "https://somos.tcc.com.co/Informesdsp?opc=1&ti=615097792",
			})
		default:
			writer.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	provider, err := NewProvider(ProviderConfig{
		Enabled:             true,
		IsSandbox:           true,
		BaseURLOverride:     server.URL,
		AccessToken:         "token",
		ParcelAccountNumber: "7000880",
		PaymentForm:         1,
		Sender:              domain.Address{Name: "Sender", ID: "900", IDType: "NIT", AddressLine: "street", CityCode: "11001"},
	})
	if err != nil {
		t.Fatalf("NewProvider() error = %v", err)
	}

	mark := &domain.ShippingMark{
		ID:           "mark-url-fallback",
		OrderID:      "order-url-fallback",
		CarrierID:    "tcc",
		Sender:       domain.Address{Name: "Sender", ID: "900", IDType: "NIT", AddressLine: "street", CityCode: "11001"},
		Recipient:    domain.Address{Name: "Recipient", ID: "800", IDType: "CC", AddressLine: "street", CityCode: "76001"},
		Units:        []domain.PackageUnit{{Description: "box", PackageType: "CLEM_CAJA", Dimensions: domain.Dimensions{HeightCM: 10, WidthCM: 10, DepthCM: 10, RealWeightKG: 2}}},
		ShipmentMode: domain.ShipmentModeParcel,
	}
	if err := provider.GenerateMark(context.Background(), mark); err != nil {
		t.Fatalf("GenerateMark() error = %v", err)
	}
	if mark.TrackingNumber != "615097792" {
		t.Fatalf("mark.TrackingNumber = %q, want %q", mark.TrackingNumber, "615097792")
	}
	if mark.DocumentRef != "https://somos.tcc.com.co/Informesdsp?opc=1&ti=615097792" {
		t.Fatalf("mark.DocumentRef = %q", mark.DocumentRef)
	}
}

// TestCalculateCollectOnDeliveryChargedAmount verifies COD fee behavior.
func TestCalculateCollectOnDeliveryChargedAmount(t *testing.T) {
	if got := calculateCollectOnDeliveryChargedAmount(100000, 4); got != 104000 {
		t.Fatalf("calculateCollectOnDeliveryChargedAmount() = %v", got)
	}
	if got := calculateCollectOnDeliveryChargedAmount(100000, -4); got != 100000 {
		t.Fatalf("calculateCollectOnDeliveryChargedAmount() negative fee = %v", got)
	}
	if got := calculateCollectOnDeliveryChargedAmount(0, 4); got != 0 {
		t.Fatalf("calculateCollectOnDeliveryChargedAmount() zero amount = %v", got)
	}
}
