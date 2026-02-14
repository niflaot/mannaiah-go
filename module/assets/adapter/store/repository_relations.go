package store

import (
	"fmt"

	"gorm.io/gorm"
	"mannaiah/module/assets/domain"
)

// replaceAssetTags replaces all asset tags for a target asset.
func replaceAssetTags(tx *gorm.DB, assetID string, tags []domain.Tag) error {
	if err := tx.Where("asset_id = ?", assetID).Delete(&assetTagRecord{}).Error; err != nil {
		return fmt.Errorf("delete asset tags: %w", err)
	}

	records, err := toAssetTagRecords(assetID, tags)
	if err != nil {
		return err
	}
	if len(records) == 0 {
		return nil
	}
	if err := tx.Create(&records).Error; err != nil {
		return fmt.Errorf("create asset tags: %w", err)
	}

	return nil
}

// replaceAssetMetadata replaces all metadata rows for a target asset.
func replaceAssetMetadata(tx *gorm.DB, assetID string, metadata map[string]string) error {
	if err := tx.Where("asset_id = ?", assetID).Delete(&assetMetadataRecord{}).Error; err != nil {
		return fmt.Errorf("delete asset metadata: %w", err)
	}

	records, err := toAssetMetadataRecords(assetID, metadata)
	if err != nil {
		return err
	}
	if len(records) == 0 {
		return nil
	}
	if err := tx.Create(&records).Error; err != nil {
		return fmt.Errorf("create asset metadata: %w", err)
	}

	return nil
}

// replaceFolderTags replaces all folder tags for a target folder.
func replaceFolderTags(tx *gorm.DB, folderID string, tags []domain.Tag) error {
	if err := tx.Where("folder_id = ?", folderID).Delete(&folderTagRecord{}).Error; err != nil {
		return fmt.Errorf("delete folder tags: %w", err)
	}

	records, err := toFolderTagRecords(folderID, tags)
	if err != nil {
		return err
	}
	if len(records) == 0 {
		return nil
	}
	if err := tx.Create(&records).Error; err != nil {
		return fmt.Errorf("create folder tags: %w", err)
	}

	return nil
}

// loadAssetTagMap loads asset tags keyed by asset identifier.
func loadAssetTagMap(tx *gorm.DB, assetIDs []string) (map[string][]domain.Tag, error) {
	result := make(map[string][]domain.Tag, len(assetIDs))
	if len(assetIDs) == 0 {
		return result, nil
	}

	records := make([]assetTagRecord, 0)
	if err := tx.Where("asset_id IN ?", assetIDs).Order("id ASC").Find(&records).Error; err != nil {
		return nil, fmt.Errorf("load asset tags: %w", err)
	}

	grouped := make(map[string][]assetTagRecord, len(assetIDs))
	for _, record := range records {
		grouped[record.AssetID] = append(grouped[record.AssetID], record)
	}
	for _, assetID := range assetIDs {
		result[assetID] = toDomainTags(grouped[assetID])
	}

	return result, nil
}

// loadAssetMetadataMap loads asset metadata keyed by asset identifier.
func loadAssetMetadataMap(tx *gorm.DB, assetIDs []string) (map[string]map[string]string, error) {
	result := make(map[string]map[string]string, len(assetIDs))
	if len(assetIDs) == 0 {
		return result, nil
	}

	records := make([]assetMetadataRecord, 0)
	if err := tx.Where("asset_id IN ?", assetIDs).Order("id ASC").Find(&records).Error; err != nil {
		return nil, fmt.Errorf("load asset metadata: %w", err)
	}

	grouped := make(map[string][]assetMetadataRecord, len(assetIDs))
	for _, record := range records {
		grouped[record.AssetID] = append(grouped[record.AssetID], record)
	}
	for _, assetID := range assetIDs {
		result[assetID] = toDomainMetadata(grouped[assetID])
	}

	return result, nil
}

// loadFolderTagMap loads folder tags keyed by folder identifier.
func loadFolderTagMap(tx *gorm.DB, folderIDs []string) (map[string][]domain.Tag, error) {
	result := make(map[string][]domain.Tag, len(folderIDs))
	if len(folderIDs) == 0 {
		return result, nil
	}

	records := make([]folderTagRecord, 0)
	if err := tx.Where("folder_id IN ?", folderIDs).Order("id ASC").Find(&records).Error; err != nil {
		return nil, fmt.Errorf("load folder tags: %w", err)
	}

	grouped := make(map[string][]folderTagRecord, len(folderIDs))
	for _, record := range records {
		grouped[record.FolderID] = append(grouped[record.FolderID], record)
	}
	for _, folderID := range folderIDs {
		result[folderID] = toDomainTags(grouped[folderID])
	}

	return result, nil
}
