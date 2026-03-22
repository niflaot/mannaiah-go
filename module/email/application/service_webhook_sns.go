package application

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	neturl "net/url"
	"strings"
	"time"

	"github.com/google/uuid"
	"mannaiah/module/email/domain"
	"mannaiah/module/email/port"
)

const (
	// snsMessageTypeNotification defines SNS notification message-type values.
	snsMessageTypeNotification = "notification"
	// snsMessageTypeSubscriptionConfirmation defines SNS subscription-confirmation message-type values.
	snsMessageTypeSubscriptionConfirmation = "subscriptionconfirmation"
	// snsMessageTypeUnsubscribeConfirmation defines SNS unsubscribe-confirmation message-type values.
	snsMessageTypeUnsubscribeConfirmation = "unsubscribeconfirmation"
)

// snsSESNotification defines the SES notification payload nested inside SNS Message.
type snsSESNotification struct {
	// NotificationType defines SES notification type values.
	NotificationType string `json:"notificationType"`
	// EventType defines SES event-type values.
	EventType string `json:"eventType"`
	// Mail defines SES mail envelope values.
	Mail snsSESMail `json:"mail"`
	// Bounce defines SES bounce details.
	Bounce snsSESBounce `json:"bounce"`
	// Complaint defines SES complaint details.
	Complaint snsSESComplaint `json:"complaint"`
}

// snsSESMail defines SES mail envelope values.
type snsSESMail struct {
	// MessageID defines SES message identifier values.
	MessageID string `json:"messageId"`
	// Destination defines SES destination email values.
	Destination []string `json:"destination"`
}

// snsSESBounce defines SES bounce details.
type snsSESBounce struct {
	// BounceType defines SES bounce type values.
	BounceType string `json:"bounceType"`
	// BounceSubType defines SES bounce subtype values.
	BounceSubType string `json:"bounceSubType"`
	// BouncedRecipients defines bounced-recipient detail rows.
	BouncedRecipients []snsSESRecipient `json:"bouncedRecipients"`
}

// snsSESComplaint defines SES complaint details.
type snsSESComplaint struct {
	// ComplaintFeedbackType defines complaint feedback-type values.
	ComplaintFeedbackType string `json:"complaintFeedbackType"`
	// ComplainedRecipients defines complained-recipient detail rows.
	ComplainedRecipients []snsSESRecipient `json:"complainedRecipients"`
}

// snsSESRecipient defines recipient rows found in bounce/complaint payloads.
type snsSESRecipient struct {
	// EmailAddress defines recipient email values.
	EmailAddress string `json:"emailAddress"`
}

// handleSNSWebhook validates one SNS envelope and handles SES webhook payload values.
func (s *EmailService) handleSNSWebhook(ctx context.Context, command WebhookCommand) error {
	messageType := strings.ToLower(strings.TrimSpace(command.MessageType))
	if messageType == "" {
		return domain.ErrInvalidWebhookPayload
	}
	message := port.SNSMessage{
		Type:             strings.TrimSpace(command.MessageType),
		Message:          command.Message,
		MessageID:        strings.TrimSpace(command.MessageID),
		Subject:          strings.TrimSpace(command.Subject),
		Timestamp:        strings.TrimSpace(command.Timestamp),
		TopicARN:         strings.TrimSpace(command.TopicARN),
		Token:            strings.TrimSpace(command.Token),
		SubscribeURL:     strings.TrimSpace(command.SubscribeURL),
		SignatureVersion: strings.TrimSpace(command.SignatureVersion),
		Signature:        strings.TrimSpace(command.Signature),
		SigningCertURL:   strings.TrimSpace(command.SigningCertURL),
	}

	expectedTopicARN := strings.TrimSpace(s.expectedWebhookTopicARN)
	if expectedTopicARN != "" && !strings.EqualFold(message.TopicARN, expectedTopicARN) {
		return domain.ErrWebhookTopicMismatch
	}

	if s.snsMessageVerifier != nil {
		if verifyErr := s.snsMessageVerifier.Verify(ctx, message); verifyErr != nil {
			return fmt.Errorf("%w: %v", domain.ErrInvalidWebhookSignature, verifyErr)
		}
	}

	switch messageType {
	case snsMessageTypeSubscriptionConfirmation:
		if confirmErr := s.confirmSNSSubscription(ctx, message.SubscribeURL); confirmErr != nil {
			return confirmErr
		}
		return nil
	case snsMessageTypeNotification:
		return s.handleSESNotification(ctx, message.Message)
	case snsMessageTypeUnsubscribeConfirmation:
		return nil
	default:
		return domain.ErrInvalidWebhookPayload
	}
}

