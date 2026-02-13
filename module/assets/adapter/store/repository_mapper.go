package store

import (
	"encoding/json"
	"fmt"
	"strings"

	"mannaiah/module/assets/domain"
)

// toAssetRecord maps domain assets into persistence records.
func toAssetRecord(asset domain.Asset) (assetRecord, error) {
	asset.Normalize()
	if err := asset.ValidateCreate(); err != nil {
		return assetRecord{}, err
	}

	tags, err := encodeTags(asset.Tags)
	if err != nil {
		return assetRecord{}, err
	}
	metadata, err := encodeMetadata(asset.Metadata)
	if err != nil {
		return assetRecord{}, err
	}

	record := assetRecord{
		ID:           asset.ID,
		Key:          asset.Key,
		Name:         asset.Name,
		OriginalName: asset.OriginalName,
		MimeType:     asset.MimeType,
		Size:         asset.Size,
		TagsJSON:     tags,
		MetadataJSON: metadata,
	}
	if asset.FolderID != "" {
		folderID := asset.FolderID
		record.FolderID = &folderID
	}

	return record, nil
}

// toAssetDomain maps persistence records into domain assets.
func toAssetDomain(record assetRecord) (domain.Asset, error) {
	tags, err := decodeTags(record.TagsJSON)
	if err != nil {
		return domain.Asset{}, err
	}
	metadata, err := decodeMetadata(record.MetadataJSON)
	if err != nil {
		return domain.Asset{}, err
	}

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

	return entity, nil
}

// toFolderRecord maps domain folders into persistence records.
func toFolderRecord(folder domain.Folder) (folderRecord, error) {
	folder.Normalize()
	if err := folder.ValidateCreate(); err != nil {
		return folderRecord{}, err
	}

	tags, err := encodeTags(folder.Tags)
	if err != nil {
		return folderRecord{}, err
	}

	return folderRecord{
		ID:       folder.ID,
		Name:     folder.Name,
		Slug:     folder.Slug,
		TagsJSON: tags,
	}, nil
}

// toFolderDomain maps persistence records into domain folders.
func toFolderDomain(record folderRecord) (domain.Folder, error) {
	tags, err := decodeTags(record.TagsJSON)
	if err != nil {
		return domain.Folder{}, err
	}

	entity := domain.Folder{
		ID:        record.ID,
		Name:      record.Name,
		Slug:      record.Slug,
		Tags:      tags,
		CreatedAt: record.CreatedAt,
		UpdatedAt: record.UpdatedAt,
		IsDeleted: record.DeletedAt.Valid,
	}
	if record.DeletedAt.Valid {
		deletedAt := record.DeletedAt.Time
		entity.DeletedAt = &deletedAt
	}
	entity.Normalize()

	return entity, nil
}

// encodeTags serializes domain tags for persistence.
func encodeTags(tags []domain.Tag) (string, error) {
	if tags == nil {
		tags = []domain.Tag{}
	}
	if err := validateAndNormalizeTags(tags); err != nil {
		return "", err
	}

	encoded, err := json.Marshal(tags)
	if err != nil {
		return "", fmt.Errorf("marshal tags: %w", err)
	}

	return string(encoded), nil
}

// decodeTags deserializes persisted tag values.
func decodeTags(raw string) ([]domain.Tag, error) {
	if raw == "" {
		return []domain.Tag{}, nil
	}

	decoded := make([]domain.Tag, 0)
	if err := json.Unmarshal([]byte(raw), &decoded); err != nil {
		return nil, fmt.Errorf("unmarshal tags: %w", err)
	}
	if err := validateAndNormalizeTags(decoded); err != nil {
		return nil, err
	}

	return decoded, nil
}

// encodeMetadata serializes asset metadata for persistence.
func encodeMetadata(metadata map[string]string) (string, error) {
	normalized := normalizeMetadata(metadata)
	if err := validateMetadata(normalized); err != nil {
		return "", err
	}

	encoded, err := json.Marshal(normalized)
	if err != nil {
		return "", fmt.Errorf("marshal metadata: %w", err)
	}

	return string(encoded), nil
}

// decodeMetadata deserializes persisted metadata values.
func decodeMetadata(raw string) (map[string]string, error) {
	if raw == "" {
		return map[string]string{}, nil
	}

	decoded := map[string]string{}
	if err := json.Unmarshal([]byte(raw), &decoded); err != nil {
		return nil, fmt.Errorf("unmarshal metadata: %w", err)
	}

	normalized := normalizeMetadata(decoded)
	if err := validateMetadata(normalized); err != nil {
		return nil, err
	}

	return normalized, nil
}

// validateAndNormalizeTags canonicalizes and validates tag arrays.
func validateAndNormalizeTags(tags []domain.Tag) error {
	for index := range tags {
		tags[index].Name = trim(tags[index].Name)
		tags[index].Color = trim(tags[index].Color)
	}

	return domain.ValidateTagsForStore(tags)
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
