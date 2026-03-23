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

// responseWithDescription builds OpenAPI responses from plain descriptions.
func responseWithDescription(description string) *openapi3.ResponseRef {
	return &openapi3.ResponseRef{Value: openapi3.NewResponse().WithDescription(description)}
}

// quotationCreateOperation defines the OpenAPI operation for quotation creation.
func quotationCreateOperation() *openapi3.Operation {
	return &openapi3.Operation{OperationID: "ShippingController_createQuotation", Summary: "Request one freight quotation", Tags: []string{shippingTag}, Security: bearerSecurityRequirements(), Responses: openapi3.NewResponses(openapi3.WithStatus(201, responseWithDescription("Quotation result.")))}
}

// quotationListOperation defines the OpenAPI operation for quotation listing.
func quotationListOperation() *openapi3.Operation {
	return &openapi3.Operation{OperationID: "ShippingController_listQuotations", Summary: "List quotations by order", Tags: []string{shippingTag}, Security: bearerSecurityRequirements(), Responses: openapi3.NewResponses(openapi3.WithStatus(200, responseWithDescription("Quotation list.")))}
}

// markCreateOperation defines the OpenAPI operation for mark generation.
func markCreateOperation() *openapi3.Operation {
	return &openapi3.Operation{OperationID: "ShippingController_createMark", Summary: "Generate one shipping mark", Tags: []string{shippingTag}, Security: bearerSecurityRequirements(), Responses: openapi3.NewResponses(openapi3.WithStatus(201, responseWithDescription("Shipping mark.")))}
}

// markGetOperation defines the OpenAPI operation for mark lookup.
func markGetOperation() *openapi3.Operation {
	return &openapi3.Operation{OperationID: "ShippingController_getMark", Summary: "Get one shipping mark", Tags: []string{shippingTag}, Security: bearerSecurityRequirements(), Parameters: openapi3.Parameters{{Value: &openapi3.Parameter{Name: "id", In: "path", Required: true, Schema: &openapi3.SchemaRef{Value: openapi3.NewStringSchema()}}}}, Responses: openapi3.NewResponses(openapi3.WithStatus(200, responseWithDescription("Shipping mark.")))}
}

// markListOperation defines the OpenAPI operation for mark listing.
func markListOperation() *openapi3.Operation {
	return &openapi3.Operation{OperationID: "ShippingController_listMarks", Summary: "List shipping marks", Tags: []string{shippingTag}, Security: bearerSecurityRequirements(), Responses: openapi3.NewResponses(openapi3.WithStatus(200, responseWithDescription("Shipping mark list.")))}
}

// markVoidOperation defines the OpenAPI operation for mark voiding.
func markVoidOperation() *openapi3.Operation {
	return &openapi3.Operation{OperationID: "ShippingController_voidMark", Summary: "Void one shipping mark", Tags: []string{shippingTag}, Security: bearerSecurityRequirements(), Parameters: openapi3.Parameters{{Value: &openapi3.Parameter{Name: "id", In: "path", Required: true, Schema: &openapi3.SchemaRef{Value: openapi3.NewStringSchema()}}}}, Responses: openapi3.NewResponses(openapi3.WithStatus(200, responseWithDescription("Shipping mark voided.")))}
}

// batchCreateOperation defines the OpenAPI operation for batch creation.
func batchCreateOperation() *openapi3.Operation {
	return &openapi3.Operation{OperationID: "ShippingController_createBatch", Summary: "Create one dispatch batch", Tags: []string{shippingTag}, Security: bearerSecurityRequirements(), Responses: openapi3.NewResponses(openapi3.WithStatus(201, responseWithDescription("Dispatch batch.")))}
}

// batchGetOperation defines the OpenAPI operation for batch lookup.
func batchGetOperation() *openapi3.Operation {
	return &openapi3.Operation{OperationID: "ShippingController_getBatch", Summary: "Get one dispatch batch", Tags: []string{shippingTag}, Security: bearerSecurityRequirements(), Parameters: openapi3.Parameters{{Value: &openapi3.Parameter{Name: "id", In: "path", Required: true, Schema: &openapi3.SchemaRef{Value: openapi3.NewStringSchema()}}}}, Responses: openapi3.NewResponses(openapi3.WithStatus(200, responseWithDescription("Dispatch batch.")))}
}

