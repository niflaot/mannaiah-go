package contact

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
	// TopicContactsSyncStarted defines sync-started integration event topics.
	TopicContactsSyncStarted = "woocommerce.v1.contacts.sync.started"
	// TopicContactsSyncCompleted defines sync-completed integration event topics.
	TopicContactsSyncCompleted = "woocommerce.v1.contacts.sync.completed"
	// TopicContactsSyncFailed defines sync-failed integration event topics.
	TopicContactsSyncFailed = "woocommerce.v1.contacts.sync.failed"
	// schemaVersionV1 defines current integration event schema versions.
	schemaVersionV1 = "v1"
)

// ContactsSyncEventPayload defines sync event payload values.
type ContactsSyncEventPayload struct {
	// Trigger defines sync trigger values.
	Trigger string `json:"trigger"`
	// Processed defines processed order counts.
	Processed int `json:"processed"`
	// Created defines created contact counts.
	Created int `json:"created"`
	// Updated defines updated contact counts.
	Updated int `json:"updated"`
	// Unchanged defines unchanged contact counts.
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
func (noopIntegrationEventPublisher) Publish(ctx context.Context, event port.IntegrationEvent) error {
	return nil
}

// resolvePublisher resolves optional integration event publisher dependencies.
func resolvePublisher(publisher port.IntegrationEventPublisher) port.IntegrationEventPublisher {
	if publisher != nil {
		return publisher
	}

	return noopIntegrationEventPublisher{}
}

// publishEvent publishes sync integration events and ignores publication failures.
func (s *ContactSyncService) publishEvent(ctx context.Context, event port.IntegrationEvent) {
	_ = s.publisher.Publish(ctx, event)
}

// buildSyncStartedEvent maps sync-started payload values to integration event envelopes.
func buildSyncStartedEvent(trigger string) port.IntegrationEvent {
	return buildSyncEvent(TopicContactsSyncStarted, ContactsSyncEventPayload{
		Trigger: strings.TrimSpace(trigger),
	})
}

// buildSyncCompletedEvent maps sync-summary values to completed integration event envelopes.
func buildSyncCompletedEvent(summary SyncSummary) port.IntegrationEvent {
	return buildSyncEvent(TopicContactsSyncCompleted, ContactsSyncEventPayload{
		Trigger:   summary.Trigger,
		Processed: summary.Processed,
		Created:   summary.Created,
		Updated:   summary.Updated,
		Unchanged: summary.Unchanged,
		Skipped:   summary.Skipped,
		Failed:    summary.Failed,
	})
}

// buildSyncFailedEvent maps sync-summary values to failed integration event envelopes.
func buildSyncFailedEvent(summary SyncSummary, syncErr error) port.IntegrationEvent {
	payload := ContactsSyncEventPayload{
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

	return buildSyncEvent(TopicContactsSyncFailed, payload)
}

// buildSyncEvent creates a sync integration event envelope from topic and payload values.
func buildSyncEvent(topic string, payload ContactsSyncEventPayload) port.IntegrationEvent {
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
