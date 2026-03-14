package application

import (
	"context"
	"errors"
	"testing"
	"time"

	"mannaiah/module/membership/domain"
	"mannaiah/module/membership/port"
)

// repositoryMock defines membership repository behavior for service tests.
type repositoryMock struct {
	saveStampFn  func(ctx context.Context, input port.StampInput) (*port.StampResult, error)
	getStatusFn  func(ctx context.Context, contactID string, channel domain.Channel) (*domain.Status, error)
	listStampsFn func(ctx context.Context, contactID string, channel domain.Channel, limit int) ([]domain.Stamp, error)
}

// SaveStamp persists immutable stamps and updates latest status snapshots.
func (m repositoryMock) SaveStamp(ctx context.Context, input port.StampInput) (*port.StampResult, error) {
	return m.saveStampFn(ctx, input)
}

// GetStatus retrieves latest status by contact and channel.
func (m repositoryMock) GetStatus(ctx context.Context, contactID string, channel domain.Channel) (*domain.Status, error) {
	return m.getStatusFn(ctx, contactID, channel)
}

// ListStamps retrieves stamps by contact and channel filters.
func (m repositoryMock) ListStamps(ctx context.Context, contactID string, channel domain.Channel, limit int) ([]domain.Stamp, error) {
	return m.listStampsFn(ctx, contactID, channel, limit)
}

// contactLookupMock defines contact lookup behavior for service tests.
type contactLookupMock struct {
	findByEmailFn    func(ctx context.Context, email string) (*port.ContactSnapshot, error)
	listByMetadataFn func(ctx context.Context, metadataKey string, metadataValue string, page int, limit int) ([]port.ContactSnapshot, int64, error)
}

// FindByEmail resolves one contact by normalized email values.
func (m contactLookupMock) FindByEmail(ctx context.Context, email string) (*port.ContactSnapshot, error) {
	return m.findByEmailFn(ctx, email)
}

// ListByMetadata resolves contacts by metadata key/value filters.
func (m contactLookupMock) ListByMetadata(ctx context.Context, metadataKey string, metadataValue string, page int, limit int) ([]port.ContactSnapshot, int64, error) {
	return m.listByMetadataFn(ctx, metadataKey, metadataValue, page, limit)
}

// TestStampByEmail verifies email lookup stamp behavior.
func TestStampByEmail(t *testing.T) {
	svc, err := NewService(
		repositoryMock{
			saveStampFn: func(ctx context.Context, input port.StampInput) (*port.StampResult, error) {
				if input.ContactID != "c-1" {
					t.Fatalf("input.ContactID = %q, want c-1", input.ContactID)
				}
				return &port.StampResult{Status: domain.Status{ContactID: "c-1", Channel: domain.ChannelEmail, Action: domain.ActionOptIn}, Created: true}, nil
			},
			getStatusFn: func(ctx context.Context, contactID string, channel domain.Channel) (*domain.Status, error) {
				return nil, nil
			},
			listStampsFn: func(ctx context.Context, contactID string, channel domain.Channel, limit int) ([]domain.Stamp, error) {
				return nil, nil
			},
		},
		contactLookupMock{
			findByEmailFn: func(ctx context.Context, email string) (*port.ContactSnapshot, error) {
				return &port.ContactSnapshot{ID: "c-1", Email: email}, nil
			},
			listByMetadataFn: func(ctx context.Context, metadataKey string, metadataValue string, page int, limit int) ([]port.ContactSnapshot, int64, error) {
				return nil, 0, nil
			},
		},
	)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}

	status, stampErr := svc.Stamp(context.Background(), port.StampCommand{Email: "john@example.com", Channel: domain.ChannelEmail, Action: domain.ActionOptIn})
	if stampErr != nil {
		t.Fatalf("Stamp() error = %v", stampErr)
	}
	if status.ContactID != "c-1" {
		t.Fatalf("status.ContactID = %q, want c-1", status.ContactID)
	}
}

// TestGetStatusValidation verifies contact-id validation behavior.
func TestGetStatusValidation(t *testing.T) {
	svc, err := NewService(
		repositoryMock{
			saveStampFn: func(ctx context.Context, input port.StampInput) (*port.StampResult, error) { return nil, nil },
			getStatusFn: func(ctx context.Context, contactID string, channel domain.Channel) (*domain.Status, error) {
				return nil, domain.ErrStatusNotFound
			},
			listStampsFn: func(ctx context.Context, contactID string, channel domain.Channel, limit int) ([]domain.Stamp, error) {
				return nil, nil
			},
		},
		nil,
	)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}

	_, getErr := svc.GetStatus(context.Background(), " ", domain.ChannelEmail)
	if !errors.Is(getErr, domain.ErrInvalidContactID) {
		t.Fatalf("GetStatus() error = %v, want ErrInvalidContactID", getErr)
	}
}

// TestParseLegacyOccurredAt verifies timestamp parsing behavior.
func TestParseLegacyOccurredAt(t *testing.T) {
	value := parseLegacyOccurredAt("2026-03-01T10:00:00Z", "")
	if value.IsZero() {
		t.Fatalf("parseLegacyOccurredAt() returned zero")
	}

	fallback := parseLegacyOccurredAt("", "")
	if time.Since(fallback) > time.Minute {
		t.Fatalf("fallback timestamp is stale: %v", fallback)
	}
}
