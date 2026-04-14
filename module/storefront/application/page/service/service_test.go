package service

import (
	"context"
	"encoding/json"
	"testing"

	"mannaiah/module/storefront/domain"
	"mannaiah/module/storefront/port"
)

// staticPageRepositoryStub defines in-memory static-page persistence behavior for tests.
type staticPageRepositoryStub struct {
	// pages stores pages by identifier.
	pages map[string]domain.StaticPage
}

// renderableRepositoryStub defines in-memory renderable lookup behavior for tests.
type renderableRepositoryStub struct {
	// renderables stores renderables by identifier.
	renderables map[string]domain.Renderable
	// versions stores versions by renderable identifier.
	versions map[string][]domain.RenderableVersion
}

// Create persists one renderable root row.
func (s *renderableRepositoryStub) Create(_ context.Context, renderable *domain.Renderable) error {
	s.renderables[renderable.ID] = *renderable
	return nil
}

// GetByID loads one renderable by identifier.
func (s *renderableRepositoryStub) GetByID(_ context.Context, id string) (*domain.Renderable, error) {
	renderable, ok := s.renderables[id]
	if !ok {
		return nil, nil
	}
	copy := renderable
	return &copy, nil
}

// Update persists mutable renderable root values.
func (s *renderableRepositoryStub) Update(_ context.Context, renderable *domain.Renderable) error {
	s.renderables[renderable.ID] = *renderable
	return nil
}

// Delete removes one renderable and its child rows.
func (s *renderableRepositoryStub) Delete(_ context.Context, id string) error {
	delete(s.renderables, id)
	delete(s.versions, id)
	return nil
}

// List returns paginated renderable rows.
func (s *renderableRepositoryStub) List(_ context.Context, _ port.RenderableListQuery) ([]domain.Renderable, int64, error) {
	rows := make([]domain.Renderable, 0, len(s.renderables))
	for _, renderable := range s.renderables {
		rows = append(rows, renderable)
	}
	return rows, int64(len(rows)), nil
}

// SavePublishedSnapshot atomically stores one published version and updates the renderable latest-published state.
func (s *renderableRepositoryStub) SavePublishedSnapshot(_ context.Context, renderable *domain.Renderable, version *domain.RenderableVersion) error {
	s.renderables[renderable.ID] = *renderable
	s.versions[renderable.ID] = append(s.versions[renderable.ID], *version)
	return nil
}

// GetVersionByID loads one published version by identifier.
func (s *renderableRepositoryStub) GetVersionByID(_ context.Context, renderableID string, versionID string) (*domain.RenderableVersion, error) {
	for _, version := range s.versions[renderableID] {
		if version.ID == versionID {
			copy := version
			return &copy, nil
		}
	}
	return nil, nil
}

// GetLatestVersion loads the latest published version for one renderable.
func (s *renderableRepositoryStub) GetLatestVersion(_ context.Context, renderableID string) (*domain.RenderableVersion, error) {
	versions := s.versions[renderableID]
	if len(versions) == 0 {
		return nil, nil
	}
	latest := versions[len(versions)-1]
	return &latest, nil
}

// ListVersions returns paginated published versions for one renderable.
func (s *renderableRepositoryStub) ListVersions(_ context.Context, renderableID string, _ int, _ int) ([]domain.RenderableVersion, int64, error) {
	versions := append([]domain.RenderableVersion(nil), s.versions[renderableID]...)
	return versions, int64(len(versions)), nil
}

// newRenderableRepositoryStub creates in-memory renderable repositories for tests.
func newRenderableRepositoryStub() *renderableRepositoryStub {
	return &renderableRepositoryStub{
		renderables: map[string]domain.Renderable{},
		versions:    map[string][]domain.RenderableVersion{},
	}
}

// Create persists one static page.
func (s *staticPageRepositoryStub) Create(_ context.Context, page *domain.StaticPage) error {
	s.pages[page.ID] = *page
	return nil
}

