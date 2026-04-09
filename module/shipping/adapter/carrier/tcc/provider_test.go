package tcc

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

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
		Enabled:              true,
		IsSandbox:            true,
		BaseURLOverride:      server.URL,
		AccessToken:          "token",
		ParcelAccountNumber:  "7000880",
		ExpressAccountNumber: "7000880",
		PaymentForm:          1,
		CODFeePercent:        4,
		Sender:               domain.Address{Name: "Sender", ID: "900", IDType: "NIT", AddressLine: "street", CityCode: "11001"},
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
	if quote.RequestSnapshot == "" {
		t.Fatalf("quote.RequestSnapshot is empty")
	}
	decodedQuoteRequest, decodeErr := base64.StdEncoding.DecodeString(quote.RequestSnapshot)
	if decodeErr != nil {
		t.Fatalf("decode quote request snapshot: %v", decodeErr)
	}
	if !strings.Contains(string(decodedQuoteRequest), "\"idciudadorigen\":\"11001000\"") {
		t.Fatalf("quote request snapshot = %s", string(decodedQuoteRequest))
	}
	if !strings.Contains(quote.RawResponse, "\"codigoResultado\":\"0\"") {
		t.Fatalf("quote.RawResponse = %q", quote.RawResponse)
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
	if mark.ManifestRef != "" {
		t.Fatalf("mark.ManifestRef = %q, want empty for response without manifest URL", mark.ManifestRef)
	}
	if dispatchRequest.OriginCityCode != "11001000" {
		t.Fatalf("dispatchRequest.OriginCityCode = %q", dispatchRequest.OriginCityCode)
	}
	if dispatchRequest.DestCityCode != "76001000" {
		t.Fatalf("dispatchRequest.DestCityCode = %q", dispatchRequest.DestCityCode)
	}
	// No quoted freight cost and no pre-computed charged amount: provider config fee is NOT
	// re-applied at generate time, so codCollectAmount = CollectOnDeliveryAmount - 0 = 100000.
	if dispatchRequest.CollectOnDeliveryAmount == nil || *dispatchRequest.CollectOnDeliveryAmount != "100000" {
		t.Fatalf("dispatchRequest.CollectOnDeliveryAmount = %v, want 100000", dispatchRequest.CollectOnDeliveryAmount)
	}
	if dispatchRequest.TotalProductValue == nil || *dispatchRequest.TotalProductValue != "100000" {
		t.Fatalf("dispatchRequest.TotalProductValue = %v, want 100000", dispatchRequest.TotalProductValue)
	}
	if dispatchRequest.PaymentForm != "2" {
		t.Fatalf("dispatchRequest.PaymentForm = %q, want \"2\" for COD", dispatchRequest.PaymentForm)
	}
	if mark.CollectOnDeliveryChargedAmount != 100000 {
		t.Fatalf("mark.CollectOnDeliveryChargedAmount = %v", mark.CollectOnDeliveryChargedAmount)
	}
	if mark.RequestSnapshot == "" {
		t.Fatal("mark.RequestSnapshot is empty")
	}
	decodedRequestSnapshot, decodeErr := base64.StdEncoding.DecodeString(mark.RequestSnapshot)
	if decodeErr != nil {
		t.Fatalf("decode mark request snapshot: %v", decodeErr)
	}
	if !strings.Contains(string(decodedRequestSnapshot), "\"formapago\":\"2\"") {
		t.Fatalf("mark request snapshot = %s", string(decodedRequestSnapshot))
	}
	if mark.ResponseSnapshot == "" {
		t.Fatal("mark.ResponseSnapshot is empty")
	}
	decodedResponseSnapshot, decodeErr := base64.StdEncoding.DecodeString(mark.ResponseSnapshot)
	if decodeErr != nil {
		t.Fatalf("decode mark response snapshot: %v", decodeErr)
	}
	if !strings.Contains(string(decodedResponseSnapshot), "\"codigoresultado\":\"0\"") {
		t.Fatalf("mark response snapshot = %s", string(decodedResponseSnapshot))
	}

	history, err := provider.GetTrackingHistory(context.Background(), mark.TrackingNumber)
	if err != nil {
		t.Fatalf("GetTrackingHistory() error = %v", err)
	}
	if history.GlobalStatus != domain.TrackingStatusCompleted {
		t.Fatalf("history status = %q", history.GlobalStatus)
	}
}

