package http

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	corehttp "mannaiah/module/core/http"
	shopifycontactservice "mannaiah/module/shopify/application/contact/service"
	shopifyorderservice "mannaiah/module/shopify/application/order/service"
	shopifyport "mannaiah/module/shopify/port"
)

var (
	// ErrNilContactService is returned when a nil Shopify contact sync service is provided.
	ErrNilContactService = errors.New("shopify contacts service must not be nil")
	// ErrNilOrderService is returned when a nil Shopify order sync service is provided.
	ErrNilOrderService = errors.New("shopify orders service must not be nil")
	// ErrNilProcessor is returned when a nil webhook processor is provided.
	ErrNilProcessor = errors.New("shopify webhook processor must not be nil")
	// ErrNilDeliveries is returned when a nil webhook delivery repository is provided.
	ErrNilDeliveries = errors.New("shopify webhook delivery repository must not be nil")
	// ErrInvalidWebhookSignature is returned when a Shopify webhook signature is invalid.
	ErrInvalidWebhookSignature = errors.New("shopify webhook signature is invalid")
	// ErrWebhookDeliveryIDRequired is returned when a webhook delivery id is missing.
	ErrWebhookDeliveryIDRequired = errors.New("shopify webhook delivery id is required")
	// ErrWebhookTopicRequired is returned when a webhook topic is missing.
	ErrWebhookTopicRequired = errors.New("shopify webhook topic is required")
	// ErrWebhookPayloadIDRequired is returned when a webhook payload id is missing.
	ErrWebhookPayloadIDRequired = errors.New("shopify webhook payload id is required")
)

// Authorizer defines authentication and authorization behavior required by Shopify manual sync endpoints.
type Authorizer interface {
	// Require authenticates and authorizes requests using required permissions.
	Require(ctx context.Context, authorizationHeader string, requiredPermissions ...string) error
	// IsUnauthorized reports authentication errors.
	IsUnauthorized(err error) bool
	// IsForbidden reports authorization errors.
	IsForbidden(err error) bool
}

// Handler defines HTTP route handlers for Shopify integration endpoints.
type Handler struct {
	// contactsService defines Shopify contact sync dependencies.
	contactsService shopifycontactservice.Service
	// ordersService defines Shopify order sync dependencies.
	ordersService shopifyorderservice.Service
	// processor defines asynchronous webhook processing dependencies.
	processor WebhookProcessor
	// deliveries defines webhook idempotency dependencies.
	deliveries shopifyport.WebhookDeliveryRepository
	// webhookSecret defines Shopify webhook secret values.
	webhookSecret string
	// authorizer defines optional auth dependencies for protected endpoints.
	authorizer Authorizer
}

// NewHandler creates Shopify HTTP handler sets.
func NewHandler(contactsService shopifycontactservice.Service, ordersService shopifyorderservice.Service, processor WebhookProcessor, deliveries shopifyport.WebhookDeliveryRepository, webhookSecret string, authorizers ...Authorizer) (*Handler, error) {
	if contactsService == nil {
		return nil, ErrNilContactService
	}
	if ordersService == nil {
		return nil, ErrNilOrderService
	}
	if processor == nil {
		return nil, ErrNilProcessor
	}
	if deliveries == nil {
		return nil, ErrNilDeliveries
	}

	var authorizer Authorizer
	if len(authorizers) > 0 {
		authorizer = authorizers[0]
	}

	return &Handler{
		contactsService: contactsService,
		ordersService:   ordersService,
		processor:       processor,
		deliveries:      deliveries,
		webhookSecret:   strings.TrimSpace(webhookSecret),
		authorizer:      authorizer,
	}, nil
}

// SetAuthorizer configures endpoint authentication dependencies.
func (h *Handler) SetAuthorizer(authorizer Authorizer) {
	if h == nil {
		return
	}

	h.authorizer = authorizer
}

// RegisterRoutes registers Shopify integration routes.
func (h *Handler) RegisterRoutes(router corehttp.Router) {
	router.Post("/shopify/sync/contacts", h.protect("contact:sync", h.syncContacts))
	router.Post("/shopify/sync/orders", h.protect("order:sync", h.syncOrders))
	router.Post("/shopify/webhooks", h.handleWebhook)
}

func (h *Handler) syncContacts(ctx corehttp.Context) error {
	targetID := strings.TrimSpace(ctx.Query("id", ""))
	if targetID == "" {
		return h.mapError(shopifycontactservice.ErrInvalidCustomerID)
	}

	summary, err := h.contactsService.SyncContactByID(ctx.Context(), "manual", targetID)
	if err != nil {
		return h.mapError(err)
	}

	return ctx.Status(200).JSON(summary)
}

