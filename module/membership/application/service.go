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

const (
	// circleOptInMetadataKey defines circle opt-in metadata key values.
	circleOptInMetadataKey = "flock_checker_circle_optin"
	// circleOptInAcceptedAtMetadataKey defines local accepted-at metadata key values.
	circleOptInAcceptedAtMetadataKey = "flock_checker_circle_optin_accepted_at"
	// circleOptInAcceptedAtUTCMetadataKey defines UTC accepted-at metadata key values.
	circleOptInAcceptedAtUTCMetadataKey = "flock_checker_circle_optin_accepted_at_utc"
	// circleOptInRejectedAtMetadataKey defines local rejected-at metadata key values.
	circleOptInRejectedAtMetadataKey = "flock_checker_circle_optin_rejected_at"
	// circleOptInRejectedAtUTCMetadataKey defines UTC rejected-at metadata key values.
	circleOptInRejectedAtUTCMetadataKey = "flock_checker_circle_optin_rejected_at_utc"
	// circleOptInLocalTimestampLayout defines local timestamp layout values.
	circleOptInLocalTimestampLayout = "2006-01-02 15:04:05"
	// circleOptInLocalTimezoneName defines local timezone values.
	circleOptInLocalTimezoneName = "America/Bogota"
)

var (
	// ErrNilRepository is returned when repository dependencies are nil.
	ErrNilRepository = errors.New("membership repository must not be nil")
)

// MigrateSummary defines migration execution summary values.
type MigrateSummary struct {
	// Processed defines contacts processed during migration.
	Processed int `json:"processed"`
	// Created defines created stamp counts.
	Created int `json:"created"`
	// Skipped defines skipped contact counts.
	Skipped int `json:"skipped"`
	// Failed defines failed contact counts.
	Failed int `json:"failed"`
}

