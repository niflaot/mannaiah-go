package application

import (
	"context"
	"errors"
	"testing"
	"time"

	"mannaiah/module/contacts/domain"
	"mannaiah/module/contacts/port"
)

// integrationEventPublisherMock defines integration event publishing behavior for unit tests.
type integrationEventPublisherMock struct {
	// publishFn defines publish behavior.
	publishFn func(ctx context.Context, event port.IntegrationEvent) error
}

// Publish executes configured publish behavior.
func (m integrationEventPublisherMock) Publish(ctx context.Context, event port.IntegrationEvent) error {
	return m.publishFn(ctx, event)
}

// TestResolvePublisherFallback verifies nil publisher fallback behavior.
func TestResolvePublisherFallback(t *testing.T) {
	publisher := resolvePublisher(nil)
	if publisher == nil {
		t.Fatalf("expected fallback publisher")
	}

	if err := publisher.Publish(context.Background(), port.IntegrationEvent{}); err != nil {
		t.Fatalf("Publish() error = %v", err)
	}
}

// TestBuildContactCreatedIntegrationEvent verifies created-event integration mapping.
func TestBuildContactCreatedIntegrationEvent(t *testing.T) {
	event := buildContactCreatedIntegrationEvent(domain.NewContactCreatedEvent(domain.Contact{
		ID:        "c-1",
		Email:     "a@example.com",
		LegalName: "Acme",
		Metadata:  map[string]string{"marketing.consent": "true"},
		CreatedAt: time.Now().UTC(),
	}))
	if event.Topic != TopicContactCreated {
		t.Fatalf("Topic = %q, want %q", event.Topic, TopicContactCreated)
	}
	if event.SchemaVersion != schemaVersionV1 {
		t.Fatalf("SchemaVersion = %q, want %q", event.SchemaVersion, schemaVersionV1)
	}
	if event.ID == "" {
		t.Fatalf("expected event id")
	}
	payload, ok := event.Payload.(ContactEventPayload)
	if !ok {
		t.Fatalf("Payload type = %T, want ContactEventPayload", event.Payload)
	}
	if payload.ID != "c-1" {
		t.Fatalf("payload ID = %q, want %q", payload.ID, "c-1")
	}
	if payload.Metadata["marketing.consent"] != "true" {
		t.Fatalf("payload.Metadata[marketing.consent] = %q, want %q", payload.Metadata["marketing.consent"], "true")
	}
}

// TestBuildContactUpdatedIntegrationEvent verifies updated-event integration mapping.
func TestBuildContactUpdatedIntegrationEvent(t *testing.T) {
	event := buildContactUpdatedIntegrationEvent(domain.NewContactUpdatedEvent(domain.Contact{ID: "c-2", Email: "b@example.com", FirstName: "John", LastName: "Doe", UpdatedAt: time.Now().UTC()}))
	if event.Topic != TopicContactUpdated {
		t.Fatalf("Topic = %q, want %q", event.Topic, TopicContactUpdated)
	}
	if event.ID == "" {
		t.Fatalf("expected event id")
	}
}

// TestGenerateEventID verifies integration event id generation behavior.
func TestGenerateEventID(t *testing.T) {
	value := generateEventID()
	if value == "" {
		t.Fatalf("expected non-empty event id")
	}
}

// TestNewServiceWithPublisher verifies publisher-aware constructor behavior.
func TestNewServiceWithPublisher(t *testing.T) {
	svc, err := NewServiceWithPublisher(repositoryMock{
		createFn:  func(ctx context.Context, contact *domain.Contact) error { return nil },
		getByIDFn: func(ctx context.Context, id string) (*domain.Contact, error) { return &domain.Contact{}, nil },
		listFn:    func(ctx context.Context, query port.ListQuery) ([]domain.Contact, int64, error) { return nil, 0, nil },
		updateFn:  func(ctx context.Context, contact *domain.Contact) error { return nil },
		deleteFn:  func(ctx context.Context, id string) error { return nil },
	}, integrationEventPublisherMock{publishFn: func(ctx context.Context, event port.IntegrationEvent) error { return nil }})
	if err != nil {
		t.Fatalf("NewServiceWithPublisher() error = %v", err)
	}
	if svc.publisher == nil {
		t.Fatalf("expected configured publisher")
	}
}

// TestNoopIntegrationEventPublisher verifies no-op publisher behavior.
func TestNoopIntegrationEventPublisher(t *testing.T) {
	publisher := noopIntegrationEventPublisher{}
	if err := publisher.Publish(context.Background(), port.IntegrationEvent{}); err != nil {
		t.Fatalf("Publish() error = %v", err)
	}
}

// TestPublisherErrorPropagationOnCreate verifies create publish error propagation.
func TestPublisherErrorPropagationOnCreate(t *testing.T) {
	svc, err := NewServiceWithPublisher(repositoryMock{
		createFn:  func(ctx context.Context, contact *domain.Contact) error { contact.ID = "c-1"; return nil },
		getByIDFn: func(ctx context.Context, id string) (*domain.Contact, error) { return &domain.Contact{}, nil },
		listFn:    func(ctx context.Context, query port.ListQuery) ([]domain.Contact, int64, error) { return nil, 0, nil },
		updateFn:  func(ctx context.Context, contact *domain.Contact) error { return nil },
		deleteFn:  func(ctx context.Context, id string) error { return nil },
	}, integrationEventPublisherMock{publishFn: func(ctx context.Context, event port.IntegrationEvent) error {
		return errors.New("publish failed")
	}})
	if err != nil {
		t.Fatalf("NewServiceWithPublisher() error = %v", err)
	}

	_, createErr := svc.Create(context.Background(), CreateCommand{Email: "john@example.com", LegalName: "Acme"})
	if createErr == nil {
		t.Fatalf("expected publish error")
	}
}

// TestPublisherErrorPropagationOnUpdate verifies update publish error propagation.
func TestPublisherErrorPropagationOnUpdate(t *testing.T) {
	record := &domain.Contact{ID: "c-1", Email: "a@example.com", FirstName: "John", LastName: "Doe"}
	svc, err := NewServiceWithPublisher(repositoryMock{
		createFn:  func(ctx context.Context, contact *domain.Contact) error { return nil },
		getByIDFn: func(ctx context.Context, id string) (*domain.Contact, error) { return record, nil },
		listFn:    func(ctx context.Context, query port.ListQuery) ([]domain.Contact, int64, error) { return nil, 0, nil },
		updateFn:  func(ctx context.Context, contact *domain.Contact) error { return nil },
		deleteFn:  func(ctx context.Context, id string) error { return nil },
	}, integrationEventPublisherMock{publishFn: func(ctx context.Context, event port.IntegrationEvent) error {
		return errors.New("publish failed")
	}})
	if err != nil {
		t.Fatalf("NewServiceWithPublisher() error = %v", err)
	}

	value := "next@example.com"
	_, updateErr := svc.Update(context.Background(), "c-1", UpdateCommand{Email: &value})
	if updateErr == nil {
		t.Fatalf("expected publish error")
	}
}
