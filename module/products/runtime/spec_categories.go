package runtime

import "github.com/getkin/kin-openapi/openapi3"

const (
	// categoriesTag defines the OpenAPI tag used by category endpoints.
	categoriesTag = "categories"
)

// categoriesPathItem returns OpenAPI path operations for category collection endpoints.
func categoriesPathItem() *openapi3.PathItem {
	return &openapi3.PathItem{
		Post: createCategoryOperation(),
		Get:  listCategoriesOperation(),
	}
}

// categoryByIDPathItem returns OpenAPI path operations for ID-scoped category endpoints.
func categoryByIDPathItem() *openapi3.PathItem {
	return &openapi3.PathItem{
		Get:    getCategoryOperation(),
		Patch:  updateCategoryOperation(),
		Delete: deleteCategoryOperation(),
	}
}

// categoryChildrenPathItem returns OpenAPI path operations for children listing.
func categoryChildrenPathItem() *openapi3.PathItem {
	return &openapi3.PathItem{
		Get: listCategoryChildrenOperation(),
	}
}

// categoryProductsPathItem returns OpenAPI path operations for category product listing.
func categoryProductsPathItem() *openapi3.PathItem {
	return &openapi3.PathItem{
		Get: listCategoryProductsOperation(),
	}
}

// createCategoryOperation defines the OpenAPI operation for category creation.
func createCategoryOperation() *openapi3.Operation {
	return &openapi3.Operation{
		OperationID: "CategoriesController_create",
		Summary:     "Create a new category",
		Tags:        []string{categoriesTag},
		Security:    bearerSecurityRequirements(),
		RequestBody: jsonRequestBodyRef("#/components/schemas/CreateCategoryDto"),
		Responses: openapi3.NewResponses(
			openapi3.WithStatus(201, responseWithDescription("The category has been successfully created.")),
			openapi3.WithStatus(400, responseWithDescription("Bad Request.")),
			openapi3.WithStatus(401, responseWithDescription("Unauthorized.")),
			openapi3.WithStatus(403, responseWithDescription("Forbidden - Insufficient permissions.")),
			openapi3.WithStatus(409, responseWithDescription("Conflict - Slug already exists.")),
		),
	}
}

// listCategoriesOperation defines the OpenAPI operation for category tree listing.
func listCategoriesOperation() *openapi3.Operation {
	return &openapi3.Operation{
		OperationID: "CategoriesController_tree",
		Summary:     "Get category tree (root-level categories)",
		Tags:        []string{categoriesTag},
		Security:    bearerSecurityRequirements(),
		Responses: openapi3.NewResponses(
			openapi3.WithStatus(200, responseWithDescription("Return all root-level categories.")),
			openapi3.WithStatus(401, responseWithDescription("Unauthorized.")),
			openapi3.WithStatus(403, responseWithDescription("Forbidden - Insufficient permissions.")),
		),
	}
}

// getCategoryOperation defines the OpenAPI operation for category retrieval by ID.
func getCategoryOperation() *openapi3.Operation {
	return &openapi3.Operation{
		OperationID: "CategoriesController_findOne",
		Summary:     "Get a category by id",
		Tags:        []string{categoriesTag},
		Security:    bearerSecurityRequirements(),
		Parameters: openapi3.Parameters{
			pathParameter("id", "Category ID", openapi3.NewStringSchema()),
		},
		Responses: openapi3.NewResponses(
			openapi3.WithStatus(200, responseWithDescription("Return the category.")),
			openapi3.WithStatus(401, responseWithDescription("Unauthorized.")),
			openapi3.WithStatus(403, responseWithDescription("Forbidden - Insufficient permissions.")),
			openapi3.WithStatus(404, responseWithDescription("Category not found.")),
		),
	}
}

// updateCategoryOperation defines the OpenAPI operation for category updates.
func updateCategoryOperation() *openapi3.Operation {
	return &openapi3.Operation{
		OperationID: "CategoriesController_update",
		Summary:     "Update a category",
		Tags:        []string{categoriesTag},
		Security:    bearerSecurityRequirements(),
		Parameters: openapi3.Parameters{
			pathParameter("id", "Category ID", openapi3.NewStringSchema()),
		},
		RequestBody: jsonRequestBodyRef("#/components/schemas/UpdateCategoryDto"),
		Responses: openapi3.NewResponses(
			openapi3.WithStatus(200, responseWithDescription("The category has been successfully updated.")),
			openapi3.WithStatus(400, responseWithDescription("Bad Request.")),
			openapi3.WithStatus(401, responseWithDescription("Unauthorized.")),
			openapi3.WithStatus(403, responseWithDescription("Forbidden - Insufficient permissions.")),
			openapi3.WithStatus(404, responseWithDescription("Category not found.")),
			openapi3.WithStatus(409, responseWithDescription("Conflict - Slug already exists.")),
		),
	}
}

// deleteCategoryOperation defines the OpenAPI operation for category deletion.
func deleteCategoryOperation() *openapi3.Operation {
	return &openapi3.Operation{
		OperationID: "CategoriesController_remove",
		Summary:     "Delete a category",
		Tags:        []string{categoriesTag},
		Security:    bearerSecurityRequirements(),
		Parameters: openapi3.Parameters{
			pathParameter("id", "Category ID", openapi3.NewStringSchema()),
		},
		Responses: openapi3.NewResponses(
			openapi3.WithStatus(200, responseWithDescription("The category has been successfully deleted.")),
			openapi3.WithStatus(401, responseWithDescription("Unauthorized.")),
			openapi3.WithStatus(403, responseWithDescription("Forbidden - Insufficient permissions.")),
			openapi3.WithStatus(404, responseWithDescription("Category not found.")),
			openapi3.WithStatus(409, responseWithDescription("Conflict - Category has children.")),
		),
	}
}

