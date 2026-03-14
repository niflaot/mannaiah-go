package application

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"mannaiah/module/email/domain"
	"mannaiah/module/email/port"
)

var (
	// ErrNilRepository is returned when nil repository dependencies are provided.
	ErrNilRepository = errors.New("email repository must not be nil")
)

// SendCommand defines email send payload values.
type SendCommand struct {
	// ContactID defines optional contact identifier values.
	ContactID string
	// Email defines recipient email values.
	Email string
	// Subject defines subject values.
	Subject string
	// HTMLBody defines html payload values.
	HTMLBody string
	// TextBody defines text payload values.
	TextBody string
	// IdempotencyKey defines idempotency values.
	IdempotencyKey string
}

// WebhookCommand defines provider webhook payload values.
type WebhookCommand struct {
	// ProviderMessageID defines provider message identifier values.
	ProviderMessageID string
	// Status defines provider status values.
	Status string
	// Reason defines optional reason values.
	Reason string
	// Email defines optional recipient email values.
	Email string
}

// Service defines email use-case behavior.
type Service interface {
	// Send dispatches one email and tracks delivery status.
	Send(ctx context.Context, command SendCommand) (*domain.Delivery, error)
	// HandleWebhook updates delivery status from provider webhook payloads.
	HandleWebhook(ctx context.Context, command WebhookCommand) error
	// Get retrieves one delivery by id.
	Get(ctx context.Context, deliveryID string) (*domain.Delivery, error)
}

// EmailService implements email use-cases.
type EmailService struct {
	// repository defines persistence dependencies.
	repository port.Repository
	// provider defines outbound provider dependencies.
	provider port.Provider
	// membershipStamper defines optional membership stamp dependencies.
	membershipStamper port.MembershipStamper
	// providerName defines provider label values.
	providerName string
}

// noopProvider defines no-op provider fallback behavior.
type noopProvider struct{}

// Send submits one email request and returns provider message ids.
func (noopProvider) Send(ctx context.Context, request port.SendRequest) (string, error) {
	return "", errors.New("email provider is not configured")
}

// NewService creates email services.
func NewService(repository port.Repository, provider port.Provider, membershipStampers ...port.MembershipStamper) (*EmailService, error) {
	if repository == nil {
		return nil, ErrNilRepository
	}
	resolvedProvider := provider
	if resolvedProvider == nil {
		resolvedProvider = noopProvider{}
	}

	membershipStamper := port.MembershipStamper(port.NoopMembershipStamper{})
	if len(membershipStampers) > 0 && membershipStampers[0] != nil {
		membershipStamper = membershipStampers[0]
	}

	return &EmailService{repository: repository, provider: resolvedProvider, membershipStamper: membershipStamper, providerName: "ses"}, nil
}

// SetMembershipStamper configures optional membership stamp dependencies.
func (s *EmailService) SetMembershipStamper(stamper port.MembershipStamper) {
	if s == nil {
		return
	}
	if stamper == nil {
		s.membershipStamper = port.NoopMembershipStamper{}
		return
	}

	s.membershipStamper = stamper
}

