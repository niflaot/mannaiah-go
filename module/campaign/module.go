package campaign

import (
	"github.com/getkin/kin-openapi/openapi3"
	"gorm.io/gorm"
	campaignruntime "mannaiah/module/campaign/runtime"
)

// Module defines composition-root wiring for campaign endpoints.
type Module = campaignruntime.Module

// Loader defines bootstrap hooks required by campaign modules.
type Loader = campaignruntime.Loader

// New creates campaign modules with adapter wiring.
func New(cfg Config, db *gorm.DB, resolver SegmentResolver, sender EmailSender) (*Module, error) {
	return campaignruntime.New(cfg, db, resolver, sender)
}

// OpenAPISpec returns campaign OpenAPI documentation.
func OpenAPISpec() *openapi3.T {
	return campaignruntime.OpenAPISpec()
}
