package event

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	ordersdomain "mannaiah/module/orders/domain"
	ordersport "mannaiah/module/orders/port"
)

const (
	// schemaVersionV1 defines current order integration event schema versions.
	schemaVersionV1 = "v1"
)

// ResolveSource resolves blank source values to default API source values.
func ResolveSource(source string) string {
	trimmed := strings.TrimSpace(source)
	if trimmed != "" {
		return trimmed
	}

	return ordersport.EventSourceAPI
}

// BuildOrderCreatedIntegrationEvent maps order-created values to integration event envelopes.
func BuildOrderCreatedIntegrationEvent(entity ordersdomain.Order, source string) ordersport.IntegrationEvent {
	return buildIntegrationEvent(ordersport.TopicOrderCreated, entity, source)
}

// BuildOrderUpdatedIntegrationEvent maps order-updated values to integration event envelopes.
func BuildOrderUpdatedIntegrationEvent(entity ordersdomain.Order, source string) ordersport.IntegrationEvent {
	return buildIntegrationEvent(ordersport.TopicOrderUpdated, entity, source)
}

// BuildOrderStatusUpdatedIntegrationEvent maps order-status-updated values to integration event envelopes.
func BuildOrderStatusUpdatedIntegrationEvent(entity ordersdomain.Order, source string) ordersport.IntegrationEvent {
	return buildIntegrationEvent(ordersport.TopicOrderStatusUpdated, entity, source)
}

// buildIntegrationEvent creates integration event envelopes from order values.
func buildIntegrationEvent(topic string, entity ordersdomain.Order, source string) ordersport.IntegrationEvent {
	payload := toOrderEventPayload(entity, source)
	occurredAt := payload.UpdatedAt.UTC()
	if occurredAt.IsZero() {
		occurredAt = time.Now().UTC()
	}

	return ordersport.IntegrationEvent{
		ID:            generateEventID(),
		Topic:         strings.TrimSpace(topic),
		SchemaVersion: schemaVersionV1,
		OccurredAt:    occurredAt,
		Payload:       payload,
		Metadata: map[string]string{
			"aggregate_id": payload.ID,
			"source":       payload.Source,
		},
	}
}

// toOrderEventPayload maps order aggregate values to integration event payload values.
func toOrderEventPayload(entity ordersdomain.Order, source string) ordersport.OrderEventPayload {
	items := make([]ordersport.OrderEventItem, 0, len(entity.Items))
	for _, row := range entity.Items {
		items = append(items, ordersport.OrderEventItem{
			SKU:              strings.TrimSpace(row.SKU),
			AlternateName:    strings.TrimSpace(row.AlternateName),
			Quantity:         row.Quantity,
			Value:            row.Value,
			ProductID:        strings.TrimSpace(row.ProductID),
			ResolutionSource: strings.TrimSpace(string(row.ResolutionSource)),
		})
	}

	shippingCharges := make([]ordersport.OrderEventShippingCharge, 0, len(entity.ShippingCharges))
	for _, row := range entity.ShippingCharges {
		shippingCharges = append(shippingCharges, ordersport.OrderEventShippingCharge{
			MethodID:    strings.TrimSpace(row.MethodID),
			MethodTitle: strings.TrimSpace(row.MethodTitle),
			Price:       row.Price,
		})
	}

	status := ordersport.OrderEventStatus{}
	if len(entity.StatusHistory) > 0 {
		latest := entity.StatusHistory[len(entity.StatusHistory)-1]
		status = ordersport.OrderEventStatus{
			Status:      strings.TrimSpace(string(latest.Status)),
			Author:      strings.TrimSpace(latest.Author),
			Description: strings.TrimSpace(latest.Description),
			NoteOwner:   strings.TrimSpace(latest.NoteOwner),
			Note:        strings.TrimSpace(latest.Note),
			OccurredAt:  latest.OccurredAt.UTC(),
		}
	}

	return ordersport.OrderEventPayload{
		ID:         strings.TrimSpace(entity.ID),
		Identifier: strings.TrimSpace(entity.Identifier),
		Realm:      strings.TrimSpace(entity.Realm),
		ContactID:  strings.TrimSpace(entity.ContactID),
		Source:     ResolveSource(source),
		CurrentStatus: strings.TrimSpace(string(
			entity.CurrentStatus,
		)),
		LatestStatus: status,
		Items:        items,
		ShippingAddress: ordersport.OrderEventShippingAddress{
			Address:  strings.TrimSpace(entity.ShippingAddress.Address),
			Address2: strings.TrimSpace(entity.ShippingAddress.Address2),
			Phone:    strings.TrimSpace(entity.ShippingAddress.Phone),
			CityCode: strings.TrimSpace(entity.ShippingAddress.CityCode),
		},
		HasCustomShippingAddress: entity.HasCustomShippingAddress,
		ShippingCharges:          shippingCharges,
		Metadata:                 cloneMetadata(entity.Metadata),
		CreatedAt:                entity.CreatedAt.UTC(),
		UpdatedAt:                entity.UpdatedAt.UTC(),
	}
}

// cloneMetadata creates shallow metadata copies.
func cloneMetadata(metadata map[string]string) map[string]string {
	if len(metadata) == 0 {
		return nil
	}

	result := make(map[string]string, len(metadata))
	for key, value := range metadata {
		trimmed := strings.TrimSpace(key)
		if trimmed == "" {
			continue
		}
		result[trimmed] = strings.TrimSpace(value)
	}
	if len(result) == 0 {
		return nil
	}

	return result
}

// noopIntegrationEventPublisher defines no-op integration event publication behavior.
type noopIntegrationEventPublisher struct{}

// Publish ignores integration events.
func (noopIntegrationEventPublisher) Publish(ctx context.Context, event ordersport.IntegrationEvent) error {
	return nil
}

// ResolvePublisher resolves optional integration event publisher dependencies.
func ResolvePublisher(publisher ordersport.IntegrationEventPublisher) ordersport.IntegrationEventPublisher {
	if publisher != nil {
		return publisher
	}

	return noopIntegrationEventPublisher{}
}

// generateEventID creates random integration event identifiers.
func generateEventID() string {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return fmt.Sprintf("event-%d", time.Now().UnixNano())
	}

	return strings.TrimSpace(hex.EncodeToString(bytes))
}
