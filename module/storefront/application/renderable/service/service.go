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
	// ErrNilRepository is returned when required repository dependencies are nil.
	ErrNilRepository = errors.New("renderable repository must not be nil")
	// ErrRenderableNotFound is returned when a renderable cannot be found.
	ErrRenderableNotFound = errors.New("renderable not found")
	// ErrRenderableVersionNotFound is returned when a published version cannot be found.
	ErrRenderableVersionNotFound = errors.New("renderable version not found")
)

// CreateCommand defines renderable creation input values.
type CreateCommand struct {
	// Kind defines the renderable kind.
	Kind string
	// Metadata defines renderable metadata JSON.
	Metadata json.RawMessage
	// Content defines renderable content JSON.
	Content json.RawMessage
}

// UpdateCommand defines renderable mutation input values.
type UpdateCommand struct {
	// ID defines the renderable to update.
	ID string
	// Metadata defines renderable metadata JSON.
	Metadata json.RawMessage
	// Content defines renderable content JSON.
	Content json.RawMessage
}

// Service defines renderable use-case behavior.
type Service struct {
	// repository defines renderable persistence dependencies.
	repository port.RenderableRepository
}

// NewService creates renderable use-case services.
func NewService(repository port.RenderableRepository) (*Service, error) {
	if repository == nil {
		return nil, ErrNilRepository
	}

	return &Service{repository: repository}, nil
}

