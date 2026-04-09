package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"mannaiah/module/shipping/domain"
	"mannaiah/module/shipping/port"
)

const trackingStatusManual = "MANUAL"
const trackingListFetchChunkSize = 100

// ListQuery defines tracking listing query values.
type ListQuery struct {
	// Term filters rows by tracking number, order id, and recipient labels.
	Term string
	// Status filters rows by last known tracking status.
	Status string
	// Page defines 1-based page values.
	Page int
	// Limit defines page-size values.
	Limit int
}

// ListItem defines one tracking-summary row.
type ListItem struct {
	// ID defines shipping mark identifier values.
	ID string `json:"id"`
	// OrderID defines related order identifier values.
	OrderID string `json:"orderId,omitempty"`
	// TrackingNumber defines carrier tracking-number values.
	TrackingNumber string `json:"trackingNumber"`
	// RecipientName defines destination recipient label values.
	RecipientName string `json:"recipientName"`
	// CarrierID defines carrier identifier values.
	CarrierID string `json:"carrierId"`
	// LastStatus defines last known tracking status values.
	LastStatus string `json:"lastStatus"`
	// CreatedAt defines mark creation timestamps.
	CreatedAt time.Time `json:"createdAt"`
}

// List resolves paginated tracking summaries for non-draft marks.
func (s *Service) List(ctx context.Context, query ListQuery) ([]ListItem, int64, error) {
	page, limit := normalizeListPagination(query.Page, query.Limit)
	normalizedStatus := strings.ToUpper(strings.TrimSpace(query.Status))
	trimmedTerm := strings.TrimSpace(query.Term)
	if normalizedStatus == "" {
		rows, total, err := s.listCandidateMarks(ctx, port.MarkListQuery{
			SearchTerm:       trimmedTerm,
			ExcludedStatuses: []domain.MarkStatus{domain.MarkStatusQuoted, domain.MarkStatusRemoved},
			RequireTracking:  true,
			Page:             page,
			Limit:            limit,
		})
		if err != nil {
			return nil, 0, err
		}

		items, err := s.buildListItems(ctx, rows)
		if err != nil {
			return nil, 0, err
		}

		return items, total, nil
	}

	rows, err := s.collectAllCandidateMarks(ctx, trimmedTerm)
	if err != nil {
		return nil, 0, err
	}
	items, err := s.buildListItems(ctx, rows)
	if err != nil {
		return nil, 0, err
	}
	filtered := make([]ListItem, 0, len(items))
	for _, item := range items {
		if strings.EqualFold(strings.TrimSpace(item.LastStatus), normalizedStatus) {
			filtered = append(filtered, item)
		}
	}
	paginated, total := paginateListItems(filtered, page, limit)

	return paginated, total, nil
}

// listCandidateMarks resolves one paginated chunk of trackable marks.
func (s *Service) listCandidateMarks(ctx context.Context, query port.MarkListQuery) ([]domain.ShippingMark, int64, error) {
	if s == nil || s.repository == nil {
		return []domain.ShippingMark{}, 0, nil
	}

	rows, total, err := s.repository.List(ctx, query)
	if err != nil {
		return nil, 0, err
	}

	return rows, total, nil
}

// collectAllCandidateMarks resolves all trackable marks required for in-memory status filtering.
func (s *Service) collectAllCandidateMarks(ctx context.Context, term string) ([]domain.ShippingMark, error) {
	rows := make([]domain.ShippingMark, 0)
	page := 1
	for {
		chunk, total, err := s.listCandidateMarks(ctx, port.MarkListQuery{
			SearchTerm:       term,
			ExcludedStatuses: []domain.MarkStatus{domain.MarkStatusQuoted, domain.MarkStatusRemoved},
			RequireTracking:  true,
			Page:             page,
			Limit:            trackingListFetchChunkSize,
		})
		if err != nil {
			return nil, err
		}
		rows = append(rows, chunk...)
		if int64(len(rows)) >= total || len(chunk) == 0 {
			break
		}
		page += 1
	}

	return rows, nil
}

