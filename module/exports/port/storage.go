package port

import "context"

// UploadRequest defines export object upload input values.
type UploadRequest struct {
	// Key defines storage object keys.
	Key string
	// ContentType defines object MIME content types.
	ContentType string
	// Body defines object bytes.
	Body []byte
}

// Storage defines object storage behavior for generated reports.
type Storage interface {
	// Upload writes report object bytes to storage.
	Upload(ctx context.Context, request UploadRequest) error
	// AvailabilityError reports storage availability failures.
	AvailabilityError() error
}
