package service

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"mannaiah/module/storefront/domain"
	"mannaiah/module/storefront/port"
)

var (
	// ErrNilPageRepository is returned when static-page repository dependencies are nil.
	ErrNilPageRepository = errors.New("static page repository must not be nil")
	// ErrNilRenderableRepository is returned when renderable repository dependencies are nil.
	ErrNilRenderableRepository = errors.New("renderable repository must not be nil")
	// ErrStaticPageNotFound is returned when a static page cannot be found.
	ErrStaticPageNotFound = errors.New("static page not found")
	// ErrStaticPageURLConflict is returned when a static page URL already exists.
	ErrStaticPageURLConflict = errors.New("static page url already exists")
	// ErrStaticPageRenderableNotFound is returned when a bound renderable cannot be found.
	ErrStaticPageRenderableNotFound = errors.New("static page renderable not found")
	// ErrStaticPageRenderableConflict is returned when a renderable is already bound to a page.
	ErrStaticPageRenderableConflict = errors.New("renderable already bound to another static page")
	// ErrStaticPageRenderableKindMismatch is returned when a renderable kind is not compatible with static pages.
	ErrStaticPageRenderableKindMismatch = errors.New("renderable kind is not valid for static pages")
)

// CreateCommand defines static-page creation input values.
type CreateCommand struct {
	// RenderableID defines the bound renderable identifier.
	RenderableID string
	// Title defines page title values.
	Title string
	// URL defines storefront URL values.
	URL string
	// SEOTags defines frontend-provided SEO JSON.
	SEOTags json.RawMessage
}

// UpdateCommand defines static-page mutation input values.
type UpdateCommand struct {
	// ID defines the page to update.
	ID string
	// RenderableID defines the bound renderable identifier.
	RenderableID string
	// Title defines page title values.
	Title string
	// URL defines storefront URL values.
	URL string
	// SEOTags defines frontend-provided SEO JSON.
	SEOTags json.RawMessage
}

// Service defines static-page use-case behavior.
type Service struct {
	// pages defines static-page persistence dependencies.
	pages port.StaticPageRepository
	// renderables defines renderable lookup dependencies.
	renderables port.RenderableRepository
}

// NewService creates static-page use-case services.
func NewService(pages port.StaticPageRepository, renderables port.RenderableRepository) (*Service, error) {
	if pages == nil {
		return nil, ErrNilPageRepository
	}
	if renderables == nil {
		return nil, ErrNilRenderableRepository
	}

	return &Service{pages: pages, renderables: renderables}, nil
}

// Create persists a new static page bound to an existing static-page renderable.
func (s *Service) Create(ctx context.Context, cmd CreateCommand) (*domain.StaticPage, error) {
	renderable, err := s.requireRenderable(ctx, cmd.RenderableID)
	if err != nil {
		return nil, err
	}
	_ = renderable

	url := strings.TrimSpace(cmd.URL)
	existingByURL, err := s.pages.GetByURL(ctx, url)
	if err != nil {
		return nil, err
	}
	if existingByURL != nil {
		return nil, ErrStaticPageURLConflict
	}

	existingByRenderable, err := s.pages.GetByRenderableID(ctx, strings.TrimSpace(cmd.RenderableID))
	if err != nil {
		return nil, err
	}
	if existingByRenderable != nil {
		return nil, ErrStaticPageRenderableConflict
	}

	seoTags, err := domain.NormalizeJSONObject(cmd.SEOTags)
	if err != nil {
		return nil, domain.ErrStaticPageSEOTagsInvalid
	}

	now := time.Now().UTC()
	page := domain.StaticPage{
		ID:           uuid.NewString(),
		RenderableID: strings.TrimSpace(cmd.RenderableID),
		Title:        strings.TrimSpace(cmd.Title),
		URL:          url,
		SEOTags:      domain.CloneJSON(seoTags),
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	page.Normalize()
	if err := page.Validate(); err != nil {
		return nil, err
	}

	if err := s.pages.Create(ctx, &page); err != nil {
		return nil, err
	}

	return &page, nil
}

// GetByID loads one static page by identifier.
func (s *Service) GetByID(ctx context.Context, id string) (*domain.StaticPage, error) {
	page, err := s.pages.GetByID(ctx, strings.TrimSpace(id))
	if err != nil {
		return nil, err
	}
	if page == nil {
		return nil, ErrStaticPageNotFound
	}

	return page, nil
}

// Update applies static-page metadata changes.
func (s *Service) Update(ctx context.Context, cmd UpdateCommand) (*domain.StaticPage, error) {
	page, err := s.GetByID(ctx, cmd.ID)
	if err != nil {
		return nil, err
	}

	renderableID := strings.TrimSpace(cmd.RenderableID)
	if renderableID != page.RenderableID {
		if _, err := s.requireRenderable(ctx, renderableID); err != nil {
			return nil, err
		}
		existingByRenderable, renderableErr := s.pages.GetByRenderableID(ctx, renderableID)
		if renderableErr != nil {
			return nil, renderableErr
		}
		if existingByRenderable != nil && existingByRenderable.ID != page.ID {
			return nil, ErrStaticPageRenderableConflict
		}
	}

	url := strings.TrimSpace(cmd.URL)
	if url != page.URL {
		existingByURL, urlErr := s.pages.GetByURL(ctx, url)
		if urlErr != nil {
			return nil, urlErr
		}
		if existingByURL != nil && existingByURL.ID != page.ID {
			return nil, ErrStaticPageURLConflict
		}
	}

	seoTags, err := domain.NormalizeJSONObject(cmd.SEOTags)
	if err != nil {
		return nil, domain.ErrStaticPageSEOTagsInvalid
	}

	page.RenderableID = renderableID
	page.Title = strings.TrimSpace(cmd.Title)
	page.URL = url
	page.SEOTags = domain.CloneJSON(seoTags)
	page.UpdatedAt = time.Now().UTC()
	page.Normalize()
	if err := page.Validate(); err != nil {
		return nil, err
	}

	if err := s.pages.Update(ctx, page); err != nil {
		return nil, err
	}

	return page, nil
}

// Delete removes one static page.
func (s *Service) Delete(ctx context.Context, id string) error {
	page, err := s.pages.GetByID(ctx, strings.TrimSpace(id))
	if err != nil {
		return err
	}
	if page == nil {
		return ErrStaticPageNotFound
	}

	return s.pages.Delete(ctx, page.ID)
}

// List returns paginated static pages matching the provided query.
func (s *Service) List(ctx context.Context, query port.StaticPageListQuery) ([]domain.StaticPage, int64, error) {
	return s.pages.List(ctx, query)
}

// requireRenderable verifies a renderable exists and is compatible with static-page bindings.
func (s *Service) requireRenderable(ctx context.Context, renderableID string) (*domain.Renderable, error) {
	renderable, err := s.renderables.GetByID(ctx, strings.TrimSpace(renderableID))
	if err != nil {
		return nil, err
	}
	if renderable == nil {
		return nil, ErrStaticPageRenderableNotFound
	}
	if renderable.Kind != domain.StaticPageRenderableKind {
		return nil, ErrStaticPageRenderableKindMismatch
	}

	return renderable, nil
}