// buildListItems builds tracking list rows for marks.
func (s *Service) buildListItems(ctx context.Context, marks []domain.ShippingMark) ([]ListItem, error) {
	items := make([]ListItem, 0, len(marks))
	for _, mark := range marks {
		item, err := s.buildListItem(ctx, mark)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}

	return items, nil
}

// buildListItem builds one tracking summary from one mark.
func (s *Service) buildListItem(ctx context.Context, mark domain.ShippingMark) (ListItem, error) {
	carrierID, lastStatus, err := s.resolveListStatus(ctx, mark)
	if err != nil {
		return ListItem{}, err
	}

	return ListItem{
		ID:             strings.TrimSpace(mark.ID),
		OrderID:        strings.TrimSpace(mark.OrderID),
		TrackingNumber: strings.TrimSpace(mark.TrackingNumber),
		RecipientName:  resolveRecipientName(mark.Recipient),
		CarrierID:      carrierID,
		LastStatus:     lastStatus,
		CreatedAt:      mark.CreatedAt,
	}, nil
}

// resolveListStatus resolves the carrier identifier and last status shown in tracking listings.
func (s *Service) resolveListStatus(ctx context.Context, mark domain.ShippingMark) (string, string, error) {
	manualCarrierID := resolveManualCarrierID(mark)
	if domain.IsManualCarrierID(mark.CarrierID) {
		return manualCarrierID, trackingStatusManual, nil
	}
	history, err := s.lookupTrackingHistory(ctx, strings.TrimSpace(mark.CarrierID), strings.TrimSpace(mark.TrackingNumber))
	if err != nil {
		return manualCarrierID, fallbackTrackingStatus(mark.Status), nil
	}
	if history == nil {
		return manualCarrierID, fallbackTrackingStatus(mark.Status), nil
	}

	return strings.TrimSpace(history.CarrierID), strings.TrimSpace(string(history.GlobalStatus)), nil
}

// lookupTrackingHistory resolves tracking history without publishing tracking-updated events.
func (s *Service) lookupTrackingHistory(ctx context.Context, carrierID string, trackingNumber string) (*domain.TrackingHistory, error) {
	if s == nil || s.registry == nil {
		return nil, domain.ErrTrackingNotSupported
	}
	provider, exists := s.registry.TrackingProvider(strings.TrimSpace(carrierID))
	if !exists || provider == nil {
		return nil, domain.ErrTrackingNotSupported
	}
	history, err := provider.GetTrackingHistory(ctx, strings.TrimSpace(trackingNumber))
	if err != nil {
		return nil, fmt.Errorf("get tracking history: %w", err)
	}

	return history, nil
}

// resolveManualCarrierID resolves the public carrier identifier used in tracking listings for manual marks.
func resolveManualCarrierID(mark domain.ShippingMark) string {
	carrierID := strings.TrimSpace(mark.CarrierID)
	if !domain.IsManualCarrierID(carrierID) {
		return carrierID
	}
	if slug := domain.NormalizeCarrierSlug(mark.Observations); slug != "" {
		return "manual_" + slug
	}

	return "manual"
}

// resolveRecipientName resolves one display-safe recipient label.
func resolveRecipientName(recipient domain.Address) string {
	name := strings.TrimSpace(recipient.Name)
	if name != "" {
		return name
	}

	return strings.TrimSpace(recipient.LegalName)
}

// fallbackTrackingStatus resolves a tracking-status fallback from the persisted mark status.
func fallbackTrackingStatus(status domain.MarkStatus) string {
	switch status {
	case domain.MarkStatusVoided:
		return string(domain.TrackingStatusVoided)
	default:
		return string(domain.TrackingStatusProcessing)
	}
}

// normalizeListPagination resolves safe pagination values.
func normalizeListPagination(page int, limit int) (int, int) {
	if page <= 0 {
		page = 1
	}
	if limit <= 0 {
		limit = 20
	}

	return page, limit
}

// paginateListItems slices tracking rows using page/limit values.
func paginateListItems(items []ListItem, page int, limit int) ([]ListItem, int64) {
	total := int64(len(items))
	if total == 0 {
		return []ListItem{}, 0
	}
	start := (page - 1) * limit
	if start >= len(items) {
		return []ListItem{}, total
	}
	end := start + limit
	if end > len(items) {
		end = len(items)
	}

	return items[start:end], total
}
