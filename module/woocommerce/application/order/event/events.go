package event

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"mannaiah/module/woocommerce/port"
)

const (
	// TopicOrdersSyncStarted defines sync-started integration event topics.
	TopicOrdersSyncStarted = "woocommerce.v1.orders.sync.started"
	// TopicOrdersSyncCompleted defines sync-completed integration event topics.
	TopicOrdersSyncCompleted = "woocommerce.v1.orders.sync.completed"
	// TopicOrdersSyncFailed defines sync-failed integration event topics.
	TopicOrdersSyncFailed = "woocommerce.v1.orders.sync.failed"
	// schemaVersionV1 defines current integration event schema versions.
	schemaVersionV1 = "v1"
)

// Summary defines order sync counters used by integration event payloads.
type Summary struct {
	// Trigger defines sync trigger names.
	Trigger string
	// Processed defines upsert-attempt counts.
	Processed int
	// Created defines created order counts.
	Created int
	// Updated defines updated order counts.
	Updated int
	// Unchanged defines no-op order counts.
	Unchanged int
	// Skipped defines skipped-order counts.
	Skipped int
	// Failed defines failed upsert counts.
	Failed int
}

// OrdersSyncEventPayload defines sync event payload values.
type OrdersSyncEventPayload struct {
	// Trigger defines sync trigger values.
	Trigger string `json:"trigger"`
	// Processed defines processed order counts.
	Processed int `json:"processed"`
	// Created defines created order counts.
	Created int `json:"created"`
	// Updated defines updated order counts.
	Updated int `json:"updated"`
	// Unchanged defines unchanged order counts.
	Unchanged int `json:"unchanged"`
	// Skipped defines skipped row counts.
	Skipped int `json:"skipped"`
	// Failed defines failed row counts.
	Failed int `json:"failed"`
	// Error defines failure descriptions when present.
	Error string `json:"error,omitempty"`
}

// noopIntegrationEventPublisher defines no-op integration event publication behavior.
type noopIntegrationEventPublisher struct{}

// Publish ignores integration events.
func (noopIntegrationEventPublisher) Publish(ctx context.Context, integrationEvent port.IntegrationEvent) error {
	return nil
}

// ResolvePublisher resolves optional integration event publisher dependencies.
func ResolvePublisher(publisher port.IntegrationEventPublisher) port.IntegrationEventPublisher {
	if publisher != nil {
		return publisher
	}

	return noopIntegrationEventPublisher{}
}

// NewSyncStartedEvent maps sync-started payload values to integration event envelopes.
func NewSyncStartedEvent(trigger string) port.IntegrationEvent {
	return buildSyncEvent(TopicOrdersSyncStarted, OrdersSyncEventPayload{
		Trigger: strings.TrimSpace(trigger),
	})
}

// NewSyncCompletedEvent maps sync-summary values to completed integration event envelopes.
func NewSyncCompletedEvent(summary Summary) port.IntegrationEvent {
	return buildSyncEvent(TopicOrdersSyncCompleted, OrdersSyncEventPayload{
		Trigger:   summary.Trigger,
		Processed: summary.Processed,
		Created:   summary.Created,
		Updated:   summary.Updated,
		Unchanged: summary.Unchanged,
		Skipped:   summary.Skipped,
		Failed:    summary.Failed,
	})
}

// NewSyncFailedEvent maps sync-summary values to failed integration event envelopes.
func NewSyncFailedEvent(summary Summary, syncErr error) port.IntegrationEvent {
	payload := OrdersSyncEventPayload{
		Trigger:   summary.Trigger,
		Processed: summary.Processed,
		Created:   summary.Created,
		Updated:   summary.Updated,
		Unchanged: summary.Unchanged,
		Skipped:   summary.Skipped,
		Failed:    summary.Failed,
	}
	if syncErr != nil {
		payload.Error = syncErr.Error()
	}

	return buildSyncEvent(TopicOrdersSyncFailed, payload)
}

// buildSyncEvent creates a sync integration event envelope from topic and payload values.
func buildSyncEvent(topic string, payload OrdersSyncEventPayload) port.IntegrationEvent {
	return port.IntegrationEvent{
		ID:            generateEventID(),
		Topic:         topic,
		SchemaVersion: schemaVersionV1,
		OccurredAt:    time.Now().UTC(),
		Payload:       payload,
	}
}

// generateEventID creates random integration event identifiers.
func generateEventID() string {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return fmt.Sprintf("event-%d", time.Now().UnixNano())
	}

	return strings.TrimSpace(hex.EncodeToString(bytes))
}
