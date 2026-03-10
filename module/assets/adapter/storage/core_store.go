package storage

import (
	"context"
	"errors"

	"mannaiah/module/assets/port"
	corestorage "mannaiah/module/core/storage"
)

var (
	// ErrNilCoreStore is returned when a nil core storage dependency is provided.
	ErrNilCoreStore = errors.New("core storage store must not be nil")
)

// CoreStoreAdapter adapts core storage stores to asset storage ports.
type CoreStoreAdapter struct {
	// store defines core storage dependencies.
	store corestorage.Store
}

var (
	// _ ensures CoreStoreAdapter satisfies asset storage ports.
	_ port.Storage = (*CoreStoreAdapter)(nil)
)

// NewCoreStoreAdapter creates a storage adapter over core storage dependencies.
func NewCoreStoreAdapter(store corestorage.Store) (*CoreStoreAdapter, error) {
	if store == nil {
		return nil, ErrNilCoreStore
	}

	return &CoreStoreAdapter{store: store}, nil
}

// Upload uploads object bytes to storage.
func (a *CoreStoreAdapter) Upload(ctx context.Context, request port.UploadRequest) error {
	return a.store.Upload(ctx, corestorage.UploadRequest{
		Key:         request.Key,
		ContentType: request.ContentType,
		Body:        request.Body,
	})
}

// Download loads object bytes from storage.
func (a *CoreStoreAdapter) Download(ctx context.Context, key string) ([]byte, error) {
	return a.store.Download(ctx, key)
}

// Delete removes object keys from storage.
func (a *CoreStoreAdapter) Delete(ctx context.Context, key string) error {
	return a.store.Delete(ctx, key)
}

// Exists verifies whether object keys exist.
func (a *CoreStoreAdapter) Exists(ctx context.Context, key string) (bool, error) {
	return a.store.Exists(ctx, key)
}

// AvailabilityError reports integration availability failures.
func (a *CoreStoreAdapter) AvailabilityError() error {
	return a.store.AvailabilityError()
}
