package service

import (
	"context"
	"encoding/json"
	"testing"

	"mannaiah/module/storefront/domain"
	"mannaiah/module/storefront/port"
)

// renderableRepositoryStub defines in-memory renderable persistence behavior for tests.
type renderableRepositoryStub struct {
	// renderables stores renderables by identifier.
	renderables map[string]domain.Renderable
	// versions stores published versions by renderable identifier.
	versions map[string][]domain.RenderableVersion
}

// Create persists one renderable root row.
func (s *renderableRepositoryStub) Create(_ context.Context, renderable *domain.Renderable) error {
	if s.renderables == nil {
		s.renderables = map[string]domain.Renderable{}
	}
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

// SavePublishedSnapshot atomically stores one published version and updates latest-published state.
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

// TestServicePublishAndRollbackLifecycle verifies publish and rollback behavior across multiple snapshots.
func TestServicePublishAndRollbackLifecycle(t *testing.T) {
	repository := newRenderableRepositoryStub()
	service, err := NewService(repository)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}

	created, err := service.Create(context.Background(), CreateCommand{
		Kind:     domain.StaticPageRenderableKind,
		Metadata: json.RawMessage(`{"title":"About"}`),
		Content:  json.RawMessage(`{"body":"first"}`),
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	firstVersion, err := service.Publish(context.Background(), created.ID)
	if err != nil {
		t.Fatalf("Publish() error = %v", err)
	}

	updated, err := service.Update(context.Background(), UpdateCommand{
		ID:       created.ID,
		Metadata: json.RawMessage(`{"title":"About Us"}`),
		Content:  json.RawMessage(`{"body":"second"}`),
	})
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	if !updated.Draft {
		t.Fatalf("updated.Draft = %v, want true", updated.Draft)
	}

	secondVersion, err := service.Publish(context.Background(), created.ID)
	if err != nil {
		t.Fatalf("Publish() second error = %v", err)
	}
	if secondVersion.ID == firstVersion.ID {
		t.Fatalf("secondVersion.ID = %q, want new version id", secondVersion.ID)
	}

	rolledBack, err := service.Rollback(context.Background(), created.ID, firstVersion.ID)
	if err != nil {
		t.Fatalf("Rollback() error = %v", err)
	}
	if rolledBack.SourceVersionID != firstVersion.ID {
		t.Fatalf("rolledBack.SourceVersionID = %q, want %q", rolledBack.SourceVersionID, firstVersion.ID)
	}

	current, err := service.GetByID(context.Background(), created.ID)
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}
	if string(current.Content) != `{"body":"first"}` {
		t.Fatalf("current.Content = %s, want first snapshot", string(current.Content))
	}
	if current.Draft {
		t.Fatalf("current.Draft = %v, want false", current.Draft)
	}
	if current.LatestPublishedVersionID != rolledBack.ID {
		t.Fatalf("current.LatestPublishedVersionID = %q, want %q", current.LatestPublishedVersionID, rolledBack.ID)
	}

	versions, total, err := service.ListVersions(context.Background(), created.ID, 1, 10)
	if err != nil {
		t.Fatalf("ListVersions() error = %v", err)
	}
	if total != 3 || len(versions) != 3 {
		t.Fatalf("ListVersions() = (%d,%d), want 3 versions", total, len(versions))
	}
}