// Service defines membership use-case behavior.
type Service interface {
	// Stamp persists membership stamps and updates latest status snapshots.
	Stamp(ctx context.Context, command port.StampCommand) (*domain.Status, error)
	// GetStatus retrieves one current status by contact and channel.
	GetStatus(ctx context.Context, contactID string, channel domain.Channel) (*domain.Status, error)
	// ListStamps retrieves stamps by contact and channel filters.
	ListStamps(ctx context.Context, contactID string, channel domain.Channel, limit int) ([]domain.Stamp, error)
	// MigrateFromContactMetadata migrates legacy contact metadata values to membership stamps.
	MigrateFromContactMetadata(ctx context.Context, pageSize int) (*MigrateSummary, error)
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

// Stamp persists membership stamps and updates latest status snapshots.
func (s *MembershipService) Stamp(ctx context.Context, command port.StampCommand) (*domain.Status, error) {
	status, _, err := s.stamp(ctx, command)
	if err != nil {
		return nil, err
	}

	return status, nil
}

// stamp persists membership stamps and returns status with creation flags.
func (s *MembershipService) stamp(ctx context.Context, command port.StampCommand) (*domain.Status, bool, error) {
	channel := command.Channel
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
	if !channel.IsValid() {
		return nil, domain.ErrInvalidChannel
	}

	status, err := s.repository.GetStatus(ctx, trimmedContactID, channel)
	if err != nil {
		return nil, fmt.Errorf("get membership status: %w", err)
	}

	return status, nil
}

// ListStamps retrieves stamps by contact and channel filters.
func (s *MembershipService) ListStamps(ctx context.Context, contactID string, channel domain.Channel, limit int) ([]domain.Stamp, error) {
	trimmedContactID := strings.TrimSpace(contactID)
	if trimmedContactID == "" {
		return nil, domain.ErrInvalidContactID
	}
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

// MigrateFromContactMetadata migrates legacy contact metadata values to membership stamps.
func (s *MembershipService) MigrateFromContactMetadata(ctx context.Context, pageSize int) (*MigrateSummary, error) {
	runID := ""
	if s.syncRecorder != nil {
		startedRunID, runErr := s.syncRecorder.StartRun(ctx, "membership.migration", "manual")
		if runErr == nil {
			runID = startedRunID
		}
	}
	syncErrors := make([]port.SyncError, 0, 16)
	appendSyncError := func(errorType string, errorCode string, message string) {
		trimmedMessage := strings.TrimSpace(message)
		if trimmedMessage == "" {
			return
		}
		syncErrors = append(syncErrors, port.SyncError{
			Type:    strings.TrimSpace(errorType),
			Code:    strings.TrimSpace(errorCode),
			Message: trimmedMessage,
		})
	}
	finalizeSyncRecord := func(summary *MigrateSummary) {
		if summary == nil || strings.TrimSpace(runID) == "" || s.syncRecorder == nil {
			return
		}
		if summary.Failed > 0 {
			_ = s.syncRecorder.FailRun(ctx, runID, summary.Processed, summary.Created, summary.Failed, summary.Skipped, syncErrors)
			return
		}

		_ = s.syncRecorder.CompleteRun(ctx, runID, summary.Processed, summary.Created, summary.Failed, summary.Skipped)
	}

	if s.contacts == nil {
		summary := &MigrateSummary{}
		finalizeSyncRecord(summary)
		return summary, nil
	}
	if pageSize <= 0 {
		pageSize = 500
	}

	summary := &MigrateSummary{}
	page := 1
	for {
		contacts, total, err := s.contacts.ListByMetadata(ctx, circleOptInMetadataKey, "", page, pageSize)
		if err != nil {
			appendSyncError("repository", "list_contacts", err.Error())
			finalizeSyncRecord(summary)
			return nil, fmt.Errorf("list contacts for membership migration: %w", err)
		}
		if len(contacts) == 0 {
			break
		}

		for _, contact := range contacts {
			if err := ctx.Err(); err != nil {
				appendSyncError("context", "canceled", err.Error())
				finalizeSyncRecord(summary)
				return nil, err
			}
			summary.Processed++
			action, occurredAt, ok := parseLegacyCircleOptIn(contact.Metadata)
			if !ok {
				summary.Skipped++
				continue
			}

			_, created, stampErr := s.stamp(ctx, port.StampCommand{
				ContactID: contact.ID,
				Channel:   domain.ChannelEmail,
				Action:    action,
				Source:    "migration",
				OccurredAt: func() *time.Time {
					value := occurredAt.UTC()
					return &value
				}(),
			})
			if stampErr != nil {
				summary.Failed++
				appendSyncError("stamp", "save_stamp", stampErr.Error())
				continue
			}
			if created {
				summary.Created++
			} else {
				summary.Skipped++
			}
		}

		if int64(page*pageSize) >= total {
			break
		}
		page++
	}

	finalizeSyncRecord(summary)
	return summary, nil
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

// parseLegacyCircleOptIn maps legacy checker metadata into action and timestamp values.
func parseLegacyCircleOptIn(metadata map[string]string) (domain.Action, time.Time, bool) {
	if len(metadata) == 0 {
		return "", time.Time{}, false
	}

	decision := strings.ToLower(strings.TrimSpace(metadata[circleOptInMetadataKey]))
	switch decision {
	case "yes":
		return domain.ActionOptIn, parseLegacyOccurredAt(metadata[circleOptInAcceptedAtUTCMetadataKey], metadata[circleOptInAcceptedAtMetadataKey]), true
	case "no":
		return domain.ActionOptOut, parseLegacyOccurredAt(metadata[circleOptInRejectedAtUTCMetadataKey], metadata[circleOptInRejectedAtMetadataKey]), true
	default:
		return "", time.Time{}, false
	}
}

// parseLegacyOccurredAt resolves legacy UTC/local metadata into a timestamp.
func parseLegacyOccurredAt(utcValue string, localValue string) time.Time {
	if parsedUTC, err := time.Parse(time.RFC3339, strings.TrimSpace(utcValue)); err == nil {
		return parsedUTC.UTC()
	}

	location, locErr := time.LoadLocation(circleOptInLocalTimezoneName)
	if locErr != nil {
		location = time.FixedZone("UTC-05", -5*60*60)
	}
	if parsedLocal, err := time.ParseInLocation(circleOptInLocalTimestampLayout, strings.TrimSpace(localValue), location); err == nil {
		return parsedLocal.UTC()
	}

	return time.Now().UTC()
}
