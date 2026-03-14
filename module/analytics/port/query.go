package port

import (
	"context"
	"mannaiah/module/analytics/domain"
)

// Resolver defines analytical contact-resolution behavior.
type Resolver interface {
	// ResolveContacts resolves contact ids for analytical filters.
	ResolveContacts(ctx context.Context, filter domain.SegmentFilter, page int, limit int) ([]string, error)
	// CountContacts counts contact ids for analytical filters.
	CountContacts(ctx context.Context, filter domain.SegmentFilter) (int64, error)
}
