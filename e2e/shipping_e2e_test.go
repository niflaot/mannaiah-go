package e2e_test

import (
	"net/http"
	"testing"

	"mannaiah/module/shipping"
)

// TestShippingManualFlowE2E verifies shipping manual-carrier flows end-to-end through HTTP.
func TestShippingManualFlowE2E(t *testing.T) {
	harness := newContactsE2EHarness(t)
	defer harness.Close(t)

	harness.tracer.Step("initialize shipping module")
	shippingModule, err := shipping.New(shipping.Config{Enabled: true}, harness.db)
	if err != nil {
		t.Fatalf("shipping.New() error = %v", err)
	}
	shippingModule.SetAuthorizer(harness.authModule)
	harness.server.RegisterRoutes(shippingModule.RegisterRoutes)

	manageToken := harness.SignToken(t, "shipping:manage")

	harness.tracer.Step("request carriers without authorization header")
	status, payload := harness.DoJSONRequest(t, http.MethodGet, "/shipping/carriers", "", nil)
	if status != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", status, http.StatusUnauthorized)
	}
	if payload["message"] != "unauthorized" {
		t.Fatalf("payload.message = %v, want %q", payload["message"], "unauthorized")
	}

	harness.tracer.Step("list available carriers")
	status, payload = harness.DoJSONRequest(t, http.MethodGet, "/shipping/carriers", manageToken, nil)
	if status != http.StatusOK {
		t.Fatalf("status = %d, want %d", status, http.StatusOK)
	}
	rows, ok := payload["data"].([]any)
	if !ok || len(rows) == 0 {
		t.Fatalf("payload.data = %v, want at least one carrier", payload["data"])
	}

	harness.tracer.Step("request quotation with manual carrier and assert controlled not-supported error")
	status, payload = harness.DoJSONRequest(t, http.MethodPost, "/shipping/quotations", manageToken, []byte(`{
	  "orderId":"order-shipping-1",
	  "carrierId":"manual",
	  "originCityCode":"11001000",
	  "destCityCode":"76001000",
	  "declaredValue":120000,
	  "shipmentMode":"parcel",
	  "units":[{"description":"box","packageType":"CAJA","dimensions":{"heightCm":15,"widthCm":20,"depthCm":30,"realWeightKg":2.2}}]
	}`))
	if status != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", status, http.StatusBadRequest)
	}
	if payload["message"] != "quotation_not_supported" {
		t.Fatalf("payload.message = %v, want %q", payload["message"], "quotation_not_supported")
	}

	harness.tracer.Step("create manual shipping mark")
	status, payload = harness.DoJSONRequest(t, http.MethodPost, "/shipping/marks", manageToken, []byte(`{
	  "orderId":"order-shipping-1",
	  "carrierId":"manual",
	  "sender":{"name":"Flock","id":"901599500","idType":"NIT","addressLine":"Sender street 123","cityCode":"11001000","phone":"3000000000","email":"contacto@flockstore.co"},
	  "recipient":{"name":"Marylu","id":"83395cf06d6837104f19a7c9a99a2517","idType":"CC","addressLine":"Recipient street 456","cityCode":"76001000","phone":"3110000000","email":"coccostoreco@gmail.com"},
	  "paymentForm":"1",
	  "declaredValue":162000,
	  "shipmentMode":"parcel",
	  "units":[{"description":"morral","packageType":"CAJA","dimensions":{"heightCm":20,"widthCm":18,"depthCm":12,"realWeightKg":1.4}}]
	}`))
	if status != http.StatusCreated {
		t.Fatalf("status = %d, want %d", status, http.StatusCreated)
	}
	markID, _ := payload["id"].(string)
	if markID == "" {
		t.Fatalf("expected mark id")
	}
	trackingNumber, _ := payload["trackingNumber"].(string)
	if trackingNumber == "" {
		t.Fatalf("expected tracking number")
	}
	if payload["status"] != "GENERATED" {
		t.Fatalf("payload.status = %v, want %q", payload["status"], "GENERATED")
	}

	harness.tracer.Step("create second manual shipping mark for same order")
	status, payload = harness.DoJSONRequest(t, http.MethodPost, "/shipping/marks", manageToken, []byte(`{
	  "orderId":"order-shipping-1",
	  "carrierId":"manual",
	  "sender":{"name":"Flock","id":"901599500","idType":"NIT","addressLine":"Sender street 123","cityCode":"11001000","phone":"3000000000","email":"contacto@flockstore.co"},
	  "recipient":{"name":"Marylu","id":"83395cf06d6837104f19a7c9a99a2517","idType":"CC","addressLine":"Recipient street 456","cityCode":"76001000","phone":"3110000000","email":"coccostoreco@gmail.com"},
	  "paymentForm":"1",
	  "declaredValue":147000,
	  "shipmentMode":"parcel",
	  "units":[{"description":"totebag","packageType":"CAJA","dimensions":{"heightCm":18,"widthCm":16,"depthCm":10,"realWeightKg":1.2}}]
	}`))
	if status != http.StatusCreated {
		t.Fatalf("status = %d, want %d", status, http.StatusCreated)
	}
	secondMarkID, _ := payload["id"].(string)
	if secondMarkID == "" {
		t.Fatalf("expected related mark id")
	}

	harness.tracer.Step("list related marks for first mark")
	status, payload = harness.DoJSONRequest(t, http.MethodGet, "/shipping/marks/"+markID+"/related", manageToken, nil)
	if status != http.StatusOK {
		t.Fatalf("status = %d, want %d", status, http.StatusOK)
	}
	relatedRows, ok := payload["data"].([]any)
	if !ok || len(relatedRows) == 0 {
		t.Fatalf("payload.data = %v, want at least one related mark", payload["data"])
	}
	foundSecond := false
	for _, row := range relatedRows {
		entity, _ := row.(map[string]any)
		if entity["id"] == secondMarkID {
			foundSecond = true
			break
		}
	}
	if !foundSecond {
		t.Fatalf("related mark list should include second mark id %q, got %v", secondMarkID, relatedRows)
	}
	if payload["total"] == nil {
		t.Fatalf("payload.total should be present")
	}

	harness.tracer.Step("create dispatch batch")
	status, payload = harness.DoJSONRequest(t, http.MethodPost, "/shipping/batches", manageToken, []byte(`{"name":"Dispatch 2026-03-22 #1","carrierId":"manual"}`))
	if status != http.StatusCreated {
		t.Fatalf("status = %d, want %d", status, http.StatusCreated)
	}
	batchID, _ := payload["id"].(string)
	if batchID == "" {
		t.Fatalf("expected batch id")
	}

	harness.tracer.Step("create draft mark in dispatch batch")
	status, payload = harness.DoJSONRequest(t, http.MethodPost, "/shipping/batches/"+batchID+"/marks", manageToken, []byte(`{
	  "orderId":"order-shipping-2",
	  "sender":{"name":"Flock","id":"901599500","idType":"NIT","addressLine":"Sender street 123","cityCode":"11001000","phone":"3000000000","email":"contacto@flockstore.co"},
	  "recipient":{"name":"Marylu","id":"83395cf06d6837104f19a7c9a99a2517","idType":"CC","addressLine":"Recipient street 456","cityCode":"76001000","phone":"3110000000","email":"coccostoreco@gmail.com"},
	  "declaredValue":162000,
	  "trackingNumber":"MANUAL-ORDER-2",
	  "customTrackingUrl":"https://rastreo.flockstore.co/guide/MANUAL-ORDER-2",
	  "shipmentMode":"parcel",
	  "units":[{"description":"morral","packageType":"CAJA","dimensions":{"heightCm":20,"widthCm":18,"depthCm":12,"realWeightKg":1.4}}]
	}`))
	if status != http.StatusCreated {
		t.Fatalf("status = %d, want %d", status, http.StatusCreated)
	}
	if payload["status"] != "QUOTED" {
		t.Fatalf("payload.status = %v, want %q", payload["status"], "QUOTED")
	}

	harness.tracer.Step("close dispatch batch")
	status, payload = harness.DoJSONRequest(t, http.MethodPatch, "/shipping/batches/"+batchID+"/close", manageToken, nil)
	if status != http.StatusOK {
		t.Fatalf("status = %d, want %d", status, http.StatusOK)
	}
	if payload["status"] != "CLOSED" {
		t.Fatalf("payload.status = %v, want %q", payload["status"], "CLOSED")
	}

	harness.tracer.Step("verify manual draft mark preserves tracking fields after batch close")
	status, payload = harness.DoJSONRequest(t, http.MethodGet, "/shipping/marks?orderID=order-shipping-2", manageToken, nil)
	if status != http.StatusOK {
		t.Fatalf("status = %d, want %d", status, http.StatusOK)
	}
	rows, ok = payload["data"].([]any)
	if !ok || len(rows) == 0 {
		t.Fatalf("payload.data = %v, want at least one mark", payload["data"])
	}
	markPayload, _ := rows[0].(map[string]any)
	if markPayload["trackingNumber"] != "MANUAL-ORDER-2" {
		t.Fatalf("payload.trackingNumber = %v, want %q", markPayload["trackingNumber"], "MANUAL-ORDER-2")
	}
	if markPayload["customTrackingUrl"] != "https://rastreo.flockstore.co/guide/MANUAL-ORDER-2" {
		t.Fatalf("payload.customTrackingUrl = %v, want %q", markPayload["customTrackingUrl"], "https://rastreo.flockstore.co/guide/MANUAL-ORDER-2")
	}

	harness.tracer.Step("resolve manual tracking history")
	status, payload = harness.DoJSONRequest(t, http.MethodGet, "/shipping/tracking/"+trackingNumber+"?carrier=manual", manageToken, nil)
	if status != http.StatusOK {
		t.Fatalf("status = %d, want %d", status, http.StatusOK)
	}
	if payload["globalStatus"] != "PROCESSING" {
		t.Fatalf("payload.globalStatus = %v, want %q", payload["globalStatus"], "PROCESSING")
	}

	harness.tracer.Step("check order-shipping-1 dispatch provisioning status")
	status, payload = harness.DoJSONRequest(t, http.MethodGet, "/shipping/orders/order-shipping-1/dispatch", manageToken, nil)
	if status != http.StatusOK {
		t.Fatalf("status = %d, want %d", status, http.StatusOK)
	}
	if payload["provisioned"] != true {
		t.Fatalf("payload.provisioned = %v, want true", payload["provisioned"])
	}
	if payload["orderId"] != "order-shipping-1" {
		t.Fatalf("payload.orderId = %v, want %q", payload["orderId"], "order-shipping-1")
	}

	harness.tracer.Step("check order-shipping-2 dispatch provisioning status")
	status, payload = harness.DoJSONRequest(t, http.MethodGet, "/shipping/orders/order-shipping-2/dispatch", manageToken, nil)
	if status != http.StatusOK {
		t.Fatalf("status = %d, want %d", status, http.StatusOK)
	}
	if payload["provisioned"] != true {
		t.Fatalf("payload.provisioned = %v, want true", payload["provisioned"])
	}

	harness.tracer.Step("assert e2e trace logs")
	harness.tracer.AssertStepCount(14)
}
