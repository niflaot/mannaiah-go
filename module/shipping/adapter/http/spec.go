package http

import "github.com/getkin/kin-openapi/openapi3"

const (
	// shippingTag defines shipping OpenAPI tag values.
	shippingTag = "shipping"
	// bearerSecurityScheme defines shipping security-scheme key values.
	bearerSecurityScheme = "shipping_bearer"
)

// Paths returns shipping OpenAPI path definitions.
func Paths() *openapi3.Paths {
	return openapi3.NewPaths(
		openapi3.WithPath("/shipping/quotations", &openapi3.PathItem{Post: quotationCreateOperation(), Get: quotationListOperation()}),
		openapi3.WithPath("/shipping/quotations/order", &openapi3.PathItem{Post: quotationFromOrderOperation()}),
		openapi3.WithPath("/shipping/quotations/order-packaging", &openapi3.PathItem{Post: quotationOrderPackagingOperation()}),
		openapi3.WithPath("/shipping/quotations/order/{identifier}", &openapi3.PathItem{Get: getOrderQuotationOperation()}),
		openapi3.WithPath("/shipping/marks", &openapi3.PathItem{Post: markCreateOperation(), Get: markListOperation()}),
		openapi3.WithPath("/shipping/marks/{id}", &openapi3.PathItem{Get: markGetOperation()}),
		openapi3.WithPath("/shipping/marks/{id}/related", &openapi3.PathItem{Get: markRelatedOperation()}),
		openapi3.WithPath("/shipping/marks/{id}/void", &openapi3.PathItem{Patch: markVoidOperation()}),
		openapi3.WithPath("/shipping/batches", &openapi3.PathItem{Post: batchCreateOperation(), Get: batchListOperation()}),
		openapi3.WithPath("/shipping/batches/{id}", &openapi3.PathItem{Get: batchGetOperation()}),
		openapi3.WithPath("/shipping/batches/{id}/marks", &openapi3.PathItem{Post: batchAddMarkOperation()}),
		openapi3.WithPath("/shipping/batches/marks", &openapi3.PathItem{Post: batchCreateMarkOperation()}),
		openapi3.WithPath("/shipping/batches/{id}/marks/{markID}", &openapi3.PathItem{Delete: batchRemoveMarkOperation()}),
		openapi3.WithPath("/shipping/batches/{id}/close", &openapi3.PathItem{Patch: batchCloseOperation()}),
		openapi3.WithPath("/shipping/batches/{id}/manifest-document", &openapi3.PathItem{Get: batchManifestDocumentOperation()}),
		openapi3.WithPath("/shipping/tracking/{trackingNumber}", &openapi3.PathItem{Get: trackingGetOperation()}),
		openapi3.WithPath("/shipping/carriers", &openapi3.PathItem{Get: carrierListOperation()}),
		openapi3.WithPath("/shipping/carriers/{id}", &openapi3.PathItem{Get: carrierGetOperation()}),
		openapi3.WithPath("/shipping/orders/{orderID}/dispatch", &openapi3.PathItem{Get: orderDispatchOperation()}),
	)
}

// SecuritySchemes returns OpenAPI security schemes used by shipping operations.
func SecuritySchemes() openapi3.SecuritySchemes {
	return openapi3.SecuritySchemes{
		bearerSecurityScheme: &openapi3.SecuritySchemeRef{Value: openapi3.NewJWTSecurityScheme()},
	}
}

// Tags returns shipping OpenAPI tags.
func Tags() openapi3.Tags {
	return openapi3.Tags{&openapi3.Tag{Name: shippingTag}}
}

// bearerSecurityRequirements builds bearer-auth operation security requirements.
func bearerSecurityRequirements() *openapi3.SecurityRequirements {
	return openapi3.NewSecurityRequirements().With(openapi3.NewSecurityRequirement().Authenticate(bearerSecurityScheme))
}

// schemaRef wraps one schema value with a schema reference.
func schemaRef(schema *openapi3.Schema) *openapi3.SchemaRef {
	return &openapi3.SchemaRef{Value: schema}
}

// jsonRequestBody builds one JSON request-body descriptor.
func jsonRequestBody(schema *openapi3.Schema, required bool) *openapi3.RequestBodyRef {
	return &openapi3.RequestBodyRef{Value: &openapi3.RequestBody{
		Required: required,
		Content: openapi3.Content{
			"application/json": &openapi3.MediaType{Schema: schemaRef(schema)},
		},
	}}
}

// jsonResponse builds one JSON response descriptor.
func jsonResponse(description string, schema *openapi3.Schema) *openapi3.ResponseRef {
	return &openapi3.ResponseRef{Value: openapi3.NewResponse().WithDescription(description).WithContent(openapi3.Content{
		"application/json": &openapi3.MediaType{Schema: schemaRef(schema)},
	})}
}

// errorResponse builds one JSON error response descriptor using the standard HTTP error payload.
func errorResponse(description string) *openapi3.ResponseRef {
	return jsonResponse(description, errorResponseSchema())
}

// binaryPDFResponse builds one binary PDF response descriptor.
func binaryPDFResponse(description string) *openapi3.ResponseRef {
	return &openapi3.ResponseRef{Value: openapi3.NewResponse().WithDescription(description).WithContent(openapi3.Content{
		"application/pdf": &openapi3.MediaType{},
	})}
}

