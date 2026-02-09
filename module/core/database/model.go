package database

import (
	"time"

	"gorm.io/gorm"
)

// Model defines the shared base model with primary key, timestamps, and soft delete support.
type Model struct {
	// ID is the primary key identifier.
	ID uint `gorm:"primaryKey"`
	// CreatedAt is the entity creation timestamp.
	CreatedAt time.Time
	// UpdatedAt is the entity update timestamp.
	UpdatedAt time.Time
	// DeletedAt is the soft-delete timestamp.
	DeletedAt gorm.DeletedAt `gorm:"index"`
}
