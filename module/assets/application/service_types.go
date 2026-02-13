package application

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"

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
	// ErrInvalidFolderID is returned when folder ids are empty.
	ErrInvalidFolderID = errors.New("asset folder id is required")
	// ErrInvalidFolderName is returned when folder names are empty.
	ErrInvalidFolderName = errors.New("asset folder name is required")
	// ErrInvalidFolderParent is returned when folder parent assignments are invalid.
	ErrInvalidFolderParent = errors.New("asset folder parent is invalid")
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
	// FolderID defines optional logical folder assignment values.
	FolderID string
	// MimeType defines file mime types.
	MimeType string
	// Size defines payload size in bytes.
	Size int64
	// Body defines raw file payload bytes.
	Body []byte
	// Tags defines optional classification tags.
	Tags []domain.Tag
	// Metadata defines optional key-value metadata values.
	Metadata map[string]string
}

// UpdateCommand defines update-asset command payloads.
type UpdateCommand struct {
	// Name defines optional custom display-name updates.
	Name *string
	// FolderID defines optional folder assignment updates.
	FolderID *string
	// Tags defines optional tag replacement updates.
	Tags *[]domain.Tag
	// Metadata defines optional metadata replacement updates.
	Metadata *map[string]string
}

// ListQuery defines list-asset query values.
type ListQuery struct {
	// Page defines page numbers.
	Page int
	// Limit defines page size values.
	Limit int
	// Filters defines optional free-text filters.
	Filters string
	// ParentFolderID defines optional parent-folder filters for nested folder queries.
	ParentFolderID string
}

// CreateFolderCommand defines create-folder command payloads.
type CreateFolderCommand struct {
	// Name defines folder display names.
	Name string
	// ParentFolderID defines optional parent-folder assignments.
	ParentFolderID string
	// Tags defines optional folder classification tags.
	Tags []domain.Tag
}

// UpdateFolderCommand defines update-folder command payloads.
type UpdateFolderCommand struct {
	// Name defines optional folder-name updates.
	Name *string
	// ParentFolderID defines optional parent-folder assignment updates.
	ParentFolderID *string
	// Tags defines optional folder-tag updates.
	Tags *[]domain.Tag
}

// Service defines asset/folder use-case behavior.
type Service interface {
	// Create uploads binaries and persists metadata.
	Create(ctx context.Context, command CreateCommand) (*domain.Asset, error)
	// Get retrieves assets by id.
	Get(ctx context.Context, id string) (*domain.Asset, error)
	// List paginates assets.
	List(ctx context.Context, query ListQuery) (*port.PageResult, error)
	// Update updates mutable asset fields.
	Update(ctx context.Context, id string, command UpdateCommand) (*domain.Asset, error)
	// UpdateName updates asset custom names.
	UpdateName(ctx context.Context, id string, name string) (*domain.Asset, error)
	// Delete soft-deletes metadata.
	Delete(ctx context.Context, id string) error
	// Exists verifies whether metadata rows exist by id.
	Exists(ctx context.Context, id string) (bool, error)
	// CreateFolder creates logical folders.
	CreateFolder(ctx context.Context, command CreateFolderCommand) (*domain.Folder, error)
	// GetFolder retrieves folders by id.
	GetFolder(ctx context.Context, id string) (*domain.Folder, error)
	// ListFolders paginates folders.
	ListFolders(ctx context.Context, query ListQuery) (*port.FolderPageResult, error)
	// UpdateFolder updates mutable folder fields.
	UpdateFolder(ctx context.Context, id string, command UpdateFolderCommand) (*domain.Folder, error)
	// DeleteFolder soft-deletes folders and detaches linked assets.
	DeleteFolder(ctx context.Context, id string) error
}

// AssetService defines asset use-case dependencies.
type AssetService struct {
	// repository defines metadata persistence dependencies.
	repository port.Repository
	// storage defines binary storage dependencies.
	storage port.Storage
	// publisher defines optional integration event dependencies.
	publisher port.IntegrationEventPublisher
	// locks serializes in-process writes for shared ids.
	locks locker
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
		locks:      newKeyedLocker(),
	}, nil
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