// Create persists a new draft renderable.
func (s *Service) Create(ctx context.Context, cmd CreateCommand) (*domain.Renderable, error) {
	now := time.Now().UTC()
	metadata, err := domain.NormalizeJSONObject(cmd.Metadata)
	if err != nil {
		return nil, domain.ErrRenderableMetadataInvalid
	}
	content, err := domain.NormalizeJSONDocument(cmd.Content)
	if err != nil {
		return nil, domain.ErrRenderableContentInvalid
	}

	renderable := domain.Renderable{
		ID:           uuid.NewString(),
		Kind:         strings.ToLower(strings.TrimSpace(cmd.Kind)),
		Metadata:     domain.CloneJSON(metadata),
		Content:      domain.CloneJSON(content),
		Draft:        true,
		SnapshotHash: domain.SnapshotHash(metadata, content),
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	renderable.Normalize()
	if err := renderable.Validate(); err != nil {
		return nil, err
	}

	if err := s.repository.Create(ctx, &renderable); err != nil {
		return nil, err
	}

	return &renderable, nil
}

// GetByID loads one renderable by identifier.
func (s *Service) GetByID(ctx context.Context, id string) (*domain.Renderable, error) {
	renderable, err := s.repository.GetByID(ctx, strings.TrimSpace(id))
	if err != nil {
		return nil, err
	}
	if renderable == nil {
		return nil, ErrRenderableNotFound
	}

	return renderable, nil
}

// Update applies draft changes to a renderable root row.
func (s *Service) Update(ctx context.Context, cmd UpdateCommand) (*domain.Renderable, error) {
	renderable, err := s.GetByID(ctx, cmd.ID)
	if err != nil {
		return nil, err
	}

	metadata, err := domain.NormalizeJSONObject(cmd.Metadata)
	if err != nil {
		return nil, domain.ErrRenderableMetadataInvalid
	}
	content, err := domain.NormalizeJSONDocument(cmd.Content)
	if err != nil {
		return nil, domain.ErrRenderableContentInvalid
	}

	renderable.Metadata = domain.CloneJSON(metadata)
	renderable.Content = domain.CloneJSON(content)
	renderable.SnapshotHash = domain.SnapshotHash(metadata, content)
	renderable.UpdatedAt = time.Now().UTC()
	renderable.Draft = true

	latest, latestErr := s.repository.GetLatestVersion(ctx, renderable.ID)
	if latestErr != nil {
		return nil, latestErr
	}
	if latest != nil && latest.SnapshotHash == renderable.SnapshotHash {
		renderable.Draft = false
	}

	renderable.Normalize()
	if err := renderable.Validate(); err != nil {
		return nil, err
	}

	if err := s.repository.Update(ctx, renderable); err != nil {
		return nil, err
	}

	return renderable, nil
}

// Delete removes a renderable and its dependent rows.
func (s *Service) Delete(ctx context.Context, id string) error {
	renderable, err := s.repository.GetByID(ctx, strings.TrimSpace(id))
	if err != nil {
		return err
	}
	if renderable == nil {
		return ErrRenderableNotFound
	}

	return s.repository.Delete(ctx, renderable.ID)
}

// List returns paginated renderables matching the provided query.
func (s *Service) List(ctx context.Context, query port.RenderableListQuery) ([]domain.Renderable, int64, error) {
	return s.repository.List(ctx, query)
}

// Publish creates a new published snapshot from the current renderable draft.
func (s *Service) Publish(ctx context.Context, id string) (*domain.RenderableVersion, error) {
	renderable, err := s.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	version := domain.RenderableVersion{
		ID:           uuid.NewString(),
		RenderableID: renderable.ID,
		Metadata:     domain.CloneJSON(renderable.Metadata),
		Content:      domain.CloneJSON(renderable.Content),
		SnapshotHash: renderable.SnapshotHash,
		PublishedAt:  now,
	}
	version.Normalize()
	if err := version.Validate(); err != nil {
		return nil, err
	}

	renderable.Draft = false
	renderable.LatestPublishedVersionID = version.ID
	renderable.LatestPublishedAt = &now
	renderable.UpdatedAt = now

	if err := s.repository.SavePublishedSnapshot(ctx, renderable, &version); err != nil {
		return nil, err
	}

	return &version, nil
}

// ListVersions returns paginated published versions for one renderable.
func (s *Service) ListVersions(ctx context.Context, id string, page int, pageSize int) ([]domain.RenderableVersion, int64, error) {
	renderable, err := s.repository.GetByID(ctx, strings.TrimSpace(id))
	if err != nil {
		return nil, 0, err
	}
	if renderable == nil {
		return nil, 0, ErrRenderableNotFound
	}

	return s.repository.ListVersions(ctx, renderable.ID, page, pageSize)
}

// GetVersionByID loads one published renderable version.
func (s *Service) GetVersionByID(ctx context.Context, id string, versionID string) (*domain.RenderableVersion, error) {
	renderable, err := s.repository.GetByID(ctx, strings.TrimSpace(id))
	if err != nil {
		return nil, err
	}
	if renderable == nil {
		return nil, ErrRenderableNotFound
	}

	version, err := s.repository.GetVersionByID(ctx, renderable.ID, strings.TrimSpace(versionID))
	if err != nil {
		return nil, err
	}
	if version == nil {
		return nil, ErrRenderableVersionNotFound
	}

	return version, nil
}

// Rollback copies one published version into a fresh published snapshot.
func (s *Service) Rollback(ctx context.Context, id string, versionID string) (*domain.RenderableVersion, error) {
	renderable, err := s.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	sourceVersion, err := s.GetVersionByID(ctx, renderable.ID, versionID)
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	newVersion := domain.RenderableVersion{
		ID:              uuid.NewString(),
		RenderableID:    renderable.ID,
		SourceVersionID: sourceVersion.ID,
		Metadata:        domain.CloneJSON(sourceVersion.Metadata),
		Content:         domain.CloneJSON(sourceVersion.Content),
		SnapshotHash:    sourceVersion.SnapshotHash,
		PublishedAt:     now,
	}
	newVersion.Normalize()
	if err := newVersion.Validate(); err != nil {
		return nil, err
	}

	renderable.Metadata = domain.CloneJSON(sourceVersion.Metadata)
	renderable.Content = domain.CloneJSON(sourceVersion.Content)
	renderable.SnapshotHash = sourceVersion.SnapshotHash
	renderable.Draft = false
	renderable.LatestPublishedVersionID = newVersion.ID
	renderable.LatestPublishedAt = &now
	renderable.UpdatedAt = now

	if err := s.repository.SavePublishedSnapshot(ctx, renderable, &newVersion); err != nil {
		return nil, err
	}

	return &newVersion, nil
}