// pathStringParameter builds one required string path parameter.
func pathStringParameter(name string, description string) *openapi3.ParameterRef {
	return &openapi3.ParameterRef{Value: &openapi3.Parameter{
		Name:        name,
		In:          "path",
		Required:    true,
		Description: description,
		Schema:      schemaRef(openapi3.NewStringSchema()),
	}}
}

// queryStringParameter builds one query string parameter.
func queryStringParameter(name string, description string, required bool) *openapi3.ParameterRef {
	return &openapi3.ParameterRef{Value: &openapi3.Parameter{
		Name:        name,
		In:          "query",
		Required:    required,
		Description: description,
		Schema:      schemaRef(openapi3.NewStringSchema()),
	}}
}

// queryIntParameter builds one query integer parameter.
func queryIntParameter(name string, description string, required bool) *openapi3.ParameterRef {
	return &openapi3.ParameterRef{Value: &openapi3.Parameter{
		Name:        name,
		In:          "query",
		Required:    required,
		Description: description,
		Schema:      schemaRef(openapi3.NewIntegerSchema()),
	}}
}

// quotationCreateOperation defines the OpenAPI operation for quotation creation.
func quotationCreateOperation() *openapi3.Operation {
	return &openapi3.Operation{
		OperationID: "ShippingController_createQuotation",
		Summary:     "Request one freight quotation",
		Tags:        []string{shippingTag},
		Security:    bearerSecurityRequirements(),
		RequestBody: jsonRequestBody(quotationRequestSchema(), true),
		Responses: openapi3.NewResponses(
			openapi3.WithStatus(201, jsonResponse("Quotation result.", quotationResultSchema())),
		),
	}
}

// quotationListOperation defines the OpenAPI operation for quotation listing.
func quotationListOperation() *openapi3.Operation {
	return &openapi3.Operation{
		OperationID: "ShippingController_listQuotations",
		Summary:     "List quotations by order",
		Tags:        []string{shippingTag},
		Security:    bearerSecurityRequirements(),
		Parameters: openapi3.Parameters{
			queryStringParameter("orderID", "Optional order identifier filter.", false),
		},
		Responses: openapi3.NewResponses(
			openapi3.WithStatus(200, jsonResponse("Quotation list.", quotationListResponseSchema())),
		),
	}
}

// quotationFromOrderOperation defines the OpenAPI operation for order-based quotation.
func quotationFromOrderOperation() *openapi3.Operation {
	return &openapi3.Operation{
		OperationID: "ShippingController_quoteFromOrder",
		Summary:     "Request a freight quotation from an order's products",
		Tags:        []string{shippingTag},
		Security:    bearerSecurityRequirements(),
		RequestBody: jsonRequestBody(quotationFromOrderRequestSchema(), true),
		Responses: openapi3.NewResponses(
			openapi3.WithStatus(201, jsonResponse("Quotation result with warnings.", quotationResultSchema())),
			openapi3.WithStatus(400, errorResponse("Bad request. Possible message codes: invalid_payload, no_valid_products, carrier_not_supported, quotation_not_supported, invalid_city_code.")),
			openapi3.WithStatus(401, errorResponse("Unauthorized. Message code: unauthorized.")),
			openapi3.WithStatus(403, errorResponse("Forbidden. Message code: forbidden.")),
			openapi3.WithStatus(404, errorResponse("Not found. Message code: shipping_resource_not_found.")),
			openapi3.WithStatus(500, errorResponse("Internal server error. Message code: internal_server_error.")),
		),
	}
}

// quotationOrderPackagingOperation defines the OpenAPI operation for order packaging previews without carrier quotation calls.
func quotationOrderPackagingOperation() *openapi3.Operation {
	return &openapi3.Operation{
		OperationID: "ShippingController_quoteOrderPackaging",
		Summary:     "Preview packed units for an order without carrier quotation",
		Tags:        []string{shippingTag},
		Security:    bearerSecurityRequirements(),
		RequestBody: jsonRequestBody(quotationFromOrderRequestSchema(), true),
		Responses: openapi3.NewResponses(
			openapi3.WithStatus(200, jsonResponse("Order packaging preview.", orderPackagingResultSchema())),
			openapi3.WithStatus(400, errorResponse("Bad request. Possible message codes: invalid_payload, no_valid_products, carrier_not_supported.")),
			openapi3.WithStatus(401, errorResponse("Unauthorized. Message code: unauthorized.")),
			openapi3.WithStatus(403, errorResponse("Forbidden. Message code: forbidden.")),
			openapi3.WithStatus(404, errorResponse("Not found. Message code: shipping_resource_not_found.")),
			openapi3.WithStatus(500, errorResponse("Internal server error. Message code: internal_server_error.")),
		),
	}
}

// getOrderQuotationOperation defines the OpenAPI operation for retrieving the latest order quotation.
func getOrderQuotationOperation() *openapi3.Operation {
	return &openapi3.Operation{
		OperationID: "ShippingController_getOrderQuotation",
		Summary:     "Get the latest non-expired quotation for an order and carrier",
		Tags:        []string{shippingTag},
		Security:    bearerSecurityRequirements(),
		Parameters: openapi3.Parameters{
			pathStringParameter("identifier", "Order internal ID or external identifier."),
			queryStringParameter("carrierId", "Carrier identifier filter.", true),
		},
		Responses: openapi3.NewResponses(
			openapi3.WithStatus(200, jsonResponse("Quotation record.", quotationRecordSchema())),
			openapi3.WithStatus(400, errorResponse("Bad request. Possible message codes: invalid_payload, carrier_not_supported.")),
			openapi3.WithStatus(401, errorResponse("Unauthorized. Message code: unauthorized.")),
			openapi3.WithStatus(403, errorResponse("Forbidden. Message code: forbidden.")),
			openapi3.WithStatus(404, errorResponse("Not found. Possible message codes: quotation_not_found, shipping_resource_not_found.")),
			openapi3.WithStatus(500, errorResponse("Internal server error. Message code: internal_server_error.")),
		),
	}
}

