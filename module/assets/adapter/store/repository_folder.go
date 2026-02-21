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

	if record.ParentFolderID != nil {
		exists, existsErr := r.ExistsFolder(ctx, *record.ParentFolderID)
		if existsErr != nil {
			return existsErr
		}
		if !exists {
			return port.ErrFolderNotFound
		}
	}

	if err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		exists, existsErr := folderSlugExistsInParent(tx, record.ParentFolderID, record.Slug, "")
		if existsErr != nil {
			return existsErr
		}
		if exists {
			return port.ErrFolderAlreadyExists
		}

		if err := tx.Create(&record).Error; err != nil {
			if isUniqueConstraintError(err) {
				return port.ErrFolderAlreadyExists
			}
			return fmt.Errorf("create folder record: %w", err)
		}
		if err := replaceFolderTags(tx, record.ID, folder.Tags); err != nil {
			return err
		}

		return nil
	}); err != nil {
		return err
	}

	loaded, err := r.GetFolderByID(ctx, record.ID)
	if err != nil {
		return err
	}
	*folder = *loaded

	return nil
}

// GetFolderByID loads folder metadata rows by id.
func (r *Repository) GetFolderByID(ctx context.Context, id string) (*domain.Folder, error) {
	trimmedID := strings.TrimSpace(id)

	var record folderRecord
	if err := r.db.WithContext(ctx).First(&record, "id = ?", trimmedID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, port.ErrNotFound
		}
		return nil, fmt.Errorf("get folder record: %w", err)
	}

	tagMap, err := loadFolderTagMap(r.db.WithContext(ctx), []string{trimmedID})
	if err != nil {
		return nil, err
	}
	entity, err := toFolderDomain(record, tagMap[trimmedID])
	if err != nil {
		return nil, err
	}

	return &entity, nil
}

