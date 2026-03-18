package tcc

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
	"mannaiah/module/shipping/domain"
)

// TestNewClientValidation verifies constructor validation behavior.
func TestNewClientValidation(t *testing.T) {
	if _, err := NewClient(Config{}); !errors.Is(err, ErrBaseURLRequired) {
		t.Fatalf("NewClient() error = %v, want %v", err, ErrBaseURLRequired)
	}
	if _, err := NewClient(Config{BaseURL: "http://127.0.0.1"}); !errors.Is(err, ErrAccessTokenRequired) {
		t.Fatalf("NewClient() error = %v, want %v", err, ErrAccessTokenRequired)
	}
	if _, err := NewClient(Config{BaseURL: "http://127.0.0.1", AccessToken: "a"}); !errors.Is(err, ErrAccountRequired) {
		t.Fatalf("NewClient() error = %v, want %v", err, ErrAccountRequired)
	}
	if _, err := NewClient(Config{BaseURL: "http://127.0.0.1", AccessToken: "a", Account: "7000880"}); !errors.Is(err, ErrIdentifierRequired) {
		t.Fatalf("NewClient() error = %v, want %v", err, ErrIdentifierRequired)
	}
}

// TestQuoteSuccess verifies successful quote mapping behavior.
func TestQuoteSuccess(t *testing.T) {
	type capturedRequest struct {
		HeaderAccessToken string
		HeaderTraceParent string
		Payload           tccQuoteRequest
	}
	captured := capturedRequest{}

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		captured.HeaderAccessToken = request.Header.Get("AccessToken")
		captured.HeaderTraceParent = request.Header.Get("traceparent")

		body, err := io.ReadAll(request.Body)
		if err != nil {
			writer.WriteHeader(http.StatusInternalServerError)
			return
		}
		if err := json.Unmarshal(body, &captured.Payload); err != nil {
			writer.WriteHeader(http.StatusBadRequest)
			return
		}

		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{"codigoResultado":"0","mensajeResultado":"OK","total":{"totaldespacho":25800,"unidadnegocio":"PAQUETERIA"}}`))
	}))
	defer server.Close()

	client, err := NewClient(Config{
		BaseURL:     server.URL,
		AccessToken: "token-1",
		Account:     "7000880",
		Identifier:  "901599500",
	})
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	previousPropagator := otel.GetTextMapPropagator()
	otel.SetTextMapPropagator(propagation.TraceContext{})
	defer otel.SetTextMapPropagator(previousPropagator)

	spanContext := trace.NewSpanContext(trace.SpanContextConfig{
		TraceID:    [16]byte{1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1},
		SpanID:     [8]byte{2, 2, 2, 2, 2, 2, 2, 2},
		TraceFlags: trace.FlagsSampled,
	})
	ctx := trace.ContextWithSpanContext(context.Background(), spanContext)

	result, err := client.Quote(ctx, domain.QuoteRequest{
		Carrier:             domain.CarrierTCC,
		BusinessUnit:        domain.BusinessUnitCourier,
		OriginCityCode:      "05001",
		DestinationCityCode: "11001000",
		DeclaredValue:       150000,
		Units: []domain.QuoteUnit{
			{Number: 1, RealWeight: 2.5, Height: 15, Width: 20, Length: 30},
		},
	})
	if err != nil {
		t.Fatalf("Quote() error = %v", err)
	}
	if result == nil {
		t.Fatalf("expected quote result")
	}
	if result.QuoteValue != 25800 {
		t.Fatalf("result.QuoteValue = %v, want %v", result.QuoteValue, 25800)
	}
	if result.BusinessUnit != domain.BusinessUnitCourier {
		t.Fatalf("result.BusinessUnit = %q, want %q", result.BusinessUnit, domain.BusinessUnitCourier)
	}
	if captured.HeaderAccessToken != "token-1" {
		t.Fatalf("AccessToken = %q, want %q", captured.HeaderAccessToken, "token-1")
	}
	if strings.TrimSpace(captured.HeaderTraceParent) == "" {
		t.Fatalf("expected traceparent header")
	}
	if captured.Payload.IDCiudadOrigen != "05001000" {
		t.Fatalf("payload.idciudadorigen = %q, want %q", captured.Payload.IDCiudadOrigen, "05001000")
	}
	if captured.Payload.IDCiudadDestino != "11001000" {
		t.Fatalf("payload.idciudaddestino = %q, want %q", captured.Payload.IDCiudadDestino, "11001000")
	}
	if captured.Payload.IDUnidadNegocio != 1 {
		t.Fatalf("payload.idunidadnegocio = %d, want %d", captured.Payload.IDUnidadNegocio, 1)
	}
	if captured.Payload.Cuenta != "7000880" {
		t.Fatalf("payload.cuenta = %q, want %q", captured.Payload.Cuenta, "7000880")
	}
	if captured.Payload.Identificacion != "901599500" {
		t.Fatalf("payload.identificacion = %q, want %q", captured.Payload.Identificacion, "901599500")
	}
	if len(captured.Payload.Unidades) != 1 {
		t.Fatalf("len(payload.unidades) = %d, want 1", len(captured.Payload.Unidades))
	}
	if captured.Payload.Unidades[0].NumeroUnidades != 1 {
		t.Fatalf("payload.unidades[0].numerounidades = %d, want %d", captured.Payload.Unidades[0].NumeroUnidades, 1)
	}
	if captured.Payload.Unidades[0].PesoVolumen != 3.6 {
		t.Fatalf("payload.unidades[0].pesovolumen = %v, want %v", captured.Payload.Unidades[0].PesoVolumen, 3.6)
	}
}

// TestQuoteRejected verifies provider rejection behavior.
func TestQuoteRejected(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{"codigoResultado":"-1","mensajeResultado":"INVALID"}`))
	}))
	defer server.Close()

	client, err := NewClient(Config{BaseURL: server.URL, AccessToken: "token", Account: "7000880", Identifier: "901599500"})
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	_, err = client.Quote(context.Background(), domain.QuoteRequest{
		Carrier:             domain.CarrierTCC,
		BusinessUnit:        domain.BusinessUnitLocals,
		OriginCityCode:      "05001",
		DestinationCityCode: "11001",
		DeclaredValue:       1,
		Units:               []domain.QuoteUnit{{Number: 1, RealWeight: 1, Height: 1, Width: 1, Length: 1}},
	})
	if !errors.Is(err, domain.ErrQuoteRejected) {
		t.Fatalf("Quote() error = %v, want %v", err, domain.ErrQuoteRejected)
	}
}

