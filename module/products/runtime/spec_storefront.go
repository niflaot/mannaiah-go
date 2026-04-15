package runtime

import "github.com/getkin/kin-openapi/openapi3"

const (
	// storefrontTag defines the OpenAPI tag used by storefront endpoints.
	storefrontTag = "storefront"
)

// storefrontNavigationPathItem returns OpenAPI path operations for storefront navigation endpoints.
func storefrontNavigationPathItem() *openapi3.PathItem {
	return &openapi3.PathItem{
		Get: getStorefrontNavigationOperation(),
	}
}

// getStorefrontNavigationOperation defines the OpenAPI operation for storefront navigation retrieval.
func getStorefrontNavigationOperation() *openapi3.Operation {
	return &openapi3.Operation{
		OperationID: "StorefrontController_navigation",
		Summary:     "Get storefront navigation rewrite tree",
		Description: "Returns the cached storefront navigation snapshot for categories, products, and static pages. Static pages expose the bound renderable identifier only through renderableId; renderable payloads are not embedded in this response.",
		Tags:        []string{storefrontTag},
		Security:    bearerSecurityRequirements(),
		Responses: openapi3.NewResponses(
			openapi3.WithStatus(200, jsonResponseBodyRef("Return the storefront navigation snapshot.", "#/components/schemas/StorefrontNavigation")),
			openapi3.WithStatus(401, responseWithDescription("Unauthorized.")),
			openapi3.WithStatus(403, responseWithDescription("Forbidden - Insufficient permissions.")),
			openapi3.WithStatus(500, responseWithDescription("Storefront navigation is unavailable.")),
		),
	}
}

// storefrontNavigationSchema returns the response schema for storefront navigation snapshots.
func storefrontNavigationSchema() *openapi3.Schema {
	return openapi3.NewObjectSchema().
		WithProperty("realm", openapi3.NewStringSchema()).
		WithProperty("generatedAt", openapi3.NewDateTimeSchema()).
		WithProperty("categories", storefrontArrayRefSchema("#/components/schemas/StorefrontCategoryNode")).
		WithProperty("staticPages", storefrontArrayRefSchema("#/components/schemas/StorefrontStaticPageNode")).
		WithRequired([]string{"realm", "generatedAt", "categories", "staticPages"})
}

// storefrontCategoryNodeSchema returns the response schema for one navigation category node.
func storefrontCategoryNodeSchema() *openapi3.Schema {
	return openapi3.NewObjectSchema().
		WithProperty("id", openapi3.NewStringSchema()).
		WithProperty("name", openapi3.NewStringSchema()).
		WithProperty("slug", openapi3.NewStringSchema()).
		WithProperty("path", openapi3.NewStringSchema()).
		WithProperty("createdAt", openapi3.NewDateTimeSchema()).
		WithProperty("updatedAt", openapi3.NewDateTimeSchema()).
		WithProperty("products", storefrontArrayRefSchema("#/components/schemas/StorefrontProductNode")).
		WithProperty("children", storefrontArrayRefSchema("#/components/schemas/StorefrontCategoryNode")).
		WithRequired([]string{"id", "name", "slug", "path", "createdAt", "updatedAt", "products", "children"})
}

// storefrontProductNodeSchema returns the response schema for one navigation product node.
func storefrontProductNodeSchema() *openapi3.Schema {
	return openapi3.NewObjectSchema().
		WithProperty("id", openapi3.NewStringSchema()).
		WithProperty("sku", openapi3.NewStringSchema()).
		WithProperty("name", openapi3.NewStringSchema()).
		WithProperty("slug", openapi3.NewStringSchema()).
		WithProperty("path", openapi3.NewStringSchema()).
		WithProperty("createdAt", openapi3.NewDateTimeSchema()).
		WithProperty("updatedAt", openapi3.NewDateTimeSchema()).
		WithRequired([]string{"id", "sku", "name", "slug", "path", "createdAt", "updatedAt"})
}

// storefrontStaticPageNodeSchema returns the response schema for one navigation static-page node.
func storefrontStaticPageNodeSchema() *openapi3.Schema {
	schema := openapi3.NewObjectSchema().
		WithProperty("id", openapi3.NewStringSchema()).
		WithProperty("renderableId", openapi3.NewStringSchema()).
		WithProperty("title", openapi3.NewStringSchema()).
		WithProperty("url", openapi3.NewStringSchema()).
		WithProperty("createdAt", openapi3.NewDateTimeSchema()).
		WithProperty("updatedAt", openapi3.NewDateTimeSchema()).
		WithRequired([]string{"id", "renderableId", "title", "url", "createdAt", "updatedAt"})
	schema.Description = "Static-page navigation entry including the bound renderable identifier only; use renderableId to load the page body separately."

	return schema
}

// storefrontArrayRefSchema returns an array schema using a component reference for its item type.
func storefrontArrayRefSchema(ref string) *openapi3.Schema {
	schema := openapi3.NewArraySchema()
	schema.Items = &openapi3.SchemaRef{Ref: ref}

	return schema
}
