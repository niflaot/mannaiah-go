package port

import "context"

// ContactSnapshot defines subset contact values required by membership use-cases.
type ContactSnapshot struct {
	// ID defines contact identifier values.
	ID string
	// Email defines contact email values.
	Email string
	// Metadata defines contact metadata values.
	Metadata map[string]string
}

// ContactLookup defines contact lookup behavior required by membership use-cases.
type ContactLookup interface {
	// FindByEmail resolves one contact by normalized email values.
	FindByEmail(ctx context.Context, email string) (*ContactSnapshot, error)
	// ListByMetadata resolves contacts by metadata key/value filters.
	ListByMetadata(ctx context.Context, metadataKey string, metadataValue string, page int, limit int) ([]ContactSnapshot, int64, error)
}
