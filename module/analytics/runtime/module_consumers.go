package runtime

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"mannaiah/module/analytics/port"
	"mannaiah/module/core/messaging/bus"
	"mannaiah/module/core/messaging/platform"
)

const (
	topicContactCreated     = "contacts.v1.created"
	topicContactUpdated     = "contacts.v1.updated"
	topicOrderCreated       = "orders.v1.created"
	topicOrderUpdated       = "orders.v1.updated"
	topicOrderStatusUpdated = "orders.v1.status.updated"
	topicMembershipChanged  = "membership.v1.changed"
	topicCampaignDelivered  = "campaign.v1.delivery"
)

type contactEventPayload struct {
	ID           string            `json:"id"`
	DocumentType string            `json:"documentType"`
	LegalName    string            `json:"legalName"`
	FirstName    string            `json:"firstName"`
	LastName     string            `json:"lastName"`
	Email        string            `json:"email"`
	Phone        string            `json:"phone"`
	CityCode     string            `json:"cityCode"`
	Metadata     map[string]string `json:"metadata"`
	CreatedAt    time.Time         `json:"createdAt"`
	UpdatedAt    time.Time         `json:"updatedAt"`
}

type orderEventPayload struct {
	ID            string           `json:"id"`
	Identifier    string           `json:"identifier"`
	Realm         string           `json:"realm"`
	ContactID     string           `json:"contactId"`
	CurrentStatus string           `json:"currentStatus"`
	LatestStatus  orderEventStatus `json:"latestStatus"`
	Items         []orderEventItem `json:"items"`
	CreatedAt     time.Time        `json:"createdAt"`
	UpdatedAt     time.Time        `json:"updatedAt"`
}

type orderEventStatus struct {
	Status string `json:"status"`
}

type orderEventItem struct {
	SKU              string  `json:"sku"`
	AlternateName    string  `json:"alternateName"`
	Quantity         int     `json:"quantity"`
	Value            float64 `json:"value"`
	ProductID        string  `json:"productId"`
	ResolutionSource string  `json:"resolutionSource"`
}

type membershipChangedPayload struct {
	ContactID  string    `json:"contactId"`
	Channel    string    `json:"channel"`
	Action     string    `json:"action"`
	Source     string    `json:"source"`
	OccurredAt time.Time `json:"occurredAt"`
}

type campaignDeliveryPayload struct {
	CampaignID      string    `json:"campaignId"`
	ContactID       string    `json:"contactId"`
	Channel         string    `json:"channel"`
	Status          string    `json:"status"`
	TemplateVersion int       `json:"templateVersion"`
	OccurredAt      time.Time `json:"occurredAt"`
}

func (m *Module) registerIntegrationHandlers(registrar bus.Registrar) error {
	if m == nil || registrar == nil || m.service == nil || !m.cfg.Enabled || m.clickhouseClient == nil {
		return nil
	}

	handlers := map[string]bus.Handler{
		topicContactCreated:     m.handleContactEvent,
		topicContactUpdated:     m.handleContactEvent,
		topicOrderCreated:       m.handleOrderEvent,
		topicOrderUpdated:       m.handleOrderEvent,
		topicOrderStatusUpdated: m.handleOrderEvent,
		topicMembershipChanged:  m.handleMembershipChanged,
		topicCampaignDelivered:  m.handleCampaignDelivered,
	}

	for topic, handler := range handlers {
		if err := registrar.AddHandler(topic, handler); err != nil {
			return fmt.Errorf("register analytics handler %q: %w", topic, err)
		}
	}

	return nil
}

func (m *Module) handleContactEvent(ctx context.Context, message bus.Message) error {
	payload := contactEventPayload{}
	if err := json.Unmarshal(message.Payload, &payload); err != nil {
		return platform.NonRetriable(fmt.Errorf("decode contact event payload: %w", err))
	}
	if strings.TrimSpace(payload.ID) == "" {
		return platform.NonRetriable(fmt.Errorf("contact event missing contact id"))
	}

	now := time.Now().UTC()
	row := port.ContactSnapshot{
		ContactID:    strings.TrimSpace(payload.ID),
		Email:        strings.TrimSpace(payload.Email),
		FirstName:    strings.TrimSpace(payload.FirstName),
		LastName:     strings.TrimSpace(payload.LastName),
		LegalName:    strings.TrimSpace(payload.LegalName),
		Phone:        strings.TrimSpace(payload.Phone),
		CityCode:     strings.TrimSpace(payload.CityCode),
		DocumentType: strings.TrimSpace(payload.DocumentType),
		Metadata:     payload.Metadata,
		CreatedAt:    normalizeTime(payload.CreatedAt, now),
		UpdatedAt:    normalizeTime(payload.UpdatedAt, now),
	}

	return m.service.IngestContacts(ctx, []port.ContactSnapshot{row})
}

