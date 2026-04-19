package service

import (
	"context"
	"strings"
	"time"

	markservice "mannaiah/module/shipping/application/mark/service"
	"mannaiah/module/shipping/domain"
)

// batchChecklistMeta defines batch metadata rendered in checklist headers.
type batchChecklistMeta struct {
	// BatchID defines batch identifier values.
	BatchID string
	// CarrierID defines batch carrier identifier values.
	CarrierID string
	// GeneratedAt defines generation timestamps.
	GeneratedAt time.Time
	// Quantity defines mark-count values included in this document.
	Quantity int
}

// batchChecklistRow defines one row rendered in checklist tables.
type batchChecklistRow struct {
	// OrderNumber defines short/public order-number values.
	OrderNumber string
	// RecipientName defines recipient display-name values.
	RecipientName string
	// City defines destination-city values.
	City string
	// Items defines row item-list values.
	Items []string
}

// ChecklistDocument builds one checklist PDF document for an open batch.
func (s *Service) ChecklistDocument(ctx context.Context, batchID string) ([]byte, error) {
	if s == nil || s.batchRepository == nil || s.markRepository == nil {
		return nil, domain.ErrInvalidID
	}
	trimmedBatchID := strings.TrimSpace(batchID)
	if trimmedBatchID == "" {
		return nil, domain.ErrInvalidID
	}

	batch, err := s.batchRepository.GetByID(ctx, trimmedBatchID)
	if err != nil {
		return nil, err
	}
	if batch.Status != domain.BatchStatusOpen {
		return nil, domain.ErrInvalidBatchStatus
	}

	normalizedCarrierID := strings.ToLower(strings.TrimSpace(batch.CarrierID))
	if normalizedCarrierID != "tcc" && !domain.IsManualCarrierID(batch.CarrierID) {
		return nil, domain.ErrCarrierNotSupported
	}

	marks, err := s.markRepository.ListByBatchID(ctx, trimmedBatchID)
	if err != nil {
		return nil, err
	}
	rows := s.resolveBatchChecklistRows(ctx, marks)

	return s.buildBatchChecklistPDF(ctx, batchChecklistMeta{
		BatchID:     batch.ID,
		CarrierID:   batch.CarrierID,
		GeneratedAt: time.Now().UTC(),
		Quantity:    len(rows),
	}, rows)
}

// resolveBatchChecklistRows resolves checklist rows from batch marks.
func (s *Service) resolveBatchChecklistRows(ctx context.Context, marks []domain.ShippingMark) []batchChecklistRow {
	rows := make([]batchChecklistRow, 0, len(marks))
	for _, mark := range marks {
		if !isBatchChecklistMarkIncluded(mark) {
			continue
		}
		orderNumber, items := s.resolveBatchManifestOrderSummary(ctx, mark)
		rows = append(rows, batchChecklistRow{
			OrderNumber:   orderNumber,
			RecipientName: firstNonEmpty(strings.TrimSpace(mark.Recipient.Name), strings.TrimSpace(mark.Recipient.LegalName), "-"),
			City:          firstNonEmpty(markservice.ResolveShippingCityDisplayName(mark.Recipient.CityCode), "-"),
			Items:         items,
		})
	}

	return rows
}

// isBatchChecklistMarkIncluded reports whether one mark should be rendered in checklist documents.
func isBatchChecklistMarkIncluded(mark domain.ShippingMark) bool {
	return mark.Status != domain.MarkStatusFailed && mark.Status != domain.MarkStatusRemoved
}