// confirmSNSSubscription confirms one SNS subscription URL.
func (s *EmailService) confirmSNSSubscription(ctx context.Context, subscribeURL string) error {
	trimmedSubscribeURL := strings.TrimSpace(subscribeURL)
	if trimmedSubscribeURL == "" {
		return domain.ErrInvalidWebhookPayload
	}
	parsedURL, parseErr := neturl.Parse(trimmedSubscribeURL)
	if parseErr != nil {
		return domain.ErrInvalidWebhookPayload
	}
	if !strings.EqualFold(parsedURL.Scheme, "https") {
		return domain.ErrInvalidWebhookPayload
	}
	host := strings.ToLower(strings.TrimSpace(parsedURL.Hostname()))
	if host == "" || !strings.HasSuffix(host, ".amazonaws.com") {
		return domain.ErrInvalidWebhookPayload
	}

	client := s.webhookHTTPClient
	if client == nil {
		client = &http.Client{Timeout: 5 * time.Second}
	}
	request, requestErr := http.NewRequestWithContext(ctx, http.MethodGet, trimmedSubscribeURL, nil)
	if requestErr != nil {
		return fmt.Errorf("%w: %v", domain.ErrWebhookSubscriptionConfirmationFailed, requestErr)
	}
	response, responseErr := client.Do(request)
	if responseErr != nil {
		return fmt.Errorf("%w: %v", domain.ErrWebhookSubscriptionConfirmationFailed, responseErr)
	}
	defer func() { _ = response.Body.Close() }()

	if response.StatusCode < 200 || response.StatusCode > 299 {
		return fmt.Errorf("%w: status %d", domain.ErrWebhookSubscriptionConfirmationFailed, response.StatusCode)
	}

	return nil
}

// handleSESNotification handles one SES notification payload embedded inside SNS Message.
func (s *EmailService) handleSESNotification(ctx context.Context, rawMessage string) error {
	if strings.TrimSpace(rawMessage) == "" {
		return domain.ErrInvalidWebhookPayload
	}

	payload := snsSESNotification{}
	if unmarshalErr := json.Unmarshal([]byte(rawMessage), &payload); unmarshalErr != nil {
		return domain.ErrInvalidWebhookPayload
	}

	providerMessageID := strings.TrimSpace(payload.Mail.MessageID)
	if providerMessageID == "" {
		return domain.ErrInvalidWebhookPayload
	}

	delivery, deliveryErr := s.repository.GetByProviderMessageID(ctx, providerMessageID)
	if errors.Is(deliveryErr, domain.ErrNotFound) {
		return nil
	}
	if deliveryErr != nil {
		return fmt.Errorf("load delivery by provider id: %w", deliveryErr)
	}

	eventType := strings.ToLower(strings.TrimSpace(payload.NotificationType))
	if eventType == "" {
		eventType = strings.ToLower(strings.TrimSpace(payload.EventType))
	}

	status := domain.StatusSubmitted
	shouldOptOut := false
	shouldSoftBounceRetry := false
	reason := strings.TrimSpace(eventType)
	source := "ses_notification"

	switch eventType {
	case "delivery":
		status = domain.StatusDelivered
	case "complaint":
		status = domain.StatusComplained
		source = "ses_complaint"
		shouldOptOut = true
		if feedbackType := strings.TrimSpace(payload.Complaint.ComplaintFeedbackType); feedbackType != "" {
			reason = "complaint_type=" + feedbackType
		}
	case "bounce":
		bounceType := strings.ToLower(strings.TrimSpace(payload.Bounce.BounceType))
		bounceSubType := strings.TrimSpace(payload.Bounce.BounceSubType)
		reasonParts := make([]string, 0, 2)
		if bounceType != "" {
			reasonParts = append(reasonParts, "bounce_type="+bounceType)
		}
		if bounceSubType != "" {
			reasonParts = append(reasonParts, "bounce_sub_type="+bounceSubType)
		}
		if len(reasonParts) > 0 {
			reason = strings.Join(reasonParts, "; ")
		}
		if bounceType == "transient" {
			status = domain.StatusFailedRetryable
			shouldSoftBounceRetry = true
			source = "ses_bounce_transient"
		} else {
			status = domain.StatusBounced
			shouldOptOut = true
			source = "ses_bounce_permanent"
		}
	case "reject", "renderingfailure":
		status = domain.StatusFailedPermanent
	default:
		status = mapWebhookStatus(eventType)
	}

	if updateErr := s.repository.UpdateDeliveryStatus(ctx, delivery.ID, status, providerMessageID); updateErr != nil {
		return fmt.Errorf("update delivery from webhook: %w", updateErr)
	}
	if entryErr := s.repository.AddStatusEntry(ctx, &domain.StatusEntry{
		ID:         uuid.NewString(),
		DeliveryID: delivery.ID,
		Status:     status,
		Reason:     reason,
		OccurredAt: time.Now().UTC(),
		CreatedAt:  time.Now().UTC(),
	}); entryErr != nil {
		return fmt.Errorf("append delivery status from webhook: %w", entryErr)
	}

	if shouldOptOut {
		recipientEmail := resolveNotificationRecipientEmail(payload, delivery.Email)
		if recipientEmail != "" {
			_ = s.membershipStamper.OptOutByEmail(ctx, recipientEmail, source)
		}
	}

	if shouldSoftBounceRetry && s.softBounceMaxRetries > 0 {
		retryCount, retryCountErr := s.repository.CountStatusEntries(ctx, delivery.ID, domain.StatusFailedRetryable)
		if retryCountErr == nil && retryCount <= int64(s.softBounceMaxRetries) {
			s.retrySoftBounce(delivery, reason)
		}
	}

	return nil
}

