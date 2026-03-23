package event

import (
	"testing"

	"mannaiah/module/shipping/domain"
	"mannaiah/module/shipping/port"
)

// TestBuildMarkEvents verifies mark event topic mapping.
func TestBuildMarkEvents(t *testing.T) {
	mark := domain.ShippingMark{ID: "mark-1", OrderID: "order-1", CarrierID: "manual", TrackingNumber: "TRACK-1"}

	generated := BuildMarkGenerated(mark)
	if generated.Topic != port.TopicMarkGenerated {
		t.Fatalf("generated topic = %q", generated.Topic)
	}
	failed := BuildMarkFailed(mark, "boom")
	if failed.Topic != port.TopicMarkFailed {
		t.Fatalf("failed topic = %q", failed.Topic)
	}
	voided := BuildMarkVoided(mark, "void")
	if voided.Topic != port.TopicMarkVoided {
		t.Fatalf("voided topic = %q", voided.Topic)
	}
}
