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
		openapi3.WithPath("/shipping/marks", &openapi3.PathItem{Post: markCreateOperation(), Get: markListOperation()}),
		openapi3.WithPath("/shipping/marks/{id}", &openapi3.PathItem{Get: markGetOperation()}),
		openapi3.WithPath("/shipping/marks/{id}/void", &openapi3.PathItem{Patch: markVoidOperation()}),
		openapi3.WithPath("/shipping/batches", &openapi3.PathItem{Post: batchCreateOperation(), Get: batchListOperation()}),
		openapi3.WithPath("/shipping/batches/{id}", &openapi3.PathItem{Get: batchGetOperation()}),
		openapi3.WithPath("/shipping/batches/{id}/marks", &openapi3.PathItem{Post: batchAddMarksOperation()}),
		openapi3.WithPath("/shipping/batches/{id}/marks/{markID}", &openapi3.PathItem{Delete: batchRemoveMarkOperation()}),
		openapi3.WithPath("/shipping/batches/{id}/close", &openapi3.PathItem{Patch: batchCloseOperation()}),
		openapi3.WithPath("/shipping/tracking/{trackingNumber}", &openapi3.PathItem{Get: trackingGetOperation()}),
		openapi3.WithPath("/shipping/carriers", &openapi3.PathItem{Get: carrierListOperation()}),
		openapi3.WithPath("/shipping/carriers/{id}", &openapi3.PathItem{Get: carrierGetOperation()}),
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

