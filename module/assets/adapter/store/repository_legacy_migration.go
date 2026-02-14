package store

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"gorm.io/gorm"
	"mannaiah/module/assets/domain"
)

type legacyAssetRow struct {
	ID           string
	TagsJSON     string
	MetadataJSON string
}

type legacyFolderRow struct {
	ID       string
	TagsJSON string
}

// migrateLegacyRelations migrates legacy JSON columns into normalized relation tables.
func (r *Repository) migrateLegacyRelations(ctx context.Context) error {
	migrator := r.db.WithContext(ctx).Migrator()
	if err := r.migrateLegacyAssetTags(ctx, migrator.HasColumn("assets", "tags_json")); err != nil {
		return err
	}
	if err := r.migrateLegacyAssetMetadata(ctx, migrator.HasColumn("assets", "metadata_json")); err != nil {
		return err
	}
	if err := r.migrateLegacyFolderTags(ctx, migrator.HasColumn("asset_folders", "tags_json")); err != nil {
		return err
	}

	return nil
}

// migrateLegacyAssetTags migrates legacy `assets.tags_json` values.
func (r *Repository) migrateLegacyAssetTags(ctx context.Context, hasColumn bool) error {
	if !hasColumn {
		return nil
	}

	rows := make([]legacyAssetRow, 0)
	if err := r.db.WithContext(ctx).Table("assets").Select("id, tags_json, metadata_json").Find(&rows).Error; err != nil {
		return fmt.Errorf("load legacy asset tag rows: %w", err)
	}
	for _, row := range rows {
		if strings.TrimSpace(row.TagsJSON) == "" {
			continue
		}

		var relationCount int64
		if err := r.db.WithContext(ctx).Model(&assetTagRecord{}).Where("asset_id = ?", row.ID).Count(&relationCount).Error; err != nil {
			return fmt.Errorf("count migrated asset tags: %w", err)
		}
		if relationCount > 0 {
			continue
		}

		tags, err := parseLegacyTags(row.TagsJSON)
		if err != nil {
			return fmt.Errorf("parse legacy asset tags for %q: %w", row.ID, err)
		}
		if err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			return replaceAssetTags(tx, row.ID, tags)
		}); err != nil {
			return fmt.Errorf("migrate legacy asset tags for %q: %w", row.ID, err)
		}
	}

	return nil
}

// migrateLegacyAssetMetadata migrates legacy `assets.metadata_json` values.
func (r *Repository) migrateLegacyAssetMetadata(ctx context.Context, hasColumn bool) error {
	if !hasColumn {
		return nil
	}

	rows := make([]legacyAssetRow, 0)
	if err := r.db.WithContext(ctx).Table("assets").Select("id, tags_json, metadata_json").Find(&rows).Error; err != nil {
		return fmt.Errorf("load legacy asset metadata rows: %w", err)
	}
	for _, row := range rows {
		if strings.TrimSpace(row.MetadataJSON) == "" {
			continue
		}

		var relationCount int64
		if err := r.db.WithContext(ctx).Model(&assetMetadataRecord{}).Where("asset_id = ?", row.ID).Count(&relationCount).Error; err != nil {
			return fmt.Errorf("count migrated asset metadata: %w", err)
		}
		if relationCount > 0 {
			continue
		}

		metadata, err := parseLegacyMetadata(row.MetadataJSON)
		if err != nil {
			return fmt.Errorf("parse legacy asset metadata for %q: %w", row.ID, err)
		}
		if err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			return replaceAssetMetadata(tx, row.ID, metadata)
		}); err != nil {
			return fmt.Errorf("migrate legacy asset metadata for %q: %w", row.ID, err)
		}
	}

	return nil
}

// migrateLegacyFolderTags migrates legacy `asset_folders.tags_json` values.
func (r *Repository) migrateLegacyFolderTags(ctx context.Context, hasColumn bool) error {
	if !hasColumn {
		return nil
	}

	rows := make([]legacyFolderRow, 0)
	if err := r.db.WithContext(ctx).Table("asset_folders").Select("id, tags_json").Find(&rows).Error; err != nil {
		return fmt.Errorf("load legacy folder tag rows: %w", err)
	}
	for _, row := range rows {
		if strings.TrimSpace(row.TagsJSON) == "" {
			continue
		}

		var relationCount int64
		if err := r.db.WithContext(ctx).Model(&folderTagRecord{}).Where("folder_id = ?", row.ID).Count(&relationCount).Error; err != nil {
			return fmt.Errorf("count migrated folder tags: %w", err)
		}
		if relationCount > 0 {
			continue
		}

		tags, err := parseLegacyTags(row.TagsJSON)
		if err != nil {
			return fmt.Errorf("parse legacy folder tags for %q: %w", row.ID, err)
		}
		if err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			return replaceFolderTags(tx, row.ID, tags)
		}); err != nil {
			return fmt.Errorf("migrate legacy folder tags for %q: %w", row.ID, err)
		}
	}

	return nil
}

// parseLegacyTags parses legacy tag JSON values.
func parseLegacyTags(raw string) ([]domain.Tag, error) {
	values := make([]domain.Tag, 0)
	if err := json.Unmarshal([]byte(raw), &values); err != nil {
		return nil, err
	}

	return normalizeAndValidateTags(values)
}

// parseLegacyMetadata parses legacy metadata JSON values.
func parseLegacyMetadata(raw string) (map[string]string, error) {
	values := map[string]string{}
	if err := json.Unmarshal([]byte(raw), &values); err != nil {
		return nil, err
	}

	normalized := normalizeMetadata(values)
	if err := validateMetadata(normalized); err != nil {
		return nil, err
	}

	return normalized, nil
}