// markCreateOperation defines the OpenAPI operation for mark generation.
func markCreateOperation() *openapi3.Operation {
	return &openapi3.Operation{
		OperationID: "ShippingController_createMark",
		Summary:     "Generate one shipping mark",
		Tags:        []string{shippingTag},
		Security:    bearerSecurityRequirements(),
		RequestBody: jsonRequestBody(markRequestSchema(), true),
		Responses: openapi3.NewResponses(
			openapi3.WithStatus(201, jsonResponse("Shipping mark.", shippingMarkSchema())),
		),
	}
}

// markGetOperation defines the OpenAPI operation for mark lookup.
func markGetOperation() *openapi3.Operation {
	return &openapi3.Operation{
		OperationID: "ShippingController_getMark",
		Summary:     "Get one shipping mark",
		Tags:        []string{shippingTag},
		Security:    bearerSecurityRequirements(),
		Parameters: openapi3.Parameters{
			pathStringParameter("id", "Shipping mark identifier."),
		},
		Responses: openapi3.NewResponses(
			openapi3.WithStatus(200, jsonResponse("Shipping mark.", shippingMarkSchema())),
		),
	}
}

// markListOperation defines the OpenAPI operation for mark listing.
func markListOperation() *openapi3.Operation {
	return &openapi3.Operation{
		OperationID: "ShippingController_listMarks",
		Summary:     "List shipping marks",
		Tags:        []string{shippingTag},
		Security:    bearerSecurityRequirements(),
		Parameters: openapi3.Parameters{
			queryStringParameter("orderID", "Optional order identifier filter.", false),
			queryStringParameter("batchID", "Optional dispatch batch identifier filter.", false),
			queryIntParameter("page", "Pagination page (1-based).", false),
			queryIntParameter("limit", "Pagination page size.", false),
		},
		Responses: openapi3.NewResponses(
			openapi3.WithStatus(200, jsonResponse("Shipping mark list.", markListResponseSchema())),
		),
	}
}

// markVoidOperation defines the OpenAPI operation for mark voiding.
func markVoidOperation() *openapi3.Operation {
	return &openapi3.Operation{
		OperationID: "ShippingController_voidMark",
		Summary:     "Void one shipping mark",
		Tags:        []string{shippingTag},
		Security:    bearerSecurityRequirements(),
		Parameters: openapi3.Parameters{
			pathStringParameter("id", "Shipping mark identifier."),
		},
		RequestBody: jsonRequestBody(voidMarkRequestSchema(), false),
		Responses: openapi3.NewResponses(
			openapi3.WithStatus(200, jsonResponse("Shipping mark voided.", shippingMarkSchema())),
		),
	}
}

// markRelatedOperation defines the OpenAPI operation for related-mark listing.
func markRelatedOperation() *openapi3.Operation {
	return &openapi3.Operation{
		OperationID: "ShippingController_listRelatedMarks",
		Summary:     "List related shipping marks",
		Tags:        []string{shippingTag},
		Security:    bearerSecurityRequirements(),
		Parameters: openapi3.Parameters{
			pathStringParameter("id", "Shipping mark identifier."),
		},
		Responses: openapi3.NewResponses(
			openapi3.WithStatus(200, jsonResponse("Related shipping marks.", markListResponseSchema())),
		),
	}
}

// batchCreateOperation defines the OpenAPI operation for batch creation.
func batchCreateOperation() *openapi3.Operation {
	return &openapi3.Operation{
		OperationID: "ShippingController_createBatch",
		Summary:     "Create one dispatch batch",
		Tags:        []string{shippingTag},
		Security:    bearerSecurityRequirements(),
		RequestBody: jsonRequestBody(createBatchRequestSchema(), true),
		Responses: openapi3.NewResponses(
			openapi3.WithStatus(201, jsonResponse("Dispatch batch.", dispatchBatchSchema())),
		),
	}
}

// batchGetOperation defines the OpenAPI operation for batch lookup.
func batchGetOperation() *openapi3.Operation {
	return &openapi3.Operation{
		OperationID: "ShippingController_getBatch",
		Summary:     "Get one dispatch batch",
		Tags:        []string{shippingTag},
		Security:    bearerSecurityRequirements(),
		Parameters: openapi3.Parameters{
			pathStringParameter("id", "Dispatch batch identifier."),
		},
		Responses: openapi3.NewResponses(
			openapi3.WithStatus(200, jsonResponse("Dispatch batch.", dispatchBatchSchema())),
		),
	}
}