// TestQuoteIntegrationUnavailable verifies transport failure behavior.
func TestQuoteIntegrationUnavailable(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("net.Listen() error = %v", err)
	}
	address := listener.Addr().String()
	_ = listener.Close()

	client, err := NewClient(Config{BaseURL: "http://" + address, AccessToken: "token", Account: "7000880", Identifier: "901599500"})
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	_, err = client.Quote(context.Background(), domain.QuoteRequest{
		Carrier:             domain.CarrierTCC,
		BusinessUnit:        domain.BusinessUnitCourier,
		OriginCityCode:      "05001",
		DestinationCityCode: "11001",
		DeclaredValue:       1,
		Units:               []domain.QuoteUnit{{Number: 1, RealWeight: 1, Height: 1, Width: 1, Length: 1}},
	})
	if !errors.Is(err, domain.ErrIntegrationUnavailable) {
		t.Fatalf("Quote() error = %v, want %v", err, domain.ErrIntegrationUnavailable)
	}
}

// BenchmarkCalculateVolumetricWeight benchmarks volumetric-weight calculation behavior.
func BenchmarkCalculateVolumetricWeight(b *testing.B) {
	unit := domain.QuoteUnit{Number: 1, RealWeight: 2, Height: 15, Width: 20, Length: 30}

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = calculateVolumetricWeight(unit)
	}
}
