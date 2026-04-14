// Package storefront provides storefront renderable and static-page management features.
package storefront

import (
	storefrontruntime "mannaiah/module/storefront/runtime"

	"gorm.io/gorm"
)

// Module defines composition-root wiring for storefront endpoints and services.
type Module = storefrontruntime.Module

// Loader defines bootstrap hooks required by the storefront module.
type Loader = storefrontruntime.Loader

// New creates a storefront module with the provided database connection.
func New(db *gorm.DB) (*Module, error) {
	return storefrontruntime.New(db)
}