func (h *Handler) syncOrders(ctx corehttp.Context) error {
	targetID := strings.TrimSpace(ctx.Query("id", ""))
	if targetID == "" {
		return h.mapError(shopifyorderservice.ErrInvalidOrderID)
	}

	summary, err := h.ordersService.SyncOrderByID(ctx.Context(), "manual", targetID)
	if err != nil {
		return h.mapError(err)
	}

	return ctx.Status(200).JSON(summary)
}

func (h *Handler) handleWebhook(ctx corehttp.Context) error {
	body := ctx.Body()
	if !VerifyWebhookSignature(h.webhookSecret, body, ctx.GetHeader("X-Shopify-Hmac-Sha256")) {
		return h.mapError(ErrInvalidWebhookSignature)
	}

	deliveryID := strings.TrimSpace(ctx.GetHeader("X-Shopify-Webhook-Id"))
	if deliveryID == "" {
		return h.mapError(ErrWebhookDeliveryIDRequired)
	}
	topic := strings.TrimSpace(ctx.GetHeader("X-Shopify-Topic"))
	if topic == "" {
		return h.mapError(ErrWebhookTopicRequired)
	}

	created, err := h.deliveries.CreateDeliveryIfAbsent(ctx.Context(), deliveryID, topic)
	if err != nil {
		return h.mapError(err)
	}
	if !created {
		return ctx.SendStatus(200)
	}

	shopifyID, err := extractWebhookEntityID(body)
	if err != nil {
		return h.mapError(err)
	}
	if enqueueErr := h.processor.Enqueue(ctx.Context(), topic, shopifyID); enqueueErr != nil {
		return h.mapError(enqueueErr)
	}

	return ctx.SendStatus(200)
}

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

func (h *Handler) mapError(err error) error {
	if h != nil && h.authorizer != nil {
		if h.authorizer.IsUnauthorized(err) {
			return corehttp.NewAppError(401, "unauthorized", err)
		}
		if h.authorizer.IsForbidden(err) {
			return corehttp.NewAppError(403, "forbidden", err)
		}
	}
	if errors.Is(err, ErrInvalidWebhookSignature) {
		return corehttp.NewAppError(401, "shopify_invalid_webhook_signature", err)
	}
	if errors.Is(err, ErrWebhookDeliveryIDRequired) || errors.Is(err, ErrWebhookTopicRequired) || errors.Is(err, ErrWebhookPayloadIDRequired) {
		return corehttp.NewAppError(400, "invalid_shopify_webhook_payload", err)
	}
	if errors.Is(err, ErrProcessorClosed) {
		return corehttp.NewAppError(503, "shopify_webhook_processor_unavailable", err)
	}
	if errors.Is(err, shopifycontactservice.ErrSyncDisabled) {
		return corehttp.NewAppError(503, "shopify_contacts_sync_disabled", err)
	}
	if errors.Is(err, shopifycontactservice.ErrInvalidCustomerID) {
		return corehttp.NewAppError(400, "invalid_shopify_customer_id", err)
	}
	if errors.Is(err, shopifycontactservice.ErrContactNotFound) {
		return corehttp.NewAppError(404, "shopify_contact_not_found", err)
	}
	if errors.Is(err, shopifyorderservice.ErrSyncDisabled) {
		return corehttp.NewAppError(503, "shopify_orders_sync_disabled", err)
	}
	if errors.Is(err, shopifyorderservice.ErrInvalidOrderID) {
		return corehttp.NewAppError(400, "invalid_shopify_order_id", err)
	}
	if errors.Is(err, shopifyorderservice.ErrOrderNotFound) {
		return corehttp.NewAppError(404, "shopify_order_not_found", err)
	}
	if errors.Is(err, shopifycontactservice.ErrIntegrationUnavailable) || errors.Is(err, shopifyorderservice.ErrIntegrationUnavailable) {
		return corehttp.NewAppError(503, "shopify_integration_unavailable", err)
	}

	return corehttp.NewAppError(500, "internal_server_error", err)
}

func extractWebhookEntityID(body []byte) (string, error) {
	var payload struct {
		ID json.Number `json:"id"`
	}
	decoder := json.NewDecoder(strings.NewReader(string(body)))
	decoder.UseNumber()
	if err := decoder.Decode(&payload); err != nil {
		return "", ErrWebhookPayloadIDRequired
	}
	resolved := strings.TrimSpace(payload.ID.String())
	if resolved == "" {
		return "", ErrWebhookPayloadIDRequired
	}

	return resolved, nil
}
