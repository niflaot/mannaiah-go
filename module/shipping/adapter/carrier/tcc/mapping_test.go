package tcc

import (
	"encoding/json"
	"testing"
	"time"

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
	if status := MapTrackingStatus("4000", "En devolucion"); status != domain.TrackingStatusReturn {
		t.Fatalf("status = %q", status)
	}
	if status := MapTrackingStatus("4100", "En continuacion"); status != domain.TrackingStatusProcessing {
		t.Fatalf("status = %q", status)
	}
	if status := MapTrackingStatus("4200", "Reemplazada"); status != domain.TrackingStatusIncidence {
		t.Fatalf("status = %q", status)
	}
	if status := MapTrackingStatus("4300", "Anulada"); status != domain.TrackingStatusVoided {
		t.Fatalf("status = %q", status)
	}
	if status := MapTrackingStatus("9999", "Novedad reportada"); status != domain.TrackingStatusIncidence {
		t.Fatalf("status = %q", status)
	}
	if status := MapTrackingStatus("9999", "Devolucion a origen"); status != domain.TrackingStatusReturn {
		t.Fatalf("status = %q", status)
	}
}

// TestMapTrackingNoveltyStatus verifies TCC novelty mapping behavior.
func TestMapTrackingNoveltyStatus(t *testing.T) {
	if status := MapTrackingNoveltyStatus("252", "MERCANCÍA NO ENTREGADA A DESTINATARIO"); status != domain.TrackingStatusIncidence {
		t.Fatalf("status = %q", status)
	}
	if status := MapTrackingNoveltyStatus("4000", "Mercancía en procesos de devolución a origen"); status != domain.TrackingStatusReturn {
		t.Fatalf("status = %q", status)
	}
	if status := MapTrackingNoveltyStatus("4300", "Mercancía no despachada por el remitente"); status != domain.TrackingStatusVoided {
		t.Fatalf("status = %q", status)
	}
	if status := MapTrackingNoveltyStatus("9999", "MERCANCÍA NO ENTREGADA A DESTINATARIO"); status != domain.TrackingStatusIncidence {
		t.Fatalf("status = %q", status)
	}
}

// TestParseTrackingDate verifies Spanish AM/PM tracking dates are parsed correctly.
func TestParseTrackingDate(t *testing.T) {
	parsed := ParseTrackingDate("8/04/2026 7:23:39 p. m.")
	if parsed.IsZero() {
		t.Fatal("parsed is zero")
	}
	if parsed.Hour() != 0 || parsed.Day() != 9 {
		t.Fatalf("parsed = %s", parsed.Format(time.RFC3339))
	}
}

// TestTrackingResponseUnmarshalWithNovelties verifies real TCC novelty payloads decode correctly.
func TestTrackingResponseUnmarshalWithNovelties(t *testing.T) {
	raw := []byte(`{
		"remesas": [
			{
				"numeroremesa": "615193194",
				"novedad": {
					"codigonovedad": "121811087",
					"observaciones": "El destinatario no se encuentra, pasar nuevamente mañana",
					"complementonovedad": "NO SE ENCUENTRA EL DESTINATARIO O NO HAY NADIE EN LA DIRECCIÓN",
					"estadonovedad": "EJECUTADA",
					"fechanovedad": "8/04/2026 7:23:39 p. m.",
					"idtiponovedad": "252",
					"novedad": "MERCANCÍA NO ENTREGADA A DESTINATARIO"
				},
				"estados": [
					{
						"codigo": "2000",
						"descripcion": "Envío en proceso de entrega",
						"fecha": "8/04/2026 10:08:41 a. m."
					}
				],
				"novedades": [
					{
						"codigo": "252",
						"fecha": "8/04/2026 7:23:39 p. m.",
						"descripcion": "MERCANCÍA NO ENTREGADA A DESTINATARIO - NO SE ENCUENTRA EL DESTINATARIO O NO HAY NADIE EN LA DIRECCIÓN",
						"estado": "Ejecutada",
						"observacion": "El destinatario no se encuentra, pasar nuevamente mañana",
						"esrechazo": "False",
						"fecharechazo": null
					}
				],
				"ciudadorigen": { "descripcion": "MEDELLIN" },
				"ciudaddestino": { "descripcion": "BOGOTA" }
			}
		],
		"respuesta": { "codigo": "0", "mensaje": "OK" }
	}`)

	var response TrackingResponse
	if err := json.Unmarshal(raw, &response); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if len(response.Remittances) != 1 {
		t.Fatalf("len(response.Remittances) = %d", len(response.Remittances))
	}
	if len(response.Remittances[0].Novelties) != 1 {
		t.Fatalf("len(response.Remittances[0].Novelties) = %d", len(response.Remittances[0].Novelties))
	}
	if response.Remittances[0].Novelties[0].Code != "252" {
		t.Fatalf("response.Remittances[0].Novelties[0].Code = %q", response.Remittances[0].Novelties[0].Code)
	}
	if response.Remittances[0].Novelty.Code != "252" {
		t.Fatalf("response.Remittances[0].Novelty.Code = %q", response.Remittances[0].Novelty.Code)
	}
}