// resolveNotificationRecipientEmail resolves recipient email values from SES notification payloads.
func resolveNotificationRecipientEmail(payload snsSESNotification, fallback string) string {
	for _, recipient := range payload.Bounce.BouncedRecipients {
		email := strings.ToLower(strings.TrimSpace(recipient.EmailAddress))
		if email != "" {
			return email
		}
	}
	for _, recipient := range payload.Complaint.ComplainedRecipients {
		email := strings.ToLower(strings.TrimSpace(recipient.EmailAddress))
		if email != "" {
			return email
		}
	}
	for _, destination := range payload.Mail.Destination {
		email := strings.ToLower(strings.TrimSpace(destination))
		if email != "" {
			return email
		}
	}

	return strings.ToLower(strings.TrimSpace(fallback))
}

// retrySoftBounce retries one transiently bounced delivery after a configured delay.
func (s *EmailService) retrySoftBounce(delivery *domain.Delivery, reason string) {
	if s == nil || delivery == nil {
		return
	}
	delay := s.softBounceRetryDelay
	clone := *delivery

	go func() {
		if delay > 0 {
			time.Sleep(delay)
		}

		retryContext, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		htmlBodyToSend := clone.HTMLBody
		if s.trackingBaseURL != "" && htmlBodyToSend != "" {
			pixelURL := s.trackingBaseURL + "/email/track/open/" + clone.ID
			htmlBodyToSend = injectOpenTrackingPixel(htmlBodyToSend, pixelURL)
		}

		providerMessageID, sendErr := s.provider.Send(retryContext, port.SendRequest{
			To:             clone.Email,
			Subject:        clone.Subject,
			HTMLBody:       htmlBodyToSend,
			TextBody:       clone.TextBody,
			IdempotencyKey: clone.IdempotencyKey + ":soft-bounce-retry:" + uuid.NewString(),
		})
		if sendErr != nil {
			_ = s.repository.UpdateDeliveryStatus(retryContext, clone.ID, domain.StatusFailedRetryable, "")
			_ = s.repository.AddStatusEntry(retryContext, &domain.StatusEntry{
				ID:         uuid.NewString(),
				DeliveryID: clone.ID,
				Status:     domain.StatusFailedRetryable,
				Reason:     "soft_bounce_retry_failed: " + strings.TrimSpace(sendErr.Error()),
				OccurredAt: time.Now().UTC(),
				CreatedAt:  time.Now().UTC(),
			})
			return
		}

		_ = s.repository.UpdateDeliveryStatus(retryContext, clone.ID, domain.StatusSubmitted, providerMessageID)
		_ = s.repository.AddStatusEntry(retryContext, &domain.StatusEntry{
			ID:         uuid.NewString(),
			DeliveryID: clone.ID,
			Status:     domain.StatusSubmitted,
			Reason:     "soft_bounce_retry_submitted: " + strings.TrimSpace(reason),
			OccurredAt: time.Now().UTC(),
			CreatedAt:  time.Now().UTC(),
		})
	}()
}