// batchListOperation defines the OpenAPI operation for batch listing.
func batchListOperation() *openapi3.Operation {
	return &openapi3.Operation{OperationID: "ShippingController_listBatches", Summary: "List dispatch batches", Tags: []string{shippingTag}, Security: bearerSecurityRequirements(), Responses: openapi3.NewResponses(openapi3.WithStatus(200, responseWithDescription("Dispatch batch list.")))}
}

// batchAddMarksOperation defines the OpenAPI operation for mark assignment.
func batchAddMarksOperation() *openapi3.Operation {
	return &openapi3.Operation{OperationID: "ShippingController_addBatchMarks", Summary: "Add mark(s) to one dispatch batch", Tags: []string{shippingTag}, Security: bearerSecurityRequirements(), Parameters: openapi3.Parameters{{Value: &openapi3.Parameter{Name: "id", In: "path", Required: true, Schema: &openapi3.SchemaRef{Value: openapi3.NewStringSchema()}}}}, Responses: openapi3.NewResponses(openapi3.WithStatus(200, responseWithDescription("Dispatch batch.")))}
}

// batchRemoveMarkOperation defines the OpenAPI operation for mark removal.
func batchRemoveMarkOperation() *openapi3.Operation {
	return &openapi3.Operation{OperationID: "ShippingController_removeBatchMark", Summary: "Remove one mark from a dispatch batch", Tags: []string{shippingTag}, Security: bearerSecurityRequirements(), Parameters: openapi3.Parameters{{Value: &openapi3.Parameter{Name: "id", In: "path", Required: true, Schema: &openapi3.SchemaRef{Value: openapi3.NewStringSchema()}}}, {Value: &openapi3.Parameter{Name: "markID", In: "path", Required: true, Schema: &openapi3.SchemaRef{Value: openapi3.NewStringSchema()}}}}, Responses: openapi3.NewResponses(openapi3.WithStatus(200, responseWithDescription("Dispatch batch.")))}
}

// batchCloseOperation defines the OpenAPI operation for batch closure.
func batchCloseOperation() *openapi3.Operation {
	return &openapi3.Operation{OperationID: "ShippingController_closeBatch", Summary: "Close one dispatch batch", Tags: []string{shippingTag}, Security: bearerSecurityRequirements(), Parameters: openapi3.Parameters{{Value: &openapi3.Parameter{Name: "id", In: "path", Required: true, Schema: &openapi3.SchemaRef{Value: openapi3.NewStringSchema()}}}}, Responses: openapi3.NewResponses(openapi3.WithStatus(200, responseWithDescription("Dispatch batch closed.")))}
}

// trackingGetOperation defines the OpenAPI operation for tracking lookups.
func trackingGetOperation() *openapi3.Operation {
	return &openapi3.Operation{OperationID: "ShippingController_getTracking", Summary: "Get normalized tracking history", Tags: []string{shippingTag}, Security: bearerSecurityRequirements(), Parameters: openapi3.Parameters{{Value: &openapi3.Parameter{Name: "trackingNumber", In: "path", Required: true, Schema: &openapi3.SchemaRef{Value: openapi3.NewStringSchema()}}}, {Value: &openapi3.Parameter{Name: "carrier", In: "query", Required: true, Schema: &openapi3.SchemaRef{Value: openapi3.NewStringSchema()}}}}, Responses: openapi3.NewResponses(openapi3.WithStatus(200, responseWithDescription("Tracking history.")))}
}

// carrierListOperation defines the OpenAPI operation for carrier listing.
func carrierListOperation() *openapi3.Operation {
	return &openapi3.Operation{OperationID: "ShippingController_listCarriers", Summary: "List configured carriers", Tags: []string{shippingTag}, Security: bearerSecurityRequirements(), Responses: openapi3.NewResponses(openapi3.WithStatus(200, responseWithDescription("Carrier list.")))}
}

// carrierGetOperation defines the OpenAPI operation for carrier lookup.
func carrierGetOperation() *openapi3.Operation {
	return &openapi3.Operation{OperationID: "ShippingController_getCarrier", Summary: "Get one carrier", Tags: []string{shippingTag}, Security: bearerSecurityRequirements(), Parameters: openapi3.Parameters{{Value: &openapi3.Parameter{Name: "id", In: "path", Required: true, Schema: &openapi3.SchemaRef{Value: openapi3.NewStringSchema()}}}}, Responses: openapi3.NewResponses(openapi3.WithStatus(200, responseWithDescription("Carrier.")))}
}
