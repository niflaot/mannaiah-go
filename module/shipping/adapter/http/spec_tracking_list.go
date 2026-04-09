package http

import "github.com/getkin/kin-openapi/openapi3"

// trackingListOperation defines the OpenAPI operation for tracking listing.
func trackingListOperation() *openapi3.Operation {
	return &openapi3.Operation{
		OperationID: "ShippingController_listTracking",
		Summary:     "List shipment tracking rows",
		Tags:        []string{shippingTag},
		Security:    bearerSecurityRequirements(),
		Parameters: openapi3.Parameters{
			queryStringParameter("term", "Optional free-text filter by tracking number, order id, or recipient name.", false),
			queryStringParameter("status", "Optional last-status filter. Supports MANUAL and normalized carrier statuses.", false),
			queryIntParameter("page", "Pagination page (1-based).", false),
			queryIntParameter("limit", "Pagination page size.", false),
		},
		Responses: openapi3.NewResponses(
			openapi3.WithStatus(200, jsonResponse("Tracking summary list.", trackingListResponseSchema())),
		),
	}
}

// trackingListResponseSchema defines schema for tracking list payloads.
func trackingListResponseSchema() *openapi3.Schema {
	return openapi3.NewObjectSchema().
		WithProperty("data", openapi3.NewArraySchema().WithItems(trackingListItemSchema())).
		WithProperty("total", openapi3.NewIntegerSchema()).
		WithProperty("page", openapi3.NewIntegerSchema()).
		WithProperty("limit", openapi3.NewIntegerSchema())
}

// trackingListItemSchema defines schema for one tracking-summary row.
func trackingListItemSchema() *openapi3.Schema {
	return openapi3.NewObjectSchema().
		WithProperty("id", openapi3.NewStringSchema()).
		WithProperty("orderId", openapi3.NewStringSchema()).
		WithProperty("trackingNumber", openapi3.NewStringSchema()).
		WithProperty("recipientName", openapi3.NewStringSchema()).
		WithProperty("carrierId", openapi3.NewStringSchema()).
		WithProperty("lastStatus", openapi3.NewStringSchema()).
		WithProperty("createdAt", openapi3.NewDateTimeSchema())
}
