package storage

import (
	"context"
	errorspkg "errors"
	"testing"

	"mannaiah/module/assets/port"
	corestorage "mannaiah/module/core/storage"
)

// coreStoreMock defines core storage behavior for adapter tests.
type coreStoreMock struct {
	// uploadFn defines upload behavior.
	uploadFn func(ctx context.Context, request corestorage.UploadRequest) error
	// downloadFn defines download behavior.
	downloadFn func(ctx context.Context, key string) ([]byte, error)
	// deleteFn defines delete behavior.
	deleteFn func(ctx context.Context, key string) error
	// existsFn defines exists behavior.
	existsFn func(ctx context.Context, key string) (bool, error)
	// availabilityErr defines availability behavior.
	availabilityErr error
}

// Upload executes configured upload behavior.
func (m coreStoreMock) Upload(ctx context.Context, request corestorage.UploadRequest) error {
	return m.uploadFn(ctx, request)
}

// Download executes configured download behavior.
func (m coreStoreMock) Download(ctx context.Context, key string) ([]byte, error) {
	return m.downloadFn(ctx, key)
}

// Delete executes configured delete behavior.
func (m coreStoreMock) Delete(ctx context.Context, key string) error {
	return m.deleteFn(ctx, key)
}

// Exists executes configured exists behavior.
func (m coreStoreMock) Exists(ctx context.Context, key string) (bool, error) {
	return m.existsFn(ctx, key)
}

// AvailabilityError returns configured availability behavior.
func (m coreStoreMock) AvailabilityError() error {
	return m.availabilityErr
}

// TestNewCoreStoreAdapter validates constructor behavior.
func TestNewCoreStoreAdapter(t *testing.T) {
	if _, err := NewCoreStoreAdapter(nil); !errorspkg.Is(err, ErrNilCoreStore) {
		t.Fatalf("NewCoreStoreAdapter(nil) error = %v, want ErrNilCoreStore", err)
	}
}

// TestCoreStoreAdapter verifies delegated behavior.
func TestCoreStoreAdapter(t *testing.T) {
	adapter, err := NewCoreStoreAdapter(coreStoreMock{
		uploadFn:        func(ctx context.Context, request corestorage.UploadRequest) error { return nil },
		downloadFn:      func(ctx context.Context, key string) ([]byte, error) { return []byte("payload"), nil },
		deleteFn:        func(ctx context.Context, key string) error { return nil },
		existsFn:        func(ctx context.Context, key string) (bool, error) { return true, nil },
		availabilityErr: errorspkg.New("disabled"),
	})
	if err != nil {
		t.Fatalf("NewCoreStoreAdapter() error = %v", err)
	}

	if uploadErr := adapter.Upload(context.Background(), port.UploadRequest{
		Key:         "assets/a.png",
		ContentType: "image/png",
		Body:        []byte("payload"),
	}); uploadErr != nil {
		t.Fatalf("Upload() error = %v", uploadErr)
	}
	downloaded, downloadErr := adapter.Download(context.Background(), "assets/a.png")
	if downloadErr != nil {
		t.Fatalf("Download() error = %v", downloadErr)
	}
	if string(downloaded) != "payload" {
		t.Fatalf("Download() = %q, want %q", string(downloaded), "payload")
	}
	if deleteErr := adapter.Delete(context.Background(), "assets/a.png"); deleteErr != nil {
		t.Fatalf("Delete() error = %v", deleteErr)
	}
	exists, existsErr := adapter.Exists(context.Background(), "assets/a.png")
	if existsErr != nil {
		t.Fatalf("Exists() error = %v", existsErr)
	}
	if !exists {
		t.Fatalf("Exists() = false, want true")
	}
	if adapter.AvailabilityError() == nil {
		t.Fatalf("expected availability error")
	}
}
