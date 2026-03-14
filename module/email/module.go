package email

import (
	"github.com/getkin/kin-openapi/openapi3"
	"gorm.io/gorm"
	emailruntime "mannaiah/module/email/runtime"
)

// Module defines composition-root wiring for email endpoints.
type Module = emailruntime.Module

// Loader defines bootstrap hooks required by email modules.
type Loader = emailruntime.Loader

// New creates email modules with adapter wiring.
func New(cfg Config, db *gorm.DB) (*Module, error) {
	return emailruntime.New(cfg, db)
}

// OpenAPISpec returns email OpenAPI documentation.
func OpenAPISpec() *openapi3.T {
	return emailruntime.OpenAPISpec()
}
