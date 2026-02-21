package store

import (
	"context"
	"fmt"

	"mannaiah/module/assets/domain"
)

// ListAllFolders loads all active folders for hierarchical tree construction.
func (r *Repository) ListAllFolders(ctx context.Context) ([]domain.Folder, error) {
	records := make([]folderRecord, 0)
	if err := r.db.WithContext(ctx).
		Order("parent_folder_id ASC").
		Order("name ASC").
		Order("id ASC").
		Find(&records).Error; err != nil {
		return nil, fmt.Errorf("list all folder records: %w", err)
	}

	folderIDs := make([]string, 0, len(records))
	for _, record := range records {
		folderIDs = append(folderIDs, record.ID)
	}

	tagMap, err := loadFolderTagMap(r.db.WithContext(ctx), folderIDs)
	if err != nil {
		return nil, err
	}

	result := make([]domain.Folder, 0, len(records))
	for _, record := range records {
		mapped, mapErr := toFolderDomain(record, tagMap[record.ID])
		if mapErr != nil {
			return nil, mapErr
		}
		result = append(result, mapped)
	}

	return result, nil
}
