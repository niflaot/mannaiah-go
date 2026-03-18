package tag

import (
	"context"
	"errors"

	tagdomain "mannaiah/module/products/domain/tag"
)

var (
	// ErrNotFound is returned when tag records are missing.
	ErrNotFound = errors.New("tag not found")
	// ErrCorrelationNotFound is returned when correlation records are missing.
	ErrCorrelationNotFound = errors.New("tag correlation not found")
	// ErrDuplicateCorrelation is returned when a correlation pair already exists.
	ErrDuplicateCorrelation = errors.New("tag correlation pair already exists")
)

// Repository defines tag persistence contracts.
type Repository interface {
	// EnsureAll creates missing tags and reintegrates soft-deleted ones.
	EnsureAll(ctx context.Context, names []string) error
	// List returns all non-deleted tags ordered by name.
	List(ctx context.Context) ([]tagdomain.Tag, error)
	// SoftDelete soft-deletes a tag by name and cascades removal to product_tags.
	SoftDelete(ctx context.Context, name string) error
	// ListCorrelations returns all tag correlations ordered by source tag.
	ListCorrelations(ctx context.Context) ([]tagdomain.TagCorrelation, error)
	// ListCorrelationsBySource returns correlations for a specific source tag.
	ListCorrelationsBySource(ctx context.Context, sourceTag string) ([]tagdomain.TagCorrelation, error)
	// CreateCorrelation persists a new tag correlation and populates ID on success.
	CreateCorrelation(ctx context.Context, correlation *tagdomain.TagCorrelation) error
	// UpdateCorrelation updates correlation probability and notes by ID.
	UpdateCorrelation(ctx context.Context, id uint, probability *float64, notes *string) (*tagdomain.TagCorrelation, error)
	// DeleteCorrelation hard-deletes a correlation record by ID.
	DeleteCorrelation(ctx context.Context, id uint) error
}
