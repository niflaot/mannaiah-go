package event

import (
	"testing"

	"mannaiah/module/shipping/domain"
	"mannaiah/module/shipping/port"
)

// TestBuildMarkEvents verifies mark event topic mapping.
func TestBuildMarkEvents(t *testing.T) {
	mark := domain.ShippingMark{ID: "mark-1", OrderID: "order-1", CarrierID: "manual", Observations: "Inter rapidísimo", TrackingNumber: "TRACK-1"}

	generated := BuildMarkGenerated(mark)
	if generated.Topic != port.TopicMarkGenerated {
		t.Fatalf("generated topic = %q", generated.Topic)
	}
	generatedPayload, ok := generated.Payload.(MarkGeneratedPayload)
	if !ok {
		t.Fatalf("generated.Payload type = %T, want MarkGeneratedPayload", generated.Payload)
	}
	if generatedPayload.TrackingCompany != "interrapidisimo" {
		t.Fatalf("generated tracking company = %q, want %q", generatedPayload.TrackingCompany, "interrapidisimo")
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

// TestBuildMarkGeneratedUsesCarrierIDForStructuredCarriers verifies non-manual carriers keep their carrier id.
func TestBuildMarkGeneratedUsesCarrierIDForStructuredCarriers(t *testing.T) {
	mark := domain.ShippingMark{ID: "mark-1", OrderID: "order-1", CarrierID: "tcc", TrackingNumber: "TRACK-1"}

	generated := BuildMarkGenerated(mark)
	payload, ok := generated.Payload.(MarkGeneratedPayload)
	if !ok {
		t.Fatalf("generated.Payload type = %T, want MarkGeneratedPayload", generated.Payload)
	}
	if payload.TrackingCompany != "tcc" {
		t.Fatalf("generated tracking company = %q, want %q", payload.TrackingCompany, "tcc")
	}
}
