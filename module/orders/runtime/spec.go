package runtime

import "github.com/getkin/kin-openapi/openapi3"

const (
	// ordersTag defines the OpenAPI tag used by order endpoints.
	ordersTag = "orders"
	// bearerSecurityScheme defines the OpenAPI security scheme key used for bearer auth.
	bearerSecurityScheme = "orders_bearer"
)

// OpenAPISpec returns order-module OpenAPI documentation.
func OpenAPISpec() *openapi3.T {
	components := openapi3.NewComponents()
	components.Schemas = openapi3.Schemas{
		"OrderCreate":         &openapi3.SchemaRef{Value: orderCreateSchema()},
		"OrderStatusUpdate":   &openapi3.SchemaRef{Value: orderStatusUpdateSchema()},
		"OrderItem":           &openapi3.SchemaRef{Value: orderItemSchema()},
		"OrderShipping":       &openapi3.SchemaRef{Value: orderShippingSchema()},
		"OrderShippingCharge": &openapi3.SchemaRef{Value: orderShippingChargeSchema()},
		"OrderStatusEntry":    &openapi3.SchemaRef{Value: orderStatusEntrySchema()},
		"Order":               &openapi3.SchemaRef{Value: orderSchema()},
		"OrderListResponse":   &openapi3.SchemaRef{Value: orderListResponseSchema()},
		"OrderListMeta":       &openapi3.SchemaRef{Value: orderListMetaSchema()},
		"OrderStatusEnumRef":  &openapi3.SchemaRef{Value: orderStatusSchema()},
	}
	components.SecuritySchemes = openapi3.SecuritySchemes{
		bearerSecurityScheme: &openapi3.SecuritySchemeRef{Value: openapi3.NewJWTSecurityScheme()},
	}

	return &openapi3.T{
		OpenAPI: "3.0.3",
		Info: &openapi3.Info{
			Title:   "Orders API",
			Version: "0.0.1",
		},
		Paths: openapi3.NewPaths(
			openapi3.WithPath("/orders", ordersPathItem()),
			openapi3.WithPath("/orders/{id}", orderByIDPathItem()),
			openapi3.WithPath("/orders/{id}/status", orderStatusPathItem()),
		),
		Components: &components,
		Tags: openapi3.Tags{
			&openapi3.Tag{Name: ordersTag},
		},
	}
}

// ordersPathItem returns OpenAPI path operations for collection endpoints.
func ordersPathItem() *openapi3.PathItem {
	return &openapi3.PathItem{
		Post: createOrderOperation(),
		Get:  listOrdersOperation(),
	}
}

// orderByIDPathItem returns OpenAPI path operations for ID-scoped endpoints.
func orderByIDPathItem() *openapi3.PathItem {
	return &openapi3.PathItem{
		Get: getOrderOperation(),
	}
}

// orderStatusPathItem returns OpenAPI path operations for status updates.
func orderStatusPathItem() *openapi3.PathItem {
	return &openapi3.PathItem{
		Patch: updateOrderStatusOperation(),
	}
}

// createOrderOperation defines the OpenAPI operation for order creation.
func createOrderOperation() *openapi3.Operation {
	return &openapi3.Operation{
		OperationID: "OrdersController_create",
		Summary:     "Create a new order",
		Tags:        []string{ordersTag},
		Security:    bearerSecurityRequirements(),
		RequestBody: jsonRequestBodyRef("#/components/schemas/OrderCreate"),
		Responses: openapi3.NewResponses(
			openapi3.WithStatus(201, responseWithDescription("The order has been successfully created.")),
			openapi3.WithStatus(400, responseWithDescription("Bad Request.")),
			openapi3.WithStatus(401, responseWithDescription("Unauthorized.")),
			openapi3.WithStatus(403, responseWithDescription("Forbidden - Insufficient permissions.")),
			openapi3.WithStatus(404, responseWithDescription("Order customer not found.")),
			openapi3.WithStatus(409, responseWithDescription("Conflict - Order identifier already exists in realm.")),
		),
	}
}

// listOrdersOperation defines the OpenAPI operation for order listing.
func listOrdersOperation() *openapi3.Operation {
	return &openapi3.Operation{
		OperationID: "OrdersController_findAll",
		Summary:     "Get orders with pagination and filtering",
		Tags:        []string{ordersTag},
		Security:    bearerSecurityRequirements(),
		Parameters: openapi3.Parameters{
			queryParameter("page", "Page number (default: 1)", openapi3.NewIntegerSchema()),
			queryParameter("limit", "Items per page (default: 10)", openapi3.NewIntegerSchema()),
			queryParameter("realm", "Filter by order realm", openapi3.NewStringSchema()),
			queryParameter("contactId", "Filter by contact id", openapi3.NewStringSchema()),
			queryParameter("identifier", "Filter by external order identifier", openapi3.NewStringSchema()),
			queryParameter("status", "Filter by current order status", orderStatusSchema()),
		},
		Responses: openapi3.NewResponses(
			openapi3.WithStatus(200, responseWithDescription("Return paginated orders.")),
			openapi3.WithStatus(400, responseWithDescription("Bad Request.")),
			openapi3.WithStatus(401, responseWithDescription("Unauthorized.")),
			openapi3.WithStatus(403, responseWithDescription("Forbidden - Insufficient permissions.")),
		),
	}
}