// ListFolders paginates folder metadata rows.
func (r *Repository) ListFolders(ctx context.Context, query port.ListQuery) (*port.FolderPageResult, error) {
	page, limit := normalizePagination(query.Page, query.Limit)

	base := r.db.WithContext(ctx).Model(&folderRecord{})
	base = applyFolderParentFilter(base, query.ParentFolderID)
	base = applyFolderListFilters(base, query.Filters)

	var total int64
	if err := base.Count(&total).Error; err != nil {
		return nil, fmt.Errorf("count folder records: %w", err)
	}

	records := make([]folderRecord, 0)
	if err := base.Order("created_at DESC").Offset((page - 1) * limit).Limit(limit).Find(&records).Error; err != nil {
		return nil, fmt.Errorf("list folder records: %w", err)
	}

	folderIDs := make([]string, 0, len(records))
	for _, record := range records {
		folderIDs = append(folderIDs, record.ID)
	}
	tagMap, err := loadFolderTagMap(r.db.WithContext(ctx), folderIDs)
	if err != nil {
		return nil, err
	}

	data := make([]domain.Folder, 0, len(records))
	for _, record := range records {
		mapped, mapErr := toFolderDomain(record, tagMap[record.ID])
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
		if update.ParentFolderID != nil {
			parentID := strings.TrimSpace(*update.ParentFolderID)
			if parentID == "" {
				record.ParentFolderID = nil
			} else {
				if parentID == trimmedID {
					return domain.ErrFolderParentSelfReference
				}
				parent, parentErr := getFolderRecordByID(tx, parentID)
				if parentErr != nil {
					if errors.Is(parentErr, gorm.ErrRecordNotFound) {
						return port.ErrFolderNotFound
					}
					return parentErr
				}
				if cycleErr := assertNoParentCycle(tx, trimmedID, parent.ID); cycleErr != nil {
					return cycleErr
				}
				record.ParentFolderID = &parent.ID
			}
		}
		if update.Tags != nil {
			if err := replaceFolderTags(tx, trimmedID, *update.Tags); err != nil {
				return err
			}
		}

		exists, existsErr := folderSlugExistsInParent(tx, record.ParentFolderID, record.Slug, trimmedID)
		if existsErr != nil {
			return existsErr
		}
		if exists {
			return port.ErrFolderAlreadyExists
		}

		if saveErr := tx.Save(&record).Error; saveErr != nil {
			if isUniqueConstraintError(saveErr) {
				return port.ErrFolderAlreadyExists
			}
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
		folderIDs, collectErr := collectFolderTreeIDs(tx, trimmedID)
		if collectErr != nil {
			return collectErr
		}
		if len(folderIDs) == 0 {
			return port.ErrNotFound
		}

		deleteTx := tx.Delete(&folderRecord{}, "id IN ?", folderIDs)
		if deleteTx.Error != nil {
			return fmt.Errorf("soft delete folder record: %w", deleteTx.Error)
		}
		if deleteTx.RowsAffected == 0 {
			return port.ErrNotFound
		}

		if err := tx.Model(&assetRecord{}).
			Where("folder_id IN ?", folderIDs).
			Where("deleted_at IS NULL").
			Update("folder_id", nil).Error; err != nil {
			return fmt.Errorf("detach assets from folder: %w", err)
		}
		if err := tx.Where("folder_id IN ?", folderIDs).Delete(&folderTagRecord{}).Error; err != nil {
			return fmt.Errorf("delete folder tags: %w", err)
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

// applyFolderParentFilter applies optional parent-folder filtering for nested trees.
func applyFolderParentFilter(tx *gorm.DB, parentFolderID string) *gorm.DB {
	trimmedParentID := strings.TrimSpace(parentFolderID)
	if trimmedParentID == "" {
		return tx
	}

	return tx.Where("parent_folder_id = ?", trimmedParentID)
}

// getFolderRecordByID loads folder records by id for hierarchy validation.
func getFolderRecordByID(tx *gorm.DB, id string) (*folderRecord, error) {
	var record folderRecord
	if err := tx.First(&record, "id = ?", strings.TrimSpace(id)).Error; err != nil {
		return nil, err
	}

	return &record, nil
}

// folderSlugExistsInParent reports whether an active folder with slug already exists under the same parent.
func folderSlugExistsInParent(tx *gorm.DB, parentFolderID *string, slug string, excludeID string) (bool, error) {
	query := tx.Model(&folderRecord{}).
		Where("slug = ?", strings.TrimSpace(slug)).
		Where("deleted_at IS NULL")

	if parentFolderID == nil || strings.TrimSpace(*parentFolderID) == "" {
		query = query.Where("parent_folder_id IS NULL")
	} else {
		query = query.Where("parent_folder_id = ?", strings.TrimSpace(*parentFolderID))
	}

	trimmedExcludeID := strings.TrimSpace(excludeID)
	if trimmedExcludeID != "" {
		query = query.Where("id <> ?", trimmedExcludeID)
	}

	var count int64
	if err := query.Count(&count).Error; err != nil {
		return false, fmt.Errorf("count folder slug in parent: %w", err)
	}

	return count > 0, nil
}

// assertNoParentCycle ensures assigning parentID to folderID does not create hierarchy cycles.
func assertNoParentCycle(tx *gorm.DB, folderID string, parentID string) error {
	visited := map[string]struct{}{}
	currentID := strings.TrimSpace(parentID)
	trimmedFolderID := strings.TrimSpace(folderID)

	for currentID != "" {
		if currentID == trimmedFolderID {
			return domain.ErrFolderParentCycle
		}
		if _, exists := visited[currentID]; exists {
			return domain.ErrFolderParentCycle
		}
		visited[currentID] = struct{}{}

		record, err := getFolderRecordByID(tx, currentID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return port.ErrFolderNotFound
			}
			return fmt.Errorf("load parent folder record: %w", err)
		}
		if record.ParentFolderID == nil {
			return nil
		}

		currentID = strings.TrimSpace(*record.ParentFolderID)
	}

	return nil
}

// collectFolderTreeIDs returns root and descendant ids for recursive soft-delete operations.
func collectFolderTreeIDs(tx *gorm.DB, rootID string) ([]string, error) {
	trimmedRootID := strings.TrimSpace(rootID)
	root, err := getFolderRecordByID(tx, trimmedRootID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("load folder record for delete: %w", err)
	}

	ids := []string{root.ID}
	seen := map[string]struct{}{root.ID: {}}
	frontier := []string{root.ID}

	for len(frontier) > 0 {
		children := make([]string, 0)
		if err := tx.Model(&folderRecord{}).
			Where("parent_folder_id IN ?", frontier).
			Where("deleted_at IS NULL").
			Pluck("id", &children).Error; err != nil {
			return nil, fmt.Errorf("load child folder records: %w", err)
		}

		next := make([]string, 0, len(children))
		for _, childID := range children {
			trimmedChildID := strings.TrimSpace(childID)
			if trimmedChildID == "" {
				continue
			}
			if _, exists := seen[trimmedChildID]; exists {
				continue
			}

			seen[trimmedChildID] = struct{}{}
			ids = append(ids, trimmedChildID)
			next = append(next, trimmedChildID)
		}

		frontier = next
	}

	return ids, nil
}
