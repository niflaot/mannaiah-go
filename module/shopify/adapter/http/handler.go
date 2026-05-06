package http

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	contactsapplication "mannaiah/module/contacts/application"
	corehttp "mannaiah/module/core/http"
	ordersapplication "mannaiah/module/orders/application"
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
	// ErrNilSyncLinks is returned when a nil sync-link repository is provided.
	ErrNilSyncLinks = errors.New("shopify sync link repository must not be nil")
	// ErrNilDeliveries is returned when a nil webhook delivery repository is provided.
	ErrNilDeliveries = errors.New("shopify webhook delivery repository must not be nil")
	// ErrNilInstallations is returned when a nil installation repository is provided.
	ErrNilInstallations = errors.New("shopify installation repository must not be nil")
	// ErrNilInstallationResolver is returned when a nil installation resolver is provided.
	ErrNilInstallationResolver = errors.New("shopify installation resolver must not be nil")
	// ErrNilOAuthClient is returned when a nil OAuth client is provided.
	ErrNilOAuthClient = errors.New("shopify oauth client must not be nil")
	// ErrNilContactLookupService is returned when a nil contact lookup service is provided.
	ErrNilContactLookupService = errors.New("shopify contact lookup service must not be nil")
	// ErrNilOrderLookupService is returned when a nil order lookup service is provided.
	ErrNilOrderLookupService = errors.New("shopify order lookup service must not be nil")
	// ErrInvalidWebhookSignature is returned when a Shopify webhook signature is invalid.
	ErrInvalidWebhookSignature = errors.New("shopify webhook signature is invalid")
	// ErrWebhookDeliveryIDRequired is returned when a webhook delivery id is missing.
	ErrWebhookDeliveryIDRequired = errors.New("shopify webhook delivery id is required")
	// ErrWebhookTopicRequired is returned when a webhook topic is missing.
	ErrWebhookTopicRequired = errors.New("shopify webhook topic is required")
	// ErrWebhookPayloadIDRequired is returned when a webhook payload id is missing.
	ErrWebhookPayloadIDRequired = errors.New("shopify webhook payload id is required")
	// ErrWebhookShopDomainRequired is returned when a Shopify webhook shop-domain header is missing.
	ErrWebhookShopDomainRequired = errors.New("shopify webhook shop domain is required")
	// ErrInvalidShopDomain is returned when Shopify shop-domain values are invalid.
	ErrInvalidShopDomain = errors.New("shopify shop domain is invalid")
)

const extensionShopDomainLocal = "shopify.shop_domain"

// Authorizer defines authentication and authorization behavior required by Shopify manual sync endpoints.
type Authorizer interface {
	// Require authenticates and authorizes requests using required permissions.
	Require(ctx context.Context, authorizationHeader string, requiredPermissions ...string) error
	// IsUnauthorized reports authentication errors.
	IsUnauthorized(err error) bool
	// IsForbidden reports authorization errors.
	IsForbidden(err error) bool
}

// OAuthClient defines Shopify OAuth and webhook-registration behavior required by HTTP handlers.
type OAuthClient interface {
	// ExchangeAuthorizationCode exchanges one OAuth authorization code for a permanent offline token.
	ExchangeAuthorizationCode(ctx context.Context, shopDomain string, code string) (string, string, error)
	// RegisterWebhooks registers required webhook topics for one Shopify installation.
	RegisterWebhooks(ctx context.Context, shopDomain string, accessToken string, address string) error
}

type manualSyncRequest struct {
	ID         string `json:"id"`
	ShopDomain string `json:"shopDomain"`
}

// Handler defines HTTP route handlers for Shopify integration endpoints.
type Handler struct {
	// contactsService defines Shopify contact sync dependencies.
	contactsService shopifycontactservice.Service
	// ordersService defines Shopify order sync dependencies.
	ordersService shopifyorderservice.Service
	// processor defines asynchronous webhook processing dependencies.
	processor WebhookProcessor
	// links defines Shopify sync link persistence dependencies.
	links shopifyport.SyncLinkRepository
	// deliveries defines webhook idempotency dependencies.
	deliveries shopifyport.WebhookDeliveryRepository
	// installations defines Shopify installation persistence dependencies.
	installations shopifyport.InstallationRepository
	// installationResolver defines cached Shopify installation lookup behavior.
	installationResolver shopifyport.InstallationResolver
	// oauthClient defines Shopify OAuth token exchange and webhook registration behavior.
	oauthClient OAuthClient
	// contactsLookup defines mainstream contact lookup dependencies for extension routes.
	contactsLookup contactsapplication.Service
	// ordersLookup defines mainstream order lookup dependencies for extension routes.
	ordersLookup ordersapplication.Service
	// clientID defines Shopify OAuth client identifier values.
	clientID string
	// clientSecret defines Shopify OAuth client secret values.
	clientSecret string
	// authorizer defines optional auth dependencies for protected endpoints.
	authorizer Authorizer
}