// batchAddMarksOperation defines the OpenAPI operation for mark assignment.
func batchAddMarksOperation() *openapi3.Operation {
	return &openapi3.Operation{
		OperationID: "ShippingController_addBatchMarks",
		Summary:     "Add mark(s) to one dispatch batch",
		Tags:        []string{shippingTag},
		Security:    bearerSecurityRequirements(),
		Parameters: openapi3.Parameters{
			pathStringParameter("id", "Dispatch batch identifier."),
		},
		RequestBody: jsonRequestBody(addBatchMarksRequestSchema(), true),
		Responses: openapi3.NewResponses(
			openapi3.WithStatus(200, jsonResponse("Dispatch batch.", dispatchBatchSchema())),
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

// quotationRequestSchema defines schema for quotation request payloads.
func quotationRequestSchema() *openapi3.Schema {
	schema := openapi3.NewObjectSchema().
		WithProperty("orderId", openapi3.NewStringSchema()).
		WithProperty("carrierId", openapi3.NewStringSchema()).
		WithProperty("originCityCode", openapi3.NewStringSchema()).
		WithProperty("destCityCode", openapi3.NewStringSchema()).
		WithProperty("declaredValue", openapi3.NewFloat64Schema()).
		WithProperty("collectOnDeliveryAmount", openapi3.NewFloat64Schema()).
		WithProperty("units", openapi3.NewArraySchema().WithItems(packageUnitSchema()))
	schema.Required = []string{"carrierId", "originCityCode", "destCityCode", "units"}

	return schema
}

// quotationResultSchema defines schema for quotation result payloads.
func quotationResultSchema() *openapi3.Schema {
	return openapi3.NewObjectSchema().
		WithProperty("id", openapi3.NewStringSchema()).
		WithProperty("orderId", openapi3.NewStringSchema()).
		WithProperty("carrierId", openapi3.NewStringSchema()).
		WithProperty("originCityCode", openapi3.NewStringSchema()).
		WithProperty("destCityCode", openapi3.NewStringSchema()).
		WithProperty("fullFreightCost", openapi3.NewFloat64Schema()).
		WithProperty("discountPercent", openapi3.NewFloat64Schema()).
		WithProperty("discountedFreightCost", openapi3.NewFloat64Schema()).
		WithProperty("freightCost", openapi3.NewFloat64Schema()).
		WithProperty("collectOnDeliveryAmount", openapi3.NewFloat64Schema()).
		WithProperty("collectOnDeliveryFeePercent", openapi3.NewFloat64Schema()).
		WithProperty("collectOnDeliveryChargedAmount", openapi3.NewFloat64Schema()).
		WithProperty("estimatedDays", openapi3.NewIntegerSchema()).
		WithProperty("currencyCode", openapi3.NewStringSchema()).
		WithProperty("expiresAt", openapi3.NewDateTimeSchema()).
		WithProperty("rawResponse", openapi3.NewStringSchema()).
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
	return openapi3.NewObjectSchema().
		WithProperty("ID", openapi3.NewStringSchema()).
		WithProperty("OrderID", openapi3.NewStringSchema()).
		WithProperty("CarrierID", openapi3.NewStringSchema()).
		WithProperty("OriginCityCode", openapi3.NewStringSchema()).
		WithProperty("DestCityCode", openapi3.NewStringSchema()).
		WithProperty("FullFreightCost", openapi3.NewFloat64Schema()).
		WithProperty("DiscountPercent", openapi3.NewFloat64Schema()).
		WithProperty("DiscountedFreightCost", openapi3.NewFloat64Schema()).
		WithProperty("FreightCost", openapi3.NewFloat64Schema()).
		WithProperty("EstimatedDays", openapi3.NewIntegerSchema()).
		WithProperty("CurrencyCode", openapi3.NewStringSchema()).
		WithProperty("ExpiresAt", openapi3.NewDateTimeSchema()).
		WithProperty("RequestSnapshot", openapi3.NewStringSchema()).
		WithProperty("RawResponse", openapi3.NewStringSchema()).
		WithProperty("CreatedAt", openapi3.NewDateTimeSchema())
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
		WithProperty("documentRef", openapi3.NewStringSchema())
	schema.Required = []string{"orderId", "carrierId", "sender", "recipient", "units"}

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
		WithProperty("name", openapi3.NewStringSchema()).
		WithProperty("carrierId", openapi3.NewStringSchema())
	schema.Required = []string{"name", "carrierId"}

	return schema
}

// addBatchMarksRequestSchema defines schema for batch mark-assignment request payloads.
func addBatchMarksRequestSchema() *openapi3.Schema {
	schema := openapi3.NewObjectSchema().
		WithProperty("markIds", openapi3.NewArraySchema().WithItems(openapi3.NewStringSchema()))
	schema.Required = []string{"markIds"}

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
		WithProperty("requiresBalanceCheck", openapi3.NewBoolSchema())
}

// carrierListResponseSchema defines schema for carrier list payloads.
func carrierListResponseSchema() *openapi3.Schema {
	return openapi3.NewObjectSchema().
		WithProperty("data", openapi3.NewArraySchema().WithItems(carrierSchema())).
		WithProperty("total", openapi3.NewIntegerSchema())
}

// shippingMarkSchema defines schema for generated shipping-mark payloads.
func shippingMarkSchema() *openapi3.Schema {
	return openapi3.NewObjectSchema().
		WithProperty("id", openapi3.NewStringSchema()).
		WithProperty("orderId", openapi3.NewStringSchema()).
		WithProperty("carrierId", openapi3.NewStringSchema()).
		WithProperty("trackingNumber", openapi3.NewStringSchema()).
		WithProperty("status", openapi3.NewStringSchema()).
		WithProperty("documentType", openapi3.NewStringSchema()).
		WithProperty("documentRef", openapi3.NewStringSchema()).
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
		WithProperty("createdAt", openapi3.NewDateTimeSchema()).
		WithProperty("updatedAt", openapi3.NewDateTimeSchema())
}

// dispatchBatchSchema defines schema for dispatch-batch payloads.
func dispatchBatchSchema() *openapi3.Schema {
	return openapi3.NewObjectSchema().
		WithProperty("id", openapi3.NewStringSchema()).
		WithProperty("name", openapi3.NewStringSchema()).
		WithProperty("carrierId", openapi3.NewStringSchema()).
		WithProperty("status", openapi3.NewStringSchema()).
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
