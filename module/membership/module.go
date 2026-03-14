package membership

import (
	"github.com/getkin/kin-openapi/openapi3"
	"gorm.io/gorm"
	membershipruntime "mannaiah/module/membership/runtime"
)

// Module defines composition-root wiring for membership endpoints.
type Module = membershipruntime.Module

// Loader defines bootstrap hooks required by membership modules.
type Loader = membershipruntime.Loader

// New creates a membership module with adapter wiring.
func New(cfg Config, db *gorm.DB, contacts ContactLookup, publishers ...IntegrationEventPublisher) (*Module, error) {
	return membershipruntime.New(cfg, db, contacts, publishers...)
}

// OpenAPISpec returns membership OpenAPI documentation.
func OpenAPISpec() *openapi3.T {
	return membershipruntime.OpenAPISpec()
}
