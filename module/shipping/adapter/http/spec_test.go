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
	if paths.Find("/shipping/batches/{id}/manifest-document") == nil {
		t.Fatalf("missing /shipping/batches/{id}/manifest-document path")
	}
	if paths.Find("/shipping/orders/{orderID}/dispatch") == nil {
		t.Fatalf("missing /shipping/orders/{orderID}/dispatch path")
	}
	if paths.Find("/shipping/marks/{id}/related") == nil {
		t.Fatalf("missing /shipping/marks/{id}/related path")
	}
}

// TestShippingOperationsExposeSchemas verifies shipping operations expose request/response schemas.
func TestShippingOperationsExposeSchemas(t *testing.T) {
	paths := Paths()
	postQuotation := paths.Find("/shipping/quotations").Post
	if postQuotation == nil || postQuotation.RequestBody == nil {
		t.Fatalf("expected /shipping/quotations POST request body")
	}
	if postQuotationRequestContent := postQuotation.RequestBody.Value.Content.Get("application/json"); postQuotationRequestContent == nil || postQuotationRequestContent.Schema == nil || postQuotationRequestContent.Schema.Value == nil {
		t.Fatalf("expected /shipping/quotations POST JSON request schema")
	} else if postQuotationRequestContent.Schema.Value.Properties["collectOnDeliveryAmount"] == nil {
		t.Fatalf("expected collectOnDeliveryAmount in quotation request schema")
	}
	if postQuotation.Responses == nil || postQuotation.Responses.Value("201") == nil {
		t.Fatalf("expected /shipping/quotations POST 201 response")
	}
	postQuotationResponse := postQuotation.Responses.Value("201")
	if postQuotationResponse.Value == nil || postQuotationResponse.Value.Content.Get("application/json") == nil {
		t.Fatalf("expected /shipping/quotations POST 201 JSON schema")
	}
	postQuotationSchema := postQuotationResponse.Value.Content.Get("application/json").Schema
	if postQuotationSchema == nil || postQuotationSchema.Value == nil {
		t.Fatalf("expected /shipping/quotations POST 201 schema object")
	}
	if postQuotationSchema.Value.Properties["freightCost"] == nil {
		t.Fatalf("expected freightCost in quotation response schema")
	}
	if postQuotationSchema.Value.Properties["collectOnDeliveryAmount"] == nil {
		t.Fatalf("expected collectOnDeliveryAmount in quotation response schema")
	}
	if postQuotationSchema.Value.Properties["collectOnDeliveryFeePercent"] == nil {
		t.Fatalf("expected collectOnDeliveryFeePercent in quotation response schema")
	}
	if postQuotationSchema.Value.Properties["collectOnDeliveryFeeAmount"] == nil {
		t.Fatalf("expected collectOnDeliveryFeeAmount in quotation response schema")
	}
	if postQuotationSchema.Value.Properties["collectOnDeliveryChargedAmount"] == nil {
		t.Fatalf("expected collectOnDeliveryChargedAmount in quotation response schema")
	}

	postMark := paths.Find("/shipping/marks").Post
	if postMark == nil || postMark.RequestBody == nil {
		t.Fatalf("expected /shipping/marks POST request body")
	}
	if postMark.Responses == nil || postMark.Responses.Value("201") == nil {
		t.Fatalf("expected /shipping/marks POST 201 response")
	}
	postMarkResponse := postMark.Responses.Value("201")
	if postMarkResponse == nil || postMarkResponse.Value == nil || postMarkResponse.Value.Content.Get("application/json") == nil {
		t.Fatalf("expected /shipping/marks POST 201 JSON schema")
	}
	postMarkSchema := postMarkResponse.Value.Content.Get("application/json").Schema
	if postMarkSchema == nil || postMarkSchema.Value == nil {
		t.Fatalf("expected /shipping/marks POST 201 schema object")
	}
	if postMarkSchema.Value.Properties["collectOnDeliveryAmount"] == nil {
		t.Fatalf("expected collectOnDeliveryAmount in mark response schema")
	}
	if postMarkSchema.Value.Properties["collectOnDeliveryChargedAmount"] == nil {
		t.Fatalf("expected collectOnDeliveryChargedAmount in mark response schema")
	}
	if postMarkSchema.Value.Properties["collectOnDeliveryFeePercent"] == nil {
		t.Fatalf("expected collectOnDeliveryFeePercent in mark response schema")
	}
	if postMarkSchema.Value.Properties["manifestType"] == nil {
		t.Fatalf("expected manifestType in mark response schema")
	}
	if postMarkSchema.Value.Properties["manifestRef"] == nil {
		t.Fatalf("expected manifestRef in mark response schema")
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
	if postBatchMarks.Responses == nil || postBatchMarks.Responses.Value("201") == nil {
		t.Fatalf("expected /shipping/batches/{id}/marks POST 201 response")
	}

	patchVoid := paths.Find("/shipping/marks/{id}/void").Patch
	if patchVoid == nil || patchVoid.RequestBody == nil {
		t.Fatalf("expected /shipping/marks/{id}/void PATCH request body")
	}
	if patchVoid.Responses == nil || patchVoid.Responses.Value("200") == nil {
		t.Fatalf("expected /shipping/marks/{id}/void PATCH 200 response")
	}
	getRelated := paths.Find("/shipping/marks/{id}/related").Get
	if getRelated == nil || getRelated.Responses == nil || getRelated.Responses.Value("200") == nil {
		t.Fatalf("expected /shipping/marks/{id}/related GET 200 response")
	}
	getRelatedResponse := getRelated.Responses.Value("200")
	if getRelatedResponse.Value == nil || getRelatedResponse.Value.Content.Get("application/json") == nil {
		t.Fatalf("expected /shipping/marks/{id}/related GET 200 JSON schema")
	}

	getTracking := paths.Find("/shipping/tracking/{trackingNumber}").Get
	if getTracking == nil || getTracking.Responses == nil || getTracking.Responses.Value("200") == nil {
		t.Fatalf("expected /shipping/tracking/{trackingNumber} GET 200 response")
	}
	getTrackingResponse := getTracking.Responses.Value("200")
	if getTrackingResponse.Value == nil || getTrackingResponse.Value.Content.Get("application/json") == nil {
		t.Fatalf("expected /shipping/tracking/{trackingNumber} GET 200 JSON schema")
	}

	getBatchManifestDocument := paths.Find("/shipping/batches/{id}/manifest-document").Get
	if getBatchManifestDocument == nil || getBatchManifestDocument.Responses == nil || getBatchManifestDocument.Responses.Value("200") == nil {
		t.Fatalf("expected /shipping/batches/{id}/manifest-document GET 200 response")
	}
	getBatchManifestDocumentResponse := getBatchManifestDocument.Responses.Value("200")
	if getBatchManifestDocumentResponse.Value == nil || getBatchManifestDocumentResponse.Value.Content.Get("application/pdf") == nil {
		t.Fatalf("expected /shipping/batches/{id}/manifest-document GET 200 application/pdf response")
	}
}
