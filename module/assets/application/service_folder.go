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

// CreateFolder creates logical folders.
func (s *AssetService) CreateFolder(ctx context.Context, command CreateFolderCommand) (*domain.Folder, error) {
	folderID := uuid.NewString()
	parentFolderID := strings.TrimSpace(command.ParentFolderID)
	if parentFolderID != "" {
		if parentErr := s.validateFolderParent(ctx, folderID, parentFolderID); parentErr != nil {
			return nil, parentErr
		}
	}

	entity := &domain.Folder{
		ID:             folderID,
		Name:           strings.TrimSpace(command.Name),
		ParentFolderID: parentFolderID,
		Tags:           command.Tags,
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
		Page:           query.Page,
		Limit:          query.Limit,
		Filters:        strings.TrimSpace(query.Filters),
		ParentFolderID: strings.TrimSpace(query.ParentFolderID),
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
	if command.ParentFolderID != nil {
		parentFolderID := strings.TrimSpace(*command.ParentFolderID)
		if parentErr := s.validateFolderParent(ctx, trimmedID, parentFolderID); parentErr != nil {
			return nil, parentErr
		}
	}

	unlock := s.locks.Lock("folder:" + trimmedID)
	defer unlock()

	entity, err := s.repository.UpdateFolder(ctx, trimmedID, port.FolderUpdate{
		Name:           command.Name,
		ParentFolderID: command.ParentFolderID,
		Tags:           command.Tags,
	})
	if err != nil {
		return nil, fmt.Errorf("update asset folder: %w", err)
	}

	if publishErr := s.publisher.Publish(ctx, buildFolderUpdatedIntegrationEvent(*entity)); publishErr != nil {
		return nil, fmt.Errorf("publish folder updated event: %w", publishErr)
	}

	return entity, nil
}

// validateFolderParent validates parent-folder assignment consistency.
func (s *AssetService) validateFolderParent(ctx context.Context, folderID string, parentFolderID string) error {
	trimmedFolderID := strings.TrimSpace(folderID)
	trimmedParentID := strings.TrimSpace(parentFolderID)
	if trimmedParentID == "" {
		return nil
	}
	if trimmedFolderID != "" && trimmedParentID == trimmedFolderID {
		return ErrInvalidFolderParent
	}

	parent, err := s.repository.GetFolderByID(ctx, trimmedParentID)
	if err != nil {
		if errors.Is(err, port.ErrNotFound) || errors.Is(err, port.ErrFolderNotFound) {
			return port.ErrFolderNotFound
		}
		return fmt.Errorf("load parent asset folder: %w", err)
	}

	visited := map[string]struct{}{}
	current := strings.TrimSpace(parent.ParentFolderID)
	for current != "" {
		if _, exists := visited[current]; exists {
			return ErrInvalidFolderParent
		}
		if trimmedFolderID != "" && current == trimmedFolderID {
			return ErrInvalidFolderParent
		}

		visited[current] = struct{}{}
		node, nodeErr := s.repository.GetFolderByID(ctx, current)
		if nodeErr != nil {
			if errors.Is(nodeErr, port.ErrNotFound) || errors.Is(nodeErr, port.ErrFolderNotFound) {
				return port.ErrFolderNotFound
			}
			return fmt.Errorf("load ancestor asset folder: %w", nodeErr)
		}
		current = strings.TrimSpace(node.ParentFolderID)
	}

	return nil
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
