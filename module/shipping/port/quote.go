package port

import (
	"context"

	"mannaiah/module/shipping/domain"
)

// RateQuoteGateway defines carrier-agnostic quote adapter behavior.
type RateQuoteGateway interface {
	// Quote retrieves one shipping quote from one carrier adapter.
	Quote(ctx context.Context, request domain.QuoteRequest) (*domain.QuoteResult, error)
}
