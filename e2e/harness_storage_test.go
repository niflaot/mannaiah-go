package e2e_test

import (
	"context"
	"errors"
	"sync"

	assetport "mannaiah/module/assets/port"
)

// inMemoryAssetStorage defines e2e in-memory storage behavior for assets.
type inMemoryAssetStorage struct {
	// mu protects concurrent object operations.
	mu sync.RWMutex
	// objects stores keyed object payload values.
	objects map[string][]byte
}

// newInMemoryAssetStorage creates an in-memory asset storage implementation.
func newInMemoryAssetStorage() *inMemoryAssetStorage {
	return &inMemoryAssetStorage{objects: map[string][]byte{}}
}

// Upload stores payload bytes by key.
func (s *inMemoryAssetStorage) Upload(ctx context.Context, request assetport.UploadRequest) error {
	if s == nil {
		return errors.New("asset storage is nil")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	copied := make([]byte, len(request.Body))
	copy(copied, request.Body)
	s.objects[request.Key] = copied

	return nil
}

// Download loads payload bytes by key.
func (s *inMemoryAssetStorage) Download(ctx context.Context, key string) ([]byte, error) {
	if s == nil {
		return nil, errors.New("asset storage is nil")
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	payload, exists := s.objects[key]
	if !exists {
		return nil, errors.New("asset object does not exist")
	}

	copied := make([]byte, len(payload))
	copy(copied, payload)
	return copied, nil
}

// Delete removes payloads by key.
func (s *inMemoryAssetStorage) Delete(ctx context.Context, key string) error {
	if s == nil {
		return errors.New("asset storage is nil")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.objects, key)
	return nil
}

// Exists verifies whether payloads exist by key.
func (s *inMemoryAssetStorage) Exists(ctx context.Context, key string) (bool, error) {
	if s == nil {
		return false, errors.New("asset storage is nil")
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	_, exists := s.objects[key]
	return exists, nil
}

// AvailabilityError reports storage availability behavior.
func (s *inMemoryAssetStorage) AvailabilityError() error {
	return nil
}