// batchListOperation defines the OpenAPI operation for batch listing.
func batchListOperation() *openapi3.Operation {
	return &openapi3.Operation{
		OperationID: "ShippingController_listBatches",
		Summary:     "List dispatch batches",
		Tags:        []string{shippingTag},
		Security:    bearerSecurityRequirements(),
		Parameters: openapi3.Parameters{
			queryStringParameter("carrierID", "Optional carrier identifier filter.", false),
			queryStringParameter("status", "Optional status filter (OPEN/CLOSED).", false),
			queryIntParameter("page", "Pagination page (1-based).", false),
			queryIntParameter("limit", "Pagination page size.", false),
		},
		Responses: openapi3.NewResponses(
			openapi3.WithStatus(200, jsonResponse("Dispatch batch list.", batchListResponseSchema())),
		),
	}
}

// batchAddMarkOperation defines the OpenAPI operation for draft mark creation in a batch.
func batchAddMarkOperation() *openapi3.Operation {
	return &openapi3.Operation{
		OperationID: "ShippingController_addBatchMark",
		Summary:     "Create one draft mark in a batch",
		Tags:        []string{shippingTag},
		Security:    bearerSecurityRequirements(),
		Parameters: openapi3.Parameters{
			pathStringParameter("id", "Dispatch batch identifier."),
		},
		RequestBody: jsonRequestBody(draftMarkRequestSchema(), true),
		Responses: openapi3.NewResponses(
			openapi3.WithStatus(201, jsonResponse("Draft shipping mark.", shippingMarkSchema())),
		),
	}
}

// batchCreateMarkOperation defines the OpenAPI operation for quoted/direct mark creation from one quotation id.
func batchCreateMarkOperation() *openapi3.Operation {
	return &openapi3.Operation{
		OperationID: "ShippingController_createBatchMark",
		Summary:     "Create one quoted or direct shipping mark in a batch",
		Tags:        []string{shippingTag},
		Security:    bearerSecurityRequirements(),
		RequestBody: jsonRequestBody(createBatchMarkRequestSchema(), true),
		Responses: openapi3.NewResponses(
			openapi3.WithStatus(201, jsonResponse("Shipping mark created.", shippingMarkSchema())),
			openapi3.WithStatus(400, errorResponse("Bad request. Possible message codes: invalid_payload, invalid_city_code, carrier_not_supported.")),
			openapi3.WithStatus(401, errorResponse("Unauthorized. Message code: unauthorized.")),
			openapi3.WithStatus(403, errorResponse("Forbidden. Message code: forbidden.")),
			openapi3.WithStatus(404, errorResponse("Not found. Message code: shipping_resource_not_found.")),
			openapi3.WithStatus(409, errorResponse("Conflict. Possible message codes: batch_closed, batch_carrier_mismatch, batch_mark_status_mismatch.")),
			openapi3.WithStatus(500, errorResponse("Internal server error. Possible message codes: internal_server_error, shipping_guardrail_violation. Guardrail errors include mark_id, order_id, guardrail rule, and request_preview in the error field.")),
		),
	}
}

// batchRemoveMarkOperation defines the OpenAPI operation for mark removal.
func batchRemoveMarkOperation() *openapi3.Operation {
	return &openapi3.Operation{
		OperationID: "ShippingController_removeBatchMark",
		Summary:     "Remove one mark from a dispatch batch",
		Tags:        []string{shippingTag},
		Security:    bearerSecurityRequirements(),
		Parameters: openapi3.Parameters{
			pathStringParameter("id", "Dispatch batch identifier."),
			pathStringParameter("markID", "Shipping mark identifier."),
		},
		Responses: openapi3.NewResponses(
			openapi3.WithStatus(200, jsonResponse("Dispatch batch.", dispatchBatchSchema())),
		),
	}
}

// batchCloseOperation defines the OpenAPI operation for batch closure.
func batchCloseOperation() *openapi3.Operation {
	return &openapi3.Operation{
		OperationID: "ShippingController_closeBatch",
		Summary:     "Close one dispatch batch",
		Tags:        []string{shippingTag},
		Security:    bearerSecurityRequirements(),
		Parameters: openapi3.Parameters{
			pathStringParameter("id", "Dispatch batch identifier."),
		},
		Responses: openapi3.NewResponses(
			openapi3.WithStatus(200, jsonResponse("Dispatch batch closed.", dispatchBatchSchema())),
			openapi3.WithStatus(400, errorResponse("Bad request. Possible message codes: invalid_payload, invalid_city_code, carrier_not_supported.")),
			openapi3.WithStatus(401, errorResponse("Unauthorized. Message code: unauthorized.")),
			openapi3.WithStatus(403, errorResponse("Forbidden. Message code: forbidden.")),
			openapi3.WithStatus(404, errorResponse("Not found. Message code: shipping_resource_not_found.")),
			openapi3.WithStatus(409, errorResponse("Conflict. Possible message codes: batch_closed, batch_status_invalid.")),
			openapi3.WithStatus(500, errorResponse("Internal server error. Possible message codes: internal_server_error, shipping_guardrail_violation. Guardrail errors include mark_id, order_id, guardrail rule, and request_preview in the error field.")),
		),
	}
}

// batchManifestDocumentOperation defines the OpenAPI operation for merged batch manifest documents.
func batchManifestDocumentOperation() *openapi3.Operation {
	return &openapi3.Operation{
		OperationID: "ShippingController_getBatchManifestDocument",
		Summary:     "Get merged batch manifest PDF document",
		Tags:        []string{shippingTag},
		Security:    bearerSecurityRequirements(),
		Parameters: openapi3.Parameters{
			pathStringParameter("id", "Dispatch batch identifier."),
		},
		Responses: openapi3.NewResponses(
			openapi3.WithStatus(200, binaryPDFResponse("Merged batch manifest PDF document.")),
		),
	}
}

