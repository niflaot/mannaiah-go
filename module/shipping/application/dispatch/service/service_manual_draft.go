package service

import (
	"context"
	"strings"
	"time"

	"mannaiah/module/shipping/domain"
)

// UpdateDraftMarkCommand defines manual draft completion input values.
type UpdateDraftMarkCommand struct {
	// BatchID defines the target batch identifier.
	BatchID string
	// MarkID defines the mark identifier being completed.
	MarkID string
	// QuotedFreightCost defines the manual freight cost entered by the operator.
	QuotedFreightCost float64
	// Observations defines the manual carrier label persisted as a normalized slug.
	Observations string
	// TrackingNumber defines the manual tracking-number value.
	TrackingNumber string
	// CustomTrackingURL defines the operator-provided tracking URL override.
	CustomTrackingURL *string
}

// UpdateDraftMark completes one existing manual QUOTED draft mark inside an open batch.
func (s *Service) UpdateDraftMark(ctx context.Context, command UpdateDraftMarkCommand) (*domain.ShippingMark, error) {
	if s == nil || s.batchRepository == nil || s.markRepository == nil {
		return nil, domain.ErrInvalidID
	}
	batch, err := s.batchRepository.GetByID(ctx, strings.TrimSpace(command.BatchID))
	if err != nil {
		return nil, err
	}
	if batch.Status != domain.BatchStatusOpen {
		return nil, domain.ErrBatchClosed
	}
	if !domain.IsManualCarrierID(batch.CarrierID) {
		return nil, domain.ErrManualDraftUpdateNotSupported
	}
	mark, err := s.markRepository.GetByID(ctx, strings.TrimSpace(command.MarkID))
	if err != nil {
		return nil, err
	}
	if mark.Status != domain.MarkStatusQuoted {
		return nil, domain.ErrMarkNotDraft
	}
	if mark.DispatchBatchID == nil || strings.TrimSpace(*mark.DispatchBatchID) != batch.ID {
		return nil, domain.ErrNotFound
	}

	mark.Observations = domain.NormalizeCarrierSlug(command.Observations)
	mark.TrackingNumber = strings.TrimSpace(command.TrackingNumber)
	mark.CustomTrackingURL = normalizeOptionalURLPointer(command.CustomTrackingURL)
	mark.QuotedFreightCost = command.QuotedFreightCost
	mark.UpdatedAt = time.Now().UTC()
	normalized := mark.Normalize()
	if err := normalized.Validate(); err != nil {
		return nil, err
	}
	if err := s.markRepository.Update(ctx, &normalized); err != nil {
		return nil, err
	}

	return &normalized, nil
}

// ValidateManualDraftsBeforeClose verifies manual drafts have all operator-supplied data required before batch close.
func (s *Service) ValidateManualDraftsBeforeClose(marks []domain.ShippingMark) error {
	for _, mark := range marks {
		if mark.Status != domain.MarkStatusQuoted {
			continue
		}
		if strings.TrimSpace(mark.Observations) == "" {
			return domain.ErrManualDraftIncomplete
		}
		if strings.TrimSpace(mark.TrackingNumber) == "" {
			return domain.ErrManualDraftIncomplete
		}
		if mark.QuotedFreightCost <= 0 {
			return domain.ErrManualDraftIncomplete
		}
	}

	return nil
}
