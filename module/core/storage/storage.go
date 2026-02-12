package storage

import (
	"context"

	"go.uber.org/zap"
	storages3 "mannaiah/module/core/storage/s3"
)

// Config defines storage runtime configuration.
type Config = storages3.Config

// UploadRequest defines object upload input values.
type UploadRequest = storages3.UploadRequest

// Store defines provider-agnostic object storage behavior.
type Store interface {
	// Upload uploads object bytes to storage.
	Upload(ctx context.Context, request UploadRequest) error
	// Delete removes object keys from storage.
	Delete(ctx context.Context, key string) error
	// Exists verifies whether object keys exist.
	Exists(ctx context.Context, key string) (bool, error)
	// AvailabilityError reports storage availability failures when disabled/unavailable.
	AvailabilityError() error
}

var (
	// ErrUnavailable is returned when storage integration is disabled or unavailable.
	ErrUnavailable = storages3.ErrUnavailable
	// ErrInvalidKey is returned when storage keys are empty.
	ErrInvalidKey = storages3.ErrInvalidKey
	// ErrEmptyBody is returned when upload bodies are empty.
	ErrEmptyBody = storages3.ErrEmptyBody
)

// NewS3 creates S3-backed storage dependencies.
func NewS3(cfg Config, logger *zap.Logger) Store {
	return storages3.New(cfg, logger)
}

// Disabled creates a disabled storage store with a fixed availability reason.
func Disabled(reason error) Store {
	return storages3.Disabled(reason)
}