// GetByID loads one static page by identifier.
func (s *staticPageRepositoryStub) GetByID(_ context.Context, id string) (*domain.StaticPage, error) {
	for _, page := range s.pages {
		if page.ID == id {
			copy := page
			return &copy, nil
		}
	}
	return nil, nil
}

// GetByURL loads one static page by URL.
func (s *staticPageRepositoryStub) GetByURL(_ context.Context, url string) (*domain.StaticPage, error) {
	for _, page := range s.pages {
		if page.URL == url {
			copy := page
			return &copy, nil
		}
	}
	return nil, nil
}

// GetByRenderableID loads one static page by its renderable binding.
func (s *staticPageRepositoryStub) GetByRenderableID(_ context.Context, renderableID string) (*domain.StaticPage, error) {
	for _, page := range s.pages {
		if page.RenderableID == renderableID {
			copy := page
			return &copy, nil
		}
	}
	return nil, nil
}

// Update persists mutable page values.
func (s *staticPageRepositoryStub) Update(_ context.Context, page *domain.StaticPage) error {
	s.pages[page.ID] = *page
	return nil
}

// Delete removes one static page.
func (s *staticPageRepositoryStub) Delete(_ context.Context, id string) error {
	delete(s.pages, id)
	return nil
}

// List returns paginated static-page rows.
func (s *staticPageRepositoryStub) List(_ context.Context, _ port.StaticPageListQuery) ([]domain.StaticPage, int64, error) {
	rows := make([]domain.StaticPage, 0, len(s.pages))
	for _, page := range s.pages {
		rows = append(rows, page)
	}
	return rows, int64(len(rows)), nil
}

// TestServiceRejectsRenderableKindMismatch verifies static pages only bind to static-page renderables.
func TestServiceRejectsRenderableKindMismatch(t *testing.T) {
	renderables := newRenderableRepositoryStub()
	renderables.renderables["renderable-1"] = domain.Renderable{ID: "renderable-1", Kind: "campaign_banner", Metadata: json.RawMessage(`{}`), Content: json.RawMessage(`{}`)}

	service, err := NewService(&staticPageRepositoryStub{pages: map[string]domain.StaticPage{}}, renderables)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}

	_, err = service.Create(context.Background(), CreateCommand{
		RenderableID: "renderable-1",
		Title:        "About",
		URL:          "/about",
		SEOTags:      json.RawMessage(`{"robots":"index,follow"}`),
	})
	if err != ErrStaticPageRenderableKindMismatch {
		t.Fatalf("Create() error = %v, want %v", err, ErrStaticPageRenderableKindMismatch)
	}
}

// TestServiceRejectsURLConflict verifies static pages enforce unique URLs.
func TestServiceRejectsURLConflict(t *testing.T) {
	renderables := newRenderableRepositoryStub()
	renderables.renderables["renderable-1"] = domain.Renderable{ID: "renderable-1", Kind: domain.StaticPageRenderableKind, Metadata: json.RawMessage(`{}`), Content: json.RawMessage(`{}`)}
	renderables.renderables["renderable-2"] = domain.Renderable{ID: "renderable-2", Kind: domain.StaticPageRenderableKind, Metadata: json.RawMessage(`{}`), Content: json.RawMessage(`{}`)}
	pages := &staticPageRepositoryStub{pages: map[string]domain.StaticPage{}}

	service, err := NewService(pages, renderables)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}

	created, err := service.Create(context.Background(), CreateCommand{
		RenderableID: "renderable-1",
		Title:        "About",
		URL:          "/about",
		SEOTags:      json.RawMessage(`{"robots":"index,follow"}`),
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if created.ID == "" {
		t.Fatalf("expected created page id")
	}

	_, err = service.Create(context.Background(), CreateCommand{
		RenderableID: "renderable-2",
		Title:        "Company",
		URL:          "/about",
		SEOTags:      json.RawMessage(`{"robots":"index,follow"}`),
	})
	if err != ErrStaticPageURLConflict {
		t.Fatalf("Create() conflict error = %v, want %v", err, ErrStaticPageURLConflict)
	}
}
