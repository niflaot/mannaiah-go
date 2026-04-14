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

// Create persists one static page.
func (r *StaticPageRepository) Create(ctx context.Context, page *domain.StaticPage) error {
	record := toStaticPageRecord(*page)
	if err := r.db.WithContext(ctx).Create(&record).Error; err != nil {
		return fmt.Errorf("create static page: %w", err)
	}

	return nil
}

// GetByID loads one static page by identifier.
func (r *StaticPageRepository) GetByID(ctx context.Context, id string) (*domain.StaticPage, error) {
	return r.loadStaticPage(ctx, func(tx *gorm.DB) *gorm.DB {
		return tx.Where("id = ?", strings.TrimSpace(id))
	})
}

// GetByURL loads one static page by URL.
func (r *StaticPageRepository) GetByURL(ctx context.Context, url string) (*domain.StaticPage, error) {
	return r.loadStaticPage(ctx, func(tx *gorm.DB) *gorm.DB {
		return tx.Where("url = ?", strings.TrimSpace(url))
	})
}

// GetByRenderableID loads one static page by its renderable binding.
func (r *StaticPageRepository) GetByRenderableID(ctx context.Context, renderableID string) (*domain.StaticPage, error) {
	return r.loadStaticPage(ctx, func(tx *gorm.DB) *gorm.DB {
		return tx.Where("renderable_id = ?", strings.TrimSpace(renderableID))
	})
}

// Update persists mutable page values.
func (r *StaticPageRepository) Update(ctx context.Context, page *domain.StaticPage) error {
	updates := map[string]any{
		"renderable_id": page.RenderableID,
		"title":         page.Title,
		"url":           page.URL,
		"seo_tags_json": string(page.SEOTags),
		"archived_at":   page.ArchivedAt,
		"updated_at":    page.UpdatedAt,
	}

	if err := r.db.WithContext(ctx).Model(&staticPageRecord{}).Where("id = ?", page.ID).Updates(updates).Error; err != nil {
		return fmt.Errorf("update static page: %w", err)
	}

	return nil
}

// Archive persists archived state for one static page.
func (r *StaticPageRepository) Archive(ctx context.Context, page *domain.StaticPage) error {
	updates := map[string]any{
		"archived_at": page.ArchivedAt,
		"updated_at":  page.UpdatedAt,
	}

	if err := r.db.WithContext(ctx).Model(&staticPageRecord{}).Where("id = ?", page.ID).Updates(updates).Error; err != nil {
		return fmt.Errorf("archive static page: %w", err)
	}

	return nil
}

// Delete removes one static page.
func (r *StaticPageRepository) Delete(ctx context.Context, id string) error {
	if err := r.db.WithContext(ctx).Where("id = ?", strings.TrimSpace(id)).Delete(&staticPageRecord{}).Error; err != nil {
		return fmt.Errorf("delete static page: %w", err)
	}

	return nil
}

// List returns paginated static-page rows.
func (r *StaticPageRepository) List(ctx context.Context, query port.StaticPageListQuery) ([]domain.StaticPage, int64, error) {
	tx := r.db.WithContext(ctx).Model(&staticPageRecord{})
	if renderableID := strings.TrimSpace(query.RenderableID); renderableID != "" {
		tx = tx.Where("renderable_id = ?", renderableID)
	}
	if query.Archived != nil {
		if *query.Archived {
			tx = tx.Where("archived_at IS NOT NULL")
		} else {
			tx = tx.Where("archived_at IS NULL")
		}
	}
	if term := strings.TrimSpace(query.Term); term != "" {
		like := "%" + strings.ToLower(term) + "%"
		tx = tx.Where("LOWER(title) LIKE ? OR LOWER(url) LIKE ?", like, like)
	}

	var total int64
	if err := tx.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count static pages: %w", err)
	}
	if total == 0 {
		return []domain.StaticPage{}, 0, nil
	}

	page, pageSize := coresearch.NormalizePagination(query.Page, query.PageSize)
	offset := (page - 1) * pageSize

	var records []staticPageRecord
	if err := tx.Order("updated_at DESC, id DESC").Limit(pageSize).Offset(offset).Find(&records).Error; err != nil {
		return nil, 0, fmt.Errorf("list static pages: %w", err)
	}

	rows := make([]domain.StaticPage, 0, len(records))
	for _, record := range records {
		rows = append(rows, toStaticPageEntity(record))
	}

	return rows, total, nil
}

// loadStaticPage resolves one static page using the provided query builder.
func (r *StaticPageRepository) loadStaticPage(ctx context.Context, builder func(*gorm.DB) *gorm.DB) (*domain.StaticPage, error) {
	var record staticPageRecord
	if err := builder(r.db.WithContext(ctx).Model(&staticPageRecord{})).First(&record).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("load static page: %w", err)
	}

	entity := toStaticPageEntity(record)
	return &entity, nil
}
