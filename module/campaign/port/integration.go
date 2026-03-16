package port

import (
	"context"
	"time"
)

// DeliveryRow defines a campaign delivery projection for query results.
type DeliveryRow struct {
	// ContactID defines recipient contact identifier values.
	ContactID string
	// Email defines recipient email values.
	Email string
	// Status defines current delivery status values.
	Status string
	// CreatedAt defines delivery creation timestamps.
	CreatedAt time.Time
	// UpdatedAt defines delivery last-update timestamps.
	UpdatedAt time.Time
}

// DeliveryReader defines campaign delivery read behavior.
type DeliveryReader interface {
	// ListByCampaignID retrieves paginated delivery rows for a campaign.
	ListByCampaignID(ctx context.Context, campaignID string, page int, limit int) ([]DeliveryRow, int64, error)
}

// SegmentResolver defines campaign audience resolution behavior.
type SegmentResolver interface {
	// ResolveSegment resolves contact ids for a segment.
	ResolveSegment(ctx context.Context, segmentID string, page int, limit int) ([]string, error)
	// ResolveEmails resolves recipient emails by contact ids.
	ResolveEmails(ctx context.Context, contactIDs []string) (map[string]string, error)
}

// EmailSender defines outbound email sending behavior.
type EmailSender interface {
	// SendCampaignEmail sends one campaign email.
	SendCampaignEmail(ctx context.Context, contactID string, email string, subject string, htmlBody string, textBody string, idempotencyKey string) error
}