func (m *Module) handleOrderEvent(ctx context.Context, message bus.Message) error {
	payload := orderEventPayload{}
	if err := json.Unmarshal(message.Payload, &payload); err != nil {
		return platform.NonRetriable(fmt.Errorf("decode order event payload: %w", err))
	}
	if strings.TrimSpace(payload.ID) == "" {
		return platform.NonRetriable(fmt.Errorf("order event missing order id"))
	}
	if strings.TrimSpace(payload.ContactID) == "" {
		return platform.NonRetriable(fmt.Errorf("order event missing contact id"))
	}

	now := time.Now().UTC()
	createdAt := normalizeTime(payload.CreatedAt, now)
	updatedAt := normalizeTime(payload.UpdatedAt, now)
	status := strings.TrimSpace(payload.CurrentStatus)
	if status == "" {
		status = strings.TrimSpace(payload.LatestStatus.Status)
	}

	order := port.OrderFact{
		OrderID:       strings.TrimSpace(payload.ID),
		Identifier:    strings.TrimSpace(payload.Identifier),
		Realm:         strings.TrimSpace(payload.Realm),
		ContactID:     strings.TrimSpace(payload.ContactID),
		CurrentStatus: status,
		TotalValue:    0,
		ItemCount:     0,
		CreatedAt:     createdAt,
		UpdatedAt:     updatedAt,
	}
	items := make([]port.OrderItemFact, 0, len(payload.Items))
	for _, row := range payload.Items {
		order.TotalValue += row.Value
		order.ItemCount += row.Quantity
		items = append(items, port.OrderItemFact{
			OrderID:          order.OrderID,
			ContactID:        order.ContactID,
			SKU:              strings.TrimSpace(row.SKU),
			AlternateName:    strings.TrimSpace(row.AlternateName),
			ProductID:        strings.TrimSpace(row.ProductID),
			Quantity:         row.Quantity,
			Value:            row.Value,
			ResolutionSource: strings.TrimSpace(row.ResolutionSource),
			OrderCreatedAt:   createdAt,
			OrderUpdatedAt:   updatedAt,
		})
	}

	return m.service.IngestOrders(ctx, []port.OrderFact{order}, items)
}

func (m *Module) handleMembershipChanged(ctx context.Context, message bus.Message) error {
	payload := membershipChangedPayload{}
	if err := json.Unmarshal(message.Payload, &payload); err != nil {
		return platform.NonRetriable(fmt.Errorf("decode membership event payload: %w", err))
	}
	if strings.TrimSpace(payload.ContactID) == "" || strings.TrimSpace(payload.Channel) == "" || strings.TrimSpace(payload.Action) == "" {
		return platform.NonRetriable(fmt.Errorf("membership event missing required fields"))
	}

	now := time.Now().UTC()
	row := port.MembershipEvent{
		ContactID:  strings.TrimSpace(payload.ContactID),
		Channel:    strings.TrimSpace(payload.Channel),
		Action:     strings.TrimSpace(payload.Action),
		Source:     strings.TrimSpace(payload.Source),
		OccurredAt: normalizeTime(payload.OccurredAt, now),
	}

	return m.service.IngestMembershipEvents(ctx, []port.MembershipEvent{row})
}

func (m *Module) handleCampaignDelivered(ctx context.Context, message bus.Message) error {
	payload := campaignDeliveryPayload{}
	if err := json.Unmarshal(message.Payload, &payload); err != nil {
		return platform.NonRetriable(fmt.Errorf("decode campaign event payload: %w", err))
	}
	if strings.TrimSpace(payload.CampaignID) == "" || strings.TrimSpace(payload.ContactID) == "" {
		return platform.NonRetriable(fmt.Errorf("campaign event missing required fields"))
	}

	now := time.Now().UTC()
	row := port.CampaignEvent{
		CampaignID:      strings.TrimSpace(payload.CampaignID),
		ContactID:       strings.TrimSpace(payload.ContactID),
		Channel:         strings.TrimSpace(payload.Channel),
		Status:          strings.TrimSpace(payload.Status),
		TemplateVersion: payload.TemplateVersion,
		OccurredAt:      normalizeTime(payload.OccurredAt, now),
	}

	return m.service.IngestCampaignEvents(ctx, []port.CampaignEvent{row})
}

func normalizeTime(value time.Time, fallback time.Time) time.Time {
	if value.IsZero() {
		return fallback.UTC()
	}

	return value.UTC()
}
