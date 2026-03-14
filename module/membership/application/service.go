package application

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"mannaiah/module/membership/domain"
	"mannaiah/module/membership/port"
)

var (
	// ErrNilRepository is returned when repository dependencies are nil.
	ErrNilRepository = errors.New("membership repository must not be nil")
)

// Service defines membership use-case behavior.
type Service interface {
	// Stamp persists membership stamps and resolves latest status values.
	Stamp(ctx context.Context, command port.StampCommand) (*domain.Status, error)
	// GetStatus retrieves one current status by contact and channel.
	GetStatus(ctx context.Context, contactID string, channel domain.Channel) (*domain.Status, error)
	// GetStatuses retrieves current statuses by contact across all channels.
	GetStatuses(ctx context.Context, contactID string) ([]domain.Status, error)
	// ListStamps retrieves stamps by contact and channel filters.
	ListStamps(ctx context.Context, contactID string, channel domain.Channel, limit int) ([]domain.Stamp, error)
}

// MembershipService implements membership use-cases.
type MembershipService struct {
	// repository defines persistence dependencies.
	repository port.Repository
	// contacts defines optional contact lookup dependencies.
	contacts port.ContactLookup
	// publisher defines integration event publication dependencies.
	publisher port.IntegrationEventPublisher
	// syncRecorder defines optional sync run recording dependencies.
	syncRecorder port.SyncRecorder
}

var (
	// _ ensures MembershipService satisfies service contracts.
	_ Service = (*MembershipService)(nil)
	// _ ensures MembershipService satisfies external stamper contracts.
	_ port.Stamper = (*MembershipService)(nil)
)

// NewService creates membership services.
func NewService(repository port.Repository, contacts port.ContactLookup, publishers ...port.IntegrationEventPublisher) (*MembershipService, error) {
	if repository == nil {
		return nil, ErrNilRepository
	}

	var publisher port.IntegrationEventPublisher
	if len(publishers) > 0 {
		publisher = publishers[0]
	}

	return &MembershipService{
		repository:   repository,
		contacts:     contacts,
		publisher:    publisher,
		syncRecorder: port.NoopSyncRecorder{},
	}, nil
}

// SetSyncRecorder configures optional sync run recording dependencies.
func (s *MembershipService) SetSyncRecorder(recorder port.SyncRecorder) {
	if s == nil {
		return
	}
	if recorder == nil {
		s.syncRecorder = port.NoopSyncRecorder{}
		return
	}

	s.syncRecorder = recorder
}

// Stamp persists membership stamps and resolves latest status values.
func (s *MembershipService) Stamp(ctx context.Context, command port.StampCommand) (*domain.Status, error) {
	status, _, err := s.stamp(ctx, command)
	if err != nil {
		return nil, err
	}

	return status, nil
}

// stamp persists membership stamps and returns status with creation flags.
func (s *MembershipService) stamp(ctx context.Context, command port.StampCommand) (*domain.Status, bool, error) {
	channel := domain.Channel(strings.ToLower(strings.TrimSpace(string(command.Channel))))
	if !channel.IsValid() {
		return nil, false, domain.ErrInvalidChannel
	}
	action := command.Action
	if !action.IsValid() {
		return nil, false, domain.ErrInvalidAction
	}

	contactID := strings.TrimSpace(command.ContactID)
	if contactID == "" {
		email := strings.ToLower(strings.TrimSpace(command.Email))
		if email == "" {
			return nil, false, domain.ErrInvalidEmail
		}
		if s.contacts == nil {
			return nil, false, domain.ErrContactNotFound
		}

		contact, err := s.contacts.FindByEmail(ctx, email)
		if err != nil {
			return nil, false, fmt.Errorf("find contact by email: %w", err)
		}
		if contact == nil || strings.TrimSpace(contact.ID) == "" {
			return nil, false, domain.ErrContactNotFound
		}
		contactID = strings.TrimSpace(contact.ID)
	}

	source := strings.TrimSpace(command.Source)
	if source == "" {
		source = "api"
	}

	occurredAt := time.Now().UTC()
	if command.OccurredAt != nil && !command.OccurredAt.IsZero() {
		occurredAt = command.OccurredAt.UTC()
	}

	result, err := s.repository.SaveStamp(ctx, port.StampInput{
		ContactID:  contactID,
		Channel:    channel,
		Action:     action,
		Source:     source,
		OccurredAt: occurredAt,
	})
	if err != nil {
		return nil, false, fmt.Errorf("save membership stamp: %w", err)
	}
	if result == nil {
		return nil, false, errors.New("save membership stamp returned nil result")
	}

	if result.Created {
		s.publishChanged(ctx, result.Status)
	}

	status := result.Status
	return &status, result.Created, nil
}

// GetStatus retrieves one current status by contact and channel.
func (s *MembershipService) GetStatus(ctx context.Context, contactID string, channel domain.Channel) (*domain.Status, error) {
	trimmedContactID := strings.TrimSpace(contactID)
	if trimmedContactID == "" {
		return nil, domain.ErrInvalidContactID
	}
	channel = domain.Channel(strings.ToLower(strings.TrimSpace(string(channel))))
	if !channel.IsValid() {
		return nil, domain.ErrInvalidChannel
	}

	status, err := s.repository.GetStatus(ctx, trimmedContactID, channel)
	if err != nil {
		return nil, fmt.Errorf("get membership status: %w", err)
	}

	return status, nil
}

// GetStatuses retrieves current statuses by contact across all channels.
func (s *MembershipService) GetStatuses(ctx context.Context, contactID string) ([]domain.Status, error) {
	trimmedContactID := strings.TrimSpace(contactID)
	if trimmedContactID == "" {
		return nil, domain.ErrInvalidContactID
	}

	statuses, err := s.repository.GetStatuses(ctx, trimmedContactID)
	if err != nil {
		return nil, fmt.Errorf("get membership statuses: %w", err)
	}

	return statuses, nil
}

// ListStamps retrieves stamps by contact and channel filters.
func (s *MembershipService) ListStamps(ctx context.Context, contactID string, channel domain.Channel, limit int) ([]domain.Stamp, error) {
	trimmedContactID := strings.TrimSpace(contactID)
	if trimmedContactID == "" {
		return nil, domain.ErrInvalidContactID
	}
	channel = domain.Channel(strings.ToLower(strings.TrimSpace(string(channel))))
	if !channel.IsValid() {
		return nil, domain.ErrInvalidChannel
	}
	if limit <= 0 {
		limit = 100
	}

	rows, err := s.repository.ListStamps(ctx, trimmedContactID, channel, limit)
	if err != nil {
		return nil, fmt.Errorf("list membership stamps: %w", err)
	}

	return rows, nil
}

// publishChanged publishes membership change integration events.
func (s *MembershipService) publishChanged(ctx context.Context, status domain.Status) {
	if s == nil || s.publisher == nil {
		return
	}

	_ = s.publisher.Publish(ctx, port.IntegrationEvent{
		ID:            uuid.NewString(),
		Topic:         port.TopicMembershipChanged,
		SchemaVersion: "1.0.0",
		OccurredAt:    status.OccurredAt.UTC(),
		Payload: port.MembershipChangedPayload{
			ContactID:  status.ContactID,
			Channel:    string(status.Channel),
			Action:     string(status.Action),
			Source:     status.Source,
			OccurredAt: status.OccurredAt.UTC(),
		},
	})
}
