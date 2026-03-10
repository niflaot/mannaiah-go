package runtime

import "github.com/getkin/kin-openapi/openapi3"

const (
	// productsTag defines the OpenAPI tag used by product endpoints.
	productsTag = "products"
	// variationsTag defines the OpenAPI tag used by variation endpoints.
	variationsTag = "variations"
	// bearerSecurityScheme defines the OpenAPI security scheme key used for bearer auth.
	bearerSecurityScheme = "products_bearer"
)

// OpenAPISpec returns product-module OpenAPI documentation.
func OpenAPISpec() *openapi3.T {
	components := openapi3.NewComponents()
	components.Schemas = openapi3.Schemas{
		"CreateProductDto":   &openapi3.SchemaRef{Value: createProductSchema()},
		"UpdateProductDto":   &openapi3.SchemaRef{Value: updateProductSchema()},
		"Product":            &openapi3.SchemaRef{Value: productSchema()},
		"GalleryItemDto":     &openapi3.SchemaRef{Value: galleryItemSchema()},
		"DatasheetDto":       &openapi3.SchemaRef{Value: datasheetSchema()},
		"ProductVariantDto":  &openapi3.SchemaRef{Value: productVariantSchema()},
		"CreateVariationDto": &openapi3.SchemaRef{Value: createVariationSchema()},
		"UpdateVariationDto": &openapi3.SchemaRef{Value: updateVariationSchema()},
		"Variation":          &openapi3.SchemaRef{Value: variationSchema()},
	}
	components.SecuritySchemes = openapi3.SecuritySchemes{
		bearerSecurityScheme: &openapi3.SecuritySchemeRef{Value: openapi3.NewJWTSecurityScheme()},
	}

	return &openapi3.T{
		OpenAPI: "3.0.3",
		Info: &openapi3.Info{
			Title:   "Products API",
			Version: "1.0.0",
		},
		Paths: openapi3.NewPaths(
			openapi3.WithPath("/products", productsPathItem()),
			openapi3.WithPath("/products/{id}", productByIDPathItem()),
			openapi3.WithPath("/variations", variationsPathItem()),
			openapi3.WithPath("/variations/{id}", variationByIDPathItem()),
		),
		Components: &components,
		Tags: openapi3.Tags{
			&openapi3.Tag{Name: productsTag},
			&openapi3.Tag{Name: variationsTag},
		},
	}
}

// productsPathItem returns OpenAPI path operations for collection endpoints.
func productsPathItem() *openapi3.PathItem {
	return &openapi3.PathItem{
		Post: createProductOperation(),
		Get:  listProductsOperation(),
	}
}

// productByIDPathItem returns OpenAPI path operations for ID-scoped endpoints.
func productByIDPathItem() *openapi3.PathItem {
	return &openapi3.PathItem{
		Get:    getProductOperation(),
		Patch:  updateProductOperation(),
		Delete: deleteProductOperation(),
	}
}

// createProductOperation defines the OpenAPI operation for product creation.
func createProductOperation() *openapi3.Operation {
	return &openapi3.Operation{
		OperationID: "ProductsController_create",
		Summary:     "Create a new product",
		Tags:        []string{productsTag},
		Security:    bearerSecurityRequirements(),
		RequestBody: jsonRequestBodyRef("#/components/schemas/CreateProductDto"),
		Responses: openapi3.NewResponses(
			openapi3.WithStatus(201, responseWithDescription("The product has been successfully created.")),
			openapi3.WithStatus(400, responseWithDescription("Bad Request.")),
			openapi3.WithStatus(401, responseWithDescription("Unauthorized.")),
			openapi3.WithStatus(403, responseWithDescription("Forbidden - Insufficient permissions.")),
			openapi3.WithStatus(409, responseWithDescription("Conflict - SKU already exists.")),
		),
	}
}

// listProductsOperation defines the OpenAPI operation for product listing.
func listProductsOperation() *openapi3.Operation {
	return &openapi3.Operation{
		OperationID: "ProductsController_findAll",
		Summary:     "Get all products",
		Tags:        []string{productsTag},
		Security:    bearerSecurityRequirements(),
		Responses: openapi3.NewResponses(
			openapi3.WithStatus(200, responseWithDescription("Return all products.")),
			openapi3.WithStatus(401, responseWithDescription("Unauthorized.")),
			openapi3.WithStatus(403, responseWithDescription("Forbidden - Insufficient permissions.")),
		),
	}
}

// getProductOperation defines the OpenAPI operation for product retrieval by ID.
func getProductOperation() *openapi3.Operation {
	return &openapi3.Operation{
		OperationID: "ProductsController_findOne",
		Summary:     "Get a product by id",
		Tags:        []string{productsTag},
		Security:    bearerSecurityRequirements(),
		Parameters: openapi3.Parameters{
			pathParameter("id", "Product ID", openapi3.NewStringSchema()),
		},
		Responses: openapi3.NewResponses(
			openapi3.WithStatus(200, responseWithDescription("Return the product.")),
			openapi3.WithStatus(401, responseWithDescription("Unauthorized.")),
			openapi3.WithStatus(403, responseWithDescription("Forbidden - Insufficient permissions.")),
			openapi3.WithStatus(404, responseWithDescription("Product not found.")),
		),
	}
}

