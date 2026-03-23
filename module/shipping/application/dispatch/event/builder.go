package event

import (
	"time"

	"github.com/google/uuid"

	"mannaiah/module/shipping/domain"
	"mannaiah/module/shipping/port"
)

const (
	// schemaVersion defines shipping event schema-version values.
	schemaVersion = "v1"
)

// BatchCreatedPayload defines batch-created event payload values.
type BatchCreatedPayload struct {
	// BatchID defines batch identifier values.
	BatchID string `json:"batchId"`
	// CarrierID defines carrier identifier values.
	CarrierID string `json:"carrierId"`
	// Name defines batch name values.
	Name string `json:"name"`
}

// BatchClosedPayload defines batch-closed event payload values.
type BatchClosedPayload struct {
	// BatchID defines batch identifier values.
	BatchID string `json:"batchId"`
	// CarrierID defines carrier identifier values.
	CarrierID string `json:"carrierId"`
	// MarkCount defines number of marks within the batch.
	MarkCount int `json:"markCount"`
}

// BuildBatchCreated builds one batch-created integration event.
func BuildBatchCreated(batch domain.DispatchBatch) port.IntegrationEvent {
	return port.IntegrationEvent{
		ID:            uuid.NewString(),
		Topic:         port.TopicBatchCreated,
		SchemaVersion: schemaVersion,
		OccurredAt:    time.Now().UTC(),
		Payload: BatchCreatedPayload{
			BatchID:   batch.ID,
			CarrierID: batch.CarrierID,
			Name:      batch.Name,
		},
	}
}

// BuildBatchClosed builds one batch-closed integration event.
func BuildBatchClosed(batch domain.DispatchBatch) port.IntegrationEvent {
	return port.IntegrationEvent{
		ID:            uuid.NewString(),
		Topic:         port.TopicBatchClosed,
		SchemaVersion: schemaVersion,
		OccurredAt:    time.Now().UTC(),
		Payload: BatchClosedPayload{
			BatchID:   batch.ID,
			CarrierID: batch.CarrierID,
			MarkCount: len(batch.MarkIDs),
		},
	}
}
