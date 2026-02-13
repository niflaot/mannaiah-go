package store

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"gorm.io/gorm"
	"mannaiah/module/assets/domain"
	"mannaiah/module/assets/port"
)

// CreateFolder persists folder metadata rows.
func (r *Repository) CreateFolder(ctx context.Context, folder *domain.Folder) error {
	record, err := toFolderRecord(*folder)
	if err != nil {
		return err
	}

	if err := r.db.WithContext(ctx).Create(&record).Error; err != nil {
		return fmt.Errorf("create folder record: %w", err)
	}

	mapped, err := toFolderDomain(record)
	if err != nil {
		return err
	}
	*folder = mapped
	return nil
}

// GetFolderByID loads folder metadata rows by id.
func (r *Repository) GetFolderByID(ctx context.Context, id string) (*domain.Folder, error) {
	var record folderRecord
	if err := r.db.WithContext(ctx).First(&record, "id = ?", strings.TrimSpace(id)).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, port.ErrNotFound
		}
		return nil, fmt.Errorf("get folder record: %w", err)
	}

	entity, err := toFolderDomain(record)
	if err != nil {
		return nil, err
	}
	return &entity, nil
}

// ListFolders paginates folder metadata rows.
func (r *Repository) ListFolders(ctx context.Context, query port.ListQuery) (*port.FolderPageResult, error) {
	page, limit := normalizePagination(query.Page, query.Limit)

	base := r.db.WithContext(ctx).Model(&folderRecord{})
	base = applyFolderListFilters(base, query.Filters)

	var total int64
	if err := base.Count(&total).Error; err != nil {
		return nil, fmt.Errorf("count folder records: %w", err)
	}

	records := make([]folderRecord, 0)
	if err := base.Order("created_at DESC").Offset((page - 1) * limit).Limit(limit).Find(&records).Error; err != nil {
		return nil, fmt.Errorf("list folder records: %w", err)
	}

	data := make([]domain.Folder, 0, len(records))
	for _, record := range records {
		mapped, mapErr := toFolderDomain(record)
		if mapErr != nil {
			return nil, mapErr
		}
		data = append(data, mapped)
	}

	return &port.FolderPageResult{Data: data, Total: total, Page: page, Limit: limit}, nil
}

// UpdateFolder updates folder metadata fields.
func (r *Repository) UpdateFolder(ctx context.Context, id string, update port.FolderUpdate) (*domain.Folder, error) {
	trimmedID := strings.TrimSpace(id)

	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var record folderRecord
		if getErr := tx.First(&record, "id = ?", trimmedID).Error; getErr != nil {
			if errors.Is(getErr, gorm.ErrRecordNotFound) {
				return port.ErrNotFound
			}
			return fmt.Errorf("get folder record for update: %w", getErr)
		}

		if update.Name != nil {
			record.Name = strings.TrimSpace(*update.Name)
			record.Slug = domain.BuildFolderSlug(record.Name)
		}
		if update.Tags != nil {
			encodedTags, encodeErr := encodeTags(*update.Tags)
			if encodeErr != nil {
				return encodeErr
			}
			record.TagsJSON = encodedTags
		}

		if saveErr := tx.Save(&record).Error; saveErr != nil {
			return fmt.Errorf("update folder record: %w", saveErr)
		}

		return nil
	})
	if err != nil {
		if errors.Is(err, port.ErrNotFound) {
			return nil, port.ErrNotFound
		}
		return nil, err
	}

	return r.GetFolderByID(ctx, trimmedID)
}

// SoftDeleteFolder soft-deletes folder rows and detaches linked assets.
func (r *Repository) SoftDeleteFolder(ctx context.Context, id string) error {
	trimmedID := strings.TrimSpace(id)

	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		deleteTx := tx.Delete(&folderRecord{}, "id = ?", trimmedID)
		if deleteTx.Error != nil {
			return fmt.Errorf("soft delete folder record: %w", deleteTx.Error)
		}
		if deleteTx.RowsAffected == 0 {
			return port.ErrNotFound
		}

		if err := tx.Model(&assetRecord{}).
			Where("folder_id = ?", trimmedID).
			Where("deleted_at IS NULL").
			Update("folder_id", nil).Error; err != nil {
			return fmt.Errorf("detach assets from folder: %w", err)
		}

		return nil
	})
}

// ExistsFolder reports whether folders exist by id.
func (r *Repository) ExistsFolder(ctx context.Context, id string) (bool, error) {
	var count int64
	if err := r.db.WithContext(ctx).Model(&folderRecord{}).Where("id = ?", strings.TrimSpace(id)).Count(&count).Error; err != nil {
		return false, fmt.Errorf("count folder record: %w", err)
	}

	return count > 0, nil
}

// folderExists reports whether folders exist by id inside transactions.
func folderExists(tx *gorm.DB, id string) (bool, error) {
	var count int64
	if err := tx.Model(&folderRecord{}).Where("id = ?", strings.TrimSpace(id)).Count(&count).Error; err != nil {
		return false, fmt.Errorf("count folder record: %w", err)
	}

	return count > 0, nil
}

// applyFolderListFilters applies list search filters across relevant columns.
func applyFolderListFilters(tx *gorm.DB, filters string) *gorm.DB {
	trimmed := strings.TrimSpace(filters)
	if trimmed == "" {
		return tx
	}

	pattern := "%" + strings.ToLower(trimmed) + "%"
	return tx.Where("LOWER(name) LIKE ? OR LOWER(slug) LIKE ?", pattern, pattern)
}
