package segment

import (
	"github.com/getkin/kin-openapi/openapi3"
	"gorm.io/gorm"
	analyticsport "mannaiah/module/analytics/port"
	segmentruntime "mannaiah/module/segment/runtime"
)

// Module defines composition-root wiring for segment endpoints.
type Module = segmentruntime.Module

// Loader defines bootstrap hooks required by segment modules.
type Loader = segmentruntime.Loader

// New creates segment modules with adapter wiring.
func New(cfg Config, db *gorm.DB, resolver analyticsport.Resolver) (*Module, error) {
	return segmentruntime.New(cfg, db, resolver)
}

// OpenAPISpec returns segment OpenAPI documentation.
func OpenAPISpec() *openapi3.T {
	return segmentruntime.OpenAPISpec()
}