// trackingGetOperation defines the OpenAPI operation for tracking lookups.
func trackingGetOperation() *openapi3.Operation {
	return &openapi3.Operation{
		OperationID: "ShippingController_getTracking",
		Summary:     "Get normalized tracking history",
		Tags:        []string{shippingTag},
		Security:    bearerSecurityRequirements(),
		Parameters: openapi3.Parameters{
			pathStringParameter("trackingNumber", "Carrier tracking number."),
			queryStringParameter("carrier", "Carrier identifier.", true),
		},
		Responses: openapi3.NewResponses(
			openapi3.WithStatus(200, jsonResponse("Tracking history.", trackingHistorySchema())),
		),
	}
}

// carrierListOperation defines the OpenAPI operation for carrier listing.
func carrierListOperation() *openapi3.Operation {
	return &openapi3.Operation{
		OperationID: "ShippingController_listCarriers",
		Summary:     "List configured carriers",
		Tags:        []string{shippingTag},
		Security:    bearerSecurityRequirements(),
		Responses: openapi3.NewResponses(
			openapi3.WithStatus(200, jsonResponse("Carrier list.", carrierListResponseSchema())),
		),
	}
}

// carrierGetOperation defines the OpenAPI operation for carrier lookup.
func carrierGetOperation() *openapi3.Operation {
	return &openapi3.Operation{
		OperationID: "ShippingController_getCarrier",
		Summary:     "Get one carrier",
		Tags:        []string{shippingTag},
		Security:    bearerSecurityRequirements(),
		Parameters: openapi3.Parameters{
			pathStringParameter("id", "Carrier identifier."),
		},
		Responses: openapi3.NewResponses(
			openapi3.WithStatus(200, jsonResponse("Carrier.", carrierSchema())),
		),
	}
}

// shipmentModeSchema defines schema for shipment mode enum values (parcel or express).
func shipmentModeSchema() *openapi3.Schema {
	s := openapi3.NewStringSchema()
	s.Enum = []interface{}{"parcel", "express"}
	return s
}

// quotationRequestSchema defines schema for quotation request payloads.
func quotationRequestSchema() *openapi3.Schema {
	modeSchema := shipmentModeSchema()
	modeSchema.Description = "Requested shipment mode. The service normalizes to express for one unit and parcel for two or more units."
	schema := openapi3.NewObjectSchema().
		WithProperty("orderId", openapi3.NewStringSchema()).
		WithProperty("carrierId", openapi3.NewStringSchema()).
		WithProperty("originCityCode", openapi3.NewStringSchema()).
		WithProperty("destCityCode", openapi3.NewStringSchema()).
		WithProperty("declaredValue", openapi3.NewFloat64Schema()).
		WithProperty("collectOnDeliveryAmount", openapi3.NewFloat64Schema()).
		WithProperty("units", openapi3.NewArraySchema().WithItems(packageUnitSchema())).
		WithProperty("shipmentMode", modeSchema)
	schema.Required = []string{"carrierId", "originCityCode", "destCityCode", "units", "shipmentMode"}

	return schema
}

// quotationResultSchema defines schema for quotation result payloads.
func quotationResultSchema() *openapi3.Schema {
	warningSchema := openapi3.NewObjectSchema().
		WithProperty("code", openapi3.NewStringSchema()).
		WithProperty("message", openapi3.NewStringSchema())

	return openapi3.NewObjectSchema().
		WithProperty("id", openapi3.NewStringSchema()).
		WithProperty("orderId", openapi3.NewStringSchema()).
		WithProperty("orderIdentifier", openapi3.NewStringSchema()).
		WithProperty("carrierId", openapi3.NewStringSchema()).
		WithProperty("originCityCode", openapi3.NewStringSchema()).
		WithProperty("destCityCode", openapi3.NewStringSchema()).
		WithProperty("units", openapi3.NewArraySchema().WithItems(packageUnitSchema())).
		WithProperty("freightCost", openapi3.NewFloat64Schema()).
		WithProperty("collectOnDeliveryAmount", openapi3.NewFloat64Schema()).
		WithProperty("collectOnDeliveryFeePercent", openapi3.NewFloat64Schema()).
		WithProperty("collectOnDeliveryFeeAmount", openapi3.NewFloat64Schema()).
		WithProperty("collectOnDeliveryChargedAmount", openapi3.NewFloat64Schema()).
		WithProperty("estimatedDays", openapi3.NewIntegerSchema()).
		WithProperty("currencyCode", openapi3.NewStringSchema()).
		WithProperty("expiresAt", openapi3.NewDateTimeSchema()).
		WithProperty("rawResponse", openapi3.NewStringSchema()).
		WithProperty("warnings", openapi3.NewArraySchema().WithItems(warningSchema)).
		WithProperty("createdAt", openapi3.NewDateTimeSchema())
}

// quotationListResponseSchema defines schema for quotation list payloads.
func quotationListResponseSchema() *openapi3.Schema {
	return openapi3.NewObjectSchema().
		WithProperty("data", openapi3.NewArraySchema().WithItems(quotationRecordSchema())).
		WithProperty("total", openapi3.NewIntegerSchema())
}