// TestProviderTrackingGlobalStatusPriority prefers business priority over later low-priority events.
func TestProviderTrackingGlobalStatusPriority(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		switch request.URL.Path {
		case "/api/clientes/remesas/consultarestatusremesasv3":
			_ = json.NewEncoder(writer).Encode(map[string]any{
				"remesas": []any{
					map[string]any{
						"numeroremesa": "1001",
						"estados": []any{
							map[string]any{"codigo": "3000", "descripcion": "Entregado", "fecha": "2026-03-27T01:09:43Z"},
							map[string]any{"codigo": "2000", "descripcion": "Envio en proceso de entrega", "fecha": "2026-03-27T04:13:43Z"},
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

	history, err := provider.GetTrackingHistory(context.Background(), "1001")
	if err != nil {
		t.Fatalf("GetTrackingHistory() error = %v", err)
	}
	if history.GlobalStatus != domain.TrackingStatusCompleted {
		t.Fatalf("history.GlobalStatus = %q, want %q", history.GlobalStatus, domain.TrackingStatusCompleted)
	}
	if history.LastUpdate.Format(time.RFC3339) != "2026-03-27T04:13:43Z" {
		t.Fatalf("history.LastUpdate = %s", history.LastUpdate.Format(time.RFC3339))
	}
}

// TestProviderTrackingGlobalStatusPriorityReturn verifies return statuses win over delivered ones.
func TestProviderTrackingGlobalStatusPriorityReturn(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		switch request.URL.Path {
		case "/api/clientes/remesas/consultarestatusremesasv3":
			_ = json.NewEncoder(writer).Encode(map[string]any{
				"remesas": []any{
					map[string]any{
						"numeroremesa": "1002",
						"estados": []any{
							map[string]any{"codigo": "3000", "descripcion": "Entregado", "fecha": "2026-03-27T01:09:43Z"},
							map[string]any{"codigo": "4000", "descripcion": "En devolucion", "fecha": "2026-03-26T20:00:00Z"},
							map[string]any{"codigo": "4200", "descripcion": "Reemplazada", "fecha": "2026-03-27T07:00:00Z"},
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

	history, err := provider.GetTrackingHistory(context.Background(), "1002")
	if err != nil {
		t.Fatalf("GetTrackingHistory() error = %v", err)
	}
	if history.GlobalStatus != domain.TrackingStatusReturn {
		t.Fatalf("history.GlobalStatus = %q, want %q", history.GlobalStatus, domain.TrackingStatusReturn)
	}
}

// TestProviderTrackingIncludesNovelties verifies TCC incidents are appended as normalized history events.
func TestProviderTrackingIncludesNovelties(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		switch request.URL.Path {
		case "/api/clientes/remesas/consultarestatusremesasv3":
			_ = json.NewEncoder(writer).Encode(map[string]any{
				"remesas": []any{
					map[string]any{
						"numeroremesa": "615193194",
						"estados": []any{
							map[string]any{"codigo": "901", "descripcion": "Envio programado", "fecha": "7/04/2026 3:45:00 p. m."},
							map[string]any{"codigo": "2000", "descripcion": "Envio en proceso de entrega", "fecha": "8/04/2026 10:08:41 a. m."},
						},
						"novedades": []any{
							map[string]any{
								"codigo":      "252",
								"fecha":       "8/04/2026 7:23:39 p. m.",
								"descripcion": "MERCANCÍA NO ENTREGADA A DESTINATARIO - NO SE ENCUENTRA EL DESTINATARIO O NO HAY NADIE EN LA DIRECCIÓN",
								"estado":      "Ejecutada",
								"observacion": "El destinatario no se encuentra, pasar nuevamente mañana",
							},
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

	history, err := provider.GetTrackingHistory(context.Background(), "615193194")
	if err != nil {
		t.Fatalf("GetTrackingHistory() error = %v", err)
	}
	if len(history.History) != 3 {
		t.Fatalf("len(history.History) = %d", len(history.History))
	}
	noveltyEvent := history.History[2]
	if noveltyEvent.Status != domain.TrackingStatusIncidence {
		t.Fatalf("noveltyEvent.Status = %q", noveltyEvent.Status)
	}
	if noveltyEvent.Code != "252" {
		t.Fatalf("noveltyEvent.Code = %q", noveltyEvent.Code)
	}
	if !strings.Contains(noveltyEvent.Text, "El destinatario no se encuentra") {
		t.Fatalf("noveltyEvent.Text = %q", noveltyEvent.Text)
	}
	if noveltyEvent.City != "BOGOTA" {
		t.Fatalf("noveltyEvent.City = %q", noveltyEvent.City)
	}
	if history.LastUpdate.Format(time.RFC3339) != "2026-04-09T00:23:39Z" {
		t.Fatalf("history.LastUpdate = %s", history.LastUpdate.Format(time.RFC3339))
	}
	if history.GlobalStatus != domain.TrackingStatusProcessing {
		t.Fatalf("history.GlobalStatus = %q", history.GlobalStatus)
	}
}

// TestProviderGrabardespacho7RealFieldNames verifies that the provider correctly reads
// the real grabardespacho7 production response field names ("remesa", "respuesta", "mensaje").
func TestProviderGrabardespacho7RealFieldNames(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		switch request.URL.Path {
		case "/api/clientes/remesas/grabardespacho7":
			_ = json.NewEncoder(writer).Encode(map[string]any{
				"remesa":           "615093378",
				"respuesta":        "0",
				"mensaje":          "Se ha grabado con exito la remesa y la unidad",
				"urlrotulos":       "https://carrier/labels/615093378",
				"urlremesa":        " 60050612",
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
	if mark.DocumentRef != "https://carrier/labels/615093378" {
		t.Fatalf("mark.DocumentRef = %q, want label URL", mark.DocumentRef)
	}
	if mark.ManifestType != domain.MarkDocumentLink {
		t.Fatalf("mark.ManifestType = %q, want %q", mark.ManifestType, domain.MarkDocumentLink)
	}
	if mark.ManifestRef != "https://carrier/relation/615093378" {
		t.Fatalf("mark.ManifestRef = %q, want manifest URL", mark.ManifestRef)
	}
	if mark.Status != domain.MarkStatusGenerated {
		t.Fatalf("mark.Status = %q", mark.Status)
	}
}

// TestProviderGrabardespacho7WithoutManifestDoesNotFail verifies dispatch success when manifest URL is absent.
func TestProviderGrabardespacho7WithoutManifestDoesNotFail(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		switch request.URL.Path {
		case "/api/clientes/remesas/grabardespacho7":
			_ = json.NewEncoder(writer).Encode(map[string]any{
				"remesa":     "615093379",
				"respuesta":  "0",
				"mensaje":    "Se ha grabado con exito la remesa y la unidad",
				"urlrotulos": "https://carrier/labels/615093379",
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
		ID:           "mark-no-manifest",
		OrderID:      "order-no-manifest",
		CarrierID:    "tcc",
		Sender:       domain.Address{Name: "Sender", ID: "900", IDType: "NIT", AddressLine: "street", CityCode: "11001"},
		Recipient:    domain.Address{Name: "Recipient", ID: "800", IDType: "CC", AddressLine: "street", CityCode: "76001"},
		Units:        []domain.PackageUnit{{Description: "box", PackageType: "CLEM_CAJA", Dimensions: domain.Dimensions{HeightCM: 10, WidthCM: 10, DepthCM: 10, RealWeightKG: 2}}},
		ShipmentMode: domain.ShipmentModeParcel,
	}
	if err := provider.GenerateMark(context.Background(), mark); err != nil {
		t.Fatalf("GenerateMark() error = %v", err)
	}
	if mark.ManifestType != "" {
		t.Fatalf("mark.ManifestType = %q, want empty", mark.ManifestType)
	}
	if mark.ManifestRef != "" {
		t.Fatalf("mark.ManifestRef = %q, want empty", mark.ManifestRef)
	}
}

// TestProviderGrabardespacho7MissingDocumentFails verifies dispatch failure when main mark document URL is absent.
func TestProviderGrabardespacho7MissingDocumentFails(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		switch request.URL.Path {
		case "/api/clientes/remesas/grabardespacho7":
			_ = json.NewEncoder(writer).Encode(map[string]any{
				"remesa":           "615093380",
				"respuesta":        "0",
				"mensaje":          "Se ha grabado con exito la remesa y la unidad",
				"urlrelacionenvio": "https://carrier/relation/615093380",
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
		ID:           "mark-missing-doc",
		OrderID:      "order-missing-doc",
		CarrierID:    "tcc",
		Sender:       domain.Address{Name: "Sender", ID: "900", IDType: "NIT", AddressLine: "street", CityCode: "11001"},
		Recipient:    domain.Address{Name: "Recipient", ID: "800", IDType: "CC", AddressLine: "street", CityCode: "76001"},
		Units:        []domain.PackageUnit{{Description: "box", PackageType: "CLEM_CAJA", Dimensions: domain.Dimensions{HeightCM: 10, WidthCM: 10, DepthCM: 10, RealWeightKG: 2}}},
		ShipmentMode: domain.ShipmentModeParcel,
	}
	if err := provider.GenerateMark(context.Background(), mark); err == nil {
		t.Fatalf("GenerateMark() error = nil, want non-nil when mark document URL is absent")
	}
}

// TestProviderCODNetAmount verifies that when QuotedFreightCost is set the net COD
// amount sent to TCC equals chargedAmount - freightCost, formapago is "2", and
// totalvalorproducto equals the net amount.
func TestProviderCODNetAmount(t *testing.T) {
	var dispatchRequest DispatchRequest
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		switch request.URL.Path {
		case "/api/clientes/remesas/grabardespacho7":
			if err := json.NewDecoder(request.Body).Decode(&dispatchRequest); err != nil {
				writer.WriteHeader(http.StatusBadRequest)
				return
			}
			_ = json.NewEncoder(writer).Encode(map[string]any{"remesa": "2001", "respuesta": "0", "mensaje": "OK", "urlguia": "https://carrier/guide/2001"})
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
		CODFeePercent:       4,
		Sender:              domain.Address{Name: "Sender", ID: "900", IDType: "NIT", AddressLine: "street", CityCode: "11001"},
	})
	if err != nil {
		t.Fatalf("NewProvider() error = %v", err)
	}

	mark := &domain.ShippingMark{
		ID:                      "mark-cod",
		OrderID:                 "order-cod",
		CarrierID:               "tcc",
		Sender:                  domain.Address{Name: "Sender", ID: "900", IDType: "NIT", AddressLine: "street", CityCode: "11001"},
		Recipient:               domain.Address{Name: "Recipient", ID: "800", IDType: "CC", AddressLine: "street", CityCode: "76001"},
		Units:                   []domain.PackageUnit{{Dimensions: domain.Dimensions{HeightCM: 10, WidthCM: 10, DepthCM: 10, RealWeightKG: 2}}},
		CollectOnDeliveryAmount: 150000,
		QuotedFreightCost:       25000,
		ShipmentMode:            domain.ShipmentModeParcel,
	}
	if err := provider.GenerateMark(context.Background(), mark); err != nil {
		t.Fatalf("GenerateMark() error = %v", err)
	}
	// collectOnDeliveryChargedAmount defaults to CollectOnDeliveryAmount (150000) when not
	// explicitly set on the mark; CODFeePercent from provider config is NOT re-applied.
	// netCOD = 150000 - 25000 = 125000.
	if dispatchRequest.PaymentForm != "2" {
		t.Fatalf("dispatchRequest.PaymentForm = %q, want \"2\"", dispatchRequest.PaymentForm)
	}
	if dispatchRequest.CollectOnDeliveryAmount == nil || *dispatchRequest.CollectOnDeliveryAmount != "125000" {
		t.Fatalf("dispatchRequest.CollectOnDeliveryAmount = %v, want 125000", dispatchRequest.CollectOnDeliveryAmount)
	}
	if dispatchRequest.TotalProductValue == nil || *dispatchRequest.TotalProductValue != "125000" {
		t.Fatalf("dispatchRequest.TotalProductValue = %v, want 125000", dispatchRequest.TotalProductValue)
	}
}

// TestProviderCODNetAmountWithPreComputedFee verifies that when CollectOnDeliveryChargedAmount
// is explicitly set on the mark (fee pre-applied at quote time), the net COD uses that
// stored value instead of the raw CollectOnDeliveryAmount.
func TestProviderCODNetAmountWithPreComputedFee(t *testing.T) {
	var dispatchRequest DispatchRequest
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		switch request.URL.Path {
		case "/api/clientes/remesas/grabardespacho7":
			if err := json.NewDecoder(request.Body).Decode(&dispatchRequest); err != nil {
				writer.WriteHeader(http.StatusBadRequest)
				return
			}
			_ = json.NewEncoder(writer).Encode(map[string]any{"remesa": "2002", "respuesta": "0", "mensaje": "OK", "urlguia": "https://carrier/guide/2002"})
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
		CODFeePercent:       4,
		Sender:              domain.Address{Name: "Sender", ID: "900", IDType: "NIT", AddressLine: "street", CityCode: "11001"},
	})
	if err != nil {
		t.Fatalf("NewProvider() error = %v", err)
	}

	mark := &domain.ShippingMark{
		ID:                             "mark-cod-fee",
		OrderID:                        "order-cod-fee",
		CarrierID:                      "tcc",
		Sender:                         domain.Address{Name: "Sender", ID: "900", IDType: "NIT", AddressLine: "street", CityCode: "11001"},
		Recipient:                      domain.Address{Name: "Recipient", ID: "800", IDType: "CC", AddressLine: "street", CityCode: "76001"},
		Units:                          []domain.PackageUnit{{Dimensions: domain.Dimensions{HeightCM: 10, WidthCM: 10, DepthCM: 10, RealWeightKG: 2}}},
		CollectOnDeliveryAmount:        150000,
		CollectOnDeliveryFeePercent:    4,
		CollectOnDeliveryChargedAmount: 156000,
		QuotedFreightCost:              25000,
		ShipmentMode:                   domain.ShipmentModeParcel,
	}
	if err := provider.GenerateMark(context.Background(), mark); err != nil {
		t.Fatalf("GenerateMark() error = %v", err)
	}
	// chargedAmount = mark.CollectOnDeliveryChargedAmount = 156000 (pre-applied 4% fee);
	// netCOD = 156000 - 25000 = 131000.
	if dispatchRequest.PaymentForm != "2" {
		t.Fatalf("dispatchRequest.PaymentForm = %q, want \"2\"", dispatchRequest.PaymentForm)
	}
	if dispatchRequest.CollectOnDeliveryAmount == nil || *dispatchRequest.CollectOnDeliveryAmount != "131000" {
		t.Fatalf("dispatchRequest.CollectOnDeliveryAmount = %v, want 131000", dispatchRequest.CollectOnDeliveryAmount)
	}
	if dispatchRequest.TotalProductValue == nil || *dispatchRequest.TotalProductValue != "131000" {
		t.Fatalf("dispatchRequest.TotalProductValue = %v, want 131000", dispatchRequest.TotalProductValue)
	}
}

// TestProviderDispatchDeclaredValueFallback verifies merchandise values default to 10000 when missing.
func TestProviderDispatchDeclaredValueFallback(t *testing.T) {
	var dispatchRequest DispatchRequest
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		switch request.URL.Path {
		case "/api/clientes/remesas/grabardespacho7":
			if err := json.NewDecoder(request.Body).Decode(&dispatchRequest); err != nil {
				writer.WriteHeader(http.StatusBadRequest)
				return
			}
			_ = json.NewEncoder(writer).Encode(map[string]any{"remesa": "3001", "respuesta": "0", "mensaje": "OK", "urlguia": "https://carrier/guide/3001"})
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
		ID:           "mark-declared-value-fallback",
		OrderID:      "order-declared-value-fallback",
		CarrierID:    "tcc",
		Sender:       domain.Address{Name: "Sender", ID: "900", IDType: "NIT", AddressLine: "street", CityCode: "11001"},
		Recipient:    domain.Address{Name: "Recipient", ID: "800", IDType: "CC", AddressLine: "street", CityCode: "76001"},
		Units:        []domain.PackageUnit{{Description: "box", PackageType: "CLEM_CAJA", Dimensions: domain.Dimensions{HeightCM: 10, WidthCM: 10, DepthCM: 10, RealWeightKG: 2}}},
		ShipmentMode: domain.ShipmentModeParcel,
	}
	if err := provider.GenerateMark(context.Background(), mark); err != nil {
		t.Fatalf("GenerateMark() error = %v", err)
	}
	if len(dispatchRequest.Units) != 1 {
		t.Fatalf("dispatchRequest.Units len = %d, want 1", len(dispatchRequest.Units))
	}
	if dispatchRequest.Units[0].DeclaredValue != "10000" {
		t.Fatalf("dispatchRequest.Units[0].DeclaredValue = %q, want \"10000\"", dispatchRequest.Units[0].DeclaredValue)
	}
	if dispatchRequest.TotalProductValue != nil {
		t.Fatalf("dispatchRequest.TotalProductValue = %v, want nil for non-COD shipments", dispatchRequest.TotalProductValue)
	}
	if dispatchRequest.CollectOnDeliveryAmount != nil {
		t.Fatalf("dispatchRequest.CollectOnDeliveryAmount = %v, want nil for non-COD shipments", dispatchRequest.CollectOnDeliveryAmount)
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

// TestProviderGenerateMarkGuardrailRejectsNonCODPaymentFormTwo verifies non-COD payloads fail when formapago is not 1.
func TestProviderGenerateMarkGuardrailRejectsNonCODPaymentFormTwo(t *testing.T) {
	dispatchCalls := 0
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path == "/api/clientes/remesas/grabardespacho7" {
			dispatchCalls++
		}
		writer.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	provider, err := NewProvider(ProviderConfig{
		Enabled:             true,
		IsSandbox:           true,
		BaseURLOverride:     server.URL,
		AccessToken:         "token",
		ParcelAccountNumber: "7000880",
		PaymentForm:         2,
		Sender:              domain.Address{Name: "Sender", ID: "900", IDType: "NIT", AddressLine: "street", CityCode: "11001"},
	})
	if err != nil {
		t.Fatalf("NewProvider() error = %v", err)
	}

	mark := &domain.ShippingMark{
		ID:           "mark-noncod-formapago",
		OrderID:      "order-noncod-formapago",
		CarrierID:    "tcc",
		Sender:       domain.Address{Name: "Sender", ID: "900", IDType: "NIT", AddressLine: "street", CityCode: "11001"},
		Recipient:    domain.Address{Name: "Recipient", ID: "800", IDType: "CC", AddressLine: "street", CityCode: "76001"},
		Units:        []domain.PackageUnit{{Description: "box", PackageType: "CLEM_CAJA", Dimensions: domain.Dimensions{HeightCM: 10, WidthCM: 10, DepthCM: 10, RealWeightKG: 2}}},
		ShipmentMode: domain.ShipmentModeParcel,
	}
	err = provider.GenerateMark(context.Background(), mark)
	if err == nil {
		t.Fatalf("GenerateMark() error = nil, want guardrail violation")
	}
	var guardrailErr *domain.GuardrailViolationError
	if !errors.As(err, &guardrailErr) {
		t.Fatalf("GenerateMark() error type = %T, want *domain.GuardrailViolationError", err)
	}
	if guardrailErr.Rule != guardrailNonCODPaymentFormRule {
		t.Fatalf("guardrailErr.Rule = %q, want %q", guardrailErr.Rule, guardrailNonCODPaymentFormRule)
	}
	if guardrailErr.MarkID != "mark-noncod-formapago" {
		t.Fatalf("guardrailErr.MarkID = %q", guardrailErr.MarkID)
	}
	if guardrailErr.OrderID != "order-noncod-formapago" {
		t.Fatalf("guardrailErr.OrderID = %q", guardrailErr.OrderID)
	}
	if !strings.Contains(guardrailErr.RequestPreview, "\"formapago\":\"2\"") {
		t.Fatalf("guardrailErr.RequestPreview = %q, want formapago=2", guardrailErr.RequestPreview)
	}
	if dispatchCalls != 0 {
		t.Fatalf("dispatchCalls = %d, want 0", dispatchCalls)
	}
}

// TestProviderGenerateMarkGuardrailRejectsCODWithoutPositiveCollect verifies COD payloads fail when net recaudo is zero or negative.
func TestProviderGenerateMarkGuardrailRejectsCODWithoutPositiveCollect(t *testing.T) {
	dispatchCalls := 0
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path == "/api/clientes/remesas/grabardespacho7" {
			dispatchCalls++
		}
		writer.WriteHeader(http.StatusNotFound)
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
		ID:                      "mark-cod-zero-net",
		OrderID:                 "order-cod-zero-net",
		CarrierID:               "tcc",
		Sender:                  domain.Address{Name: "Sender", ID: "900", IDType: "NIT", AddressLine: "street", CityCode: "11001"},
		Recipient:               domain.Address{Name: "Recipient", ID: "800", IDType: "CC", AddressLine: "street", CityCode: "76001"},
		Units:                   []domain.PackageUnit{{Description: "box", PackageType: "CLEM_CAJA", Dimensions: domain.Dimensions{HeightCM: 10, WidthCM: 10, DepthCM: 10, RealWeightKG: 2}}},
		CollectOnDeliveryAmount: 15000,
		QuotedFreightCost:       15000,
		ShipmentMode:            domain.ShipmentModeParcel,
	}
	err = provider.GenerateMark(context.Background(), mark)
	if err == nil {
		t.Fatalf("GenerateMark() error = nil, want guardrail violation")
	}
	var guardrailErr *domain.GuardrailViolationError
	if !errors.As(err, &guardrailErr) {
		t.Fatalf("GenerateMark() error type = %T, want *domain.GuardrailViolationError", err)
	}
	if guardrailErr.Rule != guardrailCODCollectAmountRule {
		t.Fatalf("guardrailErr.Rule = %q, want %q", guardrailErr.Rule, guardrailCODCollectAmountRule)
	}
	if !strings.Contains(guardrailErr.RequestPreview, "\"formapago\":\"2\"") {
		t.Fatalf("guardrailErr.RequestPreview = %q, want formapago=2", guardrailErr.RequestPreview)
	}
	if !strings.Contains(guardrailErr.RequestPreview, "\"recaudoproducto\":\"0\"") {
		t.Fatalf("guardrailErr.RequestPreview = %q, want recaudo 0", guardrailErr.RequestPreview)
	}
	if dispatchCalls != 0 {
		t.Fatalf("dispatchCalls = %d, want 0", dispatchCalls)
	}
}
