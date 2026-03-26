package application

import (
	"context"
	"errors"
	"fmt"
	"net/http"
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
	// MessageType defines optional SNS message-type values.
	MessageType string
	// Message defines optional SNS embedded-message values.
	Message string
	// MessageID defines optional SNS message identifier values.
	MessageID string
	// Subject defines optional SNS subject values.
	Subject string
	// Timestamp defines optional SNS timestamp values.
	Timestamp string
	// TopicARN defines optional SNS topic arn values.
	TopicARN string
	// Token defines optional SNS subscription token values.
	Token string
	// SubscribeURL defines optional SNS subscription confirmation URL values.
	SubscribeURL string
	// SignatureVersion defines optional SNS signature version values.
	SignatureVersion string
	// Signature defines optional SNS signature values.
	Signature string
	// SigningCertURL defines optional SNS signing certificate URL values.
	SigningCertURL string
}

// Service defines email use-case behavior.
type Service interface {
	// Send dispatches one email and tracks delivery status.
	Send(ctx context.Context, command SendCommand) (*domain.Delivery, error)
	// HandleWebhook updates delivery status from provider webhook payloads.
	HandleWebhook(ctx context.Context, command WebhookCommand) error
	// Get retrieves one delivery by id.
	Get(ctx context.Context, deliveryID string) (*domain.Delivery, error)
	// ListByEmail retrieves deliveries sent to one recipient email.
	ListByEmail(ctx context.Context, email string) ([]*domain.Delivery, error)
	// TrackOpen records an open event for a delivery identified by deliveryID.
	TrackOpen(ctx context.Context, deliveryID string) error
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
	// trackingBaseURL defines the public base URL for open-tracking pixel injection.
	// Empty string disables pixel injection.
	trackingBaseURL string
	// snsMessageVerifier defines optional SNS signature verification dependencies.
	snsMessageVerifier port.SNSMessageVerifier
	// expectedWebhookTopicARN defines optional topic ARN values expected from SNS webhook messages.
	expectedWebhookTopicARN string
	// webhookHTTPClient defines HTTP dependencies for SNS subscription confirmation requests.
	webhookHTTPClient *http.Client
	// softBounceRetryDelay defines retry delay values for transient bounce handling.
	softBounceRetryDelay time.Duration
	// softBounceMaxRetries defines max retry attempts for transient bounce handling.
	softBounceMaxRetries int
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

	return &EmailService{
		repository:           repository,
		provider:             resolvedProvider,
		membershipStamper:    membershipStamper,
		providerName:         "ses",
		snsMessageVerifier:   port.NoopSNSMessageVerifier{},
		webhookHTTPClient:    &http.Client{Timeout: 5 * time.Second},
		softBounceRetryDelay: 5 * time.Minute,
		softBounceMaxRetries: 1,
	}, nil
}

// SetTrackingBaseURL configures the public base URL used for open-tracking pixel injection.
// An empty value disables pixel injection.
func (s *EmailService) SetTrackingBaseURL(baseURL string) {
	if s == nil {
		return
	}

	s.trackingBaseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
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

// SetSNSMessageVerifier configures SNS signature verification dependencies.
func (s *EmailService) SetSNSMessageVerifier(verifier port.SNSMessageVerifier) {
	if s == nil {
		return
	}
	if verifier == nil {
		s.snsMessageVerifier = port.NoopSNSMessageVerifier{}
		return
	}

	s.snsMessageVerifier = verifier
}

// SetWebhookPolicy configures SNS topic checks and transient-bounce retry behavior.
func (s *EmailService) SetWebhookPolicy(expectedTopicARN string, softBounceRetryDelay time.Duration, softBounceMaxRetries int) {
	if s == nil {
		return
	}

	s.expectedWebhookTopicARN = strings.TrimSpace(expectedTopicARN)
	if softBounceRetryDelay < 0 {
		softBounceRetryDelay = 0
	}
	s.softBounceRetryDelay = softBounceRetryDelay
	if softBounceMaxRetries < 0 {
		softBounceMaxRetries = 0
	}
	s.softBounceMaxRetries = softBounceMaxRetries
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

	htmlBodyToSend := command.HTMLBody
	if s.trackingBaseURL != "" && htmlBodyToSend != "" {
		pixelURL := s.trackingBaseURL + "/email/track/open/" + delivery.ID
		htmlBodyToSend = injectOpenTrackingPixel(htmlBodyToSend, pixelURL)
	}

	providerMessageID, sendErr := s.provider.Send(ctx, port.SendRequest{
		To:             email,
		Subject:        subject,
		HTMLBody:       htmlBodyToSend,
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
	if strings.TrimSpace(command.MessageType) != "" {
		return s.handleSNSWebhook(ctx, command)
	}

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
	if status == domain.StatusBounced {
		_ = s.membershipStamper.OptOutByEmail(ctx, delivery.Email, "ses_bounce_permanent")
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

// ListByEmail retrieves deliveries sent to one recipient email.
func (s *EmailService) ListByEmail(ctx context.Context, email string) ([]*domain.Delivery, error) {
	normalizedEmail := strings.ToLower(strings.TrimSpace(email))
	if normalizedEmail == "" {
		return nil, domain.ErrInvalidEmail
	}

	deliveries, err := s.repository.ListByEmail(ctx, normalizedEmail)
	if err != nil {
		return nil, fmt.Errorf("list deliveries by email: %w", err)
	}

	return deliveries, nil
}

// TrackOpen records an open event for a delivery identified by deliveryID.
// It is fail-open: if the delivery does not exist the error is silently dropped by callers.
func (s *EmailService) TrackOpen(ctx context.Context, deliveryID string) error {
	trimmedID := strings.TrimSpace(deliveryID)
	if trimmedID == "" {
		return domain.ErrNotFound
	}

	if statusErr := s.repository.UpdateDeliveryStatus(ctx, trimmedID, domain.StatusOpened, ""); statusErr != nil {
		return statusErr
	}

	return s.repository.AddStatusEntry(ctx, &domain.StatusEntry{
		ID:         uuid.NewString(),
		DeliveryID: trimmedID,
		Status:     domain.StatusOpened,
		OccurredAt: time.Now().UTC(),
		CreatedAt:  time.Now().UTC(),
	})
}

// injectOpenTrackingPixel appends a 1×1 transparent tracking pixel before </body>.
// Falls back to appending at the end of html when no </body> tag is found.
func injectOpenTrackingPixel(html string, pixelURL string) string {
	pixel := `<img src="` + pixelURL + `" width="1" height="1" style="display:none;border:0;" alt="" />`
	if idx := strings.Index(strings.ToLower(html), "</body>"); idx >= 0 {
		return html[:idx] + pixel + html[idx:]
	}

	return html + pixel
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
