package database

import (
	"context"
	"errors"
	"fmt"

	"gorm.io/gorm"
)

var (
	// ErrNilDB is returned when a nil GORM DB dependency is provided.
	ErrNilDB = errors.New("gorm db must not be nil")
	// ErrNilEntity is returned when a nil entity pointer is provided.
	ErrNilEntity = errors.New("entity must not be nil")
	// ErrInvalidID is returned when a record ID is invalid.
	ErrInvalidID = errors.New("id must be greater than zero")
	// ErrEmptyUpdates is returned when update payload is empty.
	ErrEmptyUpdates = errors.New("update payload must not be empty")
	// ErrNotFound is returned when the target record is missing.
	ErrNotFound = errors.New("record not found")
)

// Query defines dynamic query options for Find operations.
type Query struct {
	// Where defines a SQL WHERE clause expression.
	Where string
	// Args defines the WHERE clause positional arguments.
	Args []any
	// Order defines an ORDER BY clause expression.
	Order string
	// Limit defines result limit for pagination.
	Limit int
	// Offset defines result offset for pagination.
	Offset int
	// Preloads defines association preload names.
	Preloads []string
	// Unscoped enables querying soft-deleted rows.
	Unscoped bool
}

// CRUDService defines the generic CRUD behavior available for typed models.
type CRUDService[T any] interface {
	// Create inserts a new model row.
	Create(ctx context.Context, entity *T) error
	// Read retrieves a model by primary key.
	Read(ctx context.Context, id uint) (*T, error)
	// Find retrieves models using query filters.
	Find(ctx context.Context, query Query) ([]T, error)
	// Update applies partial updates to a model by primary key.
	Update(ctx context.Context, id uint, updates map[string]any) error
	// Delete soft-deletes a model by primary key.
	Delete(ctx context.Context, id uint) error
}

// Service provides a reusable generic CRUD implementation over GORM.
type Service[T any] struct {
	// db is the underlying GORM database handle.
	db *gorm.DB
}

// NewService creates a generic CRUD service for the provided model type.
func NewService[T any](db *gorm.DB) (*Service[T], error) {
	if db == nil {
		return nil, ErrNilDB
	}

	return &Service[T]{db: db}, nil
}

// Create inserts a new model row.
func (s *Service[T]) Create(ctx context.Context, entity *T) error {
	if entity == nil {
		return ErrNilEntity
	}

	if err := s.db.WithContext(ctx).Create(entity).Error; err != nil {
		return fmt.Errorf("create record: %w", err)
	}

	return nil
}

// Read retrieves a model by primary key.
func (s *Service[T]) Read(ctx context.Context, id uint) (*T, error) {
	if id == 0 {
		return nil, ErrInvalidID
	}

	record := new(T)
	if err := s.db.WithContext(ctx).First(record, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("read record id %d: %w", id, err)
	}

	return record, nil
}

// Find retrieves models using query filters.
func (s *Service[T]) Find(ctx context.Context, query Query) ([]T, error) {
	tx := s.db.WithContext(ctx).Model(new(T))
	tx = applyQuery(tx, query)

	records := make([]T, 0)
	if err := tx.Find(&records).Error; err != nil {
		return nil, fmt.Errorf("find records: %w", err)
	}

	return records, nil
}

// Update applies partial updates to a model by primary key.
func (s *Service[T]) Update(ctx context.Context, id uint, updates map[string]any) error {
	if id == 0 {
		return ErrInvalidID
	}
	if len(updates) == 0 {
		return ErrEmptyUpdates
	}

	tx := s.db.WithContext(ctx).Model(new(T)).Where("id = ?", id).Updates(updates)
	if tx.Error != nil {
		return fmt.Errorf("update record id %d: %w", id, tx.Error)
	}
	if tx.RowsAffected == 0 {
		return ErrNotFound
	}

	return nil
}

// Delete soft-deletes a model by primary key.
func (s *Service[T]) Delete(ctx context.Context, id uint) error {
	if id == 0 {
		return ErrInvalidID
	}

	tx := s.db.WithContext(ctx).Delete(new(T), id)
	if tx.Error != nil {
		return fmt.Errorf("delete record id %d: %w", id, tx.Error)
	}
	if tx.RowsAffected == 0 {
		return ErrNotFound
	}

	return nil
}

// applyQuery applies dynamic query options to a GORM transaction.
func applyQuery(tx *gorm.DB, query Query) *gorm.DB {
	next := tx
	if query.Unscoped {
		next = next.Unscoped()
	}
	if query.Where != "" {
		next = next.Where(query.Where, query.Args...)
	}
	if query.Order != "" {
		next = next.Order(query.Order)
	}
	if query.Limit > 0 {
		next = next.Limit(query.Limit)
	}
	if query.Offset > 0 {
		next = next.Offset(query.Offset)
	}
	for _, preload := range query.Preloads {
		next = next.Preload(preload)
	}

	return next
}
