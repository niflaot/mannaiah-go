package http

import (
	"context"
	"errors"
	"strings"

	corehttp "mannaiah/module/core/http"
	"mannaiah/module/email/application"
	"mannaiah/module/email/domain"
)

var (
	// ErrNilService is returned when nil service dependencies are provided.
	ErrNilService = errors.New("email service must not be nil")
)

// Authorizer defines authentication and authorization behavior required by email endpoints.
type Authorizer interface {
	// Require authenticates and authorizes requests using required permissions.
	Require(ctx context.Context, authorizationHeader string, requiredPermissions ...string) error
	// IsUnauthorized reports authentication errors.
	IsUnauthorized(err error) bool
	// IsForbidden reports authorization errors.
	IsForbidden(err error) bool
}

// Service defines email use-case behavior required by HTTP handlers.
type Service interface {
	// Send dispatches one email and tracks delivery status.
	Send(ctx context.Context, command application.SendCommand) (*domain.Delivery, error)
	// HandleWebhook updates delivery status from provider webhook payloads.
	HandleWebhook(ctx context.Context, command application.WebhookCommand) error
	// Get retrieves one delivery by id.
	Get(ctx context.Context, deliveryID string) (*domain.Delivery, error)
	// TrackOpen records an open event for a delivery identified by deliveryID.
	TrackOpen(ctx context.Context, deliveryID string) error
}

// Handler defines HTTP route handlers for email endpoints.
type Handler struct {
	// service defines email use-case dependencies.
	service Service
	// authorizer defines optional auth dependencies.
	authorizer Authorizer
}

// sendRequest defines email send request payload values.
type sendRequest struct {
	// ContactID defines optional contact identifier values.
	ContactID string `json:"contactId"`
	// Email defines recipient email values.
	Email string `json:"email"`
	// Subject defines subject values.
	Subject string `json:"subject"`
	// HTMLBody defines html payload values.
	HTMLBody string `json:"htmlBody"`
	// TextBody defines text payload values.
	TextBody string `json:"textBody"`
	// IdempotencyKey defines idempotency values.
	IdempotencyKey string `json:"idempotencyKey"`
}

// webhookRequest defines webhook request payload values.
type webhookRequest struct {
	// ProviderMessageID defines provider message identifier values.
	ProviderMessageID string `json:"providerMessageId"`
	// Status defines provider status values.
	Status string `json:"status"`
	// Reason defines optional reason values.
	Reason string `json:"reason"`
	// Email defines optional recipient email values.
	Email string `json:"email"`
	// MessageType defines optional SNS message-type values.
	MessageType string `json:"Type"`
	// Message defines optional SNS embedded-message values.
	Message string `json:"Message"`
	// MessageID defines optional SNS message id values.
	MessageID string `json:"MessageId"`
	// Subject defines optional SNS subject values.
	Subject string `json:"Subject"`
	// Timestamp defines optional SNS timestamp values.
	Timestamp string `json:"Timestamp"`
	// TopicARN defines optional SNS topic arn values.
	TopicARN string `json:"TopicArn"`
	// Token defines optional SNS subscription token values.
	Token string `json:"Token"`
	// SubscribeURL defines optional SNS subscription confirmation URL values.
	SubscribeURL string `json:"SubscribeURL"`
	// SignatureVersion defines optional SNS signature version values.
	SignatureVersion string `json:"SignatureVersion"`
	// Signature defines optional SNS signature values.
	Signature string `json:"Signature"`
	// SigningCertURL defines optional SNS signing certificate URL values.
	SigningCertURL string `json:"SigningCertURL"`
}

// NewHandler creates email HTTP handlers.
func NewHandler(service Service, authorizers ...Authorizer) (*Handler, error) {
	if service == nil {
		return nil, ErrNilService
	}

	var authorizer Authorizer
	if len(authorizers) > 0 {
		authorizer = authorizers[0]
	}

	return &Handler{service: service, authorizer: authorizer}, nil
}

// SetAuthorizer configures endpoint authentication and permission dependencies.
func (h *Handler) SetAuthorizer(authorizer Authorizer) {
	if h == nil {
		return
	}

	h.authorizer = authorizer
}

// transparentGIF is a 1×1 transparent GIF served by the open-tracking endpoint.
var transparentGIF = []byte{
	0x47, 0x49, 0x46, 0x38, 0x39, 0x61, 0x01, 0x00, 0x01, 0x00, 0x80, 0x00, 0x00,
	0xFF, 0xFF, 0xFF, 0x00, 0x00, 0x00, 0x21, 0xF9, 0x04, 0x01, 0x00, 0x00, 0x00,
	0x00, 0x2C, 0x00, 0x00, 0x00, 0x00, 0x01, 0x00, 0x01, 0x00, 0x00, 0x02, 0x02,
	0x44, 0x01, 0x00, 0x3B,
}

// RegisterRoutes registers email routes.
func (h *Handler) RegisterRoutes(router corehttp.Router) {
	router.Post("/email/send", h.protect("marketing:manage", h.send))
	router.Get("/email/deliveries/:id", h.protect("marketing:manage", h.delivery))
	router.Post("/email/webhooks/ses", h.webhook)
	router.Get("/email/track/open/:id", h.trackOpen)
}

