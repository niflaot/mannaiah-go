package storage

import (
	"context"
	"errors"

	corestorage "mannaiah/module/core/storage"
	"mannaiah/module/exports/port"
)

var (
	// ErrNilCoreStore is returned when core storage dependencies are nil.
	ErrNilCoreStore = errors.New("core storage store must not be nil")
)

// CoreStoreAdapter adapts core storage stores to export storage ports.
type CoreStoreAdapter struct {
	// store defines core storage dependencies.
	store corestorage.Store
}

var (
	// _ ensures CoreStoreAdapter satisfies export storage ports.
	_ port.Storage = (*CoreStoreAdapter)(nil)
)

// NewCoreStoreAdapter creates storage adapters over core storage dependencies.
func NewCoreStoreAdapter(store corestorage.Store) (*CoreStoreAdapter, error) {
	if store == nil {
		return nil, ErrNilCoreStore
	}

	return &CoreStoreAdapter{store: store}, nil
}

// Upload writes report object bytes to storage.
func (a *CoreStoreAdapter) Upload(ctx context.Context, request port.UploadRequest) error {
	return a.store.Upload(ctx, corestorage.UploadRequest{
		Key:         request.Key,
		ContentType: request.ContentType,
		Body:        request.Body,
	})
}

// AvailabilityError reports storage availability failures.
func (a *CoreStoreAdapter) AvailabilityError() error {
	return a.store.AvailabilityError()
}