// quotationRecordSchema defines schema for persisted quotation records.
func quotationRecordSchema() *openapi3.Schema {
	requestSnapshotSchema := openapi3.NewStringSchema()
	requestSnapshotSchema.Description = "Base64-encoded carrier request snapshot (TCC payload when carrier=tcc; normalized quotation request for providers without carrier-native snapshot)."
	rawResponseSchema := openapi3.NewStringSchema()
	rawResponseSchema.Description = "Base64-encoded raw carrier response payload."

	return openapi3.NewObjectSchema().
		WithProperty("ID", openapi3.NewStringSchema()).
		WithProperty("OrderID", openapi3.NewStringSchema()).
		WithProperty("OrderIdentifier", openapi3.NewStringSchema()).
		WithProperty("CarrierID", openapi3.NewStringSchema()).
		WithProperty("OriginCityCode", openapi3.NewStringSchema()).
		WithProperty("DestCityCode", openapi3.NewStringSchema()).
		WithProperty("FreightCost", openapi3.NewFloat64Schema()).
		WithProperty("EstimatedDays", openapi3.NewIntegerSchema()).
		WithProperty("CurrencyCode", openapi3.NewStringSchema()).
		WithProperty("ExpiresAt", openapi3.NewDateTimeSchema()).
		WithProperty("RequestSnapshot", requestSnapshotSchema).
		WithProperty("RawResponse", rawResponseSchema).
		WithProperty("CreatedAt", openapi3.NewDateTimeSchema())
}

// errorResponseSchema defines schema for standard HTTP error payloads.
func errorResponseSchema() *openapi3.Schema {
	return openapi3.NewObjectSchema().
		WithProperty("message", openapi3.NewStringSchema()).
		WithProperty("error", openapi3.NewStringSchema())
}

// quotationFromOrderRequestSchema defines schema for order-based quotation request payloads.
func quotationFromOrderRequestSchema() *openapi3.Schema {
	schema := openapi3.NewObjectSchema().
		WithProperty("orderIdentifier", openapi3.NewStringSchema()).
		WithProperty("carrierId", openapi3.NewStringSchema()).
		WithProperty("originCityCode", openapi3.NewStringSchema())
	schema.Required = []string{"orderIdentifier", "carrierId", "originCityCode"}

	return schema
}

// orderPackagingResultSchema defines schema for order packaging preview payloads.
func orderPackagingResultSchema() *openapi3.Schema {
	warningSchema := openapi3.NewObjectSchema().
		WithProperty("code", openapi3.NewStringSchema()).
		WithProperty("message", openapi3.NewStringSchema())

	return openapi3.NewObjectSchema().
		WithProperty("orderId", openapi3.NewStringSchema()).
		WithProperty("orderIdentifier", openapi3.NewStringSchema()).
		WithProperty("carrierId", openapi3.NewStringSchema()).
		WithProperty("originCityCode", openapi3.NewStringSchema()).
		WithProperty("destCityCode", openapi3.NewStringSchema()).
		WithProperty("units", openapi3.NewArraySchema().WithItems(packageUnitSchema())).
		WithProperty("declaredValue", openapi3.NewFloat64Schema()).
		WithProperty("collectOnDeliveryAmount", openapi3.NewFloat64Schema()).
		WithProperty("shipmentMode", shipmentModeSchema()).
		WithProperty("warnings", openapi3.NewArraySchema().WithItems(warningSchema))
}

// markRequestSchema defines schema for mark generation request payloads.
func markRequestSchema() *openapi3.Schema {
	schema := openapi3.NewObjectSchema().
		WithProperty("orderId", openapi3.NewStringSchema()).
		WithProperty("carrierId", openapi3.NewStringSchema()).
		WithProperty("sender", addressSchema()).
		WithProperty("recipient", addressSchema()).
		WithProperty("units", openapi3.NewArraySchema().WithItems(packageUnitSchema())).
		WithProperty("declaredValue", openapi3.NewFloat64Schema()).
		WithProperty("paymentForm", openapi3.NewStringSchema()).
		WithProperty("collectOnDeliveryAmount", openapi3.NewFloat64Schema()).
		WithProperty("observations", openapi3.NewStringSchema()).
		WithProperty("trackingNumber", openapi3.NewStringSchema()).
		WithProperty("documentType", openapi3.NewStringSchema()).
		WithProperty("documentRef", openapi3.NewStringSchema()).
		WithProperty("manifestType", openapi3.NewStringSchema()).
		WithProperty("manifestRef", openapi3.NewStringSchema()).
		WithProperty("customTrackingUrl", openapi3.NewStringSchema()).
		WithProperty("shipmentMode", shipmentModeSchema())
	schema.Required = []string{"orderId", "carrierId", "sender", "recipient", "units", "shipmentMode"}

	return schema
}

// markListResponseSchema defines schema for mark list payloads.
func markListResponseSchema() *openapi3.Schema {
	return openapi3.NewObjectSchema().
		WithProperty("data", openapi3.NewArraySchema().WithItems(shippingMarkSchema())).
		WithProperty("total", openapi3.NewIntegerSchema()).
		WithProperty("page", openapi3.NewIntegerSchema()).
		WithProperty("limit", openapi3.NewIntegerSchema())
}

// voidMarkRequestSchema defines schema for mark-void request payloads.
func voidMarkRequestSchema() *openapi3.Schema {
	return openapi3.NewObjectSchema().WithProperty("reason", openapi3.NewStringSchema())
}

