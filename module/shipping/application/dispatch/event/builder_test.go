package event

import (
	"testing"

	"mannaiah/module/shipping/domain"
	"mannaiah/module/shipping/port"
)

// TestBuildBatchEvents verifies dispatch batch event topic mapping.
func TestBuildBatchEvents(t *testing.T) {
	batch := domain.DispatchBatch{ID: "batch-1", CarrierID: "manual", CreatedBy: "user-123", MarkIDs: []string{"m1", "m2"}}

	created := BuildBatchCreated(batch)
	if created.Topic != port.TopicBatchCreated {
		t.Fatalf("created topic = %q", created.Topic)
	}
	closed := BuildBatchClosed(batch)
	if closed.Topic != port.TopicBatchClosed {
		t.Fatalf("closed topic = %q", closed.Topic)
	}
}
