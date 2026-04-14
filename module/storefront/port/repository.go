package port

import (
	"context"

	"mannaiah/module/storefront/domain"
)

// RenderableListQuery defines renderable listing filters.
type RenderableListQuery struct {
	// Kind filters rows by renderable kind when provided.
	Kind string
	// Draft filters rows by draft state when provided.
	Draft *bool
	// Page defines the 1-based page number.
	Page int
	// PageSize defines the page size.
	PageSize int
}

// StaticPageListQuery defines static-page listing filters.
type StaticPageListQuery struct {
	// Term filters by partial title or URL values.
	Term string
	// RenderableID filters by bound renderable when provided.
	RenderableID string
	// Page defines the 1-based page number.
	Page int
	// PageSize defines the page size.
	PageSize int
}

// RenderableRepository defines persistence behavior for renderables and version snapshots.
type RenderableRepository interface {
	// Create persists one renderable root row.
	Create(ctx context.Context, renderable *domain.Renderable) error
	// GetByID loads one renderable by identifier.
	GetByID(ctx context.Context, id string) (*domain.Renderable, error)
	// Update persists mutable renderable root values.
	Update(ctx context.Context, renderable *domain.Renderable) error
	// Delete removes one renderable and its child rows.
	Delete(ctx context.Context, id string) error
	// List returns paginated renderable rows.
	List(ctx context.Context, query RenderableListQuery) ([]domain.Renderable, int64, error)
	// SavePublishedSnapshot atomically stores one published version and updates the renderable latest-published state.
	SavePublishedSnapshot(ctx context.Context, renderable *domain.Renderable, version *domain.RenderableVersion) error
	// GetVersionByID loads one published version by identifier.
	GetVersionByID(ctx context.Context, renderableID string, versionID string) (*domain.RenderableVersion, error)
	// GetLatestVersion loads the latest published version for one renderable.
	GetLatestVersion(ctx context.Context, renderableID string) (*domain.RenderableVersion, error)
	// ListVersions returns paginated published versions for one renderable.
	ListVersions(ctx context.Context, renderableID string, page int, pageSize int) ([]domain.RenderableVersion, int64, error)
}

// StaticPageRepository defines persistence behavior for static pages.
type StaticPageRepository interface {
	// Create persists one static page.
	Create(ctx context.Context, page *domain.StaticPage) error
	// GetByID loads one static page by identifier.
	GetByID(ctx context.Context, id string) (*domain.StaticPage, error)
	// GetByURL loads one static page by URL.
	GetByURL(ctx context.Context, url string) (*domain.StaticPage, error)
	// GetByRenderableID loads one static page by its renderable binding.
	GetByRenderableID(ctx context.Context, renderableID string) (*domain.StaticPage, error)
	// Update persists mutable page values.
	Update(ctx context.Context, page *domain.StaticPage) error
	// Delete removes one static page.
	Delete(ctx context.Context, id string) error
	// List returns paginated static-page rows.
	List(ctx context.Context, query StaticPageListQuery) ([]domain.StaticPage, int64, error)
}
