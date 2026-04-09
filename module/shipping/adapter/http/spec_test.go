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
	if paths.Find("/shipping/marks/{id}/rotulus-document") == nil {
		t.Fatalf("missing /shipping/marks/{id}/rotulus-document path")
	}
	if paths.Find("/shipping/batches/marks") == nil {
		t.Fatalf("missing /shipping/batches/marks path")
	}
	if paths.Find("/shipping/batches/{id}/marks/{markID}") == nil {
		t.Fatalf("missing /shipping/batches/{id}/marks/{markID} path")
	}
	if paths.Find("/shipping/quotations/order-packaging") == nil {
		t.Fatalf("missing /shipping/quotations/order-packaging path")
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

	postQuotationFromOrder := paths.Find("/shipping/quotations/order").Post
	if postQuotationFromOrder == nil || postQuotationFromOrder.Responses == nil {
		t.Fatalf("expected /shipping/quotations/order POST responses")
	}
	for _, status := range []string{"400", "401", "403", "404", "500"} {
		errorResponse := postQuotationFromOrder.Responses.Value(status)
		if errorResponse == nil || errorResponse.Value == nil {
			t.Fatalf("expected /shipping/quotations/order POST %s response", status)
		}
		content := errorResponse.Value.Content.Get("application/json")
		if content == nil || content.Schema == nil || content.Schema.Value == nil {
			t.Fatalf("expected /shipping/quotations/order POST %s JSON error schema", status)
		}
		if content.Schema.Value.Properties["message"] == nil || content.Schema.Value.Properties["error"] == nil {
			t.Fatalf("expected /shipping/quotations/order POST %s error schema properties", status)
		}
	}
	postOrderPackaging := paths.Find("/shipping/quotations/order-packaging").Post
	if postOrderPackaging == nil || postOrderPackaging.RequestBody == nil {
		t.Fatalf("expected /shipping/quotations/order-packaging POST request body")
	}
	if postOrderPackaging.Responses == nil || postOrderPackaging.Responses.Value("200") == nil {
		t.Fatalf("expected /shipping/quotations/order-packaging POST 200 response")
	}
	orderPackagingResponse := postOrderPackaging.Responses.Value("200")
	if orderPackagingResponse == nil || orderPackagingResponse.Value == nil || orderPackagingResponse.Value.Content.Get("application/json") == nil {
		t.Fatalf("expected /shipping/quotations/order-packaging POST 200 JSON schema")
	}
	orderPackagingSchema := orderPackagingResponse.Value.Content.Get("application/json").Schema
	if orderPackagingSchema == nil || orderPackagingSchema.Value == nil {
		t.Fatalf("expected /shipping/quotations/order-packaging POST 200 schema object")
	}
	if orderPackagingSchema.Value.Properties["units"] == nil {
		t.Fatalf("expected units in order packaging response schema")
	}
	if orderPackagingSchema.Value.Properties["shipmentMode"] == nil {
		t.Fatalf("expected shipmentMode in order packaging response schema")
	}

	getOrderQuotation := paths.Find("/shipping/quotations/order/{identifier}").Get
	if getOrderQuotation == nil || getOrderQuotation.Responses == nil {
		t.Fatalf("expected /shipping/quotations/order/{identifier} GET responses")
	}
	for _, status := range []string{"400", "401", "403", "404", "500"} {
		errorResponse := getOrderQuotation.Responses.Value(status)
		if errorResponse == nil || errorResponse.Value == nil {
			t.Fatalf("expected /shipping/quotations/order/{identifier} GET %s response", status)
		}
		content := errorResponse.Value.Content.Get("application/json")
		if content == nil || content.Schema == nil || content.Schema.Value == nil {
			t.Fatalf("expected /shipping/quotations/order/{identifier} GET %s JSON error schema", status)
		}
		if content.Schema.Value.Properties["message"] == nil || content.Schema.Value.Properties["error"] == nil {
			t.Fatalf("expected /shipping/quotations/order/{identifier} GET %s error schema properties", status)
		}
	}

	postMark := paths.Find("/shipping/marks").Post
	if postMark == nil || postMark.RequestBody == nil {
		t.Fatalf("expected /shipping/marks POST request body")
	}
	postMarkRequest := postMark.RequestBody.Value.Content.Get("application/json")
	if postMarkRequest == nil || postMarkRequest.Schema == nil || postMarkRequest.Schema.Value == nil {
		t.Fatalf("expected /shipping/marks POST JSON request schema")
	}
	if postMarkRequest.Schema.Value.Properties["customTrackingUrl"] == nil {
		t.Fatalf("expected customTrackingUrl in /shipping/marks POST request schema")
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
	if postMarkSchema.Value.Properties["draftSnapshot"] == nil {
		t.Fatalf("expected draftSnapshot in mark response schema")
	}
	if postMarkSchema.Value.Properties["responseSnapshot"] == nil {
		t.Fatalf("expected responseSnapshot in mark response schema")
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
	postBatchMarksRequest := postBatchMarks.RequestBody.Value.Content.Get("application/json")
	if postBatchMarksRequest == nil || postBatchMarksRequest.Schema == nil || postBatchMarksRequest.Schema.Value == nil {
		t.Fatalf("expected /shipping/batches/{id}/marks POST JSON request schema")
	}
	if postBatchMarksRequest.Schema.Value.Properties["trackingNumber"] == nil {
		t.Fatalf("expected trackingNumber in /shipping/batches/{id}/marks POST request schema")
	}
	if postBatchMarksRequest.Schema.Value.Properties["customTrackingUrl"] == nil {
		t.Fatalf("expected customTrackingUrl in /shipping/batches/{id}/marks POST request schema")
	}
	if postBatchMarks.Responses == nil || postBatchMarks.Responses.Value("201") == nil {
		t.Fatalf("expected /shipping/batches/{id}/marks POST 201 response")
	}
	postCreateBatchMark := paths.Find("/shipping/batches/marks").Post
	if postCreateBatchMark == nil || postCreateBatchMark.RequestBody == nil {
		t.Fatalf("expected /shipping/batches/marks POST request body")
	}
	postCreateBatchMarkRequest := postCreateBatchMark.RequestBody.Value.Content.Get("application/json")
	if postCreateBatchMarkRequest == nil || postCreateBatchMarkRequest.Schema == nil || postCreateBatchMarkRequest.Schema.Value == nil {
		t.Fatalf("expected /shipping/batches/marks POST JSON request schema")
	}
	if postCreateBatchMarkRequest.Schema.Value.Properties["batch"] == nil {
		t.Fatalf("expected batch in /shipping/batches/marks POST request schema")
	}
	if postCreateBatchMarkRequest.Schema.Value.Properties["direct"] == nil {
		t.Fatalf("expected direct in /shipping/batches/marks POST request schema")
	}
	if postCreateBatchMarkRequest.Schema.Value.Properties["quotationId"] == nil {
		t.Fatalf("expected quotationId in /shipping/batches/marks POST request schema")
	}
	if postCreateBatchMark.Responses == nil || postCreateBatchMark.Responses.Value("201") == nil {
		t.Fatalf("expected /shipping/batches/marks POST 201 response")
	}
	patchBatchMark := paths.Find("/shipping/batches/{id}/marks/{markID}").Patch
	if patchBatchMark == nil || patchBatchMark.RequestBody == nil {
		t.Fatalf("expected /shipping/batches/{id}/marks/{markID} PATCH request body")
	}
	patchBatchMarkRequest := patchBatchMark.RequestBody.Value.Content.Get("application/json")
	if patchBatchMarkRequest == nil || patchBatchMarkRequest.Schema == nil || patchBatchMarkRequest.Schema.Value == nil {
		t.Fatalf("expected /shipping/batches/{id}/marks/{markID} PATCH JSON request schema")
	}
	if patchBatchMarkRequest.Schema.Value.Properties["quotedFreightCost"] == nil {
		t.Fatalf("expected quotedFreightCost in /shipping/batches/{id}/marks/{markID} PATCH request schema")
	}
	if patchBatchMarkRequest.Schema.Value.Properties["observations"] == nil {
		t.Fatalf("expected observations in /shipping/batches/{id}/marks/{markID} PATCH request schema")
	}
	if patchBatchMarkRequest.Schema.Value.Properties["trackingNumber"] == nil {
		t.Fatalf("expected trackingNumber in /shipping/batches/{id}/marks/{markID} PATCH request schema")
	}
	if patchBatchMarkRequest.Schema.Value.Properties["customTrackingUrl"] == nil {
		t.Fatalf("expected customTrackingUrl in /shipping/batches/{id}/marks/{markID} PATCH request schema")
	}
	if patchBatchMark.Responses == nil || patchBatchMark.Responses.Value("200") == nil {
		t.Fatalf("expected /shipping/batches/{id}/marks/{markID} PATCH 200 response")
	}
	for _, status := range []string{"400", "401", "403", "404", "409", "500"} {
		errorResponse := patchBatchMark.Responses.Value(status)
		if errorResponse == nil || errorResponse.Value == nil {
			t.Fatalf("expected /shipping/batches/{id}/marks/{markID} PATCH %s response", status)
		}
		content := errorResponse.Value.Content.Get("application/json")
		if content == nil || content.Schema == nil || content.Schema.Value == nil {
			t.Fatalf("expected /shipping/batches/{id}/marks/{markID} PATCH %s JSON error schema", status)
		}
		if content.Schema.Value.Properties["message"] == nil || content.Schema.Value.Properties["error"] == nil {
			t.Fatalf("expected /shipping/batches/{id}/marks/{markID} PATCH %s error schema properties", status)
		}
	}
	for _, status := range []string{"400", "401", "403", "404", "409", "500"} {
		errorResponse := postCreateBatchMark.Responses.Value(status)
		if errorResponse == nil || errorResponse.Value == nil {
			t.Fatalf("expected /shipping/batches/marks POST %s response", status)
		}
		content := errorResponse.Value.Content.Get("application/json")
		if content == nil || content.Schema == nil || content.Schema.Value == nil {
			t.Fatalf("expected /shipping/batches/marks POST %s JSON error schema", status)
		}
		if content.Schema.Value.Properties["message"] == nil || content.Schema.Value.Properties["error"] == nil {
			t.Fatalf("expected /shipping/batches/marks POST %s error schema properties", status)
		}
	}
	patchCloseBatch := paths.Find("/shipping/batches/{id}/close").Patch
	if patchCloseBatch == nil || patchCloseBatch.Responses == nil {
		t.Fatalf("expected /shipping/batches/{id}/close PATCH responses")
	}
	for _, status := range []string{"400", "401", "403", "404", "409", "500"} {
		errorResponse := patchCloseBatch.Responses.Value(status)
		if errorResponse == nil || errorResponse.Value == nil {
			t.Fatalf("expected /shipping/batches/{id}/close PATCH %s response", status)
		}
		content := errorResponse.Value.Content.Get("application/json")
		if content == nil || content.Schema == nil || content.Schema.Value == nil {
			t.Fatalf("expected /shipping/batches/{id}/close PATCH %s JSON error schema", status)
		}
		if content.Schema.Value.Properties["message"] == nil || content.Schema.Value.Properties["error"] == nil {
			t.Fatalf("expected /shipping/batches/{id}/close PATCH %s error schema properties", status)
		}
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
	getMarkRotulusDocument := paths.Find("/shipping/marks/{id}/rotulus-document").Get
	if getMarkRotulusDocument == nil || getMarkRotulusDocument.Responses == nil || getMarkRotulusDocument.Responses.Value("200") == nil {
		t.Fatalf("expected /shipping/marks/{id}/rotulus-document GET 200 response")
	}
	getMarkRotulusDocumentResponse := getMarkRotulusDocument.Responses.Value("200")
	if getMarkRotulusDocumentResponse.Value == nil || getMarkRotulusDocumentResponse.Value.Content.Get("application/pdf") == nil {
		t.Fatalf("expected /shipping/marks/{id}/rotulus-document GET 200 application/pdf response")
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
