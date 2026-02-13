package store

import (
	"context"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"
	"mannaiah/module/assets/port"
)

var (
	// ErrNilDB is returned when a nil DB dependency is provided.
	ErrNilDB = errors.New("assets db must not be nil")
)

// Repository implements asset persistence using GORM.
type Repository struct {
	// db is the underlying GORM handle.
	db *gorm.DB
}

// assetRecord defines persistence schema for assets.
type assetRecord struct {
	// ID defines primary key identifiers.
	ID string `gorm:"primaryKey;size:64"`
	// Key defines storage object key paths.
	Key string `gorm:"uniqueIndex:idx_assets_key;size:512;not null"`
	// Name defines custom display names.
	Name string `gorm:"size:255;not null"`
	// OriginalName defines original uploaded file names.
	OriginalName string `gorm:"size:255;not null"`
	// FolderID defines logical folder identifiers.
	FolderID *string `gorm:"size:64;index"`
	// MimeType defines object mime type values.
	MimeType string `gorm:"size:255;not null"`
	// Size defines object size in bytes.
	Size int64 `gorm:"not null"`
	// TagsJSON defines serialized tag values.
	TagsJSON string `gorm:"type:text"`
	// MetadataJSON defines serialized metadata values.
	MetadataJSON string `gorm:"type:text"`
	// CreatedAt defines creation timestamps.
	CreatedAt time.Time
	// UpdatedAt defines update timestamps.
	UpdatedAt time.Time
	// DeletedAt defines soft-delete timestamps.
	DeletedAt gorm.DeletedAt `gorm:"index"`
}

// folderRecord defines persistence schema for logical folders.
type folderRecord struct {
	// ID defines primary key identifiers.
	ID string `gorm:"primaryKey;size:64"`
	// Name defines folder names.
	Name string `gorm:"size:255;not null"`
	// Slug defines normalized folder slugs.
	Slug string `gorm:"size:191;uniqueIndex:idx_asset_folders_slug;not null"`
	// ParentFolderID defines optional parent-folder identifiers for nested folders.
	ParentFolderID *string `gorm:"size:64;index"`
	// TagsJSON defines serialized tag values.
	TagsJSON string `gorm:"type:text"`
	// CreatedAt defines creation timestamps.
	CreatedAt time.Time
	// UpdatedAt defines update timestamps.
	UpdatedAt time.Time
	// DeletedAt defines soft-delete timestamps.
	DeletedAt gorm.DeletedAt `gorm:"index"`
}

// TableName defines storage table names.
func (assetRecord) TableName() string {
	return "assets"
}

// TableName defines storage table names.
func (folderRecord) TableName() string {
	return "asset_folders"
}

var (
	// _ ensures Repository satisfies asset repository contracts.
	_ port.Repository = (*Repository)(nil)
)

// NewRepository creates an asset repository over GORM.
func NewRepository(db *gorm.DB) (*Repository, error) {
	if db == nil {
		return nil, ErrNilDB
	}

	return &Repository{db: db}, nil
}

// EnsureSchema migrates asset persistence schema.
func (r *Repository) EnsureSchema(ctx context.Context) error {
	if err := r.db.WithContext(ctx).AutoMigrate(&folderRecord{}, &assetRecord{}); err != nil {
		return fmt.Errorf("migrate asset schema: %w", err)
	}

	return nil
}
