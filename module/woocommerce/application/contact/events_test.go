package contact

import (
	"context"
	"testing"

	"mannaiah/module/woocommerce/port"
)

// TestResolvePublisher verifies optional publisher resolution behavior.
func TestResolvePublisher(t *testing.T) {
	if value := resolvePublisher(nil); value == nil {
		t.Fatalf("expected noop publisher")
	}

	publisher := &publisherProbe{}
	if value := resolvePublisher(publisher); value != publisher {
		t.Fatalf("expected provided publisher to be preserved")
	}
}

// publisherProbe defines integration event publication behavior for event tests.
type publisherProbe struct{}

// Publish emits integration events.
func (publisherProbe) Publish(ctx context.Context, event port.IntegrationEvent) error {
	return nil
}

// TestBuildSyncEvents verifies integration event factory behavior.
func TestBuildSyncEvents(t *testing.T) {
	started := buildSyncStartedEvent("manual")
	if started.Topic != TopicContactsSyncStarted {
		t.Fatalf("started.Topic = %q, want %q", started.Topic, TopicContactsSyncStarted)
	}
	if started.SchemaVersion != schemaVersionV1 {
		t.Fatalf("started.SchemaVersion = %q, want %q", started.SchemaVersion, schemaVersionV1)
	}

	summary := SyncSummary{
		Trigger:   "cron",
		Processed: 10,
		Created:   4,
		Updated:   5,
		Unchanged: 1,
		Skipped:   1,
		Failed:    0,
	}
	completed := buildSyncCompletedEvent(summary)
	if completed.Topic != TopicContactsSyncCompleted {
		t.Fatalf("completed.Topic = %q, want %q", completed.Topic, TopicContactsSyncCompleted)
	}

	failed := buildSyncFailedEvent(summary, context.Canceled)
	if failed.Topic != TopicContactsSyncFailed {
		t.Fatalf("failed.Topic = %q, want %q", failed.Topic, TopicContactsSyncFailed)
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
