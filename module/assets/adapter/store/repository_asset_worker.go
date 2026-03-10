package store

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"

	"gorm.io/gorm"
	"mannaiah/module/assets/domain"
	"mannaiah/module/assets/port"
)

const (
	defaultTagListLimit = 100
	maxTagListLimit     = 500
)

// UpdateBinary updates binary-related fields for an existing asset.
func (r *Repository) UpdateBinary(ctx context.Context, id string, update port.AssetBinaryUpdate) (*domain.Asset, error) {
	trimmedID := strings.TrimSpace(id)
	trimmedKey := strings.TrimSpace(update.Key)
	trimmedOriginalName := strings.TrimSpace(update.OriginalName)
	trimmedMimeType := strings.TrimSpace(update.MimeType)

	if update.Size <= 0 {
		return nil, domain.ErrInvalidSize
	}
	if trimmedKey == "" {
		return nil, domain.ErrKeyRequired
	}
	if trimmedOriginalName == "" {
		return nil, domain.ErrOriginalNameRequired
	}
	if trimmedMimeType == "" {
		return nil, domain.ErrMimeTypeRequired
	}

	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var record assetRecord
		if getErr := tx.First(&record, "id = ?", trimmedID).Error; getErr != nil {
			if errors.Is(getErr, gorm.ErrRecordNotFound) {
				return port.ErrNotFound
			}
			return fmt.Errorf("get asset record for binary update: %w", getErr)
		}

		record.Key = trimmedKey
		record.OriginalName = trimmedOriginalName
		record.MimeType = trimmedMimeType
		record.Size = update.Size

		if saveErr := tx.Save(&record).Error; saveErr != nil {
			return fmt.Errorf("update asset binary record: %w", saveErr)
		}

		return nil
	})
	if err != nil {
		if errors.Is(err, port.ErrNotFound) {
			return nil, port.ErrNotFound
		}
		return nil, err
	}

	return r.GetByID(ctx, trimmedID)
}

// ListByTagNames loads assets that contain one or more provided tag names.
func (r *Repository) ListByTagNames(ctx context.Context, tagNames []string, limit int) ([]domain.Asset, error) {
	normalizedTagNames := normalizeTagNames(tagNames)
	if len(normalizedTagNames) == 0 {
		return []domain.Asset{}, nil
	}

	resolvedLimit := resolveTagListLimit(limit)
	assetIDs := make([]string, 0, resolvedLimit)
	if err := r.db.WithContext(ctx).
		Model(&assetTagRecord{}).
		Distinct("asset_tags.asset_id").
		Select("asset_tags.asset_id").
		Joins("JOIN assets ON assets.id = asset_tags.asset_id").
		Where("asset_tags.name IN ?", normalizedTagNames).
		Where("assets.deleted_at IS NULL").
		Order("assets.updated_at ASC, assets.id ASC").
		Limit(resolvedLimit).
		Pluck("asset_tags.asset_id", &assetIDs).Error; err != nil {
		return nil, fmt.Errorf("list tagged asset ids: %w", err)
	}
	if len(assetIDs) == 0 {
		return []domain.Asset{}, nil
	}

	uniqueIDs := uniqueAssetIDs(assetIDs)
	records := make([]assetRecord, 0, len(uniqueIDs))
	if err := r.db.WithContext(ctx).Where("id IN ?", uniqueIDs).Find(&records).Error; err != nil {
		return nil, fmt.Errorf("list tagged asset records: %w", err)
	}

	tagMap, err := loadAssetTagMap(r.db.WithContext(ctx), uniqueIDs)
	if err != nil {
		return nil, err
	}
	metadataMap, err := loadAssetMetadataMap(r.db.WithContext(ctx), uniqueIDs)
	if err != nil {
		return nil, err
	}

	recordByID := make(map[string]assetRecord, len(records))
	for _, record := range records {
		recordByID[record.ID] = record
	}

	result := make([]domain.Asset, 0, len(uniqueIDs))
	for _, assetID := range uniqueIDs {
		record, exists := recordByID[assetID]
		if !exists {
			continue
		}

		entity, mapErr := toAssetDomain(record, tagMap[assetID], metadataMap[assetID])
		if mapErr != nil {
			return nil, mapErr
		}
		result = append(result, entity)
	}

	return result, nil
}

// resolveTagListLimit normalizes tag-list result limits.
func resolveTagListLimit(limit int) int {
	resolvedLimit := limit
	if resolvedLimit <= 0 {
		resolvedLimit = defaultTagListLimit
	}
	if resolvedLimit > maxTagListLimit {
		resolvedLimit = maxTagListLimit
	}

	return resolvedLimit
}

// normalizeTagNames normalizes tag names for case-insensitive storage queries.
func normalizeTagNames(tagNames []string) []string {
	seen := map[string]struct{}{}
	normalized := make([]string, 0, len(tagNames))

	for _, raw := range tagNames {
		trimmed := strings.ToLower(strings.TrimSpace(raw))
		if trimmed == "" {
			continue
		}
		if _, exists := seen[trimmed]; exists {
			continue
		}
		seen[trimmed] = struct{}{}
		normalized = append(normalized, trimmed)
	}

	sort.Strings(normalized)
	return normalized
}

// uniqueAssetIDs preserves order while removing duplicate ids.
func uniqueAssetIDs(ids []string) []string {
	result := make([]string, 0, len(ids))
	seen := make(map[string]struct{}, len(ids))

	for _, id := range ids {
		trimmed := strings.TrimSpace(id)
		if trimmed == "" {
			continue
		}
		if _, exists := seen[trimmed]; exists {
			continue
		}
		seen[trimmed] = struct{}{}
		result = append(result, trimmed)
	}

	return result
}
