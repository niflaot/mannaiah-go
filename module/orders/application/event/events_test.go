package event

import (
	"context"
	"testing"
	"time"

	ordersdomain "mannaiah/module/orders/domain"
	ordersport "mannaiah/module/orders/port"
)

// publisherProbe defines integration publication behavior for event tests.
type publisherProbe struct{}

// Publish accepts integration events.
func (publisherProbe) Publish(ctx context.Context, integrationEvent ordersport.IntegrationEvent) error {
	return nil
}

// TestResolveSource verifies source normalization behavior.
func TestResolveSource(t *testing.T) {
	if value := ResolveSource("  "); value != ordersport.EventSourceAPI {
		t.Fatalf("ResolveSource(blank) = %q, want %q", value, ordersport.EventSourceAPI)
	}
	if value := ResolveSource("mainstream"); value != "mainstream" {
		t.Fatalf("ResolveSource(non-blank) = %q, want %q", value, "mainstream")
	}
}

// TestBuildEvents verifies order integration event mapping behavior.
func TestBuildEvents(t *testing.T) {
	updatedAt := time.Date(2026, time.February, 15, 12, 0, 0, 0, time.UTC)
	entity := ordersdomain.Order{
		ID:            "o-1",
		Identifier:    "1001",
		Realm:         "woocommerce",
		ContactID:     "c-1",
		CurrentStatus: ordersdomain.StatusCreated,
		StatusHistory: []ordersdomain.StatusEntry{
			{
				Status:      ordersdomain.StatusCreated,
				Author:      "system",
				Description: "created",
				NoteOwner:   "system",
				Note:        "note",
				OccurredAt:  updatedAt,
			},
		},
		Items: []ordersdomain.Item{
			{
				SKU:              "SKU-1",
				AlternateName:    "fallback",
				Quantity:         2,
				Value:            10,
				ProductID:        "p-1",
				ResolutionSource: ordersdomain.ItemResolutionSourceSKU,
			},
		},
		ShippingAddress: ordersdomain.ShippingAddress{
			Address:  "A",
			Address2: "B",
			Phone:    "300",
			CityCode: "11001",
		},
		HasCustomShippingAddress: true,
		ShippingCharges: []ordersdomain.ShippingCharge{
			{MethodID: "flat_rate", MethodTitle: "Flat", Price: 9},
		},
		Metadata:  map[string]string{"x": "y"},
		CreatedAt: updatedAt,
		UpdatedAt: updatedAt,
	}

	created := BuildOrderCreatedIntegrationEvent(entity, "mainstream")
	updated := BuildOrderUpdatedIntegrationEvent(entity, "mainstream")
	statusUpdated := BuildOrderStatusUpdatedIntegrationEvent(entity, "mainstream")
	for _, integrationEvent := range []ordersport.IntegrationEvent{created, updated, statusUpdated} {
		if integrationEvent.ID == "" {
			t.Fatalf("event ID should not be empty")
		}
		if integrationEvent.SchemaVersion != "v1" {
			t.Fatalf("schemaVersion = %q, want %q", integrationEvent.SchemaVersion, "v1")
		}
		payload, ok := integrationEvent.Payload.(ordersport.OrderEventPayload)
		if !ok {
			t.Fatalf("payload type = %T, want ordersport.OrderEventPayload", integrationEvent.Payload)
		}
		if payload.Source != "mainstream" {
			t.Fatalf("payload.Source = %q, want %q", payload.Source, "mainstream")
		}
		if payload.Identifier != "1001" {
			t.Fatalf("payload.Identifier = %q, want %q", payload.Identifier, "1001")
		}
		if payload.LatestStatus.Status != "CREATED" {
			t.Fatalf("payload.LatestStatus.Status = %q, want %q", payload.LatestStatus.Status, "CREATED")
		}
	}
}

// TestResolvePublisher verifies optional publisher resolution behavior.
func TestResolvePublisher(t *testing.T) {
	if value := ResolvePublisher(nil); value == nil {
		t.Fatalf("ResolvePublisher(nil) should not return nil")
	}

	publisher := publisherProbe{}
	if value := ResolvePublisher(publisher); value != publisher {
		t.Fatalf("ResolvePublisher(publisher) = %v, want same instance", value)
	}
}

