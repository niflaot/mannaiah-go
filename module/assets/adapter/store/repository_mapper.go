package store

import (
	"sort"
	"strings"

	"mannaiah/module/assets/domain"
)

// toAssetRecord maps domain assets into persistence records.
func toAssetRecord(asset domain.Asset) (assetRecord, error) {
	asset.Normalize()
	if err := asset.ValidateCreate(); err != nil {
		return assetRecord{}, err
	}

	record := assetRecord{
		ID:           asset.ID,
		Key:          asset.Key,
		Name:         asset.Name,
		OriginalName: asset.OriginalName,
		MimeType:     asset.MimeType,
		Size:         asset.Size,
	}
	if asset.FolderID != "" {
		folderID := asset.FolderID
		record.FolderID = &folderID
	}

	return record, nil
}

// toAssetDomain maps persistence records and relations into domain assets.
func toAssetDomain(record assetRecord, tags []domain.Tag, metadata map[string]string) (domain.Asset, error) {
	entity := domain.Asset{
		ID:           record.ID,
		Key:          record.Key,
		Name:         record.Name,
		OriginalName: record.OriginalName,
		MimeType:     record.MimeType,
		Size:         record.Size,
		Tags:         tags,
		Metadata:     metadata,
		CreatedAt:    record.CreatedAt,
		UpdatedAt:    record.UpdatedAt,
		IsDeleted:    record.DeletedAt.Valid,
	}
	if record.FolderID != nil {
		entity.FolderID = *record.FolderID
	}
	if record.DeletedAt.Valid {
		deletedAt := record.DeletedAt.Time
		entity.DeletedAt = &deletedAt
	}
	entity.Normalize()
	if err := entity.ValidateCreate(); err != nil {
		return domain.Asset{}, err
	}

	return entity, nil
}

// toFolderRecord maps domain folders into persistence records.
func toFolderRecord(folder domain.Folder) (folderRecord, error) {
	folder.Normalize()
	if err := folder.ValidateCreate(); err != nil {
		return folderRecord{}, err
	}

	record := folderRecord{
		ID:   folder.ID,
		Name: folder.Name,
		Slug: folder.Slug,
	}
	if folder.ParentFolderID != "" {
		parentID := folder.ParentFolderID
		record.ParentFolderID = &parentID
	}

	return record, nil
}

// toFolderDomain maps persistence records and relations into domain folders.
func toFolderDomain(record folderRecord, tags []domain.Tag) (domain.Folder, error) {
	entity := domain.Folder{
		ID:             record.ID,
		Name:           record.Name,
		Slug:           record.Slug,
		Tags:           tags,
		CreatedAt:      record.CreatedAt,
		UpdatedAt:      record.UpdatedAt,
		IsDeleted:      record.DeletedAt.Valid,
		ParentFolderID: "",
	}
	if record.ParentFolderID != nil {
		entity.ParentFolderID = *record.ParentFolderID
	}
	if record.DeletedAt.Valid {
		deletedAt := record.DeletedAt.Time
		entity.DeletedAt = &deletedAt
	}
	entity.Normalize()
	if err := entity.ValidateCreate(); err != nil {
		return domain.Folder{}, err
	}

	return entity, nil
}

// toAssetTagRecords maps domain tags into normalized asset-tag rows.
func toAssetTagRecords(assetID string, tags []domain.Tag) ([]assetTagRecord, error) {
	normalized, err := normalizeAndValidateTags(tags)
	if err != nil {
		return nil, err
	}

	records := make([]assetTagRecord, 0, len(normalized))
	for _, value := range normalized {
		records = append(records, assetTagRecord{
			AssetID: strings.TrimSpace(assetID),
			Name:    value.Name,
			Color:   value.Color,
		})
	}

	return records, nil
}

// toFolderTagRecords maps domain tags into normalized folder-tag rows.
func toFolderTagRecords(folderID string, tags []domain.Tag) ([]folderTagRecord, error) {
	normalized, err := normalizeAndValidateTags(tags)
	if err != nil {
		return nil, err
	}

	records := make([]folderTagRecord, 0, len(normalized))
	for _, value := range normalized {
		records = append(records, folderTagRecord{
			FolderID: strings.TrimSpace(folderID),
			Name:     value.Name,
			Color:    value.Color,
		})
	}

	return records, nil
}

// toAssetMetadataRecords maps metadata maps into normalized rows.
func toAssetMetadataRecords(assetID string, metadata map[string]string) ([]assetMetadataRecord, error) {
	normalized := normalizeMetadata(metadata)
	if err := validateMetadata(normalized); err != nil {
		return nil, err
	}

	keys := make([]string, 0, len(normalized))
	for key := range normalized {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	records := make([]assetMetadataRecord, 0, len(keys))
	for _, key := range keys {
		records = append(records, assetMetadataRecord{
			AssetID: strings.TrimSpace(assetID),
			Key:     key,
			Value:   normalized[key],
		})
	}

	return records, nil
}

// toDomainTags maps normalized rows into domain tags.
func toDomainTags[T interface {
	getName() string
	getColor() string
}](values []T) []domain.Tag {
	tags := make([]domain.Tag, 0, len(values))
	for _, value := range values {
		tags = append(tags, domain.Tag{Name: value.getName(), Color: value.getColor()})
	}

	return tags
}

// normalizeAndValidateTags canonicalizes and validates tag arrays.
func normalizeAndValidateTags(tags []domain.Tag) ([]domain.Tag, error) {
	normalized := make([]domain.Tag, len(tags))
	copy(normalized, tags)
	for index := range normalized {
		normalized[index].Name = trim(normalized[index].Name)
		normalized[index].Color = trim(normalized[index].Color)
	}
	if err := domain.ValidateTagsForStore(normalized); err != nil {
		return nil, err
	}

	return normalized, nil
}

// normalizeMetadata canonicalizes metadata keys and values.
func normalizeMetadata(metadata map[string]string) map[string]string {
	if metadata == nil {
		return map[string]string{}
	}

	normalized := make(map[string]string, len(metadata))
	for key, value := range metadata {
		trimmedKey := trim(key)
		if trimmedKey == "" {
			continue
		}
		normalized[trimmedKey] = trim(value)
	}

	return normalized
}

// validateMetadata verifies normalized metadata constraints.
func validateMetadata(metadata map[string]string) error {
	for key, value := range metadata {
		if len(key) > 128 || len(value) > 2048 {
			return domain.ErrInvalidMetadata
		}
	}

	return nil
}

// trim normalizes string values by trimming whitespace.
func trim(value string) string {
	return strings.TrimSpace(value)
}

// getName returns tag names from asset tag rows.
func (r assetTagRecord) getName() string {
	return r.Name
}

// getColor returns tag colors from asset tag rows.
func (r assetTagRecord) getColor() string {
	return r.Color
}

// getName returns tag names from folder tag rows.
func (r folderTagRecord) getName() string {
	return r.Name
}

// getColor returns tag colors from folder tag rows.
func (r folderTagRecord) getColor() string {
	return r.Color
}

// toDomainMetadata maps normalized metadata rows into metadata maps.
func toDomainMetadata(records []assetMetadataRecord) map[string]string {
	metadata := make(map[string]string, len(records))
	for _, record := range records {
		metadata[record.Key] = record.Value
	}

	return metadata
}
