package domain

import (
	"testing"
	"time"
)

// TestNewContactCreatedEvent verifies created-event construction behavior.
func TestNewContactCreatedEvent(t *testing.T) {
	createdAt := time.Now().UTC().Add(-time.Minute)
	event := NewContactCreatedEvent(Contact{ID: "c-1", CreatedAt: createdAt})

	if event.Name() != EventTypeContactCreated {
		t.Fatalf("Name() = %q, want %q", event.Name(), EventTypeContactCreated)
	}
	if event.AggregateID() != "c-1" {
		t.Fatalf("AggregateID() = %q, want %q", event.AggregateID(), "c-1")
	}
	if !event.OccurredAt().Equal(createdAt) {
		t.Fatalf("OccurredAt() = %v, want %v", event.OccurredAt(), createdAt)
	}
}

// TestNewContactCreatedEventFallbackTime verifies fallback timestamps for created events.
func TestNewContactCreatedEventFallbackTime(t *testing.T) {
	event := NewContactCreatedEvent(Contact{ID: "c-1"})
	if event.OccurredAt().IsZero() {
		t.Fatalf("expected fallback occurredAt")
	}
}

// TestNewContactUpdatedEvent verifies updated-event construction behavior.
func TestNewContactUpdatedEvent(t *testing.T) {
	updatedAt := time.Now().UTC().Add(-time.Minute)
	event := NewContactUpdatedEvent(Contact{ID: "c-2", UpdatedAt: updatedAt})

	if event.Name() != EventTypeContactUpdated {
		t.Fatalf("Name() = %q, want %q", event.Name(), EventTypeContactUpdated)
	}
	if event.AggregateID() != "c-2" {
		t.Fatalf("AggregateID() = %q, want %q", event.AggregateID(), "c-2")
	}
	if !event.OccurredAt().Equal(updatedAt) {
		t.Fatalf("OccurredAt() = %v, want %v", event.OccurredAt(), updatedAt)
	}
}

// TestNewContactUpdatedEventFallbackTime verifies fallback timestamps for updated events.
func TestNewContactUpdatedEventFallbackTime(t *testing.T) {
	event := NewContactUpdatedEvent(Contact{ID: "c-2"})
	if event.OccurredAt().IsZero() {
		t.Fatalf("expected fallback occurredAt")
	}
}
