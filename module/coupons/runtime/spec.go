package runtime

import (
	"github.com/getkin/kin-openapi/openapi3"
	couponhttp "mannaiah/module/coupons/adapter/http"
)

// OpenAPISpec returns the aggregated OpenAPI specification for the coupons module.
func OpenAPISpec() *openapi3.T {
	schemas := couponhttp.Schemas()
	return &openapi3.T{
		OpenAPI: "3.0.3",
		Info: &openapi3.Info{
			Title:   "Coupons API",
			Version: "1.3.0",
		},
		Paths: couponhttp.Paths(),
		Components: &openapi3.Components{
			SecuritySchemes: couponhttp.SecuritySchemes(),
			Schemas:         schemas,
		},
		Tags: couponhttp.Tags(),
	}
}
