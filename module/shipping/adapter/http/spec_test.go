package http

import "testing"

// TestPaths verifies shipping path coverage.
func TestPaths(t *testing.T) {
	paths := Paths()
	if paths == nil {
		t.Fatalf("Paths() returned nil")
	}
	if paths.Find("/shipping/marks") == nil {
		t.Fatalf("missing /shipping/marks path")
	}
	if paths.Find("/shipping/batches/{id}/close") == nil {
		t.Fatalf("missing /shipping/batches/{id}/close path")
	}
}

// TestShippingOperationsExposeSchemas verifies shipping operations expose request/response schemas.
func TestShippingOperationsExposeSchemas(t *testing.T) {
	paths := Paths()
	postQuotation := paths.Find("/shipping/quotations").Post
	if postQuotation == nil || postQuotation.RequestBody == nil {
		t.Fatalf("expected /shipping/quotations POST request body")
	}
	if postQuotation.Responses == nil || postQuotation.Responses.Value("201") == nil {
		t.Fatalf("expected /shipping/quotations POST 201 response")
	}
	postQuotationResponse := postQuotation.Responses.Value("201")
	if postQuotationResponse.Value == nil || postQuotationResponse.Value.Content.Get("application/json") == nil {
		t.Fatalf("expected /shipping/quotations POST 201 JSON schema")
	}

	postMark := paths.Find("/shipping/marks").Post
	if postMark == nil || postMark.RequestBody == nil {
		t.Fatalf("expected /shipping/marks POST request body")
	}
	if postMark.Responses == nil || postMark.Responses.Value("201") == nil {
		t.Fatalf("expected /shipping/marks POST 201 response")
	}

	postBatch := paths.Find("/shipping/batches").Post
	if postBatch == nil || postBatch.RequestBody == nil {
		t.Fatalf("expected /shipping/batches POST request body")
	}
	if postBatch.Responses == nil || postBatch.Responses.Value("201") == nil {
		t.Fatalf("expected /shipping/batches POST 201 response")
	}

	postBatchMarks := paths.Find("/shipping/batches/{id}/marks").Post
	if postBatchMarks == nil || postBatchMarks.RequestBody == nil {
		t.Fatalf("expected /shipping/batches/{id}/marks POST request body")
	}
	if postBatchMarks.Responses == nil || postBatchMarks.Responses.Value("200") == nil {
		t.Fatalf("expected /shipping/batches/{id}/marks POST 200 response")
	}

	patchVoid := paths.Find("/shipping/marks/{id}/void").Patch
	if patchVoid == nil || patchVoid.RequestBody == nil {
		t.Fatalf("expected /shipping/marks/{id}/void PATCH request body")
	}
	if patchVoid.Responses == nil || patchVoid.Responses.Value("200") == nil {
		t.Fatalf("expected /shipping/marks/{id}/void PATCH 200 response")
	}

	getTracking := paths.Find("/shipping/tracking/{trackingNumber}").Get
	if getTracking == nil || getTracking.Responses == nil || getTracking.Responses.Value("200") == nil {
		t.Fatalf("expected /shipping/tracking/{trackingNumber} GET 200 response")
	}
	getTrackingResponse := getTracking.Responses.Value("200")
	if getTrackingResponse.Value == nil || getTrackingResponse.Value.Content.Get("application/json") == nil {
		t.Fatalf("expected /shipping/tracking/{trackingNumber} GET 200 JSON schema")
	}
}
