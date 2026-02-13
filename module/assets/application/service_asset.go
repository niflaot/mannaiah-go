package application

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"mannaiah/module/assets/domain"
	"mannaiah/module/assets/port"
)

// Create uploads binaries and persists metadata.
func (s *AssetService) Create(ctx context.Context, command CreateCommand) (*domain.Asset, error) {
	if err := s.ensureStorage(); err != nil {
		return nil, err
	}
	if len(command.Body) == 0 {
		return nil, ErrFileRequired
	}
	if command.Size <= 0 {
		command.Size = int64(len(command.Body))
	}
	if command.Size > maxUploadBytes {
		return nil, ErrFileTooLarge
	}

	folderID := strings.TrimSpace(command.FolderID)
	if folderID != "" {
		exists, err := s.repository.ExistsFolder(ctx, folderID)
		if err != nil {
			return nil, fmt.Errorf("check asset folder exists: %w", err)
		}
		if !exists {
			return nil, port.ErrFolderNotFound
		}
	}

	assetID := uuid.NewString()
	key := buildStorageKey(assetID, command.OriginalName)
	name := strings.TrimSpace(command.Name)
	if name == "" {
		name = strings.TrimSpace(command.OriginalName)
	}

	entity := &domain.Asset{
		ID:           assetID,
		Key:          key,
		Name:         name,
		OriginalName: strings.TrimSpace(command.OriginalName),
		FolderID:     folderID,
		MimeType:     strings.TrimSpace(command.MimeType),
		Size:         command.Size,
		Tags:         command.Tags,
		Metadata:     command.Metadata,
	}
	entity.Normalize()
	if err := entity.ValidateCreate(); err != nil {
		return nil, err
	}

	if err := s.storage.Upload(ctx, port.UploadRequest{Key: entity.Key, ContentType: entity.MimeType, Body: command.Body}); err != nil {
		return nil, fmt.Errorf("upload asset object: %w", err)
	}

	if err := s.repository.Create(ctx, entity); err != nil {
		if rollbackErr := s.storage.Delete(ctx, entity.Key); rollbackErr != nil {
			return nil, fmt.Errorf("create asset metadata: %w (rollback error: %v)", err, rollbackErr)
		}
		return nil, fmt.Errorf("create asset metadata: %w", err)
	}

	if publishErr := s.publisher.Publish(ctx, buildAssetCreatedIntegrationEvent(*entity)); publishErr != nil {
		return nil, fmt.Errorf("publish asset created event: %w", publishErr)
	}

	return entity, nil
}

// Get retrieves assets by id.
func (s *AssetService) Get(ctx context.Context, id string) (*domain.Asset, error) {
	trimmedID := strings.TrimSpace(id)
	if trimmedID == "" {
		return nil, ErrInvalidID
	}

	entity, err := s.repository.GetByID(ctx, trimmedID)
	if err != nil {
		return nil, fmt.Errorf("get asset: %w", err)
	}

	return entity, nil
}

// List paginates assets.
func (s *AssetService) List(ctx context.Context, query ListQuery) (*port.PageResult, error) {
	result, err := s.repository.List(ctx, port.ListQuery{
		Page:           query.Page,
		Limit:          query.Limit,
		Filters:        strings.TrimSpace(query.Filters),
		ParentFolderID: strings.TrimSpace(query.ParentFolderID),
	})
	if err != nil {
		return nil, fmt.Errorf("list assets: %w", err)
	}

	return result, nil
}

// Update updates mutable asset fields.
func (s *AssetService) Update(ctx context.Context, id string, command UpdateCommand) (*domain.Asset, error) {
	trimmedID := strings.TrimSpace(id)
	if trimmedID == "" {
		return nil, ErrInvalidID
	}
	if command.Name != nil && strings.TrimSpace(*command.Name) == "" {
		return nil, ErrInvalidName
	}
	if command.FolderID != nil {
		folderID := strings.TrimSpace(*command.FolderID)
		if folderID != "" {
			exists, err := s.repository.ExistsFolder(ctx, folderID)
			if err != nil {
				return nil, fmt.Errorf("check asset folder exists: %w", err)
			}
			if !exists {
				return nil, port.ErrFolderNotFound
			}
		}
	}

	unlock := s.locks.Lock("asset:" + trimmedID)
	defer unlock()

	entity, err := s.repository.Update(ctx, trimmedID, port.AssetUpdate{
		Name:     command.Name,
		FolderID: command.FolderID,
		Tags:     command.Tags,
		Metadata: command.Metadata,
	})
	if err != nil {
		return nil, fmt.Errorf("update asset: %w", err)
	}

	if publishErr := s.publisher.Publish(ctx, buildAssetUpdatedIntegrationEvent(*entity)); publishErr != nil {
		return nil, fmt.Errorf("publish asset updated event: %w", publishErr)
	}

	return entity, nil
}

// UpdateName updates asset custom names.
func (s *AssetService) UpdateName(ctx context.Context, id string, name string) (*domain.Asset, error) {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return nil, ErrInvalidName
	}

	return s.Update(ctx, id, UpdateCommand{Name: &trimmed})
}

// Delete soft-deletes metadata.
func (s *AssetService) Delete(ctx context.Context, id string) error {
	trimmedID := strings.TrimSpace(id)
	if trimmedID == "" {
		return ErrInvalidID
	}

	unlock := s.locks.Lock("asset:" + trimmedID)
	defer unlock()

	entity, err := s.repository.GetByID(ctx, trimmedID)
	if err != nil {
		return fmt.Errorf("load asset for delete: %w", err)
	}
	if err := s.repository.SoftDelete(ctx, trimmedID); err != nil {
		return fmt.Errorf("soft delete asset metadata: %w", err)
	}

	if publishErr := s.publisher.Publish(ctx, buildAssetDeletedIntegrationEvent(*entity)); publishErr != nil {
		return fmt.Errorf("publish asset deleted event: %w", publishErr)
	}

	return nil
}

// Exists verifies whether metadata rows exist by id.
func (s *AssetService) Exists(ctx context.Context, id string) (bool, error) {
	trimmedID := strings.TrimSpace(id)
	if trimmedID == "" {
		return false, ErrInvalidID
	}

	_, err := s.repository.GetByID(ctx, trimmedID)
	if err != nil {
		if errors.Is(err, port.ErrNotFound) {
			return false, nil
		}
		return false, fmt.Errorf("check asset exists: %w", err)
	}

	return true, nil
}
