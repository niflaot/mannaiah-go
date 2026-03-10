package port

import (
	"context"
	"errors"

	"mannaiah/module/assets/domain"
)

var (
	// ErrNotFound is returned when asset records are missing.
	ErrNotFound = errors.New("asset not found")
	// ErrFolderNotFound is returned when folder records are missing.
	ErrFolderNotFound = errors.New("asset folder not found")
	// ErrFolderAlreadyExists is returned when folder name already exists under the same parent path.
	ErrFolderAlreadyExists = errors.New("asset folder already exists")
)

// ListQuery defines list-assets query values.
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

// PageResult defines paginated list response values.
type PageResult struct {
	// Data defines current page rows.
	Data []domain.Asset
	// Total defines total rows.
	Total int64
	// Page defines current page numbers.
	Page int
	// Limit defines current page size values.
	Limit int
}

// FolderPageResult defines paginated folder-list response values.
type FolderPageResult struct {
	// Data defines current page rows.
	Data []domain.Folder
	// Total defines total rows.
	Total int64
	// Page defines current page numbers.
	Page int
	// Limit defines current page size values.
	Limit int
}

// AssetUpdate defines partial asset update operations.
type AssetUpdate struct {
	// Name defines optional display name updates.
	Name *string
	// FolderID defines optional folder assignment updates.
	FolderID *string
	// Tags defines optional tags updates.
	Tags *[]domain.Tag
	// Metadata defines optional metadata updates.
	Metadata *map[string]string
}

// AssetBinaryUpdate defines immutable-ish binary field updates.
type AssetBinaryUpdate struct {
	// Key defines storage object key paths.
	Key string
	// OriginalName defines uploaded file names.
	OriginalName string
	// MimeType defines payload mime types.
	MimeType string
	// Size defines payload size in bytes.
	Size int64
}

// FolderUpdate defines partial folder update operations.
type FolderUpdate struct {
	// Name defines optional folder name updates.
	Name *string
	// ParentFolderID defines optional parent-folder assignment updates.
	ParentFolderID *string
	// Tags defines optional folder tag updates.
	Tags *[]domain.Tag
}

// Repository defines asset metadata persistence behavior.
type Repository interface {
	// EnsureSchema ensures storage schema availability.
	EnsureSchema(ctx context.Context) error
	// Create persists asset metadata rows.
	Create(ctx context.Context, asset *domain.Asset) error
	// GetByID loads asset metadata rows by id.
	GetByID(ctx context.Context, id string) (*domain.Asset, error)
	// List paginates asset metadata rows.
	List(ctx context.Context, query ListQuery) (*PageResult, error)
	// Update updates asset metadata fields.
	Update(ctx context.Context, id string, update AssetUpdate) (*domain.Asset, error)
	// UpdateBinary updates binary-related fields for an existing asset.
	UpdateBinary(ctx context.Context, id string, update AssetBinaryUpdate) (*domain.Asset, error)
	// ListByTagNames loads tagged assets that still require JPG conversion.
	ListByTagNames(ctx context.Context, tagNames []string, limit int) ([]domain.Asset, error)
	// SoftDelete soft-deletes asset metadata rows.
	SoftDelete(ctx context.Context, id string) error
	// CreateFolder persists folder metadata rows.
	CreateFolder(ctx context.Context, folder *domain.Folder) error
	// GetFolderByID loads folder metadata rows by id.
	GetFolderByID(ctx context.Context, id string) (*domain.Folder, error)
	// ListFolders paginates folder metadata rows.
	ListFolders(ctx context.Context, query ListQuery) (*FolderPageResult, error)
	// ListAllFolders loads all folder metadata rows for hierarchical tree construction.
	ListAllFolders(ctx context.Context) ([]domain.Folder, error)
	// UpdateFolder updates folder metadata fields.
	UpdateFolder(ctx context.Context, id string, update FolderUpdate) (*domain.Folder, error)
	// SoftDeleteFolder soft-deletes folder rows and detaches linked assets.
	SoftDeleteFolder(ctx context.Context, id string) error
	// ExistsFolder reports whether folders exist by id.
	ExistsFolder(ctx context.Context, id string) (bool, error)
}
