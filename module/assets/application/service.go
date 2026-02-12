package application

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
	"mannaiah/module/assets/domain"
	"mannaiah/module/assets/port"
)

const (
	// maxUploadBytes defines maximum accepted upload size.
	maxUploadBytes = int64(10 * 1024 * 1024)
	// keyPrefix defines uploaded object key prefix values.
	keyPrefix = "assets"
)

var (
	// ErrNilRepository is returned when repository dependencies are nil.
	ErrNilRepository = errors.New("assets repository must not be nil")
	// ErrNilStorage is returned when storage dependencies are nil.
	ErrNilStorage = errors.New("assets storage must not be nil")
	// ErrInvalidID is returned when id values are empty.
	ErrInvalidID = errors.New("asset id is required")
	// ErrInvalidName is returned when update names are empty.
	ErrInvalidName = errors.New("asset name is required")
	// ErrFileRequired is returned when file content is missing.
	ErrFileRequired = errors.New("asset file is required")
	// ErrFileTooLarge is returned when upload files exceed size limits.
	ErrFileTooLarge = errors.New("asset file size exceeds max 10MB")
	// ErrStorageUnavailable is returned when storage integration is unavailable.
	ErrStorageUnavailable = errors.New("asset storage is unavailable")
)

// CreateCommand defines create-asset command payloads.
type CreateCommand struct {
	// Name defines optional custom display names.
	Name string
	// OriginalName defines uploaded file names.
	OriginalName string
	// MimeType defines file mime types.
	MimeType string
	// Size defines payload size in bytes.
	Size int64
	// Body defines raw file payload bytes.
	Body []byte
}

// ListQuery defines list-asset query values.
type ListQuery struct {
	// Page defines page numbers.
	Page int
	// Limit defines page size values.
	Limit int
	// Filters defines optional free-text filters.
	Filters string
}

// Service defines asset use-case behavior.
type Service interface {
	// Create uploads binaries and persists metadata.
	Create(ctx context.Context, command CreateCommand) (*domain.Asset, error)
	// Get retrieves assets by id.
	Get(ctx context.Context, id string) (*domain.Asset, error)
	// List paginates assets.
	List(ctx context.Context, query ListQuery) (*port.PageResult, error)
	// UpdateName updates asset custom names.
	UpdateName(ctx context.Context, id string, name string) (*domain.Asset, error)
	// Delete hard-deletes storage objects and soft-deletes metadata.
	Delete(ctx context.Context, id string) error
	// Exists verifies whether metadata rows exist by id.
	Exists(ctx context.Context, id string) (bool, error)
}

// AssetService defines asset use-case dependencies.
type AssetService struct {
	// repository defines metadata persistence dependencies.
	repository port.Repository
	// storage defines binary storage dependencies.
	storage port.Storage
	// publisher defines optional integration event dependencies.
	publisher port.IntegrationEventPublisher
}

var (
	// _ ensures AssetService satisfies service contracts.
	_ Service = (*AssetService)(nil)
)

// NewService creates asset use-case services.
func NewService(repository port.Repository, storage port.Storage, publishers ...port.IntegrationEventPublisher) (*AssetService, error) {
	if repository == nil {
		return nil, ErrNilRepository
	}
	if storage == nil {
		return nil, ErrNilStorage
	}

	return &AssetService{
		repository: repository,
		storage:    storage,
		publisher:  resolvePublisher(publishers),
	}, nil
}

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
		MimeType:     strings.TrimSpace(command.MimeType),
		Size:         command.Size,
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
		Page:    query.Page,
		Limit:   query.Limit,
		Filters: strings.TrimSpace(query.Filters),
	})
	if err != nil {
		return nil, fmt.Errorf("list assets: %w", err)
	}

	return result, nil
}

// UpdateName updates asset custom names.
func (s *AssetService) UpdateName(ctx context.Context, id string, name string) (*domain.Asset, error) {
	trimmedID := strings.TrimSpace(id)
	if trimmedID == "" {
		return nil, ErrInvalidID
	}
	trimmedName := strings.TrimSpace(name)
	if trimmedName == "" {
		return nil, ErrInvalidName
	}

	entity, err := s.repository.UpdateName(ctx, trimmedID, trimmedName)
	if err != nil {
		return nil, fmt.Errorf("update asset name: %w", err)
	}

	if publishErr := s.publisher.Publish(ctx, buildAssetUpdatedIntegrationEvent(*entity)); publishErr != nil {
		return nil, fmt.Errorf("publish asset updated event: %w", publishErr)
	}

	return entity, nil
}

// Delete hard-deletes storage objects and soft-deletes metadata.
func (s *AssetService) Delete(ctx context.Context, id string) error {
	if err := s.ensureStorage(); err != nil {
		return err
	}
	trimmedID := strings.TrimSpace(id)
	if trimmedID == "" {
		return ErrInvalidID
	}

	entity, err := s.repository.GetByID(ctx, trimmedID)
	if err != nil {
		return fmt.Errorf("load asset for delete: %w", err)
	}

	if err := s.storage.Delete(ctx, entity.Key); err != nil {
		return fmt.Errorf("delete asset object: %w", err)
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

// ensureStorage verifies storage integration availability.
func (s *AssetService) ensureStorage() error {
	if availabilityErr := s.storage.AvailabilityError(); availabilityErr != nil {
		return fmt.Errorf("%w: %v", ErrStorageUnavailable, availabilityErr)
	}

	return nil
}

// buildStorageKey builds deterministic object key paths.
func buildStorageKey(id string, originalName string) string {
	trimmedID := strings.TrimSpace(id)
	trimmedOriginalName := strings.TrimSpace(originalName)
	base := filepath.Base(trimmedOriginalName)
	base = strings.ReplaceAll(base, " ", "-")
	if base == "" || base == "." || base == string(filepath.Separator) {
		base = "file"
	}

	return fmt.Sprintf("%s/%s-%s", keyPrefix, trimmedID, base)
}
