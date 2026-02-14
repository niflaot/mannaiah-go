package application

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"mannaiah/module/contacts/domain"
	"mannaiah/module/contacts/port"
)

const (
	// TopicContactCreated defines contact-created integration event topics.
	TopicContactCreated = "contacts.v1.created"
	// TopicContactUpdated defines contact-updated integration event topics.
	TopicContactUpdated = "contacts.v1.updated"
	// schemaVersionV1 defines current integration event schema versions.
	schemaVersionV1 = "v1"
)

// ContactEventPayload defines integration event payload fields for contact events.
type ContactEventPayload struct {
	// ID defines contact identifiers.
	ID string `json:"id"`
	// DocumentType defines document category values.
	DocumentType domain.DocumentType `json:"documentType"`
	// DocumentNumber defines document number values.
	DocumentNumber string `json:"documentNumber"`
	// LegalName defines legal name values.
	LegalName string `json:"legalName"`
	// FirstName defines first name values.
	FirstName string `json:"firstName"`
	// LastName defines last name values.
	LastName string `json:"lastName"`
	// Email defines email values.
	Email string `json:"email"`
	// Phone defines phone values.
	Phone string `json:"phone"`
	// Address defines address values.
	Address string `json:"address"`
	// AddressExtra defines address extra values.
	AddressExtra string `json:"addressExtra"`
	// CityCode defines city code values.
	CityCode string `json:"cityCode"`
	// Metadata defines optional contact metadata values.
	Metadata map[string]string `json:"metadata,omitempty"`
	// CreatedAt defines creation timestamps.
	CreatedAt time.Time `json:"createdAt"`
	// UpdatedAt defines update timestamps.
	UpdatedAt time.Time `json:"updatedAt"`
}

// noopIntegrationEventPublisher defines no-op event publishing behavior.
type noopIntegrationEventPublisher struct{}

// Publish ignores integration events.
func (noopIntegrationEventPublisher) Publish(ctx context.Context, event port.IntegrationEvent) error {
	return nil
}

// resolvePublisher resolves integration publisher dependencies.
func resolvePublisher(publisher port.IntegrationEventPublisher) port.IntegrationEventPublisher {
	if publisher != nil {
		return publisher
	}

	return noopIntegrationEventPublisher{}
}

// buildContactCreatedIntegrationEvent maps created domain events to integration event envelopes.
func buildContactCreatedIntegrationEvent(event domain.ContactCreatedEvent) port.IntegrationEvent {
	occurredAt := event.OccurredAt().UTC()
	eventID := generateEventID()

	return port.IntegrationEvent{
		ID:            eventID,
		Topic:         TopicContactCreated,
		SchemaVersion: schemaVersionV1,
		OccurredAt:    occurredAt,
		Payload:       toContactEventPayload(event.Contact),
		Metadata: map[string]string{
			"aggregate_id": event.AggregateID(),
		},
	}
}

// buildContactUpdatedIntegrationEvent maps updated domain events to integration event envelopes.
func buildContactUpdatedIntegrationEvent(event domain.ContactUpdatedEvent) port.IntegrationEvent {
	occurredAt := event.OccurredAt().UTC()
	eventID := generateEventID()

	return port.IntegrationEvent{
		ID:            eventID,
		Topic:         TopicContactUpdated,
		SchemaVersion: schemaVersionV1,
		OccurredAt:    occurredAt,
		Payload:       toContactEventPayload(event.Contact),
		Metadata: map[string]string{
			"aggregate_id": event.AggregateID(),
		},
	}
}

// toContactEventPayload maps contacts to integration event payload values.
func toContactEventPayload(contact domain.Contact) ContactEventPayload {
	return ContactEventPayload{
		ID:             contact.ID,
		DocumentType:   contact.DocumentType,
		DocumentNumber: contact.DocumentNumber,
		LegalName:      contact.LegalName,
		FirstName:      contact.FirstName,
		LastName:       contact.LastName,
		Email:          contact.Email,
		Phone:          contact.Phone,
		Address:        contact.Address,
		AddressExtra:   contact.AddressExtra,
		CityCode:       contact.CityCode,
		Metadata:       contact.Metadata,
		CreatedAt:      contact.CreatedAt,
		UpdatedAt:      contact.UpdatedAt,
	}
}

// generateEventID creates random integration event identifiers.
func generateEventID() string {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return fmt.Sprintf("event-%d", time.Now().UnixNano())
	}

	return strings.TrimSpace(hex.EncodeToString(bytes))
}
