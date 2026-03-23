package domain

import (
	"strings"
	"time"
)

// BatchStatus defines dispatch-batch status values.
type BatchStatus string

const (
	// BatchStatusOpen defines open batch statuses.
	BatchStatusOpen BatchStatus = "OPEN"
	// BatchStatusClosed defines closed batch statuses.
	BatchStatusClosed BatchStatus = "CLOSED"
)

// DispatchBatch defines one dispatch grouping for generated marks.
type DispatchBatch struct {
	// ID defines batch identifier values.
	ID string `json:"id"`
	// Name defines display-name values.
	Name string `json:"name"`
	// CarrierID defines batch carrier identifier values.
	CarrierID string `json:"carrierId"`
	// Status defines batch status values.
	Status BatchStatus `json:"status"`
	// MarkIDs defines assigned mark identifier values.
	MarkIDs []string `json:"markIds,omitempty"`
	// CreatedAt defines row creation timestamps.
	CreatedAt time.Time `json:"createdAt"`
	// ClosedAt defines batch closed timestamps.
	ClosedAt *time.Time `json:"closedAt,omitempty"`
}

// Normalize normalizes dispatch-batch fields.
func (b DispatchBatch) Normalize() DispatchBatch {
	markIDs := make([]string, 0, len(b.MarkIDs))
	seen := map[string]struct{}{}
	for _, markID := range b.MarkIDs {
		trimmed := strings.TrimSpace(markID)
		if trimmed == "" {
			continue
		}
		if _, exists := seen[trimmed]; exists {
			continue
		}
		seen[trimmed] = struct{}{}
		markIDs = append(markIDs, trimmed)
	}
	copy := DispatchBatch{
		ID:        strings.TrimSpace(b.ID),
		Name:      strings.TrimSpace(b.Name),
		CarrierID: strings.TrimSpace(b.CarrierID),
		Status:    b.Status,
		MarkIDs:   markIDs,
		CreatedAt: b.CreatedAt,
		ClosedAt:  b.ClosedAt,
	}
	if copy.Status == "" {
		copy.Status = BatchStatusOpen
	}

	return copy
}

// Validate validates dispatch-batch fields.
func (b DispatchBatch) Validate() error {
	normalized := b.Normalize()
	if normalized.ID == "" {
		return ErrInvalidID
	}
	if normalized.Name == "" {
		return ErrInvalidID
	}
	if normalized.CarrierID == "" {
		return ErrInvalidCarrierID
	}

	return nil
}
