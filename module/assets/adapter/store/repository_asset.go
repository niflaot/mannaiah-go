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

// Create persists asset metadata rows.
func (r *Repository) Create(ctx context.Context, asset *domain.Asset) error {
	record, err := toAssetRecord(*asset)
	if err != nil {
		return err
	}

	if err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&record).Error; err != nil {
			return fmt.Errorf("create asset record: %w", err)
		}
		if err := replaceAssetTags(tx, record.ID, asset.Tags); err != nil {
			return err
		}
		if err := replaceAssetMetadata(tx, record.ID, asset.Metadata); err != nil {
			return err
		}

		return nil
	}); err != nil {
		return err
	}

	loaded, err := r.GetByID(ctx, record.ID)
	if err != nil {
		return err
	}
	*asset = *loaded

	return nil
}

// GetByID retrieves asset rows by id.
func (r *Repository) GetByID(ctx context.Context, id string) (*domain.Asset, error) {
	trimmedID := strings.TrimSpace(id)

	var record assetRecord
	if err := r.db.WithContext(ctx).First(&record, "id = ?", trimmedID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, port.ErrNotFound
		}

		return nil, fmt.Errorf("get asset record: %w", err)
	}

	tagMap, err := loadAssetTagMap(r.db.WithContext(ctx), []string{trimmedID})
	if err != nil {
		return nil, err
	}
	metadataMap, err := loadAssetMetadataMap(r.db.WithContext(ctx), []string{trimmedID})
	if err != nil {
		return nil, err
	}

	entity, err := toAssetDomain(record, tagMap[trimmedID], metadataMap[trimmedID])
	if err != nil {
		return nil, err
	}

	return &entity, nil
}

// List paginates asset metadata rows.
func (r *Repository) List(ctx context.Context, query port.ListQuery) (*port.PageResult, error) {
	page, limit := normalizePagination(query.Page, query.Limit)

	base := r.db.WithContext(ctx).Model(&assetRecord{})
	base = applyAssetListFilters(base, query.Filters)

	var total int64
	if err := base.Count(&total).Error; err != nil {
		return nil, fmt.Errorf("count asset records: %w", err)
	}

	records := make([]assetRecord, 0)
	if err := base.Order("created_at DESC").Offset((page - 1) * limit).Limit(limit).Find(&records).Error; err != nil {
		return nil, fmt.Errorf("list asset records: %w", err)
	}

	assetIDs := make([]string, 0, len(records))
	for _, record := range records {
		assetIDs = append(assetIDs, record.ID)
	}
	tagMap, err := loadAssetTagMap(r.db.WithContext(ctx), assetIDs)
	if err != nil {
		return nil, err
	}
	metadataMap, err := loadAssetMetadataMap(r.db.WithContext(ctx), assetIDs)
	if err != nil {
		return nil, err
	}

	result := make([]domain.Asset, 0, len(records))
	for _, record := range records {
		mapped, mapErr := toAssetDomain(record, tagMap[record.ID], metadataMap[record.ID])
		if mapErr != nil {
			return nil, mapErr
		}
		result = append(result, mapped)
	}

	return &port.PageResult{
		Data:  result,
		Total: total,
		Page:  page,
		Limit: limit,
	}, nil
}

// Update updates asset metadata fields.
func (r *Repository) Update(ctx context.Context, id string, update port.AssetUpdate) (*domain.Asset, error) {
	trimmedID := strings.TrimSpace(id)

	txErr := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var record assetRecord
		if err := tx.First(&record, "id = ?", trimmedID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return port.ErrNotFound
			}
			return fmt.Errorf("get asset record for update: %w", err)
		}

		if update.Name != nil {
			record.Name = strings.TrimSpace(*update.Name)
		}
		if update.FolderID != nil {
			folderID := strings.TrimSpace(*update.FolderID)
			if folderID == "" {
				record.FolderID = nil
			} else {
				exists, err := folderExists(tx, folderID)
				if err != nil {
					return err
				}
				if !exists {
					return port.ErrFolderNotFound
				}
				record.FolderID = &folderID
			}
		}
		if update.Tags != nil {
			if err := replaceAssetTags(tx, trimmedID, *update.Tags); err != nil {
				return err
			}
		}
		if update.Metadata != nil {
			if err := replaceAssetMetadata(tx, trimmedID, *update.Metadata); err != nil {
				return err
			}
		}

		if err := tx.Save(&record).Error; err != nil {
			return fmt.Errorf("update asset record: %w", err)
		}

		return nil
	})
	if txErr != nil {
		if errors.Is(txErr, port.ErrNotFound) {
			return nil, port.ErrNotFound
		}
		if errors.Is(txErr, port.ErrFolderNotFound) {
			return nil, port.ErrFolderNotFound
		}
		return nil, txErr
	}

	return r.GetByID(ctx, trimmedID)
}

// SoftDelete soft-deletes asset metadata rows.
func (r *Repository) SoftDelete(ctx context.Context, id string) error {
	trimmedID := strings.TrimSpace(id)

	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		deleteTx := tx.Delete(&assetRecord{}, "id = ?", trimmedID)
		if deleteTx.Error != nil {
			return fmt.Errorf("soft delete asset record: %w", deleteTx.Error)
		}
		if deleteTx.RowsAffected == 0 {
			return port.ErrNotFound
		}
		if err := tx.Where("asset_id = ?", trimmedID).Delete(&assetTagRecord{}).Error; err != nil {
			return fmt.Errorf("delete asset tag records: %w", err)
		}
		if err := tx.Where("asset_id = ?", trimmedID).Delete(&assetMetadataRecord{}).Error; err != nil {
			return fmt.Errorf("delete asset metadata records: %w", err)
		}

		return nil
	})
}

// applyAssetListFilters applies list search filters across relevant columns.
func applyAssetListFilters(tx *gorm.DB, filters string) *gorm.DB {
	trimmed := strings.TrimSpace(filters)
	if trimmed == "" {
		return tx
	}

	pattern := "%" + strings.ToLower(trimmed) + "%"
	return tx.Where(
		"LOWER(name) LIKE ? OR LOWER(original_name) LIKE ? OR LOWER(mime_type) LIKE ? OR LOWER(`key`) LIKE ?",
		pattern,
		pattern,
		pattern,
		pattern,
	)
}

// normalizePagination resolves list pagination defaults.
func normalizePagination(page int, limit int) (int, int) {
	resolvedPage := page
	if resolvedPage <= 0 {
		resolvedPage = 1
	}

	resolvedLimit := limit
	if resolvedLimit <= 0 {
		resolvedLimit = 10
	}
	if resolvedLimit > 100 {
		resolvedLimit = 100
	}

	return resolvedPage, resolvedLimit
}
