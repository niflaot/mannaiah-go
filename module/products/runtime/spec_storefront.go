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
		Tags:        []string{storefrontTag},
		Security:    bearerSecurityRequirements(),
		Responses: openapi3.NewResponses(
			openapi3.WithStatus(200, responseWithDescription("Return the storefront navigation snapshot.")),
			openapi3.WithStatus(401, responseWithDescription("Unauthorized.")),
			openapi3.WithStatus(403, responseWithDescription("Forbidden - Insufficient permissions.")),
			openapi3.WithStatus(500, responseWithDescription("Storefront navigation is unavailable.")),
		),
	}
}
