package syncrecord

import (
	"github.com/getkin/kin-openapi/openapi3"
	"gorm.io/gorm"
	syncrecordruntime "mannaiah/module/syncrecord/runtime"
)

// Module defines composition-root wiring for sync record endpoints.
type Module = syncrecordruntime.Module

// Loader defines bootstrap hooks required by sync record modules.
type Loader = syncrecordruntime.Loader

// New creates a sync record module with adapter wiring.
func New(cfg Config, db *gorm.DB) (*Module, error) {
	return syncrecordruntime.New(cfg, db)
}

// OpenAPISpec returns sync record OpenAPI documentation.
func OpenAPISpec() *openapi3.T {
	return syncrecordruntime.OpenAPISpec()
}