// getOrderOperation defines the OpenAPI operation for order retrieval by ID.
func getOrderOperation() *openapi3.Operation {
	return &openapi3.Operation{
		OperationID: "OrdersController_findOne",
		Summary:     "Get an order by id",
		Tags:        []string{ordersTag},
		Security:    bearerSecurityRequirements(),
		Parameters: openapi3.Parameters{
			pathParameter("id", "Order ID", openapi3.NewStringSchema()),
		},
		Responses: openapi3.NewResponses(
			openapi3.WithStatus(200, responseWithDescription("Return the order.")),
			openapi3.WithStatus(401, responseWithDescription("Unauthorized.")),
			openapi3.WithStatus(403, responseWithDescription("Forbidden - Insufficient permissions.")),
			openapi3.WithStatus(404, responseWithDescription("Order not found.")),
		),
	}
}

// updateOrderStatusOperation defines the OpenAPI operation for status updates.
func updateOrderStatusOperation() *openapi3.Operation {
	return &openapi3.Operation{
		OperationID: "OrdersController_updateStatus",
		Summary:     "Append status for an order",
		Tags:        []string{ordersTag},
		Security:    bearerSecurityRequirements(),
		Parameters: openapi3.Parameters{
			pathParameter("id", "Order ID", openapi3.NewStringSchema()),
		},
		RequestBody: jsonRequestBodyRef("#/components/schemas/OrderStatusUpdate"),
		Responses: openapi3.NewResponses(
			openapi3.WithStatus(200, responseWithDescription("The order status has been successfully updated.")),
			openapi3.WithStatus(400, responseWithDescription("Bad Request.")),
			openapi3.WithStatus(401, responseWithDescription("Unauthorized.")),
			openapi3.WithStatus(403, responseWithDescription("Forbidden - Insufficient permissions.")),
			openapi3.WithStatus(404, responseWithDescription("Order not found.")),
		),
	}
}

// bearerSecurityRequirements builds bearer-auth operation security requirements.
func bearerSecurityRequirements() *openapi3.SecurityRequirements {
	return openapi3.NewSecurityRequirements().With(openapi3.NewSecurityRequirement().Authenticate(bearerSecurityScheme))
}

// responseWithDescription builds an OpenAPI response from a plain description.
func responseWithDescription(description string) *openapi3.ResponseRef {
	return &openapi3.ResponseRef{Value: openapi3.NewResponse().WithDescription(description)}
}

// jsonRequestBodyRef builds a required JSON request body referencing a component schema.
func jsonRequestBodyRef(schemaRef string) *openapi3.RequestBodyRef {
	return &openapi3.RequestBodyRef{
		Value: openapi3.NewRequestBody().
			WithRequired(true).
			WithContent(openapi3.Content{
				"application/json": &openapi3.MediaType{
					Schema: &openapi3.SchemaRef{Ref: schemaRef},
				},
			}),
	}
}

// queryParameter builds optional query-parameter OpenAPI definitions.
func queryParameter(name, description string, schema *openapi3.Schema) *openapi3.ParameterRef {
	return &openapi3.ParameterRef{
		Value: openapi3.NewQueryParameter(name).
			WithDescription(description).
			WithSchema(schema),
	}
}

// pathParameter builds required path-parameter OpenAPI definitions.
func pathParameter(name, description string, schema *openapi3.Schema) *openapi3.ParameterRef {
	return &openapi3.ParameterRef{
		Value: openapi3.NewPathParameter(name).
			WithDescription(description).
			WithSchema(schema),
	}
}

// orderCreateSchema returns request schema for order creation payloads.
func orderCreateSchema() *openapi3.Schema {
	return openapi3.NewObjectSchema().
		WithProperty("identifier", openapi3.NewStringSchema()).
		WithProperty("realm", openapi3.NewStringSchema()).
		WithProperty("contactId", openapi3.NewStringSchema()).
		WithProperty("items", openapi3.NewArraySchema().WithItems(orderItemSchema())).
		WithProperty("initialStatus", orderStatusSchema()).
		WithProperty("author", openapi3.NewStringSchema()).
		WithProperty("description", openapi3.NewStringSchema()).
		WithProperty("shippingAddress", orderShippingSchema()).
		WithProperty("shippingCharges", openapi3.NewArraySchema().WithItems(orderShippingChargeSchema())).
		WithProperty("metadata", metadataSchema()).
		WithRequired([]string{"identifier", "realm", "contactId", "items"})
}