// listCategoryChildrenOperation defines the OpenAPI operation for listing category children.
func listCategoryChildrenOperation() *openapi3.Operation {
	return &openapi3.Operation{
		OperationID: "CategoriesController_children",
		Summary:     "Get children of a category",
		Tags:        []string{categoriesTag},
		Security:    bearerSecurityRequirements(),
		Parameters: openapi3.Parameters{
			pathParameter("id", "Parent Category ID", openapi3.NewStringSchema()),
		},
		Responses: openapi3.NewResponses(
			openapi3.WithStatus(200, responseWithDescription("Return direct children of the category.")),
			openapi3.WithStatus(401, responseWithDescription("Unauthorized.")),
			openapi3.WithStatus(403, responseWithDescription("Forbidden - Insufficient permissions.")),
			openapi3.WithStatus(404, responseWithDescription("Category not found.")),
		),
	}
}

// listCategoryProductsOperation defines the OpenAPI operation for listing products within a category.
func listCategoryProductsOperation() *openapi3.Operation {
	return &openapi3.Operation{
		OperationID: "CategoriesController_listProducts",
		Summary:     "List products in a category",
		Tags:        []string{categoriesTag},
		Security:    bearerSecurityRequirements(),
		Parameters: openapi3.Parameters{
			pathParameter("id", "Category ID", openapi3.NewStringSchema()),
			queryParameter("page", "Page number (1-based)", openapi3.NewInt32Schema()),
			queryParameter("pageSize", "Page size", openapi3.NewInt32Schema()),
		},
		Responses: openapi3.NewResponses(
			openapi3.WithStatus(200, responseWithDescription("Return paginated products for the category.")),
			openapi3.WithStatus(401, responseWithDescription("Unauthorized.")),
			openapi3.WithStatus(403, responseWithDescription("Forbidden - Insufficient permissions.")),
			openapi3.WithStatus(404, responseWithDescription("Category not found.")),
		),
	}
}

// queryParameter builds an optional query-parameter OpenAPI definition.
func queryParameter(name, description string, schema *openapi3.Schema) *openapi3.ParameterRef {
	return &openapi3.ParameterRef{Value: openapi3.NewQueryParameter(name).WithDescription(description).WithSchema(schema)}
}

// createCategorySchema returns the request schema for category creation payloads.
func createCategorySchema() *openapi3.Schema {
	return openapi3.NewObjectSchema().
		WithProperty("slug", openapi3.NewStringSchema()).
		WithProperty("name", openapi3.NewStringSchema()).
		WithProperty("description", openapi3.NewStringSchema()).
		WithProperty("parentId", openapi3.NewStringSchema()).
		WithProperty("includeChildren", openapi3.NewBoolSchema()).
		WithProperty("filterTags", openapi3.NewArraySchema().WithItems(openapi3.NewStringSchema())).
		WithProperty("filterMinPrice", openapi3.NewFloat64Schema()).
		WithProperty("filterMaxPrice", openapi3.NewFloat64Schema()).
		WithProperty("filterCategoryRefs", openapi3.NewArraySchema().WithItems(openapi3.NewStringSchema())).
		WithProperty("productIds", openapi3.NewArraySchema().WithItems(openapi3.NewStringSchema())).
		WithRequired([]string{"slug", "name"})
}

// updateCategorySchema returns the request schema for category update payloads.
func updateCategorySchema() *openapi3.Schema {
	return openapi3.NewObjectSchema().
		WithProperty("slug", openapi3.NewStringSchema()).
		WithProperty("name", openapi3.NewStringSchema()).
		WithProperty("description", openapi3.NewStringSchema()).
		WithProperty("parentId", openapi3.NewStringSchema()).
		WithProperty("includeChildren", openapi3.NewBoolSchema()).
		WithProperty("filterTags", openapi3.NewArraySchema().WithItems(openapi3.NewStringSchema())).
		WithProperty("filterMinPrice", openapi3.NewFloat64Schema()).
		WithProperty("filterMaxPrice", openapi3.NewFloat64Schema()).
		WithProperty("filterCategoryRefs", openapi3.NewArraySchema().WithItems(openapi3.NewStringSchema())).
		WithProperty("productIds", openapi3.NewArraySchema().WithItems(openapi3.NewStringSchema()))
}

// categorySchema returns the response schema for category payloads.
func categorySchema() *openapi3.Schema {
	filterSchema := openapi3.NewObjectSchema().
		WithProperty("tags", openapi3.NewArraySchema().WithItems(openapi3.NewStringSchema())).
		WithProperty("priceRange", openapi3.NewObjectSchema().
			WithProperty("min", openapi3.NewFloat64Schema()).
			WithProperty("max", openapi3.NewFloat64Schema())).
		WithProperty("categoryRefs", openapi3.NewArraySchema().WithItems(openapi3.NewStringSchema()))

	return openapi3.NewObjectSchema().
		WithProperty("id", openapi3.NewStringSchema()).
		WithProperty("slug", openapi3.NewStringSchema()).
		WithProperty("name", openapi3.NewStringSchema()).
		WithProperty("description", openapi3.NewStringSchema()).
		WithProperty("parentId", openapi3.NewStringSchema()).
		WithProperty("includeChildren", openapi3.NewBoolSchema()).
		WithProperty("filter", filterSchema).
		WithProperty("productIds", openapi3.NewArraySchema().WithItems(openapi3.NewStringSchema())).
		WithProperty("createdAt", openapi3.NewDateTimeSchema()).
		WithProperty("updatedAt", openapi3.NewDateTimeSchema())
}
