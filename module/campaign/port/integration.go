package port

import "context"

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
