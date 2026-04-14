package store

import (
	"errors"

	"gorm.io/gorm"
	"mannaiah/module/storefront/port"
)

var (
	// ErrNilDB is returned when required database dependencies are nil.
	ErrNilDB = errors.New("storefront repository: db must not be nil")
	// _ verifies renderable repository compliance.
	_ port.RenderableRepository = (*RenderableRepository)(nil)
	// _ verifies static-page repository compliance.
	_ port.StaticPageRepository = (*StaticPageRepository)(nil)
)

// RenderableRepository defines renderable persistence adapters.
type RenderableRepository struct {
	// db defines GORM database dependencies.
	db *gorm.DB
}

// StaticPageRepository defines static-page persistence adapters.
type StaticPageRepository struct {
	// db defines GORM database dependencies.
	db *gorm.DB
}

// NewRenderableRepository creates renderable persistence adapters.
func NewRenderableRepository(db *gorm.DB) (*RenderableRepository, error) {
	if db == nil {
		return nil, ErrNilDB
	}

	return &RenderableRepository{db: db}, nil
}

// NewStaticPageRepository creates static-page persistence adapters.
func NewStaticPageRepository(db *gorm.DB) (*StaticPageRepository, error) {
	if db == nil {
		return nil, ErrNilDB
	}

	return &StaticPageRepository{db: db}, nil
}
