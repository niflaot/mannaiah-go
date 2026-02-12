package port

import "context"

// Storage defines binary object storage behavior required by assets.
type Storage interface {
	// Upload uploads object bytes to storage.
	Upload(ctx context.Context, request UploadRequest) error
	// Delete removes object keys from storage.
	Delete(ctx context.Context, key string) error
	// Exists verifies whether object keys exist.
	Exists(ctx context.Context, key string) (bool, error)
	// AvailabilityError reports integration availability failures.
	AvailabilityError() error
}

// UploadRequest defines object upload input values.
type UploadRequest struct {
	// Key defines object key paths.
	Key string
	// ContentType defines object mime types.
	ContentType string
	// Body defines raw payload bytes.
	Body []byte
}
