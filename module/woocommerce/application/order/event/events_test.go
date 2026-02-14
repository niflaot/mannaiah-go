package event

import (
	"context"
	"testing"

	"mannaiah/module/woocommerce/port"
)

// TestResolvePublisher verifies optional publisher resolution behavior.
func TestResolvePublisher(t *testing.T) {
	if value := ResolvePublisher(nil); value == nil {
		t.Fatalf("expected noop publisher")
	}

	publisher := &publisherProbe{}
	if value := ResolvePublisher(publisher); value != publisher {
		t.Fatalf("expected provided publisher to be preserved")
	}
}

// publisherProbe defines integration event publication behavior for event tests.
type publisherProbe struct{}

// Publish emits integration events.
func (publisherProbe) Publish(ctx context.Context, integrationEvent port.IntegrationEvent) error {
	return nil
}

// TestBuildSyncEvents verifies integration event factory behavior.
func TestBuildSyncEvents(t *testing.T) {
	started := NewSyncStartedEvent("manual")
	if started.Topic != TopicOrdersSyncStarted {
		t.Fatalf("started.Topic = %q, want %q", started.Topic, TopicOrdersSyncStarted)
	}
	if started.SchemaVersion != schemaVersionV1 {
		t.Fatalf("started.SchemaVersion = %q, want %q", started.SchemaVersion, schemaVersionV1)
	}

	summary := Summary{
		Trigger:   "cron",
		Processed: 10,
		Created:   4,
		Updated:   5,
		Unchanged: 1,
		Skipped:   1,
		Failed:    0,
	}
	completed := NewSyncCompletedEvent(summary)
	if completed.Topic != TopicOrdersSyncCompleted {
		t.Fatalf("completed.Topic = %q, want %q", completed.Topic, TopicOrdersSyncCompleted)
	}

	failed := NewSyncFailedEvent(summary, context.Canceled)
	if failed.Topic != TopicOrdersSyncFailed {
		t.Fatalf("failed.Topic = %q, want %q", failed.Topic, TopicOrdersSyncFailed)
	}
}

// TestGenerateEventID verifies event identifier generation behavior.
func TestGenerateEventID(t *testing.T) {
	idA := generateEventID()
	idB := generateEventID()
	if idA == "" || idB == "" {
		t.Fatalf("expected non-empty event ids")
	}
	if idA == idB {
		t.Fatalf("expected unique event ids")
	}
}
