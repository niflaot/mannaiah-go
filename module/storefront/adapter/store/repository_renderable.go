package store

import (
	"context"
	"errors"
	"fmt"
	"strings"

	coresearch "mannaiah/module/core/search"
	"mannaiah/module/storefront/domain"
	"mannaiah/module/storefront/port"

	"gorm.io/gorm"
)

// Create persists one renderable root row.
func (r *RenderableRepository) Create(ctx context.Context, renderable *domain.Renderable) error {
	record := toRenderableRecord(*renderable)
	if err := r.db.WithContext(ctx).Create(&record).Error; err != nil {
		return fmt.Errorf("create renderable: %w", err)
	}

	return nil
}

// GetByID loads one renderable by identifier.
func (r *RenderableRepository) GetByID(ctx context.Context, id string) (*domain.Renderable, error) {
	var record renderableRecord
	if err := r.db.WithContext(ctx).Where("id = ?", strings.TrimSpace(id)).First(&record).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("get renderable: %w", err)
	}

	entity := toRenderableEntity(record)
	return &entity, nil
}

// Update persists mutable renderable root values.
func (r *RenderableRepository) Update(ctx context.Context, renderable *domain.Renderable) error {
	updates := map[string]any{
		"kind":                        renderable.Kind,
		"metadata_json":               string(renderable.Metadata),
		"content_json":                string(renderable.Content),
		"snapshot_hash":               renderable.SnapshotHash,
		"draft":                       renderable.Draft,
		"latest_published_version_id": stringPtrOrEmpty(renderable.LatestPublishedVersionID),
		"latest_published_at":         renderable.LatestPublishedAt,
		"updated_at":                  renderable.UpdatedAt,
	}

	if err := r.db.WithContext(ctx).
		Model(&renderableRecord{}).
		Where("id = ?", renderable.ID).
		Updates(updates).Error; err != nil {
		return fmt.Errorf("update renderable: %w", err)
	}

	return nil
}

// Delete removes one renderable and all dependent rows.
func (r *RenderableRepository) Delete(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		trimmedID := strings.TrimSpace(id)
		if err := tx.Where("renderable_id = ?", trimmedID).Delete(&staticPageRecord{}).Error; err != nil {
			return fmt.Errorf("delete static page bindings: %w", err)
		}
		if err := tx.Where("renderable_id = ?", trimmedID).Delete(&renderableVersionRecord{}).Error; err != nil {
			return fmt.Errorf("delete renderable versions: %w", err)
		}
		if err := tx.Where("id = ?", trimmedID).Delete(&renderableRecord{}).Error; err != nil {
			return fmt.Errorf("delete renderable: %w", err)
		}
		return nil
	})
}

// List returns paginated renderables matching the provided query.
func (r *RenderableRepository) List(ctx context.Context, query port.RenderableListQuery) ([]domain.Renderable, int64, error) {
	tx := r.db.WithContext(ctx).Model(&renderableRecord{})
	if kind := strings.TrimSpace(query.Kind); kind != "" {
		tx = tx.Where("kind = ?", strings.ToLower(kind))
	}
	if query.Draft != nil {
		tx = tx.Where("draft = ?", *query.Draft)
	}

	var total int64
	if err := tx.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count renderables: %w", err)
	}
	if total == 0 {
		return []domain.Renderable{}, 0, nil
	}

	page, pageSize := coresearch.NormalizePagination(query.Page, query.PageSize)
	offset := (page - 1) * pageSize

	var records []renderableRecord
	if err := tx.Order("updated_at DESC, id DESC").Limit(pageSize).Offset(offset).Find(&records).Error; err != nil {
		return nil, 0, fmt.Errorf("list renderables: %w", err)
	}

	rows := make([]domain.Renderable, 0, len(records))
	for _, record := range records {
		rows = append(rows, toRenderableEntity(record))
	}

	return rows, total, nil
}

// SavePublishedSnapshot atomically stores one published version and updates latest published state.
func (r *RenderableRepository) SavePublishedSnapshot(ctx context.Context, renderable *domain.Renderable, version *domain.RenderableVersion) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		versionRecord := toRenderableVersionRecord(*version)
		if err := tx.Create(&versionRecord).Error; err != nil {
			return fmt.Errorf("create renderable version: %w", err)
		}

		updates := map[string]any{
			"metadata_json":               string(renderable.Metadata),
			"content_json":                string(renderable.Content),
			"snapshot_hash":               renderable.SnapshotHash,
			"draft":                       false,
			"latest_published_version_id": stringPtrOrEmpty(version.ID),
			"latest_published_at":         version.PublishedAt,
			"updated_at":                  renderable.UpdatedAt,
		}

		if err := tx.Model(&renderableRecord{}).Where("id = ?", renderable.ID).Updates(updates).Error; err != nil {
			return fmt.Errorf("update renderable published snapshot: %w", err)
		}

		return nil
	})
}

// GetVersionByID loads one published version by identifier.
func (r *RenderableRepository) GetVersionByID(ctx context.Context, renderableID string, versionID string) (*domain.RenderableVersion, error) {
	var record renderableVersionRecord
	if err := r.db.WithContext(ctx).
		Where("renderable_id = ? AND id = ?", strings.TrimSpace(renderableID), strings.TrimSpace(versionID)).
		First(&record).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("get renderable version: %w", err)
	}

	entity := toRenderableVersionEntity(record)
	return &entity, nil
}

// GetLatestVersion loads the latest published version for one renderable.
func (r *RenderableRepository) GetLatestVersion(ctx context.Context, renderableID string) (*domain.RenderableVersion, error) {
	var record renderableVersionRecord
	if err := r.db.WithContext(ctx).
		Where("renderable_id = ?", strings.TrimSpace(renderableID)).
		Order("published_at DESC, id DESC").
		First(&record).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("get latest renderable version: %w", err)
	}

	entity := toRenderableVersionEntity(record)
	return &entity, nil
}

// ListVersions returns paginated published versions for one renderable.
func (r *RenderableRepository) ListVersions(ctx context.Context, renderableID string, page int, pageSize int) ([]domain.RenderableVersion, int64, error) {
	tx := r.db.WithContext(ctx).Model(&renderableVersionRecord{}).Where("renderable_id = ?", strings.TrimSpace(renderableID))

	var total int64
	if err := tx.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count renderable versions: %w", err)
	}
	if total == 0 {
		return []domain.RenderableVersion{}, 0, nil
	}

	resolvedPage, resolvedPageSize := coresearch.NormalizePagination(page, pageSize)
	offset := (resolvedPage - 1) * resolvedPageSize

	var records []renderableVersionRecord
	if err := tx.Order("published_at DESC, id DESC").Limit(resolvedPageSize).Offset(offset).Find(&records).Error; err != nil {
		return nil, 0, fmt.Errorf("list renderable versions: %w", err)
	}

	rows := make([]domain.RenderableVersion, 0, len(records))
	for _, record := range records {
		rows = append(rows, toRenderableVersionEntity(record))
	}

	return rows, total, nil
}