// send handles email send requests.
func (h *Handler) send(ctx corehttp.Context) error {
	request := sendRequest{}
	if err := ctx.BodyParser(&request); err != nil {
		return corehttp.NewAppError(400, "invalid_payload", err)
	}

	delivery, err := h.service.Send(ctx.Context(), application.SendCommand{
		ContactID:      strings.TrimSpace(request.ContactID),
		Email:          strings.TrimSpace(request.Email),
		Subject:        strings.TrimSpace(request.Subject),
		HTMLBody:       request.HTMLBody,
		TextBody:       request.TextBody,
		IdempotencyKey: strings.TrimSpace(request.IdempotencyKey),
	})
	if err != nil {
		return h.mapError(err)
	}

	return ctx.Status(202).JSON(delivery)
}

// webhook handles SES webhook requests.
func (h *Handler) webhook(ctx corehttp.Context) error {
	request := webhookRequest{}
	if err := ctx.BodyParser(&request); err != nil {
		return corehttp.NewAppError(400, "invalid_payload", err)
	}
	if strings.TrimSpace(request.MessageType) == "" {
		request.MessageType = strings.TrimSpace(ctx.GetHeader("x-amz-sns-message-type"))
	}

	if err := h.service.HandleWebhook(ctx.Context(), application.WebhookCommand{
		ProviderMessageID: strings.TrimSpace(request.ProviderMessageID),
		Status:            strings.TrimSpace(request.Status),
		Reason:            strings.TrimSpace(request.Reason),
		Email:             strings.TrimSpace(request.Email),
		MessageType:       strings.TrimSpace(request.MessageType),
		Message:           request.Message,
		MessageID:         strings.TrimSpace(request.MessageID),
		Subject:           strings.TrimSpace(request.Subject),
		Timestamp:         strings.TrimSpace(request.Timestamp),
		TopicARN:          strings.TrimSpace(request.TopicARN),
		Token:             strings.TrimSpace(request.Token),
		SubscribeURL:      strings.TrimSpace(request.SubscribeURL),
		SignatureVersion:  strings.TrimSpace(request.SignatureVersion),
		Signature:         strings.TrimSpace(request.Signature),
		SigningCertURL:    strings.TrimSpace(request.SigningCertURL),
	}); err != nil {
		return h.mapError(err)
	}

	return ctx.Status(200).JSON(map[string]string{"status": "ok"})
}

// delivery handles delivery lookup requests.
func (h *Handler) delivery(ctx corehttp.Context) error {
	delivery, err := h.service.Get(ctx.Context(), strings.TrimSpace(ctx.Params("id")))
	if err != nil {
		return h.mapError(err)
	}

	return ctx.Status(200).JSON(delivery)
}

// trackOpen handles open-tracking pixel requests. No authentication is required.
// It records an open event for the delivery and responds with a 1×1 transparent GIF.
func (h *Handler) trackOpen(ctx corehttp.Context) error {
	deliveryID := strings.TrimSpace(ctx.Params("id"))
	if deliveryID != "" {
		_ = h.service.TrackOpen(ctx.Context(), deliveryID)
	}
	ctx.SetHeader("Content-Type", "image/gif")
	ctx.SetHeader("Cache-Control", "no-cache, no-store, must-revalidate")

	return ctx.Status(200).SendBytes(transparentGIF)
}

// protect wraps endpoint handlers with optional authentication and permission checks.
func (h *Handler) protect(permission string, next corehttp.Handler) corehttp.Handler {
	if h == nil || h.authorizer == nil {
		return next
	}

	return func(ctx corehttp.Context) error {
		err := h.authorizer.Require(ctx.Context(), ctx.GetHeader("Authorization"), permission)
		if err != nil {
			return h.mapError(err)
		}

		return next(ctx)
	}
}

// mapError maps app/auth errors to HTTP-layer app errors.
func (h *Handler) mapError(err error) error {
	if h != nil && h.authorizer != nil {
		if h.authorizer.IsUnauthorized(err) {
			return corehttp.NewAppError(401, "unauthorized", err)
		}
		if h.authorizer.IsForbidden(err) {
			return corehttp.NewAppError(403, "forbidden", err)
		}
	}
	if errors.Is(err, domain.ErrInvalidEmail) || errors.Is(err, domain.ErrInvalidSubject) {
		return corehttp.NewAppError(400, "invalid_payload", err)
	}
	if errors.Is(err, domain.ErrInvalidWebhookPayload) {
		return corehttp.NewAppError(400, "invalid_webhook_payload", err)
	}
	if errors.Is(err, domain.ErrInvalidWebhookSignature) {
		return corehttp.NewAppError(401, "invalid_webhook_signature", err)
	}
	if errors.Is(err, domain.ErrWebhookTopicMismatch) {
		return corehttp.NewAppError(403, "webhook_topic_mismatch", err)
	}
	if errors.Is(err, domain.ErrWebhookSubscriptionConfirmationFailed) {
		return corehttp.NewAppError(503, "webhook_subscription_confirmation_failed", err)
	}
	if errors.Is(err, domain.ErrNotFound) {
		return corehttp.NewAppError(404, "email_delivery_not_found", err)
	}

	return corehttp.NewAppError(500, "internal_server_error", err)
}
