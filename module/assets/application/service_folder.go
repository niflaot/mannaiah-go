package application

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"mannaiah/module/assets/domain"
	"mannaiah/module/assets/port"
)

// CreateFolder creates logical folders.
func (s *AssetService) CreateFolder(ctx context.Context, command CreateFolderCommand) (*domain.Folder, error) {
	entity := &domain.Folder{
		ID:   uuid.NewString(),
		Name: strings.TrimSpace(command.Name),
		Tags: command.Tags,
	}
	entity.Normalize()
	if err := entity.ValidateCreate(); err != nil {
		if err == domain.ErrFolderNameRequired {
			return nil, ErrInvalidFolderName
		}
		return nil, err
	}

	if err := s.repository.CreateFolder(ctx, entity); err != nil {
		return nil, fmt.Errorf("create asset folder: %w", err)
	}

	if publishErr := s.publisher.Publish(ctx, buildFolderCreatedIntegrationEvent(*entity)); publishErr != nil {
		return nil, fmt.Errorf("publish folder created event: %w", publishErr)
	}

	return entity, nil
}

// GetFolder retrieves folders by id.
func (s *AssetService) GetFolder(ctx context.Context, id string) (*domain.Folder, error) {
	trimmedID := strings.TrimSpace(id)
	if trimmedID == "" {
		return nil, ErrInvalidFolderID
	}

	entity, err := s.repository.GetFolderByID(ctx, trimmedID)
	if err != nil {
		return nil, fmt.Errorf("get asset folder: %w", err)
	}

	return entity, nil
}

// ListFolders paginates folders.
func (s *AssetService) ListFolders(ctx context.Context, query ListQuery) (*port.FolderPageResult, error) {
	result, err := s.repository.ListFolders(ctx, port.ListQuery{
		Page:    query.Page,
		Limit:   query.Limit,
		Filters: strings.TrimSpace(query.Filters),
	})
	if err != nil {
		return nil, fmt.Errorf("list asset folders: %w", err)
	}

	return result, nil
}

// UpdateFolder updates mutable folder fields.
func (s *AssetService) UpdateFolder(ctx context.Context, id string, command UpdateFolderCommand) (*domain.Folder, error) {
	trimmedID := strings.TrimSpace(id)
	if trimmedID == "" {
		return nil, ErrInvalidFolderID
	}
	if command.Name != nil && strings.TrimSpace(*command.Name) == "" {
		return nil, ErrInvalidFolderName
	}

	unlock := s.locks.Lock("folder:" + trimmedID)
	defer unlock()

	entity, err := s.repository.UpdateFolder(ctx, trimmedID, port.FolderUpdate{
		Name: command.Name,
		Tags: command.Tags,
	})
	if err != nil {
		return nil, fmt.Errorf("update asset folder: %w", err)
	}

	if publishErr := s.publisher.Publish(ctx, buildFolderUpdatedIntegrationEvent(*entity)); publishErr != nil {
		return nil, fmt.Errorf("publish folder updated event: %w", publishErr)
	}

	return entity, nil
}

// DeleteFolder soft-deletes folders and detaches linked assets.
func (s *AssetService) DeleteFolder(ctx context.Context, id string) error {
	trimmedID := strings.TrimSpace(id)
	if trimmedID == "" {
		return ErrInvalidFolderID
	}

	unlock := s.locks.Lock("folder:" + trimmedID)
	defer unlock()

	entity, err := s.repository.GetFolderByID(ctx, trimmedID)
	if err != nil {
		return fmt.Errorf("load asset folder for delete: %w", err)
	}

	if err := s.repository.SoftDeleteFolder(ctx, trimmedID); err != nil {
		return fmt.Errorf("soft delete asset folder: %w", err)
	}

	if publishErr := s.publisher.Publish(ctx, buildFolderDeletedIntegrationEvent(*entity)); publishErr != nil {
		return fmt.Errorf("publish folder deleted event: %w", publishErr)
	}

	return nil
}
