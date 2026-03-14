package port

import "context"

// SendRequest defines provider send payload values.
type SendRequest struct {
	// To defines recipient email values.
	To string
	// Subject defines subject values.
	Subject string
	// HTMLBody defines html payload values.
	HTMLBody string
	// TextBody defines text payload values.
	TextBody string
	// IdempotencyKey defines idempotency values.
	IdempotencyKey string
}

// Provider defines outbound email delivery behavior.
type Provider interface {
	// Send submits one email request and returns provider message ids.
	Send(ctx context.Context, request SendRequest) (string, error)
}
