package manual

import (
	"context"
	"strings"
	"time"

	"mannaiah/module/shipping/domain"
)

// Provider defines manual carrier-provider behavior.
type Provider struct {
	// carrier defines static manual-carrier metadata.
	carrier domain.Carrier
}

// NewProvider creates manual carrier providers.
func NewProvider() *Provider {
	return &Provider{carrier: domain.Carrier{
		ID:                   "manual",
		Name:                 "Manual",
		Type:                 domain.CarrierTypeManual,
		Active:               true,
		RequiresBalanceCheck: false,
		HasQuotation:         false,
		HasManifestDocument:  false,
		HasTracking:          false,
		NeedsURL:             true,
	}}
}

// CarrierID returns the manual carrier identifier.
func (p *Provider) CarrierID() string {
	return p.carrier.ID
}

// Carrier returns manual carrier metadata.
func (p *Provider) Carrier() domain.Carrier {
	return p.carrier
}

// Quote returns not-supported errors for manual carriers.
func (p *Provider) Quote(ctx context.Context, request domain.QuotationRequest) (*domain.QuotationResult, error) {
	return nil, domain.ErrQuotationNotSupported
}

// GenerateMark marks manual requests as generated and auto-assigns tracking placeholders when needed.
func (p *Provider) GenerateMark(ctx context.Context, mark *domain.ShippingMark) error {
	if mark == nil {
		return domain.ErrInvalidID
	}
	if strings.TrimSpace(mark.TrackingNumber) == "" {
		mark.TrackingNumber = "MANUAL-" + strings.ReplaceAll(strings.ToUpper(strings.TrimSpace(mark.ID)), "-", "")
	}
	mark.Status = domain.MarkStatusGenerated
	if strings.TrimSpace(mark.DocumentRef) != "" {
		if mark.DocumentType == "" {
			mark.DocumentType = domain.MarkDocumentFile
		}
	} else {
		mark.DocumentType = ""
	}
	mark.UpdatedAt = time.Now().UTC()

	return nil
}

// VoidMark accepts manual void requests without remote APIs.
func (p *Provider) VoidMark(ctx context.Context, trackingNumber string) error {
	return nil
}

// CheckBalance always succeeds for manual carriers.
func (p *Provider) CheckBalance(ctx context.Context) error {
	return nil
}

// SupportsQuotation reports quotation support for manual carriers.
func (p *Provider) SupportsQuotation() bool {
	return false
}

// SupportsCourier reports whether manual providers support one carrier identifier.
func (p *Provider) SupportsCourier(carrierID string) bool {
	return strings.EqualFold(strings.TrimSpace(carrierID), p.carrier.ID)
}

// GetTrackingHistory returns placeholder tracking history for manual carriers.
func (p *Provider) GetTrackingHistory(ctx context.Context, trackingNumber string) (*domain.TrackingHistory, error) {
	trimmedTracking := strings.TrimSpace(trackingNumber)
	if trimmedTracking == "" {
		return nil, domain.ErrInvalidID
	}
	now := time.Now().UTC()

	return &domain.TrackingHistory{
		CarrierID:      p.carrier.ID,
		TrackingNumber: trimmedTracking,
		GlobalStatus:   domain.TrackingStatusProcessing,
		LastUpdate:     now,
		History: []domain.TrackingEvent{
			{Date: now, Code: "MANUAL", Text: "Manual tracking only", Status: domain.TrackingStatusProcessing},
		},
	}, nil
}
