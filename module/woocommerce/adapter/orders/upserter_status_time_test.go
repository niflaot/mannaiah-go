package orders

import (
	"testing"
	"time"

	ordersdomain "mannaiah/module/orders/domain"
)

// TestResolveStatusOccurredAt verifies status timestamp resolution behavior.
func TestResolveStatusOccurredAt(t *testing.T) {
	latest := time.Date(2026, time.February, 14, 10, 0, 0, 0, time.UTC)
	order := ordersdomain.Order{
		StatusHistory: []ordersdomain.StatusEntry{
			{Status: ordersdomain.StatusCreated, Author: "system", OccurredAt: latest},
		},
	}

	older := latest.Add(-time.Hour)
	resolved := resolveStatusOccurredAt(order, &older)
	if resolved == nil {
		t.Fatalf("resolveStatusOccurredAt() returned nil")
	}
	if !resolved.UTC().After(latest) {
		t.Fatalf("resolved = %v, want > latest %v", resolved.UTC(), latest)
	}

	newer := latest.Add(time.Hour)
	resolved = resolveStatusOccurredAt(order, &newer)
	if resolved == nil || !resolved.UTC().Equal(newer) {
		t.Fatalf("resolved = %v, want %v", resolved, newer)
	}
}

// TestLatestStatusOccurredAt verifies latest-status timestamp helper behavior.
func TestLatestStatusOccurredAt(t *testing.T) {
	if !latestStatusOccurredAt(nil).IsZero() {
		t.Fatalf("latestStatusOccurredAt(nil) should return zero time")
	}

	older := time.Date(2026, time.February, 14, 9, 0, 0, 0, time.UTC)
	latest := time.Date(2026, time.February, 14, 10, 0, 0, 0, time.UTC)
	value := latestStatusOccurredAt([]ordersdomain.StatusEntry{
		{Status: ordersdomain.StatusCreated, Author: "system", OccurredAt: older},
		{Status: ordersdomain.StatusCompleted, Author: "system", OccurredAt: latest},
	})
	if !value.Equal(latest) {
		t.Fatalf("latestStatusOccurredAt() = %v, want %v", value, latest)
	}
}