// NewHandler creates Shopify HTTP handler sets.
func NewHandler(
	contactsService shopifycontactservice.Service,
	ordersService shopifyorderservice.Service,
	processor WebhookProcessor,
	links shopifyport.SyncLinkRepository,
	deliveries shopifyport.WebhookDeliveryRepository,
	installations shopifyport.InstallationRepository,
	installationResolver shopifyport.InstallationResolver,
	oauthClient OAuthClient,
	contactsLookup contactsapplication.Service,
	ordersLookup ordersapplication.Service,
	clientID string,
	clientSecret string,
	authorizers ...Authorizer,
) (*Handler, error) {
	if contactsService == nil {
		return nil, ErrNilContactService
	}
	if ordersService == nil {
		return nil, ErrNilOrderService
	}
	if processor == nil {
		return nil, ErrNilProcessor
	}
	if links == nil {
		return nil, ErrNilSyncLinks
	}
	if deliveries == nil {
		return nil, ErrNilDeliveries
	}
	if installations == nil {
		return nil, ErrNilInstallations
	}
	if installationResolver == nil {
		return nil, ErrNilInstallationResolver
	}
	if oauthClient == nil {
		return nil, ErrNilOAuthClient
	}
	if contactsLookup == nil {
		return nil, ErrNilContactLookupService
	}
	if ordersLookup == nil {
		return nil, ErrNilOrderLookupService
	}

	var authorizer Authorizer
	if len(authorizers) > 0 {
		authorizer = authorizers[0]
	}

	return &Handler{
		contactsService:     contactsService,
		ordersService:       ordersService,
		processor:           processor,
		links:               links,
		deliveries:          deliveries,
		installations:       installations,
		installationResolver: installationResolver,
		oauthClient:         oauthClient,
		contactsLookup:      contactsLookup,
		ordersLookup:        ordersLookup,
		clientID:            strings.TrimSpace(clientID),
		clientSecret:        strings.TrimSpace(clientSecret),
		authorizer:          authorizer,
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
	router.Get("/shopify/oauth/install", h.installOAuth)
	router.Get("/shopify/oauth/callback", h.oauthCallback)
	router.Post("/shopify/sync/contacts", h.protect("contact:sync", h.syncContacts))
	router.Post("/shopify/sync/orders", h.protect("order:sync", h.syncOrders))
	router.Post("/shopify/webhooks", h.handleWebhook)
	router.Options("/shopify/ext/orders/:shopifyOrderId", h.handleExtensionOptions)
	router.Options("/shopify/ext/orders/:shopifyOrderId/sync", h.handleExtensionOptions)
	router.Options("/shopify/ext/contacts/:shopifyCustomerId", h.handleExtensionOptions)
	router.Options("/shopify/ext/contacts/:shopifyCustomerId/sync", h.handleExtensionOptions)
	router.Get("/shopify/ext/orders/:shopifyOrderId", h.protectExtension(h.getExtensionOrder))
	router.Post("/shopify/ext/orders/:shopifyOrderId/sync", h.protectExtension(h.syncExtensionOrder))
	router.Get("/shopify/ext/contacts/:shopifyCustomerId", h.protectExtension(h.getExtensionContact))
	router.Post("/shopify/ext/contacts/:shopifyCustomerId/sync", h.protectExtension(h.syncExtensionContact))
}

func (h *Handler) syncContacts(ctx corehttp.Context) error {
	request := parseManualSyncRequest(ctx)
	targetID := strings.TrimSpace(request.ID)
	if targetID == "" {
		return h.mapError(shopifycontactservice.ErrInvalidCustomerID)
	}

	requestCtx := shopifyport.WithShopDomain(ctx.Context(), request.ShopDomain)
	summary, err := h.contactsService.SyncContactByID(requestCtx, "manual", targetID)
	if err != nil {
		return h.mapError(err)
	}

	return ctx.Status(200).JSON(summary)
}

func (h *Handler) syncOrders(ctx corehttp.Context) error {
	request := parseManualSyncRequest(ctx)
	targetID := strings.TrimSpace(request.ID)
	if targetID == "" {
		return h.mapError(shopifyorderservice.ErrInvalidOrderID)
	}

	requestCtx := shopifyport.WithShopDomain(ctx.Context(), request.ShopDomain)
	summary, err := h.ordersService.SyncOrderByID(requestCtx, "manual", targetID)
	if err != nil {
		return h.mapError(err)
	}

	return ctx.Status(200).JSON(summary)
}

func (h *Handler) handleWebhook(ctx corehttp.Context) error {
	body := ctx.Body()
	if !VerifyWebhookSignature(h.clientSecret, body, ctx.GetHeader("X-Shopify-Hmac-Sha256")) {
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
	shopDomain := shopifyport.NormalizeShopDomain(ctx.GetHeader("X-Shopify-Shop-Domain"))
	if !isValidShopDomain(shopDomain) {
		return h.mapError(ErrWebhookShopDomainRequired)
	}

	created, err := h.deliveries.CreateDeliveryIfAbsent(ctx.Context(), deliveryID, topic)
	if err != nil {
		return h.mapError(err)
	}
	if !created {
		return ctx.SendStatus(200)
	}

	if strings.EqualFold(strings.TrimSpace(topic), "app/uninstalled") {
		if err := h.installations.MarkUninstalled(ctx.Context(), shopDomain, time.Now().UTC()); err != nil {
			return h.mapError(err)
		}
		if h.installationResolver != nil {
			if err := h.installationResolver.Refresh(ctx.Context()); err != nil {
				return h.mapError(err)
			}
		}

		return ctx.SendStatus(200)
	}

	shopifyID, err := extractWebhookEntityID(body)
	if err != nil {
		return h.mapError(err)
	}
	if enqueueErr := h.processor.Enqueue(ctx.Context(), topic, shopDomain, shopifyID); enqueueErr != nil {
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
	if errors.Is(err, ErrInvalidWebhookSignature) || errors.Is(err, ErrOAuthHMACInvalid) || errors.Is(err, ErrOAuthStateExpired) || errors.Is(err, ErrSessionTokenMissing) || errors.Is(err, ErrSessionTokenInvalid) || errors.Is(err, ErrSessionTokenExpired) || errors.Is(err, ErrSessionTokenNotYetValid) {
		return corehttp.NewAppError(401, "shopify_authentication_failed", err)
	}
	if errors.Is(err, ErrWebhookDeliveryIDRequired) || errors.Is(err, ErrWebhookTopicRequired) || errors.Is(err, ErrWebhookPayloadIDRequired) || errors.Is(err, ErrWebhookShopDomainRequired) || errors.Is(err, ErrInvalidShopDomain) || errors.Is(err, ErrOAuthCodeRequired) || errors.Is(err, ErrOAuthStateInvalid) {
		return corehttp.NewAppError(400, "invalid_shopify_request", err)
	}
	if errors.Is(err, ErrPublicBaseURLRequired) {
		return corehttp.NewAppError(500, "shopify_public_base_url_unavailable", err)
	}
	if errors.Is(err, ErrProcessorClosed) || errors.Is(err, ErrOAuthUnavailable) || errors.Is(err, shopifyport.ErrInstallationNotFound) || errors.Is(err, shopifyport.ErrAmbiguousInstallations) {
		return corehttp.NewAppError(503, "shopify_integration_unavailable", err)
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

func parseManualSyncRequest(ctx corehttp.Context) manualSyncRequest {
	request := manualSyncRequest{}
	if len(ctx.Body()) > 0 {
		_ = ctx.BodyParser(&request)
	}
	if strings.TrimSpace(request.ID) == "" {
		request.ID = strings.TrimSpace(ctx.Query("id", ""))
	}
	if strings.TrimSpace(request.ShopDomain) == "" {
		request.ShopDomain = strings.TrimSpace(ctx.Query("shop", ctx.Query("shopDomain", "")))
	}
	request.ShopDomain = shopifyport.NormalizeShopDomain(request.ShopDomain)
	return request
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

func isValidShopDomain(shopDomain string) bool {
	resolved := shopifyport.NormalizeShopDomain(shopDomain)
	if resolved == "" || !strings.HasSuffix(resolved, ".myshopify.com") {
		return false
	}

	label := strings.TrimSuffix(resolved, ".myshopify.com")
	if label == "" {
		return false
	}
	for _, char := range label {
		if (char >= 'a' && char <= 'z') || (char >= '0' && char <= '9') || char == '-' {
			continue
		}
		return false
	}

	return true
}