// Send dispatches one email and tracks delivery status.
func (s *EmailService) Send(ctx context.Context, command SendCommand) (*domain.Delivery, error) {
	email := strings.ToLower(strings.TrimSpace(command.Email))
	if email == "" {
		return nil, domain.ErrInvalidEmail
	}
	subject := strings.TrimSpace(command.Subject)
	if subject == "" {
		return nil, domain.ErrInvalidSubject
	}

	delivery := &domain.Delivery{
		ID:             uuid.NewString(),
		ContactID:      strings.TrimSpace(command.ContactID),
		Email:          email,
		Subject:        subject,
		HTMLBody:       command.HTMLBody,
		TextBody:       command.TextBody,
		IdempotencyKey: strings.TrimSpace(command.IdempotencyKey),
		Provider:       s.providerName,
		Status:         domain.StatusPending,
	}
	if delivery.IdempotencyKey == "" {
		delivery.IdempotencyKey = delivery.ID
	}
	if err := s.repository.CreateDelivery(ctx, delivery); err != nil {
		return nil, fmt.Errorf("create delivery: %w", err)
	}
	_ = s.repository.AddStatusEntry(ctx, &domain.StatusEntry{ID: uuid.NewString(), DeliveryID: delivery.ID, Status: domain.StatusPending, OccurredAt: time.Now().UTC(), CreatedAt: time.Now().UTC()})

	providerMessageID, sendErr := s.provider.Send(ctx, port.SendRequest{
		To:             email,
		Subject:        subject,
		HTMLBody:       command.HTMLBody,
		TextBody:       command.TextBody,
		IdempotencyKey: delivery.IdempotencyKey,
	})
	if sendErr != nil {
		_ = s.repository.UpdateDeliveryStatus(ctx, delivery.ID, domain.StatusFailedRetryable, "")
		_ = s.repository.AddStatusEntry(ctx, &domain.StatusEntry{
			ID:         uuid.NewString(),
			DeliveryID: delivery.ID,
			Status:     domain.StatusFailedRetryable,
			Reason:     sendErr.Error(),
			OccurredAt: time.Now().UTC(),
			CreatedAt:  time.Now().UTC(),
		})
		delivery.Status = domain.StatusFailedRetryable
		return delivery, fmt.Errorf("send delivery: %w", sendErr)
	}

	_ = s.repository.UpdateDeliveryStatus(ctx, delivery.ID, domain.StatusSubmitted, providerMessageID)
	_ = s.repository.AddStatusEntry(ctx, &domain.StatusEntry{
		ID:         uuid.NewString(),
		DeliveryID: delivery.ID,
		Status:     domain.StatusSubmitted,
		OccurredAt: time.Now().UTC(),
		CreatedAt:  time.Now().UTC(),
	})
	delivery.Status = domain.StatusSubmitted
	delivery.ProviderMessageID = providerMessageID

	return delivery, nil
}

// HandleWebhook updates delivery status from provider webhook payloads.
func (s *EmailService) HandleWebhook(ctx context.Context, command WebhookCommand) error {
	providerMessageID := strings.TrimSpace(command.ProviderMessageID)
	if providerMessageID == "" {
		return domain.ErrNotFound
	}

	delivery, err := s.repository.GetByProviderMessageID(ctx, providerMessageID)
	if err != nil {
		return fmt.Errorf("load delivery by provider id: %w", err)
	}

	status := mapWebhookStatus(command.Status)
	if updateErr := s.repository.UpdateDeliveryStatus(ctx, delivery.ID, status, providerMessageID); updateErr != nil {
		return fmt.Errorf("update delivery from webhook: %w", updateErr)
	}
	if entryErr := s.repository.AddStatusEntry(ctx, &domain.StatusEntry{
		ID:         uuid.NewString(),
		DeliveryID: delivery.ID,
		Status:     status,
		Reason:     strings.TrimSpace(command.Reason),
		OccurredAt: time.Now().UTC(),
		CreatedAt:  time.Now().UTC(),
	}); entryErr != nil {
		return fmt.Errorf("append delivery status from webhook: %w", entryErr)
	}

	if status == domain.StatusComplained {
		_ = s.membershipStamper.OptOutByEmail(ctx, delivery.Email, "ses_complaint")
	}

	return nil
}

// Get retrieves one delivery by id.
func (s *EmailService) Get(ctx context.Context, deliveryID string) (*domain.Delivery, error) {
	trimmedID := strings.TrimSpace(deliveryID)
	if trimmedID == "" {
		return nil, domain.ErrNotFound
	}

	delivery, err := s.repository.GetByID(ctx, trimmedID)
	if err != nil {
		return nil, fmt.Errorf("get delivery: %w", err)
	}

	return delivery, nil
}

// mapWebhookStatus maps webhook status labels into domain statuses.
func mapWebhookStatus(value string) domain.DeliveryStatus {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "delivered":
		return domain.StatusDelivered
	case "bounce", "bounced":
		return domain.StatusBounced
	case "complaint", "complained":
		return domain.StatusComplained
	case "failed_permanent":
		return domain.StatusFailedPermanent
	default:
		return domain.StatusSubmitted
	}
}
