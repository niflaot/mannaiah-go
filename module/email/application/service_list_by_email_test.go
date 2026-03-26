package application

import (
	"context"
	"errors"
	"testing"
	"time"

	"mannaiah/module/email/domain"
)

// TestListByEmailReturnsRecipientDeliveries verifies recipient email listing behavior.
func TestListByEmailReturnsRecipientDeliveries(t *testing.T) {
	t.Parallel()

	repository := &snsWebhookRepositoryStub{
		deliveriesByID: map[string]*domain.Delivery{
			"d-1": {ID: "d-1", Email: "user@example.com", CreatedAt: time.Date(2026, time.March, 26, 10, 0, 0, 0, time.UTC)},
			"d-2": {ID: "d-2", Email: "other@example.com", CreatedAt: time.Date(2026, time.March, 26, 11, 0, 0, 0, time.UTC)},
			"d-3": {ID: "d-3", Email: "USER@example.com", CreatedAt: time.Date(2026, time.March, 26, 12, 0, 0, 0, time.UTC)},
		},
		providerToID: map[string]string{},
	}
	service, err := NewService(repository, snsWebhookProviderStub{})
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}

	rows, listErr := service.ListByEmail(context.Background(), "  user@example.com ")
	if listErr != nil {
		t.Fatalf("ListByEmail() error = %v", listErr)
	}
	if len(rows) != 2 {
		t.Fatalf("len(rows) = %d, want 2", len(rows))
	}
	if rows[0].ID != "d-3" || rows[1].ID != "d-1" {
		t.Fatalf("rows order = [%s, %s], want [d-3, d-1]", rows[0].ID, rows[1].ID)
	}
}

// TestListByEmailRejectsEmptyEmail verifies validation behavior for empty recipient email values.
func TestListByEmailRejectsEmptyEmail(t *testing.T) {
	t.Parallel()

	service, err := NewService(&snsWebhookRepositoryStub{
		deliveriesByID: map[string]*domain.Delivery{},
		providerToID:   map[string]string{},
	}, snsWebhookProviderStub{})
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}

	_, listErr := service.ListByEmail(context.Background(), " ")
	if !errors.Is(listErr, domain.ErrInvalidEmail) {
		t.Fatalf("ListByEmail() error = %v, want %v", listErr, domain.ErrInvalidEmail)
	}
}