// createBatchRequestSchema defines schema for batch creation request payloads.
func createBatchRequestSchema() *openapi3.Schema {
	schema := openapi3.NewObjectSchema().
		WithProperty("carrierId", openapi3.NewStringSchema())
	schema.Required = []string{"carrierId"}

	return schema
}

// draftMarkRequestSchema defines schema for draft mark creation request payloads.
func draftMarkRequestSchema() *openapi3.Schema {
	schema := openapi3.NewObjectSchema().
		WithProperty("quotationId", openapi3.NewStringSchema()).
		WithProperty("quotedFreightCost", openapi3.NewFloat64Schema()).
		WithProperty("orderId", openapi3.NewStringSchema()).
		WithProperty("sender", addressSchema()).
		WithProperty("recipient", addressSchema()).
		WithProperty("units", openapi3.NewArraySchema().WithItems(packageUnitSchema())).
		WithProperty("declaredValue", openapi3.NewFloat64Schema()).
		WithProperty("paymentForm", openapi3.NewStringSchema()).
		WithProperty("collectOnDeliveryAmount", openapi3.NewFloat64Schema()).
		WithProperty("observations", openapi3.NewStringSchema()).
		WithProperty("trackingNumber", openapi3.NewStringSchema()).
		WithProperty("documentType", openapi3.NewStringSchema()).
		WithProperty("documentRef", openapi3.NewStringSchema()).
		WithProperty("manifestType", openapi3.NewStringSchema()).
		WithProperty("manifestRef", openapi3.NewStringSchema()).
		WithProperty("customTrackingUrl", openapi3.NewStringSchema()).
		WithProperty("shipmentMode", shipmentModeSchema())
	schema.Required = []string{"orderId", "sender", "recipient", "units", "shipmentMode"}

	return schema
}

// createBatchMarkRequestSchema defines schema for quoted/direct batch mark creation request payloads.
func createBatchMarkRequestSchema() *openapi3.Schema {
	schema := openapi3.NewObjectSchema().
		WithProperty("batch", openapi3.NewStringSchema()).
		WithProperty("direct", openapi3.NewBoolSchema()).
		WithProperty("quotationId", openapi3.NewStringSchema())
	schema.Required = []string{"batch", "quotationId"}

	return schema
}

// batchListResponseSchema defines schema for dispatch batch list payloads.
func batchListResponseSchema() *openapi3.Schema {
	return openapi3.NewObjectSchema().
		WithProperty("data", openapi3.NewArraySchema().WithItems(dispatchBatchSchema())).
		WithProperty("total", openapi3.NewIntegerSchema()).
		WithProperty("page", openapi3.NewIntegerSchema()).
		WithProperty("limit", openapi3.NewIntegerSchema())
}

// trackingHistorySchema defines schema for tracking history payloads.
func trackingHistorySchema() *openapi3.Schema {
	return openapi3.NewObjectSchema().
		WithProperty("carrierId", openapi3.NewStringSchema()).
		WithProperty("trackingNumber", openapi3.NewStringSchema()).
		WithProperty("globalStatus", openapi3.NewStringSchema()).
		WithProperty("lastUpdate", openapi3.NewDateTimeSchema()).
		WithProperty("history", openapi3.NewArraySchema().WithItems(trackingEventSchema()))
}

// trackingEventSchema defines schema for tracking checkpoint payloads.
func trackingEventSchema() *openapi3.Schema {
	return openapi3.NewObjectSchema().
		WithProperty("date", openapi3.NewDateTimeSchema()).
		WithProperty("code", openapi3.NewStringSchema()).
		WithProperty("text", openapi3.NewStringSchema()).
		WithProperty("city", openapi3.NewStringSchema()).
		WithProperty("status", openapi3.NewStringSchema())
}

// carrierSchema defines schema for shipping carrier payloads.
func carrierSchema() *openapi3.Schema {
	return openapi3.NewObjectSchema().
		WithProperty("id", openapi3.NewStringSchema()).
		WithProperty("name", openapi3.NewStringSchema()).
		WithProperty("type", openapi3.NewStringSchema()).
		WithProperty("active", openapi3.NewBoolSchema()).
		WithProperty("requiresBalanceCheck", openapi3.NewBoolSchema()).
		WithProperty("hasQuotation", openapi3.NewBoolSchema()).
		WithProperty("hasManifestDocument", openapi3.NewBoolSchema()).
		WithProperty("hasTracking", openapi3.NewBoolSchema()).
		WithProperty("needsUrl", openapi3.NewBoolSchema())
}

// carrierListResponseSchema defines schema for carrier list payloads.
func carrierListResponseSchema() *openapi3.Schema {
	return openapi3.NewObjectSchema().
		WithProperty("data", openapi3.NewArraySchema().WithItems(carrierSchema())).
		WithProperty("total", openapi3.NewIntegerSchema())
}