// orderStatusUpdateSchema returns request schema for order status update payloads.
func orderStatusUpdateSchema() *openapi3.Schema {
	return openapi3.NewObjectSchema().
		WithProperty("status", orderStatusSchema()).
		WithProperty("author", openapi3.NewStringSchema()).
		WithProperty("description", openapi3.NewStringSchema()).
		WithProperty("noteOwner", openapi3.NewStringSchema()).
		WithProperty("note", openapi3.NewStringSchema()).
		WithRequired([]string{"status", "author"})
}

// orderSchema returns response schema for order payloads.
func orderSchema() *openapi3.Schema {
	return openapi3.NewObjectSchema().
		WithProperty("id", openapi3.NewStringSchema()).
		WithProperty("identifier", openapi3.NewStringSchema()).
		WithProperty("realm", openapi3.NewStringSchema()).
		WithProperty("contactId", openapi3.NewStringSchema()).
		WithProperty("items", openapi3.NewArraySchema().WithItems(orderItemSchema())).
		WithProperty("currentStatus", orderStatusSchema()).
		WithProperty("statusHistory", openapi3.NewArraySchema().WithItems(orderStatusEntrySchema())).
		WithProperty("shippingAddress", orderShippingSchema()).
		WithProperty("hasCustomShippingAddress", openapi3.NewBoolSchema()).
		WithProperty("shippingCharges", openapi3.NewArraySchema().WithItems(orderShippingChargeSchema())).
		WithProperty("metadata", metadataSchema()).
		WithProperty("createdAt", openapi3.NewDateTimeSchema()).
		WithProperty("updatedAt", openapi3.NewDateTimeSchema())
}

// orderItemSchema returns schema for order item payloads.
func orderItemSchema() *openapi3.Schema {
	return openapi3.NewObjectSchema().
		WithProperty("sku", openapi3.NewStringSchema()).
		WithProperty("alternateName", openapi3.NewStringSchema()).
		WithProperty("quantity", openapi3.NewIntegerSchema()).
		WithProperty("value", openapi3.NewFloat64Schema()).
		WithProperty("productId", openapi3.NewStringSchema()).
		WithProperty("resolutionSource", openapi3.NewStringSchema()).
		WithRequired([]string{"quantity"})
}

// orderStatusEntrySchema returns schema for order status history payloads.
func orderStatusEntrySchema() *openapi3.Schema {
	return openapi3.NewObjectSchema().
		WithProperty("status", orderStatusSchema()).
		WithProperty("author", openapi3.NewStringSchema()).
		WithProperty("description", openapi3.NewStringSchema()).
		WithProperty("noteOwner", openapi3.NewStringSchema()).
		WithProperty("note", openapi3.NewStringSchema()).
		WithProperty("occurredAt", openapi3.NewDateTimeSchema()).
		WithRequired([]string{"status", "author", "occurredAt"})
}

// orderShippingSchema returns schema for shipping-address payloads.
func orderShippingSchema() *openapi3.Schema {
	return openapi3.NewObjectSchema().
		WithProperty("address", openapi3.NewStringSchema()).
		WithProperty("address2", openapi3.NewStringSchema()).
		WithProperty("phone", openapi3.NewStringSchema()).
		WithProperty("cityCode", openapi3.NewStringSchema())
}

// orderShippingChargeSchema returns schema for shipping-charge payloads.
func orderShippingChargeSchema() *openapi3.Schema {
	return openapi3.NewObjectSchema().
		WithProperty("methodId", openapi3.NewStringSchema()).
		WithProperty("methodTitle", openapi3.NewStringSchema()).
		WithProperty("price", openapi3.NewFloat64Schema())
}

// orderStatusSchema returns enum schema for supported order statuses.
func orderStatusSchema() *openapi3.Schema {
	schema := openapi3.NewStringSchema()
	schema.Enum = []any{"CANCELLED", "CREATED", "PENDING", "HOLD", "COMPLETED"}
	return schema
}

// orderListResponseSchema returns schema for paginated order list payloads.
func orderListResponseSchema() *openapi3.Schema {
	return openapi3.NewObjectSchema().
		WithProperty("data", openapi3.NewArraySchema().WithItems(orderSchema())).
		WithProperty("meta", orderListMetaSchema())
}

// orderListMetaSchema returns schema for order list pagination metadata.
func orderListMetaSchema() *openapi3.Schema {
	return openapi3.NewObjectSchema().
		WithProperty("page", openapi3.NewIntegerSchema()).
		WithProperty("total", openapi3.NewIntegerSchema()).
		WithProperty("limit", openapi3.NewIntegerSchema()).
		WithProperty("totalPages", openapi3.NewIntegerSchema())
}

// metadataSchema returns object schema for metadata maps.
func metadataSchema() *openapi3.Schema {
	return openapi3.NewObjectSchema().WithAdditionalProperties(openapi3.NewStringSchema())
}