// updateProductOperation defines the OpenAPI operation for product updates.
func updateProductOperation() *openapi3.Operation {
	return &openapi3.Operation{
		OperationID: "ProductsController_update",
		Summary:     "Update a product",
		Tags:        []string{productsTag},
		Security:    bearerSecurityRequirements(),
		Parameters: openapi3.Parameters{
			pathParameter("id", "Product ID", openapi3.NewStringSchema()),
		},
		RequestBody: jsonRequestBodyRef("#/components/schemas/UpdateProductDto"),
		Responses: openapi3.NewResponses(
			openapi3.WithStatus(200, responseWithDescription("The product has been successfully updated.")),
			openapi3.WithStatus(400, responseWithDescription("Bad Request.")),
			openapi3.WithStatus(401, responseWithDescription("Unauthorized.")),
			openapi3.WithStatus(403, responseWithDescription("Forbidden - Insufficient permissions.")),
			openapi3.WithStatus(404, responseWithDescription("Product not found.")),
			openapi3.WithStatus(409, responseWithDescription("Conflict - SKU already exists.")),
		),
	}
}

// deleteProductOperation defines the OpenAPI operation for product deletion.
func deleteProductOperation() *openapi3.Operation {
	return &openapi3.Operation{
		OperationID: "ProductsController_remove",
		Summary:     "Delete a product",
		Tags:        []string{productsTag},
		Security:    bearerSecurityRequirements(),
		Parameters: openapi3.Parameters{
			pathParameter("id", "Product ID", openapi3.NewStringSchema()),
		},
		Responses: openapi3.NewResponses(
			openapi3.WithStatus(200, responseWithDescription("The product has been successfully deleted.")),
			openapi3.WithStatus(401, responseWithDescription("Unauthorized.")),
			openapi3.WithStatus(403, responseWithDescription("Forbidden - Insufficient permissions.")),
			openapi3.WithStatus(404, responseWithDescription("Product not found.")),
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
				"application/json": &openapi3.MediaType{Schema: &openapi3.SchemaRef{Ref: schemaRef}},
			}),
	}
}

// pathParameter builds a required path-parameter OpenAPI definition.
func pathParameter(name, description string, schema *openapi3.Schema) *openapi3.ParameterRef {
	return &openapi3.ParameterRef{Value: openapi3.NewPathParameter(name).WithDescription(description).WithSchema(schema)}
}

// createProductSchema returns the request schema for product creation payloads.
func createProductSchema() *openapi3.Schema {
	return openapi3.NewObjectSchema().
		WithProperty("sku", openapi3.NewStringSchema()).
		WithProperty("gallery", openapi3.NewArraySchema().WithItems(galleryItemSchema())).
		WithProperty("datasheets", openapi3.NewArraySchema().WithItems(datasheetSchema())).
		WithProperty("variations", openapi3.NewArraySchema().WithItems(openapi3.NewStringSchema())).
		WithProperty("variants", openapi3.NewArraySchema().WithItems(productVariantSchema())).
		WithRequired([]string{"sku"})
}

// updateProductSchema returns the request schema for product update payloads.
func updateProductSchema() *openapi3.Schema {
	return openapi3.NewObjectSchema().
		WithProperty("gallery", openapi3.NewArraySchema().WithItems(galleryItemSchema())).
		WithProperty("datasheets", openapi3.NewArraySchema().WithItems(datasheetSchema())).
		WithProperty("variations", openapi3.NewArraySchema().WithItems(openapi3.NewStringSchema())).
		WithProperty("variants", openapi3.NewArraySchema().WithItems(productVariantSchema()))
}

// productSchema returns the response schema for product payloads.
func productSchema() *openapi3.Schema {
	return openapi3.NewObjectSchema().
		WithProperty("_id", openapi3.NewStringSchema()).
		WithProperty("sku", openapi3.NewStringSchema()).
		WithProperty("gallery", openapi3.NewArraySchema().WithItems(galleryItemSchema())).
		WithProperty("datasheets", openapi3.NewArraySchema().WithItems(datasheetSchema())).
		WithProperty("variations", openapi3.NewArraySchema().WithItems(openapi3.NewStringSchema())).
		WithProperty("variants", openapi3.NewArraySchema().WithItems(productVariantSchema())).
		WithProperty("createdAt", openapi3.NewDateTimeSchema()).
		WithProperty("updatedAt", openapi3.NewDateTimeSchema()).
		WithProperty("isDeleted", openapi3.NewBoolSchema()).
		WithProperty("deletedAt", openapi3.NewDateTimeSchema())
}

// galleryItemSchema returns gallery-item schema values.
func galleryItemSchema() *openapi3.Schema {
	return openapi3.NewObjectSchema().
		WithProperty("assetId", openapi3.NewStringSchema()).
		WithProperty("position", openapi3.NewInt32Schema()).
		WithProperty("variationPosition", openapi3.NewInt32Schema()).
		WithProperty("isMain", openapi3.NewBoolSchema()).
		WithProperty("excludedRealms", openapi3.NewArraySchema().WithItems(openapi3.NewStringSchema())).
		WithProperty("variationIds", openapi3.NewArraySchema().WithItems(openapi3.NewStringSchema())).
		WithRequired([]string{"assetId"})
}

// datasheetSchema returns datasheet schema values.
func datasheetSchema() *openapi3.Schema {
	return openapi3.NewObjectSchema().
		WithProperty("realm", openapi3.NewStringSchema()).
		WithProperty("name", openapi3.NewStringSchema()).
		WithProperty("description", openapi3.NewStringSchema()).
		WithProperty("attributes", openapi3.NewObjectSchema()).
		WithRequired([]string{"realm", "name"})
}

// productVariantSchema returns product-variant schema values.
func productVariantSchema() *openapi3.Schema {
	return openapi3.NewObjectSchema().
		WithProperty("variationIds", openapi3.NewArraySchema().WithItems(openapi3.NewStringSchema())).
		WithProperty("sku", openapi3.NewStringSchema()).
		WithRequired([]string{"variationIds"})
}