// shippingMarkSchema defines schema for generated shipping-mark payloads.
func shippingMarkSchema() *openapi3.Schema {
	draftSnapshotSchema := openapi3.NewStringSchema()
	draftSnapshotSchema.Description = "Base64-encoded JSON snapshot captured before carrier submission."
	responseSnapshotSchema := openapi3.NewStringSchema()
	responseSnapshotSchema.Description = "Base64-encoded JSON snapshot captured after carrier response handling."

	return openapi3.NewObjectSchema().
		WithProperty("id", openapi3.NewStringSchema()).
		WithProperty("orderId", openapi3.NewStringSchema()).
		WithProperty("carrierId", openapi3.NewStringSchema()).
		WithProperty("trackingNumber", openapi3.NewStringSchema()).
		WithProperty("status", openapi3.NewStringSchema()).
		WithProperty("documentType", openapi3.NewStringSchema()).
		WithProperty("documentRef", openapi3.NewStringSchema()).
		WithProperty("manifestType", openapi3.NewStringSchema()).
		WithProperty("manifestRef", openapi3.NewStringSchema()).
		WithProperty("sender", addressSchema()).
		WithProperty("recipient", addressSchema()).
		WithProperty("units", openapi3.NewArraySchema().WithItems(packageUnitSchema())).
		WithProperty("totalWeight", openapi3.NewFloat64Schema()).
		WithProperty("totalVolumetricWeight", openapi3.NewFloat64Schema()).
		WithProperty("declaredValue", openapi3.NewFloat64Schema()).
		WithProperty("paymentForm", openapi3.NewStringSchema()).
		WithProperty("collectOnDeliveryAmount", openapi3.NewFloat64Schema()).
		WithProperty("collectOnDeliveryFeePercent", openapi3.NewFloat64Schema()).
		WithProperty("collectOnDeliveryChargedAmount", openapi3.NewFloat64Schema()).
		WithProperty("observations", openapi3.NewStringSchema()).
		WithProperty("dispatchBatchId", openapi3.NewStringSchema()).
		WithProperty("quotationId", openapi3.NewStringSchema()).
		WithProperty("quotedFreightCost", openapi3.NewFloat64Schema()).
		WithProperty("draftSnapshot", draftSnapshotSchema).
		WithProperty("responseSnapshot", responseSnapshotSchema).
		WithProperty("shipmentMode", shipmentModeSchema()).
		WithProperty("failureReason", openapi3.NewStringSchema()).
		WithProperty("customTrackingUrl", openapi3.NewStringSchema()).
		WithProperty("createdAt", openapi3.NewDateTimeSchema()).
		WithProperty("updatedAt", openapi3.NewDateTimeSchema())
}

// dispatchBatchSchema defines schema for dispatch-batch payloads.
func dispatchBatchSchema() *openapi3.Schema {
	return openapi3.NewObjectSchema().
		WithProperty("id", openapi3.NewStringSchema()).
		WithProperty("carrierId", openapi3.NewStringSchema()).
		WithProperty("status", openapi3.NewStringSchema()).
		WithProperty("createdBy", openapi3.NewStringSchema()).
		WithProperty("markIds", openapi3.NewArraySchema().WithItems(openapi3.NewStringSchema())).
		WithProperty("createdAt", openapi3.NewDateTimeSchema()).
		WithProperty("closedAt", openapi3.NewDateTimeSchema())
}

// packageUnitSchema defines schema for package-unit payloads.
func packageUnitSchema() *openapi3.Schema {
	return openapi3.NewObjectSchema().
		WithProperty("description", openapi3.NewStringSchema()).
		WithProperty("packageType", openapi3.NewStringSchema()).
		WithProperty("dimensions", dimensionsSchema())
}

// dimensionsSchema defines schema for package dimension payloads.
func dimensionsSchema() *openapi3.Schema {
	return openapi3.NewObjectSchema().
		WithProperty("heightCm", openapi3.NewFloat64Schema()).
		WithProperty("widthCm", openapi3.NewFloat64Schema()).
		WithProperty("depthCm", openapi3.NewFloat64Schema()).
		WithProperty("realWeightKg", openapi3.NewFloat64Schema()).
		WithProperty("volumetricWeightKg", openapi3.NewFloat64Schema()).
		WithProperty("declaredValueCop", openapi3.NewFloat64Schema())
}

// orderDispatchOperation defines the OpenAPI operation for order dispatch provisioning status.
func orderDispatchOperation() *openapi3.Operation {
	return &openapi3.Operation{
		OperationID: "ShippingController_getOrderDispatch",
		Summary:     "Get order dispatch provisioning status",
		Tags:        []string{shippingTag},
		Security:    bearerSecurityRequirements(),
		Parameters: openapi3.Parameters{
			pathStringParameter("orderID", "Order identifier."),
		},
		Responses: openapi3.NewResponses(
			openapi3.WithStatus(200, jsonResponse("Order dispatch provisioning status.", orderDispatchResponseSchema())),
		),
	}
}

// orderDispatchResponseSchema defines schema for order dispatch provisioning response payloads.
func orderDispatchResponseSchema() *openapi3.Schema {
	return openapi3.NewObjectSchema().
		WithProperty("orderId", openapi3.NewStringSchema()).
		WithProperty("provisioned", openapi3.NewBoolSchema()).
		WithProperty("markId", openapi3.NewStringSchema()).
		WithProperty("batchId", openapi3.NewStringSchema()).
		WithProperty("status", openapi3.NewStringSchema())
}

// addressSchema defines schema for address payloads.
func addressSchema() *openapi3.Schema {
	return openapi3.NewObjectSchema().
		WithProperty("name", openapi3.NewStringSchema()).
		WithProperty("id", openapi3.NewStringSchema()).
		WithProperty("idType", openapi3.NewStringSchema()).
		WithProperty("addressLine", openapi3.NewStringSchema()).
		WithProperty("cityCode", openapi3.NewStringSchema()).
		WithProperty("phone", openapi3.NewStringSchema()).
		WithProperty("email", openapi3.NewStringSchema())
}
